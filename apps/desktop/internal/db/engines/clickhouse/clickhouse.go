package clickhouse

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Dialect struct{ dialect.Standard }

var (
	_ dialect.Dialect     = Dialect{}
	_ dialect.SyntaxCheck = Dialect{}
	_ dialect.SortKey     = Dialect{}
)

const defaultPort = 9000

func (Dialect) Driver() model.Driver { return model.ClickHouse }

func (Dialect) DefaultPort() int { return defaultPort }

func (Dialect) Open(c model.Connection) (*sql.DB, error) {
	port := c.Port
	if port == 0 {
		port = defaultPort
	}
	user := c.User
	if user == "" {
		user = "default"
	}
	o := &ch.Options{
		Addr:        []string{net.JoinHostPort(c.Host, strconv.Itoa(port))},
		Protocol:    ch.Native,
		Auth:        ch.Auth{Database: strings.TrimSpace(c.Database), Username: user, Password: c.Password},
		DialTimeout: 10 * time.Second,
		Settings:    ch.Settings{},
	}
	if c.ReadOnly {
		o.Settings["readonly"] = 2
	}
	switch c.SSLMode {
	case "require":
		o.TLS = &tls.Config{InsecureSkipVerify: true}
	case "verify-ca", "verify-full":
		o.TLS = &tls.Config{ServerName: c.Host}
	}
	return ch.OpenDB(o), nil
}

func (Dialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT version()`).Scan(&v)
	return "ClickHouse " + v, err
}

func (Dialect) Syntax() sqltext.Syntax {
	return sqltext.Syntax{HashComments: true, BackslashEscapes: true}
}

func (Dialect) QuoteString(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

func (Dialect) TextOf(expr string) string { return "toString(" + expr + ")" }

func (Dialect) LikeIgnoringCase() string { return "ILIKE" }

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (Dialect) EscapeLike(text string) (string, string) { return likeEscaper.Replace(text), "" }

func (Dialect) BooleanType() bool { return true }

func (Dialect) CanUpdateToDefault() bool { return false }

func (Dialect) Editable() bool { return false }

func (Dialect) ReturnsRows(stmt string) bool {
	return sqltext.ReturnsRows(stmt, "DESCRIBE", "DESC", "EXISTS", "CHECK")
}

func (Dialect) LockWait(int) (string, string) { return "", "" }

func (Dialect) TransactionalDDL() bool { return false }

var (
	syntaxPos = regexp.MustCompile(`Syntax error[^:]*: failed at position (\d+)`)
	failedAt  = regexp.MustCompile(`^Syntax error[^:]*: failed at position \d+ \(([^)]*)\):`)
)

func exception(err error) (int32, string, bool) {
	var e *ch.Exception
	if errors.As(err, &e) {
		return e.Code, e.Message, true
	}
	return 0, "", false
}

func (Dialect) Classify(err error) dialect.ErrorKind {
	code, msg, ok := exception(err)
	if !ok {
		if errors.Is(err, ch.ErrAcquireConnTimeout) {
			return dialect.ConnLost
		}
		return dialect.Other
	}
	switch code {
	case 62:
		return dialect.BadSyntax
	case 159, 160:
		return dialect.LockTimeout
	}
	if strings.Contains(msg, "Syntax error") {
		return dialect.BadSyntax
	}
	return dialect.Other
}

func (Dialect) ErrorPosition(err error, stmt string) int {
	_, msg, ok := exception(err)
	if !ok {
		return -1
	}
	m := syntaxPos.FindStringSubmatch(msg)
	if m == nil {
		return -1
	}
	n, _ := strconv.Atoi(m[1])
	var sh *shifted
	if errors.As(err, &sh) {
		n -= sh.shift
	}
	if n < 1 || n-1 > len(stmt) {
		return -1
	}
	return n - 1
}

func (Dialect) ErrorMessage(err error) string {
	_, msg, ok := exception(err)
	if !ok {
		return err.Error()
	}
	if m := failedAt.FindStringSubmatchIndex(msg); m != nil {
		near := msg[m[2]:m[3]]
		rest := msg[m[1]:]
		if i := strings.Index(rest, ". Expected "); i >= 0 {
			rest = shortList(rest[i+2:])
		} else {
			rest = ""
		}
		msg = strings.TrimSpace("Syntax error near “" + near + "”. " + rest)
	}
	return strings.TrimSpace(msg)
}

const explainAST = "EXPLAIN AST "

type shifted struct {
	err   error
	shift int
}

func (e *shifted) Error() string { return e.err.Error() }

func (e *shifted) Unwrap() error { return e.err }

func (Dialect) CheckSyntax(ctx context.Context, conn *sql.Conn, stmt string) error {
	rows, err := conn.QueryContext(ctx, explainAST+stmt)
	if err != nil {
		return &shifted{err: err, shift: len(explainAST)}
	}
	return rows.Close()
}

func shortList(expected string) string {
	head, list, ok := strings.Cut(expected, ": ")
	if !ok {
		return expected
	}
	items := strings.Split(list, ", ")
	if len(items) <= 6 {
		return expected
	}
	return head + ": " + strings.Join(items[:6], ", ") + "…"
}
