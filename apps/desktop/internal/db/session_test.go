package db

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func openSQLite(t *testing.T) *Session {
	t.Helper()
	ctx := context.Background()
	// URI characters in the name check the path is escaped; Windows forbids '?'.
	name := "test #1?%.db"
	if runtime.GOOS == "windows" {
		name = "test #1%.db"
	}
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(ctx, "t", model.Connection{Driver: model.SQLite, File: file}, OpenOptions{})
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

func TestSQLiteCatalog(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.DefaultSchema != "main" || len(info.Schemas) != 1 {
		t.Fatalf("info = %+v", info)
	}
	tables, err := s.Dialect.ListTables(ctx, s.DB, "main")
	if err != nil {
		t.Fatal(err)
	}
	want := []model.TableInfo{{Schema: "main", Name: "top_users", Kind: "view"}, {Schema: "main", Name: "users", Kind: "table"}}
	if len(tables) != 2 || tables[0] != want[0] || tables[1] != want[1] {
		t.Fatalf("tables = %+v", tables)
	}
	cols, err := s.Dialect.ListColumns(ctx, s.DB, "main", "users")
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 4 || !cols[0].PrimaryKey || cols[1].Nullable || cols[3].Default == nil || *cols[3].Default != "0" {
		t.Fatalf("cols = %+v", cols)
	}
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
	se, ok := err.(*StatementError)
	if !ok || se.Index != 2 {
		t.Fatalf("err = %#v, want StatementError at index 2", err)
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

func TestSQLiteDoesNotCreateMissingFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "typo.db")
	if _, err := Open(context.Background(), "t", model.Connection{Driver: model.SQLite, File: file}, OpenOptions{}); err == nil {
		t.Fatal("opening a missing file succeeded")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("missing file was created: %v", err)
	}
}

func TestSQLiteReadOnly(t *testing.T) {
	rw := openSQLite(t)
	file := rw.Conn.File
	ctx := context.Background()
	s, err := Open(ctx, "ro", model.Connection{Driver: model.SQLite, File: file, ReadOnly: true}, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// The screen refuses the whole script, so the first statement doesn't run either.
	res, err := s.Run(ctx, "select 1; delete from users")
	var ro *ReadOnlyError
	if !errors.As(err, &ro) || len(res) != 0 {
		t.Fatalf("res = %v, err = %v", res, err)
	}
	// And the server side holds even if a write slips past the screen.
	if _, err := s.DB.ExecContext(ctx, "delete from users"); err == nil {
		t.Fatal("write succeeded on a read-only SQLite connection")
	}
	if res, err := s.Run(ctx, "select count(*) from users"); err != nil || res[0].Rows[0][0] != int64(3) {
		t.Fatalf("read failed: %v %v", res, err)
	}
}

func TestTunnelEndpoints(t *testing.T) {
	c := model.Connection{Host: "203.0.113.7", SSH: model.SSHTunnel{Enabled: true, User: "root"}}
	ssh, target := TunnelEndpoints(c)
	if ssh.Host != "203.0.113.7" || target != "127.0.0.1" {
		t.Fatalf("no SSH host: ssh=%q target=%q", ssh.Host, target)
	}
	c.SSH.Host = "bastion"
	c.Host = "db.internal"
	ssh, target = TunnelEndpoints(c)
	if ssh.Host != "bastion" || target != "db.internal" {
		t.Fatalf("bastion: ssh=%q target=%q", ssh.Host, target)
	}
}
