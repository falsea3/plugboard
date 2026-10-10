package mcp

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestCreatesALivePostgresConnection(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_PG")
	if addr == "" {
		t.Skip("PLUGBOARD_TEST_PG not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	store := conns(sample(t))
	cs, _ := client(t, store, Options{Create: true})
	args := map[string]any{"name": "Live shop", "engine": "postgres", "host": addr[:i], "port": port, "user": "relay", "password": "nope", "database": "shop", "ssl_mode": "disable"}
	if out, isErr := call(t, cs, "create_connection", args); !isErr || !strings.Contains(out, "nothing was saved") || len(store.list) != 3 {
		t.Fatalf("wrong password = %s (%d saved)", out, len(store.list))
	}
	args["password"] = "relay"
	if out, isErr := call(t, cs, "create_connection", args); isErr || !strings.Contains(out, "PostgreSQL") {
		t.Fatalf("create = %s", out)
	}
	if out, _ := call(t, cs, "run_query", map[string]any{"connection": "Live shop", "sql": "SELECT count(*) FROM customers"}); !strings.Contains(out, "2500") {
		t.Fatalf("query = %s", out)
	}
}
