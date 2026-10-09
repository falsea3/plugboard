package engines

import (
	"context"
	"fmt"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/cassandra"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/clickhouse"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/redis"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/sqlite"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var (
	_ db.Session  = (*redis.Session)(nil)
	_ db.KeyStore = (*redis.Session)(nil)

	_ db.Session      = (*cassandra.Session)(nil)
	_ db.RowEditor    = (*cassandra.Session)(nil)
	_ db.SchemaReader = (*cassandra.Session)(nil)
)

func Open(ctx context.Context, id string, c model.Connection, opts db.OpenOptions) (db.Session, error) {
	switch c.Driver {
	case model.Redis:
		return redis.Open(ctx, id, c, opts)
	case model.Cassandra:
		return cassandra.Open(ctx, id, c, opts)
	}
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
	case model.ClickHouse:
		return clickhouse.Dialect{}, nil
	}
	return nil, fmt.Errorf("unsupported driver %q", d)
}
