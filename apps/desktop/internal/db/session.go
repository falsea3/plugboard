package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	"github.com/relay-client/relay-db/apps/desktop/internal/sshtunnel"
)

// MaxResultRows caps how many rows a single editor statement brings back.
const MaxResultRows = 5000

// Session is one open connection. Catalog reads use the pool; statements typed
// into the SQL editor run on a single pinned connection so BEGIN/SET/temp
// tables behave the way the user expects.
type Session struct {
	ID      string
	Conn    model.Connection
	Dialect Dialect
	DB      *sql.DB
	// Version is how the server describes itself, e.g. "PostgreSQL 17.2".
	Version string
	tunnel  io.Closer

	editorMu sync.Mutex
	editor   *sql.Conn
	// editorTx follows BEGIN … COMMIT on the editor connection for engines
	// whose driver doesn't report it (MySQL, SQLite).
	editorTx bool
}

// OpenOptions are environment details a session needs but a profile doesn't carry.
type OpenOptions struct {
	KnownHostsFile string
	// OnTunnelState hears about SSH drops and reconnects for this session.
	OnTunnelState func(sshtunnel.State, error)
}

func Open(ctx context.Context, id string, c model.Connection, opts OpenOptions) (*Session, error) {
	d, err := DialectFor(c.Driver)
	if err != nil {
		return nil, err
	}
	dial := c
	var tunnel *sshtunnel.Tunnel
	if c.SSH.Enabled && c.Driver != model.SQLite {
		port := c.Port
		if port == 0 {
			port = defaultPort(c.Driver)
		}
		ssh, target := TunnelEndpoints(c)
		tunnel, err = sshtunnel.Open(ctx, ssh, target, port, sshtunnel.Options{KnownHostsFile: opts.KnownHostsFile, OnState: opts.OnTunnelState})
		if err != nil {
			return nil, err
		}
		// The driver talks to the local end of the tunnel; the SSH server
		// connects onwards to the real host and port.
		dial.Host = "127.0.0.1"
		dial.Port = tunnel.LocalPort()
	}
	closeTunnel := func() {
		if tunnel != nil {
			tunnel.Close()
		}
	}
	pool, err := d.Open(dial)
	if err != nil {
		closeTunnel()
		return nil, err
	}
	pool.SetMaxOpenConns(4)
	pool.SetConnMaxIdleTime(5 * time.Minute)
	version, err := d.ServerVersion(ctx, pool)
	if err != nil {
		pool.Close()
		closeTunnel()
		return nil, err
	}
	s := &Session{ID: id, Conn: c, Dialect: d, DB: pool, Version: version}
	if tunnel != nil {
		s.tunnel = tunnel
	}
	return s, nil
}

// TunnelEndpoints resolves where to SSH and where to forward to. With no SSH
// host the database server itself is the SSH server — the common "database
// port is firewalled, SSH is open" setup — so the tunnel dials the database
// host over SSH and forwards to its loopback.
func TunnelEndpoints(c model.Connection) (ssh model.SSHTunnel, targetHost string) {
	ssh = c.SSH
	ssh.Host = c.SSHHost()
	if strings.TrimSpace(c.SSH.Host) == "" {
		return ssh, "127.0.0.1"
	}
	return ssh, c.Host
}

// Columns lists a table's columns, retrying once if a pooled connection died.
func (s *Session) Columns(ctx context.Context, schema, table string) ([]model.Column, error) {
	return retry(ctx, func() ([]model.Column, error) { return s.Dialect.ListColumns(ctx, s.DB, schema, table) })
}

// Tables lists a schema's tables and views, retrying once if a pooled connection died.
func (s *Session) Tables(ctx context.Context, schema string) ([]model.TableInfo, error) {
	return retry(ctx, func() ([]model.TableInfo, error) { return s.Dialect.ListTables(ctx, s.DB, schema) })
}

func defaultPort(d model.Driver) int {
	switch d {
	case model.Postgres:
		return 5432
	case model.MySQL:
		return 3306
	}
	return 0
}

func (s *Session) Close() error {
	s.editorMu.Lock()
	if s.editor != nil {
		s.editor.Close()
		s.editor = nil
	}
	s.editorMu.Unlock()
	err := s.DB.Close()
	if s.tunnel != nil {
		s.tunnel.Close()
	}
	return err
}

func (s *Session) Info(ctx context.Context) (model.SessionInfo, error) {
	schemas, err := s.Dialect.ListSchemas(ctx, s.DB)
	if err != nil {
		return model.SessionInfo{}, err
	}
	def, err := s.Dialect.DefaultSchema(ctx, s.DB, s.Conn)
	if err != nil {
		return model.SessionInfo{}, err
	}
	if def == "" && len(schemas) > 0 {
		def = schemas[0]
	}
	return model.SessionInfo{
		SessionID:     s.ID,
		Connection:    s.Conn.WithoutSecrets(),
		ServerVersion: s.Version,
		Schemas:       schemas,
		DefaultSchema: def,
	}, nil
}

// Run executes every statement in script on the pinned editor connection and
// returns one result per statement. It stops at the first error.
func (s *Session) Run(ctx context.Context, script string) ([]model.ResultSet, error) {
	stmts := SplitStatements(script, s.Conn.Driver == model.MySQL)
	if len(stmts) == 0 {
		return nil, errors.New("nothing to run")
	}
	if s.Conn.ReadOnly {
		// Refuse the whole script up front, so a write in statement 3 doesn't
		// leave statements 1–2 half-applied.
		for i, stmt := range stmts {
			if reason := ReadOnlyReason(stmt, s.Conn.Driver); reason != "" {
				return nil, &StatementError{Statement: stmt, Index: i, Err: &ReadOnlyError{Reason: reason}}
			}
		}
	}
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	results := make([]model.ResultSet, 0, len(stmts))
	for i, stmt := range stmts {
		rs, rolledBack, err := s.onEditor(ctx, stmt, func() (model.ResultSet, error) { return s.runOne(ctx, stmt) })
		if err != nil {
			return results, &StatementError{Statement: stmt, Index: i, Err: err, RolledBack: rolledBack}
		}
		s.noteTransaction(stmt)
		results = append(results, rs)
	}
	return results, nil
}

// onEditor runs stmt through run on the editor connection. If the connection
// dies (server restart, SSH drop, idle timeout) under a read with no
// transaction open, nothing is lost and run goes once more on a fresh
// connection. Otherwise the loss is reported: the statement may have been
// applied, and rolledBack says an open transaction went with the connection.
func (s *Session) onEditor(ctx context.Context, stmt string, run func() (model.ResultSet, error)) (rs model.ResultSet, rolledBack bool, err error) {
	if err := s.openEditor(ctx); err != nil {
		return model.ResultSet{}, false, err
	}
	inTx := s.inTransaction()
	rs, err = run()
	if err != nil && isConnLost(err) && ctx.Err() == nil && !inTx && !mayWrite(stmt, s.Conn.Driver) {
		s.dropEditor()
		if err = s.openEditor(ctx); err == nil {
			rs, err = run()
		}
	}
	if err == nil || (ctx.Err() == nil && !isConnLost(err)) {
		return rs, false, err
	}
	// Cancelled or disconnected: the connection is closed, and with it any
	// open transaction and session settings. The next run starts afresh.
	s.dropEditor()
	if ctx.Err() == nil {
		err = ErrConnLost
	}
	return model.ResultSet{}, inTx, err
}

func (s *Session) openEditor(ctx context.Context) error {
	if s.editor != nil {
		return nil
	}
	c, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	s.editor = c
	return nil
}

func (s *Session) dropEditor() {
	if s.editor != nil {
		s.editor.Raw(func(any) error { return driver.ErrBadConn }) // evict from the pool
		s.editor.Close()
		s.editor = nil
	}
	s.editorTx = false
}

// inTransaction reports whether the editor connection has a transaction
// open, which losing the connection would roll back.
func (s *Session) inTransaction() bool {
	if s.Conn.Driver == model.Postgres {
		st := pgTxStatus(s.editor)
		return st == 'T' || st == 'E'
	}
	return s.editorTx
}

var autocommit = regexp.MustCompile(`(?i)\bautocommit\s*:?=\s*(0|1|off|on|false|true)\b`)

// noteTransaction follows transactions opened and closed by stmt, for the
// engines where inTransaction can't ask the driver.
func (s *Session) noteTransaction(stmt string) {
	switch firstKeyword(stmt) {
	case "BEGIN":
		s.editorTx = true
	case "START":
		s.editorTx = s.editorTx || containsKeyword(stmt, "TRANSACTION")
	case "COMMIT", "END":
		s.editorTx = false
	case "ROLLBACK":
		s.editorTx = s.editorTx && containsKeyword(stmt, "TO") // ROLLBACK TO SAVEPOINT keeps it open
	case "SET":
		// MySQL with autocommit off is always inside a transaction.
		if m := autocommit.FindStringSubmatch(stripStringsAndComments(stmt, true)); m != nil {
			switch strings.ToLower(m[1]) {
			case "0", "off", "false":
				s.editorTx = true
			default:
				s.editorTx = false
			}
		}
	}
}

func (s *Session) runOne(ctx context.Context, stmt string) (model.ResultSet, error) {
	if s.pageable(stmt) && !(s.Conn.Driver == model.Postgres && pgTxStatus(s.editor) == 'E') {
		rs, err := s.runPage(ctx, stmt, 0)
		if err == nil || isConnLost(err) || ctx.Err() != nil {
			return rs, err
		}
		// The server rejected the wrapped form (e.g. duplicate column names
		// in MySQL): run the statement as typed instead. It is a plain read,
		// so running it a second time is harmless.
	}
	start := time.Now()
	if returnsRows(stmt) {
		rows, err := s.editor.QueryContext(ctx, stmt)
		if err != nil {
			return model.ResultSet{}, err
		}
		defer rows.Close()
		rs, err := readRows(rows, MaxResultRows)
		if err != nil {
			return model.ResultSet{}, err
		}
		rs.Statement = stmt
		rs.DurationMs = msSince(start)
		return rs, nil
	}
	res, err := s.editor.ExecContext(ctx, stmt)
	if err != nil {
		return model.ResultSet{}, err
	}
	affected, _ := res.RowsAffected()
	return model.ResultSet{
		Statement:    stmt,
		Columns:      []model.ResultColumn{},
		Rows:         [][]any{},
		RowsAffected: affected,
		DurationMs:   msSince(start),
	}, nil
}

type StatementError struct {
	Statement string
	Index     int
	Err       error
	// RolledBack says the editor connection was closed with a transaction open.
	RolledBack bool
}

func (e *StatementError) Error() string {
	msg := e.Err.Error()
	if e.Index > 0 {
		msg = fmt.Sprintf("statement %d: %s", e.Index+1, msg)
	}
	if e.RolledBack {
		msg += " (the open transaction was rolled back)"
	}
	return msg
}

func (e *StatementError) Unwrap() error { return e.Err }

func readRows(rows *sql.Rows, maxRows int) (model.ResultSet, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return model.ResultSet{}, err
	}
	cols := make([]model.ResultColumn, len(types))
	for i, t := range types {
		cols[i] = model.ResultColumn{Name: t.Name(), Type: strings.ToLower(t.DatabaseTypeName())}
	}
	rs := model.ResultSet{Columns: cols, Rows: [][]any{}, HasRows: true}
	for rows.Next() {
		if len(rs.Rows) >= maxRows {
			rs.Truncated = true
			break
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return model.ResultSet{}, err
		}
		for i, v := range vals {
			vals[i] = normalizeValue(v, cols[i].Type)
		}
		rs.Rows = append(rs.Rows, vals)
	}
	return rs, rows.Err()
}

func msSince(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000
}
