package sqlcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqltext"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

const EditorChunk = 1000

func (s *Session) pageable(stmt string) bool {
	switch sqltext.FirstKeyword(stmt) {
	case "SELECT", "WITH", "VALUES", "TABLE":
	default:
		return false
	}
	if s.mayWrite(stmt) {
		return false
	}
	return s.Dialect.CanPage(stmt)
}

func pagedSQL(stmt string, offset int) string {
	return fmt.Sprintf("SELECT * FROM (\n%s\n) AS relaydb_q LIMIT %d OFFSET %d", stmt, EditorChunk+1, offset)
}

func (s *Session) runPage(ctx context.Context, stmt string, offset int) (model.ResultSet, error) {
	useSavepoint := s.Dialect.ErrorAbortsTransaction() && s.Dialect.TxStatus(s.editor) == dialect.TxOpen
	if useSavepoint {
		if _, err := s.editor.ExecContext(ctx, "SAVEPOINT relaydb_page"); err != nil {
			return model.ResultSet{}, err
		}
	}
	start := time.Now()
	rs, err := func() (model.ResultSet, error) {
		rows, err := s.editor.QueryContext(ctx, pagedSQL(stmt, offset))
		if err != nil {
			return model.ResultSet{}, err
		}
		defer rows.Close()
		return s.readRows(rows, EditorChunk+1)
	}()
	if useSavepoint {
		if err != nil {
			s.editor.ExecContext(context.WithoutCancel(ctx), "ROLLBACK TO SAVEPOINT relaydb_page")
		} else {
			s.editor.ExecContext(ctx, "RELEASE SAVEPOINT relaydb_page")
		}
	}
	if err != nil {
		return model.ResultSet{}, err
	}
	rs.Truncated = false
	if len(rs.Rows) > EditorChunk {
		rs.Rows = rs.Rows[:EditorChunk]
		rs.HasMore = true
	}
	rs.Statement = stmt
	rs.Offset = offset
	rs.Pageable = true
	rs.DurationMs = msSince(start)
	return rs, nil
}

func (s *Session) RunMore(ctx context.Context, stmt string, offset int) (model.ResultSet, error) {
	if !s.pageable(stmt) {
		return model.ResultSet{}, errors.New("this statement can't be paged")
	}
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	rs, rolledBack, err := s.onEditor(ctx, stmt, func() (model.ResultSet, error) { return s.runPage(ctx, stmt, offset) })
	if err != nil {
		return model.ResultSet{}, &db.StatementError{Statement: stmt, Err: err, RolledBack: rolledBack}
	}
	return rs, nil
}
