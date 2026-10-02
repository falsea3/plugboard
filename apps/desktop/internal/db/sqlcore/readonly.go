package sqlcore

import (
	"regexp"
	"slices"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqltext"
)

var standardReads = []string{
	"SELECT", "WITH", "VALUES", "TABLE", "SHOW", "EXPLAIN", "FETCH", "SET",
	"BEGIN", "START", "COMMIT", "END", "ROLLBACK", "SAVEPOINT", "RELEASE",
}

var (
	writeWord    = regexp.MustCompile(`(?i)\b(insert|update|delete|merge|upsert|replace|truncate|drop|alter|create|grant|revoke|call|copy|lock|vacuum|reindex|cluster|refresh|import|load)\b`)
	unsafeSet    = regexp.MustCompile(`(?i)(read[\s_]*(write|only)|transaction|session\s+characteristics|role|session[\s_]+authorization)`)
	selectInto   = regexp.MustCompile(`(?i)\binto\b`)
	intoVariable = regexp.MustCompile(`(?i)\binto\s+@`)
	beginWrite   = regexp.MustCompile(`(?i)read\s+write`)
	startTx      = regexp.MustCompile(`(?i)^\s*start\s+transaction\b`)
	rowLock      = regexp.MustCompile(`(?i)\bfor\s+(no\s+key\s+)?(update|share|key\s+share)\b|\block\s+in\s+share\s+mode\b`)
)

func ReadOnlyReason(d dialect.Dialect, stmt string) string {
	kw := sqltext.FirstKeyword(stmt)
	if kw == "" {
		return ""
	}
	if !slices.Contains(standardReads, kw) && !slices.Contains(d.ReadKeywords(), kw) {
		return kw + " statements"
	}
	body := sqltext.Strip(stmt, d.Syntax())
	if reason := d.WriteReason(kw, body); reason != "" {
		return reason
	}
	switch kw {
	case "WITH", "EXPLAIN":
		if m := writeWord.FindString(body); m != "" {
			return strings.ToUpper(m) + " inside " + kw
		}
	case "SELECT":
		if selectInto.MatchString(body) && !intoVariable.MatchString(body) {
			return "SELECT … INTO"
		}
		if rowLock.MatchString(body) {
			return "SELECT … FOR UPDATE/SHARE (row locks)"
		}
	case "SET":
		if unsafeSet.MatchString(body) {
			return "SET of transaction, session or server settings"
		}
	case "BEGIN":
		if beginWrite.MatchString(body) {
			return "READ WRITE transactions"
		}
	case "START":
		if !startTx.MatchString(body) {
			return "START statements other than START TRANSACTION"
		}
		if beginWrite.MatchString(body) {
			return "READ WRITE transactions"
		}
	}
	return ""
}

func (s *Session) mayWrite(stmt string) bool {
	return ReadOnlyReason(s.Dialect, stmt) != ""
}

func (s *Session) WriteStatements(script string) []string {
	out := []string{}
	for _, stmt := range sqltext.Split(script, s.Dialect.Syntax()) {
		if s.mayWrite(stmt) {
			out = append(out, stmt)
		}
	}
	return out
}
