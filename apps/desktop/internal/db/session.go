package db

import (
	"context"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/sshtunnel"
)

type Session interface {
	Info(ctx context.Context) (model.SessionInfo, error)
	Connection() model.Connection
	ServerVersion() string
	Close() error

	Tables(ctx context.Context, schema string) ([]model.TableInfo, error)
	Columns(ctx context.Context, schema, table string) ([]model.Column, error)
	TablePage(ctx context.Context, q model.TableQuery) (model.TablePage, error)
	CountRows(ctx context.Context, q model.TableQuery, exact bool) (model.RowCount, error)

	Run(ctx context.Context, script string) ([]model.ResultSet, error)
	RunMore(ctx context.Context, stmt string, offset int) (model.ResultSet, error)
	WriteStatements(script string) []string
}

type SyntaxChecker interface {
	CheckSyntax(ctx context.Context, script string) ([]model.SyntaxProblem, error)
}

type SchemaReader interface {
	Relations(ctx context.Context, schema string) ([]model.Relation, error)
	Indexes(ctx context.Context, schema, table string) ([]model.Index, error)
	Diagram(ctx context.Context, schema string) (model.Diagram, error)
	Objects(ctx context.Context, schema string) ([]model.DBObject, error)
	DDL(ctx context.Context, obj model.DBObject) (string, error)
}

type RowEditor interface {
	ApplyChanges(ctx context.Context, cs model.ChangeSet) (int, error)
	PreviewChanges(ctx context.Context, cs model.ChangeSet) ([]string, error)
}

type StructureEditor interface {
	ApplyStructure(ctx context.Context, sc model.StructureChange) (applied int, partial bool, err error)
	PreviewStructure(ctx context.Context, sc model.StructureChange) ([]string, error)
	RenameTableSQL(schema, from, to string) string
}

var ErrNotSupported = apperr.New("not_supported", "this database doesn't support that")

type OpenOptions struct {
	KnownHostsFile string
	OnTunnelState  func(sshtunnel.State, error)
}
