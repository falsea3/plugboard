package postgres

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/sqltext"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

type Dialect struct{ dialect.Standard }

var _ dialect.Dialect = Dialect{}

func (Dialect) Driver() model.Driver { return model.Postgres }

func (Dialect) Open(c model.Connection) (*sql.DB, error) {
	return sql.Open("pgx", dsn(c))
}

func dsn(c model.Connection) string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	database := c.Database
	if database == "" {
		database = "postgres"
	}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}
	u := url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(host, strconv.Itoa(port)),
		Path:     "/" + database,
		RawQuery: params(c, sslMode).Encode(),
	}
	if c.User != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	return u.String()
}

func params(c model.Connection, sslMode string) url.Values {
	q := url.Values{"sslmode": {sslMode}, "application_name": {"Relay DB"}}
	if c.ReadOnly {
		q.Set("default_transaction_read_only", "on")
	}
	return q
}

func (Dialect) DefaultPort() int { return 5432 }

func (Dialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SHOW server_version`).Scan(&v)
	return "PostgreSQL " + v, err
}

func (Dialect) Syntax() sqltext.Syntax {
	return sqltext.Syntax{EscapeStrings: true, DollarQuotes: true}
}

func (Dialect) Placeholder(n int) string { return "$" + strconv.Itoa(n) }

func (Dialect) Param(n int, col model.Column) string {
	return "CAST($" + strconv.Itoa(n) + "::text AS " + col.CastType + ")"
}

func (Dialect) TextOf(expr string) string { return "CAST(" + expr + " AS TEXT)" }

func (Dialect) LikeIgnoringCase() string { return "ILIKE" }

func (Dialect) BooleanType() bool { return true }

func (Dialect) TxStatus(conn *sql.Conn) dialect.TxStatus {
	var status byte
	conn.Raw(func(dc any) error {
		if c, ok := dc.(*stdlib.Conn); ok {
			status = c.Conn().PgConn().TxStatus()
		}
		return nil
	})
	switch status {
	case 'I':
		return dialect.TxIdle
	case 'T':
		return dialect.TxOpen
	case 'E':
		return dialect.TxFailed
	}
	return dialect.TxUnknown
}

func (Dialect) ErrorAbortsTransaction() bool { return true }

func (Dialect) Classify(err error) dialect.ErrorKind {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return dialect.Other
	}
	switch {
	case pgErr.Code == "42601":
		return dialect.BadSyntax
	case pgErr.Code == "55P03":
		return dialect.LockTimeout
	case pgErr.Code == "0A000" && strings.Contains(pgErr.Message, "cached plan"):
		return dialect.StalePlan
	}
	return dialect.Other
}

func (Dialect) ErrorPosition(err error, stmt string) int {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Position <= 0 {
		return -1
	}
	return dialect.ByteOffset(stmt, int(pgErr.Position)-1)
}

func (Dialect) ErrorMessage(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Message
	}
	return err.Error()
}

func (Dialect) EstimateRows(ctx context.Context, db *sql.DB, schema, table string) (int64, bool, error) {
	var n sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT c.reltuples::bigint FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2 AND c.reltuples >= 0`, schema, table).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !n.Valid) {
		return 0, false, nil
	}
	return n.Int64, err == nil, err
}

func (Dialect) TypeOf(t string) dialect.Type {
	t = strings.ToLower(strings.TrimSpace(t))
	if strings.HasSuffix(t, "[]") || strings.HasPrefix(t, "_") {
		return dialect.Type{}
	}
	switch w := dialect.FirstWord(t); w {
	case "int2", "int4", "int8", "smallint", "integer", "bigint", "serial", "smallserial", "bigserial",
		"numeric", "decimal", "double", "float8", "money", "oid":
		return dialect.Type{Kind: model.KindNumber}
	case "real", "float4":
		return dialect.Type{Kind: model.KindNumber, Float32: true}
	case "bool", "boolean":
		return dialect.Type{Kind: model.KindBool}
	case "date":
		return dialect.Type{Kind: model.KindDateTime, Date: true}
	case "time", "timetz":
		return dialect.Type{Kind: model.KindDateTime, Time: true}
	case "timestamp", "timestamptz":
		return dialect.Type{Kind: model.KindDateTime, Zone: w == "timestamptz" || strings.Contains(t, "with time zone")}
	case "bytea":
		return dialect.Type{Kind: model.KindBinary}
	}
	if strings.Contains(t, "char") || strings.Contains(t, "text") {
		return dialect.Type{Kind: model.KindText}
	}
	return dialect.Type{}
}
