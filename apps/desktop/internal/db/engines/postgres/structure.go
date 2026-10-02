package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) TransactionalDDL() bool { return true }

func (Dialect) CanAlterColumns() bool { return true }

func (Dialect) LockWait(seconds int) (set, reset string) {
	return fmt.Sprintf("SET LOCAL lock_timeout = '%ds'", seconds), ""
}

func (d Dialect) ColumnDDL(_ context.Context, _ *sql.DB, _, _, alter string, cur model.Column, ch model.ColumnChange) ([]string, error) {
	col := d.QuoteIdent(cur.Name)
	var out []string
	dropFirst := dialect.TypeChanged(cur, ch) && ch.DefaultSet && cur.Default != nil
	if dropFirst {
		out = append(out, alter+"ALTER COLUMN "+col+" DROP DEFAULT")
	}
	if dialect.TypeChanged(cur, ch) {
		t := strings.TrimSpace(*ch.Type)
		out = append(out, alter+"ALTER COLUMN "+col+" TYPE "+t+" USING "+col+"::"+t)
	}
	if ch.DefaultSet {
		if ch.Default != nil {
			out = append(out, alter+"ALTER COLUMN "+col+" SET DEFAULT "+*ch.Default)
		} else if !dropFirst {
			out = append(out, alter+"ALTER COLUMN "+col+" DROP DEFAULT")
		}
	}
	if ch.Nullable != nil && *ch.Nullable != cur.Nullable {
		if *ch.Nullable {
			out = append(out, alter+"ALTER COLUMN "+col+" DROP NOT NULL")
		} else {
			out = append(out, alter+"ALTER COLUMN "+col+" SET NOT NULL")
		}
	}
	if name := dialect.NameOf(cur, ch); name != cur.Name {
		out = append(out, alter+"RENAME COLUMN "+col+" TO "+d.QuoteIdent(name))
	}
	return out, nil
}
