package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-appkit/systemd"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/httputil"
	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/cli"
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

// serve runs the read-only web UI until SIGINT/SIGTERM (appkit Run owns the
// signal handling and the graceful drain). One drain at startup catches the
// read model up with whatever the ingest commands left unprojected.
func serve(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	addr := fs.String("addr", defaultAddr, "listen address")
	keyFile := fs.String("telnyx-key-file", "", "Telnyx API key file for the recording proxy (default: disabled)")
	recordingsDir := fs.String(
		"recordings-dir",
		"/var/lib/telephony/recordings",
		"local FreeSWITCH recordings directory (stems tried: indexed ref, "+
			"{uuid}_{dest}.wav, bare uuid; empty = local playback disabled)",
	)
	recTimeout := fs.Duration(
		"recording-timeout",
		web.DefaultUpstreamTimeout,
		"upstream bound for the recording proxy: metadata deadline + WAV response headers "+
			"(0 = default; the audio body itself streams unbounded)",
	)
	authDB := fs.String(
		"auth-db",
		"",
		"sqlite identity database enabling passkey login (empty = no auth for localhost dev)",
	)
	authRPID := fs.String(
		"auth-rpid",
		"",
		"WebAuthn Relying Party ID — the registrable domain the browser sees "+
			"(default: the host part of -addr)",
	)
	authOrigin := fs.String(
		"auth-origin",
		"",
		"comma-separated WebAuthn origins that must byte-match the browser address bar, "+
			"scheme included (default: http://<addr>)",
	)
	pollEvery := fs.Duration(
		"poll-every",
		cli.DefaultPollEvery,
		"poll cadence for keeping the index fresh (0 = disabled)",
	)
	pollSince := fs.Duration(
		"poll-since",
		cli.DefaultPollSince,
		"Telnyx look-back window per poll",
	)
	pollDirection := fs.String(
		"poll-direction",
		"outbound",
		"caller-declared direction for polled Telnyx conversations",
	)
	aiDID := fs.String(
		"ai-did",
		"",
		"comma-separated AI-facing DIDs: polled conversations touching one are inbound (empty = keep declared direction)",
	)
	maxAge := fs.Duration(
		"max-age",
		0,
		"refuse polled observations older than this (0 = keep everything)",
	)
	cdrFile := fs.String("cdr-file", "", "FreeSWITCH Master.csv re-ingested every poll (empty = skip)")
	webhookLog := fs.String("webhook-log", "", "receiver webhook JSONL re-ingested every poll (empty = skip)")
	transcriptsDir := fs.String("transcripts-dir", "", "agent transcripts dir re-scanned every poll (empty = skip)")
	wiredOps := fs.Bool(
		"wired-ops",
		false,
		"dev probe fixture: serve a canned STALE poller snapshot on /ops and /api/ops instead of the live counters",
	)
	_ = fs.Parse(args)

	requireDB(fs, *db)

	pollDir, err := cli.ParseDirection(*pollDirection)
	if err != nil {
		fatal(err)
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), serveTimeout)
	defer cancel()

	sys, index, closeAll, err := openServeIndex(setupCtx, *db)
	if err != nil {
		fatal(err)
	}

	defer closeAll()

	// The identity context (passkey login) is opt-in via -auth-db: without
	// it the serve runs unauthenticated — the localhost dev posture the
	// probes rely on.
	var identity *web.Identity

	if *authDB != "" {
		identity, err = openServeIdentity(setupCtx, *addr, *authDB, *authRPID, *authOrigin)
		if err != nil {
			fatal(err)
		}

		defer func() { _ = identity.Close() }()
	}
	// The composition the read-only surface runs on: no fixed response
	// deadline (exports and recording WAVs stream unbounded — the appkit
	// default 30s WriteTimeout would truncate them mid-stream), the build
	// visible over HTTP (/version + appkit_build_info), the Prometheus
	// surface behind the same posture as every route (the identity gate
	// when -auth-db is set, loopback-only dev otherwise — hence the explicit
	// opt-in), and the extras in wire order: auth gates everything,
	// Server-Timing observes the full handler span, Compression sits outer
	// of ETag so tags hash identity bodies (go-etag's ordering doctrine).
	cacheStats := web.NewCacheCounters()

	var authMW func(http.Handler) http.Handler

	if identity != nil {
		authMW = identity.Middleware()
	}

	serviceConfig := buildServiceConfig(*addr, authMW, cacheStats, slogLog)

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
		fatal(err)
	}

	if identity != nil {
		identity.Register(svc.Mux)
	}

	counters := cli.NewSourceCounters(*pollEvery)

	opsProvider, recordings, telnyxClient := wireServeSources(
		counters, *wiredOps, *recordingsDir, *recTimeout, *keyFile,
	)

	web.Register(
		svc.Mux,
		sys,
		web.Options{
			Recordings: recordings, Search: index, Log: slogLog,
			Ops: opsProvider, Cache: cacheStats,
			RetentionMaxAge: *maxAge,
		},
	)

	// The poller keeps the index fresh until shutdown; each pass drains
	// projections and rebuilds the FTS index so / and /search never serve
	// yesterday's state while serve runs.
	pollCtx, stopPoller := context.WithCancel(context.Background())
	defer stopPoller()

	go cli.PollLoop(pollCtx, sys, index, cli.PollConfig{
		Every:          *pollEvery,
		Telnyx:         telnyxClient,
		SinceAgo:       *pollSince,
		Direction:      pollDir,
		AIDIDs:         telnyx.ParseAIDIDs(*aiDID),
		CDRFile:        *cdrFile,
		WebhookLog:     *webhookLog,
		TranscriptsDir: *transcriptsDir,
		MaxAge:         *maxAge,
		Counters:       counters,
	}, slogLog)

	// The serve lifetime is signal-owned (Run handles SIGINT/SIGTERM) — no
	// timeout context here, or the server would die with it.
	if err := svc.Run(context.Background()); err != nil {
		fatal(err)
	}
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
) (func() any, web.RecordingOptions, *telnyx.Client) {
	opsProvider := counters.SnapshotAny
	if wiredOps {
		opsProvider = cannedStaleOps
	}

	recordings := web.RecordingOptions{RecordingsDir: recordingsDir, UpstreamTimeout: recTimeout}

	var telnyxClient *telnyx.Client

	if keyFile != "" {
		key, err := cli.ResolveKey(keyFile)
		if err != nil {
			fatal(err)
		}

		telnyxClient = telnyx.NewClient(key, "")
		recordings.Telnyx = telnyxClient
	}

	return opsProvider, recordings, telnyxClient
}

// cannedStaleOps is the -wired-ops fixture payload: the wire shape /api/ops
// serves, carrying an unambiguous stale verdict (age 2h, stale=true).
func cannedStaleOps() any {
	const (
		fixtureOfferedCDR     = 12
		fixtureRecordedCDR    = 12
		fixtureOfferedTelnyx  = 8
		fixtureRecordedTelnyx = 8
		fixturePassAgeSecs    = 7200.0
	)

	return map[string]any{
		"sources": map[string]any{
			"freeswitch-cdr": map[string]any{
				"offered": fixtureOfferedCDR, "recorded": fixtureRecordedCDR, "corrected": 0, "rejected": []string{},
			},
			"telnyx-conversation": map[string]any{
				"offered":   fixtureOfferedTelnyx,
				"recorded":  fixtureRecordedTelnyx,
				"corrected": 0,
				"rejected":  []string{},
			},
		},
		"last_pass":             time.Now().Add(-2 * time.Hour).UTC(),
		"last_pass_age_seconds": fixturePassAgeSecs,
		"stale":                 true,
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
	addr string,
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
					w.Header().Set("Content-Security-Policy-Report-Only", nonceCSP(httputil.NonceFromRequest(r)))
					next.ServeHTTP(w, r)
				})
			})
	}

	serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares,
		servertiming.ServerTimingMiddleware(),
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
// -auth-db flag first, so an empty DSN never reaches here.
func openServeIdentity(
	ctx context.Context,
	addr, authDB, authRPID, authOrigin string,
) (*web.Identity, error) {
	rpid, origins := deriveAuthDefaults(addr, authRPID, authOrigin)

	identity, err := web.OpenIdentity(ctx, web.IdentityConfig{
		DSN: authDB, RPID: rpid, RPOrigins: origins,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("identity enabled: the first visitor to /login becomes the operator "+
		"(registration closes after the first account); passkeys need HTTPS or localhost",
		"db", authDB, "rpid", rpid)

	return identity, nil
}

// deriveAuthDefaults resolves the WebAuthn Relying Party values. An
// explicit -auth-rpid/-auth-origin wins; otherwise a domain listen address
// maps directly (host is the RPID, http://addr the origin), while an IP or
// localhost address collapses to RPID "localhost" — WebAuthn forbids
// IP-address RPID outright (go-webauthn refuses 127.0.0.1 at construction),
// and the browser-facing address for local dev is http://localhost:<port>.
func deriveAuthDefaults(addr, rpidFlag, originFlag string) (string, []string) {
	if rpidFlag != "" && originFlag != "" {
		return rpidFlag, strings.Split(originFlag, ",")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, ""
	}

	if rpidFlag == "" {
		if net.ParseIP(host) != nil || host == "localhost" {
			rpidFlag = "localhost"
		} else {
			rpidFlag = host
		}
	}

	if originFlag == "" {
		originHost := host
		if rpidFlag == "localhost" {
			originHost = "localhost"
		}

		originFlag = "http://" + originHost
		if port != "" && port != "0" {
			originFlag += ":" + port
		}
	}

	return rpidFlag, strings.Split(originFlag, ",")
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
