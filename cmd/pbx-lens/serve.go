package main

import (
	"context"
	"flag"
	"log/slog"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-appkit/systemd"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/search"
	"github.com/larsartmann/pbx-lens/internal/web"
)

const (
	defaultAddr  = "127.0.0.1:8300"
	serveTimeout = 5 * time.Minute
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
	htpasswdPath := fs.String(
		"htpasswd",
		"",
		"bcrypt htpasswd file gating every route (htpasswd -B; empty = no auth for localhost dev)",
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
	_ = fs.Parse(args)

	requireDB(fs, *db)

	pollDir, err := cli.ParseDirection(*pollDirection)
	if err != nil {
		fatal(err)
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), serveTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(setupCtx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	if err := cli.DrainProjections(setupCtx, sys); err != nil {
		fatal(err)
	}

	index, err := search.Open(setupCtx, *db)
	if err != nil {
		fatal(err)
	}

	defer func() { _ = index.Close() }()

	if err := cli.RebuildOpenIndex(setupCtx, sys, index); err != nil {
		fatal(err)
	}

	// The composition the read-only surface runs on: no fixed response
	// deadline (exports and recording WAVs stream unbounded — the appkit
	// default 30s WriteTimeout would truncate them mid-stream), the build
	// visible over HTTP (/version + appkit_build_info), the Prometheus
	// surface behind the same posture as every route (htpasswd middleware
	// when configured, loopback-only dev otherwise — hence the explicit
	// opt-in), and the extras in wire order: auth gates everything,
	// Server-Timing observes the full handler span, Compression sits outer
	// of ETag so tags hash identity bodies (go-etag's ordering doctrine).
	cacheStats := web.NewCacheCounters()

	serviceConfig, err := buildServiceConfig(*addr, *htpasswdPath, cacheStats, slogLog)
	if err != nil {
		fatal(err)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		fatal(err)
	}

	counters := cli.NewSourceCounters()

	recordings := web.RecordingOptions{RecordingsDir: *recordingsDir, UpstreamTimeout: *recTimeout}

	var telnyxClient *telnyx.Client

	if *keyFile != "" {
		key, err := cli.ResolveKey(*keyFile)
		if err != nil {
			fatal(err)
		}

		telnyxClient = telnyx.NewClient(key, "")
		recordings.Telnyx = telnyxClient
	}

	web.Register(
		svc.Mux,
		sys,
		web.Options{Recordings: recordings, Search: index, Log: slogLog, Ops: counters.SnapshotAny, Cache: cacheStats},
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

// buildServiceConfig assembles the serve composition. sd_notify (T7/F23)
// installs first: READY=1 after the listener binds, STOPPING=1 at the
// drain, WATCHDOG keepalive while serving — a no-op outside systemd, so the
// same binary serves localhost dev and the service manager unchanged. The
// cache counters and log hook ride in so the ETag middleware's outcomes
// surface on /api/ops (R26).
func buildServiceConfig(
	addr, htpasswdPath string,
	cacheStats *web.CacheCounters,
	log func(msg string, args ...any),
) (appkit.ServiceConfig, error) {
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

	if htpasswdPath != "" {
		auth, err := web.HtpasswdMiddleware(htpasswdPath)
		if err != nil {
			return appkit.ServiceConfig{}, err
		}

		serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares, auth)
	}

	serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares,
		web.ServerTimingMiddleware(),
		httputil.Compression(httputil.DefaultCompressionConfig()),
		web.ETagMiddleware(cacheStats, log),
	)

	return serviceConfig, nil
}
