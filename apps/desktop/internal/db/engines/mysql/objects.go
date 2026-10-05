package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) ListObjects(ctx context.Context, db *sql.DB, schema string) ([]model.DBObject, error) {
	return dialect.QueryObjects(ctx, db, schema, `
		SELECT CAST(routine_name AS CHAR), CAST(LOWER(routine_type) AS CHAR), CAST('' AS CHAR), NULL, NULL
		FROM information_schema.routines WHERE routine_schema = ?
		UNION ALL
		SELECT CAST(trigger_name AS CHAR), 'trigger', CAST(event_object_table AS CHAR), NULL, NULL
		FROM information_schema.triggers WHERE trigger_schema = ?
		UNION ALL
		SELECT CAST(event_name AS CHAR), 'event', CAST('' AS CHAR), NULL, NULL
		FROM information_schema.events WHERE event_schema = ?
		ORDER BY 2, 1`, schema, schema, schema)
}

func (d Dialect) ObjectDDL(ctx context.Context, db *sql.DB, obj model.DBObject) (string, error) {
	name := d.QuoteIdent(obj.Schema) + "." + d.QuoteIdent(obj.Name)
	show := map[string]struct {
		what   string
		column int
	}{
		"table":     {"TABLE", 1},
		"view":      {"VIEW", 1},
		"function":  {"FUNCTION", 2},
		"procedure": {"PROCEDURE", 2},
		"trigger":   {"TRIGGER", 2},
		"event":     {"EVENT", 3},
	}
	s, ok := show[obj.Kind]
	if !ok {
		return "", fmt.Errorf("no DDL for a %s", obj.Kind)
	}
	out, err := dialect.ShowCreate(ctx, db, obj, "SHOW CREATE "+s.what+" "+name, s.column)
	var myErr *mysqldriver.MySQLError
	if errors.As(err, &myErr) && slices.Contains(missing, myErr.Number) {
		return "", dialect.Gone(obj)
	}
	return out, err
}

var missing = []uint16{1146, 1305, 1360, 1539}

func (d Dialect) RenameTable(schema, from, to string) string {
	return "RENAME TABLE " + d.QuoteIdent(schema) + "." + d.QuoteIdent(from) + " TO " + d.QuoteIdent(schema) + "." + d.QuoteIdent(to)
}

func (d Dialect) CreateIndex(schema, table string, idx model.NewIndex) string {
	return dialect.CreateIndex(d.QuoteIdent, d.QuoteIdent(idx.Name), d.QuoteIdent(schema)+"."+d.QuoteIdent(table), idx)
}

func (d Dialect) DropIndex(schema, table, name string) string {
	return "DROP INDEX " + d.QuoteIdent(name) + " ON " + d.QuoteIdent(schema) + "." + d.QuoteIdent(table)
}
