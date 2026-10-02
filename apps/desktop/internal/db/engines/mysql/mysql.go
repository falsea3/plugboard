package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Dialect struct {
	dialect.Standard
	mariaDB bool
	version [3]int
}

func (d Dialect) ForVersion(v string) dialect.Dialect {
	d.mariaDB = strings.HasPrefix(v, "MariaDB")
	_, number, _ := strings.Cut(v, " ")
	for i, part := range strings.SplitN(number, ".", 3) {
		end := strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' })
		if end < 0 {
			end = len(part)
		}
		d.version[i], _ = strconv.Atoi(part[:end])
	}
	return d
}

func (d Dialect) atLeast(want ...int) bool {
	return slices.Compare(d.version[:len(want)], want) >= 0
}

func (d Dialect) canRenameColumn() bool {
	if d.mariaDB {
		return d.atLeast(10, 5, 2)
	}
	return d.atLeast(8, 0)
}

func (d Dialect) writesDefaultsAsSQL() bool {
	return d.mariaDB && d.atLeast(10, 2, 7)
}

var (
	_ dialect.Dialect   = Dialect{}
	_ dialect.Versioned = Dialect{}
)

func (Dialect) Open(c model.Connection) (*sql.DB, error) {
	conn, err := mysqldriver.NewConnector(config(c))
	if err != nil {
		return nil, err
	}
	if c.ReadOnly {
		return sql.OpenDB(initConnector{Connector: conn, init: "SET SESSION TRANSACTION READ ONLY"}), nil
	}
	return sql.OpenDB(conn), nil
}

type initConnector struct {
	driver.Connector
	init string
}

func (c initConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	ex, ok := conn.(driver.ExecerContext)
	if !ok {
		conn.Close()
		return nil, errors.New("driver cannot run the session setup statement")
	}
	if _, err := ex.ExecContext(ctx, c.init, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func config(c model.Connection) *mysqldriver.Config {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 3306
	}
	cfg := mysqldriver.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	cfg.DBName = c.Database
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.AllowNativePasswords = true
	cfg.ClientFoundRows = true
	cfg.Timeout = 10 * time.Second
	switch c.SSLMode {
	case "disable":
		cfg.TLSConfig = "false"
	case "require":
		cfg.TLSConfig = "skip-verify"
	case "verify-full":
		cfg.TLSConfig = "true"
	default:
		cfg.TLSConfig = "preferred"
	}
	return cfg
}

func (Dialect) Driver() model.Driver { return model.MySQL }

func (Dialect) DefaultPort() int { return 3306 }

func (Dialect) QuoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func (Dialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT VERSION()`).Scan(&v)
	if strings.Contains(strings.ToLower(v), "mariadb") {
		return "MariaDB " + v, err
	}
	return "MySQL " + v, err
}

func (Dialect) Syntax() sqltext.Syntax {
	return sqltext.Syntax{
		HashComments: true, DashCommentNeedsSpace: true, BackslashEscapes: true,
		DoubleQuotedStrings: true, ExecutableComments: true,
	}
}

func (Dialect) QuoteString(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, `\`, `\\`)
	return "'" + s + "'"
}

func (Dialect) TextOf(expr string) string { return "CAST(" + expr + " AS CHAR)" }

func (Dialect) LikeIgnoringCase() string { return "LIKE" }

func (Dialect) BooleanType() bool { return false }

func (Dialect) InsertDefaults(table string) string {
	return "INSERT INTO " + table + " () VALUES ()"
}

func (Dialect) ReturnsRows(stmt string) bool {
	return sqltext.ReturnsRows(stmt, "DESCRIBE", "DESC")
}

func (d Dialect) CanPage(stmt string) bool {
	return !(d.mariaDB && sqltext.ContainsKeyword(sqltext.Strip(stmt, d.Syntax()), "ORDER"))
}

var autocommit = regexp.MustCompile(`(?i)\bautocommit\s*:?=\s*(0|1|off|on|false|true)\b`)

func (d Dialect) TransactionAfter(stmt string, open bool) bool {
	if sqltext.FirstKeyword(stmt) == "SET" {
		if m := autocommit.FindStringSubmatch(sqltext.Strip(stmt, d.Syntax())); m != nil {
			return slices.Contains([]string{"0", "off", "false"}, strings.ToLower(m[1]))
		}
	}
	return dialect.StandardTransactionAfter(stmt, open)
}

func (Dialect) Classify(err error) dialect.ErrorKind {
	if errors.Is(err, mysqldriver.ErrInvalidConn) {
		return dialect.ConnLost
	}
	var myErr *mysqldriver.MySQLError
	if errors.As(err, &myErr) {
		switch myErr.Number {
		case 1205:
			return dialect.LockTimeout
		case 1064, 1149:
			return dialect.BadSyntax
		}
	}
	return dialect.Other
}

var nearAtLine = regexp.MustCompile(`(?s)near '(.*)' at line (\d+)$`)

func (Dialect) ErrorPosition(err error, stmt string) int {
	var myErr *mysqldriver.MySQLError
	if !errors.As(err, &myErr) {
		return -1
	}
	m := nearAtLine.FindStringSubmatch(myErr.Message)
	if m == nil {
		return -1
	}
	if m[1] == "" {
		return len(stmt)
	}
	line, _ := strconv.Atoi(m[2])
	start := 0
	for ; line > 1; line-- {
		nl := strings.IndexByte(stmt[start:], '\n')
		if nl < 0 {
			break
		}
		start += nl + 1
	}
	near, _, _ := strings.Cut(m[1], "\n")
	if i := strings.Index(stmt[start:], near); i >= 0 {
		return start + i
	}
	return start
}

func (Dialect) ErrorMessage(err error) string {
	var myErr *mysqldriver.MySQLError
	if !errors.As(err, &myErr) {
		return err.Error()
	}
	m := nearAtLine.FindStringSubmatch(myErr.Message)
	if m == nil {
		return myErr.Message
	}
	if m[1] == "" {
		return "syntax error at the end of the statement"
	}
	near, _, _ := strings.Cut(m[1], "\n")
	if r := []rune(near); len(r) > 40 {
		near = string(r[:40]) + "…"
	}
	return "syntax error near '" + near + "'"
}

func (Dialect) EstimateRows(ctx context.Context, db *sql.DB, schema, table string) (int64, bool, error) {
	var n sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT table_rows FROM information_schema.tables
		WHERE table_schema = ? AND table_name = ? AND table_rows IS NOT NULL`, schema, table).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !n.Valid) {
		return 0, false, nil
	}
	return n.Int64, err == nil, err
}

func (Dialect) TypeOf(t string) dialect.Type {
	t = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(t)), "unsigned ")
	if strings.HasPrefix(t, "tinyint(1)") {
		return dialect.Type{Kind: model.KindBool}
	}
	switch dialect.FirstWord(t) {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint", "decimal", "numeric", "double", "real":
		return dialect.Type{Kind: model.KindNumber}
	case "float":
		return dialect.Type{Kind: model.KindNumber, Float32: true}
	case "bool", "boolean":
		return dialect.Type{Kind: model.KindBool}
	case "date":
		return dialect.Type{Kind: model.KindDateTime, Date: true}
	case "time":
		return dialect.Type{Kind: model.KindDateTime, Time: true}
	case "datetime", "timestamp":
		return dialect.Type{Kind: model.KindDateTime}
	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob", "bit":
		return dialect.Type{Kind: model.KindBinary}
	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext":
		return dialect.Type{Kind: model.KindText}
	}
	return dialect.Type{}
}
