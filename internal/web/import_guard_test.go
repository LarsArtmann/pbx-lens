package web

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkFileImports fails the test when a production file imports an
// internal/ingest package.
func checkFileImports(t *testing.T, fset *token.FileSet, file string) {
	t.Helper()

	src, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}

	for _, imp := range src.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "/internal/ingest") {
			t.Errorf("%s imports %s — the web read side must not depend on ingest clients", file, path)
		}
	}
}

// prodGoFiles lists the package's production Go files: the directory's
// non-test files plus any sub-package's (a future web/<sub>/ directory
// cannot quietly reopen the edge).
func prodGoFiles(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		files = append(files, name)
	}

	subdirFiles, err := filepath.Glob("*/*.go")
	if err != nil {
		t.Fatalf("glob subdirs: %v", err)
	}

	for _, file := range subdirFiles {
		if !strings.HasSuffix(file, "_test.go") {
			files = append(files, file)
		}
	}

	return files
}

// The web read side must never import an ingest client (T10, 2026-11-20):
// the recording proxy speaks calls.RecordingMedia through its
// RecordingFetcher interface, and every other route is a query or a proxy
// over the index. Ingest belongs to the POLLER's side of the house; one
// web→ingest import would re-couple the UI to the API clients and let
// source-specific shapes leak into routes. Test files may still import
// ingest (they wire the real client against fake APIs).
func TestWebProdFilesNeverImportIngest(t *testing.T) {
	fset := token.NewFileSet()

	for _, file := range prodGoFiles(t) {
		checkFileImports(t, fset, file)
	}
}
