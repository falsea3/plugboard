package dialect

import (
	"context"
	"database/sql"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Standard struct{}

func (Standard) QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (Standard) QuoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (s Standard) RenameTable(schema, from, to string) string {
	return "ALTER TABLE " + s.QuoteIdent(schema) + "." + s.QuoteIdent(from) + " RENAME TO " + s.QuoteIdent(to)
}

func (s Standard) CreateIndex(schema, table string, idx model.NewIndex) string {
	return createIndex(s.QuoteIdent, s.QuoteIdent(idx.Name), s.QuoteIdent(schema)+"."+s.QuoteIdent(table), idx)
}

func (s Standard) DropIndex(schema, _, name string) string {
	return "DROP INDEX " + s.QuoteIdent(schema) + "." + s.QuoteIdent(name)
}

func CreateIndex(quote func(string) string, name, table string, idx model.NewIndex) string {
	return createIndex(quote, name, table, idx)
}

func createIndex(quote func(string) string, name, table string, idx model.NewIndex) string {
	cols := make([]string, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = quote(c)
	}
	unique := ""
	if idx.Unique {
		unique = "UNIQUE "
	}
	return "CREATE " + unique + "INDEX " + name + " ON " + table + " (" + strings.Join(cols, ", ") + ")"
}

func (Standard) Placeholder(int) string { return "?" }

func (Standard) Param(int, model.Column) string { return "?" }

func (Standard) InsertDefaults(table string) string {
	return "INSERT INTO " + table + " DEFAULT VALUES"
}

func (Standard) CanUpdateToDefault() bool { return true }

func (Standard) EstimateRows(context.Context, *sql.DB, string, string) (int64, bool, error) {
	return 0, false, nil
}

func (Standard) ReturnsRows(stmt string) bool { return sqltext.ReturnsRows(stmt) }

func (Standard) CanPage(string) bool { return true }

func (Standard) TxStatus(*sql.Conn) TxStatus { return TxUnknown }

func (Standard) TransactionAfter(stmt string, open bool) bool {
	return StandardTransactionAfter(stmt, open)
}

func (Standard) ErrorAbortsTransaction() bool { return false }

func (Standard) Classify(error) ErrorKind { return Other }

func (Standard) ErrorPosition(error, string) int { return -1 }

func (Standard) ErrorMessage(err error) string { return err.Error() }

func (Standard) ReadKeywords() []string { return nil }

func (Standard) LockWait(int) (set, reset string) { return "", "" }

func StandardTransactionAfter(stmt string, open bool) bool {
	switch sqltext.FirstKeyword(stmt) {
	case "BEGIN":
		return true
	case "START":
		return open || sqltext.ContainsKeyword(stmt, "TRANSACTION")
	case "COMMIT", "END":
		return false
	case "ROLLBACK":
		return open && sqltext.ContainsKeyword(stmt, "TO")
	}
	return open
}
