package db

import (
	"context"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func names(t *testing.T, s *Session, filters ...model.Filter) []any {
	t.Helper()
	p, err := s.TablePage(context.Background(), model.TableQuery{Schema: "main", Table: "users", Filters: filters, OrderBy: "id"})
	if err != nil {
		t.Fatal(err)
	}
	out := []any{}
	for _, r := range p.Result.Rows {
		out = append(out, r[1])
	}
	return out
}

func TestTableFiltersSQLite(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	if _, err := s.Run(ctx, "insert into users (name, score) values ('50%_off', null), ('Annabel', 4), ('wow!', 5)"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		f    []model.Filter
		want []any
	}{
		{[]model.Filter{{Column: "name", Op: "=", Value: "bob"}}, []any{"bob"}},
		{[]model.Filter{{Column: "score", Op: ">=", Value: "2"}}, []any{"bob", "cy", "Annabel", "wow!"}},
		{[]model.Filter{{Column: "name", Op: "contains", Value: "ANN"}}, []any{"ann", "Annabel"}},
		{[]model.Filter{{Column: "name", Op: "starts", Value: "50%_"}}, []any{"50%_off"}},
		{[]model.Filter{{Column: "name", Op: "contains", Value: "%"}}, []any{"50%_off"}},
		{[]model.Filter{{Column: "name", Op: "ends", Value: "y"}}, []any{"cy"}},
		{[]model.Filter{{Column: "name", Op: "ends", Value: "!"}}, []any{"wow!"}},
		{[]model.Filter{{Column: "name", Op: "not_contains", Value: "a"}}, []any{"bob", "cy", "50%_off", "wow!"}},
		{[]model.Filter{{Column: "id", Op: "in", Value: "1, 3"}}, []any{"ann", "cy"}},
		{[]model.Filter{{Column: "id", Op: "not_in", Value: "1,3"}}, []any{"bob", "50%_off", "Annabel", "wow!"}},
		{[]model.Filter{{Column: "score", Op: "null"}}, []any{"50%_off"}},
		{[]model.Filter{{Column: "name", Op: "empty"}}, []any{}},
		{[]model.Filter{{Column: "name", Op: "not_empty"}, {Column: "id", Op: "<", Value: "3"}}, []any{"ann", "bob"}},
		{[]model.Filter{{Column: "score", Op: "not_null"}, {Column: "score", Op: "<", Value: "3"}}, []any{"ann", "bob"}},
		{[]model.Filter{{Column: "name", Op: "!=", Value: "ann"}, {Column: "id", Op: "<=", Value: "3"}}, []any{"bob", "cy"}},
	}
	for _, tc := range cases {
		got := names(t, s, tc.f...)
		if len(got) != len(tc.want) {
			t.Errorf("%+v: got %v, want %v", tc.f, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%+v: got %v, want %v", tc.f, got, tc.want)
				break
			}
		}
	}
}

func TestTableFiltersRefuseBadInput(t *testing.T) {
	s := openSQLite(t)
	for _, f := range []model.Filter{
		{Column: `name" = '' or 1=1 --`, Op: "=", Value: "x"},
		{Column: "name", Op: "; drop table users", Value: "x"},
		{Column: "id", Op: "in", Value: " , "},
	} {
		if _, err := s.TablePage(context.Background(), model.TableQuery{Schema: "main", Table: "users", Filters: []model.Filter{f}}); err == nil {
			t.Errorf("%+v was accepted", f)
		}
	}
	if _, err := s.TablePage(context.Background(), model.TableQuery{Schema: "main", Table: "users", OrderBy: "nope"}); err == nil {
		t.Error("unknown sort column was accepted")
	}
}

func TestDefaultOrderIsPrimaryKey(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	// Re-insert row 1 last so storage order differs from key order.
	if _, err := s.Run(ctx, "delete from users where id = 1; insert into users (id, name) values (1, 'ann again')"); err != nil {
		t.Fatal(err)
	}
	p, err := s.TablePage(ctx, model.TableQuery{Schema: "main", Table: "users"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.DefaultOrder) != 1 || p.DefaultOrder[0] != "id" || p.Result.Rows[0][0] != int64(1) {
		t.Fatalf("order = %v, first row = %v", p.DefaultOrder, p.Result.Rows[0])
	}
	view, err := s.TablePage(ctx, model.TableQuery{Schema: "main", Table: "top_users"})
	if err != nil || view.DefaultOrder != nil {
		t.Fatalf("views have no key to order by: %v %v", view.DefaultOrder, err)
	}
}
