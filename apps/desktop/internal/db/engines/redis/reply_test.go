package redis

import (
	"reflect"
	"testing"
)

func TestReplyShapes(t *testing.T) {
	c := func(args ...string) command { return command{text: "x", args: args} }
	rs := reply(c("GET", "k"), "v")
	if len(rs.Columns) != 1 || !reflect.DeepEqual(rs.Rows, [][]any{{"v"}}) {
		t.Errorf("string reply = %+v", rs)
	}
	rs = reply(c("GET", "k"), nil)
	if !reflect.DeepEqual(rs.Rows, [][]any{{nil}}) {
		t.Errorf("nil reply = %+v", rs.Rows)
	}
	rs = reply(c("LRANGE"), []any{"a", int64(2), []any{"x"}})
	if rs.Columns[0].Name != "#" || !reflect.DeepEqual(rs.Rows, [][]any{{1, "a"}, {2, int64(2)}, {3, `["x"]`}}) {
		t.Errorf("array reply = %+v", rs.Rows)
	}
	rs = reply(c("HGETALL"), map[any]any{"b": "2", "a": "1"})
	if !reflect.DeepEqual(rs.Rows, [][]any{{"a", "1"}, {"b", "2"}}) {
		t.Errorf("map reply = %+v", rs.Rows)
	}
	rs = reply(c("INCR"), int64(1<<60))
	if rs.Rows[0][0] != "1152921504606846976" {
		t.Errorf("big int = %v", rs.Rows[0][0])
	}
	rs = reply(c("INFO"), "# Server\r\nredis_version:7.4.0\r\n\r\n# Clients\r\nconnected_clients:1\r\n")
	if !reflect.DeepEqual(rs.Rows, [][]any{{"Server", "redis_version", "7.4.0"}, {"Clients", "connected_clients", "1"}}) {
		t.Errorf("info = %+v", rs.Rows)
	}
	rs = reply(c("ZRANGE", "z", "0", "-1", "WITHSCORES"), []any{[]any{"a", 1.5}, []any{"b", float64(2)}})
	if rs.Columns[1].Name != "member" || !reflect.DeepEqual(rs.Rows, [][]any{{1, "a", "1.5"}, {2, "b", "2"}}) {
		t.Errorf("withscores = %+v %+v", rs.Columns, rs.Rows)
	}
	rs = reply(c("GEOPOS"), []any{[]any{"1", "2", "3"}, []any{"x"}})
	if len(rs.Columns) != 2 {
		t.Errorf("ragged tuples = %+v", rs.Columns)
	}
	if got := cell(3.5); got != "3.5" {
		t.Errorf("double = %v", got)
	}
}

func TestKeyCounts(t *testing.T) {
	got := keyCounts("# Keyspace\r\ndb0:keys=17,expires=1,avg_ttl=0,subexpiry=0\r\ndb9:keys=7,expires=0,avg_ttl=0\r\n")
	if len(got) != 2 || got["0"] != 17 || got["9"] != 7 {
		t.Fatalf("keyCounts = %v", got)
	}
}
