package web

import (
	"strings"
	"testing"
)

// The ds-brand bridge gate (plan M1/F3): display.Heatmap is the one library
// component off the tc-* token chain — it fills cells via
// rgba(var(--ds-brand-rgb), α) and rings peak cells via var(--ds-brand),
// both defaulting to violet-600. static/app.css re-bridges the pair onto the
// signal palette AFTER the imports; this gate pins the BUILT stylesheet (the
// embedded copy) so a regen that drops or reorders the bridge fails the build
// instead of leaking violet back into the dashboard silently.
//
// The triplets must stay comma-separated (legacy rgba contract) and stay in
// :root (teal-700) / .dark (teal-400) blocks.
func TestCSSGateBridgesHeatmapBrandToTeal(t *testing.T) {
	css, err := staticFS.ReadFile("static/styles.css")
	if err != nil {
		t.Fatalf("styles.css missing from embed — regenerate it (tailwindcss -i internal/web/static/app.css "+
			"-o internal/web/static/styles.css --minify): %v", err)
	}

	built := string(css)

	for _, want := range []string{
		"--ds-brand-rgb:15, 118, 110", // teal-700 (light, = tc-primary)
		"--ds-brand-rgb:45, 212, 191", // teal-400 (dark, = bridged blue-400)
		"--color-tc-primary:#0f766e",  // the token the bridge and tc-* chain hang on
	} {
		if !strings.Contains(built, want) {
			t.Errorf("built styles.css lost the ds-brand bridge: %q not found", want)
		}
	}

	for _, bad := range []string{
		"--ds-brand-rgb:124, 58, 237", // violet-600 default from the library custom.css
		"--ds-brand-rgb:139, 92, 246", // violet-500 dark default from the library custom.css
	} {
		if strings.Contains(built, bad) {
			t.Errorf("built styles.css carries the violet default (%q) — heatmap leaked off the signal palette", bad)
		}
	}
}
