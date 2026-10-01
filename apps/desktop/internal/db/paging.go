package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// EditorChunk is how many rows the SQL editor fetches at a time.
const EditorChunk = 1000

// Plain row-returning reads in the editor are wrapped as
//
//	SELECT * FROM (<statement>) AS relaydb_q LIMIT n OFFSET m
//
// so the server stops after one chunk instead of streaming every row to be
// thrown away. The user's SQL is untouched inside the parentheses, so CTEs,
// its own LIMIT and ORDER BY keep working. Anything that can't be wrapped
// (writes, EXPLAIN, SHOW, PRAGMA, a wrap the server rejects) runs as typed.
func (s *Session) pageable(stmt string) bool {
	switch firstKeyword(stmt) {
	case "SELECT", "WITH", "VALUES", "TABLE":
	default:
		return false
	}
	if mayWrite(stmt, s.Conn.Driver) { // data-modifying CTEs, INTO, row locks
		return false
	}
	// MariaDB ignores ORDER BY inside a subquery in FROM, so a sorted read
	// would come back in another order: those run as typed.
	return !(s.mariaDB() && containsKeyword(stripStringsAndComments(stmt, true), "ORDER"))
}

func (s *Session) mariaDB() bool {
	return strings.HasPrefix(s.Version, "MariaDB")
}

func pagedSQL(stmt string, offset int) string {
	// Newlines keep a trailing "-- comment" in stmt from swallowing the ")".
	return fmt.Sprintf("SELECT * FROM (\n%s\n) AS relaydb_q LIMIT %d OFFSET %d", stmt, EditorChunk+1, offset)
}

// runPage runs one chunk of a wrapped read on the editor connection.
func (s *Session) runPage(ctx context.Context, stmt string, offset int) (model.ResultSet, error) {
	// In a PostgreSQL transaction an error aborts the whole transaction, so a
	// wrap the server rejects must not take the user's open work with it.
	useSavepoint := s.Conn.Driver == model.Postgres && pgTxStatus(s.editor) == 'T'
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
		return readRows(rows, EditorChunk+1)
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

// RunMore fetches the next chunk of a statement the editor already ran. It
// re-runs the statement from offset, on the same connection and transaction.
func (s *Session) RunMore(ctx context.Context, stmt string, offset int) (model.ResultSet, error) {
	if !s.pageable(stmt) {
		return model.ResultSet{}, errors.New("this statement can't be paged")
	}
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	rs, rolledBack, err := s.onEditor(ctx, stmt, func() (model.ResultSet, error) { return s.runPage(ctx, stmt, offset) })
	if err != nil {
		return model.ResultSet{}, &StatementError{Statement: stmt, Err: err, RolledBack: rolledBack}
	}
	return rs, nil
}

// pgTxStatus reports the PostgreSQL transaction state of conn: 'I' idle,
// 'T' in a transaction, 'E' in a failed one, 0 if unknown.
func pgTxStatus(conn *sql.Conn) byte {
	var status byte
	conn.Raw(func(dc any) error {
		if c, ok := dc.(*stdlib.Conn); ok {
			status = c.Conn().PgConn().TxStatus()
		}
		return nil
	})
	return status
}
