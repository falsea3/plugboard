package sqlite_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/sqlite"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func openSQLite(t *testing.T) *sqlcore.Session {
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
	s, err := sqlcore.Open(ctx, "t", model.Connection{Driver: model.SQLite, File: file}, sqlite.Dialect{}, db.OpenOptions{})
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

func ptr[T any](v T) *T { return &v }

func columnNamed(t *testing.T, s *sqlcore.Session, schema, table, name string) model.Column {
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

func TestCatalog(t *testing.T) {
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

func TestDoesNotCreateMissingFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "typo.db")
	if _, err := sqlcore.Open(context.Background(), "t", model.Connection{Driver: model.SQLite, File: file}, sqlite.Dialect{}, db.OpenOptions{}); err == nil {
		t.Fatal("opening a missing file succeeded")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("missing file was created: %v", err)
	}
}

func TestReadOnly(t *testing.T) {
	rw := openSQLite(t)
	file := rw.Conn.File
	ctx := context.Background()
	s, err := sqlcore.Open(ctx, "ro", model.Connection{Driver: model.SQLite, File: file, ReadOnly: true}, sqlite.Dialect{}, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	res, err := s.Run(ctx, "select 1; delete from users")
	var ro *db.ReadOnlyError
	if !errors.As(err, &ro) || len(res) != 0 {
		t.Fatalf("res = %v, err = %v", res, err)
	}
	if _, err := s.DB.ExecContext(ctx, "delete from users"); err == nil {
		t.Fatal("write succeeded on a read-only SQLite connection")
	}
	if res, err := s.Run(ctx, "select count(*) from users"); err != nil || res[0].Rows[0][0] != int64(3) {
		t.Fatalf("read failed: %v %v", res, err)
	}
}

func TestStructure(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	sc := model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeInsert, Name: ptr("nick"), Type: ptr("TEXT"), DefaultSet: true, Default: ptr("'none'")},
		{Kind: model.ChangeUpdate, Column: "score", Name: ptr("points")},
	}}
	if _, _, err := s.ApplyStructure(ctx, sc); err != nil {
		t.Fatal(err)
	}
	if c := columnNamed(t, s, "main", "users", "nick"); c.Default == nil || *c.Default != "'none'" {
		t.Fatalf("added column = %+v", c)
	}
	columnNamed(t, s, "main", "users", "points")

	_, _, err := s.ApplyStructure(ctx, model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "name", Type: ptr("BLOB")},
	}})
	if err == nil || !strings.Contains(err.Error(), "recreating the table") {
		t.Fatalf("type change: %v", err)
	}

	_, err = s.PreviewStructure(ctx, model.StructureChange{Schema: "main", Table: "users", Changes: []model.ColumnChange{
		{Kind: model.ChangeUpdate, Column: "name", Type: ptr(" ")},
	}})
	if err == nil || !strings.Contains(err.Error(), "empty type") {
		t.Fatalf("empty type: %v", err)
	}
}

func TestFloatIsDouble(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	res, err := s.Run(ctx, "create table f (x FLOAT, y REAL); insert into f values (3.14159265358979, 3.14159265358979); select x, y from f")
	if err != nil {
		t.Fatal(err)
	}
	if row := res[2].Rows[0]; row[0] != 3.14159265358979 || row[1] != 3.14159265358979 {
		t.Fatalf("row = %#v", row)
	}
}

func TestCheckSyntax(t *testing.T) {
	s := openSQLite(t)
	problems, err := s.CheckSyntax(context.Background(), "select 1; selec 2; select * from users where; select * from nope")
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 2 || problems[0].Index != 1 || problems[0].Position != 0 || problems[1].Index != 2 || problems[1].Position != len("select * from users where") {
		t.Fatalf("problems = %+v", problems)
	}
}

func TestDiagramReadsForeignKeys(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	if _, err := s.Run(ctx, `
		CREATE TABLE teams (code TEXT, season INTEGER, PRIMARY KEY (code, season));
		CREATE TABLE players (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users ON DELETE SET NULL,
			team_code TEXT, team_season INTEGER, FOREIGN KEY (team_code, team_season) REFERENCES teams);
	`); err != nil {
		t.Fatal(err)
	}
	d, err := s.Diagram(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	var toUsers, toTeams *model.Relation
	for i, r := range d.Relations {
		switch r.RefTable {
		case "users":
			toUsers = &d.Relations[i]
		case "teams":
			toTeams = &d.Relations[i]
		}
	}
	if toUsers == nil || toUsers.Table != "players" || strings.Join(toUsers.Columns, ",") != "user_id" || strings.Join(toUsers.RefColumns, ",") != "id" || toUsers.OnDelete != "SET NULL" {
		t.Errorf("players → users = %+v", toUsers)
	}
	if toTeams == nil || strings.Join(toTeams.Columns, ",") != "team_code,team_season" || strings.Join(toTeams.RefColumns, ",") != "code,season" || toTeams.OnDelete != "NO ACTION" {
		t.Errorf("players → teams = %+v", toTeams)
	}
	if len(d.Tables) != 4 {
		t.Errorf("tables = %d, want users, top_users, teams, players", len(d.Tables))
	}
}

func TestIndexesListKeysAndUniqueIndexes(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	if _, err := s.Run(ctx, `
		CREATE TABLE seats (row TEXT, num INTEGER, label TEXT, PRIMARY KEY (row, num));
		CREATE UNIQUE INDEX seats_label ON seats (label);
	`); err != nil {
		t.Fatal(err)
	}
	got, err := s.Indexes(ctx, "main", "seats")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].Primary || strings.Join(got[0].Columns, ",") != "row,num" {
		t.Fatalf("indexes = %+v", got)
	}
	if !got[1].Unique || got[1].Name != "seats_label" || !strings.Contains(got[1].Definition, "CREATE UNIQUE INDEX") {
		t.Errorf("unique index = %+v", got[1])
	}
	if ix, _ := s.Indexes(ctx, "main", "users"); len(ix) != 0 {
		t.Errorf("a rowid primary key shows as an index: %+v", ix)
	}
}
