package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// whereClause turns filter-bar conditions into SQL. Column names must exist
// in cols and values are always bound as parameters, numbered from $1; a
// caller binding more appends to the returned args.
func whereClause(d Dialect, driver model.Driver, cols map[string]model.Column, filters []model.Filter) (string, []any, error) {
	var parts []string
	var args []any
	bind := func(col model.Column, v string) string {
		args = append(args, v)
		if driver == model.Postgres {
			return pgParam(len(args), col)
		}
		return "?"
	}
	bindText := func(v string) string {
		args = append(args, v)
		if driver == model.Postgres {
			return "$" + strconv.Itoa(len(args))
		}
		return "?"
	}
	asText := func(name string) string {
		switch driver {
		case model.MySQL:
			return "CAST(" + d.QuoteIdent(name) + " AS CHAR)"
		default:
			return "CAST(" + d.QuoteIdent(name) + " AS TEXT)"
		}
	}
	like := "LIKE"
	if driver == model.Postgres {
		like = "ILIKE" // MySQL and SQLite LIKE are already case-insensitive
	}
	// '!' rather than backslash: it needs no escaping in any engine or SQL mode.
	const escape = ` ESCAPE '!'`

	for _, f := range filters {
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
			pattern := escapeLike(f.Value)
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

var likeEscaper = strings.NewReplacer(`!`, `!!`, `%`, `!%`, `_`, `!_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}
