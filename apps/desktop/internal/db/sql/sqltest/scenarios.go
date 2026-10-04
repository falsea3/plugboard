package sqltest

import (
	"context"
	"errors"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
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
