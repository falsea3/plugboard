package clickhouse

import (
	"context"
	"database/sql"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) CanAlterColumns() bool { return true }

func (d Dialect) ColumnDDL(_ context.Context, _ *sql.DB, _, _, alter string, cur model.Column, ch model.ColumnChange) ([]string, error) {
	var out []string
	name := dialect.NameOf(cur, ch)
	if name != cur.Name {
		out = append(out, alter+" RENAME COLUMN "+d.QuoteIdent(cur.Name)+" TO "+d.QuoteIdent(name))
	}
	typ := cur.Type
	if ch.Type != nil {
		typ = strings.TrimSpace(*ch.Type)
	}
	if ch.Nullable != nil {
		inner := unwrap(typ, "Nullable")
		typ = inner
		if *ch.Nullable {
			typ = "Nullable(" + inner + ")"
		}
	}
	if typ != cur.Type {
		out = append(out, alter+" MODIFY COLUMN "+d.QuoteIdent(name)+" "+typ)
	}
	if ch.DefaultSet {
		if ch.Default == nil || strings.TrimSpace(*ch.Default) == "" {
			out = append(out, alter+" MODIFY COLUMN "+d.QuoteIdent(name)+" REMOVE DEFAULT")
		} else {
			out = append(out, alter+" MODIFY COLUMN "+d.QuoteIdent(name)+" DEFAULT "+strings.TrimSpace(*ch.Default))
		}
	}
	return out, nil
}

func (d Dialect) RenameTable(schema, from, to string) string {
	return "RENAME TABLE " + dialect.Qualified(d, schema, from) + " TO " + dialect.Qualified(d, schema, to)
}

func (d Dialect) CreateIndex(schema, table string, idx model.NewIndex) string {
	cols := make([]string, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = d.QuoteIdent(c)
	}
	expr := strings.Join(cols, ", ")
	if len(cols) > 1 {
		expr = "(" + expr + ")"
	}
	return "ALTER TABLE " + dialect.Qualified(d, schema, table) + " ADD INDEX " + d.QuoteIdent(idx.Name) + " " + expr + " TYPE minmax GRANULARITY 1"
}

func (d Dialect) Truncate(schema, table string) string {
	return "TRUNCATE TABLE " + dialect.Qualified(d, schema, table)
}

func (d Dialect) DropIndex(schema, table, name string) string {
	return "ALTER TABLE " + dialect.Qualified(d, schema, table) + " DROP INDEX " + d.QuoteIdent(name)
}
