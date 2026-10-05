package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (d Dialect) ListObjects(ctx context.Context, db *sql.DB, schema string) ([]model.DBObject, error) {
	return dialect.QueryObjects(ctx, db, schema, `
		SELECT name, type, tbl_name, NULL, NULL FROM `+d.QuoteIdent(schema)+`.sqlite_master
		WHERE type = 'trigger' ORDER BY name`)
}

func (d Dialect) ObjectDDL(ctx context.Context, db *sql.DB, obj model.DBObject) (string, error) {
	switch obj.Kind {
	case "table", "view", "trigger":
	default:
		return "", fmt.Errorf("no DDL for a %s", obj.Kind)
	}
	master := d.QuoteIdent(obj.Schema) + ".sqlite_master"
	var text sql.NullString
	err := db.QueryRowContext(ctx, `SELECT sql FROM `+master+` WHERE type = ? AND name = ?`, obj.Kind, obj.Name).Scan(&text)
	if err != nil {
		return "", dialect.NoRows(err, obj)
	}
	if !text.Valid {
		return "", dialect.Gone(obj)
	}
	out := dialect.Statement(text.String)
	if obj.Kind != "table" {
		return out, nil
	}
	more, err := dialect.QueryLines(ctx, db, `SELECT sql FROM `+master+`
		WHERE tbl_name = ? AND type IN ('index', 'trigger') AND sql IS NOT NULL ORDER BY type, name`, obj.Name)
	if err != nil {
		return "", err
	}
	for i, s := range more {
		more[i] = dialect.Statement(s)
	}
	if len(more) > 0 {
		out += "\n\n" + strings.Join(more, "\n")
	}
	return out, nil
}

func (d Dialect) CreateIndex(schema, table string, idx model.NewIndex) string {
	return dialect.CreateIndex(d.QuoteIdent, d.QuoteIdent(schema)+"."+d.QuoteIdent(idx.Name), d.QuoteIdent(table), idx)
}
