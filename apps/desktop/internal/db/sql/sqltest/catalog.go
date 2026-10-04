package sqltest

import (
	"context"
	"slices"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func relations(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	drop := func() {
		s.Run(context.Background(), "drop table if exists zz_child")
		s.Run(context.Background(), "drop table if exists zz_parent")
	}
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, "create table zz_parent ("+srv.KeyColumn+"); "+
		"create table zz_child ("+srv.KeyColumn+", parent_id int, foreign key (parent_id) references zz_parent(id) on delete cascade)"); err != nil {
		t.Fatal(err)
	}
	d, err := s.Diagram(ctx, srv.Schema)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := FindRelation(d, "zz_child")
	if !ok {
		t.Fatalf("no relation from zz_child in %+v", d.Relations)
	}
	if r.RefTable != "zz_parent" || r.RefSchema != srv.Schema || !slices.Equal(r.Columns, []string{"parent_id"}) || !slices.Equal(r.RefColumns, []string{"id"}) || r.OnDelete != "CASCADE" {
		t.Errorf("zz_child relation = %+v", r)
	}
	if r, ok := FindRelation(d, "orders"); !ok || r.RefTable != "customers" || !slices.Equal(r.Columns, []string{"customer_id"}) {
		t.Errorf("orders relation = %+v, %v", r, ok)
	}
	for _, tb := range d.Tables {
		if tb.Name == "zz_parent" {
			if len(tb.Columns) != 1 || !tb.Columns[0].PrimaryKey {
				t.Errorf("zz_parent columns = %+v", tb.Columns)
			}
			return
		}
	}
	t.Errorf("zz_parent missing from %d tables", len(d.Tables))
}

func indexes(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	drop := func() { s.Run(context.Background(), "drop table if exists zz_idx") }
	drop()
	t.Cleanup(drop)
	if _, err := s.Run(ctx, "create table zz_idx ("+srv.KeyColumn+", code varchar(20), title varchar(50), status int); "+
		"create unique index zz_idx_code_title on zz_idx (code, title); "+
		"create index zz_idx_status on zz_idx (status)"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Indexes(ctx, srv.Schema, "zz_idx")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || !got[0].Primary || !got[0].Unique || !slices.Equal(got[0].Columns, []string{"id"}) {
		t.Fatalf("indexes = %+v, want the primary key first", got)
	}
	byName := map[string]model.Index{}
	for _, ix := range got {
		byName[ix.Name] = ix
	}
	if ix := byName["zz_idx_code_title"]; !ix.Unique || ix.Primary || !slices.Equal(ix.Columns, []string{"code", "title"}) || ix.Method != "btree" {
		t.Errorf("unique index = %+v", ix)
	}
	if ix := byName["zz_idx_status"]; ix.Unique || !slices.Equal(ix.Columns, []string{"status"}) {
		t.Errorf("plain index = %+v", ix)
	}
}
