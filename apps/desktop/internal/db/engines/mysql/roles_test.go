package mysql_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/db/engines/mysql"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/sqltest"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestUsers(t *testing.T) {
	c := sqltest.Conn(t, "PLUGBOARD_TEST_MYSQL", model.MySQL)
	c.User = "root"
	ctx := context.Background()
	s := sqltest.Open(t, mysql.Dialect{}, c)
	roleID := "`plugboard_r`@`%`"
	if mariaDB(s) {
		roleID = "`plugboard_r`"
	}
	cleanup := func() {
		for _, stmt := range []string{"DROP USER IF EXISTS plugboard_app@'%'", "DROP USER IF EXISTS plugboard_app2@'%'", "DROP ROLE IF EXISTS plugboard_r"} {
			s.DB.ExecContext(ctx, stmt)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	if _, err := s.DB.ExecContext(ctx, "CREATE ROLE plugboard_r"); err != nil {
		t.Fatal(err)
	}

	spec := model.RoleSpec{Role: model.Role{Name: "plugboard_app", Host: "%", ConnLimit: 3, Flags: map[string]bool{"process": true, "createUser": true}, MemberOf: []string{roleID}}, Password: `pä's\s 1`}
	if id, err := s.SaveRole(ctx, "", spec); err != nil || id != "`plugboard_app`@`%`" {
		t.Fatal(id, err)
	}
	user := findUser(t, s, "plugboard_app")
	if user.ID != "`plugboard_app`@`%`" || user.ConnLimit != 3 || !user.Flags["process"] || !user.Flags["createUser"] || user.Flags["full"] || !slices.Equal(user.MemberOf, []string{roleID}) {
		t.Fatalf("user = %+v", user)
	}
	if !slices.ContainsFunc(user.Grants, func(g string) bool { return strings.Contains(g, "PROCESS") }) || slices.ContainsFunc(user.Grants, func(g string) bool { return strings.Contains(g, "PASSWORD") }) {
		t.Fatalf("grants = %q", user.Grants)
	}
	as := c
	as.User, as.Password, as.Database = "plugboard_app", `pä's\s 1`, ""
	if _, err := sqltest.Open(t, mysql.Dialect{}, as).Run(ctx, "select 1"); err != nil {
		t.Fatalf("logging in with the new password: %v", err)
	}

	next := model.RoleSpec{Role: user}
	next.Name = "plugboard_app2"
	next.Flags = map[string]bool{"full": true, "locked": true}
	next.ConnLimit = -1
	next.MemberOf = nil
	if _, err := s.SaveRole(ctx, user.ID, next); err != nil {
		t.Fatal(err)
	}
	user = findUser(t, s, "plugboard_app2")
	if !user.Flags["full"] || !user.Flags["locked"] || user.ConnLimit != -1 || len(user.MemberOf) != 0 {
		t.Fatalf("after the change: %+v", user)
	}

	next = model.RoleSpec{Role: user}
	next.Flags = map[string]bool{"process": true}
	if _, err := s.SaveRole(ctx, user.ID, next); err != nil {
		t.Fatal(err)
	}
	user = findUser(t, s, "plugboard_app2")
	if user.Flags["full"] || !user.Flags["process"] || user.Flags["createUser"] || user.Flags["locked"] {
		t.Fatalf("all privileges taken back but PROCESS: %+v", user)
	}
	if err := s.DropRole(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DropRole(ctx, roleID); err != nil {
		t.Fatal(err)
	}
	list, _ := s.Roles(ctx)
	if slices.ContainsFunc(list.Roles, func(r model.Role) bool { return strings.HasPrefix(r.Name, "plugboard_") }) {
		t.Fatalf("still listed: %+v", list.Roles)
	}
	if !slices.ContainsFunc(list.Roles, func(r model.Role) bool { return r.Current && r.Name == "root" }) {
		t.Fatal("the current account isn't marked")
	}
}

func findUser(t *testing.T, s *sqlcore.Session, name string) model.Role {
	t.Helper()
	list, err := s.Roles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if list.Form.Noun != "user" || !list.Form.Host {
		t.Fatalf("form = %+v", list.Form)
	}
	for _, r := range list.Roles {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no %s in %+v", name, list.Roles)
	return model.Role{}
}
