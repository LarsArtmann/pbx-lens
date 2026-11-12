package web

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	webauthn "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
	_ "modernc.org/sqlite"
)

// Identity is the usermgmt-backed authentication surface: an event-sourced
// operator account in its own SQLite database, passwordless WebAuthn login
// through the loginpage panel, and a session gate on every other route.
// It replaces the htpasswd gate (2026-11-11): credentials live server-side,
// sessions survive restarts (SQL session store), and registration closes
// itself after the first account (MaxUsers = 1 — the library's designed
// single-operator posture; the first visitor to /login on a trusted LAN
// becomes the operator).
type Identity struct {
	service *usermgmt.Service
	db      *sql.DB
	panel   *loginpage.Handler
	session func(http.Handler) http.Handler
	csrf    func(http.Handler) http.Handler
	gate    func(http.Handler) http.Handler
	// secureCookie follows the WebAuthn origins' scheme: browsers drop
	// Secure cookies on plain http (localhost exempt), so upstream's
	// always-true default would break the password factor over LAN http —
	// the stack's documented posture until a proxy fronts the lens.
	secureCookie bool
	// rpidValue is the derived WebAuthn Relying Party ID (/api/ops
	// surfaces it; the ceremonies read it through the webauthn provider).
	rpidValue string
}

// IdentityConfig carries everything OpenIdentity needs. DSN is a SQLite
// path (the file is created on first open); RPID/RPOrigins are the WebAuthn
// Relying Party values and must match the browser-facing host exactly.
type IdentityConfig struct {
	DSN        string
	RPID       string
	RPOrigins  []string
	CookieName string
	Logger     *slog.Logger
}

// publicAuthPath reports whether the request path bypasses the session
// gate: /login and the /auth/* ceremonies are the login flow itself
// (they gate themselves — ADR-0055 posture upstream — and registration
// closes after the first user via MaxUsers); app.css styles the login page
// (it renders Tailwind classes from the same stylesheet as the app);
// robots.txt must stay fetchable to be honored (M15/F46).
func publicAuthPath(path string) bool {
	for _, p := range []string{"/login", "/auth/", "/app.css", "/robots.txt"} {
		if p == path || (strings.HasSuffix(p, "/") && strings.HasPrefix(path, p)) {
			return true
		}
	}

	return false
}

// wantsBrowser reports whether the caller renders HTML: real navigations
// (Accept: text/html) and htmx requests. Browser callers get a redirect to
// /login; API callers (curl exports, scripts) get a plain 401 they can
// script against.
func wantsBrowser(r *http.Request) bool {
	if r.Header.Get("Hx-Request") != "" {
		return true
	}

	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// allOriginsHTTPS reports whether every configured WebAuthn origin fronts
// TLS. The session cookie's Secure flag follows this verdict: browsers
// silently drop Secure cookies on plain http (only localhost is exempt),
// so the flag must match the browser-facing scheme rather than upstream's
// always-true default.
func allOriginsHTTPS(origins []string) bool {
	if len(origins) == 0 {
		return false
	}

	for _, origin := range origins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || parsed.Scheme != "https" {
			return false
		}
	}

	return true
}

// OpenIdentity brings up the identity bounded context: one SQLite database
// holding the event journal, the SQL read models, and the SQL session store;
// a WebAuthn provider for passkey ceremonies; and the loginpage panel. The
// returned Identity owns both the service and the DB handle — Close stops
// the projections and closes the database, last of any shutdown sequence.
//
// Persistence wiring follows the upstream SQL recipe (usermgmt v4.20.0):
// the event store, session store, and checkpoint store are explicit
// constructors over the shared *sql.DB (NewSQLiteEventSourcedSetup is a
// //go:build ignore reference template, not compiled code), and the
// checkpoint schema is applied by hand because the store does not
// self-migrate.
func OpenIdentity(ctx context.Context, cfg IdentityConfig) (*Identity, error) {
	if cfg.DSN == "" {
		return nil, errorfamily.NewRejection(
			"lens.web.identity_dsn",
			"identity database DSN is empty; pass -auth-db to enable login",
		)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	db, stores, err := openIdentityStores(ctx, cfg.DSN)
	if err != nil {
		return nil, err
	}

	provider, err := webauthn.New(webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: "pbx-lens",
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		_ = db.Close()

		return nil, errorfamily.NewRejection(
			"lens.web.identity_webauthn",
			"configuring WebAuthn: "+err.Error(),
		)
	}

	cookieName := cfg.CookieName
	if cookieName == "" {
		cookieName = defaultSessionCookie
	}

	service, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore:       stores.events,
		SessionStore:     stores.sessions,
		CheckpointStore:  stores.checkpoint,
		ReadModelDB:      db,
		ReadModelDialect: "sqlite",
		WebAuthn:         provider,
		Lockout:          usermgmt.NewAccountLockout(),
		AuditLog:         usermgmt.NewAuditLog(),
		MaxUsers:         1,
		Logger:           logger,
	})
	if err != nil {
		_ = db.Close()

		return nil, errorfamily.NewRejection(
			"lens.web.identity_service",
			"starting the identity service: "+err.Error(),
		)
	}

	panel, err := loginpage.New(loginpage.Config{
		Service:          service,
		Title:            "Sign in — pbx-lens",
		Brand:            "pbx-lens",
		Redirect:         "/",
		CSSPath:          "/app.css",
		NonceFromRequest: httputil.NonceFromRequest,
	})
	if err != nil {
		_ = service.Close()
		_ = db.Close()

		return nil, errorfamily.NewRejection(
			"lens.web.identity_loginpage",
			"building the login page: "+err.Error(),
		)
	}

	// Session enrichment first (reads the cookie into the context), then
	// CSRF (the /auth/* ceremonies are the only mutations), then the gate.
	// The session middleware enriches but never blocks; the gate redirects
	// browsers and answers 401 to API clients.
	session := usermgmt.NewSessionMiddleware(service, cookieName)
	csrf := httputil.CSRFMiddleware(httputil.CSRFConfig{})

	identity := &Identity{
		service:      service,
		db:           db,
		panel:        panel,
		session:      session,
		csrf:         csrf,
		secureCookie: allOriginsHTTPS(cfg.RPOrigins),
		rpidValue:    cfg.RPID,
	}

	identity.gate = identity.buildGate()

	return identity, nil
}

// optimizeIdentityDB applies the upstream SQLite pragmas (WAL,
// synchronous=NORMAL, busy_timeout) so the identity journal writes do not
// fight the reader connection. A memory-resident database rejects the WAL
// switch harmlessly (the pragma answers "memory"), so tests can pass a
// file-less DSN only when they accept no durability — production always
// uses a file path.
func optimizeIdentityDB(ctx context.Context, db *sql.DB) error {
	if err := usermgmt.OptimizeSQLiteDB(ctx, db); err != nil {
		return errorfamily.NewRejection(
			"lens.web.identity_db_pragmas",
			"optimizing the identity database: "+err.Error(),
		)
	}

	return nil
}

// identityStores bundles the three SQL stores over the shared handle.
type identityStores struct {
	checkpoint event.CheckpointStore
	events     event.Store
	sessions   usermgmt.SessionStore
}

// openIdentityStores opens the shared identity SQLite handle and every
// store over it. The store constructors self-migrate their own tables; the
// checkpoint schema is the one exception and is applied here by hand. Any
// failure closes the handle before returning — the caller never sees a
// half-open database.
func openIdentityStores(ctx context.Context, dsn string) (*sql.DB, identityStores, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, identityStores{}, errorfamily.NewRejection(
			"lens.web.identity_db_open",
			"opening identity database: "+err.Error(),
		)
	}

	// One connection serializes writers on the identity journal — the
	// upstream persistence contract test runs the same setting, and SQLite
	// under WAL still serves reads concurrently through it.
	db.SetMaxOpenConns(identityMaxOpenConns)

	fail := func(err error) (*sql.DB, identityStores, error) {
		_ = db.Close()

		return nil, identityStores{}, err
	}

	if err := optimizeIdentityDB(ctx, db); err != nil {
		return fail(err)
	}

	if _, err := db.ExecContext(ctx, storage.SQLiteCheckpointSchema()); err != nil {
		return fail(errorfamily.NewRejection(
			"lens.web.identity_checkpoint_schema",
			"applying the checkpoint schema: "+err.Error(),
		))
	}

	checkpoint, err := storage.NewSQLiteCheckpointStore(db)
	if err != nil {
		return fail(errorfamily.NewRejection(
			"lens.web.identity_checkpoint_store",
			"opening the checkpoint store: "+err.Error(),
		))
	}

	events, err := usermgmt.NewSQLEventStore(ctx, db, sqliteDialect)
	if err != nil {
		return fail(errorfamily.NewRejection(
			"lens.web.identity_event_store",
			"opening the identity event store: "+err.Error(),
		))
	}

	sessions, err := usermgmt.NewSQLSessionStore(ctx, db, sqliteDialect)
	if err != nil {
		return fail(errorfamily.NewRejection(
			"lens.web.identity_session_store",
			"opening the identity session store: "+err.Error(),
		))
	}

	return db, identityStores{
		checkpoint: checkpoint,
		events:     events,
		sessions:   sessions,
	}, nil
}

// buildGate composes the request path: session enrichment, CSRF, then the
// public-path bypass and the session requirement.
func (i *Identity) buildGate() func(http.Handler) http.Handler {
	gated := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicAuthPath(r.URL.Path) {
				next.ServeHTTP(w, r)

				return
			}

			if _, authenticated := usermgmt.UserFromContext(r.Context()); authenticated {
				next.ServeHTTP(w, r)

				return
			}

			if wantsBrowser(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)

				return
			}

			http.Error(w, "authentication required", http.StatusUnauthorized)
		})
	}

	// Composition per request-free construction: the handler chain is built
	// once — session(csrf(gate(next))) — and reused.
	return func(next http.Handler) http.Handler {
		return i.session(i.csrf(gated(next)))
	}
}

// Middleware returns the auth middleware for the appkit ExtraMiddlewares
// chain (outermost of the web-local middlewares, where htpasswd sat). Every
// route behind it requires a session except the public login surface.
func (i *Identity) Middleware() func(http.Handler) http.Handler {
	return i.gate
}

// Register mounts the public login surface on the mux: the loginpage panel
// at GET /login and the usermgmt auth ceremonies under /auth/* (register,
// WebAuthn begin/finish, logout, me, credentials). Route-level hardening:
// registration and WebAuthn rate limits run per client IP, and account
// lockout (5 failures / 15 min) guards the login ceremonies.
func (i *Identity) Register(mux *http.ServeMux) {
	i.panel.Mount(mux, "GET /login")

	auth := usermgmt.NewAuthHandler(i.service, usermgmt.HandlerConfig{
		CookieName: defaultSessionCookie,
		Secure:     &i.secureCookie,
		RegistrationRateLimit: usermgmt.RateLimitConfig{
			Enabled: true, MaxRequests: registrationRateLimit, Window: minuteWindow,
		},
		WebAuthnRateLimit: usermgmt.RateLimitConfig{
			Enabled: true, MaxRequests: webauthnRateLimitMax, Window: minuteWindow,
		},
	})
	auth.RegisterRoutes(mux)
}

// Named auth-surface constants (mnd): the session cookie the gate and the
// AuthHandler must agree on, the SQLite dialect string, the journal handle
// cap, and the per-IP ceremony rate limits (registration 10/min — the door
// is closed after one account anyway; WebAuthn 30/min — ceremonies are
// 2 requests each, so 15 logins/min per peer is generous).
const (
	defaultSessionCookie = "session_token"
	sqliteDialect        = "sqlite"

	identityMaxOpenConns  = 1
	registrationRateLimit = 10
	webauthnRateLimitMax  = 30
	minuteWindow          = time.Minute
)

// Ready reports whether the identity projections are serving: live workers
// (or cleanly stopped ones) are ready; backoff/draining/failed are not.
// Serve's readiness gate ANDs this with the index projections' verdict so
// /health/ready reflects both bounded contexts.
func (i *Identity) Ready() bool {
	for _, entry := range i.service.ProjectionStatuses() {
		switch entry.Status {
		case "live", "stopped":
		default:
			return false
		}
	}

	return true
}

// IdentityOps is the identity section of /api/ops (plan P08): the counts
// the single operator's health page deserves. Numbers are real or the
// section is absent — no fake zeros: RPID is the derived Relying Party ID,
// Users/Sessions are live COUNT reads over the shared handle, and the
// audit block aggregates usermgmt's in-memory ring (total, dropped, and
// per-action counts — register/credential_used are the login story).
type IdentityOps struct {
	Enabled      bool           `json:"enabled"`
	RPID         string         `json:"rpid"`
	Users        int            `json:"users"`
	Sessions     *int           `json:"sessions"`
	AuditTotal   int            `json:"audit_total"`
	AuditDropped int            `json:"audit_dropped"`
	AuditActions map[string]int `json:"audit_actions"`
}

// rpid reports the configured WebAuthn Relying Party ID ("" when the
// identity surface is unwired — the field is only meaningful then).
func (i *Identity) rpid() string {
	return i.rpidValue
}

// OpsSnapshot reads the identity section for /api/ops. The queries run on
// the ONE shared identity handle (identityMaxOpenConns is deliberate), so
// they are single cheap COUNTs — fine at operator-page cadence. Sessions
// is a pointer on purpose: null means the COUNT failed (unknown), never a
// fake zero.
func (i *Identity) OpsSnapshot(ctx context.Context) IdentityOps {
	snapshot := IdentityOps{
		Enabled: true,
		RPID:    i.rpid(),
		Users:   i.service.ReadModel().Count(),
	}

	var sessions int

	if err := i.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_sessions").Scan(&sessions); err == nil {
		snapshot.Sessions = &sessions
	}

	if audit := i.service.AuditLog(); audit != nil {
		snapshot.AuditTotal = audit.Count()
		snapshot.AuditDropped = audit.Dropped()

		snapshot.AuditActions = make(map[string]int)
		for _, entry := range audit.Entries() {
			snapshot.AuditActions[entry.Action]++
		}
	}

	return snapshot
}

// Close stops the identity service (projection host, event bus, eviction
// goroutines) and closes the database — in that order, so the journal
// drains before the handle drops. The service never closes the shared DB
// itself; this is the single owner of both lifecycles.
func (i *Identity) Close() error {
	// The service stops first (journal drains), then the handle drops;
	// Join reports BOTH failures and passes a clean nil through.
	return errors.Join(i.service.Close(), i.db.Close())
}
