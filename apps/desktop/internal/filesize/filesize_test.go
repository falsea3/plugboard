package filesize

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const maxLines = 300

var skipDirs = map[string]bool{
	"node_modules":      true,
	"wailsjs":           true,
	"dist":              true,
	"bin":               true,
	"test-results":      true,
	"playwright-report": true,
}

var codeFiles = map[string]bool{
	".go":     true,
	".ts":     true,
	".svelte": true,
	".js":     true,
	".mjs":    true,
	".css":    true,
	".py":     true,
	".sh":     true,
	".yml":    true,
}

func TestNoFileIsTooLong(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	var long []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (skipDirs[name] || strings.HasPrefix(name, ".") && name != ".github") {
				return filepath.SkipDir
			}
			return nil
		}
		if !codeFiles[filepath.Ext(name)] && name != "Makefile" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if n := bytes.Count(data, []byte("\n")); n > maxLines {
			rel, _ := filepath.Rel(root, path)
			long = append(long, fmt.Sprintf("%s: %d lines", rel, n))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(long) > 0 {
		t.Errorf("%d files are longer than %d lines; split them:\n%s", len(long), maxLines, strings.Join(long, "\n"))
	}
}
