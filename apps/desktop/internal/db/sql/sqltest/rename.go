package sqltest

import (
	"context"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func renameTable(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	drop := func() {
		s.Run(context.Background(), "drop table if exists zz_old")
		s.Run(context.Background(), "drop table if exists zz_new")
	}
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, "create table zz_old ("+srv.KeyColumn+")"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(ctx, s.RenameTableSQL(srv.Schema, "zz_old", "zz_new")); err != nil {
		t.Fatal(err)
	}
	tables, err := s.Tables(ctx, srv.Schema)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tb := range tables {
		found = found || tb.Name == "zz_new"
	}
	if !found {
		t.Fatalf("zz_new isn't in %s after the rename: %+v", srv.Schema, tables)
	}
}
