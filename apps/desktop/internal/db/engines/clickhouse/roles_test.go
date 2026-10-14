package clickhouse

import (
	"reflect"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestUserSQL(t *testing.T) {
	to := model.RoleSpec{Role: model.Role{Name: "app", Flags: map[string]bool{"full": true}, MemberOf: []string{"role:readers"}}, Password: `it's\`}
	got, err := Dialect{}.RoleSQL(nil, to, nil, false)
	want := []string{`CREATE USER "app" IDENTIFIED WITH sha256_password BY 'it\'s\\'`, `GRANT ALL ON *.* TO "app"`, `GRANT "readers" TO "app"`}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%q %v", got, err)
	}
	preview, _ := Dialect{}.RoleSQL(nil, to, nil, true)
	if preview[0] != `CREATE USER "app" IDENTIFIED WITH sha256_password BY '********'` {
		t.Fatal(preview[0])
	}
	if _, err := (Dialect{}).RoleSQL(nil, model.RoleSpec{Role: model.Role{Name: "x"}}, nil, false); err == nil {
		t.Fatal("a new user needs a password")
	}

	role := &model.Role{ID: "role:readers", Name: "readers", Kind: "role", Flags: map[string]bool{"full": true}}
	change := model.RoleSpec{Role: *role}
	change.Name = "viewers"
	change.Flags = map[string]bool{}
	got, _ = Dialect{}.RoleSQL(role, change, nil, false)
	if want := []string{`ALTER ROLE "readers" RENAME TO "viewers"`, `REVOKE ALL ON *.* FROM "viewers"`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	change.Password = "x"
	if _, err := (Dialect{}).RoleSQL(role, change, nil, false); err == nil {
		t.Fatal("a role has no password")
	}
	if (Dialect{}).RoleID(role, change) != "role:viewers" || (Dialect{}).DropRoleSQL(*role) != `DROP ROLE "readers"` || (Dialect{}).DropRoleSQL(model.Role{Name: "a"}) != `DROP USER "a"` {
		t.Fatal("ids and drops")
	}
}
