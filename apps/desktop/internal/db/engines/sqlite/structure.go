package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func (Dialect) TransactionalDDL() bool { return true }

func (Dialect) CanAlterColumns() bool { return false }

func (d Dialect) ColumnDDL(_ context.Context, _ *sql.DB, _, _, alter string, cur model.Column, ch model.ColumnChange) ([]string, error) {
	if dialect.TypeChanged(cur, ch) || ch.DefaultSet || (ch.Nullable != nil && *ch.Nullable != cur.Nullable) {
		return nil, errors.New("SQLite can only rename a column in place; changing its type, NULL or default means recreating the table in the SQL editor")
	}
	if name := dialect.NameOf(cur, ch); name != cur.Name {
		return []string{alter + "RENAME COLUMN " + d.QuoteIdent(cur.Name) + " TO " + d.QuoteIdent(name)}, nil
	}
	return nil, nil
}
