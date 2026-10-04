package sqlite_test

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/sqlite"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestTypeOf(t *testing.T) {
	kinds := map[model.ColumnKind][]string{
		model.KindNumber:   {"INTEGER", "INT", "BIGINT", "REAL", "FLOAT", "DOUBLE", "NUMERIC", "DECIMAL(10,2)"},
		model.KindBool:     {"BOOLEAN", "bool"},
		model.KindDateTime: {"DATE", "DATETIME", "TIMESTAMP", "TIME"},
		model.KindText:     {"TEXT", "VARCHAR(20)", "CHARACTER(2)", "CLOB", "STRING"},
		model.KindJSON:     {"JSON", "jsonb"},
		model.KindBinary:   {"BLOB", "VARBINARY"},
		model.KindOther:    {""},
	}
	for want, types := range kinds {
		for _, typ := range types {
			if got := (sqlite.Dialect{}).TypeOf(typ).Kind; got != want {
				t.Errorf("TypeOf(%q) = %q, want %q", typ, got, want)
			}
		}
	}
	if (sqlite.Dialect{}).TypeOf("FLOAT").Float32 {
		t.Error("SQLite FLOAT is a double")
	}
}
