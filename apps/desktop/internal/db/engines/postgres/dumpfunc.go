package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type pgFunc struct {
	oid   uint32
	name  string
	kind  string
	early bool
	late  bool
}

func (d Dialect) loadFunctions(ctx context.Context, q queryer, schemas []string, version int) ([]pgFunc, error) {
	atomic := "false"
	if version >= 140000 {
		atomic = "p.prosqlbody IS NOT NULL"
	}
	return scanAll(ctx, q, func(r *sql.Rows) (pgFunc, error) {
		var x pgFunc
		var schema, name, args string
		err := r.Scan(&x.oid, &schema, &name, &x.kind, &args, &x.early, &x.late)
		x.name = d.QuoteIdent(schema) + "." + d.QuoteIdent(name) + "(" + args + ")"
		return x, err
	}, `
		SELECT p.oid, n.nspname, p.proname, p.prokind::text, pg_get_function_identity_arguments(p.oid),
		       NOT (`+atomic+`) AND EXISTS (SELECT 1 FROM pg_depend dp WHERE dp.refclassid = 'pg_proc'::regclass AND dp.refobjid = p.oid
		         AND dp.classid IN ('pg_attrdef'::regclass, 'pg_constraint'::regclass, 'pg_type'::regclass)),
		       EXISTS (SELECT 1 FROM pg_depend dp JOIN pg_class vc ON vc.oid = dp.refobjid AND dp.refclassid = 'pg_class'::regclass AND vc.relkind IN ('v', 'm')
		         WHERE dp.classid = 'pg_proc'::regclass AND dp.objid = p.oid)
		       OR EXISTS (SELECT 1 FROM pg_depend dp JOIN pg_type vt ON vt.oid = dp.refobjid AND dp.refclassid = 'pg_type'::regclass
		         JOIN pg_class vc ON vc.oid = vt.typrelid AND vc.relkind IN ('v', 'm') WHERE dp.classid = 'pg_proc'::regclass AND dp.objid = p.oid)
		FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = ANY($1) AND `+noExt("pg_proc", "p.oid")+`
		  AND NOT EXISTS (SELECT 1 FROM pg_depend dp WHERE dp.classid = 'pg_proc'::regclass AND dp.objid = p.oid AND dp.deptype = 'i')
		ORDER BY n.nspname, p.proname, p.oid`, schemas)
}

func (d Dialect) functionDDL(ctx context.Context, q queryer, f pgFunc) (string, error) {
	if f.kind == "a" {
		return aggregateDDL(ctx, q, f)
	}
	var def string
	if err := q.QueryRowContext(ctx, `SELECT pg_get_functiondef($1)`, f.oid).Scan(&def); err != nil {
		return "", err
	}
	return strings.TrimRight(def, "\n; ") + ";", nil
}

func aggregateDDL(ctx context.Context, q queryer, f pgFunc) (string, error) {
	var kind, sfunc, stype string
	var final, initcond, combine, sortop sql.NullString
	err := q.QueryRowContext(ctx, `
		SELECT a.aggkind::text, a.aggtransfn::regproc::text, format_type(a.aggtranstype, NULL),
		       NULLIF(a.aggfinalfn::regproc::text, '-'), a.agginitval, NULLIF(a.aggcombinefn::regproc::text, '-'),
		       CASE WHEN a.aggsortop <> 0 THEN a.aggsortop::regoperator::text END
		FROM pg_aggregate a WHERE a.aggfnoid = $1`, f.oid).Scan(&kind, &sfunc, &stype, &final, &initcond, &combine, &sortop)
	if err != nil {
		return "", err
	}
	if kind != "n" {
		return "-- ordered-set aggregate " + f.name + " is not included", nil
	}
	parts := []string{"SFUNC = " + sfunc, "STYPE = " + stype}
	if final.Valid {
		parts = append(parts, "FINALFUNC = "+final.String)
	}
	if initcond.Valid {
		parts = append(parts, "INITCOND = "+quoteLiteral(initcond.String))
	}
	if combine.Valid {
		parts = append(parts, "COMBINEFUNC = "+combine.String)
	}
	if sortop.Valid {
		op := sortop.String
		if i := strings.IndexByte(op, '('); i > 0 {
			op = op[:i]
		}
		parts = append(parts, "SORTOP = OPERATOR("+op+")")
	}
	return fmt.Sprintf("CREATE AGGREGATE %s (\n    %s\n);", f.name, strings.Join(parts, ",\n    ")), nil
}

func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
