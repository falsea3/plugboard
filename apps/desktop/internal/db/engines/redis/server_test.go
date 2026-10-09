package redis_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/redis"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const scratch = "9"

func open(t *testing.T, readOnly bool) *redis.Session {
	t.Helper()
	addr := os.Getenv("PLUGBOARD_TEST_REDIS")
	if addr == "" {
		t.Skip("PLUGBOARD_TEST_REDIS not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{Driver: model.Redis, Host: addr[:i], Port: port, Password: "relay", Database: scratch, ReadOnly: readOnly}
	s, err := redis.Open(context.Background(), "t", c, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func run(t *testing.T, s *redis.Session, script string) []model.ResultSet {
	t.Helper()
	res, err := s.Run(context.Background(), script)
	if err != nil {
		t.Fatalf("run %q: %v", script, err)
	}
	return res
}

func seed(t *testing.T) *redis.Session {
	s := open(t, false)
	run(t, s, `FLUSHDB
SET t:str hello
HSET t:hash a 1 b 2
RPUSH t:list x y z
SADD t:set m1 m2
ZADD t:zset 1.5 one 2 two
XADD t:stream * type signup
SET "t:bin\xff" "\x00\x01"`)
	return s
}

func TestInfoAndDatabases(t *testing.T) {
	s := open(t, false)
	info, err := s.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(info.ServerVersion, "Redis ") && !strings.HasPrefix(info.ServerVersion, "Valkey ") {
		t.Errorf("version = %q", info.ServerVersion)
	}
	if len(info.Schemas) < 16 || info.DefaultSchema != scratch || !info.Engine.KeyValue {
		t.Errorf("info = %+v", info)
	}
}

func TestScanPagesThroughEveryKey(t *testing.T) {
	s := seed(t)
	ctx := context.Background()
	seen := map[string]string{}
	cursor := ""
	for range 100 {
		page, err := s.ScanKeys(ctx, scratch, "t:*", cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range page.Keys {
			seen[k.Name] = k.Type
		}
		if page.Done {
			break
		}
		cursor = page.Cursor
	}
	want := map[string]string{"t:str": "string", "t:hash": "hash", "t:list": "list", "t:set": "set", "t:zset": "zset", "t:stream": "stream", `t:bin\xff`: "string"}
	for k, typ := range want {
		if seen[k] != typ {
			t.Errorf("key %s = %q, want %q (seen %v)", k, seen[k], typ, seen)
		}
	}
	if _, err := s.ScanKeys(ctx, "999", "*", "", 10); err == nil {
		t.Error("a database past the last one was accepted")
	}
}

func TestReadsEachType(t *testing.T) {
	s := seed(t)
	ctx := context.Background()
	read := func(key string) model.KeyValue {
		t.Helper()
		v, err := s.ReadKey(ctx, scratch, key, "")
		if err != nil {
			t.Fatalf("read %s: %v", key, err)
		}
		return v
	}
	if v := read("t:str"); v.Text != "hello" || v.Size != 5 || v.TTL != -1 {
		t.Errorf("string = %+v", v)
	}
	if v := read("t:hash"); v.Size != 2 || len(v.Items) != 2 || !v.Done {
		t.Errorf("hash = %+v", v)
	}
	if v := read("t:list"); len(v.Items) != 3 || v.Items[2].Field != "2" || v.Items[2].Value != "z" {
		t.Errorf("list = %+v", v)
	}
	if v := read("t:zset"); v.Items[0].Value != "one" || v.Items[0].Score != "1.5" {
		t.Errorf("zset = %+v", v)
	}
	if v := read("t:stream"); len(v.Items) != 1 || v.Items[0].Value != `{"type":"signup"}` {
		t.Errorf("stream = %+v", v)
	}
	page, _ := s.ScanKeys(ctx, scratch, "t:bin*", "", 100)
	if len(page.Keys) != 1 {
		t.Fatalf("binary key page = %+v", page)
	}
	if v := read(page.Keys[0].Key); !v.Binary || v.Text != `\x00\x01` {
		t.Errorf("binary = %+v", v)
	}
	if _, err := s.ReadKey(ctx, scratch, "t:nope", ""); err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Errorf("missing key = %v", err)
	}
}

func TestPagesALongList(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	run(t, s, "FLUSHDB")
	args := []string{"RPUSH t:long"}
	for i := range 450 {
		args = append(args, strconv.Itoa(i))
	}
	run(t, s, strings.Join(args, " "))
	var got []string
	cursor := ""
	for {
		v, err := s.ReadKey(ctx, scratch, "t:long", cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range v.Items {
			got = append(got, it.Value)
		}
		if v.Done {
			break
		}
		cursor = v.Cursor
	}
	if len(got) != 450 || got[449] != "449" {
		t.Fatalf("read %d items, last %q", len(got), got[len(got)-1])
	}
}
