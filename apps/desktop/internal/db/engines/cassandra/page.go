package cassandra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const maxCursors = 500

var errPageGone = apperr.New("page_gone", "the next page is no longer known — reload the table")

func (s *Session) remember(key string, state []byte) {
	s.pagesMu.Lock()
	defer s.pagesMu.Unlock()
	if len(s.pages) >= maxCursors {
		clear(s.pages)
	}
	s.pages[key] = state
}

func (s *Session) recall(key string) ([]byte, bool) {
	s.pagesMu.Lock()
	defer s.pagesMu.Unlock()
	state, ok := s.pages[key]
	return state, ok
}

func cursorKey(parts ...any) string {
	b, _ := json.Marshal(parts)
	return string(b)
}

func (s *Session) tableQuery(ctx context.Context, q model.TableQuery, what string) (string, []any, []model.Column, error) {
	cols, err := s.Columns(ctx, q.Schema, q.Table)
	if err != nil {
		return "", nil, nil, err
	}
	if len(cols) == 0 {
		return "", nil, nil, fmt.Errorf("table %s not found", q.Table)
	}
	byName := map[string]model.Column{}
	for _, c := range cols {
		byName[c.Name] = c
	}
	var conds []string
	var args []any
	for _, f := range q.Filters {
		col, ok := byName[f.Column]
		if !ok {
			return "", nil, nil, fmt.Errorf("unknown column %s", f.Column)
		}
		cond, vals, err := filterCond(col, f)
		if err != nil {
			return "", nil, nil, err
		}
		conds = append(conds, cond)
		args = append(args, vals...)
	}
	stmt := "SELECT " + what + " FROM " + qualified(q.Schema, q.Table)
	if len(conds) > 0 {
		stmt += " WHERE " + strings.Join(conds, " AND ") + " ALLOW FILTERING"
	}
	return stmt, args, cols, nil
}

var ops = map[string]string{"=": "=", "<": "<", ">": ">", "<=": "<=", ">=": ">="}

func filterCond(col model.Column, f model.Filter) (string, []any, error) {
	name := quote(col.Name)
	if op, ok := ops[f.Op]; ok {
		v, err := jsonFor(col, f.Value)
		return name + " " + op + " fromJson(?)", []any{v}, err
	}
	switch f.Op {
	case "in":
		var marks []string
		var args []any
		for _, item := range strings.Split(f.Value, ",") {
			v, err := jsonFor(col, strings.TrimSpace(item))
			if err != nil {
				return "", nil, err
			}
			marks = append(marks, "fromJson(?)")
			args = append(args, v)
		}
		return name + " IN (" + strings.Join(marks, ", ") + ")", args, nil
	case "contains":
		if col.Kind == model.KindOther {
			return name + " CONTAINS ?", []any{f.Value}, nil
		}
	}
	return "", nil, apperr.New("not_supported", "Cassandra can't filter "+col.Name+" that way — it filters with =, <, >, IN, and CONTAINS on collections")
}

func (s *Session) TablePage(ctx context.Context, q model.TableQuery) (model.TablePage, error) {
	if q.OrderBy != "" {
		return model.TablePage{}, errNoSort
	}
	if q.Last || len(q.Before) > 0 || q.Offset > 0 {
		return model.TablePage{}, db.ErrNotSupported
	}
	limit := q.Limit
	if limit <= 0 || limit > 10000 {
		limit = 300
	}
	stmt, args, cols, err := s.tableQuery(ctx, q, "*")
	if err != nil {
		return model.TablePage{}, err
	}
	var keys []int
	var names []string
	gs, err := s.main()
	if err != nil {
		return model.TablePage{}, err
	}
	var state []byte
	if len(q.After) > 0 {
		var ok bool
		if state, ok = s.recall(cursorKey(stmt, args, q.After)); !ok {
			return model.TablePage{}, errPageGone
		}
	}
	got, err := fetch(ctx, gs, stmt, args, state, limit)
	if err != nil {
		return model.TablePage{}, err
	}
	for _, c := range cols {
		if c.PrimaryKey {
			for i, rc := range got.columns {
				if rc.Name == c.Name {
					keys = append(keys, i)
					names = append(names, c.Name)
				}
			}
		}
	}
	page := model.TablePage{
		Result:       model.ResultSet{Columns: got.columns, Rows: got.rows, HasRows: true},
		HasMore:      len(got.next) > 0,
		DefaultOrder: names,
		Keyset:       true,
		Offset:       -1,
	}
	if page.Result.Rows == nil {
		page.Result.Rows = [][]any{}
	}
	if page.HasMore && len(got.rows) > 0 {
		last := got.rows[len(got.rows)-1]
		after := make([]any, len(keys))
		for i, k := range keys {
			after[i] = last[k]
		}
		s.remember(cursorKey(stmt, args, after), got.next)
	}
	return page, nil
}

func (s *Session) CountRows(ctx context.Context, q model.TableQuery, exact bool) (model.RowCount, error) {
	if !exact {
		return model.RowCount{}, nil
	}
	stmt, args, _, err := s.tableQuery(ctx, q, "COUNT(*)")
	if err != nil {
		return model.RowCount{}, err
	}
	gs, err := s.main()
	if err != nil {
		return model.RowCount{}, err
	}
	var n int64
	if err := gs.Query(stmt, args...).WithContext(ctx).Scan(&n); err != nil {
		return model.RowCount{}, err
	}
	return model.RowCount{Count: n, Exact: true, Known: true}, nil
}
