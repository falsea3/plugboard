package cassandra

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const maxSafeInt = 1 << 53

var columnTypes = []string{
	"text", "int", "bigint", "smallint", "tinyint", "varint", "float", "double", "decimal", "boolean",
	"uuid", "timeuuid", "timestamp", "date", "time", "duration", "inet", "blob",
	"list<text>", "set<text>", "map<text, text>", "frozen<list<int>>",
}

func baseType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if i := strings.IndexAny(t, "<("); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

func kindOf(t string) model.ColumnKind {
	switch baseType(t) {
	case "int", "bigint", "smallint", "tinyint", "varint", "counter", "float", "double", "decimal":
		return model.KindNumber
	case "boolean":
		return model.KindBool
	case "timestamp", "date", "time":
		return model.KindDateTime
	case "text", "varchar", "ascii", "uuid", "timeuuid", "inet":
		return model.KindText
	case "blob":
		return model.KindBinary
	}
	return model.KindOther
}

func normalize(v any, cqlType string) any {
	rv := reflect.ValueOf(v)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil
	}
	v = rv.Interface()
	switch x := v.(type) {
	case bool, string:
		return x
	case int, int8, int16, int32:
		return x
	case int64:
		if x > maxSafeInt || x < -maxSafeInt {
			return strconv.FormatInt(x, 10)
		}
		return x
	case float32:
		return finite(float64(x), 32)
	case float64:
		return finite(x, 64)
	case time.Time:
		if baseType(cqlType) == "date" {
			return x.UTC().Format(time.DateOnly)
		}
		return x.UTC().Format("2006-01-02 15:04:05.000-0700")
	case time.Duration:
		if baseType(cqlType) == "time" {
			return clock(x)
		}
		return x.String()
	case []byte:
		return "0x" + hex.EncodeToString(x)
	case big.Int:
		return x.String()
	case net.IP:
		return x.String()
	case fmt.Stringer:
		return x.String()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		b, err := json.Marshal(plain(v))
		if err == nil {
			return string(b)
		}
	}
	return fmt.Sprint(v)
}

func finite(f float64, bits int) any {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return strconv.FormatFloat(f, 'g', -1, bits)
	}
	if bits == 32 {
		g, _ := strconv.ParseFloat(strconv.FormatFloat(f, 'g', -1, 32), 64)
		return g
	}
	return f
}

func clock(d time.Duration) string {
	h := d / time.Hour
	m := d % time.Hour / time.Minute
	s := d % time.Minute / time.Second
	ns := d % time.Second
	out := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	if ns > 0 {
		out += "." + strings.TrimRight(fmt.Sprintf("%09d", int64(ns)), "0")
	}
	return out
}

func plain(v any) any {
	rv := reflect.ValueOf(v)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil
	}
	switch rv.Kind() {
	case reflect.Map:
		out := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprint(plain(iter.Key().Interface()))] = plain(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return normalize(rv.Interface(), "blob")
		}
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = plain(rv.Index(i).Interface())
		}
		return out
	}
	n := normalize(rv.Interface(), "")
	if s, ok := n.(string); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil && rv.Kind() >= reflect.Int && rv.Kind() <= reflect.Float64 {
			return f
		}
	}
	return n
}

func jsonFor(col model.Column, text string) (string, error) {
	t := strings.TrimSpace(text)
	switch baseType(col.Type) {
	case "int", "bigint", "smallint", "tinyint", "varint", "counter", "float", "double", "decimal":
		switch {
		case t == "NaN" || t == "Infinity" || t == "-Infinity":
			return strconv.Quote(t), nil
		case t != "" && (t[0] == '-' || t[0] >= '0' && t[0] <= '9') && json.Valid([]byte(t)):
			return t, nil
		}
		return "", fmt.Errorf("%s needs a number, not %q", col.Name, text)
	case "boolean":
		switch strings.ToLower(t) {
		case "true", "false":
			return strings.ToLower(t), nil
		}
		return "", fmt.Errorf("%s is true or false", col.Name)
	case "text", "varchar", "ascii", "uuid", "timeuuid", "inet", "timestamp", "date", "time", "duration", "blob":
		b, _ := json.Marshal(text)
		return string(b), nil
	}
	if !json.Valid([]byte(t)) {
		return "", fmt.Errorf("%s takes JSON, like [1, 2] or {\"a\": 1}", col.Name)
	}
	return t, nil
}
