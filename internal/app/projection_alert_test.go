package app_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/app"
)

// TestDomainConfigWiresProjectionFailureAlert pins plan T16: the composition
// root must hand the projection host a failure callback, because a worker
// that exhausts its restart budget is otherwise silent — the read model just
// stops advancing. The callback is the first witness (journald ERROR at the
// moment it happens); /health/ready is the second (the worker reads as
// WorkerFailed and the predicate turns the probe 503).
//
// The HostOption itself is a func, so the pin is behavioural: capture the
// default logger, invoke the exported callback, and assert the record is an
// ERROR naming the projection and carrying the terminal error.
func TestDomainConfigWiresProjectionFailureAlert(t *testing.T) {
	if got := len(app.DomainConfig().ProjectionHostOptions); got == 0 {
		t.Fatal("DomainConfig() wires no ProjectionHostOptions: a failed worker would go unreported")
	}

	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	var captured bytes.Buffer

	slog.SetDefault(slog.New(slog.NewTextHandler(&captured, &slog.HandlerOptions{Level: slog.LevelDebug})))

	app.OnProjectionFailed("call_views", "sqlite: database is locked")

	got := captured.String()
	for _, want := range []string{
		"level=ERROR",
		"projection=call_views",
		"last_error=",
		"database is locked",
		"restart budget",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("failure alert missing %q; log line: %s", want, got)
		}
	}
}
