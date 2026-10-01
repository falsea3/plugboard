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

// pgx widens a PostgreSQL real to float64 before we see it.
func TestRealWidenedByTheDriver(t *testing.T) {
	widened := float64(float32(5.2)) // 5.199999809265137
	if got := normalizeValue(widened, "float4"); got != 5.2 {
		t.Errorf("real = %v, want 5.2", got)
	}
	if got := normalizeValue(widened, "float8"); got != widened {
		t.Errorf("double precision lost digits: %v", got)
	}
}
