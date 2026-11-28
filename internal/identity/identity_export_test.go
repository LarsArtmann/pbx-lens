package identity

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/httputil/endpoints"
	"github.com/larsartmann/pbx-lens/internal/identity/identitytest"
)

// What the JSON export is and is not (verified 2026-11-21): a readable
// record of who the identity database holds — NOT a restore path. The
// runbook's migration guidance rests on both facts, so both are pinned.

// TestExportCarriesNoCredentials pins the export's exact wire shape. The
// runbook promises "no credentials at all"; an upstream release that adds
// a credential-bearing field must fail HERE, so the claim is re-checked
// before anyone relies on it again.
func TestExportCarriesNoCredentials(t *testing.T) {
	ident, _ := openFileBackedIdentity(t)

	mux := http.NewServeMux()
	ident.Register(mux, endpoints.NewRegistry())

	identitytest.Session(t, ident.Middleware()(mux), "operator@pbx.artmann.tech")

	var exported bytes.Buffer

	if err := ident.ExportUsersToJSON(t.Context(), &exported); err != nil {
		t.Fatalf("ExportUsersToJSON: %v", err)
	}

	var users []map[string]jsontext.Value
	if err := json.Unmarshal(exported.Bytes(), &users); err != nil {
		t.Fatalf("export is not a JSON array of objects: %v\n%s", err, exported.String())
	}

	if len(users) != 1 {
		t.Fatalf("exported %d users, want the one operator", len(users))
	}

	keys := make([]string, 0, len(users[0]))
	for key := range users[0] {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	want := []string{"display_name", "email", "email_verified", "id", "totp_enabled"}

	if !slices.Equal(keys, want) {
		t.Fatalf(
			"export fields = %v, want exactly %v — if upstream added a field, re-verify the runbook's \"no credentials\" promise",
			keys,
			want,
		)
	}
}

// TestImportedAccountIsLockedOut pins why the JSON is not a restore path:
// the upstream import registers a user WITHOUT a passkey, and on this
// single-operator deployment (MaxUsers = 1) that one import closes the
// registration door — so nobody can enroll a credential, and login is
// passkey-only. If upstream ever grows a credential-bootstrap path this
// fails, and the runbook may say the JSON restores logins.
func TestImportedAccountIsLockedOut(t *testing.T) {
	ident, _ := openFileBackedIdentity(t)

	result, err := ident.service.ImportUsersFromJSON(
		t.Context(),
		strings.NewReader(`[{"email":"operator@pbx.artmann.tech","display_name":"Operator"}]`),
	)
	if err != nil || result.Imported != 1 {
		t.Fatalf("import the exported operator: result = %+v, err = %v", result, err)
	}

	mux := http.NewServeMux()
	ident.Register(mux, endpoints.NewRegistry())

	handler := ident.Middleware()(mux)

	csrfCookie, token := identitytest.Challenge(t, handler)

	registered := identitytest.PostAuth(t, handler, "/auth/register", map[string]string{
		"email": "operator@pbx.artmann.tech", "display_name": "Operator",
	}, csrfCookie, token)

	if registered.Code != http.StatusForbidden {
		t.Fatalf(
			"POST /auth/register after the import = %d, want 403 (the door closed on the credential-less account); body: %s",
			registered.Code,
			registered.Body.String(),
		)
	}

	if !strings.Contains(registered.Body.String(), "usermgmt.registration_closed") {
		t.Errorf("refusal body = %s, want the registration_closed code", registered.Body.String())
	}
}
