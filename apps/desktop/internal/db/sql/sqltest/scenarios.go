package sqltest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func readOnly(t *testing.T, srv Server, c model.Connection) {
	c.ReadOnly = true
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	for i := 0; i < 6; i++ {
		if _, err := s.DB.ExecContext(ctx, "update customers set name = name where id = 1"); err == nil {
			t.Fatalf("attempt %d: write succeeded on a read-only connection", i)
		}
	}
	if _, err := s.Run(ctx, "set session characteristics as transaction read write"); err == nil {
		t.Fatal("editor let the session switch back to read write")
	}
	if _, err := s.Run(ctx, "select count(*) from customers"); err != nil {
		t.Fatal(err)
	}
}

func applyChanges(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	s.Run(ctx, "delete from customers where email like 'edit-test-%'")
	isNull := func(v any) bool { return v == true || v == int64(1) }
	table := func(changes ...model.RowChange) model.ChangeSet {
		return model.ChangeSet{Schema: srv.Schema, Table: "customers", Changes: changes}
	}

	insert := map[string]any{"email": "edit-test-1@example.com", "name": "Edit Test", "country": "NL", "is_active": srv.True, "meta": `{"plan":"team"}`}
	if _, err := s.ApplyChanges(ctx, table(model.RowChange{Kind: model.ChangeInsert, Values: insert})); err != nil {
		t.Fatalf("insert: %v", err)
	}
	res, err := s.Run(ctx, "select id from customers where email = 'edit-test-1@example.com'")
	if err != nil || len(res[0].Rows) != 1 {
		t.Fatalf("inserted row: %v %v", res, err)
	}
	id := res[0].Rows[0][0]

	if _, err := s.ApplyChanges(ctx, table(model.RowChange{Kind: model.ChangeUpdate, Key: map[string]any{"id": id},
		Values: map[string]any{"name": "Edit Test", "country": nil, "created_at": "2026-01-02 03:04:05"}})); err != nil {
		t.Fatalf("update: %v", err)
	}
	res, _ = s.Run(ctx, "select country is null from customers where email = 'edit-test-1@example.com'")
	if v := res[0].Rows[0][0]; !isNull(v) {
		t.Fatalf("country not nulled: %v", v)
	}

	if _, err := s.ApplyChanges(ctx, table(
		model.RowChange{Kind: model.ChangeUpdate, Key: map[string]any{"id": id}, Values: map[string]any{"name": "should roll back"}},
		model.RowChange{Kind: model.ChangeUpdate, Key: map[string]any{"id": id}, Values: map[string]any{"email": "user2@example.com"}},
	)); err == nil {
		t.Fatal("unique violation was applied")
	}
	res, _ = s.Run(ctx, "select name from customers where email = 'edit-test-1@example.com'")
	if res[0].Rows[0][0] != "Edit Test" {
		t.Fatalf("not rolled back: %v", res[0].Rows[0][0])
	}

	if _, err := s.ApplyChanges(ctx, table(model.RowChange{Kind: model.ChangeUpdate, Key: map[string]any{"id": id},
		Values: map[string]any{"country": "NLX"}})); err == nil {
		t.Fatal("an over-long value was written")
	}
	res, _ = s.Run(ctx, "select country is null from customers where email = 'edit-test-1@example.com'")
	if v := res[0].Rows[0][0]; !isNull(v) {
		t.Fatalf("country changed to %v", v)
	}

	if _, err := s.ApplyChanges(ctx, table(model.RowChange{Kind: model.ChangeDelete, Key: map[string]any{"id": id}})); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func tableFilters(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	page := func(f ...model.Filter) model.TablePage {
		t.Helper()
		p, err := s.TablePage(context.Background(), model.TableQuery{Schema: srv.Schema, Table: "customers", Limit: 5000, Filters: f})
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
	if p := page(model.Filter{Column: "email", Op: "contains", Value: "USER12@"}); len(p.Result.Rows) != 1 {
		t.Fatalf("contains: %d rows", len(p.Result.Rows))
	}
	if p := page(model.Filter{Column: "created_at", Op: "<", Value: "2000-01-01"}); len(p.Result.Rows) != 0 {
		t.Fatalf("date compare: %d rows", len(p.Result.Rows))
	}
	if p := page(model.Filter{Column: "country", Op: "=", Value: "USA"}); len(p.Result.Rows) != 0 {
		t.Fatalf("country = USA matched %d rows", len(p.Result.Rows))
	}
	if p := page(); p.DefaultOrder[0] != "id" {
		t.Fatalf("default order = %v", p.DefaultOrder)
	}
}

func editorPaging(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	res, err := s.Run(context.Background(), "select * from orders; select 1 as a, 2 as a")
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].Pageable || !res[0].HasMore || len(res[0].Rows) != sqlcore.EditorChunk {
		t.Fatalf("orders: pageable=%v more=%v rows=%d", res[0].Pageable, res[0].HasMore, len(res[0].Rows))
	}
	if len(res[1].Columns) != 2 {
		t.Fatalf("duplicate names: %v", res[1].Columns)
	}
	more, err := s.RunMore(context.Background(), res[0].Statement, sqlcore.EditorChunk)
	if err != nil || len(more.Rows) != sqlcore.EditorChunk || more.Rows[0][0] == res[0].Rows[0][0] {
		t.Fatalf("second chunk: %d rows, %v", len(more.Rows), err)
	}
}

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

func syntaxErrors(t *testing.T, srv Server, c model.Connection) {
	s := Open(t, srv.Dialect, c)
	ctx := context.Background()
	problems, err := s.CheckSyntax(ctx, "select 1; selec 2; select * from customers where; select * from nope_table")
	if err != nil {
		t.Fatal(err)
	}
	want := []model.SyntaxProblem{{Index: 1, Position: 0}, {Index: 2, Position: len("select * from customers where")}}
	if len(problems) != len(want) {
		t.Fatalf("problems = %+v", problems)
	}
	for i, p := range problems {
		if p.Index != want[i].Index || p.Position != want[i].Position || p.Message == "" {
			t.Errorf("problem %d = %+v, want index %d at %d", i, p, want[i].Index, want[i].Position)
		}
	}
	_, err = s.Run(ctx, "select 1;\nselect * from customers wher id = 1")
	var se *db.StatementError
	if !errors.As(err, &se) || se.Index != 1 || se.Position < 0 {
		t.Fatalf("run error = %#v", err)
	}
}
