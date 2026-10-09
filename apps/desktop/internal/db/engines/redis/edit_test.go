package redis_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestEditsEachType(t *testing.T) {
	s := seed(t)
	ctx := context.Background()
	edit := func(e model.KeyEdit) error { return s.EditKey(ctx, scratch, e) }
	must := func(e model.KeyEdit) {
		t.Helper()
		if err := edit(e); err != nil {
			t.Fatalf("%+v: %v", e, err)
		}
	}
	taken := func(e model.KeyEdit, code string) {
		t.Helper()
		err := edit(e)
		if err == nil || !strings.Contains(codeOf(err), code) {
			t.Fatalf("%+v = %v, want %s", e, err, code)
		}
	}
	value := func(script string) any {
		t.Helper()
		return run(t, s, script)[0].Rows[0][0]
	}

	must(model.KeyEdit{Key: "t:str", Op: "set", Value: "bye"})
	must(model.KeyEdit{Key: "t:hash", Op: "hset", Field: "a", Value: "10"})
	taken(model.KeyEdit{Key: "t:hash", Op: "hadd", Field: "b", Value: "x"}, "exists")
	must(model.KeyEdit{Key: "t:hash", Op: "hadd", Field: "c", Value: "3"})
	taken(model.KeyEdit{Key: "t:hash", Op: "hrename", Field: "a", Value: "b"}, "exists")
	must(model.KeyEdit{Key: "t:hash", Op: "hrename", Field: "a", Value: "aa"})
	must(model.KeyEdit{Key: "t:hash", Op: "hdel", Field: "b"})
	if v := value("HGET t:hash aa"); v != "10" {
		t.Errorf("renamed field = %v", v)
	}
	taken(model.KeyEdit{Key: "t:list", Op: "lset", Index: 1, Old: "nope", Value: "Y"}, "row_gone")
	must(model.KeyEdit{Key: "t:list", Op: "lset", Index: 1, Old: "y", Value: "Y"})
	must(model.KeyEdit{Key: "t:list", Op: "lrem", Index: 0, Old: "x"})
	must(model.KeyEdit{Key: "t:list", Op: "rpush", Value: "w"})
	if v := value("LRANGE t:list 0 -1"); v != 1 {
		t.Errorf("list first cell = %v", v)
	}
	if rows := run(t, s, "LRANGE t:list 0 -1")[0].Rows; len(rows) != 3 || rows[0][1] != "Y" || rows[2][1] != "w" {
		t.Errorf("list = %v", rows)
	}
	taken(model.KeyEdit{Key: "t:set", Op: "sadd", Value: "m1"}, "exists")
	must(model.KeyEdit{Key: "t:set", Op: "sreplace", Field: "m1", Value: "m3"})
	must(model.KeyEdit{Key: "t:set", Op: "srem", Field: "m2"})
	if v := value("SMEMBERS t:set"); v != 1 {
		t.Errorf("set = %v", v)
	}
	taken(model.KeyEdit{Key: "t:zset", Op: "zadd", Value: "x", Score: "abc"}, "bad_value")
	must(model.KeyEdit{Key: "t:zset", Op: "zadd", Value: "three", Score: "-inf"})
	must(model.KeyEdit{Key: "t:zset", Op: "zscore", Field: "one", Score: "9"})
	must(model.KeyEdit{Key: "t:zset", Op: "zrename", Field: "two", Value: "deux"})
	if v := value("ZSCORE t:zset deux"); v != "2" {
		t.Errorf("renamed member score = %v", v)
	}
	must(model.KeyEdit{Key: "t:stream", Op: "xadd", Value: `{"type":"order"}`})
	taken(model.KeyEdit{Key: "t:stream", Op: "xadd", Value: `[1]`}, "bad_value")
	must(model.KeyEdit{Key: "t:str", Op: "expire", Index: 100})
	if v := value("TTL t:str"); v.(int64) <= 0 {
		t.Errorf("ttl = %v", v)
	}
	must(model.KeyEdit{Key: "t:str", Op: "expire", Index: -1})
	taken(model.KeyEdit{Key: "t:str", Op: "rename", Value: "t:hash"}, "exists")
	must(model.KeyEdit{Key: "t:str", Op: "rename", Value: "t:str2"})
	taken(model.KeyEdit{Key: "t:hash", Op: "create:hash", Field: "f", Value: "v"}, "exists")
	must(model.KeyEdit{Key: "t:new", Op: "create:zset", Value: "m", Score: "1"})
	must(model.KeyEdit{Key: "t:new", Op: "delete"})
	taken(model.KeyEdit{Key: "t:new", Op: "hset", Field: "a", Value: "b"}, "key_gone")
}

func codeOf(err error) string {
	var c interface{ Code() string }
	if errors.As(err, &c) {
		return c.Code()
	}
	return err.Error()
}

func TestReadOnlyRefusesWrites(t *testing.T) {
	seed(t)
	s := open(t, true)
	ctx := context.Background()
	if _, err := s.Run(ctx, "GET t:str\nHGETALL t:hash\nINFO server\nSELECT 0\nPING"); err != nil {
		t.Fatalf("reads refused: %v", err)
	}
	_, err := s.Run(ctx, "GET t:str\nSET t:str x")
	var se *db.StatementError
	var ro *db.ReadOnlyError
	if !errors.As(err, &se) || se.Index != 1 || !errors.As(err, &ro) {
		t.Fatalf("write in read-only = %v", err)
	}
	for _, w := range []string{"EVAL \"return 1\" 0", "FLUSHALL", "DEL t:str", "NOTACOMMAND x"} {
		if _, err := s.Run(ctx, w); !errors.As(err, &ro) {
			t.Errorf("%s = %v", w, err)
		}
	}
	if err := s.EditKey(ctx, scratch, model.KeyEdit{Key: "t:str", Op: "set", Value: "x"}); !errors.As(err, &ro) {
		t.Errorf("edit in read-only = %v", err)
	}
	got := s.WriteStatements("GET a\nSET a 1\n# c\nZRANGE z 0 -1\nHDEL h f")
	if strings.Join(got, "|") != "SET a 1|HDEL h f" {
		t.Errorf("write statements = %q", got)
	}
}

func TestConsoleKeepsItsConnection(t *testing.T) {
	s := seed(t)
	ctx := context.Background()
	run(t, s, "SELECT 1")
	if v := run(t, s, "GET t:str")[0].Rows[0][0]; v != nil {
		t.Errorf("SELECT didn't stick: %v", v)
	}
	run(t, s, "SELECT "+scratch)
	_, err := s.Run(ctx, "GET t:str\nHGET t:str x\nGET t:str")
	var se *db.StatementError
	if !errors.As(err, &se) || se.Index != 1 || !strings.Contains(err.Error(), "WRONGTYPE") {
		t.Errorf("wrong type = %v", err)
	}
	if _, err := s.Run(ctx, "SUBSCRIBE news"); err == nil {
		t.Error("SUBSCRIBE ran in the console")
	}
	if _, err := s.Run(ctx, `GET "open`); err == nil {
		t.Error("unbalanced quotes ran")
	}
}
