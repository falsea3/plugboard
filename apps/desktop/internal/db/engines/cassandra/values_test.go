package cassandra

import (
	"math/big"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"gopkg.in/inf.v0"
	"testing"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestNormalize(t *testing.T) {
	ts := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)
	i := 42
	var nilInt *int
	big1 := big.NewInt(0).Lsh(big.NewInt(1), 70)
	cases := []struct {
		in   any
		typ  string
		want any
	}{
		{&i, "int", 42},
		{nilInt, "int", nil},
		{int64(1 << 60), "bigint", "1152921504606846976"},
		{ts, "timestamp", "2026-01-03 10:00:00.000+0000"},
		{ts, "date", "2026-01-03"},
		{10*time.Hour + 30*time.Minute + 5*time.Second + 500*time.Millisecond, "time", "10:30:05.5"},
		{[]byte{1, 0xff}, "blob", "0x01ff"},
		{*big1, "varint", "1180591620717411303424"},
		{map[string]int{"hub": 1}, "map<text, int>", `{"hub":1}`},
		{[]string{"a", "b"}, "set<text>", `["a","b"]`},
		{map[string]any{"city": "Austin", "zip": nil}, "frozen<address>", `{"city":"Austin","zip":null}`},
		{float32(0.1), "float", 0.1},
	}
	dec := inf.NewDec(12900, 2)
	cases = append(cases,
		struct {
			in   any
			typ  string
			want any
		}{dec, "decimal", "129.00"},
		struct {
			in   any
			typ  string
			want any
		}{&dec, "decimal", "129.00"},
		struct {
			in   any
			typ  string
			want any
		}{&gocql.Duration{Months: 14, Days: 2, Nanoseconds: 3*int64(time.Hour) + 5e6}, "duration", "1y2mo2d3h5ms"},
		struct {
			in   any
			typ  string
			want any
		}{map[string]*inf.Dec{"a": inf.NewDec(15, 1)}, "map<text, decimal>", `{"a":"1.5"}`},
		struct {
			in   any
			typ  string
			want any
		}{struct{ A int }{1}, "", `{"A":1}`},
	)
	for _, c := range cases {
		if got := normalize(c.in, c.typ); got != c.want {
			t.Errorf("normalize(%#v, %s) = %#v, want %#v", c.in, c.typ, got, c.want)
		}
	}
}

func TestJSONFor(t *testing.T) {
	col := func(typ string) model.Column { return model.Column{Name: "c", Type: typ} }
	ok := map[[2]string]string{
		{"int", "42"}:                        "42",
		{"decimal", "-1.50"}:                 "-1.50",
		{"boolean", "TRUE"}:                  "true",
		{"text", `say "hi"`}:                 `"say \"hi\""`,
		{"timestamp", "2026-01-03 10:00:00"}: `"2026-01-03 10:00:00"`,
		{"set<text>", `["a"]`}:               `["a"]`,
		{"frozen<address>", `{"city":"x"}`}:  `{"city":"x"}`,
		{"double", "NaN"}:                    `"NaN"`,
	}
	for in, want := range ok {
		got, err := jsonFor(col(in[0]), in[1])
		if err != nil || got != want {
			t.Errorf("jsonFor(%s, %q) = %q, %v; want %q", in[0], in[1], got, err, want)
		}
	}
	for _, bad := range [][2]string{{"int", "abc"}, {"int", `"1"`}, {"boolean", "yes"}, {"list<int>", "[1,"}} {
		if _, err := jsonFor(col(bad[0]), bad[1]); err == nil {
			t.Errorf("jsonFor(%s, %q) accepted", bad[0], bad[1])
		}
	}
}

func TestKindOf(t *testing.T) {
	for typ, want := range map[string]model.ColumnKind{"int": model.KindNumber, "frozen<address>": model.KindOther, "timeuuid": model.KindText, "blob": model.KindBinary, "date": model.KindDateTime, "map<text, int>": model.KindOther} {
		if got := kindOf(typ); got != want {
			t.Errorf("kindOf(%s) = %q", typ, got)
		}
	}
}
