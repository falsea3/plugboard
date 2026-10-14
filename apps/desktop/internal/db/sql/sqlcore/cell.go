package sqlcore

import (
	"context"
	"fmt"
	"strconv"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ db.CellReader = (*Session)(nil)

const MaxFullCellBytes = 64 << 20

func (s *Session) ReadCell(ctx context.Context, q model.CellQuery) (string, error) {
	cols, err := s.Columns(ctx, q.Schema, q.Table)
	if err != nil {
		return "", err
	}
	byName := make(map[string]model.Column, len(cols))
	var keys []model.Column
	for _, c := range cols {
		byName[c.Name] = c
		if c.PrimaryKey {
			keys = append(keys, c)
		}
	}
	col, ok := byName[q.Column]
	if !ok {
		return "", fmt.Errorf("unknown column %s", q.Column)
	}
	b := changeBuilder{d: s.Dialect, table: s.qualified(q.Schema, q.Table), cols: byName, keys: keys}
	where, err := b.where(q.Key)
	if err != nil {
		return "", err
	}
	stmt := "SELECT " + s.Dialect.QuoteIdent(col.Name) + " FROM " + b.table + " WHERE " + where
	return retry(ctx, s, func() (string, error) {
		rows, err := s.DB.QueryContext(ctx, stmt, b.args...)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return "", err
			}
			return "", db.ErrRowGone
		}
		var v any
		if err := rows.Scan(&v); err != nil {
			return "", err
		}
		text, ok := normalizeValue(v, s.Dialect.TypeOf(col.Type)).(string)
		if !ok {
			return fmt.Sprint(normalizeValue(v, s.Dialect.TypeOf(col.Type))), nil
		}
		if len(text) > MaxFullCellBytes {
			return "", apperr.New("too_big", "this value is "+strconv.Itoa(len(text)>>20)+" MB — too big to open here; read it with SQL in parts, like substring(…)")
		}
		return text, nil
	})
}
