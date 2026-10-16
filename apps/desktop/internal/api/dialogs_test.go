package api

import (
	"strings"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestDialogFiltersAreSingleExtensions(t *testing.T) {
	all := append(append([]runtime.FileFilter{}, dumpFilters...), sqliteFilters...)
	for _, f := range exportKinds {
		all = append(all, f)
	}
	for _, f := range saveFilters {
		all = append(all, f)
	}
	for _, f := range all {
		for _, p := range strings.Split(f.Pattern, ";") {
			ext, ok := strings.CutPrefix(strings.TrimSpace(p), "*.")
			if p == "*" {
				continue
			}
			if !ok || ext == "" || strings.ContainsAny(ext, ".*") {
				t.Errorf("%s: pattern %q — macOS has no file type for it and the dialog crashes", f.DisplayName, p)
			}
		}
	}
}

func TestDumpPath(t *testing.T) {
	for _, c := range []struct {
		name, chosen, want string
		gz                 bool
	}{
		{"shop.sql", "/x/shop", "/x/shop.sql", false},
		{"shop.sql", "/x/shop.sql.gz", "/x/shop.sql.gz", true},
		{"shop.dump", "/x/shop.backup", "/x/shop.backup", false},
		{"shop.dump", "/x/shop.gz", "/x/shop.gz.dump", false},
		{"shop.tar", "/x/shop", "/x/shop.tar", false},
	} {
		got, gz := dumpPath(c.name, c.chosen)
		if got != c.want || gz != c.gz {
			t.Errorf("dumpPath(%q, %q) = %q, %v", c.name, c.chosen, got, gz)
		}
	}
}
