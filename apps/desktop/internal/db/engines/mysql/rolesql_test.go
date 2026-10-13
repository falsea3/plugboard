package mysql

import (
	"reflect"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func on(keys ...string) map[string]bool {
	out := map[string]bool{}
	for _, k := range keys {
		out[k] = true
	}
	return out
}

func TestCreateUserSQL(t *testing.T) {
	to := model.RoleSpec{Role: model.Role{Name: "app", Flags: on("process", "grant", "locked"), ConnLimit: 5, MemberOf: []string{"`r`@`%`"}}, Password: `it's\`}
	got, err := Dialect{}.RoleSQL(nil, to, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"CREATE USER `app`@`%` IDENTIFIED BY 'it''s\\\\' WITH MAX_USER_CONNECTIONS 5 ACCOUNT LOCK",
		"GRANT PROCESS ON *.* TO `app`@`%`",
		"GRANT USAGE ON *.* TO `app`@`%` WITH GRANT OPTION",
		"GRANT `r`@`%` TO `app`@`%`",
		"SET DEFAULT ROLE ALL TO `app`@`%`",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	preview, _ := Dialect{}.RoleSQL(nil, to, nil, true)
	if preview[0] != "CREATE USER `app`@`%` IDENTIFIED BY '********' WITH MAX_USER_CONNECTIONS 5 ACCOUNT LOCK" {
		t.Fatalf("preview %q", preview[0])
	}
}

func TestAlterUserSQL(t *testing.T) {
	from := &model.Role{ID: "`app`@`%`", Name: "app", Host: "%", Flags: on("full", "process", "createUser", "reload", "replication"), ConnLimit: -1, MemberOf: []string{"`r`"}}
	to := model.RoleSpec{Role: *from, ClearPassword: true}
	to.Host = "10.%"
	to.Flags = on("process", "full", "createUser", "reload")
	to.MemberOf = nil
	got, _ := Dialect{mariaDB: true}.RoleSQL(from, to, nil, false)
	want := []string{
		"RENAME USER `app`@`%` TO `app`@`10.%`",
		"ALTER USER `app`@`10.%` IDENTIFIED BY ''",
		"REVOKE ALL PRIVILEGES ON *.* FROM `app`@`10.%`",
		"GRANT CREATE USER, PROCESS, RELOAD ON *.* TO `app`@`10.%`",
		"REVOKE `r` FROM `app`@`10.%`",
		"SET DEFAULT ROLE NONE FOR `app`@`10.%`",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turning one privilege off takes ALL back and keeps the rest: %q", got)
	}
	same := model.RoleSpec{Role: *from}
	if got, _ := (Dialect{}).RoleSQL(from, same, nil, false); len(got) != 0 {
		t.Fatalf("nothing changed: %q", got)
	}
}

func TestMariaDBRoles(t *testing.T) {
	d := Dialect{mariaDB: true}
	role := model.Role{ID: "`readers`", Name: "readers", Flags: on(), ConnLimit: -1}
	to := model.RoleSpec{Role: role}
	to.Flags = on("full")
	got, _ := d.RoleSQL(&role, to, nil, false)
	if !reflect.DeepEqual(got, []string{"GRANT ALL PRIVILEGES ON *.* TO `readers`"}) {
		t.Fatalf("got %q", got)
	}
	if d.DropRoleSQL(role) != "DROP ROLE `readers`" || d.DropRoleSQL(model.Role{ID: "`a`@`%`"}) != "DROP USER `a`@`%`" {
		t.Fatal("drop")
	}
	if d.account("a`b", "%") != "`a``b`@`%`" {
		t.Fatal(d.account("a`b", "%"))
	}
}
