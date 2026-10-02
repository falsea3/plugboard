package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func ptr[T any](v T) *T { return &v }

func columnNamed(t *testing.T, s *Session, schema, table, name string) model.Column {
	t.Helper()
	cols, err := s.Columns(context.Background(), schema, table)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cols {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no column %s in %v", name, cols)
	return model.Column{}
}

func TestStructureSQLite(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	sc := model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeInsert, Name: ptr("nick"), Type: ptr("TEXT"), DefaultSet: true, Default: ptr("'none'")},
		{Kind: model.ChangeUpdate, Column: "score", Name: ptr("points")},
	}}
	if _, _, err := s.ApplyStructure(ctx, sc); err != nil {
		t.Fatal(err)
	}
	if c := columnNamed(t, s, "main", "users", "nick"); c.Default == nil || *c.Default != "'none'" {
		t.Fatalf("added column = %+v", c)
	}
	columnNamed(t, s, "main", "users", "points")

	// What SQLite can't do in place is refused, not attempted.
	_, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "name", Type: ptr("BLOB")},
	}})
	if err == nil || !strings.Contains(err.Error(), "recreating the table") {
		t.Fatalf("type change: %v", err)
	}

	_, err = s.PreviewStructure(ctx, model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "name", Type: ptr(" ")},
	}})
	if err == nil || !strings.Contains(err.Error(), "empty type") {
		t.Fatalf("empty type: %v", err)
	}
}

func TestStructureServers(t *testing.T) {
	for _, tc := range []struct {
		env    string
		driver model.Driver
		schema string
		create string
	}{
		{"RELAYDB_TEST_PG", model.Postgres, "public", `create table zz_structure (id serial primary key, code varchar(20), note text)`},
		{"RELAYDB_TEST_MYSQL", model.MySQL, "shop", "create table zz_structure (id int not null auto_increment primary key, code varchar(20) collate utf8mb4_bin, " +
			"note text comment 'keep me', touched timestamp null default current_timestamp on update current_timestamp)"},
	} {
		t.Run(string(tc.driver), func(t *testing.T) {
			c := serverConn(t, tc.env, tc.driver)
			ctx := context.Background()
			s, err := Open(ctx, "structure", c, OpenOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			s.Run(ctx, "drop table if exists zz_structure")
			defer s.Run(context.Background(), "drop table if exists zz_structure")
			if _, err := s.Run(ctx, tc.create+"; insert into zz_structure (code) values ('12'), ('34')"); err != nil {
				t.Fatal(err)
			}

			// Read the table first, so pgx has its SELECT cached when the shape changes.
			for _, run := range []func() error{
				func() error {
					_, err := s.TablePage(ctx, model.TableQuery{Schema: tc.schema, Table: "zz_structure"})
					return err
				},
				func() error { _, err := s.Run(ctx, "select * from zz_structure"); return err },
			} {
				if err := run(); err != nil {
					t.Fatal(err)
				}
			}

			sc := model.StructureChange{Schema: tc.schema, Table: "zz_structure", Changes: []model.ColumnChange{
				{Kind: model.ChangeInsert, Name: ptr("status"), Type: ptr("varchar(10)"), Nullable: ptr(false), DefaultSet: true, Default: ptr("'new'")},
				{Kind: model.ChangeUpdate, Column: "code", Name: ptr("number"), Type: ptr("integer"), Nullable: ptr(false)},
				{Kind: model.ChangeUpdate, Column: "note", DefaultSet: true, Default: nil, Nullable: ptr(true)},
			}}
			if _, _, err := s.ApplyStructure(ctx, sc); err != nil {
				t.Fatal(err)
			}
			if c := columnNamed(t, s, tc.schema, "zz_structure", "number"); c.Nullable || !strings.HasPrefix(strings.ToLower(c.Type), "int") {
				t.Fatalf("altered column = %+v", c)
			}
			if c := columnNamed(t, s, tc.schema, "zz_structure", "status"); c.Nullable || c.Default == nil || !strings.Contains(*c.Default, "'new'") {
				t.Fatalf("added column = %+v", c)
			}
			// The cached statements are stale now; reading again must still work.
			want := 4 // id, number, note, status
			if tc.driver == model.MySQL {
				want = 5 // and touched
			}
			page, err := s.TablePage(ctx, model.TableQuery{Schema: tc.schema, Table: "zz_structure"})
			if err != nil || len(page.Result.Columns) != want {
				t.Fatalf("page after the change: %v %v", page.Result.Columns, err)
			}
			if res, err := s.Run(ctx, "select * from zz_structure"); err != nil || len(res[0].Columns) != want {
				t.Fatalf("editor after the change: %v", err)
			}
			res, err := s.Run(ctx, "select number, status from zz_structure order by id")
			if err != nil || res[0].Rows[0][1] != "new" {
				t.Fatalf("rows after the change: %v %v", res, err)
			}

			if tc.driver == model.MySQL {
				// CHANGE COLUMN restates the whole definition: what it leaves out is lost.
				if _, err := s.Run(ctx, "update zz_structure set note = ''"); err != nil {
					t.Fatal(err)
				}
				if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: tc.schema, Table: "zz_structure", Changes: []model.ColumnChange{
					{Kind: model.ChangeUpdate, Column: "note", Nullable: ptr(false)},
					{Kind: model.ChangeUpdate, Column: "touched", Nullable: ptr(false)},
					{Kind: model.ChangeUpdate, Column: "id", Type: ptr("bigint")},
				}}); err != nil {
					t.Fatal(err)
				}
				res, err := s.Run(ctx, "select column_name, column_comment, extra, column_default from information_schema.columns where table_schema = 'shop' and table_name = 'zz_structure' order by ordinal_position")
				if err != nil {
					t.Fatal(err)
				}
				got := map[string][]any{}
				for _, r := range res[0].Rows {
					got[r[0].(string)] = r
				}
				if got["note"][1] != "keep me" {
					t.Errorf("comment lost: %v", got["note"])
				}
				// MySQL says CURRENT_TIMESTAMP, MariaDB current_timestamp().
				extra, def := strings.ToLower(got["touched"][2].(string)), strings.ToLower(fmt.Sprint(got["touched"][3]))
				if !strings.Contains(extra, "on update current_timestamp") || strings.TrimSuffix(def, "()") != "current_timestamp" {
					t.Errorf("on update / default lost: %v", got["touched"])
				}
				if !strings.Contains(got["id"][2].(string), "auto_increment") {
					t.Errorf("auto_increment lost: %v", got["id"])
				}
			}

			// A failing change: PostgreSQL rolls back everything, MySQL keeps what ran.
			applied, partial, err := s.ApplyStructure(ctx, model.StructureChange{Schema: tc.schema, Table: "zz_structure", Changes: []model.ColumnChange{
				{Kind: model.ChangeInsert, Name: ptr("extra"), Type: ptr("int")},
				{Kind: model.ChangeUpdate, Column: "status", Type: ptr("integer")}, // 'new' is no integer
			}})
			var ae *ApplyError
			if !errors.As(err, &ae) || ae.Index != 1 {
				t.Fatalf("err = %v", err)
			}
			_, extraErr := s.Run(ctx, "select extra from zz_structure")
			if tc.driver == model.Postgres && (extraErr == nil || applied != 0 || partial) {
				t.Fatalf("PostgreSQL kept part of a failed change: applied=%d partial=%v", applied, partial)
			}
			if tc.driver == model.MySQL && (extraErr != nil || applied != 1 || !partial) {
				t.Fatalf("MySQL should report the applied first change: applied=%d partial=%v err=%v", applied, partial, extraErr)
			}

			if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: tc.schema, Table: "zz_structure", Changes: []model.ColumnChange{
				{Kind: model.ChangeDelete, Column: "status"},
			}}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStructureRefusedReadOnly(t *testing.T) {
	rw := openSQLite(t)
	s, err := Open(context.Background(), "ro", model.Connection{Driver: model.SQLite, File: rw.Conn.File, ReadOnly: true}, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, _, err = s.ApplyStructure(context.Background(), model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeDelete, Column: "name"},
	}})
	var ro *ReadOnlyError
	if !errors.As(err, &ro) {
		t.Fatalf("err = %v", err)
	}
}

func TestMySQLVersion(t *testing.T) {
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
		d := mysqlDialect{}.forVersion(tc.version).(mysqlDialect)
		if d.canRenameColumn() != tc.rename || d.writesDefaultsAsSQL() != tc.asSQL {
			t.Errorf("%s: rename=%v asSQL=%v (%+v)", tc.version, d.canRenameColumn(), d.writesDefaultsAsSQL(), d)
		}
	}
}

func TestMySQLDefaultSQL(t *testing.T) {
	mysql8 := mysqlDialect{}.forVersion("MySQL 8.4.3").(mysqlDialect)
	mariaDB := mysqlDialect{}.forVersion("MariaDB 10.11.6-MariaDB").(mysqlDialect)
	null := sql.NullString{}
	val := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	for _, tc := range []struct {
		d               mysqlDialect
		raw             sql.NullString
		dataType, extra string
		want            string // "" for no default
	}{
		{mysql8, val("pro"), "varchar", "", "'pro'"},
		{mysql8, val("'lead"), "varchar", "", "'''lead'"},
		{mysql8, val(`back\slash`), "varchar", "", `'back\\slash'`},
		{mysql8, val("1.50"), "decimal", "", "1.50"},
		{mysql8, val("0x6162"), "varbinary", "", "0x6162"},
		{mysql8, val("CURRENT_TIMESTAMP(3)"), "timestamp", "DEFAULT_GENERATED on update CURRENT_TIMESTAMP(3)", "CURRENT_TIMESTAMP(3)"},
		{mysql8, val("CURRENT_TIMESTAMP"), "timestamp", "on update CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP"}, // MySQL 5.7
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

// TestStructureMySQLDefinitions checks that CHANGE COLUMN, which restates a
// column, writes back what information_schema gave in a form the server takes.
func TestStructureMySQLDefinitions(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	ctx := context.Background()
	s, err := Open(ctx, "structure", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Run(ctx, "drop table if exists zz_structure")
	defer s.Run(context.Background(), "drop table if exists zz_structure")
	if _, err := s.Run(ctx, "create table zz_structure (id int primary key, "+
		"expr varchar(40) default (concat('it''s', 'a\\\\b')), bin varbinary(4) default 'ab', "+
		"apos varchar(10) default '''x', code varchar(10) collate utf8mb4_bin default 'q', old int, "+
		"hidden int invisible, seen timestamp null default current_timestamp on update current_timestamp invisible)"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "shop", Table: "zz_structure", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "expr", Nullable: ptr(false)},
		{Kind: model.ChangeUpdate, Column: "bin", Nullable: ptr(false)},
		{Kind: model.ChangeUpdate, Column: "apos", Nullable: ptr(false)},
		{Kind: model.ChangeUpdate, Column: "code", Type: ptr("varchar(20)")},
		{Kind: model.ChangeUpdate, Column: "hidden", Type: ptr("bigint")},
		{Kind: model.ChangeUpdate, Column: "seen", Nullable: ptr(false)},
	}}); err != nil {
		t.Fatal(err)
	}
	res, err := s.Run(ctx, "insert into zz_structure (id) values (1); select expr, hex(bin), apos from zz_structure; "+
		"select collation_name from information_schema.columns where table_schema = 'shop' and table_name = 'zz_structure' and column_name = 'code'")
	if err != nil {
		t.Fatal(err)
	}
	if row := res[1].Rows[0]; row[0] != `it'sa\b` || row[1] != "6162" || row[2] != "'x" {
		t.Errorf("defaults after CHANGE COLUMN: %v", row)
	}
	if got := res[2].Rows[0][0]; got != "utf8mb4_bin" {
		t.Errorf("collation after widening = %v", got)
	}
	res, err = s.Run(ctx, "select column_name, lower(extra) from information_schema.columns where table_schema = 'shop' and table_name = 'zz_structure' and column_name in ('hidden', 'seen') order by column_name")
	if err != nil {
		t.Fatal(err)
	}
	hidden, seen := res[0].Rows[0][1].(string), res[0].Rows[1][1].(string)
	if hidden != "invisible" || !strings.Contains(seen, "on update current_timestamp") || !strings.Contains(seen, "invisible") {
		t.Errorf("extra after CHANGE COLUMN: hidden %q, seen %q", hidden, seen)
	}

	// Without RENAME COLUMN (MySQL 5.7, MariaDB before 10.5.2) a rename restates the column.
	old := "MySQL 5.7.44"
	if s.mariaDB() {
		old = "MariaDB 10.4.0"
	}
	s.Dialect = mysqlDialect{}.forVersion(old)
	sc := model.StructureChange{Schema: "shop", Table: "zz_structure", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "code", Name: ptr("label")},
	}}
	sql, err := s.PreviewStructure(ctx, sc)
	if err != nil || len(sql) != 1 || !strings.Contains(sql[0], "CHANGE COLUMN `code` `label` varchar(20) COLLATE utf8mb4_bin") {
		t.Fatalf("rename on 5.7 = %v %v", sql, err)
	}
	if _, _, err := s.ApplyStructure(ctx, sc); err != nil {
		t.Fatal(err)
	}
	if c := columnNamed(t, s, "shop", "zz_structure", "label"); c.Default == nil || *c.Default != "'q'" {
		t.Errorf("default after the restating rename = %v", show(c.Default))
	}
}

func TestStructurePostgresTypeAndDefault(t *testing.T) {
	c := serverConn(t, "RELAYDB_TEST_PG", model.Postgres)
	ctx := context.Background()
	s, err := Open(ctx, "structure", c, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Run(ctx, "drop table if exists zz_structure")
	defer s.Run(context.Background(), "drop table if exists zz_structure")
	if _, err := s.Run(ctx, "create table zz_structure (id int primary key, flag text default 'no', kind text default 'a')"); err != nil {
		t.Fatal(err)
	}
	// 'no' can't be cast to boolean: the old default has to go before the type changes.
	if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "public", Table: "zz_structure", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "flag", Type: ptr("boolean"), DefaultSet: true, Default: ptr("false")},
		{Kind: model.ChangeUpdate, Column: "kind", Type: ptr("varchar(5)"), DefaultSet: true, Default: nil},
	}}); err != nil {
		t.Fatal(err)
	}
	if c := columnNamed(t, s, "public", "zz_structure", "flag"); c.Type != "boolean" || c.Default == nil || *c.Default != "false" {
		t.Errorf("flag = %+v", c)
	}
	if c := columnNamed(t, s, "public", "zz_structure", "kind"); c.Default != nil {
		t.Errorf("kind kept its default: %+v", c)
	}
}

// TestStructureTableBusy: a transaction left open in the SQL editor holds a
// lock the ALTER TABLE needs. The change gives up after ddlLockWait instead of
// queueing every other query on the table behind it.
func TestStructureTableBusy(t *testing.T) {
	for _, tc := range []struct {
		env    string
		driver model.Driver
		schema string
	}{
		{"RELAYDB_TEST_PG", model.Postgres, "public"},
		{"RELAYDB_TEST_MYSQL", model.MySQL, "shop"},
	} {
		t.Run(string(tc.driver), func(t *testing.T) {
			c := serverConn(t, tc.env, tc.driver)
			ctx := context.Background()
			s, err := Open(ctx, "structure", c, OpenOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			s.Run(ctx, "drop table if exists zz_structure")
			defer s.Run(context.Background(), "drop table if exists zz_structure")
			if _, err := s.Run(ctx, "create table zz_structure (id int primary key)"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Run(ctx, "begin; select * from zz_structure"); err != nil {
				t.Fatal(err)
			}
			add := model.StructureChange{Schema: tc.schema, Table: "zz_structure", Changes: []model.ColumnChange{
				{Kind: model.ChangeInsert, Name: ptr("more"), Type: ptr("int")},
			}}
			started := time.Now()
			_, _, err = s.ApplyStructure(ctx, add)
			if !errors.Is(err, ErrTableBusy) || time.Since(started) > 3*ddlLockWait*time.Second {
				t.Fatalf("err = %v after %v", err, time.Since(started))
			}
			if _, err := s.Run(ctx, "rollback"); err != nil {
				t.Fatal(err)
			}
			if _, _, err := s.ApplyStructure(ctx, add); err != nil {
				t.Fatalf("once the transaction is over: %v", err)
			}
			if tc.driver == model.MySQL {
				// The lock wait doesn't stay on the pooled connections: hold
				// every one the editor doesn't, so each is looked at.
				for i := 0; i < 3; i++ {
					conn, err := s.DB.Conn(ctx)
					if err != nil {
						t.Fatal(err)
					}
					defer conn.Close()
					var wait int
					if err := conn.QueryRowContext(ctx, "SELECT @@session.lock_wait_timeout").Scan(&wait); err != nil || wait == ddlLockWait {
						t.Fatalf("pooled connection lock_wait_timeout = %d %v", wait, err)
					}
				}
			}
		})
	}
}
