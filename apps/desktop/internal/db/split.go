package db

import (
	"strings"
	"unicode"
)

// SplitStatements cuts a script on top-level semicolons. Semicolons inside
// quotes, quoted identifiers, comments and PostgreSQL dollar-quoted bodies
// don't count. Empty and comment-only statements are dropped. MySQL alone
// treats '#' as a comment and backslash as an escape inside strings; in
// PostgreSQL '#' is an operator (jsonb #>) and backslashes are literal
// except in E'…' strings.
func SplitStatements(script string, mysql bool) []string {
	var out []string
	start := 0
	n := len(script)
	flush := func(end int) {
		stmt := strings.TrimSpace(script[start:end])
		if stmt != "" && !onlyComments(stmt) {
			out = append(out, stmt)
		}
	}
	for i := 0; i < n; i++ {
		switch c := script[i]; {
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(script, i, c, escapesBackslash(script, i, mysql))
		case lineCommentAt(script, i, mysql):
			i = skipLine(script, i)
		case c == '/' && i+1 < n && script[i+1] == '*':
			i = skipBlockComment(script, i)
		case c == '$' && !mysql:
			if end, ok := skipDollarQuoted(script, i); ok {
				i = end
			}
		case c == ';':
			flush(i)
			start = i + 1
		}
	}
	flush(n)
	return out
}

// skipQuoted returns the index of the closing quote. A doubled quote is an
// escape everywhere; a backslash only when backslashEscapes is set.
func skipQuoted(s string, i int, q byte, backslashEscapes bool) int {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			if backslashEscapes {
				j++
			}
		case q:
			if j+1 < len(s) && s[j+1] == q {
				j++
				continue
			}
			return j
		}
	}
	return len(s) - 1
}

// lineCommentAt reports a comment running to the end of the line from s[i]:
// "--", and in MySQL "#". MySQL needs a space or control character after the
// dashes — there 1--1 is 1 - -1, and SELECT 1--1 INTO OUTFILE … must not read
// as a harmless SELECT 1.
func lineCommentAt(s string, i int, mysql bool) bool {
	if s[i] == '#' {
		return mysql
	}
	if s[i] != '-' || i+1 >= len(s) || s[i+1] != '-' {
		return false
	}
	return !mysql || i+2 >= len(s) || s[i+2] <= ' '
}

func skipLine(s string, i int) int {
	if nl := strings.IndexByte(s[i:], '\n'); nl >= 0 {
		return i + nl
	}
	return len(s) - 1
}

func skipBlockComment(s string, i int) int {
	if end := strings.Index(s[i+2:], "*/"); end >= 0 {
		return i + 2 + end + 1
	}
	return len(s) - 1
}

// escapesBackslash reports whether the literal opening at s[i] treats
// backslash as an escape: every MySQL string does, and PostgreSQL's E'…'
// strings. Backtick identifiers never do.
func escapesBackslash(s string, i int, mysql bool) bool {
	if s[i] == '`' {
		return false
	}
	if mysql {
		return true
	}
	return s[i] == '\'' && i > 0 && (s[i-1] == 'E' || s[i-1] == 'e') && (i == 1 || !isWordByte(s[i-2]))
}

// skipDollarQuoted returns the index of the last byte of a PostgreSQL
// $$…$$ or $tag$…$tag$ string opening at s[i]; false if none opens there.
func skipDollarQuoted(s string, i int) (int, bool) {
	tag, ok := dollarTag(s, i)
	if !ok {
		return i, false
	}
	if end := strings.Index(s[i+len(tag):], tag); end >= 0 {
		return i + len(tag) + end + len(tag) - 1, true
	}
	return len(s) - 1, true
}

// dollarTag recognises $$ and $tag$ openers. $1-style placeholders are not
// tags, and neither is a $ inside a name: a$b$ is one identifier.
func dollarTag(s string, i int) (string, bool) {
	if i > 0 && (isWordByte(s[i-1]) || s[i-1] >= 0x80) {
		return "", false
	}
	for j := i + 1; j < len(s); j++ {
		c := rune(s[j])
		if c == '$' {
			tag := s[i : j+1]
			if len(tag) > 2 && unicode.IsDigit(rune(tag[1])) {
				return "", false
			}
			return tag, true
		}
		if !(c == '_' || unicode.IsLetter(c) || unicode.IsDigit(c)) {
			return "", false
		}
	}
	return "", false
}

func onlyComments(stmt string) bool {
	return firstKeyword(stmt) == ""
}

// firstKeyword returns the upper-cased first word after leading comments and
// parentheses.
func firstKeyword(stmt string) string {
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

var rowKeywords = map[string]bool{
	"SELECT": true, "WITH": true, "SHOW": true, "EXPLAIN": true, "VALUES": true,
	"TABLE": true, "DESCRIBE": true, "DESC": true, "PRAGMA": true, "FETCH": true,
}

func returnsRows(stmt string) bool {
	if rowKeywords[firstKeyword(stmt)] {
		return true
	}
	return containsKeyword(stmt, "RETURNING")
}

func containsKeyword(stmt, kw string) bool {
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

func isWordByte(b byte) bool {
	return b == '_' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}
