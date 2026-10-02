package engines

import (
	"context"
	"fmt"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/mysql"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/postgres"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlcore"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlite"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func Open(ctx context.Context, id string, c model.Connection, opts db.OpenOptions) (db.Session, error) {
	d, err := Dialect(c.Driver)
	if err != nil {
		return nil, err
	}
	return sqlcore.Open(ctx, id, c, d, opts)
}

func Dialect(d model.Driver) (dialect.Dialect, error) {
	switch d {
	case model.Postgres:
		return postgres.Dialect{}, nil
	case model.MySQL:
		return mysql.Dialect{}, nil
	case model.SQLite:
		return sqlite.Dialect{}, nil
	}
	return nil, fmt.Errorf("unsupported driver %q", d)
}
