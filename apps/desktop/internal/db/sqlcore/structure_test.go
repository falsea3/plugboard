package sqlcore

import (
	"context"
	"errors"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlite"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func TestStructureRefusedReadOnly(t *testing.T) {
	rw := openSQLite(t)
	s, err := Open(context.Background(), "ro", model.Connection{Driver: model.SQLite, File: rw.Conn.File, ReadOnly: true}, sqlite.Dialect{}, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, _, err = s.ApplyStructure(context.Background(), model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeDelete, Column: "name"},
	}})
	var ro *db.ReadOnlyError
	if !errors.As(err, &ro) {
		t.Fatalf("err = %v", err)
	}
}
