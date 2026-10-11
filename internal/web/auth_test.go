package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const authTestPassword = "correct horse battery staple"

// writeHtpasswd materializes an htpasswd file with one bcrypt user.
func writeHtpasswd(t *testing.T, user, password string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	path := filepath.Join(t.TempDir(), "htpasswd")
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%s:%s\n", user, hash)), 0o600); err != nil {
		t.Fatalf("write htpasswd: %v", err)
	}

	return path
}

// authedServer wraps a fresh test server in the auth middleware.
func authedServer(t *testing.T, htpasswdPath string) http.Handler {
	t.Helper()

	middleware, err := HtpasswdMiddleware(htpasswdPath)
	if err != nil {
		t.Fatalf("HtpasswdMiddleware: %v", err)
	}

	return middleware(newTestServer(t, webObservation("conv-a", time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC))))
}

func getWithBasicAuth(handler http.Handler, path, user, password string) *http.Response {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if user != "" {
		request.SetBasicAuth(user, password)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder.Result()
}

func TestAuthRejectsMissingAndWrongCredentials(t *testing.T) {
	handler := authedServer(t, writeHtpasswd(t, "lars", authTestPassword))

	response := getWithBasicAuth(handler, "/", "", "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("no credentials status = %d, want 401", response.StatusCode)
	}

	if response.Header.Get("WWW-Authenticate") == "" {
		t.Error("401 must carry WWW-Authenticate so browsers prompt")
	}

	response = getWithBasicAuth(handler, "/", "lars", "wrong")
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong password status = %d, want 401", response.StatusCode)
	}

	response = getWithBasicAuth(handler, "/", "nobody", authTestPassword)
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("unknown user status = %d, want 401", response.StatusCode)
	}
}

func TestAuthAcceptsCorrectCredentials(t *testing.T) {
	handler := authedServer(t, writeHtpasswd(t, "lars", authTestPassword))

	response := getWithBasicAuth(handler, "/", "lars", authTestPassword)
	if response.StatusCode != http.StatusOK {
		t.Errorf("correct credentials status = %d, want 200", response.StatusCode)
	}
}

func TestAuthAlsoGatesStaticAndDetailRoutes(t *testing.T) {
	handler := authedServer(t, writeHtpasswd(t, "lars", authTestPassword))

	for _, path := range []string{"/app.css", "/calls/telnyx-conversation/conv-a", "/search?q=x"} {
		response := getWithBasicAuth(handler, path, "", "")
		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s without credentials status = %d, want 401", path, response.StatusCode)
		}
	}
}

func TestHtpasswdRejectsWeakSchemesAtStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "htpasswd")
	if err := os.WriteFile(path, []byte("lars:{SHA}W6ph5Mm5Pz8GgiULbPgzG37mj9g=\n"), 0o600); err != nil {
		t.Fatalf("write htpasswd: %v", err)
	}

	_, err := HtpasswdMiddleware(path)
	if err == nil {
		t.Fatal("sha1 htpasswd entry must reject at startup, not at login")
	}

	if want := "htpasswd -B"; !strings.Contains(err.Error(), want) {
		t.Errorf("error should name the fix %q, got: %v", want, err)
	}
}
