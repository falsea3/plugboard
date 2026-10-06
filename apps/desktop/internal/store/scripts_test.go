package store

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestScriptsCreateListReadWrite(t *testing.T) {
	s := NewScripts(t.TempDir())
	if names, err := s.List("c1"); err != nil || len(names) != 0 {
		t.Fatalf("empty list = %v, %v", names, err)
	}
	for _, n := range []string{"orders", "Audit", "b report"} {
		if err := s.Create("c1", n, "select "+n); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Create("c1", "orders", "x"); !errors.Is(err, ErrScriptExists) {
		t.Fatalf("duplicate create = %v", err)
	}
	names, _ := s.List("c1")
	if !slices.Equal(names, []string{"Audit", "b report", "orders"}) {
		t.Fatalf("list = %v", names)
	}
	if err := s.Write("c1", "orders", "select 2"); err != nil {
		t.Fatal(err)
	}
	if text, _ := s.Read("c1", "orders"); text != "select 2" {
		t.Fatalf("read = %q", text)
	}
	if _, err := s.Read("c1", "nope"); !errors.Is(err, ErrNoScript) {
		t.Fatalf("missing read = %v", err)
	}
	if names, _ := s.List("c2"); len(names) != 0 {
		t.Fatalf("other connection sees %v", names)
	}
}

func TestScriptsRenameDelete(t *testing.T) {
	s := NewScripts(t.TempDir())
	_ = s.Create("c1", "a", "1")
	_ = s.Create("c1", "b", "2")
	if err := s.Rename("c1", "a", "b"); !errors.Is(err, ErrScriptExists) {
		t.Fatalf("rename onto existing = %v", err)
	}
	if err := s.Rename("c1", "a", "A"); err != nil {
		t.Fatalf("case-only rename = %v", err)
	}
	if err := s.Rename("c1", "zz", "y"); !errors.Is(err, ErrNoScript) {
		t.Fatalf("rename missing = %v", err)
	}
	if err := s.Delete("c1", "b"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("c1", "b"); err != nil {
		t.Fatalf("second delete = %v", err)
	}
	if names, _ := s.List("c1"); !slices.Equal(names, []string{"A"}) {
		t.Fatalf("list = %v", names)
	}
}

func TestScriptNamesAndConnectionsAreChecked(t *testing.T) {
	s := NewScripts(t.TempDir())
	for _, n := range []string{"", " a", "a ", ".tabs", "../x", `a\b`, "a:b", "a\nb", string(make([]byte, 121))} {
		if err := s.Create("c1", n, "x"); err == nil {
			t.Errorf("name %q accepted", n)
		}
	}
	for _, c := range []string{"", "..", "a/b", "."} {
		if _, err := s.List(c); err == nil {
			t.Errorf("connection %q accepted", c)
		}
	}
}

func TestQueryTabsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewScripts(dir)
	_ = s.Create("c1", "kept", "select 1")
	_ = s.Create("c1", "gone", "select 2")
	tabs := []model.QueryTab{
		{Title: "Query 1", SQL: "draft", Saved: "dr"},
		{Title: "kept", SQL: "select 1 -- edited", Script: "kept", Saved: "stale", Active: true},
		{Title: "gone", SQL: "select 2", Script: "gone", Saved: "select 2"},
	}
	if err := s.SaveTabs("c1", tabs); err != nil {
		t.Fatal(err)
	}
	_ = s.Delete("c1", "gone")
	got, err := s.Tabs("c1")
	if err != nil {
		t.Fatal(err)
	}
	want := []model.QueryTab{
		{Title: "Query 1", SQL: "draft", Saved: "dr"},
		{Title: "kept", SQL: "select 1 -- edited", Script: "kept", Saved: "select 1", Active: true},
		{Title: "gone", SQL: "select 2"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("tabs = %+v", got)
	}
	if names, _ := s.List("c1"); !slices.Equal(names, []string{"kept"}) {
		t.Fatalf("tabs file listed as a script: %v", names)
	}
	if err := s.ForgetTabs("c1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Tabs("c1"); len(got) != 0 {
		t.Fatalf("after forget = %v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "c1", "kept.sql")); err != nil {
		t.Fatalf("forget removed scripts: %v", err)
	}
}

func TestQueryTabsSurviveABrokenFile(t *testing.T) {
	dir := t.TempDir()
	s := NewScripts(dir)
	_ = os.MkdirAll(filepath.Join(dir, "scripts", "c1"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "scripts", "c1", tabsFile), []byte("{nope"), 0o600)
	if got, err := s.Tabs("c1"); err != nil || len(got) != 0 {
		t.Fatalf("broken file = %v, %v", got, err)
	}
}
