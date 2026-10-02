package sqlcore

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlite"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func openSQLite(t *testing.T) *Session {
	t.Helper()
	ctx := context.Background()
	name := "test #1?%.db"
	if runtime.GOOS == "windows" {
		name = "test #1%.db"
	}
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(ctx, "t", model.Connection{Driver: model.SQLite, File: file}, sqlite.Dialect{}, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if _, err := s.Run(ctx, `
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, avatar BLOB, score REAL DEFAULT 0);
		INSERT INTO users (name, avatar, score) VALUES ('ann', x'00ff', 1.5), ('bob', NULL, 2), ('cy', NULL, 3);
		CREATE VIEW top_users AS SELECT * FROM users WHERE score > 1;
	`); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTablePagePagination(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	p, err := s.TablePage(ctx, model.TableQuery{Schema: "main", Table: "users", Limit: 2, OrderBy: "id", OrderDesc: true})
	if err != nil {
		t.Fatal(err)
	}
	if !p.HasMore || len(p.Result.Rows) != 2 || p.Result.Rows[0][1] != "cy" {
		t.Fatalf("page 1 = %+v", p)
	}
	p, err = s.TablePage(ctx, model.TableQuery{Schema: "main", Table: "users", Limit: 2, Offset: 2, OrderBy: "id", OrderDesc: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.HasMore || len(p.Result.Rows) != 1 || p.Result.Rows[0][1] != "ann" {
		t.Fatalf("page 2 = %+v", p)
	}
	if got := p.Result.Rows[0][2]; got != "0x00ff" {
		t.Fatalf("blob rendered as %v", got)
	}
}

func TestRunReportsFailingStatement(t *testing.T) {
	s := openSQLite(t)
	results, err := s.Run(context.Background(), "select count(*) from users; update users set score = 9 where id = 1; select * from nope; select 1")
	if len(results) != 2 {
		t.Fatalf("got %d results before the error, want 2", len(results))
	}
	if results[0].Rows[0][0] != int64(3) || results[1].RowsAffected != 1 || results[1].HasRows {
		t.Fatalf("results = %+v", results)
	}
	se, ok := err.(*db.StatementError)
	if !ok || se.Index != 2 {
		t.Fatalf("err = %#v, want db.StatementError at index 2", err)
	}
}

func TestEditorKeepsTransactionOnPinnedConn(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	for _, stmt := range []string{"BEGIN", "DELETE FROM users", "ROLLBACK"} {
		if _, err := s.Run(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	res, err := s.Run(ctx, "select count(*) from users")
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Rows[0][0] != int64(3) {
		t.Fatalf("rollback lost: %v rows", res[0].Rows[0][0])
	}
}

func TestEditorFollowsTransactions(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	for _, step := range []struct {
		stmt string
		inTx bool
	}{
		{"begin", true},
		{"savepoint a", true},
		{"rollback to a", true},
		{"commit", false},
		{"begin", true},
		{"rollback", false},
	} {
		if _, err := s.Run(ctx, step.stmt); err != nil {
			t.Fatalf("%s: %v", step.stmt, err)
		}
		if got := s.inTransaction(); got != step.inTx {
			t.Fatalf("after %s: in transaction = %v", step.stmt, got)
		}
	}
}
