package db

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	_ "modernc.org/sqlite"
)

type sqliteDialect struct{}

func (sqliteDialect) Driver() model.Driver { return model.SQLite }

func (sqliteDialect) Open(c model.Connection) (*sql.DB, error) {
	if strings.TrimSpace(c.File) == "" {
		return nil, errors.New("choose a SQLite database file")
	}
	q := url.Values{}
	// rw, not rwc: a typo in the path must not silently create an empty database.
	mode := "rw"
	if c.ReadOnly {
		mode = "ro"
	}
	q.Add("mode", mode)
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	return sql.Open("sqlite", "file:"+sqliteURIPath.Replace(c.File)+"?"+q.Encode())
}

var sqliteURIPath = strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23")

func (sqliteDialect) QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (sqliteDialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&v)
	return "SQLite " + v, err
}

func (sqliteDialect) DefaultSchema(context.Context, *sql.DB, model.Connection) (string, error) {
	return "main", nil
}

func (sqliteDialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA database_list`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var seq int
		var name string
		var file sql.NullString
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return nil, err
		}
		if name != "temp" {
			out = append(out, name)
		}
	}
	return out, rows.Err()
}

func (d sqliteDialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return queryTables(ctx, db, schema, `
		SELECT name, type FROM `+d.QuoteIdent(schema)+`.sqlite_master
		WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%'
		ORDER BY name`)
}

func (d sqliteDialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, type, "notnull", dflt_value, pk FROM pragma_table_info(?, ?)`, table, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Column{}
	for rows.Next() {
		var c model.Column
		var notNull, pk int
		var def sql.NullString
		if err := rows.Scan(&c.Name, &c.Type, &notNull, &def, &pk); err != nil {
			return nil, err
		}
		c.Nullable = notNull == 0
		c.PrimaryKey = pk > 0
		c.Binary = isBinaryType(baseType(c.Type))
		if def.Valid {
			c.Default = &def.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (sqliteDialect) TransactionalDDL() bool { return true }

// LockWait: the busy_timeout set in Open already bounds the wait.
func (sqliteDialect) LockWait(int) (set, reset string) { return "", "" }

// ColumnDDL: of changes to a column, SQLite's ALTER TABLE can only rename it.
// Changing a type, NOT NULL or a default means rebuilding the table, which is
// left to the SQL editor.
func (d sqliteDialect) ColumnDDL(_ context.Context, _ *sql.DB, _, _, alter string, cur model.Column, ch model.ColumnChange) ([]string, error) {
	if typeChanged(cur, ch) || ch.DefaultSet || (ch.Nullable != nil && *ch.Nullable != cur.Nullable) {
		return nil, errors.New("SQLite can only rename a column in place; changing its type, NULL or default means recreating the table in the SQL editor")
	}
	if name := nameOf(cur, ch); name != cur.Name {
		return []string{alter + "RENAME COLUMN " + d.QuoteIdent(cur.Name) + " TO " + d.QuoteIdent(name)}, nil
	}
	return nil, nil
}
