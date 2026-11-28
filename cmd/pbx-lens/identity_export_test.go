package main

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil/endpoints"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/identity"
	"github.com/larsartmann/pbx-lens/internal/identity/identitytest"
)

// The export's ceremony defaults: an export never runs a ceremony, so the
// dev values are all the provider construction needs.
const (
	exportTestRPID   = "localhost"
	exportTestOrigin = "http://localhost"
)

// openRegisteredIdentity boots a file-backed identity database and registers
// the operator through the REAL /auth/register walk (CSRF pair and all). It
// returns the live handle — the caller decides whether serve "keeps running"
// (the live-store test) or shuts down first (every other test).
func openRegisteredIdentity(t *testing.T, dsn, email string) (*identity.Identity, *http.ServeMux) {
	t.Helper()

	ident, err := identity.OpenIdentity(t.Context(), identity.IdentityConfig{
		DSN:       dsn,
		RPID:      exportTestRPID,
		RPOrigins: []string{exportTestOrigin},
	})
	if err != nil {
		t.Fatalf("OpenIdentity: %v", err)
	}

	mux := http.NewServeMux()
	ident.Register(mux, endpoints.NewRegistry())
	identitytest.Session(t, ident.Middleware()(mux), email)

	return ident, mux
}

// exportedEmails reads the artifact back as the user array a consumer gets.
func exportedEmails(t *testing.T, out string) []string {
	t.Helper()

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}

	var users []struct {
		Email string `json:"email"`
	}

	if err := json.Unmarshal(data, &users); err != nil {
		t.Fatalf("export is not a JSON user array: %v\n%s", err, data)
	}

	emails := make([]string, 0, len(users))
	for _, user := range users {
		emails = append(emails, user.Email)
	}

	return emails
}

func TestIdentityExportCarriesTheOperatorSeat(t *testing.T) {
	// The runbook's migration artifact (identity-first-deploy.md): a real
	// file-backed identity DB, the operator registered through the REAL
	// /auth/register walk, then the verb's export read back as JSON.
	dsn := filepath.Join(t.TempDir(), "identity.db")

	ident, _ := openRegisteredIdentity(t, dsn, "operator@pbx.artmann.tech")

	if err := ident.Close(); err != nil {
		t.Fatalf("close identity: %v", err)
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "users.json")

	users, err := runIdentityExport(t.Context(), dsn, out, exportTestRPID, exportTestOrigin)
	if err != nil {
		t.Fatalf("runIdentityExport: %v", err)
	}

	if users != 1 {
		t.Fatalf("reported users = %d, want 1", users)
	}

	if got := exportedEmails(t, out); !slices.Equal(got, []string{"operator@pbx.artmann.tech"}) {
		t.Fatalf("exported users = %v, want exactly the operator seat", got)
	}

	// The artifact is published by rename: no staging file survives.
	if left, err := os.ReadDir(dir); err != nil || len(left) != 1 {
		t.Fatalf(
			"artifact directory = %v (err %v), want only users.json — a staging file leaked",
			left,
			err,
		)
	}
}

// TestIdentityExportRefusesAMissingDatabase pins the guard that makes an
// empty "success" impossible: before it, a typo in the path bootstrapped a
// brand-new identity database and exported [] with exit 0.
func TestIdentityExportRefusesAMissingDatabase(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "does-not-exist.db")
	out := filepath.Join(t.TempDir(), "users.json")

	users, err := runIdentityExport(t.Context(), dsn, out, exportTestRPID, exportTestOrigin)
	if errorfamily.Code(err) != "lens.web.identity_db_missing" {
		t.Fatalf(
			"runIdentityExport on a missing file: users = %d, err = %v, want lens.web.identity_db_missing",
			users,
			err,
		)
	}

	if _, statErr := os.Stat(dsn); !os.IsNotExist(statErr) {
		t.Errorf("the refusal must not bootstrap the identity database: stat err = %v", statErr)
	}

	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("a refused export must write no artifact: stat err = %v", statErr)
	}
}

// TestIdentityExportRefusesAnIndexDatabase pins the mirror footgun: the
// identity opener creates its tables wherever it is pointed, so before the
// guard an index database gained eight identity tables and the verb still
// reported success.
func TestIdentityExportRefusesAnIndexDatabase(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "calls.db")

	_, cleanup, err := cli.OpenSystem(t.Context(), dsn)
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	before := identitytest.TableNames(t, dsn)

	users, err := runIdentityExport(
		t.Context(),
		dsn,
		filepath.Join(t.TempDir(), "users.json"),
		exportTestRPID,
		exportTestOrigin,
	)
	if errorfamily.Code(err) != "lens.web.identity_not_identity_db" {
		t.Fatalf(
			"runIdentityExport on an index database: users = %d, err = %v, want lens.web.identity_not_identity_db",
			users,
			err,
		)
	}

	if after := identitytest.TableNames(t, dsn); !slices.Equal(before, after) {
		t.Fatalf("the refusal wrote into the index database: %v -> %v", before, after)
	}
}

// TestIdentityExportWhileAnotherInstanceIsLive pins the realistic run: serve
// holds the identity database open, and the export boots a second, short-
// lived identity instance on the same file. Both must come out intact — the
// export carries the seat, and the live instance still serves it.
func TestIdentityExportWhileAnotherInstanceIsLive(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "identity.db")

	live, mux := openRegisteredIdentity(t, dsn, "operator@pbx.artmann.tech")

	t.Cleanup(func() {
		if cerr := live.Close(); cerr != nil {
			t.Errorf("live identity close: %v", cerr)
		}
	})

	users, err := runIdentityExport(
		t.Context(),
		dsn,
		filepath.Join(t.TempDir(), "users.json"),
		exportTestRPID,
		exportTestOrigin,
	)
	if err != nil {
		t.Fatalf("runIdentityExport while serve is live: %v", err)
	}

	if users != 1 {
		t.Fatalf("exported users = %d, want the live seat", users)
	}

	if got := live.OpsSnapshot(t.Context()).Users; got != 1 {
		t.Errorf("live instance sees %d users after the export, want 1", got)
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("live /login after the export = %d, want 200", recorder.Code)
	}
}

// TestIdentityExportReportsAnEmptySeatHonestly pins the zero-user case: a
// real identity database nobody registered into exports [] AND reports 0,
// so the verb can say so instead of printing a bare success line.
func TestIdentityExportReportsAnEmptySeatHonestly(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "identity.db")

	ident, err := identity.OpenIdentity(t.Context(), identity.IdentityConfig{
		DSN:       dsn,
		RPID:      exportTestRPID,
		RPOrigins: []string{exportTestOrigin},
	})
	if err != nil {
		t.Fatalf("OpenIdentity: %v", err)
	}

	if err := ident.Close(); err != nil {
		t.Fatalf("close identity: %v", err)
	}

	out := filepath.Join(t.TempDir(), "users.json")

	users, err := runIdentityExport(t.Context(), dsn, out, exportTestRPID, exportTestOrigin)
	if err != nil {
		t.Fatalf("runIdentityExport on an empty seat: %v", err)
	}

	if users != 0 {
		t.Fatalf("reported users = %d, want 0", users)
	}

	if got := exportedEmails(t, out); len(got) != 0 {
		t.Fatalf("exported users = %v, want none", got)
	}
}

// TestIdentityExportKeepsThePreviousArtifactWhenRefused pins the artifact
// discipline from the failure side: a refused export touches nothing, so
// the artifact of the last good export survives to be compared or reused.
func TestIdentityExportKeepsThePreviousArtifactWhenRefused(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "users.json")

	previous := []byte(`[{"email":"previous@example.com"}]`)
	if err := os.WriteFile(out, previous, 0o600); err != nil {
		t.Fatalf("seed the previous artifact: %v", err)
	}

	dsn := filepath.Join(t.TempDir(), "does-not-exist.db")

	if _, err := runIdentityExport(
		t.Context(),
		dsn,
		out,
		exportTestRPID,
		exportTestOrigin,
	); err == nil {
		t.Fatal("runIdentityExport on a missing file must fail")
	}

	if got, err := os.ReadFile(out); err != nil || string(got) != string(previous) {
		t.Fatalf("previous artifact = %q (err %v), want it untouched", got, err)
	}

	if left, err := os.ReadDir(dir); err != nil || len(left) != 1 {
		t.Fatalf("artifact directory = %v (err %v), want only users.json", left, err)
	}
}
