package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestDumpPagesForVisualAudit writes rendered pages to /tmp for eyeballing.
// Temporary harness — not part of the suite contract.
func TestDumpPagesForVisualAudit(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)

	inbound := webObservation("conv-in", started.Add(time.Minute))
	inbound.Direction = calls.DirectionInbound
	inbound.IsAI = true
	inbound.MOS = 4.2

	seeded := []calls.Observation{webObservation("conv-out", started), inbound}

	mux := newTestServerWithSearch(t, seeded...)

	pages := map[string]string{
		"list":      "/",
		"dashboard": "/dashboard",
		"search":    "/search?q=hello",
		"detail":    "/calls/telnyx-conversation/conv-out",
		"empty":     "/search?q=zzzznothing",
	}

	for name, path := range pages {
		status, body := getBody(t, mux, path)
		if status != 200 {
			t.Fatalf("%s status = %d", path, status)
		}

		out := filepath.Join("/tmp", "pbxlens-baseline", name+".html")
		if err := os.WriteFile(out, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", out, err)
		}
	}
}
