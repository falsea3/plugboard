package sqltext

import (
	"strings"
	"unicode"
)

func Split(script string, syn Syntax) []string {
	var out []string
	delim := ";"
	for pos := 0; pos < len(script); {
		from, end, next, d, _ := scanStatement(script, pos, delim, syn, true)
		delim = d
		if stmt := strings.TrimSpace(script[from:end]); stmt != "" && FirstKeyword(stmt) != "" {
			out = append(out, stmt)
		}
		pos = next
	}
	return out
}

func scanStatement(s string, start int, delim string, syn Syntax, eof bool) (from, end, next int, d string, found bool) {
	n := len(s)
	from = start
	var b blocks
	for i := start; i < n; i++ {
		switch c := s[i]; {
		case b.words == 0 && c == '\\' && blank(s[from:i], syn):
			nl := strings.IndexByte(s[i:], '\n')
			if nl < 0 {
				if !eof {
					return from, n, n, delim, false
				}
				return i, n, n, delim, true
			}
			return i, i + nl, i + nl + 1, delim, true
		case b.words == 0 && (c == 'D' || c == 'd') && blank(s[from:i], syn):
			if nd, after, ok := delimiterAt(s, i); ok {
				if after == n && !eof {
					return from, n, n, delim, false
				}
				delim, from, i = nd, after, after
				continue
			}
			i = wordEnd(s, i, &b)
		case delim != ";" && strings.HasPrefix(s[i:], delim):
			return from, i, i + len(delim), delim, true
		case isWordByte(c) && (i == 0 || !isWordByte(s[i-1])):
			i = wordEnd(s, i, &b)
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(s, i, c, escapesBackslash(s, i, syn))
		case lineCommentAt(s, i, syn):
			i = skipLine(s, i)
		case c == '/' && i+1 < n && s[i+1] == '*':
			i = skipBlockComment(s, i)
		case c == '$' && syn.DollarQuotes:
			if e, ok := skipDollarQuoted(s, i); ok {
				i = e
			}
		case c == ';' && delim == ";" && !b.open():
			return from, i, i + 1, delim, true
		}
	}
	return from, n, n, delim, false
}

func blank(s string, syn Syntax) bool {
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] <= ' ':
		case lineCommentAt(s, i, syn):
			i = skipLine(s, i)
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '*':
			i = skipBlockComment(s, i)
		default:
			return false
		}
	}
	return true
}

func wordEnd(s string, i int, b *blocks) int {
	j := i
	for j < len(s) && isWordByte(s[j]) {
		j++
	}
	b.word(s[i:j])
	return j - 1
}

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

func lineCommentAt(s string, i int, syn Syntax) bool {
	if s[i] == '#' {
		return syn.HashComments
	}
	if s[i] != '-' || i+1 >= len(s) || s[i+1] != '-' {
		return false
	}
	return !syn.DashCommentNeedsSpace || i+2 >= len(s) || s[i+2] <= ' '
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

func escapesBackslash(s string, i int, syn Syntax) bool {
	if s[i] == '`' {
		return false
	}
	if syn.BackslashEscapes {
		return true
	}
	return syn.EscapeStrings && s[i] == '\'' && i > 0 && (s[i-1] == 'E' || s[i-1] == 'e') && (i == 1 || !isWordByte(s[i-2]))
}

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

func isWordByte(b byte) bool {
	return b == '_' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}
