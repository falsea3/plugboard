package db

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/relay-client/relay-db/apps/desktop/internal/"

var shared = []string{".", "dialect", "sqltext", "sqlcore", "dbtest"}

var driverImports = []string{"github.com/jackc/pgx", "github.com/go-sql-driver/mysql", "modernc.org/sqlite"}

func TestLayout(t *testing.T) {
	var engines []string
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() && !slices.Contains(shared, e.Name()) && e.Name() != "engines" {
			engines = append(engines, e.Name())
		}
	}
	if len(engines) == 0 {
		t.Fatal("found no engine packages")
	}
	isEngine := func(path string) bool {
		rest, ok := strings.CutPrefix(path, module+"db/")
		return ok && slices.Contains(engines, rest)
	}

	for _, dir := range shared {
		for file, f := range parseDir(t, dir) {
			for _, imp := range importsOf(f) {
				if isEngine(imp) || imp == module+"db/engines" || hasAnyPrefix(imp, driverImports) {
					t.Errorf("%s imports %s: shared code must not know the engines", file, imp)
				}
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == "model" && slices.Contains([]string{"Postgres", "MySQL", "SQLite"}, sel.Sel.Name) {
						t.Errorf("%s names model.%s: ask the dialect instead", file, sel.Sel.Name)
					}
				}
				return true
			})
		}
	}

	for _, engine := range engines {
		for file, f := range parseDir(t, engine) {
			for _, imp := range importsOf(f) {
				if imp == module+"db/sqlcore" || imp == module+"db/engines" || (isEngine(imp) && imp != module+"db/"+engine) {
					t.Errorf("%s imports %s: an engine only implements dialect.Dialect", file, imp)
				}
			}
		}
	}

	err = filepath.WalkDir("..", func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == ".." {
			return err
		}
		if filepath.Base(path) == "db" {
			return filepath.SkipDir
		}
		for file, f := range parseDir(t, path) {
			for _, imp := range importsOf(f) {
				if isEngine(imp) {
					t.Errorf("%s imports %s: use package engines", file, imp)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func parseDir(t *testing.T, dir string) map[string]*ast.File {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*ast.File{}
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		out[p] = f
	}
	return out
}

func importsOf(f *ast.File) []string {
	var out []string
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		out = append(out, path)
	}
	return out
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
