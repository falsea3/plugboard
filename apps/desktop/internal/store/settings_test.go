package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestSettingsRoundTrip(t *testing.T) {
	s := NewSettings(t.TempDir())
	if got := s.Load(); got != model.DefaultSettings() {
		t.Fatalf("fresh load = %+v, want defaults", got)
	}
	want := model.Settings{Theme: model.ThemeDark, PageSize: 1000, EditorFontSize: 15, ConfirmProdWrites: false}
	if _, err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); got != want {
		t.Fatalf("load = %+v, want %+v", got, want)
	}
}

func TestSettingsSanitizesBadValues(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"neon","pageSize":3,"editorFontSize":99,"confirmProdWrites":false}`), 0o600)
	got := NewSettings(dir).Load()
	d := model.DefaultSettings()
	if got.Theme != d.Theme || got.PageSize != d.PageSize || got.EditorFontSize != d.EditorFontSize || got.ConfirmProdWrites {
		t.Fatalf("got %+v", got)
	}
}

func TestSettingsCorruptFileFallsBack(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{nope`), 0o600)
	if got := NewSettings(dir).Load(); got != model.DefaultSettings() {
		t.Fatalf("got %+v", got)
	}
}
