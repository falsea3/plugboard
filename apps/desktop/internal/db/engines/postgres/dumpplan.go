package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

type pgCatalog struct {
	rels       []pgRel
	byOid      map[uint32]pgRel
	cols       map[uint32][]pgCol
	cons       map[uint32][]pgConstraint
	indexes    []pgIndex
	types      []pgType
	funcs      []pgFunc
	sequences  []pgSequence
	views      []pgView
	extensions []string
	largeObjs  []uint32
}

func (d Dialect) loadCatalog(ctx context.Context, q queryer, schemas []string, version int) (*pgCatalog, error) {
	c := &pgCatalog{byOid: map[uint32]pgRel{}, cons: map[uint32][]pgConstraint{}}
	rels, err := loadRels(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	c.rels = topoOrder(rels, func(r pgRel) uint32 { return r.oid }, func(r pgRel) []uint32 { return []uint32{r.parent} })
	for _, r := range c.rels {
		c.byOid[r.oid] = r
	}
	if c.cols, err = loadColumns(ctx, q, schemas); err != nil {
		return nil, err
	}
	cons, err := loadConstraints(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	for _, x := range cons {
		c.cons[x.rel] = append(c.cons[x.rel], x)
	}
	if c.indexes, err = loadIndexes(ctx, q, schemas); err != nil {
		return nil, err
	}
	if c.types, err = loadTypes(ctx, q, schemas); err != nil {
		return nil, err
	}
	if c.funcs, err = d.loadFunctions(ctx, q, schemas, version); err != nil {
		return nil, err
	}
	if c.sequences, err = d.loadSequences(ctx, q, schemas); err != nil {
		return nil, err
	}
	if c.views, err = d.loadViews(ctx, q, c.rels); err != nil {
		return nil, err
	}
	c.extensions, err = loadLines(ctx, q, `SELECT 'CREATE EXTENSION IF NOT EXISTS ' || quote_ident(e.extname) || ' WITH SCHEMA ' || quote_ident(n.nspname) || ';'
		FROM pg_extension e JOIN pg_namespace n ON n.oid = e.extnamespace
		WHERE n.nspname = ANY($1) AND e.extname <> 'plpgsql' ORDER BY e.extname`, schemas)
	return c, err
}

func (c *pgCatalog) dataRels() []pgRel {
	var out []pgRel
	for _, r := range c.rels {
		if r.kind == "r" {
			out = append(out, r)
		}
	}
	return out
}

func (c *pgCatalog) alwaysIdentity(oid uint32) bool {
	for _, col := range c.cols[oid] {
		if col.identity == "a" {
			return true
		}
	}
	return false
}

func (c *pgCatalog) identityFor(d Dialect, r pgRel) func(pgCol) string {
	return func(col pgCol) string {
		owner := d.relName(r) + "." + d.QuoteIdent(col.name)
		for _, s := range c.sequences {
			if s.ident && s.owned.Valid && s.owned.String == owner {
				return " (" + s.opts + ")"
			}
		}
		return ""
	}
}

func inlineCheck(c pgConstraint) bool {
	return c.kind == "c" && c.local && !c.parent && !strings.HasSuffix(c.def, " NOT VALID")
}

func (d Dialect) dumpBefore(ctx context.Context, q queryer, c *pgCatalog, schemas []string, opts model.DumpOptions, o *dumpOut) error {
	if opts.Clean {
		d.dumpClean(c, o)
	}
	for _, s := range schemas {
		o.obj("SCHEMA", "", s, "CREATE SCHEMA IF NOT EXISTS "+d.QuoteIdent(s)+";")
	}
	for _, e := range c.extensions {
		o.obj("EXTENSION", "", "", e)
	}
	for _, oid := range c.largeObjs {
		o.obj("BLOB", "", fmt.Sprint(oid), fmt.Sprintf("SELECT pg_catalog.lo_create('%d');", oid))
	}
	late := map[uint32]bool{}
	for _, t := range c.types {
		for _, dep := range t.deps {
			if late[dep] {
				t.late = true
			}
		}
		if t.late {
			late[t.oid] = true
			continue
		}
		if err := d.emitType(ctx, q, t, o); err != nil {
			return err
		}
	}
	for _, f := range c.funcs {
		if f.early {
			if err := d.emitFunc(ctx, q, f, o); err != nil {
				return err
			}
		}
	}
	for _, t := range c.types {
		if late[t.oid] {
			if err := d.emitType(ctx, q, t, o); err != nil {
				return err
			}
		}
	}
	for _, s := range c.sequences {
		if !s.ident {
			o.obj("SEQUENCE", s.schema, s.rel, s.ddl)
		}
	}
	for _, r := range c.rels {
		if r.kind != "r" && r.kind != "p" {
			continue
		}
		parent := ""
		if p, ok := c.byOid[r.parent]; ok {
			parent = d.relName(p)
		} else if r.parent != 0 {
			return fmt.Errorf("%s: its parent table is outside the dumped schemas", d.relName(r))
		}
		var checks []pgConstraint
		for _, x := range c.cons[r.oid] {
			if inlineCheck(x) {
				checks = append(checks, x)
			}
		}
		o.obj("TABLE", r.schema, r.name, d.dumpTableDDL(r, parent, c.cols[r.oid], checks, c.identityFor(d, r)))
	}
	for _, s := range c.sequences {
		if !s.ident && s.owned.Valid {
			o.obj("SEQUENCE OWNED BY", s.schema, s.rel, "ALTER SEQUENCE "+s.name+" OWNED BY "+s.owned.String+";")
		}
	}
	return nil
}

func (d Dialect) emitType(ctx context.Context, q queryer, t pgType, o *dumpOut) error {
	ddl, err := d.typeDDL(ctx, q, t)
	if err != nil {
		return fmt.Errorf("type %s.%s: %w", t.schema, t.name, err)
	}
	desc := "TYPE"
	if t.kind == "d" {
		desc = "DOMAIN"
	}
	o.obj(desc, t.schema, t.name, ddl)
	return nil
}

var funcDesc = map[string]string{"a": "AGGREGATE", "p": "PROCEDURE"}

func (d Dialect) emitFunc(ctx context.Context, q queryer, f pgFunc, o *dumpOut) error {
	ddl, err := d.functionDDL(ctx, q, f)
	if err != nil {
		return fmt.Errorf("function %s: %w", f.name, err)
	}
	desc := funcDesc[f.kind]
	if desc == "" {
		desc = "FUNCTION"
	}
	o.obj(desc, "", f.name, ddl)
	return nil
}

func (d Dialect) dumpClean(c *pgCatalog, o *dumpOut) {
	for i := len(c.views) - 1; i >= 0; i-- {
		kind := "VIEW"
		if c.views[i].rel.kind == "m" {
			kind = "MATERIALIZED VIEW"
		}
		o.line(fmt.Sprintf("DROP %s IF EXISTS %s CASCADE;", kind, d.relName(c.views[i].rel)))
	}
	for i := len(c.rels) - 1; i >= 0; i-- {
		if r := c.rels[i]; r.kind == "r" || r.kind == "p" {
			o.line("DROP TABLE IF EXISTS " + d.relName(r) + " CASCADE;")
		}
	}
	for _, oid := range c.largeObjs {
		o.line(fmt.Sprintf("SELECT pg_catalog.lo_unlink(oid) FROM pg_catalog.pg_largeobject_metadata WHERE oid = '%d';", oid))
	}
	for _, s := range c.sequences {
		if !s.ident {
			o.line("DROP SEQUENCE IF EXISTS " + s.name + " CASCADE;")
		}
	}
	for _, f := range c.funcs {
		kind := map[string]string{"a": "AGGREGATE", "p": "PROCEDURE"}[f.kind]
		if kind == "" {
			kind = "FUNCTION"
		}
		o.line("DROP " + kind + " IF EXISTS " + f.name + " CASCADE;")
	}
	for i := len(c.types) - 1; i >= 0; i-- {
		t := c.types[i]
		kind := "TYPE"
		if t.kind == "d" {
			kind = "DOMAIN"
		}
		o.line("DROP " + kind + " IF EXISTS " + d.QuoteIdent(t.schema) + "." + d.QuoteIdent(t.name) + " CASCADE;")
	}
}
