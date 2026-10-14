package sqlcore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

const (
	maxSafeInteger = 1<<53 - 1
	maxBinaryBytes = 512
)

func normalizeValue(v any, t dialect.Type) any {
	if t.List && v != nil {
		return jsonText(v)
	}
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
	switch reflect.ValueOf(v).Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return jsonText(v)
	}
	return fmt.Sprint(v)
}

const MaxCellBytes = 64 << 10

func keepRows(rs *model.ResultSet, n int) {
	rs.Rows = rs.Rows[:n]
	rs.Cut = slices.DeleteFunc(rs.Cut, func(c model.CutCell) bool { return c.Row >= n })
}

func reverseRows(rs *model.ResultSet) {
	slices.Reverse(rs.Rows)
	for i := range rs.Cut {
		rs.Cut[i].Row = len(rs.Rows) - 1 - rs.Cut[i].Row
	}
}

func cutText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func jsonText(v any) string {
	if b, ok := v.([]byte); ok {
		nums := make([]int, len(b))
		for i, n := range b {
			nums[i] = int(n)
		}
		v = nums
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
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
