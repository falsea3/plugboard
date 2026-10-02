package mysql_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/dbtest"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/mysql"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlcore"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func TestServer(t *testing.T) {
	dbtest.Run(t, dbtest.Server{
		Env:       "RELAYDB_TEST_MYSQL",
		Dialect:   mysql.Dialect{},
		Schema:    "shop",
		True:      "1",
		KeyColumn: "id int not null auto_increment primary key",
	})
}

func mariaDB(s *sqlcore.Session) bool { return strings.HasPrefix(s.Version, "MariaDB") }

func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestCatalogAndValues(t *testing.T) {
	c := dbtest.Conn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	ctx := context.Background()
	s := dbtest.Open(t, mysql.Dialect{}, c)

	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.DefaultSchema != "shop" {
		t.Fatalf("info = %+v", info)
	}
	tables, err := s.Dialect.ListTables(ctx, s.DB, "shop")
	if err != nil {
		t.Fatal(err)
	}
	if !dbtest.HasTable(tables, "orders", "table") || !dbtest.HasTable(tables, "order_summary", "view") {
		t.Fatalf("tables = %+v", tables)
	}
	cols, err := s.Dialect.ListColumns(ctx, s.DB, "shop", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if !cols[0].PrimaryKey || cols[2].Type != "enum('new','paid','shipped','cancelled')" {
		t.Fatalf("cols = %+v", cols)
	}

	res, err := s.Run(ctx, `select 'a\';b' as s; select cast(12.5 as decimal(10,2)) as d, null as n, cast('2026-01-02 03:04:05' as datetime) as dt, x'dead' as bin # trailing ; comment
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0].Rows[0][0] != "a';b" {
		t.Fatalf("res = %+v", res)
	}
	row := res[1].Rows[0]
	if row[0] != "12.50" || row[1] != nil || row[2] != "2026-01-02 03:04:05" {
		t.Fatalf("row = %#v", row)
	}

	page, err := s.TablePage(ctx, model.TableQuery{Schema: "shop", Table: "customers", Limit: 10, OrderBy: "id", OrderDesc: true})
	if err != nil || len(page.Result.Rows) != 10 || !page.HasMore {
		t.Fatalf("page = %+v, err = %v", page, err)
	}

	_, err = s.Run(ctx, "select * from nope")
	var se *db.StatementError
	if !errors.As(err, &se) {
		t.Fatalf("err = %v", err)
	}
}

func TestEditorPaging(t *testing.T) {
	c := dbtest.Conn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	s := dbtest.Open(t, mysql.Dialect{}, c)
	res, err := s.Run(context.Background(), "select 1 as a, 2 as a; select * from orders order by id desc")
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Pageable || len(res[0].Columns) != 2 {
		t.Fatalf("duplicate names: pageable=%v cols=%v", res[0].Pageable, res[0].Columns)
	}
	if res[1].Pageable == mariaDB(s) || res[1].Rows[0][0] != int64(5000) {
		t.Fatalf("order by survived the wrap? pageable=%v first id = %v (%s)", res[1].Pageable, res[1].Rows[0][0], s.Version)
	}
}

func TestEnumsAndExpressions(t *testing.T) {
	c := dbtest.Conn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	ctx := context.Background()
	s := dbtest.Open(t, mysql.Dialect{}, c)
	cols, err := s.Columns(ctx, "shop", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if got := cols[2].Enum; len(got) != 4 || got[0] != "new" {
		t.Fatalf("enum = %v", got)
	}
	_, err = s.ApplyChanges(ctx, model.ChangeSet{Schema: "shop", Table: "orders", Changes: []model.RowChange{
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(1)}, Values: map[string]any{
			"placed_at": map[string]any{"$expr": "now"}, "status": map[string]any{"$expr": "default"},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	res, _ := s.Run(ctx, "select status, placed_at > now() - interval 1 minute from orders where id = 1")
	if row := res[0].Rows[0]; row[0] != "new" || row[1] != int64(1) {
		t.Fatalf("row = %v", row)
	}
}

func TestRestatedColumns(t *testing.T) {
	c := dbtest.Conn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	s := dbtest.Open(t, mysql.Dialect{}, c)
	ctx := context.Background()
	s.Run(ctx, "drop table if exists zz_restated")
	t.Cleanup(func() { s.Run(context.Background(), "drop table if exists zz_restated") })
	if _, err := s.Run(ctx, "create table zz_restated (id int not null auto_increment primary key, note text comment 'keep me', "+
		"touched timestamp null default current_timestamp on update current_timestamp); insert into zz_restated (note) values ('')"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "shop", Table: "zz_restated", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "note", Nullable: dbtest.Ptr(false)},
		{Kind: model.ChangeUpdate, Column: "touched", Nullable: dbtest.Ptr(false)},
		{Kind: model.ChangeUpdate, Column: "id", Type: dbtest.Ptr("bigint")},
	}}); err != nil {
		t.Fatal(err)
	}
	res, err := s.Run(ctx, "select column_name, column_comment, extra, column_default from information_schema.columns where table_schema = 'shop' and table_name = 'zz_restated' order by ordinal_position")
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
	extra, def := strings.ToLower(got["touched"][2].(string)), strings.ToLower(fmt.Sprint(got["touched"][3]))
	if !strings.Contains(extra, "on update current_timestamp") || strings.TrimSuffix(def, "()") != "current_timestamp" {
		t.Errorf("on update / default lost: %v", got["touched"])
	}
	if !strings.Contains(got["id"][2].(string), "auto_increment") {
		t.Errorf("auto_increment lost: %v", got["id"])
	}
}

func TestStructureDefinitions(t *testing.T) {
	c := dbtest.Conn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	ctx := context.Background()
	s := dbtest.Open(t, mysql.Dialect{}, c)
	s.Run(ctx, "drop table if exists zz_structure")
	defer s.Run(context.Background(), "drop table if exists zz_structure")
	if _, err := s.Run(ctx, "create table zz_structure (id int primary key, "+
		"expr varchar(40) default (concat('it''s', 'a\\\\b')), bin varbinary(4) default 'ab', "+
		"apos varchar(10) default '''x', code varchar(10) collate utf8mb4_bin default 'q', old int, "+
		"hidden int invisible, seen timestamp null default current_timestamp on update current_timestamp invisible)"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "shop", Table: "zz_structure", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "expr", Nullable: dbtest.Ptr(false)},
		{Kind: model.ChangeUpdate, Column: "bin", Nullable: dbtest.Ptr(false)},
		{Kind: model.ChangeUpdate, Column: "apos", Nullable: dbtest.Ptr(false)},
		{Kind: model.ChangeUpdate, Column: "code", Type: dbtest.Ptr("varchar(20)")},
		{Kind: model.ChangeUpdate, Column: "hidden", Type: dbtest.Ptr("bigint")},
		{Kind: model.ChangeUpdate, Column: "seen", Nullable: dbtest.Ptr(false)},
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

	old := "MySQL 5.7.44"
	if mariaDB(s) {
		old = "MariaDB 10.4.0"
	}
	s.Dialect = mysql.Dialect{}.ForVersion(old)
	sc := model.StructureChange{Schema: "shop", Table: "zz_structure", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "code", Name: dbtest.Ptr("label")},
	}}
	sql, err := s.PreviewStructure(ctx, sc)
	if err != nil || len(sql) != 1 || !strings.Contains(sql[0], "CHANGE COLUMN `code` `label` varchar(20) COLLATE utf8mb4_bin") {
		t.Fatalf("rename on 5.7 = %v %v", sql, err)
	}
	if _, _, err := s.ApplyStructure(ctx, sc); err != nil {
		t.Fatal(err)
	}
	if c := dbtest.Column(t, s, "shop", "zz_structure", "label"); c.Default == nil || *c.Default != "'q'" {
		t.Errorf("default after the restating rename = %v", show(c.Default))
	}
}

func TestLockWaitIsReset(t *testing.T) {
	c := dbtest.Conn(t, "RELAYDB_TEST_MYSQL", model.MySQL)
	s := dbtest.Open(t, mysql.Dialect{}, c)
	ctx := context.Background()
	s.Run(ctx, "drop table if exists zz_wait")
	t.Cleanup(func() { s.Run(context.Background(), "drop table if exists zz_wait") })
	if _, err := s.Run(ctx, "create table zz_wait (id int primary key); begin; select * from zz_wait"); err != nil {
		t.Fatal(err)
	}
	add := model.StructureChange{Schema: "shop", Table: "zz_wait", Changes: []model.ColumnChange{
		{Kind: model.ChangeInsert, Name: dbtest.Ptr("more"), Type: dbtest.Ptr("int")},
	}}
	if _, _, err := s.ApplyStructure(ctx, add); !errors.Is(err, db.ErrTableBusy) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.Run(ctx, "rollback"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ApplyStructure(ctx, add); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		conn, err := s.DB.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		var wait int
		if err := conn.QueryRowContext(ctx, "SELECT @@session.lock_wait_timeout").Scan(&wait); err != nil || wait <= 60 {
			t.Fatalf("pooled connection lock_wait_timeout = %d %v", wait, err)
		}
	}
}
