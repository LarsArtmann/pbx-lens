package web

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestTemplGeneratedInSync guards the codegen-drift class (plan T6): a
// .templ edit without a following `templ generate` leaves *_templ.go stale,
// and because this repo GITIGNORES generated code, no diff-based CI step can
// see the drift — only a check against the live sources can. CI inherits the
// guard through the test lane; the writing `go tool templ generate` CI step
// runs before it, so a green lane means the sources and generated code
// agree. Skip under -short (pure codegen, no behavior under test).
func TestTemplGeneratedInSync(t *testing.T) {
	if testing.Short() {
		t.Skip("codegen check skipped in -short mode")
	}

	root := moduleRoot(t)

	cmd := exec.Command("go", "tool", "templ", "generate", "-check", "./internal/web/")
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code is stale; run: go tool templ generate ./internal/web/\n%s", out)
	}
}

// moduleRoot walks up to the directory containing go.mod (tests run with the
// package dir as cwd, and `go tool` resolves the tool directive from there,
// but anchoring at the module root keeps the check stable if the package
// ever moves).
func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test directory")
		}

		dir = parent
	}
}
