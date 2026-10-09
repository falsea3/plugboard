package clickhouse

import "regexp"

var (
	unsafeSet = regexp.MustCompile(`(?i)\b(readonly|allow_ddl|allow_introspection_functions|profile)\b`)
	writeIn   = regexp.MustCompile(`(?i)\binto\s+outfile\b`)
)

func (Dialect) ReadKeywords() []string {
	return []string{"DESCRIBE", "DESC", "EXISTS", "USE", "CHECK"}
}

func (Dialect) WriteReason(keyword, body string) string {
	switch {
	case keyword == "SET" && unsafeSet.MatchString(body):
		return "SET of read-only or permission settings"
	case writeIn.MatchString(body):
		return "INTO OUTFILE"
	}
	return ""
}
