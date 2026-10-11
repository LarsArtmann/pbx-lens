package web

import (
	"encoding/json/v2"
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
