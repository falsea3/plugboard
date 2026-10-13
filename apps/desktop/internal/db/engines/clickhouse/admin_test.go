package clickhouse_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/clickhouse"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestUsersAndRoles(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	cleanup := func() {
		for _, stmt := range []string{"DROP USER IF EXISTS plugboard_app", "DROP USER IF EXISTS plugboard_app2", "DROP ROLE IF EXISTS plugboard_readers"} {
			s.DB.ExecContext(ctx, stmt)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	if _, err := s.DB.ExecContext(ctx, "CREATE ROLE plugboard_readers"); err != nil {
		t.Fatal(err)
	}
	spec := model.RoleSpec{Role: model.Role{Name: "plugboard_app", Flags: map[string]bool{}, MemberOf: []string{"role:plugboard_readers"}}, Password: `p'w\1`}
	if id, err := s.SaveRole(ctx, "", spec); err != nil || id != "plugboard_app" {
		t.Fatal(id, err)
	}
	user := chFind(t, s.Roles, "plugboard_app")
	if user.Kind != "" || user.System || !slices.Equal(user.MemberOf, []string{"role:plugboard_readers"}) || user.Flags["full"] {
		t.Fatalf("user = %+v", user)
	}
	relay := chFind(t, s.Roles, "relay")
	if !relay.System || !relay.Current {
		t.Fatalf("relay comes from users.xml: %+v", relay)
	}
	if role := chFind(t, s.Roles, "plugboard_readers"); role.Kind != "role" || !role.Grantable || role.ID != "role:plugboard_readers" {
		t.Fatalf("role = %+v", role)
	}
	c := sqltest.Conn(t, "PLUGBOARD_TEST_CLICKHOUSE", model.ClickHouse)
	c.User, c.Password, c.Database = "plugboard_app", `p'w\1`, ""
	if _, err := sqltest.Open(t, clickhouse.Dialect{}, c).Run(ctx, "SELECT 1"); err != nil {
		t.Fatalf("logging in as the new user: %v", err)
	}

	all := model.RoleSpec{Role: user}
	all.Flags = map[string]bool{"full": true}
	if _, err := s.SaveRole(ctx, user.ID, all); err == nil || !strings.Contains(err.Error(), "Not enough privileges") {
		t.Fatalf("relay can't hand out ALL it doesn't hold: %v", err)
	}
	next := model.RoleSpec{Role: user}
	next.Name = "plugboard_app2"
	next.MemberOf = nil
	if _, err := s.SaveRole(ctx, user.ID, next); err != nil {
		t.Fatal(err)
	}
	user = chFind(t, s.Roles, "plugboard_app2")
	if len(user.MemberOf) != 0 || slices.ContainsFunc(user.Grants, func(g string) bool { return strings.Contains(g, "plugboard_readers") }) {
		t.Fatalf("after the change: %+v", user)
	}
	if err := s.DropRole(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DropRole(ctx, "role:plugboard_readers"); err != nil {
		t.Fatal(err)
	}
}

func chFind(t *testing.T, list func(context.Context) (model.RoleList, error), name string) model.Role {
	t.Helper()
	roles, err := list(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles.Roles {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no %s in %+v", name, roles.Roles)
	return model.Role{}
}

func TestServerSettings(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	files, err := s.ConfigFiles(ctx)
	if err != nil || len(files) != 3 {
		t.Fatalf("%+v %v", files, err)
	}
	got, err := s.ReadConfig(ctx, files[0].Path)
	if err != nil || !got.ReadOnly || got.Via != "live" || !strings.Contains(got.Text, "\nmax_connections: ") {
		t.Fatalf("%.200q %v", got.Text, err)
	}
	if _, err := s.WriteConfig(ctx, files[0].Path, got.Text); err == nil {
		t.Fatal("server settings are read-only")
	}
	if _, err := s.ReadConfig(ctx, files[1].Path); err == nil || !strings.Contains(err.Error(), "SSH") {
		t.Fatalf("config.xml without SSH: %v", err)
	}
}
