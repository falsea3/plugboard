package redis

import (
	"slices"
	"testing"
)

func TestSplitArgsLikeRedisCli(t *testing.T) {
	cases := map[string][]string{
		`SET a b`:                   {"SET", "a", "b"},
		`  set   "a b"  'c d'  `:    {"set", "a b", "c d"},
		`set k "line\nnext\x41\"q"`: {"set", "k", "line\nnextA\"q"},
		`set k 'it\'s'`:             {"set", "k", "it's"},
		`set k a"b c"d`:             {"set", "k", "ab cd"},
		`hset h "" x`:               {"hset", "h", "", "x"},
	}
	for in, want := range cases {
		got, err := splitArgs(in)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("splitArgs(%s) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{`set "a`, `set 'a`} {
		if _, err := splitArgs(bad); err == nil {
			t.Errorf("splitArgs(%s) accepted unbalanced quotes", bad)
		}
	}
}

func TestParseScriptSkipsBlankAndComments(t *testing.T) {
	cmds, err := parseScript("# note\n\nGET a\n// x\n  INCR b  \n")
	if err != nil || len(cmds) != 2 || cmds[0].line != 2 || cmds[1].text != "INCR b" {
		t.Fatalf("parseScript = %+v, %v", cmds, err)
	}
	_, err = parseScript("GET a\nSET \"b")
	if le, ok := err.(*lineError); !ok || le.line != 1 {
		t.Fatalf("bad line error = %v", err)
	}
}

func TestCommandKeysAndFlags(t *testing.T) {
	if commandKey([]string{"CONFIG", "GET", "x"}) != "config|get" || commandKey([]string{"GET", "x"}) != "get" {
		t.Error("commandKey")
	}
	if !readFlags([]string{"readonly", "fast"}) || readFlags([]string{"write"}) || readFlags([]string{"admin"}) {
		t.Error("readFlags")
	}
}
