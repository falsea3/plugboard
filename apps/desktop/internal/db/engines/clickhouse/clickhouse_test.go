package clickhouse

import (
	"errors"
	"slices"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestTypeOf(t *testing.T) {
	cases := map[string]model.ColumnKind{
		"UInt64": model.KindNumber, "Nullable(Decimal(12, 2))": model.KindNumber, "LowCardinality(Nullable(String))": model.KindText,
		"Bool": model.KindBool, "Date32": model.KindDateTime, "DateTime64(3, 'UTC')": model.KindDateTime, "Enum8('a' = 1)": model.KindText,
		"UUID": model.KindText, "JSON": model.KindJSON, "Array(String)": model.KindOther, "Map(String, String)": model.KindOther,
	}
	for in, want := range cases {
		if got := (Dialect{}).TypeOf(in).Kind; got != want {
			t.Errorf("TypeOf(%s) = %q, want %q", in, got, want)
		}
	}
	if !(Dialect{}).TypeOf("map(string, string)").List || (Dialect{}).TypeOf("lowcardinality(string)").Kind != model.KindText {
		t.Error("lower-case names from the driver")
	}
	if !(Dialect{}).TypeOf("Float32").Float32 {
		t.Error("Float32")
	}
}

func TestEnumValues(t *testing.T) {
	got := enumValues(`LowCardinality(Nullable(Enum8('free' = 1, 'it\'s' = 2, 'a, b' = -3)))`)
	if !slices.Equal(got, []string{"free", "it's", "a, b"}) {
		t.Fatalf("enumValues = %q", got)
	}
	if enumValues("String") != nil {
		t.Fatal("not an enum")
	}
}

func TestSplitKey(t *testing.T) {
	if got := splitKey("customer_id, toDate(created_at), cityHash64(a, b)"); !slices.Equal(got, []string{"customer_id", "toDate(created_at)", "cityHash64(a, b)"}) {
		t.Fatalf("splitKey = %q", got)
	}
}

func TestReadOnlyScreen(t *testing.T) {
	d := Dialect{}
	for stmt, ok := range map[string]bool{
		"SELECT 1": true, "DESCRIBE t": true, "EXISTS TABLE t": true, "SHOW TABLES": true, "USE shop": true, "SET max_threads = 4": true,
		"SET readonly = 0": false, "INSERT INTO t VALUES (1)": false, "OPTIMIZE TABLE t": false, "SYSTEM DROP DNS CACHE": false,
		"SELECT * FROM t INTO OUTFILE 'x.csv'": false, "TRUNCATE t": false, "KILL QUERY WHERE 1": false,
	} {
		if got := sqlcore.ReadOnlyReason(d, stmt) == ""; got != ok {
			t.Errorf("%s: read = %v, want %v", stmt, got, ok)
		}
	}
}

func TestErrors(t *testing.T) {
	d := Dialect{}
	syntax := &shifted{err: &ch.Exception{Code: 62, Message: "Syntax error: failed at position 19 (2): 2. Expected one of: a, b, c, d, e, f, g, h"}, shift: len(explainAST)}
	if d.Classify(syntax) != dialect.BadSyntax || d.ErrorPosition(syntax, "SELEC 2") != 6 {
		t.Errorf("syntax: %v %d", d.Classify(syntax), d.ErrorPosition(syntax, "SELEC 2"))
	}
	if got := d.ErrorMessage(syntax); got != "Syntax error near “2”. Expected one of: a, b, c, d, e, f…" {
		t.Errorf("message = %q", got)
	}
	other := &ch.Exception{Code: 60, Message: "Unknown table expression identifier 'nope'"}
	if d.Classify(other) != dialect.Other || d.ErrorPosition(other, "x") != -1 || d.ErrorMessage(other) != other.Message {
		t.Error("unknown table")
	}
	if d.Classify(errors.New("x")) != dialect.Other {
		t.Error("plain error")
	}
}

func TestDDL(t *testing.T) {
	d := Dialect{}
	if got := d.RenameTable("shop", "a", "b"); got != `RENAME TABLE "shop"."a" TO "shop"."b"` {
		t.Errorf("rename = %s", got)
	}
	if got := d.CreateIndex("shop", "t", model.NewIndex{Name: "i", Columns: []string{"a", "b"}}); got != `ALTER TABLE "shop"."t" ADD INDEX "i" ("a", "b") TYPE minmax GRANULARITY 1` {
		t.Errorf("index = %s", got)
	}
	cur := model.Column{Name: "a", Type: "String"}
	typ, null := "UInt8", true
	got, _ := d.ColumnDDL(t.Context(), nil, "shop", "t", "ALTER TABLE t", cur, model.ColumnChange{Type: &typ, Nullable: &null})
	if !slices.Equal(got, []string{`ALTER TABLE t MODIFY COLUMN "a" Nullable(UInt8)`}) {
		t.Errorf("column = %q", got)
	}
	if p, c := d.EscapeLike(`50%_a\`); p != `50\%\_a\\` || c != "" {
		t.Errorf("like = %q %q", p, c)
	}
}
