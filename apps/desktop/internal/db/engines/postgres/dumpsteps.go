package postgres

import (
	"context"
	"sort"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (d Dialect) dumpAfter(ctx context.Context, q queryer, c *pgCatalog, schemas []string, version int, opts model.DumpOptions, o *dumpOut) error {
	o.state.Phase = "constraints"
	o.report()
	if opts.Data {
		if err := d.dumpSequenceValues(ctx, q, c, o); err != nil {
			return err
		}
	}
	o.sec = sectionPostData
	for _, r := range c.rels {
		for _, x := range c.cons[r.oid] {
			if x.local && !x.parent && x.kind != "f" && !inlineCheck(x) {
				o.obj(constraintDesc[x.kind], r.schema, r.name+" "+x.name, d.constraintSQL(r, x))
			}
		}
	}
	for _, ix := range c.indexes {
		if r := c.byOid[ix.rel]; r.kind == "r" || r.kind == "p" {
			o.obj("INDEX", r.schema, indexName(ix.def), indexSQL(r, ix.def))
		}
	}
	for _, r := range c.rels {
		for _, x := range c.cons[r.oid] {
			if x.local && !x.parent && x.kind == "f" {
				o.obj("FK CONSTRAINT", r.schema, r.name+" "+x.name, d.constraintSQL(r, x))
			}
		}
	}
	funcs := append([]pgFunc(nil), c.funcs...)
	sort.SliceStable(funcs, func(i, j int) bool { return funcs[i].kind != "a" && funcs[j].kind == "a" })
	o.sec = sectionPreData
	for _, f := range funcs {
		if !f.early && !f.late {
			if err := d.emitFunc(ctx, q, f, o); err != nil {
				return err
			}
		}
	}
	for _, v := range c.views {
		desc := "VIEW"
		if v.rel.kind == "m" {
			desc = "MATERIALIZED VIEW"
		}
		o.obj(desc, v.rel.schema, v.rel.name, d.viewSQL(v))
	}
	for _, f := range funcs {
		if !f.early && f.late {
			if err := d.emitFunc(ctx, q, f, o); err != nil {
				return err
			}
		}
	}
	o.sec = sectionPostData
	for _, ix := range c.indexes {
		if r := c.byOid[ix.rel]; r.kind == "m" {
			o.obj("INDEX", r.schema, indexName(ix.def), indexSQL(r, ix.def))
		}
	}
	return d.dumpLast(ctx, q, c, schemas, version, opts, o)
}

func (d Dialect) dumpLast(ctx context.Context, q queryer, c *pgCatalog, schemas []string, version int, opts model.DumpOptions, o *dumpOut) error {
	for _, step := range []struct {
		desc string
		load func() ([]string, error)
	}{
		{"TRIGGER", func() ([]string, error) { return d.loadTriggers(ctx, q, schemas, version) }},
		{"RULE", func() ([]string, error) { return d.loadRules(ctx, q, schemas) }},
	} {
		lines, err := step.load()
		if err != nil {
			return err
		}
		for _, l := range lines {
			o.obj(step.desc, "", "", terminated(l))
		}
	}
	for _, r := range c.rels {
		if r.rowSecurity {
			o.obj("ROW SECURITY", r.schema, r.name, "ALTER TABLE "+d.relName(r)+" ENABLE ROW LEVEL SECURITY;")
		}
		if r.forceRLS {
			o.obj("ROW SECURITY", r.schema, r.name, "ALTER TABLE "+d.relName(r)+" FORCE ROW LEVEL SECURITY;")
		}
	}
	policies, err := d.loadPolicies(ctx, q, schemas)
	if err != nil {
		return err
	}
	for _, p := range policies {
		o.obj("POLICY", "", "", p)
	}
	comments, err := d.loadComments(ctx, q, schemas, len(opts.Schemas) == 0)
	if err != nil {
		return err
	}
	o.sec = sectionNone
	for _, cm := range comments {
		o.obj("COMMENT", "", "", cm)
	}
	o.sec = sectionPostData
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
			o.obj("MATERIALIZED VIEW DATA", v.rel.schema, v.rel.name, "REFRESH MATERIALIZED VIEW "+d.relName(v.rel)+";")
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

var constraintDesc = map[string]string{"p": "CONSTRAINT", "u": "CONSTRAINT", "x": "CONSTRAINT", "c": "CHECK CONSTRAINT"}

func indexName(def string) string {
	_, rest, _ := strings.Cut(def, " INDEX ")
	name, _, _ := strings.Cut(rest, " ON ")
	return name
}
