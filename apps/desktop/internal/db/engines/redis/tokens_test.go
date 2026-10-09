package redis

import "testing"

func TestTokensRoundTrip(t *testing.T) {
	for _, k := range []string{"user:1", "", "ключ", "\xff\x00a", hexMark + "abc", "a\\b"} {
		got, err := untoken(token(k))
		if err != nil || got != k {
			t.Errorf("token(%q) round trip = %q, %v", k, got, err)
		}
	}
	if token("user:1") != "user:1" {
		t.Error("a UTF-8 key travels as itself")
	}
	if _, err := untoken(hexMark + "zz"); err == nil {
		t.Error("bad hex accepted")
	}
}

func TestDisplayEscapesBinary(t *testing.T) {
	if s, bin := display("plain"); s != "plain" || bin {
		t.Errorf("display(plain) = %q %v", s, bin)
	}
	if s, bin := display("a\x00"); s != `a\x00` || !bin {
		t.Errorf("display(NUL) = %q %v", s, bin)
	}
	if s, bin := display("a\xff\\\n"); s != `a\xff\\\x0a` || !bin {
		t.Errorf("display(binary) = %q %v", s, bin)
	}
}
