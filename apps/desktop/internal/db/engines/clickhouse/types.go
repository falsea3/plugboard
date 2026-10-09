package clickhouse

import (
	"strconv"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func itoa(n uint64) string { return strconv.FormatUint(n, 10) }

func (Dialect) TypeOf(t string) dialect.Type {
	t = strings.ToLower(strings.TrimSpace(t))
	for _, w := range []string{"lowcardinality", "nullable", "simpleaggregatefunction"} {
		t = unwrap(t, w)
	}
	switch w := dialect.FirstWord(t); {
	case w == "array" || w == "map" || w == "tuple" || w == "nested":
		return dialect.Type{List: true}
	case w == "bool" || w == "boolean":
		return dialect.Type{Kind: model.KindBool}
	case w == "date" || w == "date32":
		return dialect.Type{Kind: model.KindDateTime, Date: true}
	case w == "datetime" || w == "datetime64":
		return dialect.Type{Kind: model.KindDateTime, Zone: strings.Contains(t, "'")}
	case w == "json" || w == "object":
		return dialect.Type{Kind: model.KindJSON}
	case w == "float32":
		return dialect.Type{Kind: model.KindNumber, Float32: true}
	case strings.HasPrefix(w, "int") || strings.HasPrefix(w, "uint") || strings.HasPrefix(w, "float") || strings.HasPrefix(w, "decimal") || w == "bfloat16":
		return dialect.Type{Kind: model.KindNumber}
	case w == "string" || w == "fixedstring" || w == "uuid" || w == "ipv4" || w == "ipv6" || strings.HasPrefix(w, "enum"):
		return dialect.Type{Kind: model.KindText}
	}
	return dialect.Type{}
}

var columnTypes = []string{
	"String", "UInt8", "UInt16", "UInt32", "UInt64", "Int8", "Int16", "Int32", "Int64", "Float32", "Float64",
	"Decimal(18, 2)", "Bool", "Date", "Date32", "DateTime", "DateTime64(3)", "UUID", "IPv4", "IPv6",
	"LowCardinality(String)", "Nullable(String)", "Array(String)", "Map(String, String)", "JSON",
}

func (Dialect) ColumnTypes() []string { return columnTypes }
