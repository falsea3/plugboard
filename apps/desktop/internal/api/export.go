package api

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var exportKinds = map[string]runtime.FileFilter{
	".png": {DisplayName: "PNG image", Pattern: "*.png"},
	".svg": {DisplayName: "SVG image", Pattern: "*.svg"},
	".csv": {DisplayName: "CSV table", Pattern: "*.csv"},
}

func exportPath(name, chosen string) (string, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := exportKinds[ext]; !ok {
		return "", apperr.New("bad_value", "export as .png, .svg or .csv")
	}
	if chosen != "" && strings.ToLower(filepath.Ext(chosen)) != ext {
		chosen += ext
	}
	return chosen, nil
}

func (a *App) SaveExport(name string, data []byte) (string, error) {
	if _, err := exportPath(name, ""); err != nil {
		return "", err
	}
	chosen, err := runtime.SaveFileDialog(a.context(), runtime.SaveDialogOptions{
		Title:                "Export",
		DefaultFilename:      filepath.Base(name),
		Filters:              []runtime.FileFilter{exportKinds[strings.ToLower(filepath.Ext(name))]},
		CanCreateDirectories: true,
	})
	if err != nil || chosen == "" {
		return "", err
	}
	path, err := exportPath(name, chosen)
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}
