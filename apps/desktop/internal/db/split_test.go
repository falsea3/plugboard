package db

import (
	"reflect"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	cases := []struct {
		name  string
		mysql bool
		in    string
		want  []string
	}{
		{"single", false, "select 1", []string{"select 1"}},
		{"two", false, "select 1; select 2;", []string{"select 1", "select 2"}},
		{"empty pieces", false, ";; select 1 ;\n;", []string{"select 1"}},
		{"semicolon in string", false, "select 'a;b'; select 2", []string{"select 'a;b'", "select 2"}},
		{"doubled quote", false, "select 'it''s;'; select 2", []string{"select 'it''s;'", "select 2"}},
		{"quoted ident", false, `select "a;b" from t; select 2`, []string{`select "a;b" from t`, "select 2"}},
		{"line comment", false, "select 1 -- ; nope\n; select 2", []string{"select 1 -- ; nope", "select 2"}},
		{"block comment", false, "select /* ; */ 1; select 2", []string{"select /* ; */ 1", "select 2"}},
		{"comment only", false, "-- just a note\n/* and this */", nil},
		{
			"dollar quoted", false,
			"create function f() returns int as $$ begin return 1; end; $$ language plpgsql; select f()",
			[]string{"create function f() returns int as $$ begin return 1; end; $$ language plpgsql", "select f()"},
		},
		{
			"tagged dollar", false,
			"do $body$ begin perform 1; end $body$; select 2",
			[]string{"do $body$ begin perform 1; end $body$", "select 2"},
		},
		{"placeholder is not a tag", false, "select $1; select 2", []string{"select $1", "select 2"}},
		{"dollar inside a name", false, "select a$b$ from t; select 2", []string{"select a$b$ from t", "select 2"}},
		{"pg backslash is literal", false, `select 'C:\'; select 2`, []string{`select 'C:\'`, "select 2"}},
		{"pg hash is an operator", false, "select d #> '{a}' from t; select 2", []string{"select d #> '{a}' from t", "select 2"}},
		{"pg E-string escapes", false, `select E'it\'s; x'; select 2`, []string{`select E'it\'s; x'`, "select 2"}},
		{"pg E-string needs a word start", false, `select type'a\'; select 2`, []string{`select type'a\'`, "select 2"}},

		{"mysql backslash escape", true, `select 'a\';b'; select 2`, []string{`select 'a\';b'`, "select 2"}},
		{"mysql double-quoted string", true, `select "a\";b"; select 2`, []string{`select "a\";b"`, "select 2"}},
		{"mysql backtick", true, "select `a;b` from t; select 2", []string{"select `a;b` from t", "select 2"}},
		{"mysql backtick has no escapes", true, "select `a\\`; select 2", []string{"select `a\\`", "select 2"}},
		{"mysql hash comment", true, "select 1 # ; nope\n; select 2", []string{"select 1 # ; nope", "select 2"}},
		{"mysql -- needs a space", true, "select 1--1; select 2", []string{"select 1--1", "select 2"}},
		{"mysql -- comment", true, "select 1 -- ; nope\n; select 2", []string{"select 1 -- ; nope", "select 2"}},
		{"pg -- needs no space", false, "select 1--; nope\n; select 2", []string{"select 1--; nope", "select 2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitStatements(tc.in, tc.mysql)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReturnsRows(t *testing.T) {
	yes := []string{
		"select 1",
		"  -- hi\n SELECT 1",
		"/* c */ with x as (select 1) select * from x",
		"(select 1) union (select 2)",
		"insert into t values (1) returning id",
		"PRAGMA table_info(t)",
		"show tables",
	}
	no := []string{
		"insert into t values (1)",
		"update t set returning_flag = 1",
		"create table t (id int)",
		"begin",
	}
	for _, s := range yes {
		if !returnsRows(s) {
			t.Errorf("returnsRows(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if returnsRows(s) {
			t.Errorf("returnsRows(%q) = true, want false", s)
		}
	}
}
