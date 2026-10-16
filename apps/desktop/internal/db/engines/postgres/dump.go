package postgres

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"time"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (d Dialect) Dump(ctx context.Context, conn *sql.Conn, opts model.DumpOptions, w io.Writer, progress func(model.DumpProgress)) error {
	if _, err := conn.ExecContext(ctx, "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	for _, set := range []string{"SET LOCAL search_path = pg_catalog", "SET LOCAL synchronize_seqscans = off", "SET LOCAL statement_timeout = 0"} {
		if _, err := conn.ExecContext(ctx, set); err != nil {
			return err
		}
	}
	out := &dumpOut{w: w, progress: progress}
	err := d.dump(ctx, conn, opts, out)
	if err == nil {
		err = out.err
	}
	out.state.Phase = "done"
	out.report()
	return err
}

func (d Dialect) dump(ctx context.Context, conn *sql.Conn, opts model.DumpOptions, o *dumpOut) error {
	h := dumpHead{at: time.Now().UTC()}
	var version int
	if err := conn.QueryRowContext(ctx, `SELECT current_setting('server_version_num')::int, current_setting('server_version'), current_database(), version()`).Scan(&version, &h.version, &h.db, &h.server); err != nil {
		return err
	}
	h.server = firstLine(h.server)
	schemas := opts.Schemas
	if len(schemas) == 0 {
		var err error
		if schemas, err = loadLines(ctx, conn, `SELECT nspname FROM pg_namespace WHERE nspname NOT IN ('pg_catalog', 'information_schema') AND nspname NOT LIKE 'pg\_%' ORDER BY nspname`); err != nil {
			return err
		}
	}
	h.schemas = schemas
	c, err := d.loadCatalog(ctx, conn, schemas, version)
	if err != nil {
		return err
	}
	if len(opts.Schemas) == 0 {
		if c.largeObjs, err = loadLargeObjects(ctx, conn); err != nil {
			return err
		}
	}
	o.state = model.DumpProgress{Phase: "structure", Tables: len(c.dataRels())}
	o.report()
	if err := d.plan(ctx, conn, c, schemas, version, opts, o); err != nil {
		return err
	}
	o.state.Phase = "data"
	switch opts.Format {
	case "custom":
		return writeCustom(o, h)
	case "tar":
		return writeTar(o, h)
	}
	return writePlain(o, h)
}

func (d Dialect) plan(ctx context.Context, conn *sql.Conn, c *pgCatalog, schemas []string, version int, opts model.DumpOptions, o *dumpOut) error {
	o.sec = sectionPreData
	if opts.Structure {
		if err := d.dumpBefore(ctx, conn, c, schemas, opts, o); err != nil {
			return err
		}
	}
	if opts.Data {
		if err := d.dumpData(ctx, conn, c, schemas, opts, o); err != nil {
			return err
		}
	}
	if opts.Structure {
		return d.dumpAfter(ctx, conn, c, schemas, version, opts, o)
	}
	return d.dumpSequenceValues(ctx, conn, c, o)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, ','); i > 0 {
		return s[:i]
	}
	return s
}

func (d Dialect) dumpSequenceValues(ctx context.Context, q queryer, c *pgCatalog, o *dumpOut) error {
	for _, s := range c.sequences {
		v, err := d.sequenceValue(ctx, q, s)
		if err != nil {
			return err
		}
		o.data(dumpEntry{desc: "SEQUENCE SET", ns: s.schema, tag: s.rel, defn: v + "\n"})
	}
	return nil
}

func (Dialect) DumpFormats() []model.DumpFormat {
	return []model.DumpFormat{
		{ID: "sql", Label: "SQL", Ext: ".sql", Hint: "Plain SQL that psql or any SQL tool can run. Name the file .sql.gz to compress it."},
		{ID: "custom", Label: "Custom", Ext: ".dump", Hint: "pg_dump's custom format, compressed: pg_restore can list it, restore single tables and run in parallel."},
		{ID: "tar", Label: "Tar", Ext: ".tar", Hint: "pg_dump's tar format: pg_restore reads it, and tar unpacks it into a file per table."},
	}
}
