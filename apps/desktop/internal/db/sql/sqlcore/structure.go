package sqlcore

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

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

const ddlLockWait = 5

func (s *Session) ApplyStructure(ctx context.Context, sc model.StructureChange) (applied int, partial bool, err error) {
	if s.Conn.ReadOnly {
		return 0, false, &db.ReadOnlyError{Reason: "structure changes"}
	}
	built, err := s.buildStructure(ctx, sc)
	if err != nil {
		return 0, false, err
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return 0, false, err
	}
	defer conn.Close()
	set, reset := s.Dialect.LockWait(ddlLockWait)
	if reset != "" {
		defer func() {
			if _, err := conn.ExecContext(context.WithoutCancel(ctx), reset); err != nil {
				conn.Raw(func(any) error { return driver.ErrBadConn })
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
					return i, i > 0, &db.ApplyError{Index: i, Err: s.ddlError(err)}
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
				return 0, false, &db.ApplyError{Index: i, Err: s.ddlError(err)}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return len(built), false, nil
}

func (s *Session) ddlError(err error) error {
	if s.Dialect.Classify(err) == dialect.LockTimeout {
		return db.ErrTableBusy
	}
	return err
}

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
	alter := "ALTER TABLE " + s.qualified(sc.Schema, sc.Table) + " "
	out := make([][]string, 0, len(sc.Changes))
	for i, ch := range sc.Changes {
		stmts, err := s.columnStatements(ctx, sc, alter, byName, ch)
		if err != nil {
			return nil, &db.ApplyError{Index: i, Err: err}
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
