package main

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// TestFatalCodeFamilyMapping pins the CLI boundary contract: family-classified
// errors map through the library's sysexits table (Transient = 75
// EX_TEMPFAIL, e.g. the projection-timeout fatal), while non-family errors
// keep the legacy exit 1 — the registry fail-opens unclassified errors to
// Transient, and a plain bug must never read as "try again later".
func TestFatalCodeFamilyMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"rejection stays 1", errorfamily.NewRejection("lens.cli.missing_db", "needs -db"), 1},
		{"projection timeout exits 75", errorfamily.NewTransient("lens.cli.projection_timeout", "did not converge"), 75},
		{"infrastructure exits 69", errorfamily.WrapInfrastructure(errors.New("no index"), "lens.cli.catalog", "x"), 69},
		{"corruption exits 65", errorfamily.NewCorruption("lens.cli.corrupt", "bad"), 65},
		{"non-family keeps legacy 1", errors.New("plain bug"), 1},
		{"wrapped non-family keeps 1", errorfamily.Compose(errors.New("a"), errors.New("b")), 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fatalCode(tc.err); got != tc.want {
				t.Errorf("fatalCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
