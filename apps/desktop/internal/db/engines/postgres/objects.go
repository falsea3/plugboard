package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) ListObjects(ctx context.Context, db *sql.DB, schema string) ([]model.DBObject, error) {
	return dialect.QueryObjects(ctx, db, schema, `
		SELECT p.proname, CASE p.prokind WHEN 'p' THEN 'procedure' ELSE 'function' END,
		       pg_get_function_identity_arguments(p.oid), NULL::text, NULL::text
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = $1 AND p.prokind IN ('f', 'p')
		  AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_proc'::regclass AND d.objid = p.oid AND d.deptype = 'e')
		UNION ALL
		SELECT c.relname, 'sequence', NULL, NULL, NULL
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind = 'S'
		UNION ALL
		SELECT t.typname, 'enum', NULL,
		       (SELECT json_agg(e.enumlabel ORDER BY e.enumsortorder)::text FROM pg_enum e WHERE e.enumtypid = t.oid), NULL
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1 AND t.typtype = 'e'
		UNION ALL
		SELECT t.typname, 'domain', format_type(t.typbasetype, t.typtypmod), NULL, NULL
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1 AND t.typtype = 'd'
		UNION ALL
		SELECT tg.tgname, 'trigger', c.relname, NULL, NULL
		FROM pg_trigger tg
		JOIN pg_class c ON c.oid = tg.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND NOT tg.tgisinternal
		UNION ALL
		SELECT e.extname, 'extension', e.extversion, NULL, n.nspname
		FROM pg_extension e JOIN pg_namespace n ON n.oid = e.extnamespace
		ORDER BY 2, 1`, schema)
}

func (d Dialect) ObjectDDL(ctx context.Context, db *sql.DB, obj model.DBObject) (string, error) {
	name := d.QuoteIdent(obj.Schema) + "." + d.QuoteIdent(obj.Name)
	one := func(query string, args ...any) (string, error) {
		var s sql.NullString
		err := db.QueryRowContext(ctx, query, args...).Scan(&s)
		if err == nil && !s.Valid {
			return "", dialect.Gone(obj)
		}
		return s.String, dialect.NoRows(err, obj)
	}
	switch obj.Kind {
	case "table":
		return d.tableDDL(ctx, db, obj, name)
	case "view", "matview":
		var kind string
		var def sql.NullString
		err := db.QueryRowContext(ctx, `SELECT c.relkind::text, pg_get_viewdef(c.oid, true) FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = $1 AND c.relname = $2`, obj.Schema, obj.Name).Scan(&kind, &def)
		if err != nil {
			return "", dialect.NoRows(err, obj)
		}
		head := "CREATE OR REPLACE VIEW "
		if kind == "m" {
			head = "CREATE MATERIALIZED VIEW "
		}
		return dialect.Statement(head + name + " AS\n" + strings.TrimRight(def.String, "; \n")), nil
	case "function", "procedure":
		s, err := one(`SELECT pg_get_functiondef(p.oid) FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE n.nspname = $1 AND p.proname = $2 AND pg_get_function_identity_arguments(p.oid) = $3`, obj.Schema, obj.Name, obj.Detail)
		return dialect.Statement(s), err
	case "sequence":
		var typ string
		var start, min, max, inc, cache int64
		var cycle bool
		err := db.QueryRowContext(ctx, `SELECT format_type(s.seqtypid, NULL), s.seqstart, s.seqmin, s.seqmax, s.seqincrement, s.seqcache, s.seqcycle
			FROM pg_sequence s JOIN pg_class c ON c.oid = s.seqrelid JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2`, obj.Schema, obj.Name).Scan(&typ, &start, &min, &max, &inc, &cache, &cycle)
		if err != nil {
			return "", dialect.NoRows(err, obj)
		}
		cyc := "NO CYCLE"
		if cycle {
			cyc = "CYCLE"
		}
		return fmt.Sprintf("CREATE SEQUENCE %s\n  AS %s\n  INCREMENT BY %d\n  MINVALUE %d\n  MAXVALUE %d\n  START WITH %d\n  CACHE %d\n  %s;", name, typ, inc, min, max, start, cache, cyc), nil
	case "enum":
		labels, err := dialect.QueryLines(ctx, db, `SELECT e.enumlabel FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
			JOIN pg_namespace n ON n.oid = t.typnamespace WHERE n.nspname = $1 AND t.typname = $2 ORDER BY e.enumsortorder`, obj.Schema, obj.Name)
		if err != nil {
			return "", err
		}
		quoted := make([]string, len(labels))
		for i, l := range labels {
			quoted[i] = "  " + d.QuoteString(l)
		}
		return fmt.Sprintf("CREATE TYPE %s AS ENUM (\n%s\n);", name, strings.Join(quoted, ",\n")), nil
	case "domain":
		return d.domainDDL(ctx, db, obj, name)
	case "trigger":
		s, err := one(`SELECT pg_get_triggerdef(tg.oid, true) FROM pg_trigger tg JOIN pg_class c ON c.oid = tg.tgrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = $1 AND tg.tgname = $2 AND c.relname = $3`, obj.Schema, obj.Name, obj.Detail)
		return dialect.Statement(s), err
	case "extension":
		return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s\n  WITH SCHEMA %s\n  VERSION %s;", d.QuoteIdent(obj.Name), d.QuoteIdent(obj.Schema), d.QuoteString(obj.Detail)), nil
	}
	return "", fmt.Errorf("no DDL for a %s", obj.Kind)
}

func (d Dialect) domainDDL(ctx context.Context, db *sql.DB, obj model.DBObject, name string) (string, error) {
	var base string
	var def sql.NullString
	var notNull bool
	err := db.QueryRowContext(ctx, `SELECT format_type(t.typbasetype, t.typtypmod), t.typdefault, t.typnotnull
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace WHERE n.nspname = $1 AND t.typname = $2`, obj.Schema, obj.Name).Scan(&base, &def, &notNull)
	if err != nil {
		return "", dialect.NoRows(err, obj)
	}
	checks, err := dialect.QueryLines(ctx, db, `SELECT 'CONSTRAINT ' || quote_ident(c.conname) || ' ' || pg_get_constraintdef(c.oid, true)
		FROM pg_constraint c JOIN pg_type t ON t.oid = c.contypid JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1 AND t.typname = $2 ORDER BY c.conname`, obj.Schema, obj.Name)
	if err != nil {
		return "", err
	}
	out := "CREATE DOMAIN " + name + " AS " + base
	if def.Valid {
		out += "\n  DEFAULT " + def.String
	}
	if notNull {
		out += "\n  NOT NULL"
	}
	for _, c := range checks {
		out += "\n  " + c
	}
	return out + ";", nil
}
