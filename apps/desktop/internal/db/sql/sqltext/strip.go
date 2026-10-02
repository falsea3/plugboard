package sqltext

import "strings"

func Strip(s string, syn Syntax) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' || (c == '"' && syn.DoubleQuotedStrings):
			i = skipQuoted(s, i, c, escapesBackslash(s, i, syn))
			b.WriteByte(' ')
		case c == '"' || c == '`':
			end := skipQuoted(s, i, c, false)
			b.WriteString(s[i : end+1])
			i = end
		case lineCommentAt(s, i, syn):
			i = skipLine(s, i)
			b.WriteByte(' ')
		case c == '/' && strings.HasPrefix(s[i:], "/*!") && syn.ExecutableComments:
			end := skipBlockComment(s, i)
			b.WriteByte(' ')
			b.WriteString(strings.TrimSuffix(s[i+3:end+1], "*/"))
			b.WriteByte(' ')
			i = end
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			i = skipBlockComment(s, i)
			b.WriteByte(' ')
		case c == '$' && syn.DollarQuotes:
			if end, ok := skipDollarQuoted(s, i); ok {
				i = end
				b.WriteByte(' ')
			} else {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
