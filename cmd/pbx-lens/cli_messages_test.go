package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/identity"
)

// openExistingErr runs the production open guard and returns its error,
// failing the test when the guard lets the DSN through.
func openExistingErr(t *testing.T, dsn string) error {
	t.Helper()

	ident, err := identity.OpenIdentity(t.Context(), identity.IdentityConfig{
		DSN:       dsn,
		RPID:      exportTestRPID,
		RPOrigins: []string{exportTestOrigin},
		Policy:    identity.OpenExisting,
	})
	if err == nil {
		_ = ident.Close()

		t.Fatalf("OpenIdentity(OpenExisting) let %s through", dsn)
	}

	return err
}

// TestIdentityExportErrorsTellTheOperatorWhatToDo renders the REAL errors
// the verb produces through the same renderer fatalCode uses. The library's
// fallback ("Check your input and try again") is wrong advice for a wrong
// file; every code here must name the file, say what happened, and point at
// the fix — with every placeholder filled.
func TestIdentityExportErrorsTellTheOperatorWhatToDo(t *testing.T) {
	registerCLITemplates()

	missing := filepath.Join(t.TempDir(), "identity.db")

	index := filepath.Join(t.TempDir(), "calls.db")

	_, cleanup, err := cli.OpenSystem(t.Context(), index)
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	notSQLite := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(notSQLite, []byte("this is not a database\n"), 0o600); err != nil {
		t.Fatalf("seed the non-database file: %v", err)
	}

	// The artifact failure needs a real identity database to get past the
	// guard; its output directory does not exist.
	seat := filepath.Join(t.TempDir(), "identity.db")

	ident, _ := openRegisteredIdentity(t, seat, "operator@pbx.artmann.tech")

	if err := ident.Close(); err != nil {
		t.Fatalf("close identity: %v", err)
	}

	_, artifactErr := runIdentityExport(
		t.Context(),
		seat,
		filepath.Join(t.TempDir(), "no-such-dir", "users.json"),
		exportTestRPID,
		exportTestOrigin,
	)
	if artifactErr == nil {
		t.Fatal("an export into a missing directory must fail")
	}

	tests := []struct {
		name  string
		err   error
		wants []string
	}{
		{
			name: "missing -auth-db",
			err:  errMissingAuthDB("identity-export"),
			wants: []string{
				"identity-export reads",
				"-auth-db <identity.db>",
				"pbx-lens identity-export -h",
			},
		},
		{
			name:  "missing -out",
			err:   errMissingOut("backup"),
			wants: []string{"backup writes a file", "-out <path>"},
		},
		{
			name:  "missing database",
			err:   openExistingErr(t, missing),
			wants: []string{missing, "never creates", "-auth-db"},
		},
		{
			name:  "index database",
			err:   openExistingErr(t, index),
			wants: []string{index, "not an identity database", "Nothing was written", "calls.db"},
		},
		{
			name:  "not a SQLite file",
			err:   openExistingErr(t, notSQLite),
			wants: []string{notSQLite, "cannot be read as a SQLite database", "permissions"},
		},
		{
			name:  "unwritable artifact",
			err:   artifactErr,
			wants: []string{"no-such-dir", "could not be written", "was not changed"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var console bytes.Buffer

			if code := errorfamily.HandleErrorWithConfig(
				test.err,
				errorfamily.HandleConfig{Output: &console},
			); code != 1 {
				t.Errorf("exit code = %d, want 1 (a rejection)", code)
			}

			rendered := console.String()

			for _, want := range test.wants {
				if !strings.Contains(rendered, want) {
					t.Errorf("console text lacks %q:\n%s", want, rendered)
				}
			}

			if strings.Contains(rendered, "Check your input") {
				t.Errorf("console text fell back to the family default:\n%s", rendered)
			}

			for _, placeholder := range []string{"{dsn}", "{cause}", "{out}", "{command}"} {
				if strings.Contains(rendered, placeholder) {
					t.Errorf(
						"placeholder %s left unfilled — a context key drifted:\n%s",
						placeholder,
						rendered,
					)
				}
			}
		})
	}
}
