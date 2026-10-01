package db

import (
	"context"
	"database/sql"
	"net"
	"net/url"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

type postgresDialect struct{}

func (postgresDialect) Driver() model.Driver { return model.Postgres }

func (postgresDialect) Open(c model.Connection) (*sql.DB, error) {
	return sql.Open("pgx", postgresDSN(c))
}

func postgresDSN(c model.Connection) string {
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
		RawQuery: postgresParams(c, sslMode).Encode(),
	}
	if c.User != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	return u.String()
}

func postgresParams(c model.Connection, sslMode string) url.Values {
	q := url.Values{"sslmode": {sslMode}, "application_name": {"Relay DB"}}
	if c.ReadOnly {
		// Sent in the startup packet, so every pooled connection starts read-only.
		q.Set("default_transaction_read_only", "on")
	}
	return q
}

func (postgresDialect) QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (postgresDialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SHOW server_version`).Scan(&v)
	return "PostgreSQL " + v, err
}

func (postgresDialect) DefaultSchema(ctx context.Context, db *sql.DB, _ model.Connection) (string, error) {
	var s sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&s); err != nil {
		return "", err
	}
	if !s.Valid {
		return "public", nil
	}
	return s.String, nil
}

func (postgresDialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	return queryStrings(ctx, db, `
		SELECT nspname FROM pg_namespace
		WHERE nspname NOT IN ('pg_catalog', 'information_schema')
		  AND nspname NOT LIKE 'pg_toast%'
		  AND nspname NOT LIKE 'pg_temp_%'
		ORDER BY nspname`)
}

func (postgresDialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return queryTables(ctx, db, schema, `
		SELECT c.relname,
		       CASE c.relkind WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' ELSE 'BASE TABLE' END
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
		ORDER BY c.relname`, schema)
}

// ListColumns also returns each column's type by its catalog name
// ("pg_catalog"."varchar", "public"."mood"), which carries no length or
// precision. format_type(oid, NULL) would not do: it says "character",
// and CAST(… AS character) means char(1).
func (postgresDialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	return queryColumns(ctx, db, `
		SELECT a.attname,
		       format_type(a.atttypid, a.atttypmod),
		       NOT a.attnotnull,
		       pg_get_expr(d.adbin, d.adrelid),
		       COALESCE(a.attnum = ANY(i.indkey), false),
		       (SELECT json_agg(e.enumlabel ORDER BY e.enumsortorder)::text FROM pg_enum e WHERE e.enumtypid = a.atttypid),
		       quote_ident(tn.nspname) || '.' || quote_ident(t.typname)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_type t ON t.oid = a.atttypid
		JOIN pg_namespace tn ON tn.oid = t.typnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		LEFT JOIN pg_index i ON i.indrelid = c.oid AND i.indisprimary
		WHERE n.nspname = $1 AND c.relname = $2 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, schema, table)
}

// pgParam renders bound parameter n for a value of col. Values travel as
// text (pgx won't encode a Go string into, say, an int4 parameter) and the
// server casts them to the column's type.
func pgParam(n int, col model.Column) string {
	return "CAST($" + strconv.Itoa(n) + "::text AS " + col.CastType + ")"
}
