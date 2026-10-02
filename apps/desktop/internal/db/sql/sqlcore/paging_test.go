package sqlcore

import (
	"context"
	"testing"
)

func TestEditorPagesLargeReads(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	if _, err := s.Run(ctx, `create table nums (n integer primary key);
		with recursive c(n) as (select 1 union all select n + 1 from c where n < 2500) insert into nums select n from c`); err != nil {
		t.Fatal(err)
	}
	res, err := s.Run(ctx, "select n from nums order by n desc -- newest first")
	if err != nil {
		t.Fatal(err)
	}
	first := res[0]
	if !first.Pageable || !first.HasMore || len(first.Rows) != EditorChunk || first.Rows[0][0] != int64(2500) {
		t.Fatalf("first chunk: pageable=%v more=%v rows=%d first=%v", first.Pageable, first.HasMore, len(first.Rows), first.Rows[0][0])
	}
	second, err := s.RunMore(ctx, first.Statement, EditorChunk)
	if err != nil || !second.HasMore || second.Rows[0][0] != int64(1500) {
		t.Fatalf("second chunk: %v %v %v", second.HasMore, second.Rows[0][0], err)
	}
	last, err := s.RunMore(ctx, first.Statement, 2*EditorChunk)
	if err != nil || last.HasMore || len(last.Rows) != 500 {
		t.Fatalf("last chunk: more=%v rows=%d err=%v", last.HasMore, len(last.Rows), err)
	}
}

func TestEditorKeepsUserLimitsAndCTEs(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	cases := map[string]int{
		"select * from users limit 2":                                       2,
		"with t as (select * from users where score > 1.5) select * from t": 2,
		"values (1), (2), (3)":                                              3,
		"select * from users order by id desc limit 1 offset 1":             1,
	}
	for q, want := range cases {
		res, err := s.Run(ctx, q)
		if err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if len(res[0].Rows) != want || res[0].HasMore {
			t.Errorf("%s: %d rows (more=%v), want %d", q, len(res[0].Rows), res[0].HasMore, want)
		}
	}
	for _, q := range []string{"pragma table_info(users)", "explain query plan select * from users", "insert into users (name) values ('z') returning id"} {
		res, err := s.Run(ctx, q)
		if err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if res[0].Pageable {
			t.Errorf("%s was wrapped", q)
		}
	}
}
