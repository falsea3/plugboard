package sqltest

import (
	"context"
	"slices"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func indexSQL(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	drop := func() { s.Run(context.Background(), "drop table if exists zz_ix") }
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, "create table zz_ix ("+srv.KeyColumn+", a int, b varchar(20))"); err != nil {
		t.Fatal(err)
	}
	create := s.CreateIndexSQL(srv.Schema, "zz_ix", model.NewIndex{Name: "zz_ix_a_b", Columns: []string{"a", "b"}, Unique: true})
	if _, err := s.Run(ctx, create); err != nil {
		t.Fatalf("%s: %v", create, err)
	}
	has := func() bool {
		ixs, err := s.Indexes(ctx, srv.Schema, "zz_ix")
		if err != nil {
			t.Fatal(err)
		}
		return slices.ContainsFunc(ixs, func(ix model.Index) bool {
			return ix.Name == "zz_ix_a_b" && ix.Unique && slices.Equal(ix.Columns, []string{"a", "b"})
		})
	}
	if !has() {
		t.Fatalf("no unique index on (a, b) after %s", create)
	}
	drop2 := s.DropIndexSQL(srv.Schema, "zz_ix", "zz_ix_a_b")
	if _, err := s.Run(ctx, drop2); err != nil {
		t.Fatalf("%s: %v", drop2, err)
	}
	if has() {
		t.Fatalf("the index is still there after %s", drop2)
	}
}
