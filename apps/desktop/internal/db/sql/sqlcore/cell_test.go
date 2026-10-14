package sqlcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestCutsBigCellsAndReadsThemWhole(t *testing.T) {
	s := openSQLite(t)
	ctx := context.Background()
	big := strings.Repeat("ü", MaxCellBytes)
	if _, err := s.DB.ExecContext(ctx, "insert into users (id, name) values (100, ?)", big); err != nil {
		t.Fatal(err)
	}
	p, err := s.TablePage(ctx, model.TableQuery{Schema: "main", Table: "users", Limit: 300})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Result.Cut) != 1 || p.Result.Columns[1].Kind != model.KindText {
		t.Fatalf("cut = %+v", p.Result.Cut)
	}
	c := p.Result.Cut[0]
	shown := p.Result.Rows[c.Row][c.Col].(string)
	if c.Size != 0 || len(shown) > MaxCellBytes || !utf8.ValidString(shown) || p.Result.Rows[c.Row][0] != int64(100) {
		t.Fatalf("cut %+v, shown %d bytes", c, len(shown))
	}
	full, err := s.ReadCell(ctx, model.CellQuery{Schema: "main", Table: "users", Column: "name", Key: map[string]any{"id": "100"}})
	if err != nil || full != big {
		t.Fatalf("whole value: %d bytes, %v", len(full), err)
	}
	if _, err := s.ReadCell(ctx, model.CellQuery{Schema: "main", Table: "users", Column: "name", Key: map[string]any{"id": "999"}}); !errors.Is(err, db.ErrRowGone) {
		t.Fatalf("a row that's gone: %v", err)
	}
	res, err := s.Run(ctx, "select name from users where id = 100")
	if err != nil || len(res[0].Cut) != 1 || res[0].Cut[0].Row != 0 || res[0].Cut[0].Size != len(big) {
		t.Fatalf("the editor cuts too: %+v %v", res[0].Cut, err)
	}
}

func TestCutMarkersFollowTheRows(t *testing.T) {
	rs := model.ResultSet{Rows: [][]any{{1}, {2}, {3}}, Cut: []model.CutCell{{Row: 0, Col: 0}, {Row: 2, Col: 0}}}
	keepRows(&rs, 2)
	if len(rs.Rows) != 2 || len(rs.Cut) != 1 {
		t.Fatalf("%+v", rs)
	}
	reverseRows(&rs)
	if rs.Rows[1][0] != 1 || rs.Cut[0].Row != 1 {
		t.Fatalf("%+v", rs)
	}
	if cutText("aü", 2) != "a" || cutText("abc", 5) != "abc" {
		t.Fatal("cutText")
	}
}
