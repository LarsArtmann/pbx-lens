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

// timingPadSeed seeds the dummy hash compared against unknown users so the
// failure and success paths run identical bcrypt work. It is a fixed
// non-secret constant, never a credential.
const timingPadSeed = "pbx-lens-timing-pad"

// HtpasswdMiddleware gates every route behind HTTP Basic auth (D1/T24):
// call metadata must not be readable without operator credentials. The
// file is an htpasswd file with BCRYPT entries only (`htpasswd -B`); weaker
// schemes reject at startup, not at first login.
func HtpasswdMiddleware(path string) (func(http.Handler) http.Handler, error) {
	users, err := parseHtpasswd(path)
	if err != nil {
		return nil, err
	}

	// The dummy hash keeps the failure path running the same bcrypt work as
	// the success path, so timing never reveals whether a username exists.
	// The pad cost matches the entries' cost (one htpasswd file shares one
	// cost in practice): a MinCost pad would make unknown-user failures
	// measurably cheaper than real logins and defeat the equalization.
	dummyHash, err := bcrypt.GenerateFromPassword([]byte(timingPadSeed), padCost(users))
	if err != nil {
		return nil, errorfamily.NewRejection(
			"lens.web.htpasswd_timing_pad",
			"preparing timing pad hash: "+err.Error(),
		)
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

// padCost reads the bcrypt work factor the loaded entries use, so the
// timing pad can match it. Entries carry their cost inside the hash
// ($2a$10$...); the first parse wins because one file shares one cost in
// practice. DefaultCost covers a file whose entries all failed to parse a
// cost (unreachable via parseHtpasswd's $2 gate, kept as a safe fallback).
func padCost(users map[string][]byte) int {
	for _, hash := range users {
		if cost, err := bcrypt.Cost(hash); err == nil {
			return cost
		}
	}

	return bcrypt.DefaultCost
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
