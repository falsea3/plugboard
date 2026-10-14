package redis

import (
	"reflect"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestParseACL(t *testing.T) {
	r, ok := parseACL("user app on sanitize-payload #2d71 ~cache:* resetchannels -@all +@read (~sel:* resetchannels -@all +get)")
	if !ok || r.ID != "app" || r.Flags["disabled"] || r.Flags["nopass"] || r.Rules != "~cache:* resetchannels -@all +@read (~sel:* resetchannels -@all +get)" {
		t.Fatalf("%+v", r)
	}
	r, _ = parseACL("user ghost off nopass")
	if !r.Flags["disabled"] || !r.Flags["nopass"] || r.Rules != "" {
		t.Fatalf("%+v", r)
	}
	if _, ok := parseACL("nonsense"); ok {
		t.Fatal("not a user line")
	}
}

func TestACLPlan(t *testing.T) {
	to := model.RoleSpec{Role: model.Role{Name: "app", Flags: map[string]bool{}, Rules: "~cache:*  +@read (~a +get)"}, Password: "s3 cret"}
	got, _ := aclPlan(nil, to, false)
	if want := []string{"ACL", "SETUSER", "app", "on", "resetpass", ">s3 cret", "~cache:*", "+@read", "(~a +get)"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("create: %q", got)
	}
	got, _ = aclPlan(nil, to, true)
	if shown(got) != `ACL SETUSER app on resetpass >******** ~cache:* +@read (~a +get)` {
		t.Fatalf("preview: %s", shown(got))
	}
	from := &model.Role{ID: "app", Name: "app", Flags: map[string]bool{"nopass": true}, Rules: "~* +@all"}
	change := model.RoleSpec{Role: *from}
	change.Flags = map[string]bool{"disabled": true}
	change.Rules = "~* +@read"
	got, _ = aclPlan(from, change, false)
	if want := []string{"ACL", "SETUSER", "app", "off", "resetpass", "clearselectors", "resetkeys", "resetchannels", "nocommands", "~*", "+@read"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("alter: %q", got)
	}
	if got, _ := aclPlan(from, model.RoleSpec{Role: *from}, false); got != nil {
		t.Fatalf("nothing changed: %q", got)
	}
	renamed := model.RoleSpec{Role: *from}
	renamed.Name = "other"
	for _, bad := range []model.RoleSpec{renamed, {Role: model.Role{Name: "a b"}}} {
		if _, err := aclPlan(from, bad, false); err == nil {
			t.Fatalf("%+v passed", bad)
		}
	}
}

func TestLiveSettings(t *testing.T) {
	text := liveText(map[string]string{"save": "3600 1 300 100", "maxmemory": "0", "dir": ""})
	if text != "dir \"\"\nmaxmemory 0\nsave 3600 1 300 100\n" {
		t.Fatalf("%q", text)
	}
	got := liveSettingsOf(liveHeader + text + "\n# a comment\nMaxmemory-Policy  allkeys-lru\n")
	want := []setting{{2, "dir", ""}, {3, "maxmemory", "0"}, {4, "save", "3600 1 300 100"}, {7, "maxmemory-policy", "allkeys-lru"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}
