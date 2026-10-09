package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	_ "modernc.org/sqlite"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type fakeStore []model.Connection

func (f fakeStore) List() ([]model.Connection, error) { return f, nil }

func (f fakeStore) Get(id string) (model.Connection, error) {
	for _, c := range f {
		if c.ID == id {
			return c, nil
		}
	}
	return model.Connection{}, fmt.Errorf("no %s", id)
}

func sample(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shop.db")
	d, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	_, err = d.Exec(`CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT, bio TEXT);
		INSERT INTO customers VALUES (1, 'Ann', 'short'), (2, 'Bob', '` + strings.Repeat("x", 1500) + `'), (3, 'Chen', NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func client(t *testing.T, conns fakeStore, opts Options) (*sdk.ClientSession, string) {
	t.Helper()
	dir := t.TempDir()
	a := NewAgent(conns, dir, opts)
	t.Cleanup(func() { a.Close() })
	st, ct := sdk.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := NewServer(a, "test").Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs, dir
}

func call(t *testing.T, cs *sdk.ClientSession, tool string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String(), res.IsError
}

func conns(path string) fakeStore {
	return fakeStore{
		{ID: "a", Name: "Shop", Driver: model.SQLite, File: path, Env: "dev", AIAccess: true},
		{ID: "b", Name: "Prod shop", Driver: model.SQLite, File: path, Env: "prod", AIAccess: true},
		{ID: "c", Name: "Private", Driver: model.SQLite, File: path, Env: "dev"},
	}
}

func TestReadsWhatTheUserOpened(t *testing.T) {
	cs, dir := client(t, conns(sample(t)), Options{})
	out, _ := call(t, cs, "list_connections", nil)
	if !strings.Contains(out, `"Shop"`) || !strings.Contains(out, `"Prod shop"`) || strings.Contains(out, "Private") || strings.Contains(out, "read-write") {
		t.Fatalf("list_connections = %s", out)
	}
	if out, isErr := call(t, cs, "list_tables", map[string]any{"connection": "Private"}); !isErr || !strings.Contains(out, "AI agents") {
		t.Fatalf("a connection that isn't opened = %s", out)
	}
	out, _ = call(t, cs, "list_tables", map[string]any{"connection": "shop"})
	if !strings.Contains(out, `"customers"`) {
		t.Fatalf("list_tables = %s", out)
	}
	out, _ = call(t, cs, "describe_table", map[string]any{"connection": "Shop", "table": "customers"})
	if !strings.Contains(out, `"primaryKey": true`) || !strings.Contains(out, "CREATE TABLE") {
		t.Fatalf("describe_table = %s", out)
	}
	out, _ = call(t, cs, "run_query", map[string]any{"connection": "Shop", "sql": "SELECT id, bio FROM customers ORDER BY id", "max_rows": 2})
	var got struct {
		Results []struct {
			Columns  []string `json:"columns"`
			Rows     [][]any  `json:"rows"`
			MoreRows bool     `json:"more_rows"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got.Results[0].Rows) != 2 || !got.Results[0].MoreRows {
		t.Fatalf("run_query = %s (%v)", out, err)
	}
	if bio := got.Results[0].Rows[1][1].(string); !strings.HasSuffix(bio, "… (1500 characters)") {
		t.Errorf("long value = %q", bio[len(bio)-30:])
	}
	log, err := os.ReadFile(filepath.Join(dir, "logs", "mcp.log"))
	if err != nil || !strings.Contains(string(log), `run_query "Shop" SELECT id, bio FROM customers ORDER BY id`) {
		t.Errorf("audit log = %q, %v", log, err)
	}
}

func TestWritesNeedTheFlagAndNeverReachProduction(t *testing.T) {
	path := sample(t)
	cs, _ := client(t, conns(path), Options{})
	if out, _ := call(t, cs, "run_query", map[string]any{"connection": "Shop", "sql": "DELETE FROM customers"}); !strings.Contains(out, "read-only") {
		t.Fatalf("write without --write = %s", out)
	}
	cs, _ = client(t, conns(path), Options{Write: true})
	if out, _ := call(t, cs, "run_query", map[string]any{"connection": "Prod shop", "sql": "DELETE FROM customers"}); !strings.Contains(out, "read-only") {
		t.Fatalf("write to Production = %s", out)
	}
	if out, _ := call(t, cs, "run_query", map[string]any{"connection": "Shop", "sql": "DELETE FROM customers WHERE id = 3"}); strings.Contains(out, "error") {
		t.Fatalf("write with --write = %s", out)
	}
}

func TestEnvFilter(t *testing.T) {
	cs, _ := client(t, conns(sample(t)), Options{Envs: []string{"prod"}})
	out, _ := call(t, cs, "list_connections", nil)
	if strings.Contains(out, `"Shop"`) || !strings.Contains(out, "Prod shop") {
		t.Fatalf("--env prod = %s", out)
	}
}

func TestParseFlags(t *testing.T) {
	o, err := ParseFlags([]string{"--env", "Staging, dev", "--write"}, os.Stderr)
	if err != nil || !o.Write || len(o.Envs) != 2 || o.Envs[0] != "staging" {
		t.Fatalf("flags = %+v, %v", o, err)
	}
}
