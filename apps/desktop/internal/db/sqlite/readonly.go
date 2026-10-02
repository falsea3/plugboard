package sqlite

import (
	"regexp"
	"strings"
)

var (
	pragmaLookup = regexp.MustCompile(`(?i)^\s*pragma\s+(\w+\.)?\w+_(info|xinfo|list)\s*\(`)
	pragmaWrites = regexp.MustCompile(`(?i)\b(wal_checkpoint|optimize|incremental_vacuum)\b`)
)

func (Dialect) ReadKeywords() []string { return []string{"PRAGMA"} }

func (Dialect) WriteReason(keyword, body string) string {
	if keyword != "PRAGMA" {
		return ""
	}
	if strings.Contains(body, "=") || pragmaWrites.MatchString(body) {
		return "PRAGMA that changes settings or the file"
	}
	if strings.Contains(body, "(") && !pragmaLookup.MatchString(body) {
		return "PRAGMA that changes settings or the file"
	}
	return ""
}
