package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type pgType struct {
	oid    uint32
	schema string
	name   string
	kind   string
	deps   []uint32
	late   bool
}

func loadTypes(ctx context.Context, q queryer, schemas []string) ([]pgType, error) {
	types, err := scanAll(ctx, q, func(r *sql.Rows) (pgType, error) {
		var x pgType
		var deps string
		err := r.Scan(&x.oid, &x.schema, &x.name, &x.kind, &deps, &x.late)
		for _, d := range strings.Fields(deps) {
			var oid uint32
			fmt.Sscan(d, &oid)
			x.deps = append(x.deps, oid)
		}
		return x, err
	}, `
		SELECT t.oid, n.nspname, t.typname, t.typtype::text,
		       array_to_string(ARRAY(
		         SELECT COALESCE(NULLIF(rt.typelem, 0), rt.oid) FROM pg_type rt WHERE rt.oid IN (
		           SELECT t.typbasetype UNION ALL
		           SELECT a.atttypid FROM pg_attribute a WHERE a.attrelid = t.typrelid AND a.attnum > 0 AND NOT a.attisdropped UNION ALL
		           SELECT r.rngsubtype FROM pg_range r WHERE r.rngtypid = t.oid)), ' '),
		       EXISTS (SELECT 1 FROM pg_depend dp WHERE dp.refclassid = 'pg_proc'::regclass AND (dp.classid = 'pg_type'::regclass AND dp.objid = t.oid
		         OR dp.classid = 'pg_constraint'::regclass AND dp.objid IN (SELECT co.oid FROM pg_constraint co WHERE co.contypid = t.oid)))
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = ANY($1) AND `+noExt("pg_type", "t.oid")+`
		  AND (t.typtype IN ('e', 'd', 'r') OR t.typtype = 'c' AND (SELECT c.relkind FROM pg_class c WHERE c.oid = t.typrelid) = 'c')
		ORDER BY n.nspname, t.typname`, schemas)
	if err != nil {
		return nil, err
	}
	rank := map[string]int{"e": 0, "r": 1, "d": 2, "c": 2}
	ranked := make([]pgType, 0, len(types))
	for r := 0; r <= 2; r++ {
		for _, t := range types {
			if rank[t.kind] == r {
				ranked = append(ranked, t)
			}
		}
	}
	return topoOrder(ranked, func(t pgType) uint32 { return t.oid }, func(t pgType) []uint32 { return t.deps }), nil
}

func (d Dialect) typeDDL(ctx context.Context, q queryer, t pgType) (string, error) {
	name := d.QuoteIdent(t.schema) + "." + d.QuoteIdent(t.name)
	switch t.kind {
	case "e":
		labels, err := loadLines(ctx, q, `SELECT quote_literal(enumlabel) FROM pg_enum WHERE enumtypid = $1 ORDER BY enumsortorder`, t.oid)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CREATE TYPE %s AS ENUM (\n    %s\n);", name, strings.Join(labels, ",\n    ")), nil
	case "d":
		var base string
		var def, coll sql.NullString
		var notNull bool
		err := q.QueryRowContext(ctx, `SELECT format_type(t.typbasetype, t.typtypmod), t.typdefault, t.typnotnull,
			CASE WHEN t.typcollation <> b.typcollation AND t.typcollation <> 0 THEN t.typcollation::regcollation::text END
			FROM pg_type t JOIN pg_type b ON b.oid = t.typbasetype WHERE t.oid = $1`, t.oid).Scan(&base, &def, &notNull, &coll)
		if err != nil {
			return "", err
		}
		checks, err := loadLines(ctx, q, `SELECT 'CONSTRAINT ' || quote_ident(conname) || ' ' || pg_get_constraintdef(oid)
			FROM pg_constraint WHERE contypid = $1 AND contype = 'c' ORDER BY conname`, t.oid)
		if err != nil {
			return "", err
		}
		out := "CREATE DOMAIN " + name + " AS " + base
		if coll.Valid {
			out += " COLLATE " + coll.String
		}
		if def.Valid {
			out += " DEFAULT " + def.String
		}
		if notNull {
			out += " NOT NULL"
		}
		for _, c := range checks {
			out += "\n    " + c
		}
		return out + ";", nil
	case "c":
		attrs, err := loadLines(ctx, q, `SELECT quote_ident(a.attname) || ' ' || format_type(a.atttypid, a.atttypmod)
			|| CASE WHEN a.attcollation <> bt.typcollation AND a.attcollation <> 0 THEN ' COLLATE ' || a.attcollation::regcollation::text ELSE '' END
			FROM pg_type t JOIN pg_attribute a ON a.attrelid = t.typrelid JOIN pg_type bt ON bt.oid = a.atttypid
			WHERE t.oid = $1 AND a.attnum > 0 AND NOT a.attisdropped ORDER BY a.attnum`, t.oid)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CREATE TYPE %s AS (\n    %s\n);", name, strings.Join(attrs, ",\n    ")), nil
	case "r":
		var parts []string
		err := q.QueryRowContext(ctx, `SELECT 'SUBTYPE = ' || format_type(r.rngsubtype, NULL),
			CASE WHEN NOT oc.opcdefault THEN 'SUBTYPE_OPCLASS = ' || quote_ident(ocn.nspname) || '.' || quote_ident(oc.opcname) ELSE '' END,
			CASE WHEN r.rngcollation <> 0 AND r.rngcollation <> st.typcollation THEN 'COLLATION = ' || r.rngcollation::regcollation::text ELSE '' END,
			CASE WHEN r.rngcanonical <> 0 THEN 'CANONICAL = ' || r.rngcanonical::regproc::text ELSE '' END,
			CASE WHEN r.rngsubdiff <> 0 THEN 'SUBTYPE_DIFF = ' || r.rngsubdiff::regproc::text ELSE '' END
			FROM pg_range r JOIN pg_type st ON st.oid = r.rngsubtype JOIN pg_opclass oc ON oc.oid = r.rngsubopc
			JOIN pg_namespace ocn ON ocn.oid = oc.opcnamespace WHERE r.rngtypid = $1`, t.oid).Scan(scanStrings(&parts, 5)...)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CREATE TYPE %s AS RANGE (\n    %s\n);", name, strings.Join(nonEmpty(parts), ",\n    ")), nil
	}
	return "", fmt.Errorf("no DDL for type kind %s", t.kind)
}

func scanStrings(out *[]string, n int) []any {
	*out = make([]string, n)
	dest := make([]any, n)
	for i := range dest {
		dest[i] = &(*out)[i]
	}
	return dest
}

func nonEmpty(parts []string) []string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

type pgSequence struct {
	oid    uint32
	name   string
	ddl    string
	owned  sql.NullString
	ident  bool
	opts   string
	schema string
	rel    string
}

func (d Dialect) loadSequences(ctx context.Context, q queryer, schemas []string) ([]pgSequence, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgSequence, error) {
		var x pgSequence
		var schema, rel, typ string
		var start, min, max, inc, cache int64
		var cycle bool
		var ownerSchema, ownerRel, ownerCol sql.NullString
		err := r.Scan(&x.oid, &schema, &rel, &typ, &start, &min, &max, &inc, &cache, &cycle, &ownerSchema, &ownerRel, &ownerCol, &x.ident)
		x.name = d.QuoteIdent(schema) + "." + d.QuoteIdent(rel)
		x.schema, x.rel = schema, rel
		if ownerCol.Valid {
			x.owned = sql.NullString{String: d.QuoteIdent(ownerSchema.String) + "." + d.QuoteIdent(ownerRel.String) + "." + d.QuoteIdent(ownerCol.String), Valid: true}
		}
		cyc := "NO CYCLE"
		if cycle {
			cyc = "CYCLE"
		}
		x.opts = fmt.Sprintf("SEQUENCE NAME %s START WITH %d INCREMENT BY %d MINVALUE %d MAXVALUE %d CACHE %d %s", x.name, start, inc, min, max, cache, cyc)
		x.ddl = fmt.Sprintf("CREATE SEQUENCE %s\n    AS %s\n    START WITH %d\n    INCREMENT BY %d\n    MINVALUE %d\n    MAXVALUE %d\n    CACHE %d\n    %s;", x.name, typ, start, inc, min, max, cache, cyc)
		return x, err
	}, `
		SELECT c.oid, n.nspname, c.relname, format_type(s.seqtypid, NULL), s.seqstart, s.seqmin, s.seqmax, s.seqincrement, s.seqcache, s.seqcycle,
		       o.nspname, o.relname, o.attname,
		       EXISTS (SELECT 1 FROM pg_depend dp WHERE dp.classid = 'pg_class'::regclass AND dp.objid = c.oid AND dp.deptype = 'i')
		FROM pg_sequence s JOIN pg_class c ON c.oid = s.seqrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN LATERAL (SELECT tn.nspname, tc.relname, a.attname
		  FROM pg_depend dp JOIN pg_class tc ON tc.oid = dp.refobjid JOIN pg_namespace tn ON tn.oid = tc.relnamespace
		  JOIN pg_attribute a ON a.attrelid = dp.refobjid AND a.attnum = dp.refobjsubid
		  WHERE dp.classid = 'pg_class'::regclass AND dp.objid = c.oid AND dp.refclassid = 'pg_class'::regclass AND dp.deptype IN ('a', 'i') LIMIT 1) o ON true
		WHERE n.nspname = ANY($1) AND `+noExt("pg_class", "c.oid")+`
		ORDER BY n.nspname, c.relname`, schemas)
}
