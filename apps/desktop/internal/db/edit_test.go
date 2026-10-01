package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func count(t *testing.T, s *Session, q string) any {
	t.Helper()
	res, err := s.Run(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	return res[0].Rows[0][0]
}

func TestApplyChangesSQLite(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	n, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: "main", Table: "users", Changes: []model.RowChange{
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{"name": "Ann O'Hara", "score": "7.5"}},
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": "2"}, Values: map[string]any{"avatar": nil, "score": nil}},
		{Kind: model.ChangeDelete, Key: map[string]any{"id": float64(3)}},
		{Kind: model.ChangeInsert, Values: map[string]any{"name": "dee"}},
	}})
	if err != nil || n != 4 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	if got := count(t, s, "select name || '|' || score from users where id = 1"); got != "Ann O'Hara|7.5" {
		t.Fatalf("update: %v", got)
	}
	if got := count(t, s, "select score is null from users where id = 2"); got != int64(1) {
		t.Fatalf("set null: %v", got)
	}
	// SQLite hands the freed id 3 to the inserted row, so check the deleted row by name.
	if got := count(t, s, "select count(*) from users where name = 'cy'"); got != int64(0) {
		t.Fatalf("delete: %v", got)
	}
	if got := count(t, s, "select score from users where name = 'dee'"); got != float64(0) {
		t.Fatalf("insert should take the column default, got %v", got)
	}
}

func TestApplyChangesIsAllOrNothing(t *testing.T) {
	s := openSQLite(t)
	_, err := s.ApplyChanges(context.Background(), model.ChangeSet{Schema: "main", Table: "users", Changes: []model.RowChange{
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{"name": "changed"}},
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(99)}, Values: map[string]any{"name": "ghost"}},
	}})
	var ae *ApplyError
	if !errors.As(err, &ae) || ae.Index != 1 || !errors.Is(err, ErrRowGone) {
		t.Fatalf("err = %v", err)
	}
	if got := count(t, s, "select name from users where id = 1"); got != "ann" {
		t.Fatalf("first change was not rolled back: %v", got)
	}
}

func TestApplyChangesRefusesUnsafeInput(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	cases := map[string]model.ChangeSet{
		"unknown column": {Schema: "main", Table: "users", Changes: []model.RowChange{
			{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{`name" = 'x'; --`: "x"}},
		}},
		"missing key": {Schema: "main", Table: "users", Changes: []model.RowChange{
			{Kind: model.ChangeDelete, Key: map[string]any{}},
		}},
		"binary column": {Schema: "main", Table: "users", Changes: []model.RowChange{
			{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{"avatar": "0xff"}},
		}},
		"no primary key": {Schema: "main", Table: "top_users", Changes: []model.RowChange{
			{Kind: model.ChangeDelete, Key: map[string]any{"id": float64(1)}},
		}},
	}
	for name, cs := range cases {
		if _, err := s.ApplyChanges(ctx, cs); err == nil {
			t.Errorf("%s: applied", name)
		}
	}
	if got := count(t, s, "select count(*) from users"); got != int64(3) {
		t.Fatalf("rows changed: %v", got)
	}
}

func TestPreviewChangesInlinesLiterals(t *testing.T) {
	s := openSQLite(t)
	sql, err := s.PreviewChanges(context.Background(), model.ChangeSet{Schema: "main", Table: "users", Changes: []model.RowChange{
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{"name": "it's", "score": nil}},
		{Kind: model.ChangeInsert, Values: map[string]any{}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`UPDATE "main"."users" SET "name" = 'it''s', "score" = NULL WHERE "id" = '1'`,
		`INSERT INTO "main"."users" DEFAULT VALUES`,
	}
	if strings.Join(sql, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(sql, "\n"))
	}
}

func TestReadOnlySessionRefusesEdits(t *testing.T) {
	rw := openSQLite(t)
	s, err := Open(context.Background(), "ro", model.Connection{Driver: model.SQLite, File: rw.Conn.File, ReadOnly: true}, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.ApplyChanges(context.Background(), model.ChangeSet{Schema: "main", Table: "users", Changes: []model.RowChange{
		{Kind: model.ChangeDelete, Key: map[string]any{"id": float64(1)}},
	}})
	var ro *ReadOnlyError
	if !errors.As(err, &ro) {
		t.Fatalf("err = %v", err)
	}
}

func TestEditExpressionsSQLite(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	update := func(v any) error {
		_, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: "main", Table: "users", Changes: []model.RowChange{
			{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{"name": v}},
		}})
		return err
	}
	if err := update(map[string]any{"$expr": "now"}); err != nil {
		t.Fatal(err)
	}
	if got := count(t, s, "select name glob '[0-9][0-9][0-9][0-9]-*' from users where id = 1"); got != int64(1) {
		t.Fatalf("now() was not written: %v", got)
	}
	if err := update(map[string]any{"$expr": "default"}); err == nil {
		t.Fatal("SQLite accepted SET … = DEFAULT")
	}
	if err := update(map[string]any{"$expr": "drop table users"}); err == nil {
		t.Fatal("an unknown expression was accepted")
	}
	sql, _ := s.PreviewChanges(ctx, model.ChangeSet{Schema: "main", Table: "users", Changes: []model.RowChange{
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{"name": map[string]any{"$expr": "now"}}},
	}})
	if len(sql) != 1 || !strings.Contains(sql[0], `"name" = CURRENT_TIMESTAMP`) {
		t.Fatalf("preview = %v", sql)
	}
}

func TestParseMySQLEnum(t *testing.T) {
	got := parseMySQLEnum("enum('new','it''s','a,b')")
	want := []string{"new", "it's", "a,b"}
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %q", got)
		}
	}
	if parseMySQLEnum("varchar(10)") != nil {
		t.Fatal("non-enum parsed")
	}
}
