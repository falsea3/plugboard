package clickhouse_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/clickhouse"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func open(t *testing.T, readOnly bool) *sqlcore.Session {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_CLICKHOUSE", model.ClickHouse)
	c.ReadOnly = readOnly
	return sqltest.Open(t, clickhouse.Dialect{}, c)
}

func TestCatalog(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(info.ServerVersion, "ClickHouse ") || info.DefaultSchema != "shop" || !slices.Contains(info.Schemas, "shop") || info.Engine.Editable {
		t.Fatalf("info = %+v", info)
	}
	if info.Schemas[len(info.Schemas)-1] != "system" {
		t.Errorf("system should come last: %v", info.Schemas)
	}
	tables, err := s.Tables(ctx, "shop")
	if err != nil {
		t.Fatal(err)
	}
	if !sqltest.HasTable(tables, "customers", "table") || !sqltest.HasTable(tables, "revenue_by_country", "view") {
		t.Fatalf("tables = %+v", tables)
	}
	col := func(name string) model.Column { return sqltest.Column(t, s, "shop", "customers", name) }
	if c := col("country"); !c.Nullable || c.Kind != model.KindText {
		t.Errorf("country = %+v", c)
	}
	if c := col("plan"); !slices.Equal(c.Enum, []string{"free", "team", "pro"}) || c.Default == nil || *c.Default != "'free'" {
		t.Errorf("plan = %+v", c)
	}
	if c := col("is_active"); c.Kind != model.KindBool || c.PrimaryKey {
		t.Errorf("is_active = %+v", c)
	}
	if c := col("created_at"); c.Kind != model.KindDateTime {
		t.Errorf("created_at = %+v", c)
	}
	idx, err := s.Indexes(ctx, "shop", "orders")
	if err != nil || len(idx) != 2 || !idx[0].Primary || !slices.Equal(idx[0].Columns, []string{"customer_id", "created_at"}) || idx[1].Name != "status_idx" {
		t.Fatalf("indexes = %+v, %v", idx, err)
	}
	objs, err := s.Objects(ctx, "shop")
	if err != nil || !slices.ContainsFunc(objs, func(o model.DBObject) bool { return o.Name == "with_tax" && o.Kind == "function" }) {
		t.Fatalf("objects = %+v, %v", objs, err)
	}
	ddl, err := s.DDL(ctx, model.DBObject{Schema: "shop", Name: "orders", Kind: "table"})
	if err != nil || !strings.Contains(ddl, "ENGINE = MergeTree") {
		t.Fatalf("ddl = %q, %v", ddl, err)
	}
	if n, err := s.CountRows(ctx, model.TableQuery{Schema: "shop", Table: "orders"}, false); err != nil || n.Count != 20000 {
		t.Fatalf("estimate = %+v, %v", n, err)
	}
}

func TestTablePagesAndFilters(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	page := func(q model.TableQuery) model.TablePage {
		t.Helper()
		q.Schema, q.Table = "shop", "customers"
		if q.Limit == 0 {
			q.Limit = 5000
		}
		p, err := s.TablePage(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	if p := page(model.TableQuery{Limit: 300}); len(p.Result.Rows) != 300 || !p.HasMore || !slices.Equal(p.DefaultOrder, []string{"id"}) || p.Result.Rows[1][0] != uint64(2) {
		t.Fatalf("first page = %d rows, more %v, order %v, second id %#v", len(p.Result.Rows), p.HasMore, p.DefaultOrder, p.Result.Rows[1][0])
	}
	if p := page(model.TableQuery{Offset: 2400, Limit: 300, OrderBy: "id"}); len(p.Result.Rows) != 100 || p.HasMore {
		t.Fatalf("last page = %d rows", len(p.Result.Rows))
	}
	f := func(fs ...model.Filter) int { return len(page(model.TableQuery{Filters: fs}).Result.Rows) }
	if n := f(model.Filter{Column: "id", Op: "=", Value: "7"}); n != 1 {
		t.Errorf("id = 7: %d", n)
	}
	if n := f(model.Filter{Column: "plan", Op: "=", Value: "pro"}); n == 0 {
		t.Error("enum filter matched nothing")
	}
	if n := f(model.Filter{Column: "email", Op: "contains", Value: "CUSTOMER12@"}); n != 1 {
		t.Errorf("contains: %d", n)
	}
	if n := f(model.Filter{Column: "country", Op: "null"}); n == 0 {
		t.Error("is null matched nothing")
	}
	if n := f(model.Filter{Column: "created_at", Op: "<", Value: "2000-01-01"}); n != 0 {
		t.Errorf("date compare: %d", n)
	}
}

func TestEditorRunsAndPages(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	res, err := s.Run(ctx, "SELECT number FROM numbers(3000); CREATE TABLE IF NOT EXISTS shop.scratch (a UInt8) ENGINE = Memory; INSERT INTO shop.scratch VALUES (1), (2); SELECT count() FROM shop.scratch; DROP TABLE shop.scratch")
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].HasMore || len(res[0].Rows) != sqlcore.EditorChunk {
		t.Fatalf("paged read = %d rows, more %v", len(res[0].Rows), res[0].HasMore)
	}
	if res[3].Rows[0][0] != "2" && res[3].Rows[0][0] != int64(2) && res[3].Rows[0][0] != uint64(2) {
		t.Errorf("count = %#v", res[3].Rows[0][0])
	}
	more, err := s.RunMore(ctx, res[0].Statement, sqlcore.EditorChunk)
	if err != nil || len(more.Rows) != sqlcore.EditorChunk {
		t.Fatalf("second chunk = %d rows, %v", len(more.Rows), err)
	}
	problems, err := s.CheckSyntax(ctx, "SELECT 1; SELEC 2; INSERT INTO shop.customers (id) VALUES (999999)")
	if err != nil || len(problems) != 1 || problems[0].Index != 1 || problems[0].Position != 6 || !strings.HasPrefix(problems[0].Message, "Syntax error near “") || !strings.Contains(problems[0].Message, "2") {
		t.Fatalf("problems = %+v, %v", problems, err)
	}
	if n, _ := s.Run(ctx, "SELECT count() FROM shop.customers WHERE id = 999999"); n[0].Rows[0][0] != uint64(0) && n[0].Rows[0][0] != "0" {
		t.Errorf("the syntax check inserted a row: %v", n[0].Rows)
	}
}

func TestReadOnlyRefusesTwice(t *testing.T) {
	s := open(t, true)
	ctx := context.Background()
	if _, err := s.Run(ctx, "SELECT count() FROM customers; SHOW TABLES; DESCRIBE customers; EXISTS customers"); err != nil {
		t.Fatalf("reads refused: %v", err)
	}
	var ro *db.ReadOnlyError
	for _, w := range []string{"INSERT INTO customers (id) VALUES (1)", "SET readonly = 0", "ALTER TABLE customers DELETE WHERE 1", "OPTIMIZE TABLE customers", "SELECT 1 INTO OUTFILE 'x'"} {
		if _, err := s.Run(ctx, w); !errors.As(err, &ro) {
			t.Errorf("%s = %v", w, err)
		}
	}
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO customers (id) VALUES (1)"); err == nil || !strings.Contains(err.Error(), "readonly") {
		t.Errorf("the server took a write: %v", err)
	}
	if _, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "shop", Table: "customers"}); err == nil {
		t.Error("structure change in read-only")
	}
}

func TestEditsAreRefused(t *testing.T) {
	s := open(t, false)
	if _, err := s.ApplyChanges(context.Background(), model.ChangeSet{Schema: "shop", Table: "customers"}); !errors.Is(err, db.ErrNotSupported) {
		t.Fatalf("apply changes = %v", err)
	}
}

func TestThroughSSH(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_SSH")
	if addr == "" || os.Getenv("PLUGBOARD_TEST_CLICKHOUSE") == "" {
		t.Skip("PLUGBOARD_TEST_SSH and PLUGBOARD_TEST_CLICKHOUSE not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{
		Driver: model.ClickHouse, Host: "clickhouse", Port: 9000, User: "relay", Password: "relay", Database: "shop",
		SSH: model.SSHTunnel{Enabled: true, Host: addr[:i], Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	s, err := sqlcore.Open(context.Background(), "ssh", c, clickhouse.Dialect{}, db.OpenOptions{KnownHostsFile: t.TempDir() + "/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := s.Run(context.Background(), "SELECT count() FROM customers")
	if err != nil || len(res[0].Rows) != 1 {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}
