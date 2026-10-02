package sqltext_test

import (
	"reflect"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
)

var (
	pg = postgres.Dialect{}.Syntax()
	my = mysql.Dialect{}.Syntax()
)

func TestSplitStatements(t *testing.T) {
	cases := []struct {
		name string
		syn  sqltext.Syntax
		in   string
		want []string
	}{
		{"single", pg, "select 1", []string{"select 1"}},
		{"two", pg, "select 1; select 2;", []string{"select 1", "select 2"}},
		{"empty pieces", pg, ";; select 1 ;\n;", []string{"select 1"}},
		{"semicolon in string", pg, "select 'a;b'; select 2", []string{"select 'a;b'", "select 2"}},
		{"doubled quote", pg, "select 'it''s;'; select 2", []string{"select 'it''s;'", "select 2"}},
		{"quoted ident", pg, `select "a;b" from t; select 2`, []string{`select "a;b" from t`, "select 2"}},
		{"line comment", pg, "select 1 -- ; nope\n; select 2", []string{"select 1 -- ; nope", "select 2"}},
		{"block comment", pg, "select /* ; */ 1; select 2", []string{"select /* ; */ 1", "select 2"}},
		{"comment only", pg, "-- just a note\n/* and this */", nil},
		{
			"dollar quoted", pg,
			"create function f() returns int as $$ begin return 1; end; $$ language plpgsql; select f()",
			[]string{"create function f() returns int as $$ begin return 1; end; $$ language plpgsql", "select f()"},
		},
		{
			"tagged dollar", pg,
			"do $body$ begin perform 1; end $body$; select 2",
			[]string{"do $body$ begin perform 1; end $body$", "select 2"},
		},
		{"placeholder is not a tag", pg, "select $1; select 2", []string{"select $1", "select 2"}},
		{"dollar inside a name", pg, "select a$b$ from t; select 2", []string{"select a$b$ from t", "select 2"}},
		{"pg backslash is literal", pg, `select 'C:\'; select 2`, []string{`select 'C:\'`, "select 2"}},
		{"pg hash is an operator", pg, "select d #> '{a}' from t; select 2", []string{"select d #> '{a}' from t", "select 2"}},
		{"pg E-string escapes", pg, `select E'it\'s; x'; select 2`, []string{`select E'it\'s; x'`, "select 2"}},
		{"pg E-string needs a word start", pg, `select type'a\'; select 2`, []string{`select type'a\'`, "select 2"}},

		{"mysql backslash escape", my, `select 'a\';b'; select 2`, []string{`select 'a\';b'`, "select 2"}},
		{"mysql double-quoted string", my, `select "a\";b"; select 2`, []string{`select "a\";b"`, "select 2"}},
		{"mysql backtick", my, "select `a;b` from t; select 2", []string{"select `a;b` from t", "select 2"}},
		{"mysql backtick has no escapes", my, "select `a\\`; select 2", []string{"select `a\\`", "select 2"}},
		{"mysql hash comment", my, "select 1 # ; nope\n; select 2", []string{"select 1 # ; nope", "select 2"}},
		{"mysql -- needs a space", my, "select 1--1; select 2", []string{"select 1--1", "select 2"}},
		{"mysql -- comment", my, "select 1 -- ; nope\n; select 2", []string{"select 1 -- ; nope", "select 2"}},
		{"pg -- needs no space", pg, "select 1--; nope\n; select 2", []string{"select 1--; nope", "select 2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sqltext.Split(tc.in, tc.syn)
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
		"show tables",
	}
	no := []string{
		"insert into t values (1)",
		"update t set returning_flag = 1",
		"create table t (id int)",
		"begin",
		"PRAGMA table_info(t)",
	}
	for _, s := range yes {
		if !sqltext.ReturnsRows(s) {
			t.Errorf("ReturnsRows(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if sqltext.ReturnsRows(s) {
			t.Errorf("ReturnsRows(%q) = true, want false", s)
		}
	}
	if !sqltext.ReturnsRows("PRAGMA table_info(t)", "PRAGMA") {
		t.Error("an engine's own row-returning statement")
	}
}

func TestStrip(t *testing.T) {
	cases := []struct {
		syn      sqltext.Syntax
		in, want string
	}{
		{pg, `select 'drop' -- delete`, `select    `},
		{pg, `select "update" from t`, `select "update" from t`},
		{my, `select "update" # delete`, `select    `},
		{my, `select 1 /*! into outfile 'x' */`, `select 1   into outfile 'x'  `},
		{my, `select 1--1`, `select 1--1`},
		{pg, `do $$ delete $$`, `do  `},
	}
	for _, c := range cases {
		if got := sqltext.Strip(c.in, c.syn); got != c.want {
			t.Errorf("Strip(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
