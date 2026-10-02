package postgres_test

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
)

func TestReadOnlyReason(t *testing.T) {
	allowed := []string{
		"select * from t",
		"(select 1) union (select 2)",
		"select 'drop table t', \"update\" from t",
		"-- delete everything\nselect 1",
		"with x as (select 1) select * from x",
		"explain select * from t",
		"explain analyze select * from t",
		"show search_path",
		"begin",
		"start transaction read only",
		"commit",
		"set statement_timeout = 5000",
		"select current_setting('transaction_read_only')",
		"reset all",
	}
	refused := []string{
		"insert into t values (1)",
		"UPDATE t SET a = 1",
		"/* x */ delete from t",
		"drop table t",
		"truncate t",
		"grant select on t to bob",
		"with d as (delete from t returning *) select * from d",
		"explain analyze delete from t",
		"select * into backup from t",
		"select * from t for update",
		"set transaction read write",
		"set session characteristics as transaction read write",
		"SET default_transaction_read_only = off",
		"set session transaction read write",
		"set role admin",
		"set session authorization admin",
		"set search_path = evil, public",
		"start transaction read write",
		"call cleanup()",
		"copy t from '/tmp/x.csv'",
		"vacuum",
		"do $$ begin delete from t; end $$",
		"pragma table_info(t)",
		"select set_config('default_transaction_read_only', 'off', false)",
		`select "set_config"('default_transaction_read_only', 'off', false)`,
		`select U&"\0073et_config"('default_transaction_read_only', 'off', false)`,
		`with p as (select 'C:\' as x), d as (delete from t returning 1) select '' from d`,
		"with a as (select 1 # 2), d as (delete from t returning 1) select * from d",
	}
	for _, s := range allowed {
		if v := sqlcore.ReadOnlyReason(postgres.Dialect{}, s); v != "" {
			t.Errorf("ReadOnlyReason(%q) = %q, want allowed", s, v)
		}
	}
	for _, s := range refused {
		if sqlcore.ReadOnlyReason(postgres.Dialect{}, s) == "" {
			t.Errorf("ReadOnlyReason(%q) allowed it", s)
		}
	}
}
