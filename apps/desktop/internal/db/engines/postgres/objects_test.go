package postgres_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestObjects(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
	drop := func() {
		s.Run(ctx, `drop table if exists zz_obj cascade; drop function if exists zz_touch() cascade; drop function if exists zz_add(int, int);
			drop sequence if exists zz_seq; drop type if exists zz_mood; drop domain if exists zz_email; drop materialized view if exists zz_mv`)
	}
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, `
		create type zz_mood as enum ('sad', 'ok', 'it''s great');
		create domain zz_email as text check (value like '%@%');
		create sequence zz_seq start with 5 increment by 2;
		create table zz_obj (id int primary key, mood zz_mood, mail zz_email);
		create materialized view zz_mv as select id from zz_obj;
		create function zz_add(a int, b int) returns int language sql as $$ select a + b $$;
		create function zz_touch() returns trigger language plpgsql as $$ begin return new; end $$;
		create trigger zz_obj_touch before update on zz_obj for each row execute function zz_touch()`); err != nil {
		t.Fatal(err)
	}
	objs, err := s.Objects(ctx, "public")
	if err != nil {
		t.Fatal(err)
	}
	find := func(kind, name string) model.DBObject {
		t.Helper()
		i := slices.IndexFunc(objs, func(o model.DBObject) bool { return o.Kind == kind && o.Name == name })
		if i < 0 {
			t.Fatalf("no %s %s in %+v", kind, name, objs)
		}
		return objs[i]
	}
	if e := find("enum", "zz_mood"); !slices.Equal(e.Values, []string{"sad", "ok", "it's great"}) {
		t.Errorf("enum = %+v", e)
	}
	if f := find("function", "zz_add"); f.Detail != "a integer, b integer" {
		t.Errorf("function = %+v", f)
	}
	if tr := find("trigger", "zz_obj_touch"); tr.Detail != "zz_obj" {
		t.Errorf("trigger = %+v", tr)
	}
	for kind, want := range map[[2]string]string{
		{"enum", "zz_mood"}:         `CREATE TYPE "public"."zz_mood" AS ENUM (`,
		{"domain", "zz_email"}:      `CREATE DOMAIN "public"."zz_email" AS text`,
		{"sequence", "zz_seq"}:      "START WITH 5",
		{"function", "zz_add"}:      "CREATE OR REPLACE FUNCTION public.zz_add(a integer, b integer)",
		{"trigger", "zz_obj_touch"}: "CREATE TRIGGER zz_obj_touch BEFORE UPDATE ON zz_obj",
	} {
		ddl, err := s.DDL(ctx, find(kind[0], kind[1]))
		if err != nil || !strings.Contains(ddl, want) || !strings.HasSuffix(ddl, ";") {
			t.Errorf("%s DDL = %q, %v; want it to contain %q", kind[1], ddl, err, want)
		}
	}
	if mv, err := s.DDL(ctx, model.DBObject{Schema: "public", Name: "zz_mv", Kind: "view"}); err != nil || !strings.HasPrefix(mv, `CREATE MATERIALIZED VIEW "public"."zz_mv" AS`) {
		t.Errorf("matview DDL = %q, %v", mv, err)
	}
	ddl, err := s.DDL(ctx, model.DBObject{Schema: "public", Name: "zz_obj", Kind: "table"})
	if err != nil || !strings.Contains(ddl, `CONSTRAINT "zz_obj_pkey" PRIMARY KEY (id)`) && !strings.Contains(ddl, `CONSTRAINT zz_obj_pkey PRIMARY KEY (id)`) || !strings.Contains(ddl, "CREATE TRIGGER zz_obj_touch") {
		t.Errorf("table DDL = %s, %v", ddl, err)
	}
}
