package sqlcore

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

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
		if got := normalizeValue(c.in, dialect.Type{}); got != c.want {
			t.Errorf("normalizeValue(%#v) = %#v, want %#v", c.in, got, c.want)
		}
	}
	list := dialect.Type{List: true}
	for in, want := range map[string]string{
		`[1,2]`:             normalizeValue([]uint8{1, 2}, list).(string),
		`{"source":"blog"}`: normalizeValue(map[string]string{"source": "blog"}, dialect.Type{}).(string),
		`[1,"x"]`:           normalizeValue([]any{1, "x"}, dialect.Type{}).(string),
		`["a","b"]`:         normalizeValue([]string{"a", "b"}, list).(string),
	} {
		if in != want {
			t.Errorf("list value = %s, want %s", want, in)
		}
	}
	if got := normalizeValue(nil, list); got != nil {
		t.Errorf("NULL list = %#v", got)
	}
	if got := normalizeValue([]byte("hi"), dialect.Type{Kind: model.KindBinary}); got != "0x6869" {
		t.Errorf("binary text = %#v", got)
	}
}

func TestFloat32WidenedByTheDriver(t *testing.T) {
	widened := float64(float32(5.2))
	if got := normalizeValue(widened, dialect.Type{Float32: true}); got != 5.2 {
		t.Errorf("real = %v, want 5.2", got)
	}
	if got := normalizeValue(widened, dialect.Type{}); got != widened {
		t.Errorf("double precision lost digits: %v", got)
	}
}
