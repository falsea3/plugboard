package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

type dumpOut struct {
	w        io.Writer
	bytes    int64
	err      error
	progress func(model.DumpProgress)
	state    model.DumpProgress
	last     time.Time
}

func (o *dumpOut) Write(p []byte) (int, error) {
	if o.err != nil {
		return 0, o.err
	}
	n, err := o.w.Write(p)
	o.bytes += int64(n)
	o.err = err
	if o.progress != nil && time.Since(o.last) > 200*time.Millisecond {
		o.report()
	}
	return n, err
}

func (o *dumpOut) report() {
	o.last = time.Now()
	o.state.Bytes = o.bytes
	if o.progress != nil {
		o.progress(o.state)
	}
}

func (o *dumpOut) line(s string) {
	if s != "" {
		fmt.Fprint(o, s, "\n\n")
	}
}

func (o *dumpOut) section(title string) {
	fmt.Fprintf(o, "--\n-- %s\n--\n\n", title)
}

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
	var version int
	var dbName, server string
	if err := conn.QueryRowContext(ctx, `SELECT current_setting('server_version_num')::int, current_database(), version()`).Scan(&version, &dbName, &server); err != nil {
		return err
	}
	schemas := opts.Schemas
	if len(schemas) == 0 {
		var err error
		if schemas, err = loadLines(ctx, conn, `SELECT nspname FROM pg_namespace WHERE nspname NOT IN ('pg_catalog', 'information_schema') AND nspname NOT LIKE 'pg\_%' ORDER BY nspname`); err != nil {
			return err
		}
	}
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
	fmt.Fprintf(o, "--\n-- Plugboard dump of database %s\n-- %s\n-- Schemas: %s\n-- %s\n--\n\n", dbName, firstLine(server), strings.Join(schemas, ", "), time.Now().UTC().Format(time.RFC3339))
	o.line("SET statement_timeout = 0;\nSET lock_timeout = 0;\nSET idle_in_transaction_session_timeout = 0;\nSET client_encoding = 'UTF8';\nSET standard_conforming_strings = on;\nSELECT pg_catalog.set_config('search_path', '', false);\nSET check_function_bodies = false;\nSET xmloption = content;\nSET client_min_messages = warning;\nSET row_security = off;")
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
		o.line(v)
	}
	return nil
}
