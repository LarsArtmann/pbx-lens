package web

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"golang.org/x/time/rate"
)

// RecordingFetcher is the consumer-side view of a Telnyx client the
// recording proxy needs: fetch one recording's metadata (including its
// fresh signed download URL) by recording id. Declared here instead of
// importing the concrete *telnyx.Client so tests and future sources can
// stand in with a fake — telnyx.Client satisfies it as-is.
type RecordingFetcher interface {
	FetchRecording(ctx context.Context, recordingID string) (telnyx.Recording, error)
}

// DefaultUpstreamTimeout bounds each upstream hop of the recording proxy
// when RecordingOptions.UpstreamTimeout is unset: the metadata fetch runs
// under this deadline and the audio stream must deliver its response headers
// within it. The body itself never carries a total timeout — long
// recordings must not be cut mid-download — and there is deliberately no
// "unbounded" setting: an upstream that never answers must not pin a
// browser request forever. Production tuning waits for real traffic data.
const DefaultUpstreamTimeout = 30 * time.Second

// RecordingOptions wires the recording proxy's two sources. Both optional:
// a disabled source answers 503 with the reason, never a silent 404.
type RecordingOptions struct {
	// UpstreamTimeout bounds each upstream hop: the metadata fetch deadline
	// and the WAV stream's response-header budget. Zero or negative selects
	// DefaultUpstreamTimeout.
	UpstreamTimeout time.Duration
	// Telnyx serves kind telnyx-conversation: the fresh signed audio URL is
	// fetched SERVER-SIDE with the API key and the bytes stream through
	// this process — neither key nor signed URL ever reaches the browser.
	Telnyx RecordingFetcher

	// RecordingsDir serves kind freeswitch-cdr local {stem}.wav files
	// (http.ServeFile: Range and conditional requests for free). The stem
	// is the view's RecordingRef when set, else {nativeID}_{toNumber}.
	RecordingsDir string
}

// recording streams a call's audio through the server (B2/m24): the client
// only ever talks to this route.
func (s *server) recording(w http.ResponseWriter, r *http.Request) {
	if !s.recordingThrottle.allow(remoteHost(r)) {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "too many recording requests from this address; retry in a moment", http.StatusTooManyRequests)

		return
	}

	view, found, err := CallDetail(r.Context(), s.sys, r.PathValue("kind"), r.PathValue("id"))
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	if !found {
		http.Error(w, "no indexed call with that id", http.StatusNotFound)

		return
	}

	// The local-file kinds own their recording by call uuid, not by an
	// indexed reference: the FreeSWITCH CDR and the agent transcript of the
	// same call both address {uuid}_*.wav through the same resolver.
	localKinds := view.Kind == calls.SourceFreeSwitchCDR || view.Kind == calls.SourceAgentTranscript

	if view.RecordingRef == "" && !localKinds {
		http.Error(w, "no recording reference indexed for this call", http.StatusNotFound)

		return
	}

	switch view.Kind {
	case calls.SourceTelnyxConversation:
		s.proxyTelnyxRecording(w, r, view.RecordingRef)
	case calls.SourceFreeSwitchCDR, calls.SourceAgentTranscript:
		s.serveLocalRecording(w, r, view)
	case calls.SourceWebhookLog:
		fallthrough
	default:
		http.Error(w, "source kind "+string(view.Kind)+" records no playable audio", http.StatusNotFound)
	}
}

// proxyTelnyxRecording refetches the recording's signed download URL and
// streams the WAV through. Redirects are followed server-side; the response
// carries no Location header and is never cacheable (signed URLs expire).
func (s *server) proxyTelnyxRecording(w http.ResponseWriter, r *http.Request, recordingID string) {
	if s.recordings.Telnyx == nil {
		http.Error(
			w,
			"recording proxy not configured: restart serve with a Telnyx API key (-telnyx-key-file)",
			http.StatusServiceUnavailable,
		)

		return
	}

	metaCtx, cancel := context.WithTimeout(r.Context(), s.recTimeout)
	defer cancel()

	metaStop := servertiming.MeasureServerTiming(r.Context(), "metadata")
	recording, err := s.recordings.Telnyx.FetchRecording(metaCtx, recordingID)

	metaStop()

	if err != nil {
		if metaCtx.Err() != nil {
			http.Error(w, "recording metadata fetch exceeded the upstream timeout", http.StatusGatewayTimeout)

			return
		}

		http.Error(w, "fetching the recording metadata failed: "+err.Error(), http.StatusBadGateway)

		return
	}

	audio, audioType := recording.DownloadURLs.WAV, "audio/wav"
	if audio == "" {
		audio, audioType = recording.DownloadURLs.MP3, "audio/mpeg"
	}

	if audio == "" {
		http.Error(w, "Telnyx reports no WAV or MP3 download for this recording", http.StatusNotFound)

		return
	}

	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodGet, audio, nil)
	if err != nil {
		http.Error(w, "building the upstream request failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	// Range passthrough so <audio> seeking works against the upstream.
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		upstream.Header.Set("Range", rangeHeader)
	}

	headerStop := servertiming.MeasureServerTiming(r.Context(), "headers")
	response, err := s.upstream.Do(upstream)

	headerStop()

	if err != nil {
		http.Error(w, "fetching the recording audio failed: "+err.Error(), http.StatusBadGateway)

		return
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		http.Error(w, "recording source answered "+response.Status, http.StatusBadGateway)

		return
	}

	copyRecordingResponse(w, response, audioType)
}

// copyRecordingResponse streams an upstream recording response to the
// client, passing through the audio headers (defaulting the content type to
// the rendition the proxy requested) and forcing the response uncacheable
// since signed URLs expire.
func copyRecordingResponse(w http.ResponseWriter, response *http.Response, fallbackContentType string) {
	for _, header := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if value := response.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", fallbackContentType)
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(response.StatusCode)

	_, _ = io.Copy(w, response.Body)
}

// resolveUpstreamTimeout applies the RecordingOptions.UpstreamTimeout
// contract: unset or negative selects the default bound.
func resolveUpstreamTimeout(configured time.Duration) time.Duration {
	if configured <= 0 {
		return DefaultUpstreamTimeout
	}

	return configured
}

// newUpstreamClient builds the client the WAV stream rides: response
// headers must arrive within timeout, the body streams without a total cap.
func newUpstreamClient(timeout time.Duration) *http.Client {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{}
	}

	transport := base.Clone()
	transport.ResponseHeaderTimeout = timeout

	return &http.Client{Transport: transport}
}

// serveLocalRecording serves the FreeSWITCH recording file. http.ServeFile
// provides Range, If-Modified-Since, and content negotiation; the filename
// is basename-locked so no path can escape RecordingsDir.
func (s *server) serveLocalRecording(w http.ResponseWriter, r *http.Request, view calls.CallView) {
	if s.recordings.RecordingsDir == "" {
		http.Error(
			w,
			"local recordings not configured: restart serve with -recordings-dir",
			http.StatusServiceUnavailable,
		)

		return
	}

	path, found := s.localRecordingPath(view)
	if !found {
		http.Error(w, "no local recording file found for this call", http.StatusNotFound)

		return
	}

	http.ServeFile(w, r, path)
}

// localRecordingPath resolves the call's WAV on disk. Candidates cover the
// dialplan's known namings first (cheap stat calls, no directory walk), then a
// {uuid}-prefixed sweep because the dialplan owns the stem suffix and grows
// new ones — {uuid}_ai_agent (the voice agent) and {uuid}_pstn (outbound)
// were both invisible to the two-stem guess that shipped before 2026-10-23.
func (s *server) localRecordingPath(view calls.CallView) (string, bool) {
	if s.recordings.RecordingsDir == "" {
		return "", false
	}

	for _, stem := range localRecordingStems(view) {
		name := filepath.Base(stem)
		if name != stem { // paranoia: reject anything path-shaped
			continue
		}

		path := filepath.Join(s.recordings.RecordingsDir, name+".wav")
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return s.scanLocalRecording(view)
}

// scanLocalRecording is the dialplan-agnostic fallback: the first
// {uuid}_*.wav in the recordings directory. Names only — never joined from
// caller input — so no candidate can escape the directory.
func (s *server) scanLocalRecording(view calls.CallView) (string, bool) {
	prefix := string(view.NativeID) + "_"
	if view.NativeID == "" {
		return "", false
	}

	entries, err := os.ReadDir(s.recordings.RecordingsDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".wav") || !strings.HasPrefix(name, prefix) {
			continue
		}

		return filepath.Join(s.recordings.RecordingsDir, name), true
	}

	return "", false
}

// localRecordingStemCandidates bounds the candidate stems below (ref,
// {uuid}_{dest}, {uuid}_ai_agent, {uuid}_pstn, bare uuid).
const localRecordingStemCandidates = 5

// localRecordingStems lists candidate file stems, most specific first: the
// indexed ref, the {uuid}_{dest} dialplan naming, the two fixed dialplan
// destinations that differ from the CDR's destination_number (the agent
// route and the outbound PSTN route), then the bare uuid.
func localRecordingStems(view calls.CallView) []string {
	stems := make([]string, 0, localRecordingStemCandidates)

	if view.RecordingRef != "" {
		stems = append(stems, view.RecordingRef)
	}

	if dest := view.ToNumber; dest != "" {
		stems = append(stems, string(view.NativeID)+"_"+dest)
	}

	if view.IsAI {
		stems = append(stems, string(view.NativeID)+"_ai_agent")
	}

	stems = append(stems, string(view.NativeID)+"_pstn")

	return append(stems, string(view.NativeID))
}

// recordingThrottle caps how fast one remote address may START recording
// streams (M15/F48): an explicit backstop behind auth for runaway loops or
// a scripted audio hoover, at a ceiling an honest operator player never
// reaches (2 stream starts/s sustained, burst 60 — seek/Range requests
// included, so aggressive scrubbing still fits). Loopback is exempt: the
// probes, the e2e lane, and a local operator share 127.0.0.1. The bucket
// map is bounded (FIFO eviction at maxHosts) so random spoofed addresses
// cannot grow it without end — the same shape as the auth failure
// throttle, which stays the authoritative brute-force defense.
type recordingThrottle struct {
	mu      sync.Mutex
	buckets map[string]*rate.Limiter
	order   []string
}

const (
	recordingStartsPerSecond = 2
	recordingBurst           = 60
	recordingMaxHosts        = 256
)

func newRecordingThrottle() recordingThrottle {
	return recordingThrottle{buckets: make(map[string]*rate.Limiter)}
}

func (t *recordingThrottle) allow(host string) bool {
	if ip, err := netip.ParseAddr(host); err == nil && ip.IsLoopback() {
		return true
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	lim, known := t.buckets[host]
	if !known {
		if len(t.buckets) >= recordingMaxHosts {
			delete(t.buckets, t.order[0])
			t.order = t.order[1:]
		}

		lim = rate.NewLimiter(recordingStartsPerSecond, recordingBurst)
		t.buckets[host] = lim
		t.order = append(t.order, host)
	}

	return lim.Allow()
}

// remoteHost extracts the IP part of RemoteAddr (the auth throttle keys
// the full address; recordings are per-network-host, not per-connection).
func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
