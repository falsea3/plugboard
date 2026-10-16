package postgres_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var full = model.DumpOptions{Structure: true, Data: true}

func scratchDB(t *testing.T, name string) *sqlcore.Session {
	t.Helper()
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	admin := sqltest.Open(t, postgres.Dialect{}, c)
	ctx := context.Background()
	for _, q := range []string{`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`, `CREATE DATABASE "` + name + `"`} {
		if _, err := admin.DB.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	c.Database = name
	s := sqltest.Open(t, postgres.Dialect{}, c)
	t.Cleanup(func() {
		s.Close()
		admin.DB.ExecContext(context.Background(), `DROP DATABASE IF EXISTS "`+name+`" WITH (FORCE)`)
	})
	return s
}

func dumpText(t *testing.T, s *sqlcore.Session, opts model.DumpOptions) string {
	t.Helper()
	var buf bytes.Buffer
	if err := s.Dump(context.Background(), opts, &buf, nil); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func body(dump string) string {
	_, rest, _ := strings.Cut(dump, "--\n\n")
	return rest
}

func restore(t *testing.T, s *sqlcore.Session, dump string) model.ImportResult {
	t.Helper()
	res, err := s.Import(context.Background(), strings.NewReader(dump), int64(len(dump)), model.ImportOptions{OneTransaction: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("import failed: %+v", res.Errors[0])
	}
	return res
}

func sameDump(t *testing.T, want, got string) {
	t.Helper()
	if body(want) == body(got) {
		return
	}
	w, g := strings.Split(body(want), "\n"), strings.Split(body(got), "\n")
	for i := range min(len(w), len(g)) {
		if w[i] != g[i] {
			t.Fatalf("dumps differ at line %d:\nwant %q\ngot  %q", i+1, w[i], g[i])
		}
	}
	t.Fatalf("dumps differ in length: %d vs %d lines", len(w), len(g))
}

func fixture(t *testing.T) *sqlcore.Session {
	t.Helper()
	src := scratchDB(t, "plugboard_dump_src")
	sql, err := os.ReadFile("testdata/dump.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.DB.ExecContext(context.Background(), string(sql)); err != nil {
		t.Fatal(err)
	}
	return src
}

func TestDumpRoundTrip(t *testing.T) {
	src := fixture(t)
	want := dumpText(t, src, full)
	if os.Getenv("PLUGBOARD_DUMP_OUT") != "" {
		os.WriteFile(os.Getenv("PLUGBOARD_DUMP_OUT"), []byte(want), 0o644)
	}

	dst := scratchDB(t, "plugboard_dump_dst")
	res := restore(t, dst, want)
	if res.Rows == 0 {
		t.Fatalf("no rows copied: %+v", res)
	}
	sameDump(t, want, dumpText(t, dst, full))

	t.Run("inserts", func(t *testing.T) {
		ins := scratchDB(t, "plugboard_dump_ins")
		restore(t, ins, dumpText(t, src, model.DumpOptions{Structure: true, Data: true, Inserts: true}))
		sameDump(t, want, dumpText(t, ins, full))
	})

	for _, format := range []string{"custom", "tar"} {
		t.Run(format+" archive", func(t *testing.T) {
			owned := model.DumpOptions{Structure: true, Data: true, Owners: true}
			archived := model.DumpOptions{Structure: true, Data: true, Owners: true, Format: format}
			arch := scratchDB(t, "plugboard_dump_"+format)
			restore(t, arch, dumpText(t, src, archived))
			sameDump(t, dumpText(t, src, owned), dumpText(t, arch, owned))
		})
	}

	t.Run("owners and privileges", func(t *testing.T) {
		owned := model.DumpOptions{Structure: true, Data: true, Owners: true}
		with := dumpText(t, src, owned)
		for _, s := range []string{"ALTER TABLE app.cache OWNER TO plugboard_owner;", "ALTER TABLE app.events OWNER TO relay;", "GRANT SELECT (name), UPDATE (name) ON TABLE app.customers TO plugboard_reader;", "ALTER DEFAULT PRIVILEGES FOR ROLE relay REVOKE ALL ON FUNCTIONS FROM PUBLIC;"} {
			if !strings.Contains(with, s) {
				t.Errorf("dump has no %q", s)
			}
		}
		own := scratchDB(t, "plugboard_dump_own")
		restore(t, own, with)
		sameDump(t, with, dumpText(t, own, owned))
	})

	t.Run("clean over existing", func(t *testing.T) {
		restore(t, dst, dumpText(t, src, model.DumpOptions{Structure: true, Data: true, Clean: true}))
		sameDump(t, want, dumpText(t, dst, full))
	})

	for _, inserts := range []bool{false, true} {
		t.Run(fmt.Sprintf("structure then data, inserts %v", inserts), func(t *testing.T) {
			split := scratchDB(t, "plugboard_dump_split")
			restore(t, split, dumpText(t, src, model.DumpOptions{Structure: true}))
			if n := countRows(t, split, "app.customers"); n != 0 {
				t.Fatalf("structure-only dump carried %d rows", n)
			}
			restore(t, split, dumpText(t, src, model.DumpOptions{Data: true, Inserts: inserts}))
			sameDump(t, want, dumpText(t, split, full))
		})
	}
}

func TestDumpSchemaSubset(t *testing.T) {
	src := fixture(t)
	got := dumpText(t, src, model.DumpOptions{Structure: true, Data: true, Schemas: []string{"Odd Schema"}})
	if strings.Contains(got, "app.") || !strings.Contains(got, `"Odd Schema"."Mixed Case"`) {
		t.Fatalf("dump of one schema:\n%s", got)
	}
}

func countRows(t *testing.T, s *sqlcore.Session, table string) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRowContext(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestImportPgDump(t *testing.T) {
	src := fixture(t)
	if serverVersion(t, src) < 170000 {
		t.Skip("testdata/pg_dump17.sql needs PostgreSQL 17")
	}
	file, err := os.ReadFile("testdata/pg_dump17.sql")
	if err != nil {
		t.Fatal(err)
	}
	dst := scratchDB(t, "plugboard_dump_pgdump")
	res := restore(t, dst, string(file))
	if len(res.Skipped) != 2 || !strings.HasPrefix(res.Skipped[0], `\restrict`) {
		t.Fatalf("skipped = %q", res.Skipped)
	}
	owned := model.DumpOptions{Structure: true, Data: true, Owners: true}
	sameDump(t, dumpText(t, src, owned), dumpText(t, dst, owned))
}

func TestImportPgArchive(t *testing.T) {
	src := fixture(t)
	if serverVersion(t, src) < 170000 {
		t.Skip("the archives in testdata come from pg_dump 17")
	}
	owned := model.DumpOptions{Structure: true, Data: true, Owners: true}
	want := dumpText(t, src, owned)
	for _, name := range []string{"gzip.dump", "none.dump", "lz4.dump", "zstd.dump", "inserts.dump", "tar"} {
		t.Run(name, func(t *testing.T) {
			file, err := os.ReadFile("testdata/pg_dump17." + name)
			if err != nil {
				t.Fatal(err)
			}
			dst := scratchDB(t, "plugboard_dump_archive")
			restore(t, dst, string(file))
			sameDump(t, want, dumpText(t, dst, owned))
		})
	}
}

func serverVersion(t *testing.T, s *sqlcore.Session) int {
	t.Helper()
	var version int
	if err := s.DB.QueryRowContext(context.Background(), "SELECT current_setting('server_version_num')::int").Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}
