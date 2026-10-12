package web

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// basicAuthRealm names the protection space browsers show in their login
// prompt.
const basicAuthRealm = `Basic realm="pbx-lens"`

// authFailureInterval and authFailureBurst throttle failed basic-auth
// attempts. bcrypt comparisons are the expensive path, so capping the
// FAILURE rate caps brute-force throughput. Successful logins never wait.
// Since R13 the cap is per client address (bounded bucket map, shared
// fallback for peers without a host part).
const (
	authFailureInterval = 6 * time.Second
	authFailureBurst    = 5

	// authIPBucketCap bounds the per-address bucket map: eviction keeps a
	// spoofed-source flood from ballooning memory (FIFO — the oldest peer
	// loses its bucket and would start a fresh one on return).
	authIPBucketCap = 1024
)

// timingPadSeed seeds the dummy hash compared against unknown users so the
// failure and success paths run identical bcrypt work. It is a fixed
// non-secret constant, never a credential.
const timingPadSeed = "pbx-lens-timing-pad"

// HtpasswdMiddleware gates every route behind HTTP Basic auth (D1/T24):
// call metadata must not be readable without operator credentials. The
// file is an htpasswd file with BCRYPT entries only (`htpasswd -B`); weaker
// schemes reject at startup, not at first login. SIGHUP re-reads the file
// (R11) so credential rotation never needs a restart; a broken file keeps
// the previous table and logs why.
func HtpasswdMiddleware(path string) (func(http.Handler) http.Handler, error) {
	auth, err := newHtpasswdAuth(path)
	if err != nil {
		return nil, err
	}

	go auth.watchSIGHUP()

	return auth.middleware(), nil
}

// htpasswdAuth is the mutable credential state behind the middleware: the
// table and its timing pad swap atomically under the mutex on reload.
type htpasswdAuth struct {
	path string

	mu        sync.RWMutex
	users     map[string][]byte
	dummyHash []byte

	failures *perIPThrottler
}

func newHtpasswdAuth(path string) (*htpasswdAuth, error) {
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

	return &htpasswdAuth{
		path:      path,
		users:     users,
		dummyHash: dummyHash,
		failures:  newPerIPThrottler(authIPBucketCap),
	}, nil
}

// middleware is the auth gate. The bcrypt comparison always runs — known
// user or not — so response timing does not enumerate valid usernames. A
// failed attempt then waits for the peer's throttle token before answering:
// an attacker's guesses cost wall-clock time, not just CPU. A canceled
// context still answers 401 (best effort).
func (a *htpasswdAuth) middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, password, ok := r.BasicAuth()
			if !ok {
				unauthorized(w)

				return
			}

			a.mu.RLock()

			hash, known := a.users[user]
			if !known {
				hash = a.dummyHash
			}

			a.mu.RUnlock()

			if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
				a.failures.wait(r.Context(), r.RemoteAddr)

				unauthorized(w)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// watchSIGHUP re-reads the htpasswd file on every SIGHUP until the process
// ends. Reload failures keep the running table — a half-written rotation
// must never lock the operator out — and are logged for the operator.
func (a *htpasswdAuth) watchSIGHUP() {
	notifications := make(chan os.Signal, 1)
	signal.Notify(notifications, syscall.SIGHUP)

	for range notifications {
		if err := a.reload(); err != nil {
			log.Printf("pbx-lens: htpasswd reload failed, keeping the previous credentials: %v", err)
		}
	}
}

// reload re-reads the file and atomically swaps table and timing pad.
func (a *htpasswdAuth) reload() error {
	users, err := parseHtpasswd(a.path)
	if err != nil {
		return err
	}

	dummyHash, err := bcrypt.GenerateFromPassword([]byte(timingPadSeed), padCost(users))
	if err != nil {
		return errorfamily.NewRejection(
			"lens.web.htpasswd_timing_pad",
			"preparing timing pad hash: "+err.Error(),
		)
	}

	a.mu.Lock()

	a.users = users
	a.dummyHash = dummyHash

	a.mu.Unlock()

	return nil
}

// known reports whether a username exists in the current table (test seam
// for reload semantics).
func (a *htpasswdAuth) known(user string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	_, found := a.users[user]

	return found
}

// perIPThrottler rate-limits failed logins per client address. X-Forwarded-For
// is deliberately NOT honored: behind the systemd socket or a local reverse
// proxy every peer is the proxy itself, and a forgeable header would let an
// attacker mint fresh buckets at will. Peers without a parseable host share
// one fallback bucket, so the throttle degrades to the old global behavior
// instead of disabling itself.
type perIPThrottler struct {
	mu       sync.Mutex
	buckets  map[string]*rate.Limiter
	order    []string
	capacity int
	fallback *rate.Limiter
}

func newPerIPThrottler(capacity int) *perIPThrottler {
	return &perIPThrottler{
		buckets:  make(map[string]*rate.Limiter),
		capacity: capacity,
		fallback: rate.NewLimiter(rate.Every(authFailureInterval), authFailureBurst),
	}
}

// wait blocks until the peer's bucket has a token (or the context ends).
func (p *perIPThrottler) wait(ctx context.Context, remoteAddr string) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || host == "" {
		_ = p.fallback.Wait(ctx)

		return
	}

	p.mu.Lock()

	limiter, known := p.buckets[host]
	if !known {
		if len(p.buckets) >= p.capacity {
			oldest := p.order[0]
			p.order = p.order[1:]
			delete(p.buckets, oldest)
		}

		limiter = rate.NewLimiter(rate.Every(authFailureInterval), authFailureBurst)
		p.buckets[host] = limiter
		p.order = append(p.order, host)
	}

	p.mu.Unlock()

	_ = limiter.Wait(ctx)
}

// bucket returns the live limiter for an address (test seam: identity of
// the per-IP bookkeeping, not its timing).
func (p *perIPThrottler) bucket(remoteAddr string) *rate.Limiter {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || host == "" {
		return p.fallback
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if limiter, known := p.buckets[host]; known {
		return limiter
	}

	return nil
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
