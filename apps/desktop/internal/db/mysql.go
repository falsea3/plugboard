package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// mysqlDialect also serves MariaDB. Open fills in the server (forVersion):
// the zero value is a MySQL of unknown version.
type mysqlDialect struct {
	mariaDB bool
	version [3]int
}

func (mysqlDialect) Driver() model.Driver { return model.MySQL }

// forVersion is the dialect for the server ServerVersion described as v
// ("MySQL 8.0.36-0ubuntu…", "MariaDB 10.11.6-MariaDB-…").
func (d mysqlDialect) forVersion(v string) Dialect {
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

func (d mysqlDialect) atLeast(want ...int) bool {
	return slices.Compare(d.version[:len(want)], want) >= 0
}

// canRenameColumn: RENAME COLUMN came in MySQL 8.0 and MariaDB 10.5.2.
func (d mysqlDialect) canRenameColumn() bool {
	if d.mariaDB {
		return d.atLeast(10, 5, 2)
	}
	return d.atLeast(8, 0)
}

// writesDefaultsAsSQL: MariaDB 10.2.7+ gives column_default as SQL ('pro',
// current_timestamp(), NULL); MySQL gives it bare (pro).
func (d mysqlDialect) writesDefaultsAsSQL() bool {
	return d.mariaDB && d.atLeast(10, 2, 7)
}

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

// ListColumns shows each default as the SQL that recreates it ('pro', not
// the bare pro information_schema gives), the same as PostgreSQL does, so a
// default can be edited and written back as it reads.
func (d mysqlDialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name, column_type, data_type, is_nullable = 'YES', column_default, extra, column_key = 'PRI'
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Column{}
	for rows.Next() {
		var c model.Column
		var dataType, extra string
		var def sql.NullString
		if err := rows.Scan(&c.Name, &c.Type, &dataType, &c.Nullable, &def, &extra, &c.PrimaryKey); err != nil {
			return nil, err
		}
		c.Default = d.defaultSQL(def, dataType, extra)
		c.Enum = parseMySQLEnum(c.Type)
		// BIT values arrive as raw bytes, like BINARY.
		c.Binary = isBinaryType(baseType(c.Type)) || baseType(c.Type) == "bit"
		out = append(out, c)
	}
	return out, rows.Err()
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

func (mysqlDialect) TransactionalDDL() bool { return false }

// LockWait bounds the wait for the metadata lock ALTER TABLE needs; the
// server default is a year, with every query on the table queued behind it.
func (mysqlDialect) LockWait(seconds int) (set, reset string) {
	return fmt.Sprintf("SET SESSION lock_wait_timeout = %d", seconds), "SET SESSION lock_wait_timeout = DEFAULT"
}

// mysqlColumn is what MODIFY/CHANGE must say again about a column, since
// MySQL replaces its whole definition: anything left out is lost.
type mysqlColumn struct {
	columnType, dataType, nullable, collation, comment, extra string
	def                                                       sql.NullString
}

func (d mysqlDialect) ColumnDDL(ctx context.Context, db *sql.DB, schema, table, alter string, cur model.Column, ch model.ColumnChange) ([]string, error) {
	name := nameOf(cur, ch)
	onlyRename := !typeChanged(cur, ch) && !ch.DefaultSet && (ch.Nullable == nil || *ch.Nullable == cur.Nullable)
	if onlyRename && name == cur.Name {
		return nil, nil
	}
	if onlyRename && d.canRenameColumn() {
		// Keeps the definition as it is; older servers restate it below.
		return []string{alter + "RENAME COLUMN " + d.QuoteIdent(cur.Name) + " TO " + d.QuoteIdent(name)}, nil
	}

	var m mysqlColumn
	err := db.QueryRowContext(ctx, `
		SELECT column_type, data_type, is_nullable, COALESCE(collation_name, ''), column_comment, extra, column_default
		FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?`,
		schema, table, cur.Name).Scan(&m.columnType, &m.dataType, &m.nullable, &m.collation, &m.comment, &m.extra, &m.def)
	if err != nil {
		return nil, fmt.Errorf("read column %s: %w", cur.Name, err)
	}
	if strings.Contains(strings.ToUpper(m.extra), "GENERATED") && !strings.Contains(strings.ToUpper(m.extra), "DEFAULT_GENERATED") {
		return nil, fmt.Errorf("%s is a generated column: change it in the SQL editor", cur.Name)
	}

	typ := m.columnType
	if typeChanged(cur, ch) {
		typ = strings.TrimSpace(*ch.Type)
	}
	def := d.QuoteIdent(name) + " " + typ
	// Kept across a type change too (varchar(10) → varchar(20) mustn't turn a
	// _bin column case-insensitive), unless the new type names its own.
	if m.collation != "" && mysqlText[baseType(typ)] && !namesCollation(typ) {
		def += " COLLATE " + m.collation
	}
	nullable := m.nullable == "YES"
	if ch.Nullable != nil {
		nullable = *ch.Nullable
	}
	if nullable {
		def += " NULL"
	} else {
		def += " NOT NULL"
	}
	switch {
	case ch.DefaultSet && ch.Default != nil:
		def += " DEFAULT " + *ch.Default
	case !ch.DefaultSet:
		if expr := d.defaultSQL(m.def, m.dataType, m.extra); expr != nil {
			def += " DEFAULT " + *expr
		}
	}
	// extra reads "DEFAULT_GENERATED on update CURRENT_TIMESTAMP(3) INVISIBLE"
	// on MySQL, "on update current_timestamp(), INVISIBLE" on MariaDB.
	extra := strings.ToLower(m.extra)
	if strings.Contains(extra, "auto_increment") {
		def += " AUTO_INCREMENT"
	}
	if i := strings.Index(extra, "on update "); i >= 0 {
		expr, _, _ := strings.Cut(m.extra[i+len("on update "):], " ")
		def += " ON UPDATE " + strings.TrimSuffix(expr, ",")
	}
	if strings.Contains(extra, "invisible") {
		def += " INVISIBLE"
	}
	if m.comment != "" {
		def += " COMMENT " + quoteLiteral(model.MySQL, m.comment)
	}
	return []string{alter + "CHANGE COLUMN " + d.QuoteIdent(cur.Name) + " " + def}, nil
}

// namesCollation reports a column type that says its own collation or charset.
func namesCollation(typ string) bool {
	upper := strings.ToUpper(stripStringsAndComments(typ, true))
	return strings.Contains(upper, "COLLATE") || strings.Contains(upper, "CHARACTER SET") || strings.Contains(upper, "CHARSET")
}

var mysqlNumeric = map[string]bool{
	"tinyint": true, "smallint": true, "mediumint": true, "int": true, "integer": true, "bigint": true,
	"decimal": true, "numeric": true, "float": true, "double": true, "real": true, "bit": true, "year": true,
}

// mysqlText are the types a collation applies to.
var mysqlText = map[string]bool{
	"char": true, "varchar": true, "tinytext": true, "text": true, "mediumtext": true, "longtext": true, "enum": true, "set": true,
}

// defaultSQL turns information_schema's column_default into the SQL that
// recreates it. MySQL gives it bare — pro, 1.50, 0x6162 for a binary 'ab' —
// and an expression with its quotes backslash-escaped: concat(_utf8mb4\'a\').
// MariaDB 10.2.7+ gives SQL already, and NULL for DEFAULT NULL.
func (d mysqlDialect) defaultSQL(raw sql.NullString, dataType, extra string) *string {
	if !raw.Valid {
		return nil
	}
	v := raw.String
	if d.writesDefaultsAsSQL() {
		if v == "NULL" {
			return nil
		}
		return &v
	}
	dataType = strings.ToLower(dataType)
	upper := strings.ToUpper(v)
	now := strings.HasPrefix(upper, "CURRENT_TIMESTAMP") || strings.HasPrefix(upper, "NOW(") || strings.HasPrefix(upper, "LOCALTIME")
	switch {
	case strings.Contains(strings.ToUpper(extra), "DEFAULT_GENERATED"):
		if !now {
			v = "(" + unescapeMySQL(v) + ")"
		}
	case now && (dataType == "timestamp" || dataType == "datetime"): // before MySQL 8.0.13 it isn't marked DEFAULT_GENERATED
	case mysqlNumeric[dataType]:
	case (dataType == "binary" || dataType == "varbinary") && strings.HasPrefix(v, "0x"): // a hex literal
	default:
		v = quoteLiteral(model.MySQL, v)
	}
	return &v
}

// unescapeMySQL undoes the escaping information_schema puts on an expression
// default (\' for a quote, \\ for a backslash), giving it as SHOW CREATE TABLE does.
func unescapeMySQL(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '\'' || s[i+1] == '\\') {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
