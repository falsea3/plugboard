package db

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	"github.com/relay-client/relay-db/apps/desktop/internal/sshtunnel"
)

// These run against the sample servers from dev/docker-compose.yml:
//
//	make db-up
//	RELAYDB_TEST_PG=127.0.0.1:55432 RELAYDB_TEST_MYSQL=127.0.0.1:53306 RELAYDB_TEST_SSH=127.0.0.1:52222 go test ./internal/db/
func serverConn(t *testing.T, env string, driver model.Driver) model.Connection {
	t.Helper()
	addr := os.Getenv(env)
	if addr == "" {
		t.Skipf("%s not set", env)
	}
	host, portStr, _ := cutLast(addr, ":")
	port, _ := strconv.Atoi(portStr)
	return model.Connection{Driver: driver, Host: host, Port: port, User: "relay", Password: "relay", Database: "shop", SSLMode: "disable"}
}

func cutLast(s, sep string) (string, string, bool) {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}

func TestPostgresServer(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
	ctx := context.Background()
	s, err := Open(ctx, "pg", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.DefaultSchema != "public" || !contains(info.Schemas, "analytics") {
		t.Fatalf("info = %+v", info)
	}
	tables, err := s.Dialect.ListTables(ctx, s.DB, "public")
	if err != nil {
		t.Fatal(err)
	}
	if !hasTable(tables, "customers", "table") || !hasTable(tables, "order_summary", "view") {
		t.Fatalf("tables = %+v", tables)
	}
	cols, err := s.Dialect.ListColumns(ctx, s.DB, "public", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].Name != "id" || !cols[0].PrimaryKey || cols[0].Type != "bigint" || cols[3].Type != "numeric(12,2)" {
		t.Fatalf("cols = %+v", cols)
	}

	res, err := s.Run(ctx, `select 9007199254740993::bigint as big, 12.50::numeric(10,2) as price,
		'{"a":1}'::jsonb as doc, array['x','y'] as tags, true as flag, null::text as nothing,
		'2026-01-02 03:04:05+00'::timestamptz as at, 'NaN'::float8 as nan, '\xdead'::bytea as bin`)
	if err != nil {
		t.Fatal(err)
	}
	row := res[0].Rows[0]
	want := []any{"9007199254740993", "12.50", `{"a": 1}`, "{x,y}", true, nil, nil, "NaN", "0xdead"}
	for i, w := range want {
		if i == 6 {
			continue // timestamptz is rendered in the session time zone
		}
		if row[i] != w {
			t.Errorf("column %s = %#v, want %#v", res[0].Columns[i].Name, row[i], w)
		}
	}

	page, err := s.TablePage(ctx, model.TableQuery{Schema: "analytics", Table: "daily_revenue", Limit: 5})
	if err != nil || len(page.Result.Rows) == 0 {
		t.Fatalf("page = %+v, err = %v", page, err)
	}
}

func TestPostgresCancel(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
	s, err := Open(context.Background(), "pg", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = s.Run(ctx, "select pg_sleep(10)")
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("cancel took %v, err = %v", time.Since(start), err)
	}
	// The editor connection must be usable again afterwards.
	res, err := s.Run(context.Background(), "select 1")
	if err != nil || res[0].Rows[0][0] != int64(1) {
		t.Fatalf("after cancel: %v %v", res, err)
	}

	// Cancelling inside a transaction says the transaction went too.
	if _, err := s.Run(context.Background(), "begin"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = s.Run(ctx, "select pg_sleep(10)")
	var se *StatementError
	if !errors.As(err, &se) || !se.RolledBack {
		t.Fatalf("cancel in a transaction: err = %v", err)
	}
}

// A dropped editor connection: a read outside a transaction quietly runs
// again, but inside one the user is told, since the transaction is gone.
func TestPostgresEditorConnectionLost(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
	ctx := context.Background()
	s, err := Open(ctx, "pg", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	kill := func() {
		t.Helper()
		res, err := s.Run(ctx, "select pg_backend_pid()")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.ExecContext(ctx, "select pg_terminate_backend($1)", res[0].Rows[0][0]); err != nil {
			t.Fatal(err)
		}
	}

	kill()
	if res, err := s.Run(ctx, "select 1"); err != nil || res[0].Rows[0][0] != int64(1) {
		t.Fatalf("read after a drop: %v %v", res, err)
	}

	if _, err := s.Run(ctx, "begin"); err != nil {
		t.Fatal(err)
	}
	kill()
	_, err = s.Run(ctx, "select 1")
	var se *StatementError
	if !errors.Is(err, ErrConnLost) || !errors.As(err, &se) || !se.RolledBack {
		t.Fatalf("read in a lost transaction: err = %v", err)
	}
}

func TestMySQLServer(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	ctx := context.Background()
	s, err := Open(ctx, "my", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.DefaultSchema != "shop" {
		t.Fatalf("info = %+v", info)
	}
	tables, err := s.Dialect.ListTables(ctx, s.DB, "shop")
	if err != nil {
		t.Fatal(err)
	}
	if !hasTable(tables, "orders", "table") || !hasTable(tables, "order_summary", "view") {
		t.Fatalf("tables = %+v", tables)
	}
	cols, err := s.Dialect.ListColumns(ctx, s.DB, "shop", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if !cols[0].PrimaryKey || cols[2].Type != "enum('new','paid','shipped','cancelled')" {
		t.Fatalf("cols = %+v", cols)
	}

	res, err := s.Run(ctx, `select 'a\';b' as s; select cast(12.5 as decimal(10,2)) as d, null as n, cast('2026-01-02 03:04:05' as datetime) as dt, x'dead' as bin # trailing ; comment
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0].Rows[0][0] != "a';b" {
		t.Fatalf("res = %+v", res)
	}
	row := res[1].Rows[0]
	if row[0] != "12.50" || row[1] != nil || row[2] != "2026-01-02 03:04:05" {
		t.Fatalf("row = %#v", row)
	}

	page, err := s.TablePage(ctx, model.TableQuery{Schema: "shop", Table: "customers", Limit: 10, OrderBy: "id", OrderDesc: true})
	if err != nil || len(page.Result.Rows) != 10 || !page.HasMore {
		t.Fatalf("page = %+v, err = %v", page, err)
	}

	_, err = s.Run(ctx, "select * from nope")
	var se *StatementError
	if !errors.As(err, &se) {
		t.Fatalf("err = %v", err)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func hasTable(tables []model.TableInfo, name, kind string) bool {
	for _, t := range tables {
		if t.Name == name && t.Kind == kind {
			return true
		}
	}
	return false
}

// Server-side read-only: the statement goes straight to the pool, past the screen.
func TestServerReadOnly(t *testing.T) {
	for _, tc := range []struct {
		env    string
		driver model.Driver
	}{{"RELAYDB_TEST_PG", model.Postgres}, {"RELAYDB_TEST_MYSQL", model.MySQL}} {
		t.Run(string(tc.driver), func(t *testing.T) {
			c := serverConn(t, tc.env, tc.driver)
			c.ReadOnly = true
			ctx := context.Background()
			s, err := Open(ctx, "ro", c, OpenOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			// Several pool connections, so every one of them must be read-only.
			for i := 0; i < 6; i++ {
				_, err := s.DB.ExecContext(ctx, "update customers set name = name where id = 1")
				if err == nil {
					t.Fatalf("attempt %d: write succeeded on a read-only connection", i)
				}
			}
			if _, err := s.Run(ctx, "set session characteristics as transaction read write"); err == nil {
				t.Fatal("editor let the session switch back to read write")
			}
			if tc.driver == model.Postgres {
				// Turning the server side off, then hiding a DELETE behind a
				// backslash that only MySQL would read as an escape.
				_, err := s.Run(ctx, "select set_config('default_transaction_read_only', 'off', false);\n"+
					`with p as (select 'C:\' as x), d as (delete from orders where id < 0 returning 1) select '' from d`)
				var ro *ReadOnlyError
				if !errors.As(err, &ro) {
					t.Fatalf("read-only bypass: err = %v", err)
				}
			}
			if _, err := s.Run(ctx, "select count(*) from customers"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Postgres behind the bastion from dev/docker-compose.yml: the database host
// is the compose service name, which only resolves from the SSH server.
func TestPostgresThroughSSH(t *testing.T) {
	addr := os.Getenv("RELAYDB_TEST_SSH")
	if addr == "" {
		t.Skip("RELAYDB_TEST_SSH not set")
	}
	host, portStr, _ := cutLast(addr, ":")
	port, _ := strconv.Atoi(portStr)
	c := model.Connection{
		Driver: model.Postgres, Host: "postgres", Port: 5432, User: "relay", Password: "relay", Database: "shop", SSLMode: "disable",
		SSH: model.SSHTunnel{Enabled: true, Host: host, Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	ctx := context.Background()
	s, err := Open(ctx, "ssh", c, OpenOptions{KnownHostsFile: t.TempDir() + "/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := s.Run(ctx, "select count(*) from customers")
	if err != nil || res[0].Rows[0][0] != int64(2500) {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}

// Edits against real servers: every value travels as text and is cast by the
// server to the column type, so numerics, booleans, json, arrays, enums and
// timestamps all have to survive the round trip.
func TestApplyChangesServers(t *testing.T) {
	for _, tc := range []struct {
		env    string
		driver model.Driver
		schema string
	}{{"RELAYDB_TEST_PG", model.Postgres, "public"}, {"RELAYDB_TEST_MYSQL", model.MySQL, "shop"}} {
		t.Run(string(tc.driver), func(t *testing.T) {
			c := serverConn(t, tc.env, tc.driver)
			ctx := context.Background()
			s, err := Open(ctx, "edit", c, OpenOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			s.Run(ctx, "delete from customers where email like 'edit-test-%'")

			meta := `{"plan":"team"}`
			insert := map[string]any{"email": "edit-test-1@example.com", "name": "Edit Test", "country": "NL", "is_active": "1", "meta": meta}
			if tc.driver == model.Postgres {
				insert["is_active"] = true
			}
			if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: tc.schema, Table: "customers", Changes: []model.RowChange{
				{Kind: model.ChangeInsert, Values: insert},
			}}); err != nil {
				t.Fatalf("insert: %v", err)
			}
			res, err := s.Run(ctx, "select id from customers where email = 'edit-test-1@example.com'")
			if err != nil || len(res[0].Rows) != 1 {
				t.Fatalf("inserted row: %v %v", res, err)
			}
			id := res[0].Rows[0][0]

			// Same value again must still count as hitting the row (found rows, not changed rows).
			changes := []model.RowChange{
				{Kind: model.ChangeUpdate, Key: map[string]any{"id": id}, Values: map[string]any{"name": "Edit Test", "country": nil, "created_at": "2026-01-02 03:04:05"}},
			}
			if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: tc.schema, Table: "customers", Changes: changes}); err != nil {
				t.Fatalf("update: %v", err)
			}
			res, _ = s.Run(ctx, "select country is null from customers where email = 'edit-test-1@example.com'")
			if v := res[0].Rows[0][0]; v != true && v != int64(1) {
				t.Fatalf("country not nulled: %v", v)
			}

			// A failing second change rolls back the first.
			_, err = s.ApplyChanges(ctx, model.ChangeSet{Schema: tc.schema, Table: "customers", Changes: []model.RowChange{
				{Kind: model.ChangeUpdate, Key: map[string]any{"id": id}, Values: map[string]any{"name": "should roll back"}},
				{Kind: model.ChangeUpdate, Key: map[string]any{"id": id}, Values: map[string]any{"email": "user2@example.com"}}, // unique violation
			}})
			if err == nil {
				t.Fatal("unique violation was applied")
			}
			res, _ = s.Run(ctx, "select name from customers where email = 'edit-test-1@example.com'")
			if res[0].Rows[0][0] != "Edit Test" {
				t.Fatalf("not rolled back: %v", res[0].Rows[0][0])
			}

			// country is char(2): a longer value is an error, never cut to fit.
			if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: tc.schema, Table: "customers", Changes: []model.RowChange{
				{Kind: model.ChangeUpdate, Key: map[string]any{"id": id}, Values: map[string]any{"country": "NLX"}},
			}}); err == nil {
				t.Fatal("an over-long value was written")
			}
			res, _ = s.Run(ctx, "select country is null from customers where email = 'edit-test-1@example.com'")
			if v := res[0].Rows[0][0]; v != true && v != int64(1) {
				t.Fatalf("country changed to %v", v)
			}

			if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: tc.schema, Table: "customers", Changes: []model.RowChange{
				{Kind: model.ChangeDelete, Key: map[string]any{"id": id}},
			}}); err != nil {
				t.Fatalf("delete: %v", err)
			}
		})
	}
}

func TestTableFiltersServers(t *testing.T) {
	for _, tc := range []struct {
		env    string
		driver model.Driver
		schema string
	}{{"RELAYDB_TEST_PG", model.Postgres, "public"}, {"RELAYDB_TEST_MYSQL", model.MySQL, "shop"}} {
		t.Run(string(tc.driver), func(t *testing.T) {
			c := serverConn(t, tc.env, tc.driver)
			ctx := context.Background()
			s, err := Open(ctx, "f", c, OpenOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			page := func(f ...model.Filter) model.TablePage {
				t.Helper()
				p, err := s.TablePage(ctx, model.TableQuery{Schema: tc.schema, Table: "customers", Limit: 5000, Filters: f})
				if err != nil {
					t.Fatal(err)
				}
				return p
			}
			if p := page(model.Filter{Column: "id", Op: "=", Value: "7"}); len(p.Result.Rows) != 1 {
				t.Fatalf("id = 7: %d rows", len(p.Result.Rows))
			}
			if p := page(model.Filter{Column: "id", Op: "in", Value: "1,2,3"}, model.Filter{Column: "country", Op: "not_null"}); len(p.Result.Rows) == 0 {
				t.Fatal("in + not null returned nothing")
			}
			// Case-insensitive contains on a non-text column type works too.
			if p := page(model.Filter{Column: "email", Op: "contains", Value: "USER12@"}); len(p.Result.Rows) != 1 {
				t.Fatalf("contains: %d rows", len(p.Result.Rows))
			}
			if p := page(model.Filter{Column: "created_at", Op: "<", Value: "2000-01-01"}); len(p.Result.Rows) != 0 {
				t.Fatalf("date compare: %d rows", len(p.Result.Rows))
			}
			// The value is compared as typed, not first cut to char(2).
			if p := page(model.Filter{Column: "country", Op: "=", Value: "USA"}); len(p.Result.Rows) != 0 {
				t.Fatalf("country = USA matched %d rows", len(p.Result.Rows))
			}
			if p := page(); p.DefaultOrder[0] != "id" {
				t.Fatalf("default order = %v", p.DefaultOrder)
			}
		})
	}
}

// Restarts the bastion container under an open session. Opt in with
// RELAYDB_TEST_SSH_RESTART=1 — it touches Docker.
func TestSSHTunnelSurvivesServerRestart(t *testing.T) {
	addr := os.Getenv("RELAYDB_TEST_SSH")
	if addr == "" || os.Getenv("RELAYDB_TEST_SSH_RESTART") == "" {
		t.Skip("RELAYDB_TEST_SSH and RELAYDB_TEST_SSH_RESTART not set")
	}
	host, portStr, _ := cutLast(addr, ":")
	port, _ := strconv.Atoi(portStr)
	c := model.Connection{
		Driver: model.Postgres, Host: "postgres", Port: 5432, User: "relay", Password: "relay", Database: "shop", SSLMode: "disable",
		SSH: model.SSHTunnel{Enabled: true, Host: host, Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	var mu sync.Mutex
	var states []sshtunnel.State
	ctx := context.Background()
	s, err := Open(ctx, "ssh", c, OpenOptions{
		KnownHostsFile: t.TempDir() + "/known_hosts",
		OnTunnelState: func(st sshtunnel.State, _ error) {
			mu.Lock()
			states = append(states, st)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Run(ctx, "select 1"); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("docker", "restart", "dev-bastion-1").CombinedOutput(); err != nil {
		t.Fatalf("restart bastion: %v %s", err, out)
	}
	waitForSSH(t, addr)

	// A table page and a read in the editor just work: the dead pooled
	// connection is retried once over a fresh tunnel.
	if _, err := s.TablePage(ctx, model.TableQuery{Schema: "public", Table: "customers", Limit: 5}); err != nil {
		t.Fatalf("table page after restart: %v", err)
	}
	if _, err := s.Run(ctx, "select count(*) from customers"); err != nil {
		t.Fatalf("editor read after restart: %v", err)
	}
	mu.Lock()
	got := append([]sshtunnel.State(nil), states...)
	mu.Unlock()
	if len(got) == 0 || got[len(got)-1] != sshtunnel.StateReconnected {
		t.Fatalf("tunnel states = %v, want … reconnected", got)
	}

	// A write whose connection died is not re-run behind the user's back.
	if out, err := exec.Command("docker", "restart", "dev-bastion-1").CombinedOutput(); err != nil {
		t.Fatalf("restart bastion: %v %s", err, out)
	}
	waitForSSH(t, addr)
	_, err = s.Run(ctx, "update customers set name = name where id = 1")
	if !errors.Is(err, ErrConnLost) {
		t.Fatalf("write after drop: err = %v, want ErrConnLost", err)
	}
	if _, err := s.Run(ctx, "update customers set name = name where id = 1"); err != nil {
		t.Fatalf("write on the fresh connection: %v", err)
	}
}

func waitForSSH(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			buf := make([]byte, 4)
			c.SetReadDeadline(time.Now().Add(time.Second))
			n, _ := c.Read(buf)
			c.Close()
			if n == 4 && string(buf) == "SSH-" {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("SSH server did not come back")
}

func TestEditorPagingServers(t *testing.T) {
	t.Run("postgres", func(t *testing.T) {
		c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
		ctx := context.Background()
		s, err := Open(ctx, "pg", c, OpenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		// Inside a transaction the wrap runs under a savepoint and the transaction survives.
		res, err := s.Run(ctx, "begin; select * from orders; select 1 as a, 2 as a; commit")
		if err != nil {
			t.Fatal(err)
		}
		if !res[1].Pageable || !res[1].HasMore || len(res[1].Rows) != EditorChunk {
			t.Fatalf("orders: pageable=%v more=%v rows=%d", res[1].Pageable, res[1].HasMore, len(res[1].Rows))
		}
		if len(res[2].Columns) != 2 {
			t.Fatalf("duplicate names: %v", res[2].Columns)
		}
		if st := func() byte { s.editorMu.Lock(); defer s.editorMu.Unlock(); return pgTxStatus(s.editor) }(); st != 'I' {
			t.Fatalf("tx status after commit = %q, want idle", st)
		}
		// The 3M-row table from the review, if it's there: one chunk, quickly.
		if r, err := s.Run(ctx, "select to_regclass('public.big_events') is not null"); err == nil && r[0].Rows[0][0] == true {
			start := time.Now()
			big, err := s.Run(ctx, "select * from big_events")
			if err != nil || len(big[0].Rows) != EditorChunk || !big[0].HasMore {
				t.Fatalf("big_events: %v", err)
			}
			if d := time.Since(start); d > time.Second {
				t.Fatalf("big_events first chunk took %v", d)
			}
		}
	})
	t.Run("mysql", func(t *testing.T) {
		c := serverConn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
		ctx := context.Background()
		s, err := Open(ctx, "my", c, OpenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		// MySQL rejects duplicate column names in a derived table: falls back to running as typed.
		res, err := s.Run(ctx, "select 1 as a, 2 as a; select * from orders order by id desc")
		if err != nil {
			t.Fatal(err)
		}
		if res[0].Pageable || len(res[0].Columns) != 2 {
			t.Fatalf("duplicate names: pageable=%v cols=%v", res[0].Pageable, res[0].Columns)
		}
		if !res[1].HasMore || res[1].Rows[0][0] != int64(5000) {
			t.Fatalf("order by survived the wrap? first id = %v", res[1].Rows[0][0])
		}
	})
}

func TestEnumsAndExpressionsServers(t *testing.T) {
	t.Run("postgres", func(t *testing.T) {
		c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
		ctx := context.Background()
		s, err := Open(ctx, "pg", c, OpenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		if _, err := s.Run(ctx, `drop table if exists probe_mood; drop type if exists mood;
			create type mood as enum ('sad', 'ok', 'happy');
			create table probe_mood (id int primary key, m mood not null default 'ok', at timestamptz, d date, flag boolean default true);
			insert into probe_mood (id, m) values (1, 'sad')`); err != nil {
			t.Fatal(err)
		}
		cols, err := s.Columns(ctx, "public", "probe_mood")
		if err != nil {
			t.Fatal(err)
		}
		if got := cols[1].Enum; len(got) != 3 || got[0] != "sad" || got[2] != "happy" {
			t.Fatalf("enum = %v", got)
		}
		if cols[0].Enum != nil {
			t.Fatalf("int column has enum values: %v", cols[0].Enum)
		}
		now := map[string]any{"$expr": "now"}
		_, err = s.ApplyChanges(ctx, model.ChangeSet{Schema: "public", Table: "probe_mood", Changes: []model.RowChange{
			{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{
				"m": "happy", "at": now, "d": now, "flag": map[string]any{"$expr": "default"},
			}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		res, _ := s.Run(ctx, "select m::text, at > now() - interval '1 minute', d = current_date, flag from probe_mood")
		if row := res[0].Rows[0]; row[0] != "happy" || row[1] != true || row[2] != true || row[3] != true {
			t.Fatalf("row = %v", row)
		}
	})
	t.Run("mysql", func(t *testing.T) {
		c := serverConn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
		ctx := context.Background()
		s, err := Open(ctx, "my", c, OpenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		cols, err := s.Columns(ctx, "shop", "orders")
		if err != nil {
			t.Fatal(err)
		}
		if got := cols[2].Enum; len(got) != 4 || got[0] != "new" {
			t.Fatalf("enum = %v", got)
		}
		_, err = s.ApplyChanges(ctx, model.ChangeSet{Schema: "shop", Table: "orders", Changes: []model.RowChange{
			{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{
				"placed_at": map[string]any{"$expr": "now"}, "status": map[string]any{"$expr": "default"},
			}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		res, _ := s.Run(ctx, "select status, placed_at > now() - interval 1 minute from orders where id = 1")
		if row := res[0].Rows[0]; row[0] != "new" || row[1] != int64(1) {
			t.Fatalf("row = %v", row)
		}
	})
}

func TestBigTablePagingAndCount(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
	ctx := context.Background()
	s, err := Open(ctx, "pg", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if r, err := s.Run(ctx, "select to_regclass('public.big_events') is not null"); err != nil || r[0].Rows[0][0] != true {
		t.Skip("big_events not loaded")
	}
	q := model.TableQuery{Schema: "public", Table: "big_events", Limit: 300, Last: true}
	start := time.Now()
	p, err := s.TablePage(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 300*time.Millisecond {
		t.Fatalf("last page of 3M rows took %v", d)
	}
	if !p.Keyset || p.HasMore || !p.HasPrev {
		t.Fatalf("page flags: %+v", p)
	}
	est, err := s.CountRows(ctx, model.TableQuery{Schema: "public", Table: "big_events"}, false)
	if err != nil || !est.Known || est.Exact || est.Count < 2_500_000 {
		t.Fatalf("estimate = %+v %v", est, err)
	}
}
