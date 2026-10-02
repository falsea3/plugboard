package sqlcore

import (
	"context"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func numsSession(t *testing.T) *Session {
	t.Helper()
	s := openSQLite(t)
	if _, err := s.Run(context.Background(), `create table nums (n integer primary key, odd integer);
		with recursive c(n) as (select 1 union all select n + 1 from c where n < 2500) insert into nums select n, n % 2 from c;
		create table pairs (a integer, b integer, primary key (a, b));
		insert into pairs values (1,1),(1,2),(2,1),(2,2),(3,1)`); err != nil {
		t.Fatal(err)
	}
	return s
}

func firstLast(p model.TablePage) (any, any) {
	r := p.Result.Rows
	return r[0][0], r[len(r)-1][0]
}

func TestKeysetPaging(t *testing.T) {
	s := numsSession(t)
	ctx := context.Background()
	q := model.TableQuery{Schema: "main", Table: "nums", Limit: 1000}
	get := func(q model.TableQuery) model.TablePage {
		t.Helper()
		p, err := s.TablePage(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	p1 := get(q)
	if a, b := firstLast(p1); a != int64(1) || b != int64(1000) || !p1.HasMore || p1.HasPrev || !p1.Keyset {
		t.Fatalf("page 1: %v..%v more=%v prev=%v keyset=%v", a, b, p1.HasMore, p1.HasPrev, p1.Keyset)
	}
	q2 := q
	q2.After = []any{p1.Result.Rows[999][0]}
	p2 := get(q2)
	if a, b := firstLast(p2); a != int64(1001) || b != int64(2000) || !p2.HasMore || !p2.HasPrev {
		t.Fatalf("page 2: %v..%v", a, b)
	}

	last := q
	last.Last = true
	pl := get(last)
	if a, b := firstLast(pl); a != int64(1501) || b != int64(2500) || pl.HasMore || !pl.HasPrev {
		t.Fatalf("last page: %v..%v more=%v prev=%v", a, b, pl.HasMore, pl.HasPrev)
	}
	back := q
	back.Before = []any{pl.Result.Rows[0][0]}
	pb := get(back)
	if a, b := firstLast(pb); a != int64(501) || b != int64(1500) || !pb.HasPrev || !pb.HasMore {
		t.Fatalf("back from last: %v..%v", a, b)
	}
	back.Before = []any{float64(501)}
	pf := get(back)
	if a, b := firstLast(pf); a != int64(1) || b != int64(500) || pf.HasPrev {
		t.Fatalf("back to start: %v..%v prev=%v", a, b, pf.HasPrev)
	}

	odd := q
	odd.Filters = []model.Filter{{Column: "odd", Op: "=", Value: "1"}}
	odd.Last = true
	if p := get(odd); p.Result.Rows[len(p.Result.Rows)-1][0] != int64(2499) {
		t.Fatalf("filtered last page ends at %v", p.Result.Rows[len(p.Result.Rows)-1][0])
	}
}

func TestKeysetCompositeKey(t *testing.T) {
	s := numsSession(t)
	p, err := s.TablePage(context.Background(), model.TableQuery{Schema: "main", Table: "pairs", Limit: 2, After: []any{float64(1), float64(2)}})
	if err != nil {
		t.Fatal(err)
	}
	if r := p.Result.Rows; r[0][0] != int64(2) || r[0][1] != int64(1) || r[1][1] != int64(2) || !p.HasMore {
		t.Fatalf("rows = %v", r)
	}
}

func TestOffsetModeLastPageAndTieBreak(t *testing.T) {
	s := numsSession(t)
	ctx := context.Background()
	p, err := s.TablePage(ctx, model.TableQuery{Schema: "main", Table: "nums", Limit: 1000, OrderBy: "odd", OrderDesc: true, Last: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Keyset || p.Offset != 1500 || p.HasMore || !p.HasPrev {
		t.Fatalf("keyset=%v offset=%d more=%v prev=%v", p.Keyset, p.Offset, p.HasMore, p.HasPrev)
	}
	if a, b := firstLast(p); a != int64(502) || b != int64(2500) {
		t.Fatalf("last page by odd desc: %v..%v", a, b)
	}
}

func TestCountRowsSQLite(t *testing.T) {
	s := numsSession(t)
	ctx := context.Background()
	q := model.TableQuery{Schema: "main", Table: "nums"}
	if c, err := s.CountRows(ctx, q, false); err != nil || c.Known {
		t.Fatalf("SQLite has no estimate: %+v %v", c, err)
	}
	q.Filters = []model.Filter{{Column: "odd", Op: "=", Value: "0"}}
	if c, err := s.CountRows(ctx, q, true); err != nil || !c.Exact || c.Count != 1250 {
		t.Fatalf("exact filtered count: %+v %v", c, err)
	}
}
