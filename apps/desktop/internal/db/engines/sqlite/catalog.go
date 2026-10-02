package sqlite

import (
	"context"
	"database/sql"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func (Dialect) DefaultSchema(context.Context, *sql.DB, model.Connection) (string, error) {
	return "main", nil
}

func (Dialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
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

func (d Dialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return dialect.QueryTables(ctx, db, schema, `
		SELECT name, type FROM `+d.QuoteIdent(schema)+`.sqlite_master
		WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%'
		ORDER BY name`)
}

func (d Dialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
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
		c.Kind = d.TypeOf(c.Type).Kind
		if def.Valid {
			c.Default = &def.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
