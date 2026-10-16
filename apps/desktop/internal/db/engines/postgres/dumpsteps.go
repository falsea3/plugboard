package postgres

import (
	"context"
	"sort"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (d Dialect) dumpAfter(ctx context.Context, q queryer, c *pgCatalog, schemas []string, version int, opts model.DumpOptions, o *dumpOut) error {
	o.state.Phase = "constraints"
	o.report()
	if opts.Data {
		o.section("Sequence values")
		if err := d.dumpSequenceValues(ctx, q, c, o); err != nil {
			return err
		}
	}
	o.section("Keys, indexes and foreign keys")
	for _, r := range c.rels {
		for _, x := range c.cons[r.oid] {
			if x.local && !x.parent && x.kind != "f" && !inlineCheck(x) {
				o.line(d.constraintSQL(r, x))
			}
		}
	}
	for _, ix := range c.indexes {
		if r := c.byOid[ix.rel]; r.kind == "r" || r.kind == "p" {
			o.line(indexSQL(r, ix.def))
		}
	}
	for _, r := range c.rels {
		for _, x := range c.cons[r.oid] {
			if x.local && !x.parent && x.kind == "f" {
				o.line(d.constraintSQL(r, x))
			}
		}
	}
	funcs := append([]pgFunc(nil), c.funcs...)
	sort.SliceStable(funcs, func(i, j int) bool { return funcs[i].kind != "a" && funcs[j].kind == "a" })
	o.section("Functions and views")
	for _, f := range funcs {
		if !f.early && !f.late {
			if err := d.emitFunc(ctx, q, f, o); err != nil {
				return err
			}
		}
	}
	for _, v := range c.views {
		o.line(d.viewSQL(v))
	}
	for _, f := range funcs {
		if !f.early && f.late {
			if err := d.emitFunc(ctx, q, f, o); err != nil {
				return err
			}
		}
	}
	for _, ix := range c.indexes {
		if r := c.byOid[ix.rel]; r.kind == "m" {
			o.line(indexSQL(r, ix.def))
		}
	}
	return d.dumpLast(ctx, q, c, schemas, version, opts, o)
}

func (d Dialect) dumpLast(ctx context.Context, q queryer, c *pgCatalog, schemas []string, version int, opts model.DumpOptions, o *dumpOut) error {
	o.section("Triggers, rules and policies")
	for _, load := range []func() ([]string, error){
		func() ([]string, error) { return d.loadTriggers(ctx, q, schemas, version) },
		func() ([]string, error) { return d.loadRules(ctx, q, schemas) },
	} {
		lines, err := load()
		if err != nil {
			return err
		}
		for _, l := range lines {
			o.line(terminated(l))
		}
	}
	for _, r := range c.rels {
		if r.rowSecurity {
			o.line("ALTER TABLE " + d.relName(r) + " ENABLE ROW LEVEL SECURITY;")
		}
		if r.forceRLS {
			o.line("ALTER TABLE " + d.relName(r) + " FORCE ROW LEVEL SECURITY;")
		}
	}
	policies, err := d.loadPolicies(ctx, q, schemas)
	if err != nil {
		return err
	}
	for _, p := range policies {
		o.line(p)
	}
	comments, err := d.loadComments(ctx, q, schemas, len(opts.Schemas) == 0)
	if err != nil {
		return err
	}
	if len(comments) > 0 {
		o.section("Comments")
		for _, cm := range comments {
			o.line(cm)
		}
	}
	if opts.Data {
		d.refreshViews(c, o)
	}
	if opts.Owners {
		return d.dumpOwners(ctx, q, schemas, opts, o)
	}
	return nil
}

func (d Dialect) refreshViews(c *pgCatalog, o *dumpOut) {
	for _, v := range c.views {
		if v.rel.kind == "m" && v.rel.populated {
			o.line("REFRESH MATERIALIZED VIEW " + d.relName(v.rel) + ";")
		}
	}
}

func terminated(s string) string {
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] != ';' {
		s += ";"
	}
	return s
}
