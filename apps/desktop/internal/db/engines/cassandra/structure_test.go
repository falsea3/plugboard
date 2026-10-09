package cassandra

import (
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestStructureSQL(t *testing.T) {
	s := &Session{}
	if got := s.CreateIndexSQL("shop", "customers", model.NewIndex{Name: "by_country", Columns: []string{"country"}}); got != `CREATE INDEX "by_country" ON "shop"."customers" ("country")` {
		t.Errorf("create index = %s", got)
	}
	if got := s.DropIndexSQL("shop", "customers", "by_country"); got != `DROP INDEX "shop"."by_country"` {
		t.Errorf("drop index = %s", got)
	}
	if got := s.TruncateSQL("shop", "customers"); got != `TRUNCATE "shop"."customers"` {
		t.Errorf("truncate = %s", got)
	}
}
