package db

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// Structure edits arrive as column changes from the Structure tab and become
// ALTER TABLE statements in each engine's own words (Dialect.ColumnDDL).
// PostgreSQL and SQLite run them in one transaction; MySQL commits every
// ALTER TABLE by itself, so there a failure leaves the earlier ones applied.
// A change that would wait long for a lock gives up instead (ddlLockWait).

// PreviewStructure returns the statements ApplyStructure would run.
func (s *Session) PreviewStructure(ctx context.Context, sc model.StructureChange) ([]string, error) {
	built, err := s.buildStructure(ctx, sc)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, b := range built {
		out = append(out, b...)
	}
	return out, nil
}

// ddlLockWait is how many seconds a change waits for a lock another
// transaction holds — often one left open in the SQL editor — before giving
// up. Every other query on the table queues behind a waiting ALTER TABLE.
const ddlLockWait = 5

var ErrTableBusy = errors.New("another transaction is using the table — perhaps one left open in the SQL editor. Commit or roll it back, then try again")

// ApplyStructure runs the changes and returns how many were applied.
func (s *Session) ApplyStructure(ctx context.Context, sc model.StructureChange) (applied int, partial bool, err error) {
	if s.Conn.ReadOnly {
		return 0, false, &ReadOnlyError{Reason: "structure changes"}
	}
	built, err := s.buildStructure(ctx, sc)
	if err != nil {
		return 0, false, err
	}
	// One connection, so the lock wait set on it covers every statement.
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return 0, false, err
	}
	defer conn.Close()
	set, reset := s.Dialect.LockWait(ddlLockWait)
	if reset != "" {
		defer func() {
			if _, err := conn.ExecContext(context.WithoutCancel(ctx), reset); err != nil {
				conn.Raw(func(any) error { return driver.ErrBadConn }) // don't pass the setting on to the pool
			}
		}()
	}

	if !s.Dialect.TransactionalDDL() {
		if set != "" {
			if _, err := conn.ExecContext(ctx, set); err != nil {
				return 0, false, err
			}
		}
		for i, stmts := range built {
			for _, stmt := range stmts {
				if _, err := conn.ExecContext(ctx, stmt); err != nil {
					return i, i > 0, &ApplyError{Index: i, Err: ddlError(err)}
				}
			}
		}
		return len(built), false, nil
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	if set != "" {
		if _, err := tx.ExecContext(ctx, set); err != nil {
			return 0, false, err
		}
	}
	for i, stmts := range built {
		for _, stmt := range stmts {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return 0, false, &ApplyError{Index: i, Err: ddlError(err)}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return len(built), false, nil
}

// ddlError puts a lock wait that ran out in plain words.
func ddlError(err error) error {
	var pgErr *pgconn.PgError
	var myErr *mysql.MySQLError
	if (errors.As(err, &pgErr) && pgErr.Code == "55P03") || (errors.As(err, &myErr) && myErr.Number == 1205) {
		return ErrTableBusy
	}
	return err
}

// buildStructure renders each change as its statements, in order. Adding
// and dropping a column read the same in every engine; changing one is the
// dialect's (Dialect.ColumnDDL).
func (s *Session) buildStructure(ctx context.Context, sc model.StructureChange) ([][]string, error) {
	if sc.Table == "" {
		return nil, errors.New("table is required")
	}
	cols, err := s.Columns(ctx, sc.Schema, sc.Table)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table %s not found", sc.Table)
	}
	byName := make(map[string]model.Column, len(cols))
	for _, c := range cols {
		byName[c.Name] = c
	}
	alter := "ALTER TABLE " + qualified(s.Dialect, sc.Schema, sc.Table) + " "
	out := make([][]string, 0, len(sc.Changes))
	for i, ch := range sc.Changes {
		stmts, err := s.columnStatements(ctx, sc, alter, byName, ch)
		if err != nil {
			return nil, &ApplyError{Index: i, Err: err}
		}
		out = append(out, stmts)
	}
	return out, nil
}

func (s *Session) columnStatements(ctx context.Context, sc model.StructureChange, alter string, byName map[string]model.Column, ch model.ColumnChange) ([]string, error) {
	if ch.Kind == model.ChangeInsert {
		if ch.Name == nil || strings.TrimSpace(*ch.Name) == "" {
			return nil, errors.New("the new column needs a name")
		}
		if ch.Type == nil || strings.TrimSpace(*ch.Type) == "" {
			return nil, fmt.Errorf("column %s needs a type", *ch.Name)
		}
		def := s.Dialect.QuoteIdent(strings.TrimSpace(*ch.Name)) + " " + strings.TrimSpace(*ch.Type)
		if ch.Nullable != nil && !*ch.Nullable {
			def += " NOT NULL"
		}
		if ch.DefaultSet && ch.Default != nil {
			def += " DEFAULT " + *ch.Default
		}
		return []string{alter + "ADD COLUMN " + def}, nil
	}

	cur, ok := byName[ch.Column]
	if !ok {
		return nil, fmt.Errorf("column %s no longer exists — refresh and try again", ch.Column)
	}
	switch ch.Kind {
	case model.ChangeDelete:
		return []string{alter + "DROP COLUMN " + s.Dialect.QuoteIdent(cur.Name)}, nil
	case model.ChangeUpdate:
		if ch.Name != nil && strings.TrimSpace(*ch.Name) == "" {
			return nil, fmt.Errorf("column %s can't have an empty name", cur.Name)
		}
		if ch.Type != nil && strings.TrimSpace(*ch.Type) == "" {
			return nil, fmt.Errorf("column %s can't have an empty type", cur.Name)
		}
		return s.Dialect.ColumnDDL(ctx, s.DB, sc.Schema, sc.Table, alter, cur, ch)
	}
	return nil, fmt.Errorf("unknown change %q", ch.Kind)
}

// typeChanged reports a requested type that differs from the current one.
func typeChanged(cur model.Column, ch model.ColumnChange) bool {
	return ch.Type != nil && !strings.EqualFold(strings.TrimSpace(*ch.Type), cur.Type)
}

func nameOf(cur model.Column, ch model.ColumnChange) string {
	if ch.Name != nil {
		return strings.TrimSpace(*ch.Name)
	}
	return cur.Name
}
