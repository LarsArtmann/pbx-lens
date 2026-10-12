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

// getWithBasicAuth drives one request through handler and returns the
// recorder; HTTP Basic credentials are only sent when user is non-empty.
func getWithBasicAuth(handler http.Handler, path, user, password string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if user != "" {
		request.SetBasicAuth(user, password)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func TestAuthRejectsMissingAndWrongCredentials(t *testing.T) {
	handler := authedServer(t, writeHtpasswd(t, "lars", authTestPassword))

	response := getWithBasicAuth(handler, "/", "", "")
	if response.Code != http.StatusUnauthorized {
		t.Errorf("no credentials status = %d, want 401", response.Code)
	}

	if response.Header().Get("WWW-Authenticate") == "" {
		t.Error("401 must carry WWW-Authenticate so browsers prompt")
	}

	response = getWithBasicAuth(handler, "/", "lars", "wrong")
	if response.Code != http.StatusUnauthorized {
		t.Errorf("wrong password status = %d, want 401", response.Code)
	}

	response = getWithBasicAuth(handler, "/", "nobody", authTestPassword)
	if response.Code != http.StatusUnauthorized {
		t.Errorf("unknown user status = %d, want 401", response.Code)
	}
}

func TestAuthAcceptsCorrectCredentials(t *testing.T) {
	handler := authedServer(t, writeHtpasswd(t, "lars", authTestPassword))

	response := getWithBasicAuth(handler, "/", "lars", authTestPassword)
	if response.Code != http.StatusOK {
		t.Errorf("correct credentials status = %d, want 200", response.Code)
	}
}

func TestAuthAlsoGatesStaticAndDetailRoutes(t *testing.T) {
	handler := authedServer(t, writeHtpasswd(t, "lars", authTestPassword))

	for _, path := range []string{"/app.css", "/calls/telnyx-conversation/conv-a", "/search?q=x"} {
		response := getWithBasicAuth(handler, path, "", "")
		if response.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without credentials status = %d, want 401", path, response.Code)
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

func TestPadCostReadsLoadedEntriesCost(t *testing.T) {
	generate := func(cost int) []byte {
		t.Helper()

		hash, err := bcrypt.GenerateFromPassword([]byte("pad-cost-probe"), cost)
		if err != nil {
			t.Fatalf("hash at cost %d: %v", cost, err)
		}

		return hash
	}

	tests := []struct {
		name  string
		hash  []byte
		calls string
		want  int
	}{
		{name: "min cost entry", hash: generate(bcrypt.MinCost), want: bcrypt.MinCost},
		{name: "production cost 12 entry", hash: generate(12), want: 12},
		{name: "unparseable entries fall back to default", hash: []byte("garbage"), want: bcrypt.DefaultCost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := map[string][]byte{"lars": tt.hash}
			if got := padCost(users); got != tt.want {
				t.Errorf("padCost = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHtpasswdReloadPicksUpNewEntries(t *testing.T) {
	// SIGHUP rotation (R11): after the file changes, the new credential
	// works and the old one stops — without dropping the throttler state.
	dir := t.TempDir()
	path := filepath.Join(dir, "htpasswd")

	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte("new-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	writeFile := func(content string) {
		t.Helper()

		if werr := os.WriteFile(path, []byte(content), 0o600); werr != nil {
			t.Fatalf("write htpasswd: %v", werr)
		}
	}

	writeFile("olduser:" + string(oldHash))

	auth, err := newHtpasswdAuth(path)
	if err != nil {
		t.Fatalf("newHtpasswdAuth: %v", err)
	}

	throttler := auth.failures

	writeFile("newuser:" + string(newHash))

	if rerr := auth.reload(); rerr != nil {
		t.Fatalf("reload: %v", rerr)
	}

	if !auth.known("newuser") || auth.known("olduser") {
		t.Error("reload must swap the credential table atomically")
	}

	if auth.failures != throttler {
		t.Error("reload must not reset the failure throttler")
	}
}

func TestHtpasswdBadFileKeepsOldTable(t *testing.T) {
	// A broken rotation must never lock the operator out: a rejected file
	// keeps the running credentials and reports the error (R11).
	dir := t.TempDir()
	path := filepath.Join(dir, "htpasswd")

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if werr := os.WriteFile(path, []byte("user:"+string(hash)), 0o600); werr != nil {
		t.Fatalf("write htpasswd: %v", werr)
	}

	auth, err := newHtpasswdAuth(path)
	if err != nil {
		t.Fatalf("newHtpasswdAuth: %v", err)
	}

	if werr := os.WriteFile(path, []byte("garbage-without-colon\n"), 0o600); werr != nil {
		t.Fatalf("write htpasswd: %v", werr)
	}

	if rerr := auth.reload(); rerr == nil {
		t.Error("reload of a malformed file must fail loudly")
	}

	if !auth.known("user") {
		t.Error("a failed reload must keep the previous table")
	}
}

func TestPerIPThrottlerSeparatesBuckets(t *testing.T) {
	// Two peers must never share a bucket (R13): one IP's exhaustion must
	// not tax another's first failure.
	throttler := newPerIPThrottler(8)

	first := throttler.bucket("203.0.113.10:4432")

	second := throttler.bucket("198.51.100.7:9")
	if first == nil || second == nil {
		t.Fatal("both peers must get buckets")
	}

	if first == second {
		t.Error("distinct peers share a bucket")
	}

	if again := throttler.bucket("203.0.113.10:4432"); again != first {
		t.Error("the same peer must keep its bucket across calls")
	}
}

func TestPerIPThrottlerEvictsOldestUnderCap(t *testing.T) {
	// A spoofed-source flood must not balloon the map: the oldest bucket
	// yields (FIFO) once the cap is reached.
	throttler := newPerIPThrottler(2)

	a := throttler.bucket("10.0.0.1:1")
	throttler.bucket("10.0.0.2:2")
	throttler.bucket("10.0.0.3:3")

	if returned := throttler.bucket("10.0.0.1:1"); returned == a {
		t.Error("evicted peer must not keep its old bucket")
	}
}

func TestPerIPThrottlerFallbackBucketForHostlessPeers(t *testing.T) {
	// Peers without a parseable host share the fallback bucket — the
	// throttle degrades to global, it does not disable itself.
	throttler := newPerIPThrottler(8)

	first := throttler.bucket("bogus")

	second := throttler.bucket("")
	if first == nil || first != second {
		t.Error("hostless peers must share the fallback bucket")
	}
}
