package mysql_test

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestTypeOf(t *testing.T) {
	kinds := map[model.ColumnKind][]string{
		model.KindNumber:   {"int(11)", "int", "bigint unsigned", "unsigned bigint", "decimal(12,2)", "double", "float", "tinyint(4)", "tinyint", "mediumint"},
		model.KindBool:     {"tinyint(1)", "boolean"},
		model.KindDateTime: {"date", "datetime", "datetime(6)", "timestamp", "time"},
		model.KindText:     {"varchar(255)", "char(2)", "text", "longtext"},
		model.KindJSON:     {"json", "JSON"},
		model.KindBinary:   {"blob", "varbinary(16)", "binary(2)", "bit(1)", "bit"},
		model.KindOther:    {"enum('a','b')", "set('a')", "year", "geometry"},
	}
	for want, types := range kinds {
		for _, typ := range types {
			if got := (mysql.Dialect{}).TypeOf(typ).Kind; got != want {
				t.Errorf("TypeOf(%q) = %q, want %q", typ, got, want)
			}
		}
	}
	if !(mysql.Dialect{}).TypeOf("float").Float32 || (mysql.Dialect{}).TypeOf("double").Float32 {
		t.Error("single precision")
	}
}
