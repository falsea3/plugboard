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
