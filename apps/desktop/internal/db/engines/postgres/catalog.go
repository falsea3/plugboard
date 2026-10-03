package postgres

import (
	"context"
	"database/sql"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) DefaultSchema(ctx context.Context, db *sql.DB, _ model.Connection) (string, error) {
	var s sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&s); err != nil {
		return "", err
	}
	if !s.Valid {
		return "public", nil
	}
	return s.String, nil
}

func (Dialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	return dialect.QueryStrings(ctx, db, `
		SELECT nspname FROM pg_namespace
		WHERE nspname NOT IN ('pg_catalog', 'information_schema')
		  AND nspname NOT LIKE 'pg_toast%'
		  AND nspname NOT LIKE 'pg_temp_%'
		ORDER BY nspname`)
}

func (Dialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return dialect.QueryTables(ctx, db, schema, `
		SELECT c.relname,
		       CASE c.relkind WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' ELSE 'BASE TABLE' END
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
		ORDER BY c.relname`, schema)
}

func (Dialect) ListRelations(ctx context.Context, db *sql.DB, schema string) ([]model.Relation, error) {
	return dialect.QueryRelations(ctx, db, `
		SELECT con.conname, src.relname, a.attname, dn.nspname, dst.relname, fa.attname, con.confdeltype::text, con.confupdtype::text
		FROM pg_constraint con
		JOIN pg_class src ON src.oid = con.conrelid
		JOIN pg_namespace sn ON sn.oid = src.relnamespace
		JOIN pg_class dst ON dst.oid = con.confrelid
		JOIN pg_namespace dn ON dn.oid = dst.relnamespace
		CROSS JOIN LATERAL unnest(con.conkey, con.confkey) WITH ORDINALITY AS k(attnum, refnum, ord)
		JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.attnum
		JOIN pg_attribute fa ON fa.attrelid = con.confrelid AND fa.attnum = k.refnum
		WHERE con.contype = 'f' AND con.conparentid = 0 AND sn.nspname = $1
		ORDER BY src.relname, con.conname, k.ord`, schema)
}

func (d Dialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	cols, err := dialect.QueryColumns(ctx, db, `
		SELECT a.attname,
		       format_type(a.atttypid, a.atttypmod),
		       NOT a.attnotnull,
		       pg_get_expr(d.adbin, d.adrelid),
		       COALESCE(a.attnum = ANY(i.indkey), false),
		       (SELECT json_agg(e.enumlabel ORDER BY e.enumsortorder)::text FROM pg_enum e WHERE e.enumtypid = a.atttypid),
		       quote_ident(tn.nspname) || '.' || quote_ident(t.typname)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_type t ON t.oid = a.atttypid
		JOIN pg_namespace tn ON tn.oid = t.typnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		LEFT JOIN pg_index i ON i.indrelid = c.oid AND i.indisprimary
		WHERE n.nspname = $1 AND c.relname = $2 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, schema, table)
	for i := range cols {
		cols[i].Kind = d.TypeOf(cols[i].Type).Kind
	}
	return cols, err
}
