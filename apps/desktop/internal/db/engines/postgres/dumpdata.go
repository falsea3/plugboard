package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (c *pgCatalog) root(oid uint32) uint32 {
	for {
		r, ok := c.byOid[oid]
		if !ok || !r.partBound.Valid || r.parent == 0 {
			return oid
		}
		oid = r.parent
	}
}

func (c *pgCatalog) dataOrder() ([]pgRel, []pgConstraint) {
	var roots []uint32
	seen := map[uint32]bool{}
	deps := map[uint32][]uint32{}
	var fks []pgConstraint
	for _, r := range c.rels {
		if r.kind != "r" && r.kind != "p" {
			continue
		}
		root := c.root(r.oid)
		if !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
		for _, x := range c.cons[r.oid] {
			if x.kind == "f" && x.local && !x.parent {
				deps[root] = append(deps[root], c.root(x.ref))
				fks = append(fks, x)
			}
		}
	}
	roots = topoOrder(roots, func(o uint32) uint32 { return o }, func(o uint32) []uint32 { return deps[o] })
	at := map[uint32]int{}
	for i, o := range roots {
		at[o] = i
	}
	var loose []pgConstraint
	for _, x := range fks {
		ref, ok := at[c.root(x.ref)]
		if ok && ref >= at[c.root(x.rel)] {
			loose = append(loose, x)
		}
	}
	var out []pgRel
	for _, o := range roots {
		for _, r := range c.rels {
			if r.kind == "r" && c.root(r.oid) == o {
				out = append(out, r)
			}
		}
	}
	return out, loose
}

type pgTrigger struct {
	rel     string
	name    string
	enabled string
}

func (d Dialect) enabledTriggers(ctx context.Context, q queryer, schemas []string) ([]pgTrigger, error) {
	return scanAll(ctx, q, func(r *sql.Rows) (pgTrigger, error) {
		var x pgTrigger
		var schema, rel string
		err := r.Scan(&schema, &rel, &x.name, &x.enabled)
		x.rel = d.QuoteIdent(schema) + "." + d.QuoteIdent(rel)
		return x, err
	}, `SELECT n.nspname, c.relname, t.tgname, t.tgenabled::text FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND c.relkind = 'r' AND NOT t.tgisinternal AND t.tgenabled <> 'D'
		ORDER BY n.nspname, c.relname, t.tgname`, schemas)
}

var enableTrigger = map[string]string{"A": "ENABLE ALWAYS TRIGGER", "R": "ENABLE REPLICA TRIGGER"}

func (d Dialect) dumpData(ctx context.Context, conn *sql.Conn, c *pgCatalog, schemas []string, opts model.DumpOptions, o *dumpOut) error {
	o.state.Phase = "data"
	rels, loose := c.dataOrder()
	var triggers []pgTrigger
	if !opts.Structure {
		var err error
		if triggers, err = d.enabledTriggers(ctx, conn, schemas); err != nil {
			return err
		}
		o.section("Foreign keys and triggers off while rows load")
		for _, x := range loose {
			o.line(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", d.relName(c.byOid[x.rel]), d.QuoteIdent(x.name)))
		}
		for _, t := range triggers {
			o.line(fmt.Sprintf("ALTER TABLE %s DISABLE TRIGGER %s;", t.rel, d.QuoteIdent(t.name)))
		}
	}
	o.section("Data")
	for _, r := range rels {
		if err := d.tableData(ctx, conn, c, r, opts, o); err != nil {
			return err
		}
	}
	for _, oid := range c.largeObjs {
		if err := d.largeObjectData(ctx, conn, oid, !opts.Structure, o); err != nil {
			return err
		}
	}
	if !opts.Structure {
		o.section("Foreign keys and triggers back on")
		for _, x := range loose {
			o.line(d.constraintSQL(c.byOid[x.rel], x))
		}
		for _, t := range triggers {
			verb, ok := enableTrigger[t.enabled]
			if !ok {
				verb = "ENABLE TRIGGER"
			}
			o.line(fmt.Sprintf("ALTER TABLE %s %s %s;", t.rel, verb, d.QuoteIdent(t.name)))
		}
		d.refreshViews(c, o)
	}
	return nil
}

func (d Dialect) tableData(ctx context.Context, conn *sql.Conn, c *pgCatalog, r pgRel, opts model.DumpOptions, o *dumpOut) error {
	name := d.relName(r)
	o.state.Object = r.schema + "." + r.name
	o.report()
	defer func() { o.state.Done++ }()
	cols := d.dataColumns(c.cols[r.oid])
	if len(cols) == 0 {
		return nil
	}
	var n int64
	var err error
	if opts.Inserts {
		n, err = d.insertsOut(ctx, conn, o, name, cols, c.alwaysIdentity(r.oid))
	} else {
		n, err = d.copyOut(ctx, conn, o, name, cols)
	}
	o.state.Rows += n
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
