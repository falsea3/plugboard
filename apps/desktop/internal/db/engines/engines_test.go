package engines

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/redis"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
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
	for _, driver := range []model.Driver{model.Postgres, model.MySQL, model.SQLite, model.ClickHouse} {
		d, err := Dialect(driver)
		if err != nil {
			t.Fatal(err)
		}
		if want := sqlcore.Features(d); !reflect.DeepEqual(fixture[driver], want) {
			t.Errorf("frontend/e2e/engines.json %s = %+v, the dialect says %+v", driver, fixture[driver], want)
		}
	}
	if want := redis.Features(); !reflect.DeepEqual(fixture[model.Redis], want) {
		t.Errorf("frontend/e2e/engines.json redis = %+v, the engine says %+v", fixture[model.Redis], want)
	}
	if len(fixture) != 5 {
		t.Errorf("frontend/e2e/engines.json lists %d engines", len(fixture))
	}
}
