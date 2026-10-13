package cassandra

import (
	"reflect"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestRolePlan(t *testing.T) {
	to := model.RoleSpec{Role: model.Role{Name: "app's", Flags: map[string]bool{"login": true}, MemberOf: []string{"readers"}}, Password: "pw'1"}
	got, _ := rolePlan(nil, to, false)
	want := []string{"CREATE ROLE 'app''s' WITH PASSWORD = 'pw''1' AND LOGIN = true AND SUPERUSER = false", "GRANT 'readers' TO 'app''s'"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	got, _ = rolePlan(nil, to, true)
	if got[0] != "CREATE ROLE 'app''s' WITH PASSWORD = '********' AND LOGIN = true AND SUPERUSER = false" {
		t.Fatal(got[0])
	}
	from := &model.Role{ID: "app", Name: "app", Flags: map[string]bool{"login": true}, MemberOf: []string{"readers"}}
	change := model.RoleSpec{Role: *from}
	change.Flags = map[string]bool{"superuser": true}
	change.MemberOf = nil
	got, _ = rolePlan(from, change, false)
	want = []string{"ALTER ROLE 'app' WITH LOGIN = false AND SUPERUSER = true", "REVOKE 'readers' FROM 'app'"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	if got, _ := rolePlan(from, model.RoleSpec{Role: *from}, false); len(got) != 0 {
		t.Fatalf("nothing changed: %q", got)
	}
	renamed := model.RoleSpec{Role: *from}
	renamed.Name = "b"
	if _, err := rolePlan(from, renamed, false); err == nil {
		t.Fatal("rename")
	}
}

func TestCQLResource(t *testing.T) {
	for in, want := range map[string]string{"<all keyspaces>": "ALL KEYSPACES", "<keyspace shop>": "KEYSPACE shop", "<table shop.orders>": "TABLE shop.orders", "<role app>": "ROLE app", "odd": "odd"} {
		if got := cqlResource(in); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}
