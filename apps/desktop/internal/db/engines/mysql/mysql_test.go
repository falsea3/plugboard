package mysql

import (
	"database/sql"
	"testing"
)

func TestVersion(t *testing.T) {
	for _, tc := range []struct {
		version       string
		rename, asSQL bool
	}{
		{"MySQL 8.0.36-0ubuntu0.22.04.1", true, false},
		{"MySQL 5.7.44-log", false, false},
		{"MariaDB 10.11.6-MariaDB-1:10.11.6+maria~ubu2204", true, true},
		{"MariaDB 10.4.32-MariaDB", false, true},
		{"MariaDB 10.2.6-MariaDB", false, false},
	} {
		d := Dialect{}.ForVersion(tc.version).(Dialect)
		if d.canRenameColumn() != tc.rename || d.writesDefaultsAsSQL() != tc.asSQL {
			t.Errorf("%s: rename=%v asSQL=%v (%+v)", tc.version, d.canRenameColumn(), d.writesDefaultsAsSQL(), d)
		}
	}
}

func TestDefaultSQL(t *testing.T) {
	mysql8 := Dialect{}.ForVersion("MySQL 8.4.3").(Dialect)
	mariaDB := Dialect{}.ForVersion("MariaDB 10.11.6-MariaDB").(Dialect)
	null := sql.NullString{}
	val := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	for _, tc := range []struct {
		d               Dialect
		raw             sql.NullString
		dataType, extra string
		want            string
	}{
		{mysql8, val("pro"), "varchar", "", "'pro'"},
		{mysql8, val("'lead"), "varchar", "", "'''lead'"},
		{mysql8, val(`back\slash`), "varchar", "", `'back\\slash'`},
		{mysql8, val("1.50"), "decimal", "", "1.50"},
		{mysql8, val("0x6162"), "varbinary", "", "0x6162"},
		{mysql8, val("CURRENT_TIMESTAMP(3)"), "timestamp", "DEFAULT_GENERATED on update CURRENT_TIMESTAMP(3)", "CURRENT_TIMESTAMP(3)"},
		{mysql8, val("CURRENT_TIMESTAMP"), "timestamp", "on update CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP"},
		{mysql8, val(`concat(_utf8mb4\'it\\\'s\',_utf8mb4\'a\\\\b\')`), "varchar", "DEFAULT_GENERATED", `(concat(_utf8mb4'it\'s',_utf8mb4'a\\b'))`},
		{mysql8, null, "varchar", "", ""},
		{mariaDB, val("'pro'"), "varchar", "", "'pro'"},
		{mariaDB, val("current_timestamp()"), "datetime", "on update current_timestamp()", "current_timestamp()"},
		{mariaDB, val("NULL"), "varchar", "", ""},
	} {
		got := tc.d.defaultSQL(tc.raw, tc.dataType, tc.extra)
		if (got == nil) != (tc.want == "") || (got != nil && *got != tc.want) {
			t.Errorf("defaultSQL(%q, %s) = %v, want %s", tc.raw.String, tc.dataType, show(got), tc.want)
		}
	}
}

func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestParseEnum(t *testing.T) {
	got := parseEnum("enum('new','it''s','a,b')")
	want := []string{"new", "it's", "a,b"}
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %q", got)
		}
	}
	if parseEnum("varchar(10)") != nil {
		t.Fatal("non-enum parsed")
	}
}
