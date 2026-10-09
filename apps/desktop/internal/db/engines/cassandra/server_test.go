package cassandra_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/cassandra"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func conn(t *testing.T) model.Connection {
	t.Helper()
	addr := os.Getenv("PLUGBOARD_TEST_CASSANDRA")
	if addr == "" {
		t.Skip("PLUGBOARD_TEST_CASSANDRA not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	return model.Connection{Driver: model.Cassandra, Host: addr[:i], Port: port, User: "relay", Password: "relay", Database: "shop"}
}

func open(t *testing.T, c model.Connection) *cassandra.Session {
	t.Helper()
	s, err := cassandra.Open(context.Background(), "t", c, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func run(t *testing.T, s *cassandra.Session, script string) []model.ResultSet {
	t.Helper()
	res, err := s.Run(context.Background(), script)
	if err != nil {
		t.Fatalf("%s: %v", script, err)
	}
	return res
}

func TestCatalog(t *testing.T) {
	s := open(t, conn(t))
	ctx := context.Background()
	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(info.ServerVersion, "Cassandra ") || info.DefaultSchema != "shop" || info.Schemas[0] != "shop" {
		t.Fatalf("info = %+v", info)
	}
	tables, err := s.Tables(ctx, "shop")
	if err != nil || !slices.ContainsFunc(tables, func(t model.TableInfo) bool { return t.Name == "orders_by_customer" && t.Kind == "table" }) {
		t.Fatalf("tables = %+v, %v", tables, err)
	}
	cols, err := s.Columns(ctx, "shop", "orders_by_customer")
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	if !slices.Equal(names[:3], []string{"customer_id", "created_at", "order_id"}) || !cols[0].PrimaryKey || cols[3].PrimaryKey || cols[0].Kind != model.KindNumber {
		t.Fatalf("columns = %+v", cols)
	}
	idx, err := s.Indexes(ctx, "shop", "customers")
	if err != nil || len(idx) != 2 || !idx[0].Primary || idx[1].Name != "customers_country" {
		t.Fatalf("indexes = %+v, %v", idx, err)
	}
	objs, err := s.Objects(ctx, "shop")
	if err != nil || !slices.ContainsFunc(objs, func(o model.DBObject) bool { return o.Name == "address" && o.Kind == "domain" }) {
		t.Fatalf("objects = %+v, %v", objs, err)
	}
	ddl, err := s.DDL(ctx, model.DBObject{Schema: "shop", Name: "orders_by_customer", Kind: "table"})
	if err != nil || !strings.Contains(ddl, "CLUSTERING ORDER BY") {
		t.Fatalf("ddl = %q, %v", ddl, err)
	}
}

func TestPagesAndFilters(t *testing.T) {
	s := open(t, conn(t))
	ctx := context.Background()
	run(t, s, "CREATE TABLE IF NOT EXISTS shop.many (p int, c int, v text, PRIMARY KEY (p, c)); TRUNCATE shop.many")
	var b strings.Builder
	b.WriteString("BEGIN UNLOGGED BATCH\n")
	for p := range 5 {
		for c := range 130 {
			b.WriteString("INSERT INTO shop.many (p, c, v) VALUES (" + strconv.Itoa(p) + ", " + strconv.Itoa(c) + ", 'x');\n")
		}
	}
	b.WriteString("APPLY BATCH")
	run(t, s, b.String())
	q := model.TableQuery{Schema: "shop", Table: "many", Limit: 200}
	seen := 0
	for range 10 {
		page, err := s.TablePage(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		seen += len(page.Result.Rows)
		if !page.HasMore {
			break
		}
		last := page.Result.Rows[len(page.Result.Rows)-1]
		q.After = []any{last[0], last[1]}
	}
	if seen != 650 {
		t.Fatalf("paged through %d rows, want 650", seen)
	}
	if _, err := s.TablePage(ctx, model.TableQuery{Schema: "shop", Table: "many", After: []any{99, 99}}); err == nil {
		t.Error("an unknown cursor was accepted")
	}
	page, err := s.TablePage(ctx, model.TableQuery{Schema: "shop", Table: "customers", Filters: []model.Filter{{Column: "plan", Op: "=", Value: "pro"}}})
	if err != nil || len(page.Result.Rows) != 2 {
		t.Fatalf("plan = pro: %v, %v", page.Result.Rows, err)
	}
	if _, err := s.TablePage(ctx, model.TableQuery{Schema: "shop", Table: "customers", Filters: []model.Filter{{Column: "name", Op: "contains", Value: "Ann"}}}); err == nil {
		t.Error("text contains should be refused")
	}
	if _, err := s.TablePage(ctx, model.TableQuery{Schema: "shop", Table: "customers", OrderBy: "name"}); err == nil {
		t.Error("sorting should be refused")
	}
	n, err := s.CountRows(ctx, model.TableQuery{Schema: "shop", Table: "many"}, true)
	if err != nil || n.Count != 650 {
		t.Fatalf("count = %+v, %v", n, err)
	}
}

func TestEditorAndEdits(t *testing.T) {
	s := open(t, conn(t))
	ctx := context.Background()
	res := run(t, s, "USE shop; SELECT id, tags, address, created_at FROM customers WHERE id = 1")
	row := res[1].Rows[0]
	if row[1] != `["beta","vip"]` || !strings.Contains(row[2].(string), `"city":"Austin"`) || row[3] != "2026-01-03 10:00:00.000+0000" {
		t.Fatalf("row = %#v", row)
	}
	if _, err := s.Run(ctx, "SELECT * FROM customers;\nSELEC 1"); err == nil {
		t.Fatal("syntax error passed")
	} else {
		var se *db.StatementError
		if !errors.As(err, &se) || se.Index != 1 {
			t.Fatalf("syntax error = %v", err)
		}
	}
	cs := model.ChangeSet{Schema: "shop", Table: "customers", Changes: []model.RowChange{
		{Kind: model.ChangeInsert, Values: map[string]any{"id": "100", "name": "New", "tags": `["a"]`, "created_at": "2026-02-01 00:00:00.000+0000"}},
		{Kind: model.ChangeUpdate, Key: map[string]any{"id": float64(100)}, Values: map[string]any{"name": "Renamed", "is_active": "true", "country": nil}},
	}}
	if preview, err := s.PreviewChanges(ctx, cs); err != nil || !strings.Contains(preview[0], "IF NOT EXISTS") {
		t.Fatalf("preview = %v, %v", preview, err)
	}
	if n, err := s.ApplyChanges(ctx, cs); err != nil || n != 2 {
		t.Fatalf("apply = %d, %v", n, err)
	}
	got := run(t, s, "SELECT name, is_active FROM shop.customers WHERE id = 100")[0].Rows[0]
	if got[0] != "Renamed" || got[1] != true {
		t.Fatalf("after edit = %v", got)
	}
	if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: "shop", Table: "customers", Changes: cs.Changes[:1]}); err == nil {
		t.Error("inserting an existing key passed")
	}
	gone := model.ChangeSet{Schema: "shop", Table: "customers", Changes: []model.RowChange{{Kind: model.ChangeUpdate, Key: map[string]any{"id": "999"}, Values: map[string]any{"name": "x"}}}}
	if _, err := s.ApplyChanges(ctx, gone); !errors.Is(err, db.ErrRowGone) {
		t.Errorf("update of a missing row = %v", err)
	}
	if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: "shop", Table: "customers", Changes: []model.RowChange{{Kind: model.ChangeUpdate, Key: map[string]any{"id": "100"}, Values: map[string]any{"id": "101"}}}}); err == nil {
		t.Error("changing the primary key passed")
	}
	if n, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: "shop", Table: "customers", Changes: []model.RowChange{{Kind: model.ChangeDelete, Key: map[string]any{"id": "100"}}}}); err != nil || n != 1 {
		t.Fatalf("delete = %d, %v", n, err)
	}
}

func TestReadOnly(t *testing.T) {
	c := conn(t)
	c.ReadOnly = true
	s := open(t, c)
	ctx := context.Background()
	run(t, s, "SELECT * FROM shop.customers LIMIT 1; DESCRIBE KEYSPACES; USE shop")
	var ro *db.ReadOnlyError
	for _, w := range []string{"INSERT INTO shop.customers (id) VALUES (7)", "TRUNCATE shop.customers", "DROP TABLE shop.many"} {
		if _, err := s.Run(ctx, w); !errors.As(err, &ro) {
			t.Errorf("%s = %v", w, err)
		}
	}
	if _, err := s.ApplyChanges(ctx, model.ChangeSet{Schema: "shop", Table: "customers"}); !errors.As(err, &ro) {
		t.Errorf("edits = %v", err)
	}
	if got := s.WriteStatements("SELECT 1 FROM t; UPDATE t SET a = 1 WHERE k = 1"); len(got) != 1 {
		t.Errorf("writes = %q", got)
	}
}

func TestWrongPassword(t *testing.T) {
	c := conn(t)
	c.Password = "nope"
	if _, err := cassandra.Open(context.Background(), "x", c, db.OpenOptions{}); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("wrong password = %v", err)
	}
}

func TestStructureStatementsRun(t *testing.T) {
	s := open(t, conn(t))
	ctx := context.Background()
	run(t, s, "CREATE TABLE IF NOT EXISTS shop.scratch (p int, c int, v text, PRIMARY KEY (p, c)); INSERT INTO shop.scratch (p, c, v) VALUES (1, 1, 'x')")
	sc := func(ch model.ColumnChange) model.StructureChange {
		return model.StructureChange{Schema: "shop", Table: "scratch", Changes: []model.ColumnChange{ch}}
	}
	name, typ := "cc", "text"
	if _, err := s.PreviewStructure(ctx, sc(model.ColumnChange{Kind: model.ChangeUpdate, Column: "v", Type: &typ})); err != nil {
		t.Errorf("same type = %v", err)
	}
	typ = "int"
	if _, err := s.PreviewStructure(ctx, sc(model.ColumnChange{Kind: model.ChangeUpdate, Column: "v", Type: &typ})); err == nil {
		t.Error("a type change passed")
	}
	if _, err := s.PreviewStructure(ctx, sc(model.ColumnChange{Kind: model.ChangeUpdate, Column: "v", Name: &name})); err == nil {
		t.Error("renaming a regular column passed")
	}
	for _, ch := range []model.ColumnChange{{Kind: model.ChangeUpdate, Column: "c", Name: &name}, {Kind: model.ChangeDelete, Column: "v"}} {
		stmts, err := s.PreviewStructure(ctx, sc(ch))
		if err != nil {
			t.Fatal(err)
		}
		run(t, s, stmts[0])
	}
	run(t, s, s.TruncateSQL("shop", "scratch"))
	if n := run(t, s, "SELECT COUNT(*) FROM shop.scratch")[0].Rows[0][0]; n != int64(0) {
		t.Errorf("after truncate = %v", n)
	}
	run(t, s, s.CreateIndexSQL("shop", "scratch", model.NewIndex{Name: "scratch_cc", Columns: []string{"cc"}}))
	run(t, s, s.DropIndexSQL("shop", "scratch", "scratch_cc"))
	run(t, s, "DROP TABLE shop.scratch")
}
