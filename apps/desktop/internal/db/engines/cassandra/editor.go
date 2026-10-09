package cassandra

import (
	"context"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (s *Session) Run(ctx context.Context, script string) ([]model.ResultSet, error) {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	results := []model.ResultSet{}
	for i, stmt := range split(script) {
		fail := func(err error) ([]model.ResultSet, error) {
			return results, &db.StatementError{Statement: stmt, Index: i, Err: err, Position: errorPosition(err, stmt)}
		}
		if s.conn.ReadOnly {
			if reason := readOnlyReason(stmt); reason != "" {
				return fail(&db.ReadOnlyError{Reason: reason})
			}
		}
		start := time.Now()
		if ks, ok := usedKeyspace(stmt); ok {
			if _, err := s.session(ks); err != nil {
				return fail(err)
			}
			s.keyspace = ks
			results = append(results, model.ResultSet{Statement: stmt, Columns: []model.ResultColumn{}, Rows: [][]any{}, DurationMs: ms(start)})
			continue
		}
		rs, err := s.runOne(ctx, stmt, 0)
		if err != nil {
			return fail(err)
		}
		rs.DurationMs = ms(start)
		results = append(results, rs)
	}
	return results, nil
}

func ms(start time.Time) float64 { return float64(time.Since(start).Microseconds()) / 1000 }

func (s *Session) runOne(ctx context.Context, stmt string, offset int) (model.ResultSet, error) {
	gs, err := s.session(s.keyspace)
	if err != nil {
		return model.ResultSet{}, err
	}
	key := cursorKey("editor", s.keyspace, stmt, offset)
	var state []byte
	skip := 0
	if offset > 0 {
		var ok bool
		if state, ok = s.recall(key); !ok {
			skip = offset
		}
	}
	got, err := fetch(ctx, gs, stmt, nil, state, chunk+skip)
	if err != nil {
		return model.ResultSet{}, err
	}
	rs := model.ResultSet{Statement: stmt, Columns: got.columns, Rows: got.rows, HasRows: len(got.columns) > 0, Pageable: true, Offset: offset}
	if rs.Columns == nil {
		rs.Columns = []model.ResultColumn{}
	}
	if skip > 0 {
		rs.Rows = rs.Rows[min(skip, len(rs.Rows)):]
	}
	if rs.Rows == nil {
		rs.Rows = [][]any{}
	}
	rs.HasMore = len(got.next) > 0
	if rs.HasMore {
		s.remember(cursorKey("editor", s.keyspace, stmt, offset+len(rs.Rows)), got.next)
	}
	return rs, nil
}

func (s *Session) RunMore(ctx context.Context, stmt string, offset int) (model.ResultSet, error) {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	return s.runOne(ctx, stmt, offset)
}
