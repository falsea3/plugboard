package postgres_test

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestStress(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_STRESS_PG")
	if addr == "" {
		t.Skip("PLUGBOARD_STRESS_PG not set (load dev/stress/postgres.sql with make db-stress)")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{Driver: model.Postgres, Host: addr[:i], Port: port, User: "relay", Password: "relay", Database: "stress", SSLMode: "disable"}
	ctx := context.Background()
	s, err := sqlcore.Open(ctx, "stress", c, postgres.Dialect{}, db.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	measure := func(name string, fn func() (any, error)) {
		start := time.Now()
		v, err := fn()
		took := time.Since(start)
		data, _ := json.Marshal(v)
		t.Logf("%-34s %8.0f ms %12s sent  err=%v", name, float64(took.Milliseconds()), size(len(data)), err)
	}
	page := func(schema, table string, limit int) func() (any, error) {
		return func() (any, error) {
			return s.TablePage(ctx, model.TableQuery{Schema: schema, Table: table, Limit: limit})
		}
	}
	measure("Info (schemas)", func() (any, error) { return s.Info(ctx) })
	measure("Tables many (10k)", func() (any, error) { return s.Tables(ctx, "many") })
	measure("Relations many", func() (any, error) { return s.Relations(ctx, "many") })
	measure("Objects many", func() (any, error) { return s.Objects(ctx, "many") })
	measure("Columns wide (1500)", func() (any, error) { return s.Columns(ctx, "stress", "wide") })
	measure("TablePage wide 300", page("stress", "wide", 300))
	measure("TablePage big 300", page("stress", "big", 300))
	measure("CountRows big exact", func() (any, error) {
		return s.CountRows(ctx, model.TableQuery{Schema: "stress", Table: "big"}, true)
	})
	measure("TablePage big sorted by label", func() (any, error) {
		return s.TablePage(ctx, model.TableQuery{Schema: "stress", Table: "big", Limit: 300, OrderBy: "label"})
	})
	measure("TablePage huge_cells 300", page("stress", "huge_cells", 300))
	measure("TablePage fat_json 300", page("stress", "fat_json", 300))
	measure("Editor select * from fat_json", func() (any, error) { return s.Run(ctx, "select * from stress.fat_json") })
	measure("Editor select * from big", func() (any, error) { return s.Run(ctx, "select * from stress.big") })
	measure("Diagram many (10k tables)", func() (any, error) {
		dctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		return s.Diagram(dctx, "many")
	})
}

func size(n int) string {
	switch {
	case n > 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + " MB"
	case n > 1<<10:
		return strconv.FormatFloat(float64(n)/(1<<10), 'f', 1, 64) + " KB"
	}
	return strconv.Itoa(n) + " B"
}
