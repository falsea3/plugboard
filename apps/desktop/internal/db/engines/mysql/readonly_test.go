package mysql_test

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
)

func TestReadOnlyReason(t *testing.T) {
	allowed := []string{
		"select 'it\\'s; delete from t' from t",
		"select 1 # delete from t",
		"select 1 -- delete from t",
		"select count(*) into @n from t",
		"show tables",
		"describe t",
		"set names utf8mb4",
		"use shop",
		"start transaction",
	}
	refused := []string{
		"reset master",
		"reset persist",
		"start replica",
		"set global max_connections = 1",
		"set persist max_connections = 1",
		"set @@global.read_only = 0",
		"set password = 'x'",
		"set role admin",
		"set @@session.transaction_read_only = 0",
		"select * from t /*! into outfile '/tmp/x' */",
		"select 1--1 into outfile '/tmp/x'",
		"pragma table_info(t)",
	}
	for _, s := range allowed {
		if v := sqlcore.ReadOnlyReason(mysql.Dialect{}, s); v != "" {
			t.Errorf("ReadOnlyReason(%q) = %q, want allowed", s, v)
		}
	}
	for _, s := range refused {
		if sqlcore.ReadOnlyReason(mysql.Dialect{}, s) == "" {
			t.Errorf("ReadOnlyReason(%q) allowed it", s)
		}
	}
}
