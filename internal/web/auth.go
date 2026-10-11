package web

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/crypto/bcrypt"
)

// basicAuthRealm names the protection space browsers show in their login
// prompt.
const basicAuthRealm = `Basic realm="pbx-lens"`

// dummyHash keeps the failure path running the same bcrypt work as the
// success path, so timing never reveals whether a username exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("pbx-lens-timing-pad"), bcrypt.MinCost)

// HtpasswdMiddleware gates every route behind HTTP Basic auth (D1/T24):
// call metadata must not be readable without operator credentials. The
// file is an htpasswd file with BCRYPT entries only (`htpasswd -B`); weaker
// schemes reject at startup, not at first login.
func HtpasswdMiddleware(path string) (func(http.Handler) http.Handler, error) {
	users, err := parseHtpasswd(path)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, password, ok := r.BasicAuth()
			if !ok {
				unauthorized(w)

				return
			}

			hash, known := users[user]
			if !known {
				hash = dummyHash
			}

			// The bcrypt comparison always runs — known user or not — so
			// response timing does not enumerate valid usernames.
			if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
				unauthorized(w)

				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}

// unauthorized answers with the 401 that makes browsers prompt.
func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", basicAuthRealm)
	http.Error(w, "pbx-lens needs operator credentials (HTTP Basic auth).", http.StatusUnauthorized)
}

// parseHtpasswd reads user:hash lines; only bcrypt hashes ($2a/$2b/$2y
// prefixes) are accepted.
func parseHtpasswd(path string) (map[string][]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errorfamily.NewRejection("lens.web.htpasswd_open", "opening htpasswd file: "+err.Error())
	}

	defer func() { _ = file.Close() }()

	users := make(map[string][]byte)

	scanner := bufio.NewScanner(file)

	line := 0
	for scanner.Scan() {
		line++

		entry := strings.TrimSpace(scanner.Text())
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}

		user, hash, found := strings.Cut(entry, ":")
		if !found || user == "" || !strings.HasPrefix(hash, "$2") {
			return nil, errorfamily.NewRejection(
				"lens.web.htpasswd_entry",
				"htpasswd line "+strconv.Itoa(line)+
					": expected a bcrypt entry (generate with `htpasswd -B`)",
			)
		}

		users[user] = []byte(hash)
	}

	if err := scanner.Err(); err != nil {
		return nil, errorfamily.NewRejection("lens.web.htpasswd_read", "reading htpasswd file: "+err.Error())
	}

	if len(users) == 0 {
		return nil, errorfamily.NewRejection(
			"lens.web.htpasswd_empty",
			"htpasswd file has no entries (generate with `htpasswd -B`)",
		)
	}

	return users, nil
}
