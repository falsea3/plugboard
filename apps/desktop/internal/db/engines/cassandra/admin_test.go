package cassandra_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestRoles(t *testing.T) {
	c := conn(t)
	ctx := context.Background()
	s := open(t, c)
	drop := func() {
		s.DropRole(ctx, "plugboard_app")
		s.DropRole(ctx, "plugboard_readers")
	}
	drop()
	t.Cleanup(drop)
	if _, err := s.SaveRole(ctx, "", model.RoleSpec{Role: model.Role{Name: "plugboard_readers", Flags: map[string]bool{}}}); err != nil {
		t.Fatal(err)
	}
	spec := model.RoleSpec{Role: model.Role{Name: "plugboard_app", Flags: map[string]bool{"login": true}, MemberOf: []string{"plugboard_readers"}}, Password: "p'w 1"}
	if _, err := s.SaveRole(ctx, "", spec); err != nil {
		t.Fatal(err)
	}
	app := find(t, s.Roles, "plugboard_app")
	if !app.Flags["login"] || app.Flags["superuser"] || !slices.Equal(app.MemberOf, []string{"plugboard_readers"}) {
		t.Fatalf("app = %+v", app)
	}
	as := c
	as.User, as.Password, as.Database = "plugboard_app", "p'w 1", ""
	if _, err := open(t, as).Run(ctx, "SELECT release_version FROM system.local"); err != nil {
		t.Fatalf("logging in as the new role: %v", err)
	}
	next := model.RoleSpec{Role: app}
	next.Flags = map[string]bool{}
	next.MemberOf = nil
	if _, err := s.SaveRole(ctx, app.ID, next); err != nil {
		t.Fatal(err)
	}
	app = find(t, s.Roles, "plugboard_app")
	if app.Flags["login"] || len(app.MemberOf) != 0 {
		t.Fatalf("after the change: %+v", app)
	}
	if err := s.DropRole(ctx, "plugboard_app"); err != nil {
		t.Fatal(err)
	}
	if !find(t, s.Roles, "relay").Current {
		t.Fatal("the session's role isn't marked")
	}
}

func find(t *testing.T, list func(context.Context) (model.RoleList, error), name string) model.Role {
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
	t.Fatalf("no %s", name)
	return model.Role{}
}

func TestCurrentSettings(t *testing.T) {
	s := open(t, conn(t))
	ctx := context.Background()
	files, _ := s.ConfigFiles(ctx)
	got, err := s.ReadConfig(ctx, files[0].Path)
	if err != nil || !got.ReadOnly || !strings.Contains(got.Text, "\nauthenticator: ") {
		t.Fatalf("%.120q %v", got.Text, err)
	}
	if _, err := s.WriteConfig(ctx, files[0].Path, got.Text); err == nil {
		t.Fatal("settings are read-only")
	}
	if _, err := s.ReadConfig(ctx, files[1].Path); err == nil || !strings.Contains(err.Error(), "SSH") {
		t.Fatalf("cassandra.yaml without SSH: %v", err)
	}
}
