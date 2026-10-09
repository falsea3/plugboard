package sqlcore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const MaxResultRows = 5000

func (s *Session) Run(ctx context.Context, script string) ([]model.ResultSet, error) {
	stmts := sqltext.Split(script, s.Dialect.Syntax())
	if len(stmts) == 0 {
		return nil, errors.New("nothing to run")
	}
	if s.Conn.ReadOnly {
		for i, stmt := range stmts {
			if reason := ReadOnlyReason(s.Dialect, stmt); reason != "" {
				return nil, &db.StatementError{Statement: stmt, Index: i, Err: &db.ReadOnlyError{Reason: reason}, Position: -1}
			}
		}
	}
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	results := make([]model.ResultSet, 0, len(stmts))
	for i, stmt := range stmts {
		rs, rolledBack, err := s.onEditor(ctx, stmt, func() (model.ResultSet, error) { return s.runOne(ctx, stmt) })
		if err != nil {
			return results, &db.StatementError{Statement: stmt, Index: i, Err: err, RolledBack: rolledBack, Position: s.Dialect.ErrorPosition(err, stmt)}
		}
		s.editorTx = s.Dialect.TransactionAfter(stmt, s.editorTx)
		results = append(results, rs)
	}
	return results, nil
}

func (s *Session) onEditor(ctx context.Context, stmt string, run func() (model.ResultSet, error)) (rs model.ResultSet, rolledBack bool, err error) {
	if err := s.openEditor(ctx); err != nil {
		return model.ResultSet{}, false, err
	}
	inTx := s.inTransaction()
	rs, err = run()
	if s.stalePlan(err) && ctx.Err() == nil {
		rs, err = run()
	}
	if s.connLost(err) && ctx.Err() == nil && !inTx && !s.mayWrite(stmt) {
		s.dropEditor()
		if err = s.openEditor(ctx); err == nil {
			rs, err = run()
		}
	}
	if err == nil || (ctx.Err() == nil && !s.connLost(err)) {
		return rs, false, err
	}
	s.dropEditor()
	if ctx.Err() == nil {
		err = db.ErrConnLost
	}
	return model.ResultSet{}, inTx, err
}

func (s *Session) openEditor(ctx context.Context) error {
	if s.editor != nil {
		return nil
	}
	c, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	s.editor = c
	return nil
}

func (s *Session) dropEditor() {
	if s.editor != nil {
		s.editor.Raw(func(any) error { return driver.ErrBadConn })
		s.editor.Close()
		s.editor = nil
	}
	s.editorTx = false
}

func (s *Session) inTransaction() bool {
	switch s.Dialect.TxStatus(s.editor) {
	case dialect.TxOpen, dialect.TxFailed:
		return true
	case dialect.TxIdle:
		return false
	}
	return s.editorTx
}

func (s *Session) runOne(ctx context.Context, stmt string) (model.ResultSet, error) {
	if s.pageable(stmt) && s.Dialect.TxStatus(s.editor) != dialect.TxFailed {
		rs, err := s.runPage(ctx, stmt, 0)
		if err == nil || s.connLost(err) || ctx.Err() != nil {
			return rs, err
		}
	}
	start := time.Now()
	if s.Dialect.ReturnsRows(stmt) {
		rows, err := s.editor.QueryContext(ctx, stmt)
		if err != nil {
			return model.ResultSet{}, err
		}
		defer rows.Close()
		rs, err := s.readRows(rows, MaxResultRows)
		if err != nil {
			return model.ResultSet{}, err
		}
		rs.Statement = stmt
		rs.DurationMs = msSince(start)
		return rs, nil
	}
	res, err := s.editor.ExecContext(ctx, stmt)
	if err != nil {
		return model.ResultSet{}, err
	}
	affected, _ := res.RowsAffected()
	return model.ResultSet{
		Statement:    stmt,
		Columns:      []model.ResultColumn{},
		Rows:         [][]any{},
		RowsAffected: affected,
		DurationMs:   msSince(start),
	}, nil
}

func (s *Session) readRows(rows *sql.Rows, maxRows int) (model.ResultSet, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return model.ResultSet{}, err
	}
	cols := make([]model.ResultColumn, len(types))
	kinds := make([]dialect.Type, len(types))
	for i, t := range types {
		cols[i] = model.ResultColumn{Name: t.Name(), Type: strings.ToLower(t.DatabaseTypeName())}
		kinds[i] = s.Dialect.TypeOf(cols[i].Type)
		cols[i].Kind = kinds[i].Kind
	}
	rs := model.ResultSet{Columns: cols, Rows: [][]any{}, HasRows: true}
	for rows.Next() {
		if len(rs.Rows) >= maxRows {
			rs.Truncated = true
			break
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return model.ResultSet{}, err
		}
		for i, v := range vals {
			vals[i] = normalizeValue(v, kinds[i])
		}
		rs.Rows = append(rs.Rows, vals)
	}
	return rs, rows.Err()
}

func msSince(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000
}

const maxChecked = 200

func checkStatement(ctx context.Context, d dialect.Dialect, conn *sql.Conn, stmt string) error {
	if c, ok := d.(dialect.SyntaxCheck); ok {
		return c.CheckSyntax(ctx, conn, stmt)
	}
	prepared, err := conn.PrepareContext(ctx, stmt)
	if err == nil {
		prepared.Close()
	}
	return err
}

func (s *Session) CheckSyntax(ctx context.Context, script string) ([]model.SyntaxProblem, error) {
	stmts := sqltext.Split(script, s.Dialect.Syntax())
	if len(stmts) > maxChecked {
		stmts = stmts[:maxChecked]
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	out := []model.SyntaxProblem{}
	for i, stmt := range stmts {
		err := checkStatement(ctx, s.Dialect, conn, stmt)
		if err == nil {
			continue
		}
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		if s.Dialect.Classify(err) == dialect.BadSyntax {
			out = append(out, model.SyntaxProblem{Index: i, Position: s.Dialect.ErrorPosition(err, stmt), Message: s.Dialect.ErrorMessage(err)})
		}
	}
	return out, nil
}
