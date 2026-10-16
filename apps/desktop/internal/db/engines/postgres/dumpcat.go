package postgres

import (
	"context"
	"database/sql"
	"strings"
)

const notExtension = `NOT EXISTS (SELECT 1 FROM pg_depend e WHERE e.classid = %s::regclass AND e.objid = %s AND e.deptype = 'e')`

func noExt(catalog, oid string) string {
	return strings.NewReplacer("%s::regclass", "'"+catalog+"'::regclass", "%s", oid).Replace(notExtension)
}

type pgRel struct {
	oid         uint32
	schema      string
	name        string
	kind        string
	unlogged    bool
	options     sql.NullString
	partKey     sql.NullString
	partBound   sql.NullString
	parent      uint32
	rowSecurity bool
	forceRLS    bool
	populated   bool
}

type pgCol struct {
	name      string
	typ       string
	collation sql.NullString
	def       sql.NullString
	notNull   bool
	identity  string
	generated string
	local     bool
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanAll[T any](ctx context.Context, q queryer, scan func(*sql.Rows) (T, error), query string, args ...any) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func loadRels(ctx context.Context, q queryer, schemas []string) ([]pgRel, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgRel, error) {
		var x pgRel
		var parent sql.NullInt64
		err := r.Scan(&x.oid, &x.schema, &x.name, &x.kind, &x.unlogged, &x.options, &x.partKey, &x.partBound, &parent, &x.rowSecurity, &x.forceRLS, &x.populated)
		x.parent = uint32(parent.Int64)
		return x, err
	}, `
		SELECT c.oid, n.nspname, c.relname, c.relkind::text, c.relpersistence = 'u',
		       array_to_string(c.reloptions, ', '),
		       CASE WHEN c.relkind = 'p' THEN pg_get_partkeydef(c.oid) END,
		       CASE WHEN c.relispartition THEN pg_get_expr(c.relpartbound, c.oid) END,
		       (SELECT i.inhparent FROM pg_inherits i WHERE i.inhrelid = c.oid ORDER BY i.inhseqno LIMIT 1),
		       c.relrowsecurity, c.relforcerowsecurity, c.relispopulated
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND c.relkind IN ('r', 'p', 'v', 'm') AND `+noExt("pg_class", "c.oid")+`
		ORDER BY n.nspname, c.relname`, schemas)
}

func loadColumns(ctx context.Context, q queryer, schemas []string) (map[uint32][]pgCol, error) {
	type row struct {
		rel uint32
		col pgCol
	}
	rows, err := scanAll(ctx, q, func(r *sql.Rows) (row, error) {
		var x row
		err := r.Scan(&x.rel, &x.col.name, &x.col.typ, &x.col.collation, &x.col.def, &x.col.notNull, &x.col.identity, &x.col.generated, &x.col.local)
		return x, err
	}, `
		SELECT a.attrelid, a.attname, format_type(a.atttypid, a.atttypmod),
		       CASE WHEN a.attcollation <> t.typcollation AND a.attcollation <> 0 THEN (SELECT quote_ident(cn.nspname) || '.' || quote_ident(co.collname)
		         FROM pg_collation co JOIN pg_namespace cn ON cn.oid = co.collnamespace WHERE co.oid = a.attcollation) END,
		       CASE WHEN a.attgenerated = '' THEN pg_get_expr(ad.adbin, 0) ELSE pg_get_expr(ad.adbin, ad.adrelid) END,
		       a.attnotnull, a.attidentity::text, a.attgenerated::text, a.attislocal
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_type t ON t.oid = a.atttypid
		LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
		WHERE n.nspname = ANY($1) AND c.relkind IN ('r', 'p', 'v', 'm', 'c') AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attrelid, a.attnum`, schemas)
	if err != nil {
		return nil, err
	}
	out := map[uint32][]pgCol{}
	for _, r := range rows {
		out[r.rel] = append(out[r.rel], r.col)
	}
	return out, nil
}

type pgConstraint struct {
	rel    uint32
	name   string
	kind   string
	def    string
	local  bool
	parent bool
	ref    uint32
}

func loadConstraints(ctx context.Context, q queryer, schemas []string) ([]pgConstraint, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgConstraint, error) {
		var x pgConstraint
		err := r.Scan(&x.rel, &x.name, &x.kind, &x.def, &x.local, &x.parent, &x.ref)
		return x, err
	}, `
		SELECT co.conrelid, co.conname, co.contype::text, pg_get_constraintdef(co.oid), co.conislocal, co.conparentid <> 0, co.confrelid
		FROM pg_constraint co JOIN pg_class c ON c.oid = co.conrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND co.contype IN ('p', 'u', 'c', 'f', 'x') AND c.relkind IN ('r', 'p')
		ORDER BY co.conrelid, CASE co.contype WHEN 'p' THEN 0 WHEN 'u' THEN 1 WHEN 'x' THEN 2 WHEN 'c' THEN 3 ELSE 4 END, co.conname`, schemas)
}

type pgIndex struct {
	rel uint32
	def string
}

func loadIndexes(ctx context.Context, q queryer, schemas []string) ([]pgIndex, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgIndex, error) {
		var x pgIndex
		err := r.Scan(&x.rel, &x.def)
		return x, err
	}, `
		SELECT i.indrelid, pg_get_indexdef(i.indexrelid)
		FROM pg_index i JOIN pg_class c ON c.oid = i.indrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_class ic ON ic.oid = i.indexrelid
		WHERE n.nspname = ANY($1) AND c.relkind IN ('r', 'p', 'm')
		  AND NOT EXISTS (SELECT 1 FROM pg_constraint co WHERE co.conindid = i.indexrelid AND co.contype IN ('p', 'u', 'x'))
		  AND NOT EXISTS (SELECT 1 FROM pg_inherits h WHERE h.inhrelid = i.indexrelid)
		  AND `+noExt("pg_class", "ic.oid")+`
		ORDER BY n.nspname, c.relname, ic.relname`, schemas)
}

func loadLines(ctx context.Context, q queryer, query string, args ...any) ([]string, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (string, error) {
		var s string
		err := r.Scan(&s)
		return s, err
	}, query, args...)
}
