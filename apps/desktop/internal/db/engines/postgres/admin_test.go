package postgres_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestRoles(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
	drop := func() { s.DB.ExecContext(ctx, `DROP ROLE IF EXISTS plugboard_app, plugboard_app2`) }
	drop()
	t.Cleanup(drop)

	spec := model.RoleSpec{Role: model.Role{Name: "plugboard_app", Flags: map[string]bool{"login": true, "inherit": true}, ConnLimit: 3, MemberOf: []string{"pg_read_all_data"}}, Password: "pässword 1"}
	if _, err := s.SaveRole(ctx, "", spec); err != nil {
		t.Fatal(err)
	}
	role := findRole(t, s.Roles, "plugboard_app")
	if !role.Flags["login"] || role.ConnLimit != 3 || !slices.Equal(role.MemberOf, []string{"pg_read_all_data"}) || role.System {
		t.Fatalf("role = %+v", role)
	}
	as := c
	as.User, as.Password = "plugboard_app", "pässword 1"
	app := sqltest.Open(t, postgres.Dialect{}, as)
	if res, err := app.Run(ctx, "select count(*) from customers"); err != nil || res[0].Rows[0][0] != int64(2500) {
		t.Fatalf("logging in with the new password and reading through pg_read_all_data: %v %v", res, err)
	}

	next := model.RoleSpec{Role: role, ClearPassword: true}
	next.Name, next.ValidUntil, next.MemberOf = "plugboard_app2", "2099-01-01 00:00:00+00", nil
	preview, err := s.RoleSQL(ctx, "plugboard_app", next)
	if err != nil || len(preview) != 3 {
		t.Fatalf("preview = %q, %v", preview, err)
	}
	if _, err := s.SaveRole(ctx, "plugboard_app", next); err != nil {
		t.Fatal(err)
	}
	role = findRole(t, s.Roles, "plugboard_app2")
	if !strings.HasPrefix(role.ValidUntil, "2099-01-01") || len(role.MemberOf) != 0 {
		t.Fatalf("renamed role = %+v", role)
	}
	if _, err := s.RoleSQL(ctx, "plugboard_app", next); err == nil {
		t.Fatal("a role that no longer exists")
	}
	if err := s.DropRole(ctx, "plugboard_app2"); err != nil {
		t.Fatal(err)
	}
	ro := c
	ro.ReadOnly = true
	var readOnly *db.ReadOnlyError
	if _, err := sqltest.Open(t, postgres.Dialect{}, ro).SaveRole(ctx, "", spec); !errors.As(err, &readOnly) {
		t.Fatalf("read-only: %v", err)
	}
}

func findRole(t *testing.T, list func(context.Context) (model.RoleList, error), name string) model.Role {
	t.Helper()
	roles, err := list(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if roles.Form.Noun != "role" || len(roles.Form.Flags) != 7 {
		t.Fatalf("form = %+v", roles.Form)
	}
	for _, r := range roles.Roles {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no role %s in %+v", name, roles)
	return model.Role{}
}

func TestServerConfig(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_PG", model.Postgres)
	ctx := context.Background()
	s := sqltest.Open(t, postgres.Dialect{}, c)
	files, err := s.ConfigFiles(ctx)
	if err != nil || len(files) != 4 || files[0].Name != "postgresql.conf" || !strings.HasSuffix(files[0].Path, "postgresql.conf") {
		t.Fatalf("files = %+v, %v", files, err)
	}
	conf, err := s.ReadConfig(ctx, files[0].Path)
	if err != nil || conf.Via != "sql" || !strings.Contains(conf.Text, "max_connections") {
		t.Fatalf("read = %.80q via %s, %v", conf.Text, conf.Via, err)
	}
	t.Cleanup(func() {
		if _, err := s.WriteConfig(ctx, conf.Path, conf.Text); err != nil {
			t.Errorf("restoring %s: %v", conf.Path, err)
		}
	})

	saved, err := s.WriteConfig(ctx, conf.Path, conf.Text+"\nwork_mem = '8MB'\nshared_buffers = '200MB'\n")
	if err != nil || !saved.Reloaded || len(saved.Problems) != 0 || !slices.Contains(saved.Restart, "shared_buffers") {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	again, err := s.ReadConfig(ctx, conf.Path)
	if err != nil || !strings.HasSuffix(again.Text, "work_mem = '8MB'\nshared_buffers = '200MB'\n") {
		t.Fatalf("read back %v", err)
	}
	lines := strings.Count(conf.Text, "\n") + 2
	saved, err = s.WriteConfig(ctx, conf.Path, conf.Text+"\nwork_mem = \n")
	if err != nil || saved.Reloaded || len(saved.Problems) != 1 || saved.Problems[0].Line != lines {
		t.Fatalf("a broken line: %+v (want line %d), %v", saved, lines, err)
	}

	hba := files[1].Path
	hbaText, err := s.ReadConfig(ctx, hba)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.WriteConfig(ctx, hba, hbaText.Text) })
	saved, err = s.WriteConfig(ctx, hba, hbaText.Text+"host all all 10.0.0.0/8 nosuchmethod\n")
	if err != nil || saved.Reloaded || len(saved.Problems) != 1 {
		t.Fatalf("a broken pg_hba line: %+v, %v", saved, err)
	}

	if _, err := s.ReadConfig(ctx, "/nonexistent/postgresql.conf"); err == nil {
		t.Fatal("a missing file")
	}
}
