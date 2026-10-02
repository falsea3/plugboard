package dialect

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
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
