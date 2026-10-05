package sqltext

import (
	"strings"
	"unicode"
)

func Split(script string, syn Syntax) []string {
	var out []string
	start := 0
	n := len(script)
	delim := ";"
	var b blocks
	flush := func(end int) {
		stmt := strings.TrimSpace(script[start:end])
		if stmt != "" && FirstKeyword(stmt) != "" {
			out = append(out, stmt)
		}
		b = blocks{}
	}
	for i := 0; i < n; i++ {
		switch c := script[i]; {
		case b.words == 0 && (c == 'D' || c == 'd') && strings.TrimSpace(script[start:i]) == "":
			if d, next, ok := delimiterAt(script, i); ok {
				delim, start, i = d, next, next
				continue
			}
			i = wordEnd(script, i, &b)
		case delim != ";" && strings.HasPrefix(script[i:], delim):
			flush(i)
			i += len(delim) - 1
			start = i + 1
		case isWordByte(c) && (i == 0 || !isWordByte(script[i-1])):
			i = wordEnd(script, i, &b)
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(script, i, c, escapesBackslash(script, i, syn))
		case lineCommentAt(script, i, syn):
			i = skipLine(script, i)
		case c == '/' && i+1 < n && script[i+1] == '*':
			i = skipBlockComment(script, i)
		case c == '$' && syn.DollarQuotes:
			if end, ok := skipDollarQuoted(script, i); ok {
				i = end
			}
		case c == ';' && delim == ";" && !b.open():
			flush(i)
			start = i + 1
		}
	}
	flush(n)
	return out
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
