package main

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/identity"
	"github.com/larsartmann/pbx-lens/internal/identity/identitytest"
)

func TestIdentityExportCarriesTheOperatorSeat(t *testing.T) {
	// The runbook's migration artifact (identity-first-deploy.md): a real
	// file-backed identity DB, the operator registered through the REAL
	// /auth/register walk, then the verb's export read back as JSON. An
	// empty-DB export proves wiring but not the artifact; this proves both.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	dsn := filepath.Join(t.TempDir(), "identity.db")

	ident, err := identity.OpenIdentity(ctx, identity.IdentityConfig{
		DSN:       dsn,
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost"},
	})
	if err != nil {
		t.Fatalf("OpenIdentity: %v", err)
	}

	mux := http.NewServeMux()
	ident.Register(mux)
	identitytest.Session(t, ident.Middleware()(mux), "operator@pbx.artmann.tech")

	if err := ident.Close(); err != nil {
		t.Fatalf("close identity: %v", err)
	}

	out := filepath.Join(t.TempDir(), "users.json")
	if err := runIdentityExport(ctx, dsn, out, "localhost", "http://localhost"); err != nil {
		t.Fatalf("runIdentityExport: %v", err)
	}

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

	if len(users) != 1 || users[0].Email != "operator@pbx.artmann.tech" {
		t.Fatalf("exported users = %+v, want exactly the operator seat", users)
	}
}
