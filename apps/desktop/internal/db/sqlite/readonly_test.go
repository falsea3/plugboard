package sqlite_test

import (
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlcore"
	"github.com/relay-client/relay-db/apps/desktop/internal/db/sqlite"
)

func TestReadOnlyReason(t *testing.T) {
	allowed := []string{
		"select * from t",
		"pragma table_info(t)",
		"pragma foreign_key_list(t)",
		"pragma user_version",
		"begin",
		"commit",
	}
	refused := []string{
		"insert into t values (1)",
		"pragma foreign_keys = off",
		"pragma journal_mode(wal)",
		"pragma wal_checkpoint",
		"reset all",
		"attach database 'x.db' as x",
		"vacuum",
	}
	for _, s := range allowed {
		if v := sqlcore.ReadOnlyReason(sqlite.Dialect{}, s); v != "" {
			t.Errorf("ReadOnlyReason(%q) = %q, want allowed", s, v)
		}
	}
	for _, s := range refused {
		if sqlcore.ReadOnlyReason(sqlite.Dialect{}, s) == "" {
			t.Errorf("ReadOnlyReason(%q) allowed it", s)
		}
	}
}
