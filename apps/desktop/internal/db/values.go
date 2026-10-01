package db

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxSafeInteger = 1<<53 - 1
	maxBinaryBytes = 512
)

// normalizeValue turns whatever the driver scanned into something that
// survives JSON and JavaScript intact: big integers and non-finite floats
// become strings, bytes become text or a hex literal, times become ISO strings.
func normalizeValue(v any, dbType string) any {
	switch x := v.(type) {
	case nil, bool, string:
		return x
	case int64:
		if x > maxSafeInteger || x < -maxSafeInteger {
			return strconv.FormatInt(x, 10)
		}
		return x
	case int32, int16, int8, int, uint8, uint16, uint32:
		return x
	case uint64:
		if x > maxSafeInteger {
			return strconv.FormatUint(x, 10)
		}
		return x
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return strconv.FormatFloat(x, 'g', -1, 64)
		}
		return x
	case float32:
		// Widening 0.1f to float64 gives 0.10000000149011612; go through the
		// shortest 32-bit decimal instead so a real shows as written.
		f, _ := strconv.ParseFloat(strconv.FormatFloat(float64(x), 'g', -1, 32), 64)
		return normalizeValue(f, dbType)
	case time.Time:
		return formatTime(x, dbType)
	case []byte:
		// MySQL BIT arrives as raw bytes; b'1' is a valid but invisible "\x01".
		if isBinaryType(dbType) || dbType == "bit" || !utf8.Valid(x) {
			return hexLiteral(x)
		}
		return string(x)
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(v)
}

func formatTime(t time.Time, dbType string) string {
	switch dbType {
	case "date":
		return t.Format(time.DateOnly)
	case "time", "timetz":
		return t.Format("15:04:05.999999")
	}
	if t.Location() == time.UTC && !strings.HasSuffix(dbType, "tz") {
		return t.Format("2006-01-02 15:04:05.999999")
	}
	return t.Format("2006-01-02 15:04:05.999999Z07:00")
}

func isBinaryType(t string) bool {
	switch t {
	case "bytea", "blob", "binary", "varbinary", "tinyblob", "mediumblob", "longblob":
		return true
	}
	return false
}

func hexLiteral(b []byte) string {
	if len(b) > maxBinaryBytes {
		return fmt.Sprintf("0x%s… (%d bytes)", hex.EncodeToString(b[:maxBinaryBytes]), len(b))
	}
	return "0x" + hex.EncodeToString(b)
}
