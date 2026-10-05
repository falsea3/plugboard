package sqlite_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestObjectsAndDDL(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	if _, err := s.Run(ctx, `create table zz_obj (id integer primary key, n int);
		create index zz_obj_n on zz_obj (n);
		create trigger zz_obj_touch after update on zz_obj begin select 1; end;
		create view zz_obj_view as select id from zz_obj`); err != nil {
		t.Fatal(err)
	}
	objs, err := s.Objects(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 1 || !reflect.DeepEqual(objs[0], model.DBObject{Schema: "main", Name: "zz_obj_touch", Kind: "trigger", Detail: "zz_obj"}) {
		t.Fatalf("objects = %+v", objs)
	}
	table, err := s.DDL(ctx, model.DBObject{Schema: "main", Name: "zz_obj", Kind: "table"})
	if err != nil {
		t.Fatal(err)
	}
	want := "CREATE TABLE zz_obj (id integer primary key, n int);\n\nCREATE INDEX zz_obj_n on zz_obj (n);\nCREATE TRIGGER zz_obj_touch after update on zz_obj begin select 1; end;"
	if table != want {
		t.Errorf("table DDL = %q", table)
	}
	if view, err := s.DDL(ctx, model.DBObject{Schema: "main", Name: "zz_obj_view", Kind: "view"}); err != nil || !strings.HasPrefix(strings.ToUpper(view), "CREATE VIEW") {
		t.Errorf("view DDL = %q, %v", view, err)
	}
	if _, err := s.DDL(ctx, model.DBObject{Schema: "main", Name: "nope", Kind: "table"}); err == nil || !strings.Contains(err.Error(), "isn't there any more") {
		t.Errorf("missing table err = %v", err)
	}
}

func TestRenameTable(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	if _, err := s.Run(ctx, s.RenameTableSQL("main", "users", "people")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(ctx, "select count(*) from people"); err != nil {
		t.Fatal(err)
	}
}
