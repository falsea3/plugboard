package dialect

import (
	"context"
	"database/sql"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqltext"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

type Dialect interface {
	Driver() model.Driver
	Open(c model.Connection) (*sql.DB, error)
	DefaultPort() int
	ServerVersion(ctx context.Context, db *sql.DB) (string, error)

	DefaultSchema(ctx context.Context, db *sql.DB, c model.Connection) (string, error)
	ListSchemas(ctx context.Context, db *sql.DB) ([]string, error)
	ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error)
	ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error)
	EstimateRows(ctx context.Context, db *sql.DB, schema, table string) (n int64, ok bool, err error)

	Syntax() sqltext.Syntax
	QuoteIdent(name string) string
	QuoteString(s string) string
	Placeholder(n int) string
	Param(n int, col model.Column) string
	TextOf(expr string) string
	LikeIgnoringCase() string
	InsertDefaults(table string) string
	CanUpdateToDefault() bool

	ReturnsRows(stmt string) bool
	CanPage(stmt string) bool
	TxStatus(conn *sql.Conn) TxStatus
	TransactionAfter(stmt string, open bool) bool
	ErrorAbortsTransaction() bool
	Classify(err error) ErrorKind

	ReadKeywords() []string
	WriteReason(keyword, body string) string

	TypeOf(dbType string) Type

	ColumnDDL(ctx context.Context, db *sql.DB, schema, table, alter string, cur model.Column, ch model.ColumnChange) ([]string, error)
	TransactionalDDL() bool
	LockWait(seconds int) (set, reset string)
}

type Versioned interface {
	ForVersion(version string) Dialect
}

type ErrorKind int

const (
	Other ErrorKind = iota
	ConnLost
	StalePlan
	LockTimeout
)

type TxStatus int

const (
	TxUnknown TxStatus = iota
	TxIdle
	TxOpen
	TxFailed
)

type Type struct {
	Binary  bool
	Float32 bool
	Date    bool
	Time    bool
	Zone    bool
}
