package web

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The corpus freshness gate (plan P7; 05-49 §f5): the Tailwind class corpus
// is a COMMITTED, generated file (scripts/gen-tc-classes.js scans the pinned
// templ-components' rendered markup). Two incidents make it worth guarding:
// the v1.21.0-era script silently scanned nothing and shipped an empty
// corpus, and the 2026-10-24 v1.24.0 bump shipped a class DROP because the
// corpus was never regenerated. A stale corpus does not fail loudly — the
// CSS gate just stops knowing the classes the UI actually renders — so this
// test fails instead, at the place the staleness becomes checkable: the
// corpus's OWN pin stamp versus go.mod.
var corpusStamp = regexp.MustCompile(`(?m)^# templ-components@(v\S+)\s*$`)

// goModPin matches the module's own requirement line — leading tab allowed
// (require blocks indent), anchored so the /htmx, /icons and /utils
// sub-requirements cannot satisfy it.
var goModPin = regexp.MustCompile(`(?m)^[\t ]*github\.com/larsartmann/templ-components (v\S+)$`)

func TestTailwindCorpusMatchesPinnedVersion(t *testing.T) {
	raw, err := staticFS.ReadFile("static/templ-components-classes.txt")
	if err != nil {
		t.Fatalf("templ-components-classes.txt missing from embed: %v", err)
	}

	stamp := corpusStamp.FindStringSubmatch(string(raw))
	if stamp == nil {
		t.Fatalf(
			"corpus lost its pin stamp (first line must be `# templ-components@vX.Y.Z`) — " +
				"regenerate with `node scripts/gen-tc-classes.js`",
		)
	}

	goMod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("reading go.mod from the test package: %v", err)
	}

	pin := goModPin.FindStringSubmatch(string(goMod))
	if pin == nil {
		t.Fatalf("templ-components requirement line not found in go.mod — is the module renamed?")
	}

	if stamp[1] != pin[1] {
		t.Errorf(
			"Tailwind class corpus is stale: stamped %s but go.mod pins %s — regenerate with "+
				"`node scripts/gen-tc-classes.js` (docs/templ-components-adoption.md carries the bump watchlist)",
			stamp[1], pin[1],
		)
	}
}

// The empty-corpus incident: a script that scans nothing ships a file that
// still parses, still stamps, and still passes a pin check. The line floor
// and the census (one exact line per representative utility class, each
// chosen because a rendered component depends on it) refuse that silently
// small corpus.
// corpusLineFloor refuses the empty-corpus incident (v1.21.0 era: a scan
// of nothing shipped a stamp-valid, near-empty file).
const corpusLineFloor = 3000

func TestTailwindCorpusCarriesTheRenderedClasses(t *testing.T) {
	raw, err := staticFS.ReadFile("static/templ-components-classes.txt")
	if err != nil {
		t.Fatalf("templ-components-classes.txt missing from embed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < corpusLineFloor {
		t.Errorf(
			"corpus holds only %d lines (floor %d) — the generator scanned almost nothing; "+
				"do not lower the floor, fix the scan",
			len(lines), corpusLineFloor,
		)
	}

	// One exact line each: variants in both directions (responsive, dark+hover),
	// a plain color utility the token chain re-skins, and a dark-mode stroke
	// (the heatmap's only off-chain consumer is bridged in app.css, not here).
	census := []string{"sm:hidden", "dark:hover:bg-gray-800", "bg-blue-600", "dark:stroke-gray-800"}

	corpus := make(map[string]bool, len(lines))
	for _, line := range lines {
		corpus[line] = true
	}

	for _, want := range census {
		if !corpus[want] {
			t.Errorf(
				"corpus census: %q missing — the class drop class of failure; regenerate with "+
					"`node scripts/gen-tc-classes.js` and re-run the adoption map audit",
				want,
			)
		}
	}
}
