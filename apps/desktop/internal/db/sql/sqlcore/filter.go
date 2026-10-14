package sqlcore

import (
	"fmt"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func whereClause(d dialect.Dialect, cols map[string]model.Column, filters []model.Filter, readOnly bool) (string, []any, error) {
	var parts []string
	var args []any
	bind := func(col model.Column, v string) string {
		args = append(args, v)
		return d.Param(len(args), col)
	}
	bindText := func(v string) string {
		args = append(args, v)
		return d.Placeholder(len(args))
	}
	asText := func(name string) string { return d.TextOf(d.QuoteIdent(name)) }
	like := d.LikeIgnoringCase()

	for _, f := range filters {
		if f.Op == "sql" {
			expr, err := customCondition(d, f.Value, readOnly)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, "("+expr+"\n)")
			continue
		}
		col, ok := cols[f.Column]
		if !ok {
			return "", nil, fmt.Errorf("unknown column %s", f.Column)
		}
		q := d.QuoteIdent(f.Column)
		switch f.Op {
		case "=", "!=", "<", ">", "<=", ">=":
			op := f.Op
			if op == "!=" {
				op = "<>"
			}
			parts = append(parts, q+" "+op+" "+bind(col, f.Value))
		case "contains", "not_contains", "starts", "ends":
			pattern, escape := d.EscapeLike(f.Value)
			switch f.Op {
			case "starts":
				pattern += "%"
			case "ends":
				pattern = "%" + pattern
			default:
				pattern = "%" + pattern + "%"
			}
			not := ""
			if f.Op == "not_contains" {
				not = "NOT "
			}
			parts = append(parts, asText(f.Column)+" "+not+like+" "+bindText(pattern)+escape)
		case "in", "not_in":
			var items []string
			for _, v := range strings.Split(f.Value, ",") {
				if v = strings.TrimSpace(v); v != "" {
					items = append(items, bind(col, v))
				}
			}
			if len(items) == 0 {
				return "", nil, fmt.Errorf("%s: give a comma-separated list of values", f.Column)
			}
			not := ""
			if f.Op == "not_in" {
				not = "NOT "
			}
			parts = append(parts, q+" "+not+"IN ("+strings.Join(items, ", ")+")")
		case "empty":
			parts = append(parts, asText(f.Column)+" = ''")
		case "not_empty":
			parts = append(parts, asText(f.Column)+" <> ''")
		case "null":
			parts = append(parts, q+" IS NULL")
		case "not_null":
			parts = append(parts, q+" IS NOT NULL")
		default:
			return "", nil, fmt.Errorf("unknown filter %q", f.Op)
		}
	}
	return strings.Join(parts, " AND "), args, nil
}

func customCondition(d dialect.Dialect, expr string, readOnly bool) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", apperr.New("bad_value", "write a condition, like price > 100 AND status <> 'void'")
	}
	bare := sqltext.Strip(expr, d.Syntax())
	if strings.Contains(bare, ";") {
		return "", apperr.New("bad_value", "a filter is one condition — leave out the ;")
	}
	depth := 0
	for _, ch := range bare {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth < 0 {
			break
		}
	}
	if depth != 0 {
		return "", apperr.New("bad_value", "the brackets in the filter don't match")
	}
	if readOnly {
		if reason := ReadOnlyReason(d, "SELECT 1 WHERE "+expr); reason != "" {
			return "", &db.ReadOnlyError{Reason: reason}
		}
	}
	return expr, nil
}
