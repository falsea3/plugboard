package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

type mysqlDialect struct{}

func (mysqlDialect) Driver() model.Driver { return model.MySQL }

func (mysqlDialect) Open(c model.Connection) (*sql.DB, error) {
	conn, err := mysql.NewConnector(mysqlConfig(c))
	if err != nil {
		return nil, err
	}
	if c.ReadOnly {
		// Works on MySQL 5.6.5+ and MariaDB 10.0+, unlike the variable names,
		// which differ between them (transaction_read_only vs tx_read_only).
		return sql.OpenDB(initConnector{Connector: conn, init: "SET SESSION TRANSACTION READ ONLY"}), nil
	}
	return sql.OpenDB(conn), nil
}

// initConnector runs one statement on every new pooled connection.
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

func mysqlConfig(c model.Connection) *mysql.Config {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 3306
	}
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	cfg.DBName = c.Database
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.AllowNativePasswords = true
	// Report rows matched rather than rows changed, like PostgreSQL and SQLite,
	// so an UPDATE that sets a value to itself still counts as hitting its row.
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

func (mysqlDialect) QuoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func (mysqlDialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT VERSION()`).Scan(&v)
	if strings.Contains(strings.ToLower(v), "mariadb") {
		return "MariaDB " + v, err
	}
	return "MySQL " + v, err
}

func (mysqlDialect) DefaultSchema(ctx context.Context, db *sql.DB, c model.Connection) (string, error) {
	if c.Database != "" {
		return c.Database, nil
	}
	var s sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&s); err != nil {
		return "", err
	}
	return s.String, nil
}

func (mysqlDialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	return queryStrings(ctx, db, `
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY schema_name`)
}

func (mysqlDialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return queryTables(ctx, db, schema, `
		SELECT table_name, table_type FROM information_schema.tables
		WHERE table_schema = ?
		ORDER BY table_name`, schema)
}

func (mysqlDialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	cols, err := queryColumns(ctx, db, `
		SELECT column_name, column_type, is_nullable = 'YES', column_default, column_key = 'PRI'
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	for i := range cols {
		cols[i].Enum = parseMySQLEnum(cols[i].Type)
		// BIT values arrive as raw bytes, like BINARY.
		cols[i].Binary = cols[i].Binary || baseType(cols[i].Type) == "bit"
	}
	return cols, nil
}

// parseMySQLEnum reads the labels out of a column type like enum('a','b'),
// where a quote inside a label is written twice.
func parseMySQLEnum(t string) []string {
	if !strings.HasPrefix(strings.ToLower(t), "enum(") || !strings.HasSuffix(t, ")") {
		return nil
	}
	body := t[5 : len(t)-1]
	var out []string
	for i := 0; i < len(body); i++ {
		if body[i] != '\'' {
			continue
		}
		var b strings.Builder
		for i++; i < len(body); i++ {
			if body[i] == '\'' {
				if i+1 < len(body) && body[i+1] == '\'' {
					b.WriteByte('\'')
					i++
					continue
				}
				break
			}
			b.WriteByte(body[i])
		}
		out = append(out, b.String())
	}
	return out
}
