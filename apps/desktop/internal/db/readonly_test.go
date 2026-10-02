package db

import (
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
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
		"show tables",
		"describe t",
		"pragma table_info(t)",
		"pragma foreign_key_list(t)",
		"begin",
		"start transaction read only",
		"commit",
		"set statement_timeout = 5000",
		"select count(*) into @n from t",
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
		"select * from t for update", // takes row locks
		"pragma foreign_keys = off",
		"pragma journal_mode(wal)",
		"pragma wal_checkpoint",
		"set transaction read write",
		"set session characteristics as transaction read write",
		"SET default_transaction_read_only = off",
		"set session transaction read write",
		"set role admin",
		"set session authorization admin",
		"start transaction read write",
		"call cleanup()",
		"copy t from '/tmp/x.csv'",
		"vacuum",
		"do $$ begin delete from t; end $$",
		// set_config() switches the server-side guard off from a plain SELECT.
		"select set_config('default_transaction_read_only', 'off', false)",
		`select "set_config"('default_transaction_read_only', 'off', false)`,
		`select U&"\0073et_config"('default_transaction_read_only', 'off', false)`,
		// In PostgreSQL a backslash ends nothing and # is an operator, so the
		// DELETE is real SQL, not part of a string or a comment.
		`with p as (select 'C:\' as x), d as (delete from t returning 1) select '' from d`,
		"with a as (select 1 # 2), d as (delete from t returning 1) select * from d",
	}
	for _, s := range allowed {
		if v := ReadOnlyReason(s, model.Postgres); v != "" {
			t.Errorf("ReadOnlyReason(%q) = %q, want allowed", s, v)
		}
	}
	for _, s := range refused {
		if ReadOnlyReason(s, model.Postgres) == "" {
			t.Errorf("ReadOnlyReason(%q) allowed it", s)
		}
	}
}

func TestReadOnlyReasonMySQL(t *testing.T) {
	allowed := []string{
		"select 'it\\'s; delete from t' from t",
		"select 1 # delete from t",
		"select 1 -- delete from t",
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
		"set @@session.transaction_read_only = 0",
		"select * from t /*! into outfile '/tmp/x' */",
		"select 1--1 into outfile '/tmp/x'", // --1 is minus minus one, not a comment
	}
	for _, s := range allowed {
		if v := ReadOnlyReason(s, model.MySQL); v != "" {
			t.Errorf("ReadOnlyReason(%q) = %q, want allowed", s, v)
		}
	}
	for _, s := range refused {
		if ReadOnlyReason(s, model.MySQL) == "" {
			t.Errorf("ReadOnlyReason(%q) allowed it", s)
		}
	}
}
