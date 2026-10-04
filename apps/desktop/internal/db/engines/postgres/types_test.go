package postgres_test

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestTypeOf(t *testing.T) {
	kinds := map[model.ColumnKind][]string{
		model.KindNumber:   {"integer", "bigint", "smallint", "numeric(10,2)", "double precision", "real", "money", "int4", "int8", "float8", "numeric"},
		model.KindBool:     {"boolean", "bool"},
		model.KindDateTime: {"date", "timestamp with time zone", "time without time zone", "timestamptz", "timetz", "timestamp(3) with time zone", "timestamp"},
		model.KindText:     {"text", "character varying(255)", "varchar(10)", "character(2)", "bpchar", "citext"},
		model.KindJSON:     {"json", "jsonb", "JSONB"},
		model.KindBinary:   {"bytea"},
		model.KindOther:    {"interval", "int4range", "integer[]", "_int4", "inet", "daterange", "tstzrange", "date[]", "text[]", "uuid", "mood"},
	}
	for want, types := range kinds {
		for _, typ := range types {
			if got := (postgres.Dialect{}).TypeOf(typ).Kind; got != want {
				t.Errorf("TypeOf(%q) = %q, want %q", typ, got, want)
			}
		}
	}
	d := postgres.Dialect{}
	if !d.TypeOf("timestamptz").Zone || !d.TypeOf("timestamp with time zone").Zone || d.TypeOf("timestamp without time zone").Zone {
		t.Error("time zones")
	}
	if !d.TypeOf("float4").Float32 || d.TypeOf("float8").Float32 || !d.TypeOf("date").Date || !d.TypeOf("timetz").Time {
		t.Error("value formats")
	}
}
