package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

// TablePage reads one page of a table.
//
// With no sort chosen, rows come in primary-key order and pages are found by
// key ("keyset"): next is WHERE key > last, previous is WHERE key < first, and
// the last page is the first page read backwards. That costs the same on page
// 1 and page 10,000, where OFFSET would make the server read and throw away
// every skipped row. Sorting by another column falls back to OFFSET, with the
// key added as a tie-breaker so rows don't shuffle between pages.
func (s *Session) TablePage(ctx context.Context, q model.TableQuery) (model.TablePage, error) {
	if q.Table == "" {
		return model.TablePage{}, errors.New("table is required")
	}
	limit := q.Limit
	if limit <= 0 || limit > 10000 {
		limit = 300
	}
	offset := max(q.Offset, 0)
	cols, err := s.Columns(ctx, q.Schema, q.Table)
	if err != nil {
		return model.TablePage{}, err
	}
	byName := make(map[string]model.Column, len(cols))
	var keys []model.Column
	for _, c := range cols {
		byName[c.Name] = c
		if c.PrimaryKey {
			keys = append(keys, c)
		}
	}
	if q.OrderBy != "" {
		if _, ok := byName[q.OrderBy]; !ok {
			return model.TablePage{}, fmt.Errorf("unknown column %s", q.OrderBy)
		}
	}

	where, args, err := whereClause(s.Dialect, s.Conn.Driver, byName, q.Filters)
	if err != nil {
		return model.TablePage{}, err
	}
	conds := []string{}
	if where != "" {
		conds = append(conds, where)
	}
	bind := func(col model.Column, v any) string {
		text, isNull := paramText(v)
		if isNull {
			args = append(args, nil)
		} else {
			args = append(args, text)
		}
		if s.Conn.Driver == model.Postgres {
			return pgParam(len(args), col)
		}
		return "?"
	}

	keyset := q.OrderBy == "" && len(keys) > 0
	backwards := false // read in reverse order, then flip the rows
	page := model.TablePage{Offset: -1, Keyset: keyset}
	switch {
	case keyset && len(q.After) > 0:
		if len(q.After) != len(keys) {
			return model.TablePage{}, errors.New("page cursor doesn't match the primary key")
		}
		conds = append(conds, s.keyCompare(keys, ">", q.After, bind))
		page.HasPrev = true
	case keyset && len(q.Before) > 0:
		if len(q.Before) != len(keys) {
			return model.TablePage{}, errors.New("page cursor doesn't match the primary key")
		}
		conds = append(conds, s.keyCompare(keys, "<", q.Before, bind))
		backwards = true
	case keyset && q.Last:
		backwards = true
	case q.Last:
		// OFFSET mode has no cheap way to the end: count, then jump.
		n, err := s.count(ctx, q.Schema, q.Table, where, args)
		if err != nil {
			return model.TablePage{}, err
		}
		offset = max(int(n)-limit, 0)
		page.Offset = offset
	}

	type orderCol struct {
		name string
		desc bool
	}
	var order []orderCol
	if q.OrderBy != "" {
		order = append(order, orderCol{q.OrderBy, q.OrderDesc})
	}
	for _, k := range keys {
		if k.Name != q.OrderBy {
			order = append(order, orderCol{k.Name, false})
		}
	}
	if keyset {
		page.DefaultOrder = make([]string, len(keys))
		for i, k := range keys {
			page.DefaultOrder[i] = k.Name
		}
	}

	var b strings.Builder
	b.WriteString("SELECT * FROM " + qualified(s.Dialect, q.Schema, q.Table))
	if len(conds) > 0 {
		b.WriteString(" WHERE " + strings.Join(conds, " AND "))
	}
	if len(order) > 0 {
		parts := make([]string, len(order))
		for i, o := range order {
			parts[i] = s.Dialect.QuoteIdent(o.name)
			if o.desc != backwards {
				parts[i] += " DESC"
			}
		}
		b.WriteString(" ORDER BY " + strings.Join(parts, ", "))
	}
	// One extra row tells whether there is more in the reading direction.
	fmt.Fprintf(&b, " LIMIT %d", limit+1)
	if !keyset && offset > 0 {
		fmt.Fprintf(&b, " OFFSET %d", offset)
	}

	stmt := b.String()
	start := time.Now()
	rs, err := retry(ctx, func() (model.ResultSet, error) {
		rows, err := s.DB.QueryContext(ctx, stmt, args...)
		if err != nil {
			return model.ResultSet{}, err
		}
		defer rows.Close()
		return readRows(rows, limit+1)
	})
	if err != nil {
		return model.TablePage{}, err
	}
	more := len(rs.Rows) > limit
	if more {
		rs.Rows = rs.Rows[:limit]
	}
	if backwards {
		slices.Reverse(rs.Rows)
		page.HasPrev = more
		page.HasMore = !q.Last
	} else {
		page.HasMore = more
		if !keyset {
			page.HasPrev = offset > 0
		}
	}
	rs.Statement = stmt
	rs.DurationMs = msSince(start)
	page.Result = rs
	return page, nil
}

// keyCompare renders (k1, k2) > (v1, v2) — a row-value comparison, which
// PostgreSQL, MySQL and SQLite all evaluate in key order.
func (s *Session) keyCompare(keys []model.Column, op string, values []any, bind func(model.Column, any) string) string {
	names := make([]string, len(keys))
	vals := make([]string, len(keys))
	for i, k := range keys {
		names[i] = s.Dialect.QuoteIdent(k.Name)
		vals[i] = bind(k, values[i])
	}
	if len(keys) == 1 {
		return names[0] + " " + op + " " + vals[0]
	}
	return "(" + strings.Join(names, ", ") + ") " + op + " (" + strings.Join(vals, ", ") + ")"
}

func (s *Session) count(ctx context.Context, schema, table, where string, args []any) (int64, error) {
	stmt := "SELECT COUNT(*) FROM " + qualified(s.Dialect, schema, table)
	if where != "" {
		stmt += " WHERE " + where
	}
	return retry(ctx, func() (int64, error) {
		var n int64
		err := s.DB.QueryRowContext(ctx, stmt, args...).Scan(&n)
		return n, err
	})
}

// CountRows returns the exact row count (honouring filters) when exact is
// set, else the engine's own estimate from its statistics — instant, but
// only available without filters and not for SQLite.
func (s *Session) CountRows(ctx context.Context, q model.TableQuery, exact bool) (model.RowCount, error) {
	if exact {
		cols, err := s.Columns(ctx, q.Schema, q.Table)
		if err != nil {
			return model.RowCount{}, err
		}
		byName := make(map[string]model.Column, len(cols))
		for _, c := range cols {
			byName[c.Name] = c
		}
		where, args, err := whereClause(s.Dialect, s.Conn.Driver, byName, q.Filters)
		if err != nil {
			return model.RowCount{}, err
		}
		n, err := s.count(ctx, q.Schema, q.Table, where, args)
		return model.RowCount{Count: n, Exact: true, Known: true}, err
	}
	if len(q.Filters) > 0 {
		return model.RowCount{}, nil
	}
	var stmt string
	switch s.Conn.Driver {
	case model.Postgres:
		// reltuples is -1 until the table has been analyzed or vacuumed.
		stmt = `SELECT c.reltuples::bigint FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2 AND c.reltuples >= 0`
	case model.MySQL:
		stmt = `SELECT table_rows FROM information_schema.tables WHERE table_schema = ? AND table_name = ? AND table_rows IS NOT NULL`
	default:
		return model.RowCount{}, nil
	}
	var n sql.NullInt64
	err := s.DB.QueryRowContext(ctx, stmt, q.Schema, q.Table).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) || !n.Valid {
		return model.RowCount{}, nil
	}
	if err != nil {
		return model.RowCount{}, err
	}
	return model.RowCount{Count: n.Int64, Known: true}, nil
}
