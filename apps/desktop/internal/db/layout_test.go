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

const module = "github.com/relay-client/plugboard/apps/desktop/internal/"

var driverImports = []string{"github.com/jackc/pgx", "github.com/go-sql-driver/mysql", "modernc.org/sqlite", "github.com/redis/go-redis", "github.com/ClickHouse/clickhouse-go", "github.com/apache/cassandra-gocql-driver"}

func TestLayout(t *testing.T) {
	for _, e := range readDir(t, ".") {
		if e.IsDir() && e.Name() != "engines" && e.Name() != "sql" {
			t.Errorf("internal/db/%s: the root holds package db, engines/ and sql/ only", e.Name())
		}
	}
	for _, e := range readDir(t, "sql") {
		if !e.IsDir() {
			t.Errorf("internal/db/sql/%s: sql/ holds packages only", e.Name())
		}
	}
	var engines []string
	for _, e := range readDir(t, "engines") {
		if e.IsDir() {
			engines = append(engines, e.Name())
		}
	}
	if len(engines) == 0 {
		t.Fatal("found no engine packages in internal/db/engines")
	}
	isEngine := func(path string) bool {
		name, ok := strings.CutPrefix(path, module+"db/engines/")
		return ok && slices.Contains(engines, name)
	}
	drivers := driverNames(t)

	shared := []string{"."}
	for _, e := range readDir(t, "sql") {
		if e.IsDir() {
			shared = append(shared, filepath.Join("sql", e.Name()))
		}
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
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == "model" && slices.Contains(drivers, sel.Sel.Name) {
						t.Errorf("%s names model.%s: ask the dialect instead", file, sel.Sel.Name)
					}
				}
				return true
			})
		}
	}

	for _, engine := range engines {
		for file, f := range parseDir(t, filepath.Join("engines", engine)) {
			for _, imp := range importsOf(f) {
				switch {
				case imp == module+"db/engines",
					imp == module+"db/sql/sqlcore",
					imp == module+"db/sql/sqltest",
					isEngine(imp) && imp != module+"db/engines/"+engine:
					t.Errorf("%s imports %s: an engine implements its contract and nothing more", file, imp)
				}
			}
		}
	}

	err := filepath.WalkDir("..", func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == ".." {
			return err
		}
		if filepath.Base(path) == "db" {
			return filepath.SkipDir
		}
		for file, f := range parseDir(t, path) {
			for _, imp := range importsOf(f) {
				if strings.HasPrefix(imp, module+"db/") && imp != module+"db/engines" {
					t.Errorf("%s imports %s: outside internal/db use package db and package engines only", file, imp)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func readDir(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func driverNames(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, f := range parseDir(t, filepath.Join("..", "model")) {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				if id, ok := vs.Type.(*ast.Ident); ok && id.Name == "Driver" {
					for _, n := range vs.Names {
						out = append(out, n.Name)
					}
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("found no model.Driver constants")
	}
	return out
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
