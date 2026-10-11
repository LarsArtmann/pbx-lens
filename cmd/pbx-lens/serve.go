package main

import (
	"context"
	"flag"
	"log"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-appkit/systemd"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/search"
	"github.com/larsartmann/pbx-lens/internal/web"
)

const (
	defaultAddr  = "127.0.0.1:8300"
	serveTimeout = 5 * time.Minute
)

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

	serviceConfig := appkit.ServiceConfig{Addr: *addr}

	// sd_notify (T7/F23): READY=1 after the listener binds, STOPPING=1 at
	// the drain, WATCHDOG keepalive while serving. A no-op outside systemd
	// (NOTIFY_SOCKET unset), so the same binary serves localhost dev and
	// the service manager unchanged.
	systemd.Install(&serviceConfig)

	if *htpasswdPath != "" {
		auth, err := web.HtpasswdMiddleware(*htpasswdPath)
		if err != nil {
			fatal(err)
		}

		serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares, auth)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		fatal(err)
	}

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

	web.Register(svc.Mux, sys, web.Options{Recordings: recordings, Search: index, Log: log.Printf})

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
		CDRFile:        *cdrFile,
		WebhookLog:     *webhookLog,
		TranscriptsDir: *transcriptsDir,
		MaxAge:         *maxAge,
	}, log.Printf)

	// The serve lifetime is signal-owned (Run handles SIGINT/SIGTERM) — no
	// timeout context here, or the server would die with it.
	if err := svc.Run(context.Background()); err != nil {
		fatal(err)
	}
}
