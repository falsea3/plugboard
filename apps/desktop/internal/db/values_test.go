package db

import "testing"

func TestNormalizeValue(t *testing.T) {
	cases := []struct {
		in   any
		want any
	}{
		{float32(0.1), 0.1},
		{float32(3.25), 3.25},
		{int64(1 << 53), "9007199254740992"},
		{int64(42), int64(42)},
		{[]byte("hi"), "hi"},
		{[]byte{0xff, 0xfe}, "0xfffe"},
	}
	for _, c := range cases {
		if got := normalizeValue(c.in, "text"); got != c.want {
			t.Errorf("normalizeValue(%#v) = %#v, want %#v", c.in, got, c.want)
		}
	}
}
