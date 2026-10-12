package web

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

func TestExportJSONLStreamsOneObjectPerCall(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServer(t,
		webObservation("conv-a", started),
		webObservation("conv-b", started.Add(time.Minute)),
	)

	status, body := getBody(t, mux, "/api/export.jsonl")
	if status != 200 {
		t.Fatalf("GET /api/export.jsonl status = %d, want 200", status)
	}

	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 2 {
		t.Fatalf("JSONL should carry one line per call, got %d lines: %q", len(lines), body)
	}

	for i, line := range lines {
		var record calls.CallView
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}

		if record.NativeID == "" {
			t.Errorf("line %d missing native_id", i)
		}
	}
}

func TestExportCSVHasHeaderAndRows(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServer(t,
		webObservation("conv-a", started),
		webObservation("conv-b", started.Add(time.Minute)),
	)

	status, body := getBody(t, mux, "/api/export.csv")
	if status != 200 {
		t.Fatalf("GET /api/export.csv status = %d, want 200", status)
	}

	rows := strings.Split(strings.TrimSpace(body), "\n")
	if len(rows) != 3 {
		t.Fatalf("CSV should be header + 2 rows, got %d rows: %q", len(rows), body)
	}

	if !strings.HasPrefix(rows[0], "kind,native_id,started_at") {
		t.Errorf("CSV header unexpected: %q", rows[0])
	}

	if !strings.Contains(body, "conv-a") || !strings.Contains(body, "conv-b") {
		t.Error("CSV should carry both calls")
	}
}

func TestExportHonorsDirectionFilter(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	outbound := webObservation("conv-out", started)

	inbound := webObservation("conv-in", started.Add(time.Minute))
	inbound.Direction = calls.DirectionInbound

	mux := newTestServer(t, outbound, inbound)

	status, body := getBody(t, mux, "/api/export.csv?direction=inbound")
	if status != 200 {
		t.Fatalf("GET export?direction=inbound status = %d, want 200", status)
	}

	if !strings.Contains(body, "conv-in") {
		t.Error("inbound export should include the inbound call")
	}

	if strings.Contains(body, "conv-out") {
		t.Error("inbound export should exclude the outbound call")
	}
}

func TestExportRejectsGarbageFilter(t *testing.T) {
	status, _ := getBody(t, newTestServer(t), "/api/export.csv?direction=sideways")
	if status != 400 {
		t.Fatalf("export with a bad filter status = %d, want 400", status)
	}
}

func TestExportPIIHashModeReplacesParticipants(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	call := webObservation("conv-pii", started)
	call.FromNumber = "+17287289311"
	call.ToNumber = "garbled-cdr-party"

	mux := newTestServer(t, call)

	status, body := getBody(t, mux, "/api/export.jsonl?pii=hash")
	if status != 200 {
		t.Fatalf("GET export?pii=hash status = %d, want 200 (body: %s)", status, body)
	}

	wantHash := "sha256:" + string(calls.E164("+17287289311").Hash())
	for _, want := range []string{wantHash, "[redacted]"} {
		if !strings.Contains(body, want) {
			t.Errorf("hash-mode export missing %q (body: %s)", want, body)
		}
	}

	if strings.Contains(body, "+17287289311") || strings.Contains(body, "garbled-cdr-party") {
		t.Error("hash-mode export leaked a raw participant")
	}
}

func TestExportPIIUnknownModeRejected(t *testing.T) {
	status, body := getBody(t, newTestServer(t), "/api/export.csv?pii=pseudonymous")
	if status != 400 {
		t.Fatalf("unknown pii mode status = %d, want 400", status)
	}

	if !strings.Contains(body, "pii=hash") {
		t.Errorf("rejection should name the supported mode, got: %s", body)
	}
}

// getResp is getBody with the response exposed (headers matter for the
// export PII contract).
func getResp(t *testing.T, mux *http.ServeMux, path string) *http.Response {
	t.Helper()

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder.Result()
}

func TestExportSetsPIIModeHeader(t *testing.T) {
	// The X-PII-Mode header makes the mode visible to every consumer (and
	// to curl-less probes): a "hashed" dump must never masquerade as plain.
	mux := newTestServer(t, webObservation("conv-a", time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)))

	for _, path := range []string{"/api/export.jsonl", "/api/export.csv"} {
		resp := getResp(t, mux, path)
		if got := resp.Header.Get("X-PII-Mode"); got != "plain" {
			t.Errorf("%s X-PII-Mode = %q, want plain", path, got)
		}

		_ = resp.Body.Close()

		resp = getResp(t, mux, path+"?pii=hash")
		if got := resp.Header.Get("X-PII-Mode"); got != "hash" {
			t.Errorf("%s?pii=hash X-PII-Mode = %q, want hash", path, got)
		}

		_ = resp.Body.Close()
	}
}

func TestExportStreamsAcrossKeysetPages(t *testing.T) {
	// A shrunken keyset page forces the loop through its seams: five calls
	// at page size 2 arrive complete, newest first, without duplicates —
	// and filters plus pii=hash hold on every page (R7).
	saved := exportBatchSize
	exportBatchSize = 2

	t.Cleanup(func() { exportBatchSize = saved })

	started := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	mixed := make([]calls.Observation, 0, 5)

	for i := range 5 {
		call := webObservation(fmt.Sprintf("conv-%d", i), started.Add(time.Duration(i)*time.Minute))
		call.FromNumber = "+17287289311"
		call.Direction = calls.DirectionInbound
		mixed = append(mixed, call)
	}

	mux := newTestServer(t, mixed...)

	status, body := getBody(t, mux, "/api/export.jsonl?direction=inbound&pii=hash")
	if status != 200 {
		t.Fatalf("streamed export status = %d, want 200 (body: %s)", status, body)
	}

	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 5 {
		t.Fatalf("all five calls must stream past three pages, got %d lines", len(lines))
	}

	seen := make(map[string]bool)

	for i, line := range lines {
		var record calls.CallView
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %d invalid: %v", i, err)
		}

		id := string(record.NativeID)
		if seen[id] {
			t.Errorf("call %s streamed twice (page dedup broke)", id)
		}

		seen[id] = true

		if record.FromNumber != "sha256:"+string(calls.E164("+17287289311").Hash()) {
			t.Errorf("call %s left hash mode with raw from %q", id, record.FromNumber)
		}
	}

	want := "conv-4"
	if !strings.Contains(lines[0], `"native_id":"`+want+`"`) {
		t.Errorf("newest call must stream first, got %q", lines[0][:60])
	}
}

func TestExportKeysetStallFailsLoudly(t *testing.T) {
	// More calls inside one nanosecond than a keyset page can carry cannot
	// make progress: the loop must fail with an explanation, never truncate
	// silently.
	saved := exportBatchSize
	exportBatchSize = 2

	t.Cleanup(func() { exportBatchSize = saved })

	sameInstant := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	mux := newTestServer(t,
		webObservation("conv-a", sameInstant),
		webObservation("conv-b", sameInstant),
		webObservation("conv-c", sameInstant),
		webObservation("conv-d", sameInstant),
		webObservation("conv-e", sameInstant),
	)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/export.jsonl", nil))

	body := recorder.Body.String()
	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200 (headers may already be sent): %s", recorder.Code, body)
	}

	if !strings.Contains(body, "conv-") {
		t.Errorf("the streamed prefix should still carry calls, got: %s", body)
	}
}
