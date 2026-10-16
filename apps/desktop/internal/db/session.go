package db

import (
	"context"
	"io"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
	"github.com/falsea3/plugboard/apps/desktop/internal/sshtunnel"
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

type Dumper interface {
	Dump(ctx context.Context, opts model.DumpOptions, w io.Writer, progress func(model.DumpProgress)) error
	Import(ctx context.Context, r io.Reader, total int64, opts model.ImportOptions, progress func(model.ImportProgress)) (model.ImportResult, error)
}

type SizeReader interface {
	DatabaseSize(ctx context.Context) (int64, error)
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
	CreateIndexSQL(schema, table string, idx model.NewIndex) string
	DropIndexSQL(schema, table, name string) string
	TruncateSQL(schema, table string) string
}

type KeyStore interface {
	ScanKeys(ctx context.Context, db, pattern, cursor string, count int) (model.KeyPage, error)
	ReadKey(ctx context.Context, db, key, cursor string) (model.KeyValue, error)
	EditKey(ctx context.Context, db string, e model.KeyEdit) error
	KeyCounts(ctx context.Context) (map[string]int64, error)
}

var ErrNotSupported = apperr.New("not_supported", "this database doesn't support that")

type OpenOptions struct {
	KnownHostsFile string
	OnTunnelState  func(sshtunnel.State, error)
}

type ServerConfig interface {
	ConfigFiles(ctx context.Context) ([]model.ConfigFile, error)
	ReadConfig(ctx context.Context, path string) (model.ConfigText, error)
	WriteConfig(ctx context.Context, path, text string) (model.ConfigSaved, error)
}

type RoleManager interface {
	Roles(ctx context.Context) (model.RoleList, error)
	RoleSQL(ctx context.Context, from string, to model.RoleSpec) ([]string, error)
	SaveRole(ctx context.Context, from string, to model.RoleSpec) (string, error)
	DropRole(ctx context.Context, id string) error
}

type CellReader interface {
	ReadCell(ctx context.Context, q model.CellQuery) (string, error)
}
