package sqlite

import (
	"context"
	"database/sql"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
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

func (d Dialect) ListRelations(ctx context.Context, db *sql.DB, schema string) ([]model.Relation, error) {
	return dialect.QueryRelations(ctx, db, `
		SELECT CAST(f.id AS TEXT), m.name, f."from", ?, f."table",
		       COALESCE(f."to", (SELECT p.name FROM pragma_table_info(f."table", ?) p WHERE p.pk = f.seq + 1), ''),
		       f.on_delete, f.on_update
		FROM `+d.QuoteIdent(schema)+`.sqlite_master m
		JOIN pragma_foreign_key_list(m.name, ?) f
		WHERE m.type = 'table'
		ORDER BY m.name, f.id, f.seq`, schema, schema, schema)
}

func (d Dialect) ListIndexes(ctx context.Context, db *sql.DB, schema, table string) ([]model.Index, error) {
	return dialect.QueryIndexes(ctx, db, `
		SELECT il.name, il."unique", il.origin = 'pk', 'btree', NULL,
		       (SELECT m.sql FROM `+d.QuoteIdent(schema)+`.sqlite_master m WHERE m.type = 'index' AND m.name = il.name),
		       COALESCE(ii.name, '(expression)')
		FROM pragma_index_list(?, ?) il
		JOIN pragma_index_info(il.name, ?) ii
		ORDER BY il.origin = 'pk' DESC, il.name, ii.seqno`, table, schema, schema)
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
