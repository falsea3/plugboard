package dialect

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func Qualified(q interface{ QuoteIdent(string) string }, schema, table string) string {
	if schema == "" {
		return q.QuoteIdent(table)
	}
	return q.QuoteIdent(schema) + "." + q.QuoteIdent(table)
}

func BaseType(t string) string {
	if i := strings.IndexByte(t, '('); i >= 0 {
		t = t[:i]
	}
	return strings.ToLower(strings.TrimSpace(t))
}

func ByteOffset(s string, chars int) int {
	for i := range s {
		if chars == 0 {
			return i
		}
		chars--
	}
	return len(s)
}

func FirstWord(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	end := strings.IndexFunc(t, func(r rune) bool { return !(r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') })
	if end < 0 {
		return t
	}
	return t[:end]
}

func QueryStrings(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func QueryTables(ctx context.Context, db *sql.DB, schema, query string, args ...any) ([]model.TableInfo, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.TableInfo{}
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			return nil, err
		}
		out = append(out, model.TableInfo{Schema: schema, Name: name, Kind: TableKind(kind)})
	}
	return out, rows.Err()
}

func QueryRelations(ctx context.Context, db *sql.DB, query string, args ...any) ([]model.Relation, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Relation{}
	for rows.Next() {
		var r model.Relation
		var col, refCol, onDelete, onUpdate string
		if err := rows.Scan(&r.Name, &r.Table, &col, &r.RefSchema, &r.RefTable, &refCol, &onDelete, &onUpdate); err != nil {
			return nil, err
		}
		if n := len(out); n > 0 && out[n-1].Name == r.Name && out[n-1].Table == r.Table {
			out[n-1].Columns = append(out[n-1].Columns, col)
			out[n-1].RefColumns = append(out[n-1].RefColumns, refCol)
			continue
		}
		r.Columns = []string{col}
		r.RefColumns = []string{refCol}
		r.OnDelete = RefAction(onDelete)
		r.OnUpdate = RefAction(onUpdate)
		out = append(out, r)
	}
	return out, rows.Err()
}

func QueryIndexes(ctx context.Context, db *sql.DB, query string, args ...any) ([]model.Index, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Index{}
	for rows.Next() {
		var ix model.Index
		var column string
		var method, where, definition sql.NullString
		if err := rows.Scan(&ix.Name, &ix.Unique, &ix.Primary, &method, &where, &definition, &column); err != nil {
			return nil, err
		}
		if n := len(out); n > 0 && out[n-1].Name == ix.Name {
			out[n-1].Columns = append(out[n-1].Columns, column)
			continue
		}
		ix.Method, ix.Where, ix.Definition = method.String, where.String, definition.String
		ix.Columns = []string{column}
		out = append(out, ix)
	}
	return out, rows.Err()
}

func RefAction(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "C", "CASCADE":
		return "CASCADE"
	case "R", "RESTRICT":
		return "RESTRICT"
	case "N", "SET NULL":
		return "SET NULL"
	case "D", "SET DEFAULT":
		return "SET DEFAULT"
	}
	return "NO ACTION"
}

func TableKind(raw string) string {
	switch raw {
	case "VIEW", "view", "SYSTEM VIEW", "MATERIALIZED VIEW":
		return "view"
	}
	return "table"
}

func QueryColumns(ctx context.Context, db *sql.DB, query string, args ...any) ([]model.Column, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []model.Column{}
	for rows.Next() {
		var c model.Column
		var def, enum, castType sql.NullString
		dest := []any{&c.Name, &c.Type, &c.Nullable, &def, &c.PrimaryKey, &enum, &castType}
		if err := rows.Scan(dest[:len(names)]...); err != nil {
			return nil, err
		}
		if def.Valid {
			c.Default = &def.String
		}
		if enum.Valid {
			if err := json.Unmarshal([]byte(enum.String), &c.Enum); err != nil {
				return nil, fmt.Errorf("enum values of %s: %w", c.Name, err)
			}
		}
		c.CastType = castType.String
		out = append(out, c)
	}
	return out, rows.Err()
}
