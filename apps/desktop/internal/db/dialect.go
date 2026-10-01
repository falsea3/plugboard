package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// Dialect hides the per-engine differences: how to build a DSN, how to quote
// identifiers and where to read the catalog from. Everything else goes through
// database/sql and is shared.
type Dialect interface {
	Driver() model.Driver
	Open(c model.Connection) (*sql.DB, error)
	QuoteIdent(name string) string
	ServerVersion(ctx context.Context, db *sql.DB) (string, error)
	DefaultSchema(ctx context.Context, db *sql.DB, c model.Connection) (string, error)
	ListSchemas(ctx context.Context, db *sql.DB) ([]string, error)
	ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error)
	ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error)
}

func DialectFor(d model.Driver) (Dialect, error) {
	switch d {
	case model.Postgres:
		return postgresDialect{}, nil
	case model.MySQL:
		return mysqlDialect{}, nil
	case model.SQLite:
		return sqliteDialect{}, nil
	}
	return nil, fmt.Errorf("unsupported driver %q", d)
}

func qualified(d Dialect, schema, table string) string {
	if schema == "" {
		return d.QuoteIdent(table)
	}
	return d.QuoteIdent(schema) + "." + d.QuoteIdent(table)
}

func queryStrings(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
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

func queryTables(ctx context.Context, db *sql.DB, schema, query string, args ...any) ([]model.TableInfo, error) {
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
		out = append(out, model.TableInfo{Schema: schema, Name: name, Kind: tableKind(kind)})
	}
	return out, rows.Err()
}

func tableKind(raw string) string {
	switch raw {
	case "VIEW", "view", "SYSTEM VIEW", "MATERIALIZED VIEW":
		return "view"
	}
	return "table"
}

// queryColumns reads columns from a catalog query returning name, type,
// nullable, default and primary key, then optionally enum labels as a JSON
// array and the type to cast bound values to.
func queryColumns(ctx context.Context, db *sql.DB, query string, args ...any) ([]model.Column, error) {
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
		c.Binary = isBinaryType(baseType(c.Type))
		out = append(out, c)
	}
	return out, rows.Err()
}
