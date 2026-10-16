package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (d Dialect) sequenceValue(ctx context.Context, q queryer, s pgSequence) (string, error) {
	var last int64
	var called bool
	if err := q.QueryRowContext(ctx, "SELECT last_value, is_called FROM "+s.name).Scan(&last, &called); err != nil {
		return "", err
	}
	return fmt.Sprintf("SELECT pg_catalog.setval(%s, %d, %t);", quoteLiteral(s.name), last, called), nil
}

func (d Dialect) constraintSQL(r pgRel, c pgConstraint) string {
	only := "ONLY "
	if r.kind == "p" {
		only = ""
	}
	return fmt.Sprintf("ALTER TABLE %s%s ADD CONSTRAINT %s %s;", only, d.relName(r), d.QuoteIdent(c.name), c.def)
}

func indexSQL(r pgRel, def string) string {
	if r.kind == "p" {
		def = strings.Replace(def, " ON ONLY ", " ON ", 1)
	}
	return def + ";"
}

type pgView struct {
	rel  pgRel
	def  string
	deps []uint32
}

func (d Dialect) loadViews(ctx context.Context, q queryer, rels []pgRel) ([]pgView, error) {
	var views []pgView
	for _, r := range rels {
		if r.kind != "v" && r.kind != "m" {
			continue
		}
		var def string
		var deps string
		err := q.QueryRowContext(ctx, `SELECT pg_get_viewdef($1::oid),
			array_to_string(ARRAY(SELECT DISTINCT dp.refobjid FROM pg_rewrite rw JOIN pg_depend dp ON dp.classid = 'pg_rewrite'::regclass AND dp.objid = rw.oid
			  WHERE rw.ev_class = $1::oid AND dp.refclassid = 'pg_class'::regclass AND dp.refobjid <> $1::oid), ' ')`, r.oid).Scan(&def, &deps)
		if err != nil {
			return nil, err
		}
		v := pgView{rel: r, def: strings.TrimRight(strings.TrimSpace(def), ";")}
		for _, f := range strings.Fields(deps) {
			var oid uint32
			fmt.Sscan(f, &oid)
			v.deps = append(v.deps, oid)
		}
		views = append(views, v)
	}
	return topoOrder(views, func(v pgView) uint32 { return v.rel.oid }, func(v pgView) []uint32 { return v.deps }), nil
}

func (d Dialect) viewSQL(v pgView) string {
	name := d.relName(v.rel)
	with := ""
	if v.rel.options.Valid && v.rel.options.String != "" {
		with = " WITH (" + v.rel.options.String + ")"
	}
	if v.rel.kind == "m" {
		return fmt.Sprintf("CREATE MATERIALIZED VIEW %s%s AS\n%s\nWITH NO DATA;", name, with, v.def)
	}
	return fmt.Sprintf("CREATE VIEW %s%s AS\n%s;", name, with, v.def)
}

func (d Dialect) loadTriggers(ctx context.Context, q queryer, schemas []string, version int) ([]string, error) {
	inherited := ""
	if version >= 130000 {
		inherited = " AND t.tgparentid = 0"
	}
	return loadLines(ctx, q, `SELECT pg_get_triggerdef(t.oid) || ';' FROM pg_trigger t JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND NOT t.tgisinternal`+inherited+` AND `+noExt("pg_trigger", "t.oid")+`
		ORDER BY n.nspname, c.relname, t.tgname`, schemas)
}

func (d Dialect) loadRules(ctx context.Context, q queryer, schemas []string) ([]string, error) {
	return loadLines(ctx, q, `SELECT pg_get_ruledef(r.oid) FROM pg_rewrite r JOIN pg_class c ON c.oid = r.ev_class
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND r.rulename <> '_RETURN' AND `+noExt("pg_rewrite", "r.oid")+`
		ORDER BY n.nspname, c.relname, r.rulename`, schemas)
}

func (d Dialect) loadPolicies(ctx context.Context, q queryer, schemas []string) ([]string, error) {
	return loadLines(ctx, q, `SELECT 'CREATE POLICY ' || quote_ident(p.policyname) || ' ON ' || quote_ident(p.schemaname) || '.' || quote_ident(p.tablename)
		|| ' AS ' || p.permissive || ' FOR ' || p.cmd
		|| ' TO ' || array_to_string(ARRAY(SELECT CASE WHEN r = 'public' THEN 'PUBLIC' ELSE quote_ident(r) END FROM unnest(p.roles) r), ', ')
		|| COALESCE(' USING (' || p.qual || ')', '') || COALESCE(' WITH CHECK (' || p.with_check || ')', '') || ';'
		FROM pg_policies p WHERE p.schemaname = ANY($1) ORDER BY p.schemaname, p.tablename, p.policyname`, schemas)
}

var commentKinds = map[string]string{
	"table": "TABLE", "view": "VIEW", "materialized view": "MATERIALIZED VIEW", "sequence": "SEQUENCE", "index": "INDEX",
	"table column": "COLUMN", "view column": "COLUMN", "materialized view column": "COLUMN", "foreign table": "FOREIGN TABLE",
	"schema": "SCHEMA", "type": "TYPE", "domain": "DOMAIN", "function": "FUNCTION", "procedure": "PROCEDURE",
	"aggregate": "AGGREGATE", "table constraint": "CONSTRAINT", "trigger": "TRIGGER", "rule": "RULE", "policy": "POLICY",
	"composite type": "TYPE", "domain constraint": "CONSTRAINT", "large object": "LARGE OBJECT",
}

func (d Dialect) loadComments(ctx context.Context, q queryer, schemas []string, everything bool) ([]string, error) {
	rows, err := scanAll(ctx, q, func(r *sql.Rows) ([3]string, error) {
		var x [3]string
		err := r.Scan(&x[0], &x[1], &x[2])
		return x, err
	}, `SELECT o.type, o.identity, ds.description FROM pg_description ds
		CROSS JOIN LATERAL pg_identify_object(ds.classoid, ds.objoid, ds.objsubid) o
		WHERE (o.type = 'large object' AND $2 OR COALESCE(o.schema, CASE ds.classoid
		    WHEN 'pg_trigger'::regclass THEN (SELECT c.relnamespace::regnamespace::text FROM pg_trigger x JOIN pg_class c ON c.oid = x.tgrelid WHERE x.oid = ds.objoid)
		    WHEN 'pg_rewrite'::regclass THEN (SELECT c.relnamespace::regnamespace::text FROM pg_rewrite x JOIN pg_class c ON c.oid = x.ev_class WHERE x.oid = ds.objoid)
		    WHEN 'pg_policy'::regclass THEN (SELECT c.relnamespace::regnamespace::text FROM pg_policy x JOIN pg_class c ON c.oid = x.polrelid WHERE x.oid = ds.objoid)
		  END) = ANY(ARRAY(SELECT quote_ident(s) FROM unnest($1::text[]) s) || $1::text[]) OR (o.type = 'schema' AND o.identity = ANY(ARRAY(SELECT quote_ident(s) FROM unnest($1::text[]) s))))
		  AND NOT EXISTS (SELECT 1 FROM pg_depend e WHERE e.classid = ds.classoid AND e.objid = ds.objoid AND e.deptype = 'e')
		ORDER BY o.type, o.identity`, schemas, everything)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range rows {
		kind, ok := commentKinds[r[0]]
		if !ok {
			continue
		}
		identity := r[1]
		if r[0] == "domain constraint" {
			if i := strings.LastIndex(identity, " on "); i > 0 {
				identity = identity[:i] + " ON DOMAIN " + identity[i+4:]
			}
		}
		out = append(out, fmt.Sprintf("COMMENT ON %s %s IS %s;", kind, identity, quoteLiteral(r[2])))
	}
	return out, nil
}
