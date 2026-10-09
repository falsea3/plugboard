package sqlcore

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (s *Session) ApplyChanges(ctx context.Context, cs model.ChangeSet) (int, error) {
	if s.Conn.ReadOnly {
		return 0, &db.ReadOnlyError{Reason: "edits"}
	}
	if !s.Dialect.Editable() {
		return 0, db.ErrNotSupported
	}
	stmts, err := s.buildChanges(ctx, cs, false)
	if err != nil {
		return 0, err
	}
	n, committing, err := s.applyOnce(ctx, stmts)
	if !committing && s.connLost(err) && ctx.Err() == nil {
		n, _, err = s.applyOnce(ctx, stmts)
	}
	return n, err
}

func (s *Session) applyOnce(ctx context.Context, stmts []builtChange) (n int, committing bool, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	for i, st := range stmts {
		res, err := tx.ExecContext(ctx, st.sql, st.args...)
		if err != nil {
			return 0, false, &db.ApplyError{Index: i, Err: err}
		}
		if st.kind != model.ChangeInsert {
			if n, err := res.RowsAffected(); err == nil && n != 1 {
				return 0, false, &db.ApplyError{Index: i, Err: db.ErrRowGone}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, true, err
	}
	return len(stmts), false, nil
}

func (s *Session) PreviewChanges(ctx context.Context, cs model.ChangeSet) ([]string, error) {
	stmts, err := s.buildChanges(ctx, cs, true)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(stmts))
	for i, st := range stmts {
		out[i] = st.sql
	}
	return out, nil
}

type builtChange struct {
	kind model.ChangeKind
	sql  string
	args []any
}

func (s *Session) buildChanges(ctx context.Context, cs model.ChangeSet, literal bool) ([]builtChange, error) {
	if cs.Table == "" {
		return nil, errors.New("table is required")
	}
	cols, err := s.Columns(ctx, cs.Schema, cs.Table)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table %s not found", cs.Table)
	}
	byName := make(map[string]model.Column, len(cols))
	var keys []model.Column
	for _, c := range cols {
		byName[c.Name] = c
		if c.PrimaryKey {
			keys = append(keys, c)
		}
	}
	b := changeBuilder{d: s.Dialect, table: s.qualified(cs.Schema, cs.Table), cols: byName, keys: keys, literal: literal}
	out := make([]builtChange, 0, len(cs.Changes))
	for i, ch := range cs.Changes {
		st, err := b.build(ch)
		if err != nil {
			return nil, &db.ApplyError{Index: i, Err: err}
		}
		out = append(out, st)
	}
	return out, nil
}

type changeBuilder struct {
	d       dialect.Dialect
	table   string
	cols    map[string]model.Column
	keys    []model.Column
	literal bool

	args []any
}

func (b *changeBuilder) build(ch model.RowChange) (builtChange, error) {
	b.args = nil
	var sql string
	switch ch.Kind {
	case model.ChangeUpdate:
		if len(ch.Values) == 0 {
			return builtChange{}, errors.New("nothing to update")
		}
		set, err := b.assignments(ch.Values)
		if err != nil {
			return builtChange{}, err
		}
		where, err := b.where(ch.Key)
		if err != nil {
			return builtChange{}, err
		}
		sql = "UPDATE " + b.table + " SET " + set + " WHERE " + where
	case model.ChangeDelete:
		where, err := b.where(ch.Key)
		if err != nil {
			return builtChange{}, err
		}
		sql = "DELETE FROM " + b.table + " WHERE " + where
	case model.ChangeInsert:
		names := slices.Sorted(maps.Keys(ch.Values))
		if len(names) == 0 {
			sql = b.d.InsertDefaults(b.table)
			break
		}
		quoted := make([]string, len(names))
		vals := make([]string, len(names))
		for i, n := range names {
			col, err := b.column(n, ch.Values[n])
			if err != nil {
				return builtChange{}, err
			}
			quoted[i] = b.d.QuoteIdent(n)
			vals[i] = b.value(col, ch.Values[n])
		}
		sql = "INSERT INTO " + b.table + " (" + strings.Join(quoted, ", ") + ") VALUES (" + strings.Join(vals, ", ") + ")"
	default:
		return builtChange{}, fmt.Errorf("unknown change %q", ch.Kind)
	}
	return builtChange{kind: ch.Kind, sql: sql, args: b.args}, nil
}

func (b *changeBuilder) assignments(values map[string]any) (string, error) {
	names := slices.Sorted(maps.Keys(values))
	parts := make([]string, len(names))
	for i, n := range names {
		col, err := b.column(n, values[n])
		if err != nil {
			return "", err
		}
		parts[i] = b.d.QuoteIdent(n) + " = " + b.value(col, values[n])
	}
	return strings.Join(parts, ", "), nil
}

func (b *changeBuilder) where(key map[string]any) (string, error) {
	if len(b.keys) == 0 {
		return "", errors.New("the table has no primary key, so rows can't be identified safely")
	}
	parts := make([]string, len(b.keys))
	for i, k := range b.keys {
		v, ok := key[k.Name]
		if !ok || v == nil {
			return "", fmt.Errorf("missing value for key column %s", k.Name)
		}
		parts[i] = b.d.QuoteIdent(k.Name) + " = " + b.value(k, v)
	}
	return strings.Join(parts, " AND "), nil
}

func (b *changeBuilder) column(name string, v any) (model.Column, error) {
	col, ok := b.cols[name]
	if !ok {
		return model.Column{}, fmt.Errorf("unknown column %s", name)
	}
	return col, b.writable(col, v)
}

func (b *changeBuilder) writable(col model.Column, v any) error {
	if expr, ok := exprOf(v); ok {
		switch expr {
		case "now":
			return nil
		case "default":
			if !b.d.CanUpdateToDefault() {
				return fmt.Errorf("this database can't reset %s to its default on an existing row", col.Name)
			}
			return nil
		}
		return fmt.Errorf("unknown expression %q for %s", expr, col.Name)
	}
	if v != nil && col.Kind == model.KindBinary {
		return fmt.Errorf("column %s is binary and can't be edited yet (only set to NULL)", col.Name)
	}
	return nil
}

func (b *changeBuilder) value(col model.Column, v any) string {
	if expr, ok := exprOf(v); ok {
		return exprSQL(col, expr)
	}
	text, isNull := paramText(v)
	if b.literal {
		if isNull {
			return "NULL"
		}
		return b.d.QuoteString(text)
	}
	if isNull {
		b.args = append(b.args, nil)
	} else {
		b.args = append(b.args, text)
	}
	return b.d.Param(len(b.args), col)
}

func exprOf(v any) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	e, ok := m["$expr"].(string)
	return e, ok
}

func exprSQL(col model.Column, expr string) string {
	if expr == "default" {
		return "DEFAULT"
	}
	t := dialect.BaseType(col.Type)
	switch {
	case t == "date":
		return "CURRENT_DATE"
	case strings.HasPrefix(t, "time") && !strings.HasPrefix(t, "timestamp"):
		return "CURRENT_TIME"
	}
	return "CURRENT_TIMESTAMP"
}

func paramText(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", true
	case string:
		return x, false
	case bool:
		if x {
			return "true", false
		}
		return "false", false
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return strconv.FormatInt(int64(x), 10), false
		}
		return strconv.FormatFloat(x, 'f', -1, 64), false
	}
	return fmt.Sprint(v), false
}
