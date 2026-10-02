package postgres

import "regexp"

var (
	setConfig     = regexp.MustCompile(`(?i)\bset_config\b`)
	unicodeEscape = regexp.MustCompile(`(?i)\bu&["']`)
	unsafeSet     = regexp.MustCompile(`(?i)search_path`)
)

func (Dialect) ReadKeywords() []string { return []string{"RESET"} }

func (Dialect) WriteReason(keyword, body string) string {
	if setConfig.MatchString(body) {
		return "set_config()"
	}
	if unicodeEscape.MatchString(body) {
		return "Unicode-escaped names"
	}
	if keyword == "SET" && unsafeSet.MatchString(body) {
		return "SET of transaction, session or server settings"
	}
	return ""
}
