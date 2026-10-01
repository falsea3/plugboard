package db

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// Read-only sessions are enforced twice: the server session is put in a
// read-only transaction mode (see each Dialect.Open), and statements are
// screened here before any of them run. The screen errs towards refusing:
// only statement kinds known to be reads pass.

var readOnlyKeywords = map[string]bool{
	"SELECT": true, "SHOW": true, "DESCRIBE": true, "DESC": true, "VALUES": true,
	"TABLE": true, "FETCH": true, "BEGIN": true, "START": true, "COMMIT": true, "END": true,
	"ROLLBACK": true, "SAVEPOINT": true, "RELEASE": true, "USE": true, "RESET": true,
	"WITH": true, "EXPLAIN": true, "PRAGMA": true, "SET": true,
}

var (
	writeWord = regexp.MustCompile(`(?i)\b(insert|update|delete|merge|upsert|replace|truncate|drop|alter|create|grant|revoke|call|copy|lock|vacuum|reindex|cluster|refresh|import|load)\b`)
	// set_config() changes settings from inside any statement, including the
	// default_transaction_read_only that keeps the server side read-only.
	setConfig = regexp.MustCompile(`(?i)\bset_config\b`)
	// U&"\0073et_config" spells a name with escapes the screen can't read.
	unicodeEscape = regexp.MustCompile(`(?i)\bu&["']`)
	// SET that would undo the read-only session, toggle it per transaction,
	// or reach past this session (server-wide settings, accounts).
	unsafeSet    = regexp.MustCompile(`(?i)(read[\s_]*(write|only)|transaction|session\s+characteristics|role|session[\s_]+authorization|search_path|\bglobal\b|\bpersist(_only)?\b|@@(global|persist)|\bpassword\b|resource\s+group)`)
	selectInto   = regexp.MustCompile(`(?i)\binto\b`)
	intoVariable = regexp.MustCompile(`(?i)\binto\s+@`)
	beginWrite   = regexp.MustCompile(`(?i)read\s+write`)
	startTx      = regexp.MustCompile(`(?i)^\s*start\s+transaction\b`)
	rowLock      = regexp.MustCompile(`(?i)\bfor\s+(no\s+key\s+)?(update|share|key\s+share)\b|\block\s+in\s+share\s+mode\b`)
	// PRAGMA name(arg) sets a value, except for the *_info and *_list lookups.
	pragmaLookup = regexp.MustCompile(`(?i)^\s*pragma\s+(\w+\.)?\w+_(info|xinfo|list)\s*\(`)
	pragmaWrites = regexp.MustCompile(`(?i)\b(wal_checkpoint|optimize|incremental_vacuum)\b`)
)

// ReadOnlyReason returns why stmt may modify data or undo the read-only
// session, or "" if it is a read. The statement is read the way d's server
// reads it: only MySQL has # comments and backslash escapes in every string.
func ReadOnlyReason(stmt string, d model.Driver) string {
	kw := firstKeyword(stmt)
	if kw == "" {
		return ""
	}
	if !readOnlyKeywords[kw] {
		return kw + " statements"
	}
	body := stripStringsAndComments(stmt, d == model.MySQL)
	if setConfig.MatchString(body) {
		return "set_config()"
	}
	if unicodeEscape.MatchString(body) {
		return "Unicode-escaped names"
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
	case "PRAGMA":
		if strings.Contains(body, "=") || pragmaWrites.MatchString(body) {
			return "PRAGMA that changes settings or the file"
		}
		if strings.Contains(body, "(") && !pragmaLookup.MatchString(body) {
			return "PRAGMA that changes settings or the file"
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
		// MySQL also has START REPLICA, START GROUP_REPLICATION.
		if !startTx.MatchString(body) {
			return "START statements other than START TRANSACTION"
		}
		if beginWrite.MatchString(body) {
			return "READ WRITE transactions"
		}
	case "RESET":
		// PostgreSQL's RESET goes back to the session's start-up values,
		// read-only included; MySQL's RESET clears logs and persisted settings.
		if d != model.Postgres {
			return "RESET statements"
		}
	}
	return ""
}

// mayWrite reports whether stmt may change data or session state, so it
// must not be run twice behind the user's back.
func mayWrite(stmt string, d model.Driver) bool {
	return ReadOnlyReason(stmt, d) != ""
}

// stripStringsAndComments blanks out string literals and comments so keywords
// inside them (select 'drop table x') don't count. Quoted identifiers are kept
// as they are: "set_config" is still a name the server will call.
func stripStringsAndComments(s string, mysql bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' || (c == '"' && mysql):
			i = skipQuoted(s, i, c, escapesBackslash(s, i, mysql))
			b.WriteByte(' ')
		case c == '"' || c == '`':
			end := skipQuoted(s, i, c, false)
			b.WriteString(s[i : end+1])
			i = end
		case c == '-' && i+1 < len(s) && s[i+1] == '-', c == '#' && mysql:
			i = skipLine(s, i)
			b.WriteByte(' ')
		case c == '/' && strings.HasPrefix(s[i:], "/*!") && mysql:
			// MySQL runs the text of /*! … */ comments, so it counts.
			end := skipBlockComment(s, i)
			b.WriteByte(' ')
			b.WriteString(strings.TrimSuffix(s[i+3:end+1], "*/"))
			b.WriteByte(' ')
			i = end
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			i = skipBlockComment(s, i)
			b.WriteByte(' ')
		case c == '$' && !mysql:
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

// ReadOnlyError is returned when a read-only session refuses a script.
type ReadOnlyError struct{ Reason string }

func (e *ReadOnlyError) Error() string {
	return fmt.Sprintf("read-only connection: %s are not allowed. Turn off read-only for this connection first", e.Reason)
}

// WriteStatements returns the statements of script that a read-only session
// would refuse, for the editor to confirm before running them on Production.
func (s *Session) WriteStatements(script string) []string {
	out := []string{}
	for _, stmt := range SplitStatements(script, s.Conn.Driver == model.MySQL) {
		if mayWrite(stmt, s.Conn.Driver) {
			out = append(out, stmt)
		}
	}
	return out
}
