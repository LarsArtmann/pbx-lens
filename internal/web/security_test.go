package web

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Security-bundle pins (plan 05-40 M15, F46+F48): the private-host
// posture — no crawler anywhere, and an explicit recording-stream
// backstop behind auth.

func TestRobotsTxtDisallowAll(t *testing.T) {
	mux := newTestServer(t)

	status, body := getBody(t, mux, "/robots.txt")
	if status != http.StatusOK {
		t.Fatalf("GET /robots.txt status = %d", status)
	}

	if want := "User-agent: *\nDisallow: /\n"; body != want {
		t.Errorf("robots.txt = %q, want %q", body, want)
	}
}

func TestPagesCarryNoIndex(t *testing.T) {
	mux := newTestServer(t)

	for _, path := range []string{"/", "/search", "/dashboard", "/ops"} {
		_, body := getBody(t, mux, path)
		if !strings.Contains(body, `<meta name="robots" content="noindex">`) {
			t.Errorf("%s missing the noindex meta", path)
		}
	}
}

func TestRecordingThrottleLimitsAndExemptsLoopback(t *testing.T) {
	th := newRecordingThrottle()

	for range recordingBurst {
		if !th.allow("203.0.113.7") {
			t.Fatal("burst requests must pass")
		}
	}

	if th.allow("203.0.113.7") {
		t.Error("request past the burst must be denied")
	}

	for range 3 * recordingBurst {
		if !th.allow("127.0.0.1") {
			t.Fatal("loopback must never be throttled")
		}
	}

	for i := range recordingMaxHosts + 10 {
		if !th.allow(fmt.Sprintf("198.51.100.%d", i%256)) && i < recordingBurst {
			t.Fatalf("fresh host %d denied before its burst", i)
		}
	}

	th.mu.Lock()
	size := len(th.buckets)
	th.mu.Unlock()

	if size > recordingMaxHosts {
		t.Errorf("bucket map grew to %d, want ≤ %d (bounded FIFO)", size, recordingMaxHosts)
	}
}

func TestRecordingHandlerAnswers429WhenThrottled(t *testing.T) {
	mux := newTestServerWith(t, Options{}, webObservation("conv-throttle", testCallStarted()))

	for range recordingBurst {
		getBody(t, mux, "/calls/telnyx-conversation/conv-throttle/recording")
	}

	status, body := getBody(t, mux, "/calls/telnyx-conversation/conv-throttle/recording")
	if status != http.StatusTooManyRequests {
		t.Errorf("throttled recording status = %d, want 429 (body: %s)", status, body)
	}
}
