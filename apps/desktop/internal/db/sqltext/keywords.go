package sqltext

import (
	"slices"
	"strings"
	"unicode"
)

func FirstKeyword(stmt string) string {
	s := stmt
	for {
		s = strings.TrimLeftFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == '(' })
		switch {
		case strings.HasPrefix(s, "--"), strings.HasPrefix(s, "#"):
			nl := strings.IndexByte(s, '\n')
			if nl < 0 {
				return ""
			}
			s = s[nl+1:]
		case strings.HasPrefix(s, "/*"):
			end := strings.Index(s, "*/")
			if end < 0 {
				return ""
			}
			s = s[end+2:]
		default:
			end := strings.IndexFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
			if end < 0 {
				end = len(s)
			}
			return strings.ToUpper(s[:end])
		}
	}
}

func ContainsKeyword(stmt, kw string) bool {
	upper := strings.ToUpper(stmt)
	for idx := 0; ; {
		j := strings.Index(upper[idx:], kw)
		if j < 0 {
			return false
		}
		p := idx + j
		before := p == 0 || !isWordByte(upper[p-1])
		after := p+len(kw) >= len(upper) || !isWordByte(upper[p+len(kw)])
		if before && after {
			return true
		}
		idx = p + len(kw)
	}
}

var rowKeywords = []string{"SELECT", "WITH", "SHOW", "EXPLAIN", "VALUES", "TABLE", "FETCH"}

func ReturnsRows(stmt string, more ...string) bool {
	kw := FirstKeyword(stmt)
	if slices.Contains(rowKeywords, kw) || slices.Contains(more, kw) {
		return true
	}
	return ContainsKeyword(stmt, "RETURNING")
}
