package sqlcore

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const (
	maxSafeInteger = 1<<53 - 1
	maxBinaryBytes = 512
)

func normalizeValue(v any, t dialect.Type) any {
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
		if t.Float32 {
			return normalizeValue(float32(x), t)
		}
		return x
	case float32:
		f, _ := strconv.ParseFloat(strconv.FormatFloat(float64(x), 'g', -1, 32), 64)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return strconv.FormatFloat(f, 'g', -1, 64)
		}
		return f
	case time.Time:
		return formatTime(x, t)
	case []byte:
		if t.Kind == model.KindBinary || !utf8.Valid(x) {
			return hexLiteral(x)
		}
		return string(x)
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(v)
}

func formatTime(v time.Time, t dialect.Type) string {
	switch {
	case t.Date:
		return v.Format(time.DateOnly)
	case t.Time:
		return v.Format("15:04:05.999999")
	case v.Location() == time.UTC && !t.Zone:
		return v.Format("2006-01-02 15:04:05.999999")
	}
	return v.Format("2006-01-02 15:04:05.999999Z07:00")
}

func hexLiteral(b []byte) string {
	if len(b) > maxBinaryBytes {
		return fmt.Sprintf("0x%s… (%d bytes)", hex.EncodeToString(b[:maxBinaryBytes]), len(b))
	}
	return "0x" + hex.EncodeToString(b)
}
