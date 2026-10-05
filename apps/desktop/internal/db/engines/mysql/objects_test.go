package mysql_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestObjects(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_MYSQL", model.MySQL)
	ctx := context.Background()
	s := sqltest.Open(t, mysql.Dialect{}, c)
	drop := func() {
		s.Run(ctx, "drop table if exists zz_obj; drop function if exists zz_add; drop procedure if exists zz_hello")
	}
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, "DELIMITER //\ncreate procedure zz_dump() begin select 1; select 2; end//\nDELIMITER ;\ndrop procedure zz_dump;"); err != nil {
		t.Fatalf("a dump with DELIMITER: %v", err)
	}
	for _, stmt := range []string{
		"create table zz_obj (id int primary key, n int)",
		"create function zz_add(a int, b int) returns int deterministic return a + b",
		"create procedure zz_hello(in x int) begin if x > 0 then select 1; else select 2; end if; end",
		"create trigger zz_obj_touch before update on zz_obj for each row set new.n = new.n + 1",
	} {
		if _, err := s.Run(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	objs, err := s.Objects(ctx, "shop")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []model.DBObject{
		{Schema: "shop", Name: "zz_add", Kind: "function"},
		{Schema: "shop", Name: "zz_hello", Kind: "procedure"},
		{Schema: "shop", Name: "zz_obj_touch", Kind: "trigger", Detail: "zz_obj"},
	} {
		i := slices.IndexFunc(objs, func(o model.DBObject) bool { return o.Kind == want.Kind && o.Name == want.Name })
		if i < 0 || objs[i].Detail != want.Detail {
			t.Fatalf("want %+v in %+v", want, objs)
		}
		ddl, err := s.DDL(ctx, objs[i])
		if err != nil || !strings.Contains(strings.ToUpper(ddl), "CREATE") || !strings.Contains(ddl, want.Name) {
			t.Errorf("%s DDL = %q, %v", want.Name, ddl, err)
		}
	}
	if _, err := s.DDL(ctx, model.DBObject{Schema: "shop", Name: "zz_gone", Kind: "procedure"}); err == nil || !strings.Contains(err.Error(), "isn't there any more") {
		t.Errorf("missing procedure err = %v", err)
	}
}
