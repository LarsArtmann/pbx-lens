// Package serve composes the read-only web service: appkit service + the
// identity session gate + web routes + the freshness poller, behind the
// flag surface the serve verb parses. The cmd verb stays a thin shell
// (flags, -db requirement, exit policy); everything that BOOTS the stack
// lives here so tests and the binary drive the exact same composition.
package serve

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-appkit/systemd"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/httputil/endpoints"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/identity"
	"github.com/larsartmann/pbx-lens/internal/ingest"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/search"
	"github.com/larsartmann/pbx-lens/internal/web"
)

const (
	defaultAddr  = "127.0.0.1:8300"
	serveTimeout = 5 * time.Minute
)

// Security header values (Pareto plan P1-T5/T6): no-referrer keeps the
// internal host and query strings from leaking to the recording origin;
// nosniff and noindex harden every non-HTML endpoint uniformly.
const (
	securityHeaderNosniff    = "nosniff"
	securityReferrerPolicy   = "no-referrer"
	securityRobotsTagHeader  = "X-Robots-Tag"
	securityRobotsTagNoindex = "noindex"
)

// slogLog adapts slog's record shape to the injected log func (R14): the
// structured key-value calls in the poller and web layers come out as
// greppable text records instead of printf lines.
func slogLog(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Flags carries the serve verb's flag surface, extracted so the verb
// reads as orchestration and the flag mechanics stay testable.
type Flags struct {
	Db                 *string
	Addr               *string
	KeyFile            *string
	RecordingsDir      *string
	RecTimeout         *time.Duration
	AuthDB             *string
	AuthRPID           *string
	AuthOrigin         *string
	AuthOIDCIssuer     *string
	AuthOIDCClientID   *string
	AuthOIDCSecretFile *string
	AuthOIDCRedirect   *string
	AuthOIDCGroup      *string
	PollEvery          *time.Duration
	PollSince          *time.Duration
	PollDirection      *string
	AiDID              *string
	MaxAge             *time.Duration
	CdrFile            *string
	WebhookLog         *string
	TranscriptsDir     *string
	WiredOps           *bool
}

// RegisterFlags declares every serve flag on fs and returns the
// parsed-value handles.
func RegisterFlags(fs *flag.FlagSet) *Flags {
	flags := new(Flags)

	flags.Db = fs.String("db", "", "sqlite DSN (required)")
	flags.Addr = fs.String("addr", defaultAddr, "listen address")
	flags.KeyFile = fs.String(
		"telnyx-key-file",
		"",
		"Telnyx API key file for the recording proxy (default: disabled)",
	)
	flags.RecordingsDir = fs.String(
		"recordings-dir",
		"/var/lib/telephony/recordings",
		"local FreeSWITCH recordings directory (stems tried: indexed ref, "+
			"{uuid}_{dest}.wav, bare uuid; empty = local playback disabled)",
	)
	flags.RecTimeout = fs.Duration(
		"recording-timeout",
		web.DefaultUpstreamTimeout,
		"upstream bound for the recording proxy: metadata deadline + WAV response headers "+
			"(0 = default; the audio body itself streams unbounded)",
	)
	flags.AuthDB = fs.String(
		"auth-db",
		"",
		"sqlite identity database enabling passkey login (empty = no auth for localhost dev)",
	)
	flags.AuthRPID = fs.String(
		"auth-rpid",
		"",
		"WebAuthn Relying Party ID — the registrable domain the browser sees "+
			"(default: the host part of -addr)",
	)
	flags.AuthOrigin = fs.String(
		"auth-origin",
		"",
		"comma-separated WebAuthn origins that must byte-match the browser address bar, "+
			"scheme included (default: http://<addr>)",
	)
	flags.AuthOIDCIssuer = fs.String(
		"auth-oidc-issuer",
		"",
		"OIDC issuer base URL enabling the Pocket ID sign-in lane beside passkeys "+
			"(empty = passkey-only; discovery is lazy — an IdP outage costs a login, never the boot)",
	)
	flags.AuthOIDCClientID = fs.String(
		"auth-oidc-client-id",
		"",
		"OIDC client id registered in the IdP (required with -auth-oidc-issuer)",
	)
	flags.AuthOIDCSecretFile = fs.String(
		"auth-oidc-secret-file",
		"",
		"OIDC client secret file, read per login (required with -auth-oidc-issuer; "+
			"rotation takes effect on the next sign-in)",
	)
	flags.AuthOIDCRedirect = fs.String(
		"auth-oidc-redirect",
		"",
		"registered OIDC callback URL: https://<public-host>/auth/oauth/pocketid/callback",
	)
	flags.AuthOIDCGroup = fs.String(
		"auth-oidc-group",
		"",
		"comma-separated IdP groups allowed to sign in (fail-closed: a token claiming "+
			"none of them is denied before any user matching)",
	)
	flags.PollEvery = fs.Duration(
		"poll-every",
		cli.DefaultPollEvery,
		"poll cadence for keeping the index fresh (0 = disabled)",
	)
	flags.PollSince = fs.Duration(
		"poll-since",
		cli.DefaultPollSince,
		"Telnyx look-back window per poll",
	)
	flags.PollDirection = fs.String(
		"poll-direction",
		"outbound",
		"caller-declared direction for polled Telnyx conversations",
	)
	flags.AiDID = fs.String(
		"ai-did",
		"",
		"comma-separated AI-facing DIDs: polled conversations touching one are inbound (empty = keep declared direction)",
	)
	flags.MaxAge = fs.Duration(
		"max-age",
		0,
		"refuse polled observations older than this (0 = keep everything)",
	)
	flags.CdrFile = fs.String(
		"cdr-file",
		"",
		"FreeSWITCH Master.csv re-ingested every poll (empty = skip)",
	)
	flags.WebhookLog = fs.String(
		"webhook-log",
		"",
		"receiver webhook JSONL re-ingested every poll (empty = skip)",
	)
	flags.TranscriptsDir = fs.String(
		"transcripts-dir",
		"",
		"agent transcripts dir re-scanned every poll (empty = skip)",
	)
	flags.WiredOps = fs.Bool(
		"wired-ops",
		false,
		"dev probe fixture: serve a canned STALE poller snapshot on /ops and /api/ops instead of the live counters",
	)

	return flags
}

// Run executes the serve composition: boot the index side, bring up the
// identity context when -auth-db is set, wire the middlewares and routes,
// start the poller, and block until the signal-owned shutdown (appkit Run
// owns SIGINT/SIGTERM and the graceful drain). One drain at startup catches
// the read model up with whatever the ingest commands left unprojected.
// Flag parsing, the -db requirement, and the exit policy stay in the cmd
// verb shell: Run returns the first error instead of exiting.
func Run(flags *Flags, version string) error {
	pollDir, err := cli.ParseDirection(*flags.PollDirection)
	if err != nil {
		return err
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), serveTimeout)
	defer cancel()

	// The fleet-standard error-family classifications register once at boot
	// (appkit v0.10+ exposes the same registry appkit's own middlewares use;
	// re-registration is idempotent).
	appkit.RegisterClassifications()

	sys, index, closeAll, err := openServeIndex(setupCtx, *flags.Db)
	if err != nil {
		return err
	}

	defer closeAll()

	// The identity context (passkey login) is opt-in via -auth-db: without
	// it the serve runs unauthenticated — the localhost dev posture the
	// probes rely on.
	var identity *identity.Identity

	if *flags.AuthDB != "" {
		oidcCfg := oidcConfigFromFlags(
			*flags.AuthOIDCIssuer,
			*flags.AuthOIDCClientID,
			*flags.AuthOIDCSecretFile,
			*flags.AuthOIDCRedirect,
			*flags.AuthOIDCGroup,
		)

		identity, err = openServeIdentity(
			setupCtx,
			*flags.Addr,
			*flags.AuthDB,
			*flags.AuthRPID,
			*flags.AuthOrigin,
			oidcCfg,
		)
		if err != nil {
			return err
		}

		defer func() { _ = identity.Close() }()
	}
	// The extras in wire order: auth gates everything, Compression sits
	// outer of ETag so tags hash identity bodies (go-etag's ordering
	// doctrine). Server-Timing needs NO wiring here since appkit v0.10: it
	// is the default OUTERMOST middleware (deleted from this chain on the
	// v0.12 bump), so the header observes the full request span including
	// auth.
	cacheStats := web.NewCacheCounters()

	var authMW func(http.Handler) http.Handler

	if identity != nil {
		authMW = identity.Middleware()
	}

	serviceConfig := buildServiceConfig(*flags.Addr, version, authMW, cacheStats, slogLog)

	// Readiness is the read model's honesty (plan T17): /health/ready
	// answers 503 while any projection worker is failed (restart budget
	// exhausted) or backing off — index AND identity contexts both. Every
	// other state is healthy here — the host is a batch drainer, so workers
	// sit idle before the first drain, run during it, and stop once caught
	// up; the poller below re-drains on every tick. Treating "stopped" as
	// unready made a healthy serve answer 503 forever (scripts/
	// verify-ui-contract.js Contract F caught it).
	serviceConfig.ReadyCheck = func() bool {
		if !cli.ProjectionWorkersReady(sys.ProjectionHost().Status()) {
			return false
		}

		return identity == nil || identity.Ready()
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		return err
	}

	// The pinned appkit release predates Service.Routes, so the route registry is
	// created here and handed to both registrars; web.Register mounts the
	// GET /endpoints listing last, so it covers every route above it.
	routes := endpoints.NewRegistry()

	if identity != nil {
		identity.Register(svc.Mux, routes)
	}

	counters := cli.NewSourceCounters(*flags.PollEvery)

	opsProvider, recordings, telnyxClient, err := wireServeSources(
		counters, *flags.WiredOps, *flags.RecordingsDir, *flags.RecTimeout, *flags.KeyFile,
	)
	if err != nil {
		return err
	}

	web.Register(
		svc.Mux,
		sys,
		web.Options{
			Recordings: recordings, Search: index, Log: slogLog,
			Ops: opsProvider, Cache: cacheStats,
			RetentionMaxAge: *flags.MaxAge, AuthEnabled: identity != nil,
			Identity: identity,
		},
		routes,
	)

	// The poller keeps the index fresh until shutdown; each pass drains
	// projections and rebuilds the FTS index so / and /search never serve
	// yesterday's state while serve runs.
	pollCtx, stopPoller := context.WithCancel(context.Background())
	defer stopPoller()

	go cli.PollLoop(pollCtx, sys, index, cli.PollConfig{
		Every:          *flags.PollEvery,
		Telnyx:         telnyxClient,
		SinceAgo:       *flags.PollSince,
		Direction:      pollDir,
		AIDIDs:         ingest.ParseCommaSet(*flags.AiDID),
		CDRFile:        *flags.CdrFile,
		WebhookLog:     *flags.WebhookLog,
		TranscriptsDir: *flags.TranscriptsDir,
		MaxAge:         *flags.MaxAge,
		Counters:       counters,
	}, slogLog)

	// The serve lifetime is signal-owned (Run handles SIGINT/SIGTERM) — no
	// timeout context here, or the server would die with it.
	if err := svc.Run(context.Background()); err != nil {
		return err
	}

	return nil
}

// oidcConfigFromFlags folds the five -auth-oidc-* flags into the identity
// lane's provider config: empty issuer = passkey-only (nil), and the
// comma-separated group allowlist is trimmed with blanks dropped.
func oidcConfigFromFlags(
	issuer, clientID, secretFile, redirect, group string,
) *identity.OIDCConfig {
	if issuer == "" {
		return nil
	}

	cfg := &identity.OIDCConfig{
		Issuer:           issuer,
		ClientID:         clientID,
		ClientSecretFile: secretFile,
		RedirectURL:      redirect,
	}

	for g := range strings.SplitSeq(group, ",") {
		if g = strings.TrimSpace(g); g != "" {
			cfg.AllowedGroups = append(cfg.AllowedGroups, g)
		}
	}

	return cfg
}

// wireServeSources resolves the poller-adjacent wiring: the ops snapshot
// provider (the -wired-ops probe fixture swaps in a canned STALE verdict so
// the browser probes can assert the stale alert renders without waiting on
// real poll cadences — dev-only, a production serve would cry wolf
// forever), the recording proxy options, and the Telnyx client when a key
// file is configured.
func wireServeSources(
	counters *cli.SourceCounters,
	wiredOps bool,
	recordingsDir string,
	recTimeout time.Duration,
	keyFile string,
) (func() any, web.RecordingOptions, *telnyx.Client, error) {
	// Typed snapshot inside; the any boundary stays at web Options.
	opsProvider := func() any { return counters.Snapshot() }
	if wiredOps {
		opsProvider = cannedStaleOps
	}

	recordings := web.RecordingOptions{RecordingsDir: recordingsDir, UpstreamTimeout: recTimeout}

	var telnyxClient *telnyx.Client

	if keyFile != "" {
		key, err := cli.ResolveKey(keyFile)
		if err != nil {
			return nil, web.RecordingOptions{}, nil, err
		}

		telnyxClient = telnyx.NewClient(key, "")
		recordings.Telnyx = telnyxClient
	}

	return opsProvider, recordings, telnyxClient, nil
}

// cannedStaleOps is the -wired-ops fixture payload: the typed shape
// cli.PollerHealth (the same value the live poller produces), carrying an
// unambiguous stale verdict (age 2h, stale=true).
func cannedStaleOps() any {
	const (
		fixtureOfferedCDR     = 12
		fixtureRecordedCDR    = 12
		fixtureOfferedTelnyx  = 8
		fixtureRecordedTelnyx = 8
		fixturePassAgeSecs    = 7200.0
	)

	return cli.PollerHealth{
		Sources: map[string]cli.SourceStat{
			"freeswitch-cdr":      {Offered: fixtureOfferedCDR, Recorded: fixtureRecordedCDR},
			"telnyx-conversation": {Offered: fixtureOfferedTelnyx, Recorded: fixtureRecordedTelnyx},
		},
		LastPass:   time.Now().Add(-2 * time.Hour).UTC(),
		AgeSeconds: fixturePassAgeSecs,
		Stale:      true,
	}
}

// buildServiceConfig assembles the serve composition. sd_notify (T7/F23)
// installs first: READY=1 after the listener binds, STOPPING=1 at the
// drain, WATCHDOG keepalive while serving — a no-op outside systemd, so the
// same binary serves localhost dev and the service manager unchanged. The
// cache counters and log hook ride in so the ETag middleware's outcomes
// surface on /api/ops (R26). authMW (nil in unauthenticated dev) is the
// identity session gate; it takes the outermost extra slot so login is
// required before anything web-local runs.
func buildServiceConfig(
	addr, version string,
	authMW func(http.Handler) http.Handler,
	cacheStats *web.CacheCounters,
	log func(msg string, args ...any),
) appkit.ServiceConfig {
	serviceConfig := appkit.ServiceConfig{
		Addr:    addr,
		Version: version,
		// Exports and the recording proxy stream unbounded by contract; a
		// WriteTimeout would truncate both at the 30s appkit default, so
		// the Timeout middleware drops out of the default stack entirely.
		WriteTimeout: appkit.NoTimeout,
		Metrics:      &appkit.MetricsConfig{AllowUnauthenticated: true},
	}

	systemd.Install(&serviceConfig)

	if authMW != nil {
		serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares, authMW)
	}

	// Security headers: the appkit default SecurityHeaders still runs outer
	// (Recovery → … → SecurityHeaders); the inner Set calls below win, so
	// the effective policy is ours while the default's X-Frame-Options: DENY
	// survives untouched. Header-only and set before the first write, so
	// every value rides into 304 revalidations too.
	serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares,
		httputil.SecurityHeaders(httputil.SecurityHeadersConfig{
			ContentTypeOptions: securityHeaderNosniff,
			ReferrerPolicy:     securityReferrerPolicy,
			Custom: map[string]string{
				securityRobotsTagHeader: securityRobotsTagNoindex,
			},
		}))

	// CSP nonce mode (opt-in): a per-request nonce replaces 'unsafe-inline'
	// in script-src. Every inline script (theme, htmx self-host marker,
	// toast container, theme toggle, mobile menu, global error handling)
	// carries the nonce via the template shell.
	if os.Getenv("CSP_NONCE") == "1" {
		slog.Info("CSP nonce mode enabled", "flag", "CSP_NONCE")

		serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares,
			httputil.Nonce(httputil.NonceConfig{CSPBuilder: nonceCSP}))
	}

	// CSP report-only dry run (opt-in, plan 05-40 M15/F47): stamps the same
	// nonce-based policy as CSP_NONCE, but as Report-Only — the browser
	// logs would-be violations in its console without enforcing anything.
	// The nonce middleware runs outer (appended above), so the per-request
	// nonce is already in the request context here. A MEANINGFUL run needs
	// CSP_REPORT_ONLY=1 WITH CSP_NONCE=1 (scripts carry no nonce attribute
	// in non-nonce mode, so report-only alone reports every inline script
	// against the empty 'nonce-'); scripts/csp-sweep.js drives it. The
	// 2026-10-17 sweep found zero violations after fixing the two un-nonced
	// singleton scripts it caught (table row-href, copy buttons) — the
	// policy is enforce-clean; flipping means enabling nonce mode in the
	// deploy config.
	if os.Getenv("CSP_REPORT_ONLY") == "1" {
		slog.Info("CSP report-only dry run enabled", "flag", "CSP_REPORT_ONLY")

		serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares,
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().
						Set("Content-Security-Policy-Report-Only", nonceCSP(httputil.NonceFromRequest(r)))
					next.ServeHTTP(w, r)
				})
			})
	}

	serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares,
		httputil.Compression(httputil.DefaultCompressionConfig()),
		web.ETagMiddleware(cacheStats, log),
	)

	return serviceConfig
}

// openServeIndex boots the index side of serve: the CQRS system, the
// projection drain that catches the read model up with whatever ingest
// left unprojected, the FTS index, and the startup rebuild. closeAll
// releases the FTS handle and then the system — defer it as one unit so
// the journal stays readable until the index has flushed.
func openServeIndex(
	ctx context.Context,
	dsn string,
) (*system.System, *search.Index, func(), error) {
	sys, cleanup, err := cli.OpenSystem(ctx, dsn)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		cleanup()

		return nil, nil, nil, err
	}

	index, err := search.Open(ctx, dsn)
	if err != nil {
		cleanup()

		return nil, nil, nil, err
	}

	if err := cli.RebuildOpenIndex(ctx, sys, index); err != nil {
		_ = index.Close()

		cleanup()

		return nil, nil, nil, err
	}

	closeAll := func() {
		_ = index.Close()

		cleanup()
	}

	return sys, index, closeAll, nil
}

// openServeIdentity brings the identity context up. The caller checks the
// -auth-db flag first, so an empty DSN never reaches here. A non-nil oidc
// enables the Pocket ID sign-in lane (validated by oidclogin.New — no
// network at boot).
func openServeIdentity(
	ctx context.Context,
	addr, authDB, authRPID, authOrigin string,
	oidc *identity.OIDCConfig,
) (*identity.Identity, error) {
	rpid, origins := deriveAuthDefaults(addr, authRPID, authOrigin)

	identity, err := identity.OpenIdentity(ctx, identity.IdentityConfig{
		DSN: authDB, RPID: rpid, RPOrigins: origins, OIDC: oidc,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("identity enabled: the first visitor to /login becomes the operator "+
		"(registration closes after the first account); passkeys need HTTPS or localhost",
		"db", authDB, "rpid", rpid, "oidc", oidc != nil)

	return identity, nil
}

// deriveAuthDefaults resolves the WebAuthn Relying Party values. An
// explicit -auth-rpid/-auth-origin wins; an origin without an RPID takes
// the origin's host (WebAuthn needs the RPID to be a registrable suffix
// of the origin host, and the origin host itself is always valid — while
// the listen address on a deployed host is an IP or loopback name no
// browser-facing origin shares). Only a bare address falls back to the
// listen-address mapping: a domain listen address maps directly (host is
// the RPID, http://addr the origin), while an IP or localhost address
// collapses to RPID "localhost" — WebAuthn forbids IP-address RPID
// outright (go-webauthn refuses 127.0.0.1 at construction), and the
// browser-facing address for local dev is http://localhost:<port>.
func deriveAuthDefaults(addr, rpidFlag, originFlag string) (string, []string) {
	if rpidFlag != "" && originFlag != "" {
		return rpidFlag, strings.Split(originFlag, ",")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, ""
	}

	if rpidFlag == "" {
		rpidFlag = deriveRPID(originFlag, host)
	}

	if originFlag == "" {
		originFlag = deriveOrigin(rpidFlag, host, port)
	}

	return rpidFlag, strings.Split(originFlag, ",")
}

// deriveRPID picks the WebAuthn RPID: an origin's host when only origins
// are configured (the RPID must be a registrable suffix of the origin
// host), else the listen host's browser-facing form — an IP or literal
// "localhost" collapses to "localhost" (WebAuthn forbids IP RPIDs).
func deriveRPID(originFlag, host string) string {
	if originFlag != "" {
		origins := strings.Split(originFlag, ",")

		if u, parseErr := url.Parse(
			strings.TrimSpace(origins[0]),
		); parseErr == nil &&
			u.Hostname() != "" {
			return u.Hostname()
		}
	}

	if net.ParseIP(host) != nil || host == "localhost" {
		return "localhost"
	}

	return host
}

// deriveOrigin picks the default WebAuthn origin: the listen host (or
// "localhost" when the RPID collapsed an IP/loopback address), scheme
// http, on the listen port when one is bound.
func deriveOrigin(rpidFlag, host, port string) string {
	originHost := host
	if rpidFlag == "localhost" {
		originHost = "localhost"
	}

	origin := "http://" + originHost
	if port != "" && port != "0" {
		origin += ":" + port
	}

	return origin
}

// nonceCSP is the CSP header used in nonce mode: inline scripts execute only
// with the per-request nonce; the htmx runtime, styles, images, and the
// recording proxy are all same-origin. Inline style ATTRIBUTES (blur-up
// placeholders) keep 'unsafe-inline' in style-src.
func nonceCSP(nonce string) string {
	return "default-src 'self'; script-src 'self' 'nonce-" + nonce + "';" +
		" style-src 'self' 'unsafe-inline'; img-src 'self' data:;" +
		" connect-src 'self'; media-src 'self'; frame-ancestors 'none';" +
		" base-uri 'self'; form-action 'self'"
}
