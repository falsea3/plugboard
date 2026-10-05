package sqltest

import (
	"context"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func objects(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	drop := func() {
		s.Run(context.Background(), "drop view if exists zz_ddl_view")
		s.Run(context.Background(), "drop table if exists zz_ddl")
	}
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, "create table zz_ddl ("+srv.KeyColumn+", code varchar(20) not null, note varchar(50) default 'x'); "+
		"create index zz_ddl_code on zz_ddl (code); create view zz_ddl_view as select id, code from zz_ddl"); err != nil {
		t.Fatal(err)
	}
	table := model.DBObject{Schema: srv.Schema, Name: "zz_ddl", Kind: "table"}
	view := model.DBObject{Schema: srv.Schema, Name: "zz_ddl_view", Kind: "view"}
	tableDDL, err := s.DDL(ctx, table)
	if err != nil {
		t.Fatal(err)
	}
	viewDDL, err := s.DDL(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"zz_ddl", "code", "zz_ddl_code"} {
		if !strings.Contains(tableDDL, want) {
			t.Errorf("table DDL lacks %q:\n%s", want, tableDDL)
		}
	}
	drop()
	if _, err := s.Run(ctx, tableDDL+"\n"+viewDDL); err != nil {
		t.Fatalf("the DDL doesn't run back:\n%s\n%s\n%v", tableDDL, viewDDL, err)
	}
	if again, err := s.DDL(ctx, table); err != nil || again != tableDDL {
		t.Errorf("DDL after a round trip:\n%s\nwant\n%s\n%v", again, tableDDL, err)
	}
	if _, err := s.DDL(ctx, model.DBObject{Schema: srv.Schema, Name: "zz_nothing", Kind: "view"}); err == nil || !strings.Contains(err.Error(), "isn't there any more") {
		t.Errorf("missing object err = %v", err)
	}
	if _, err := s.Objects(ctx, srv.Schema); err != nil {
		t.Fatal(err)
	}
}
