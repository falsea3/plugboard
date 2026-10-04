package sqltest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func structure(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	s.Run(ctx, "drop table if exists zz_structure")
	t.Cleanup(func() { s.Run(context.Background(), "drop table if exists zz_structure") })
	if _, err := s.Run(ctx, "create table zz_structure ("+srv.KeyColumn+", code varchar(20), note varchar(50) default 'x'); "+
		"insert into zz_structure (code) values ('12'), ('34')"); err != nil {
		t.Fatal(err)
	}
	change := func(changes ...model.ColumnChange) (int, bool, error) {
		return s.ApplyStructure(ctx, model.StructureChange{Schema: srv.Schema, Table: "zz_structure", Changes: changes})
	}

	if _, err := s.TablePage(ctx, model.TableQuery{Schema: srv.Schema, Table: "zz_structure"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(ctx, "select * from zz_structure"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := change(
		model.ColumnChange{Kind: model.ChangeInsert, Name: Ptr("status"), Type: Ptr("varchar(10)"), Nullable: Ptr(false), DefaultSet: true, Default: Ptr("'new'")},
		model.ColumnChange{Kind: model.ChangeUpdate, Column: "code", Name: Ptr("number"), Type: Ptr("integer"), Nullable: Ptr(false)},
		model.ColumnChange{Kind: model.ChangeUpdate, Column: "note", DefaultSet: true, Default: nil, Nullable: Ptr(true)},
	); err != nil {
		t.Fatal(err)
	}
	if c := Column(t, s, srv.Schema, "zz_structure", "number"); c.Nullable || !strings.HasPrefix(strings.ToLower(c.Type), "int") {
		t.Fatalf("altered column = %+v", c)
	}
	if c := Column(t, s, srv.Schema, "zz_structure", "status"); c.Nullable || c.Default == nil || !strings.Contains(*c.Default, "'new'") {
		t.Fatalf("added column = %+v", c)
	}
	if c := Column(t, s, srv.Schema, "zz_structure", "note"); c.Default != nil {
		t.Fatalf("default not dropped: %+v", c)
	}
	page, err := s.TablePage(ctx, model.TableQuery{Schema: srv.Schema, Table: "zz_structure"})
	if err != nil || len(page.Result.Columns) != 4 {
		t.Fatalf("page after the change: %v %v", page.Result.Columns, err)
	}
	if res, err := s.Run(ctx, "select * from zz_structure"); err != nil || len(res[0].Columns) != 4 {
		t.Fatalf("editor after the change: %v", err)
	}
	res, err := s.Run(ctx, "select number, status from zz_structure order by id")
	if err != nil || res[0].Rows[0][1] != "new" {
		t.Fatalf("rows after the change: %v %v", res, err)
	}

	applied, partial, err := change(
		model.ColumnChange{Kind: model.ChangeInsert, Name: Ptr("extra"), Type: Ptr("int")},
		model.ColumnChange{Kind: model.ChangeUpdate, Column: "status", Type: Ptr("integer")},
	)
	var ae *db.ApplyError
	if !errors.As(err, &ae) || ae.Index != 1 {
		t.Fatalf("err = %v", err)
	}
	_, extraErr := s.Run(ctx, "select extra from zz_structure")
	if srv.Dialect.TransactionalDDL() && (extraErr == nil || applied != 0 || partial) {
		t.Fatalf("part of a failed change stayed: applied=%d partial=%v", applied, partial)
	}
	if !srv.Dialect.TransactionalDDL() && (extraErr != nil || applied != 1 || !partial) {
		t.Fatalf("the applied first change wasn't reported: applied=%d partial=%v err=%v", applied, partial, extraErr)
	}

	if _, _, err := change(model.ColumnChange{Kind: model.ChangeDelete, Column: "status"}); err != nil {
		t.Fatal(err)
	}
}

func tableBusy(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	s.Run(ctx, "drop table if exists zz_busy")
	t.Cleanup(func() { s.Run(context.Background(), "drop table if exists zz_busy") })
	if _, err := s.Run(ctx, "create table zz_busy (id int primary key)"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(ctx, "begin; select * from zz_busy"); err != nil {
		t.Fatal(err)
	}
	add := model.StructureChange{Schema: srv.Schema, Table: "zz_busy", Changes: []model.ColumnChange{
		{Kind: model.ChangeInsert, Name: Ptr("more"), Type: Ptr("int")},
	}}
	started := time.Now()
	_, _, err := s.ApplyStructure(ctx, add)
	if !errors.Is(err, db.ErrTableBusy) || time.Since(started) > 15*time.Second {
		t.Fatalf("err = %v after %v", err, time.Since(started))
	}
	if _, err := s.Run(ctx, "rollback"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ApplyStructure(ctx, add); err != nil {
		t.Fatalf("once the transaction is over: %v", err)
	}
}

func columnKinds(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	cols, err := s.Columns(ctx, srv.Schema, "customers")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]model.ColumnKind{"id": model.KindNumber, "email": model.KindText, "is_active": model.KindBool, "created_at": model.KindDateTime}
	for _, col := range cols {
		if k, ok := want[col.Name]; ok && col.Kind != k {
			t.Errorf("catalog %s (%s) = %q, want %q", col.Name, col.Type, col.Kind, k)
		}
	}
	res, err := s.Run(ctx, "select id, email, created_at from customers where id = 1")
	if err != nil {
		t.Fatal(err)
	}
	for i, k := range []model.ColumnKind{model.KindNumber, model.KindText, model.KindDateTime} {
		if col := res[0].Columns[i]; col.Kind != k {
			t.Errorf("result %s (%s) = %q, want %q", col.Name, col.Type, col.Kind, k)
		}
	}
}
