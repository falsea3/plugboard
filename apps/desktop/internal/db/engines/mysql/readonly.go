package mysql

import "regexp"

var unsafeSet = regexp.MustCompile(`(?i)(\bglobal\b|\bpersist(_only)?\b|@@(global|persist)|\bpassword\b|resource\s+group)`)

func (Dialect) ReadKeywords() []string { return []string{"USE", "DESCRIBE", "DESC"} }

func (Dialect) WriteReason(keyword, body string) string {
	if keyword == "SET" && unsafeSet.MatchString(body) {
		return "SET of transaction, session or server settings"
	}
	return ""
}
