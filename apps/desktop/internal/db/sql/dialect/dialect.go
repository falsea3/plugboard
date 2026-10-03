package dialect

import (
	"context"
	"database/sql"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
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
	ListRelations(ctx context.Context, db *sql.DB, schema string) ([]model.Relation, error)
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
	BooleanType() bool

	ReturnsRows(stmt string) bool
	CanPage(stmt string) bool
	TxStatus(conn *sql.Conn) TxStatus
	TransactionAfter(stmt string, open bool) bool
	ErrorAbortsTransaction() bool
	Classify(err error) ErrorKind
	ErrorPosition(err error, stmt string) int
	ErrorMessage(err error) string

	ReadKeywords() []string
	WriteReason(keyword, body string) string

	TypeOf(dbType string) Type

	ColumnDDL(ctx context.Context, db *sql.DB, schema, table, alter string, cur model.Column, ch model.ColumnChange) ([]string, error)
	TransactionalDDL() bool
	CanAlterColumns() bool
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
	BadSyntax
)

type TxStatus int

const (
	TxUnknown TxStatus = iota
	TxIdle
	TxOpen
	TxFailed
)

type Type struct {
	Kind    model.ColumnKind
	Float32 bool
	Date    bool
	Time    bool
	Zone    bool
}
