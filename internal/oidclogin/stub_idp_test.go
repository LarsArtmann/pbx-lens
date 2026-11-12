package oidclogin

import (
	"crypto/sha256"
	"net/url"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/oidclogin/oidctest"
)

// newStubIdP starts the shared stub provider for the package suite.
func newStubIdP(t *testing.T, claims map[string]any) *oidctest.StubIdP {
	t.Helper()

	idp := oidctest.NewStub(claims)
	t.Cleanup(idp.Close)

	return idp
}

// parseQuery extracts the query of an authorization URL for assertions.
func parseQuery(t *testing.T, rawURL string) url.Values {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}

	return parsed.Query()
}

func sha256Sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
