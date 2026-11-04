package main

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	errPlainBug       = errors.New("plain bug")
	errPlainSecond    = errors.New("second plain bug")
	errPlainNoBackend = errors.New("no index")
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
		{
			name: "projection timeout exits 75",
			err:  errorfamily.NewTransient("lens.cli.projection_timeout", "did not converge"),
			want: 75,
		},
		{
			name: "infrastructure exits 69",
			err:  errorfamily.WrapInfrastructure(errPlainNoBackend, "lens.cli.catalog", "x"),
			want: 69,
		},
		{"corruption exits 65", errorfamily.NewCorruption("lens.cli.corrupt", "bad"), 65},
		{"non-family keeps legacy 1", errPlainBug, 1},
		{"wrapped non-family keeps 1", errorfamily.Compose(errPlainBug, errPlainSecond), 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fatalCode(tc.err); got != tc.want {
				t.Errorf("fatalCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
