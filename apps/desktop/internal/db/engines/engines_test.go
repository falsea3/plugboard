package engines

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func TestFrontendEngineFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "frontend", "e2e", "engines.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[model.Driver]model.EngineFeatures
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, driver := range []model.Driver{model.Postgres, model.MySQL, model.SQLite} {
		d, err := Dialect(driver)
		if err != nil {
			t.Fatal(err)
		}
		if want := sqlcore.Features(d); !reflect.DeepEqual(fixture[driver], want) {
			t.Errorf("frontend/e2e/engines.json %s = %+v, the dialect says %+v", driver, fixture[driver], want)
		}
	}
	if len(fixture) != 3 {
		t.Errorf("frontend/e2e/engines.json lists %d engines", len(fixture))
	}
}
