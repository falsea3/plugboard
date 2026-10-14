package postgres

import (
	"reflect"
	"strings"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestCreateRoleSQL(t *testing.T) {
	to := model.RoleSpec{Role: model.Role{Name: "app", Flags: flags("login", "inherit"), ConnLimit: 5, ValidUntil: "2027-01-01", MemberOf: []string{"readers"}}, Password: "secret"}
	got, err := Dialect{}.RoleSQL(nil, to, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`CREATE ROLE "app" WITH LOGIN CONNECTION LIMIT 5 VALID UNTIL '2027-01-01' PASSWORD '********'`,
		`GRANT "readers" TO "app"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	got, _ = Dialect{}.RoleSQL(nil, model.RoleSpec{Role: model.Role{Name: "plain", Flags: flags("inherit"), ConnLimit: -1}}, nil, true)
	if !reflect.DeepEqual(got, []string{`CREATE ROLE "plain"`}) {
		t.Fatalf("defaults: %q", got)
	}
	got, _ = Dialect{}.RoleSQL(nil, to, nil, false)
	if !strings.HasPrefix(got[0], `CREATE ROLE "app" WITH LOGIN CONNECTION LIMIT 5 VALID UNTIL '2027-01-01' PASSWORD 'SCRAM-SHA-256$4096:`) || strings.Contains(got[0], "secret") {
		t.Fatalf("the password goes as a SCRAM verifier: %q", got[0])
	}
}

func TestAlterRoleSQL(t *testing.T) {
	from := &model.Role{Name: "app", Flags: flags("login", "inherit"), ConnLimit: -1, ValidUntil: "2027-01-01 00:00:00+00", MemberOf: []string{"readers", "writers"}}
	to := model.RoleSpec{Role: *from, ClearPassword: true}
	to.Name = `new "app"`
	to.Flags = flags("inherit", "createDB")
	to.ValidUntil = ""
	to.MemberOf = []string{"writers", "admins"}
	got, err := Dialect{}.RoleSQL(from, to, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`ALTER ROLE "app" RENAME TO "new ""app"""`,
		`ALTER ROLE "new ""app""" WITH NOLOGIN CREATEDB VALID UNTIL 'infinity' PASSWORD NULL`,
		`GRANT "admins" TO "new ""app"""`,
		`REVOKE "readers" FROM "new ""app"""`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	same := model.RoleSpec{Role: *from}
	if got, _ := (Dialect{}).RoleSQL(from, same, nil, false); len(got) != 0 {
		t.Fatalf("nothing changed: %q", got)
	}
}

func TestRoleSQLRefuses(t *testing.T) {
	for _, to := range []model.RoleSpec{{Role: model.Role{Name: "  "}}, {Role: model.Role{Name: "a", ConnLimit: -2}}} {
		if _, err := (Dialect{}).RoleSQL(nil, to, nil, true); err == nil {
			t.Fatalf("%+v passed", to)
		}
	}
}

func TestLiteral(t *testing.T) {
	if got := literal(`it's`); got != `'it''s'` {
		t.Fatal(got)
	}
	if got := literal(`a\b'`); got != `E'a\\b'''` {
		t.Fatal(got)
	}
}

func TestScramVerifier(t *testing.T) {
	got, err := scramWithSalt("pencil", []byte("0123456789abcdef"), 4096)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "SCRAM-SHA-256$4096:MDEyMzQ1Njc4OWFiY2RlZg==$") || strings.Count(got, ":") != 2 {
		t.Fatal(got)
	}
	again, _ := scramWithSalt("pencil", []byte("0123456789abcdef"), 4096)
	other, _ := scramVerifier("pencil")
	if got != again || got == other {
		t.Fatal("same salt gives the same verifier, a new one a new salt")
	}
}

func flags(on ...string) map[string]bool {
	out := map[string]bool{}
	for _, k := range on {
		out[k] = true
	}
	return out
}
