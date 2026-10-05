package sqltest

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Server struct {
	Env       string
	Dialect   dialect.Dialect
	Schema    string
	True      any
	KeyColumn string
}

func Run(t *testing.T, srv Server) {
	c := Conn(t, srv.Env, srv.Dialect.Driver())
	t.Run("ReadOnly", func(t *testing.T) { readOnly(t, srv, c) })
	t.Run("ApplyChanges", func(t *testing.T) { applyChanges(t, srv, c) })
	t.Run("TableFilters", func(t *testing.T) { tableFilters(t, srv, c) })
	t.Run("EditorPaging", func(t *testing.T) { editorPaging(t, srv, c) })
	t.Run("Structure", func(t *testing.T) { structure(t, srv, c) })
	t.Run("TableBusy", func(t *testing.T) { tableBusy(t, srv, c) })
	t.Run("ColumnKinds", func(t *testing.T) { columnKinds(t, srv, c) })
	t.Run("SyntaxErrors", func(t *testing.T) { syntaxErrors(t, srv, c) })
	t.Run("Relations", func(t *testing.T) { relations(t, srv, c) })
	t.Run("Indexes", func(t *testing.T) { indexes(t, srv, c) })
	t.Run("Objects", func(t *testing.T) { objects(t, srv, c) })
	t.Run("RenameTable", func(t *testing.T) { renameTable(t, srv, c) })
	t.Run("IndexSQL", func(t *testing.T) { indexSQL(t, srv, c) })
}

func FindRelation(d model.Diagram, table string) (model.Relation, bool) {
	for _, r := range d.Relations {
		if r.Table == table {
			return r, true
		}
	}
	return model.Relation{}, false
}

func Conn(t *testing.T, env string, driver model.Driver) model.Connection {
	t.Helper()
	addr := os.Getenv(env)
	if addr == "" {
		t.Skipf("%s not set", env)
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	return model.Connection{Driver: driver, Host: addr[:i], Port: port, User: "relay", Password: "relay", Database: "shop", SSLMode: "disable"}
}

func Open(t *testing.T, d dialect.Dialect, c model.Connection) *sqlcore.Session {
	t.Helper()
	s, err := sqlcore.Open(context.Background(), "test", c, d, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func Ptr[T any](v T) *T { return &v }

func Column(t *testing.T, s *sqlcore.Session, schema, table, name string) model.Column {
	t.Helper()
	cols, err := s.Columns(context.Background(), schema, table)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cols {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no column %s in %v", name, cols)
	return model.Column{}
}

func HasTable(tables []model.TableInfo, name, kind string) bool {
	for _, t := range tables {
		if t.Name == name && t.Kind == kind {
			return true
		}
	}
	return false
}
