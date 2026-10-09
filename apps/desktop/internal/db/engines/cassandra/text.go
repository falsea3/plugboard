package cassandra

import (
	"regexp"
	"slices"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
)

var syntax = sqltext.Syntax{DollarQuotes: true}

var reads = []string{"SELECT", "DESCRIBE", "DESC", "USE", "LIST"}

var useKeyspace = regexp.MustCompile(`(?is)^\s*use\s+("(?:[^"]|"")+"|\w+)\s*;?\s*$`)

func split(script string) []string { return sqltext.Split(script, syntax) }

func readOnlyReason(stmt string) string {
	kw := sqltext.FirstKeyword(stmt)
	if kw == "" || slices.Contains(reads, kw) {
		return ""
	}
	return kw + " statements"
}

func usedKeyspace(stmt string) (string, bool) {
	m := useKeyspace.FindStringSubmatch(stmt)
	if m == nil {
		return "", false
	}
	name := m[1]
	if strings.HasPrefix(name, `"`) {
		return strings.ReplaceAll(name[1:len(name)-1], `""`, `"`), true
	}
	return strings.ToLower(name), true
}

func quote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func qualified(ks, table string) string { return quote(ks) + "." + quote(table) }
