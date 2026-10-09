package clickhouse

import (
	"context"
	"database/sql"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) ListObjects(ctx context.Context, db *sql.DB, schema string) ([]model.DBObject, error) {
	return dialect.QueryObjects(ctx, db, schema, `
		SELECT name, 'dictionary', '', CAST(NULL, 'Nullable(String)'), CAST(NULL, 'Nullable(String)') FROM system.dictionaries WHERE database = ?
		UNION ALL
		SELECT name, 'function', '', CAST(NULL, 'Nullable(String)'), CAST(NULL, 'Nullable(String)') FROM system.functions WHERE origin = 'SQLUserDefined'
		ORDER BY 2, 1`, schema)
}

func (d Dialect) ObjectDDL(ctx context.Context, db *sql.DB, obj model.DBObject) (string, error) {
	switch obj.Kind {
	case "function":
		var q string
		err := db.QueryRowContext(ctx, `SELECT create_query FROM system.functions WHERE name = ? AND origin = 'SQLUserDefined'`, obj.Name).Scan(&q)
		if err != nil {
			return "", dialect.NoRows(err, obj)
		}
		return dialect.Statement(q), nil
	case "dictionary":
		return dialect.ShowCreate(ctx, db, obj, "SHOW CREATE DICTIONARY "+dialect.Qualified(d, obj.Schema, obj.Name), 0)
	}
	return dialect.ShowCreate(ctx, db, obj, "SHOW CREATE TABLE "+dialect.Qualified(d, obj.Schema, obj.Name), 0)
}
