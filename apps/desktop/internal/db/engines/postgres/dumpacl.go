package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

const grantLines = `array_to_string(ARRAY(SELECT CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END
	|| E'\t' || a.is_grantable::text || E'\t' || string_agg(a.privilege_type, ', ' ORDER BY a.privilege_type)
	FROM aclexplode(%s) a GROUP BY a.grantee, a.is_grantable ORDER BY 1), E'\n')`

func grants(acl string) string { return fmt.Sprintf(grantLines, acl) }

type pgACL struct {
	alter  string
	on     string
	name   string
	owner  string
	set    bool
	grants string
}

func (d Dialect) loadACLs(ctx context.Context, q queryer, schemas []string) ([]pgACL, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgACL, error) {
		var x pgACL
		err := r.Scan(&x.alter, &x.on, &x.name, &x.owner, &x.set, &x.grants)
		return x, err
	}, `
		SELECT 'SCHEMA', 'SCHEMA', quote_ident(n.nspname), quote_ident(pg_get_userbyid(n.nspowner)), n.nspacl IS NOT NULL, `+grants("n.nspacl")+`
		FROM pg_namespace n WHERE n.nspname = ANY($1)
		UNION ALL
		SELECT CASE c.relkind WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' WHEN 'S' THEN 'SEQUENCE' ELSE 'TABLE' END,
		       CASE c.relkind WHEN 'S' THEN 'SEQUENCE' ELSE 'TABLE' END,
		       quote_ident(n.nspname) || '.' || quote_ident(c.relname), quote_ident(pg_get_userbyid(c.relowner)), c.relacl IS NOT NULL, `+grants("c.relacl")+`
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND c.relkind IN ('r', 'p', 'v', 'm', 'S') AND `+noExt("pg_class", "c.oid")+`
		  AND (c.relkind <> 'S' OR NOT EXISTS (SELECT 1 FROM pg_depend dp WHERE dp.classid = 'pg_class'::regclass AND dp.objid = c.oid AND dp.deptype = 'i'))
		UNION ALL
		SELECT CASE p.prokind WHEN 'a' THEN 'AGGREGATE' WHEN 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END,
		       CASE p.prokind WHEN 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END,
		       quote_ident(n.nspname) || '.' || quote_ident(p.proname) || '(' || pg_get_function_identity_arguments(p.oid) || ')',
		       quote_ident(pg_get_userbyid(p.proowner)), p.proacl IS NOT NULL, `+grants("p.proacl")+`
		FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = ANY($1) AND `+noExt("pg_proc", "p.oid")+`
		  AND NOT EXISTS (SELECT 1 FROM pg_depend dp WHERE dp.classid = 'pg_proc'::regclass AND dp.objid = p.oid AND dp.deptype = 'i')
		UNION ALL
		SELECT CASE t.typtype WHEN 'd' THEN 'DOMAIN' ELSE 'TYPE' END, CASE t.typtype WHEN 'd' THEN 'DOMAIN' ELSE 'TYPE' END,
		       quote_ident(n.nspname) || '.' || quote_ident(t.typname), quote_ident(pg_get_userbyid(t.typowner)), t.typacl IS NOT NULL, `+grants("t.typacl")+`
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = ANY($1) AND `+noExt("pg_type", "t.oid")+`
		  AND (t.typtype IN ('e', 'd', 'r') OR t.typtype = 'c' AND (SELECT c.relkind FROM pg_class c WHERE c.oid = t.typrelid) = 'c')
		ORDER BY 1, 3`, schemas)
}

func (d Dialect) loadColumnGrants(ctx context.Context, q queryer, schemas []string) ([]string, error) {
	return loadLines(ctx, q, `
		SELECT 'GRANT ' || string_agg(a.privilege_type || ' (' || quote_ident(at.attname) || ')', ', ' ORDER BY a.privilege_type)
		       || ' ON TABLE ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname)
		       || ' TO ' || CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END
		       || CASE WHEN a.is_grantable THEN ' WITH GRANT OPTION' ELSE '' END || ';'
		FROM pg_attribute at JOIN pg_class c ON c.oid = at.attrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		CROSS JOIN LATERAL aclexplode(at.attacl) a
		WHERE n.nspname = ANY($1) AND at.attacl IS NOT NULL AND at.attnum > 0 AND NOT at.attisdropped
		GROUP BY n.nspname, c.relname, at.attnum, a.grantee, a.is_grantable
		ORDER BY 1`, schemas)
}

var defaultKinds = map[string]string{"r": "TABLES", "S": "SEQUENCES", "f": "FUNCTIONS", "T": "TYPES", "n": "SCHEMAS"}

func (d Dialect) loadDefaultACLs(ctx context.Context, q queryer, schemas []string, everything bool) ([]string, error) {
	type row struct{ owner, schema, kind, grants string }
	rows, err := scanAll(ctx, q, func(r *sql.Rows) (row, error) {
		var x row
		err := r.Scan(&x.owner, &x.schema, &x.kind, &x.grants)
		return x, err
	}, `SELECT quote_ident(pg_get_userbyid(da.defaclrole)), COALESCE(quote_ident(n.nspname), ''), da.defaclobjtype::text, `+grants("da.defaclacl")+`
		FROM pg_default_acl da LEFT JOIN pg_namespace n ON n.oid = da.defaclnamespace
		WHERE n.nspname = ANY($1) OR (da.defaclnamespace = 0 AND $2)
		ORDER BY 1, 2, 3`, schemas, everything)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range rows {
		head := "ALTER DEFAULT PRIVILEGES FOR ROLE " + r.owner
		if r.schema != "" {
			head += " IN SCHEMA " + r.schema
		}
		on := defaultKinds[r.kind]
		if r.schema == "" {
			out = append(out, fmt.Sprintf("%s REVOKE ALL ON %s FROM PUBLIC;", head, on), fmt.Sprintf("%s REVOKE ALL ON %s FROM %s;", head, on, r.owner))
		}
		for _, g := range grantList(r.grants) {
			out = append(out, fmt.Sprintf("%s GRANT %s ON %s TO %s%s;", head, g[2], on, g[0], withGrant(g[1])))
		}
	}
	return out, nil
}

func grantList(lines string) [][]string {
	var out [][]string
	for _, l := range strings.Split(lines, "\n") {
		if f := strings.Split(l, "\t"); len(f) == 3 {
			out = append(out, f)
		}
	}
	return out
}

func withGrant(grantable string) string {
	if grantable == "true" {
		return " WITH GRANT OPTION"
	}
	return ""
}

func (d Dialect) dumpOwners(ctx context.Context, q queryer, schemas []string, opts model.DumpOptions, o *dumpOut) error {
	acls, err := d.loadACLs(ctx, q, schemas)
	if err != nil {
		return err
	}
	if len(opts.Schemas) == 0 {
		los, err := d.largeObjectACLs(ctx, q)
		if err != nil {
			return err
		}
		acls = append(acls, los...)
	}
	o.section("Owners and privileges")
	for _, a := range acls {
		o.line(fmt.Sprintf("ALTER %s %s OWNER TO %s;", a.alter, a.name, a.owner))
	}
	for _, a := range acls {
		if !a.set {
			continue
		}
		lines := []string{
			fmt.Sprintf("REVOKE ALL ON %s %s FROM PUBLIC;", a.on, a.name),
			fmt.Sprintf("REVOKE ALL ON %s %s FROM %s;", a.on, a.name, a.owner),
		}
		for _, g := range grantList(a.grants) {
			lines = append(lines, fmt.Sprintf("GRANT %s ON %s %s TO %s%s;", g[2], a.on, a.name, g[0], withGrant(g[1])))
		}
		o.line(strings.Join(lines, "\n"))
	}
	for _, load := range []func() ([]string, error){
		func() ([]string, error) { return d.loadColumnGrants(ctx, q, schemas) },
		func() ([]string, error) { return d.loadDefaultACLs(ctx, q, schemas, len(opts.Schemas) == 0) },
	} {
		lines, err := load()
		if err != nil {
			return err
		}
		for _, l := range lines {
			o.line(l)
		}
	}
	return nil
}
