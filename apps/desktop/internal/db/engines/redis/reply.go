package redis

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const maxSafeInt = 1 << 53

func reply(c command, v any) model.ResultSet {
	rs := model.ResultSet{Statement: c.text, HasRows: true}
	value := []model.ResultColumn{{Name: "value", Kind: model.KindText}}
	switch x := v.(type) {
	case []any:
		if names := tupleNames(c, x); names != nil {
			return tuples(rs, names, x)
		}
		rs.Columns = []model.ResultColumn{{Name: "#", Kind: model.KindNumber}, value[0]}
		for i, item := range x {
			rs.Rows = append(rs.Rows, []any{i + 1, cell(item)})
		}
	case map[any]any:
		rs.Columns = []model.ResultColumn{{Name: "field", Kind: model.KindText}, value[0]}
		keys := slices.SortedFunc(maps.Keys(x), func(a, b any) int { return strings.Compare(text(a), text(b)) })
		for _, k := range keys {
			rs.Rows = append(rs.Rows, []any{text(k), cell(x[k])})
		}
	case string:
		if strings.EqualFold(c.args[0], "info") {
			return infoRows(rs, x)
		}
		rs.Columns = value
		rs.Rows = [][]any{{cell(x)}}
	default:
		rs.Columns = value
		if _, ok := x.(int64); ok {
			rs.Columns = []model.ResultColumn{{Name: "value", Kind: model.KindNumber}}
		}
		rs.Rows = [][]any{{cell(x)}}
	}
	if rs.Rows == nil {
		rs.Rows = [][]any{}
	}
	return rs
}

func tupleNames(c command, items []any) []string {
	n := 0
	for _, item := range items {
		t, ok := item.([]any)
		if !ok || len(t) < 2 || len(t) > 6 || (n != 0 && len(t) != n) {
			return nil
		}
		for _, v := range t {
			switch v.(type) {
			case []any, map[any]any:
				return nil
			}
		}
		n = len(t)
	}
	if n == 0 {
		return nil
	}
	if n == 2 && slices.ContainsFunc(c.args, func(a string) bool { return strings.EqualFold(a, "withscores") }) {
		return []string{"member", "score"}
	}
	names := make([]string, n)
	for i := range names {
		names[i] = strconv.Itoa(i + 1)
	}
	return names
}

func tuples(rs model.ResultSet, names []string, items []any) model.ResultSet {
	rs.Columns = []model.ResultColumn{{Name: "#", Kind: model.KindNumber}}
	for _, n := range names {
		rs.Columns = append(rs.Columns, model.ResultColumn{Name: n, Kind: model.KindText})
	}
	for i, item := range items {
		row := []any{i + 1}
		for _, v := range item.([]any) {
			row = append(row, cell(v))
		}
		rs.Rows = append(rs.Rows, row)
	}
	return rs
}

func infoRows(rs model.ResultSet, info string) model.ResultSet {
	rs.Columns = []model.ResultColumn{{Name: "section", Kind: model.KindText}, {Name: "field", Kind: model.KindText}, {Name: "value", Kind: model.KindText}}
	rs.Rows = [][]any{}
	section := ""
	for line := range strings.Lines(info) {
		line = strings.TrimSpace(line)
		if name, ok := strings.CutPrefix(line, "# "); ok {
			section = name
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			rs.Rows = append(rs.Rows, []any{section, k, v})
		}
	}
	return rs
}

func cell(v any) any {
	switch x := v.(type) {
	case nil, bool:
		return x
	case int64:
		if x > maxSafeInt || x < -maxSafeInt {
			return strconv.FormatInt(x, 10)
		}
		return x
	case string:
		s, _ := display(x)
		return s
	case []any, map[any]any, map[any]bool:
		b, _ := json.Marshal(plain(x))
		return string(b)
	}
	return text(v)
}

func plain(v any) any {
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = plain(item)
		}
		return out
	case map[any]any:
		out := map[string]any{}
		for k, item := range x {
			out[text(k)] = plain(item)
		}
		return out
	case map[any]bool:
		out := []string{}
		for k := range x {
			out = append(out, text(k))
		}
		slices.Sort(out)
		return out
	case string:
		s, _ := display(x)
		return s
	}
	return v
}

func text(v any) string {
	switch x := v.(type) {
	case string:
		s, _ := display(x)
		return s
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}
