package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/dialect"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqltext"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func (Dialect) TransactionalDDL() bool { return false }

func (Dialect) LockWait(seconds int) (set, reset string) {
	return fmt.Sprintf("SET SESSION lock_wait_timeout = %d", seconds), "SET SESSION lock_wait_timeout = DEFAULT"
}

type columnDef struct {
	columnType, dataType, nullable, collation, comment, extra string
	def                                                       sql.NullString
}

func (d Dialect) ColumnDDL(ctx context.Context, db *sql.DB, schema, table, alter string, cur model.Column, ch model.ColumnChange) ([]string, error) {
	name := dialect.NameOf(cur, ch)
	onlyRename := !dialect.TypeChanged(cur, ch) && !ch.DefaultSet && (ch.Nullable == nil || *ch.Nullable == cur.Nullable)
	if onlyRename && name == cur.Name {
		return nil, nil
	}
	if onlyRename && d.canRenameColumn() {
		return []string{alter + "RENAME COLUMN " + d.QuoteIdent(cur.Name) + " TO " + d.QuoteIdent(name)}, nil
	}

	var m columnDef
	err := db.QueryRowContext(ctx, `
		SELECT column_type, data_type, is_nullable, COALESCE(collation_name, ''), column_comment, extra, column_default
		FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?`,
		schema, table, cur.Name).Scan(&m.columnType, &m.dataType, &m.nullable, &m.collation, &m.comment, &m.extra, &m.def)
	if err != nil {
		return nil, fmt.Errorf("read column %s: %w", cur.Name, err)
	}
	if strings.Contains(strings.ToUpper(m.extra), "GENERATED") && !strings.Contains(strings.ToUpper(m.extra), "DEFAULT_GENERATED") {
		return nil, fmt.Errorf("%s is a generated column: change it in the SQL editor", cur.Name)
	}

	typ := m.columnType
	if dialect.TypeChanged(cur, ch) {
		typ = strings.TrimSpace(*ch.Type)
	}
	def := d.QuoteIdent(name) + " " + typ
	if m.collation != "" && textTypes[dialect.BaseType(typ)] && !namesCollation(typ) {
		def += " COLLATE " + m.collation
	}
	nullable := m.nullable == "YES"
	if ch.Nullable != nil {
		nullable = *ch.Nullable
	}
	if nullable {
		def += " NULL"
	} else {
		def += " NOT NULL"
	}
	switch {
	case ch.DefaultSet && ch.Default != nil:
		def += " DEFAULT " + *ch.Default
	case !ch.DefaultSet:
		if expr := d.defaultSQL(m.def, m.dataType, m.extra); expr != nil {
			def += " DEFAULT " + *expr
		}
	}
	extra := strings.ToLower(m.extra)
	if strings.Contains(extra, "auto_increment") {
		def += " AUTO_INCREMENT"
	}
	if i := strings.Index(extra, "on update "); i >= 0 {
		expr, _, _ := strings.Cut(m.extra[i+len("on update "):], " ")
		def += " ON UPDATE " + strings.TrimSuffix(expr, ",")
	}
	if strings.Contains(extra, "invisible") {
		def += " INVISIBLE"
	}
	if m.comment != "" {
		def += " COMMENT " + d.QuoteString(m.comment)
	}
	return []string{alter + "CHANGE COLUMN " + d.QuoteIdent(cur.Name) + " " + def}, nil
}

func namesCollation(typ string) bool {
	upper := strings.ToUpper(sqltext.Strip(typ, Dialect{}.Syntax()))
	return strings.Contains(upper, "COLLATE") || strings.Contains(upper, "CHARACTER SET") || strings.Contains(upper, "CHARSET")
}

var textTypes = map[string]bool{
	"char": true, "varchar": true, "tinytext": true, "text": true, "mediumtext": true, "longtext": true, "enum": true, "set": true,
}
