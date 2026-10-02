package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqltext"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	_ "modernc.org/sqlite"
)

type Dialect struct{ dialect.Standard }

var _ dialect.Dialect = Dialect{}

func (Dialect) Driver() model.Driver { return model.SQLite }

func (Dialect) Open(c model.Connection) (*sql.DB, error) {
	if strings.TrimSpace(c.File) == "" {
		return nil, errors.New("choose a SQLite database file")
	}
	q := url.Values{}
	mode := "rw"
	if c.ReadOnly {
		mode = "ro"
	}
	q.Add("mode", mode)
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	return sql.Open("sqlite", "file:"+uriPath.Replace(c.File)+"?"+q.Encode())
}

var uriPath = strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23")

func (Dialect) DefaultPort() int { return 0 }

func (Dialect) ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&v)
	return "SQLite " + v, err
}

func (Dialect) Syntax() sqltext.Syntax { return sqltext.Syntax{} }

func (Dialect) TextOf(expr string) string { return "CAST(" + expr + " AS TEXT)" }

func (Dialect) LikeIgnoringCase() string { return "LIKE" }

func (Dialect) CanUpdateToDefault() bool { return false }

func (Dialect) ReturnsRows(stmt string) bool { return sqltext.ReturnsRows(stmt, "PRAGMA") }

func (Dialect) TypeOf(t string) dialect.Type {
	return dialect.Type{
		Binary: t == "blob" || t == "binary" || t == "varbinary",
		Date:   t == "date",
		Time:   t == "time",
		Zone:   strings.HasSuffix(t, "tz"),
	}
}
