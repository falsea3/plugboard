package sqltext_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/sqlite"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltext"
)

func TestSplitBlocksAsTheFrontendDoes(t *testing.T) {
	data, err := os.ReadFile("../../../../frontend/e2e/split-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Engine, In string
		Want             []string
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	syntax := map[string]sqltext.Syntax{"postgres": pg, "mysql": my, "sqlite": sqlite.Dialect{}.Syntax(), "cassandra": {DollarQuotes: true}}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if got := sqltext.Split(tc.In, syntax[tc.Engine]); !reflect.DeepEqual(got, tc.Want) {
				t.Fatalf("got %q, want %q", got, tc.Want)
			}
		})
	}
}
