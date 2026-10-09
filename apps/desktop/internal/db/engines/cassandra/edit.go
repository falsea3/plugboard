package cassandra

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var errRowExists = apperr.New("exists", "a row with this primary key already exists")

type change struct {
	cql  string
	args []any
	kind model.ChangeKind
}

func (s *Session) build(ctx context.Context, cs model.ChangeSet, literal bool) ([]change, error) {
	cols, err := s.Columns(ctx, cs.Schema, cs.Table)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table %s not found", cs.Table)
	}
	b := builder{table: qualified(cs.Schema, cs.Table), cols: map[string]model.Column{}, literal: literal}
	for _, c := range cols {
		b.cols[c.Name] = c
		if c.PrimaryKey {
			b.keys = append(b.keys, c)
		}
	}
	out := make([]change, 0, len(cs.Changes))
	for i, ch := range cs.Changes {
		c, err := b.change(ch)
		if err != nil {
			return nil, &db.ApplyError{Index: i, Err: err}
		}
		out = append(out, c)
	}
	return out, nil
}

type builder struct {
	table   string
	cols    map[string]model.Column
	keys    []model.Column
	literal bool
	args    []any
}

func (b *builder) change(ch model.RowChange) (change, error) {
	b.args = nil
	var cql string
	switch ch.Kind {
	case model.ChangeUpdate:
		if len(ch.Values) == 0 {
			return change{}, errors.New("nothing to update")
		}
		var sets []string
		for _, n := range slices.Sorted(maps.Keys(ch.Values)) {
			col, ok := b.cols[n]
			if !ok {
				return change{}, fmt.Errorf("unknown column %s", n)
			}
			if col.PrimaryKey {
				return change{}, fmt.Errorf("%s is part of the primary key — Cassandra can't change it; delete the row and add a new one", n)
			}
			v, err := b.value(col, ch.Values[n])
			if err != nil {
				return change{}, err
			}
			sets = append(sets, quote(n)+" = "+v)
		}
		where, err := b.where(ch.Key)
		if err != nil {
			return change{}, err
		}
		cql = "UPDATE " + b.table + " SET " + strings.Join(sets, ", ") + " WHERE " + where + " IF EXISTS"
	case model.ChangeDelete:
		where, err := b.where(ch.Key)
		if err != nil {
			return change{}, err
		}
		cql = "DELETE FROM " + b.table + " WHERE " + where + " IF EXISTS"
	case model.ChangeInsert:
		for _, k := range b.keys {
			if v, ok := ch.Values[k.Name]; !ok || v == nil {
				return change{}, fmt.Errorf("a new row needs %s — every primary key column", k.Name)
			}
		}
		names := slices.Sorted(maps.Keys(ch.Values))
		quoted := make([]string, len(names))
		vals := make([]string, len(names))
		for i, n := range names {
			col, ok := b.cols[n]
			if !ok {
				return change{}, fmt.Errorf("unknown column %s", n)
			}
			v, err := b.value(col, ch.Values[n])
			if err != nil {
				return change{}, err
			}
			quoted[i], vals[i] = quote(n), v
		}
		cql = "INSERT INTO " + b.table + " (" + strings.Join(quoted, ", ") + ") VALUES (" + strings.Join(vals, ", ") + ") IF NOT EXISTS"
	default:
		return change{}, fmt.Errorf("unknown change %q", ch.Kind)
	}
	return change{cql: cql, args: b.args, kind: ch.Kind}, nil
}

func (b *builder) where(key map[string]any) (string, error) {
	parts := make([]string, len(b.keys))
	for i, k := range b.keys {
		v, ok := key[k.Name]
		if !ok || v == nil {
			return "", fmt.Errorf("missing value for key column %s", k.Name)
		}
		lit, err := b.value(k, v)
		if err != nil {
			return "", err
		}
		parts[i] = quote(k.Name) + " = " + lit
	}
	return strings.Join(parts, " AND "), nil
}

func (b *builder) value(col model.Column, v any) (string, error) {
	if m, ok := v.(map[string]any); ok {
		switch m["$expr"] {
		case "now":
			if baseType(col.Type) == "timestamp" {
				return "toTimestamp(now())", nil
			}
			return "", fmt.Errorf("%s isn't a timestamp", col.Name)
		case "default":
			return "", fmt.Errorf("Cassandra has no column defaults; set %s to a value or NULL", col.Name)
		}
		return "", fmt.Errorf("unknown expression for %s", col.Name)
	}
	if v == nil {
		return "null", nil
	}
	text, ok := v.(string)
	if !ok {
		switch x := v.(type) {
		case float64:
			text = strconv.FormatFloat(x, 'f', -1, 64)
		case bool:
			text = strconv.FormatBool(x)
		default:
			text = fmt.Sprint(v)
		}
	}
	j, err := jsonFor(col, text)
	if err != nil {
		return "", err
	}
	if b.literal {
		return "fromJson('" + strings.ReplaceAll(j, "'", "''") + "')", nil
	}
	b.args = append(b.args, j)
	return "fromJson(?)", nil
}

func (s *Session) PreviewChanges(ctx context.Context, cs model.ChangeSet) ([]string, error) {
	built, err := s.build(ctx, cs, true)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(built))
	for i, c := range built {
		out[i] = c.cql
	}
	return out, nil
}

func (s *Session) ApplyChanges(ctx context.Context, cs model.ChangeSet) (int, error) {
	if s.conn.ReadOnly {
		return 0, &db.ReadOnlyError{Reason: "edits"}
	}
	built, err := s.build(ctx, cs, false)
	if err != nil {
		return 0, err
	}
	gs, err := s.main()
	if err != nil {
		return 0, err
	}
	for i, c := range built {
		applied, err := gs.Query(c.cql, c.args...).WithContext(ctx).MapScanCAS(map[string]any{})
		if err != nil {
			return i, &db.ApplyError{Index: i, Err: err}
		}
		if !applied && c.kind == model.ChangeInsert {
			return i, &db.ApplyError{Index: i, Err: errRowExists}
		}
		if !applied {
			return i, &db.ApplyError{Index: i, Err: db.ErrRowGone}
		}
	}
	return len(built), nil
}
