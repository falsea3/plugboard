package sqltext

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

var pgSyntax = Syntax{DollarQuotes: true, EscapeStrings: true, DoubleQuotedStrings: false}
var mySyntax = Syntax{HashComments: true, DashCommentNeedsSpace: true, BackslashEscapes: true, ExecutableComments: true}

func scanAll(t *testing.T, r io.Reader, syn Syntax, chunk int) []string {
	t.Helper()
	sc := NewScanner(r, syn)
	sc.chunk = chunk
	var out []string
	for {
		stmt, err := sc.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, stmt)
	}
}

func TestScannerMatchesSplitAtEveryChunkSize(t *testing.T) {
	scripts := []struct {
		syn Syntax
		sql string
	}{
		{pgSyntax, "select 1; select 'a;b';\n-- c;\nCREATE FUNCTION f() RETURNS int AS $x$ select 1; $x$ LANGUAGE sql;\nselect E'it\\'s';"},
		{mySyntax, "SET a=1;\nDELIMITER $$\nCREATE PROCEDURE p() BEGIN SELECT 1; SELECT 2; END$$\nDELIMITER ;\n# note;\nSELECT `x;y` FROM t;\n/*!40101 SET b=2 */;"},
	}
	for _, s := range scripts {
		want := Split(s.sql, s.syn)
		for _, size := range []int{1, 2, 3, 7, 64} {
			got := scanAll(t, iotest.OneByteReader(strings.NewReader(s.sql)), s.syn, size)
			if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
				t.Fatalf("got %q\nwant %q", got, want)
			}
		}
	}
}

func TestScannerSmallChunks(t *testing.T) {
	sql := strings.Repeat("insert into t values (1, 'a;b', $$x;y$$);\n", 2000)
	sc := NewScanner(strings.NewReader(sql), pgSyntax)
	sc.chunk = 7
	n := 0
	for {
		stmt, err := sc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if stmt != "insert into t values (1, 'a;b', $$x;y$$)" {
			t.Fatalf("statement %d = %q", n, stmt)
		}
		n++
	}
	if n != 2000 || sc.Consumed() != int64(len(sql)) {
		t.Fatalf("n = %d, consumed %d of %d", n, sc.Consumed(), len(sql))
	}
}

func TestScannerCopyDataAndMetaCommands(t *testing.T) {
	dump := "\\restrict abc\n\nSET x = 1;\nCOPY public.t (a, b) FROM stdin;\n1\tit's; fine\n2\t\\N\n\\.\nSELECT 2;\n\\unrestrict abc\n"
	sc := NewScanner(strings.NewReader(dump), pgSyntax)
	sc.chunk = 5
	var got []string
	for {
		stmt, err := sc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, stmt)
		if strings.HasPrefix(stmt, "COPY") {
			var data bytes.Buffer
			if err := sc.CopyData(&data); err != nil {
				t.Fatal(err)
			}
			if data.String() != "1\tit's; fine\n2\t\\N\n" {
				t.Fatalf("copy data = %q", data.String())
			}
		}
	}
	want := []string{`\restrict abc`, "SET x = 1", "COPY public.t (a, b) FROM stdin", "SELECT 2", `\unrestrict abc`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q", got)
	}
}
