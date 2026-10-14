package redis_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
	goredis "github.com/redis/go-redis/v9"
)

func TestACLUsers(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	t.Cleanup(func() { s.DropRole(ctx, "plugboard_app") })
	spec := model.RoleSpec{Role: model.Role{Name: "plugboard_app", Flags: map[string]bool{}, Rules: "~cache:* +get +set -@dangerous"}, Password: "pw 1"}
	if id, err := s.SaveRole(ctx, "", spec); err != nil || id != "plugboard_app" {
		t.Fatal(id, err)
	}
	list, err := s.Roles(ctx)
	if err != nil || list.Form.Rules == "" || list.Form.Membership {
		t.Fatalf("%+v %v", list.Form, err)
	}
	var user model.Role
	for _, r := range list.Roles {
		if r.Name == "plugboard_app" {
			user = r
		}
		if r.Name == "default" && !r.Current {
			t.Fatal("the session logs in as default")
		}
	}
	if user.Flags["disabled"] || !strings.Contains(user.Rules, "~cache:*") {
		t.Fatalf("user = %+v", user)
	}

	addr := os.Getenv("PLUGBOARD_TEST_REDIS")
	as := goredis.NewClient(&goredis.Options{Addr: addr, Username: "plugboard_app", Password: "pw 1", DB: 0})
	defer as.Close()
	if err := as.Set(ctx, "cache:plugboard", "1", 0).Err(); err != nil {
		t.Fatalf("allowed key: %v", err)
	}
	if err := as.Set(ctx, "other:plugboard", "1", 0).Err(); err == nil || !strings.Contains(err.Error(), "NOPERM") {
		t.Fatalf("a key outside ~cache:*: %v", err)
	}
	as.Close()
	s.Run(ctx, "SELECT 0\nDEL cache:plugboard")

	next := model.RoleSpec{Role: user}
	next.Flags = map[string]bool{"disabled": true, "nopass": true}
	next.Rules = "~* +@read"
	preview, err := s.RoleSQL(ctx, user.ID, next)
	if err != nil || len(preview) != 1 || !strings.HasPrefix(preview[0], "ACL SETUSER plugboard_app off") {
		t.Fatalf("preview = %q, %v", preview, err)
	}
	if _, err := s.SaveRole(ctx, user.ID, next); err != nil {
		t.Fatal(err)
	}
	list, _ = s.Roles(ctx)
	for _, r := range list.Roles {
		if r.Name == "plugboard_app" && (!r.Flags["disabled"] || !r.Flags["nopass"] || strings.Contains(r.Rules, "cache")) {
			t.Fatalf("after the change: %+v", r)
		}
	}
	if err := s.DropRole(ctx, "plugboard_app"); err != nil {
		t.Fatal(err)
	}
}

func TestLiveSettings(t *testing.T) {
	s := open(t, false)
	ctx := context.Background()
	files, err := s.ConfigFiles(ctx)
	if err != nil || files[0].Name != "Live settings" {
		t.Fatalf("%+v %v", files, err)
	}
	live, err := s.ReadConfig(ctx, files[0].Path)
	if err != nil || live.Via != "live" || !strings.Contains(live.Text, "\nmaxmemory-policy ") {
		t.Fatalf("%.200q %v", live.Text, err)
	}
	t.Cleanup(func() { s.WriteConfig(ctx, files[0].Path, live.Text) })
	lines := strings.Count(live.Text, "\n")
	changed := strings.Replace(live.Text, "\nmaxmemory-policy noeviction\n", "\nmaxmemory-policy allkeys-lru\n", 1) + "no-such-setting 1\n"
	saved, err := s.WriteConfig(ctx, files[0].Path, changed)
	if err != nil || saved.Reloaded || len(saved.Problems) != 1 || saved.Problems[0].Line != lines+1 || saved.Note == "" {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	again, _ := s.ReadConfig(ctx, files[0].Path)
	if !strings.Contains(again.Text, "\nmaxmemory-policy allkeys-lru\n") {
		t.Fatal("the good line was applied")
	}
	if _, err := s.ReadConfig(ctx, "/etc/redis/redis.conf"); err == nil || !strings.Contains(err.Error(), "SSH") {
		t.Fatalf("a file without SSH: %v", err)
	}
}
