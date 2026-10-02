package postgres_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestServer(t *testing.T) {
	sqltest.Run(t, sqltest.Server{
		Env:       "PLUGBOARD_TEST_PG",
		Dialect:   postgres.Dialect{},
		Schema:    "public",
		True:      true,
		KeyColumn: "id serial primary key",
	})
}

func TestCatalogAndValues(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)

	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.DefaultSchema != "public" || !slices.Contains(info.Schemas, "analytics") {
		t.Fatalf("info = %+v", info)
	}
	tables, err := s.Dialect.ListTables(ctx, s.DB, "public")
	if err != nil {
		t.Fatal(err)
	}
	if !sqltest.HasTable(tables, "customers", "table") || !sqltest.HasTable(tables, "order_summary", "view") {
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
			continue
		}
		if row[i] != w {
			t.Errorf("column %s = %#v, want %#v", res[0].Columns[i].Name, row[i], w)
		}
	}

	page, err := s.TablePage(ctx, model.TableQuery{Schema: "analytics", Table: "daily_revenue", Limit: 5})
	if err != nil || len(page.Result.Rows) == 0 {
		t.Fatalf("page = %+v, err = %v", page, err)
	}

	res, err = s.Run(ctx, "select 5.2::real, 5.2::double precision")
	if err != nil || res[0].Rows[0][0] != 5.2 || res[0].Rows[0][1] != 5.2 {
		t.Fatalf("real/double = %v, err = %v", res, err)
	}
}

func TestCancel(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	s := sqltest.Open(t, postgres.Dialect{}, c)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := s.Run(ctx, "select pg_sleep(10)")
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("cancel took %v, err = %v", time.Since(start), err)
	}
	res, err := s.Run(context.Background(), "select 1")
	if err != nil || res[0].Rows[0][0] != int64(1) {
		t.Fatalf("after cancel: %v %v", res, err)
	}

	if _, err := s.Run(context.Background(), "begin"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = s.Run(ctx, "select pg_sleep(10)")
	var se *db.StatementError
	if !errors.As(err, &se) || !se.RolledBack {
		t.Fatalf("cancel in a transaction: err = %v", err)
	}
}

func TestEditorConnectionLost(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
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
	_, err := s.Run(ctx, "select 1")
	var se *db.StatementError
	if !errors.Is(err, db.ErrConnLost) || !errors.As(err, &se) || !se.RolledBack {
		t.Fatalf("read in a lost transaction: err = %v", err)
	}
}

func TestReadOnlyBypass(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	c.ReadOnly = true
	s := sqltest.Open(t, postgres.Dialect{}, c)
	_, err := s.Run(context.Background(), "select set_config('default_transaction_read_only', 'off', false);\n"+
		`with p as (select 'C:\' as x), d as (delete from orders where id < 0 returning 1) select '' from d`)
	var ro *db.ReadOnlyError
	if !errors.As(err, &ro) {
		t.Fatalf("read-only bypass: err = %v", err)
	}
}

func TestPagingInTransaction(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	s := sqltest.Open(t, postgres.Dialect{}, c)
	res, err := s.Run(context.Background(), "begin; select * from orders; select 1 as a, 2 as a; select 3; commit")
	if err != nil {
		t.Fatal(err)
	}
	if !res[1].Pageable || !res[1].HasMore || len(res[1].Rows) != sqlcore.EditorChunk {
		t.Fatalf("orders: pageable=%v more=%v rows=%d", res[1].Pageable, res[1].HasMore, len(res[1].Rows))
	}
	if len(res[2].Columns) != 2 || res[3].Rows[0][0] != int64(3) {
		t.Fatalf("after the rejected wrap: %v %v", res[2].Columns, res[3].Rows)
	}
}

func TestEnumsAndExpressions(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
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
}

func TestBigTable(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
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

func TestStructureTypeAndDefault(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
	s.Run(ctx, "drop table if exists zz_structure")
	defer s.Run(context.Background(), "drop table if exists zz_structure")
	if _, err := s.Run(ctx, "create table zz_structure (id int primary key, flag text default 'no', kind text default 'a')"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "public", Table: "zz_structure", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "flag", Type: sqltest.Ptr("boolean"), DefaultSet: true, Default: sqltest.Ptr("false")},
		{Kind: model.ChangeUpdate, Column: "kind", Type: sqltest.Ptr("varchar(5)"), DefaultSet: true, Default: nil},
	}}); err != nil {
		t.Fatal(err)
	}
	if c := sqltest.Column(t, s, "public", "zz_structure", "flag"); c.Type != "boolean" || c.Default == nil || *c.Default != "false" {
		t.Errorf("flag = %+v", c)
	}
	if c := sqltest.Column(t, s, "public", "zz_structure", "kind"); c.Default != nil {
		t.Errorf("kind kept its default: %+v", c)
	}
}
