package web

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
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
// under this deadline and the WAV stream must deliver its response headers
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
	// Telnyx serves kind telnyx-conversation: the fresh signed WAV URL is
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
	view, found, err := CallDetail(r.Context(), s.sys, r.PathValue("kind"), r.PathValue("id"))
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	if !found {
		http.Error(w, "no indexed call with that id", http.StatusNotFound)

		return
	}

	if view.RecordingRef == "" && view.Kind != calls.SourceFreeSwitchCDR {
		http.Error(w, "no recording reference indexed for this call", http.StatusNotFound)

		return
	}

	switch view.Kind {
	case calls.SourceTelnyxConversation:
		s.proxyTelnyxRecording(w, r, view.RecordingRef)
	case calls.SourceFreeSwitchCDR:
		s.serveLocalRecording(w, r, view)
	case calls.SourceWebhookLog, calls.SourceAgentTranscript:
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

	wav := recording.DownloadURLs.WAV
	if wav == "" {
		http.Error(w, "Telnyx reports no WAV download for this recording", http.StatusNotFound)

		return
	}

	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodGet, wav, nil)
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

	copyRecordingResponse(w, response)
}

// copyRecordingResponse streams an upstream recording response to the
// client, passing through the audio headers (defaulting the content type)
// and forcing the response uncacheable since signed URLs expire.
func copyRecordingResponse(w http.ResponseWriter, response *http.Response) {
	for _, header := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if value := response.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "audio/wav")
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

	for _, stem := range localRecordingStems(view) {
		name := filepath.Base(stem)
		if name != stem { // paranoia: reject anything path-shaped
			continue
		}

		path := filepath.Join(s.recordings.RecordingsDir, name+".wav")
		if _, err := os.Stat(path); err != nil {
			continue
		}

		http.ServeFile(w, r, path)

		return
	}

	http.Error(w, "no local recording file found for this call", http.StatusNotFound)
}

// localRecordingStemCandidates bounds the candidate stems below (ref,
// {uuid}_{dest}, bare uuid).
const localRecordingStemCandidates = 3

// localRecordingStems lists candidate file stems, most specific first: the
// indexed ref, then the {uuid}_{dest} dialplan naming, then the bare uuid.
func localRecordingStems(view calls.CallView) []string {
	stems := make([]string, 0, localRecordingStemCandidates)

	if view.RecordingRef != "" {
		stems = append(stems, view.RecordingRef)
	}

	if dest := view.ToNumber; dest != "" {
		stems = append(stems, string(view.NativeID)+"_"+dest)
	}

	return append(stems, string(view.NativeID))
}
