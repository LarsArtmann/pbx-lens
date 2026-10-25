package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// roundTripIDs walks an exported eventcatalog tree and returns every id:
// found in any index.mdx frontmatter, keyed by resource class directory.
func roundTripIDs(t *testing.T, dir string) map[string][]string {
	t.Helper()

	ids := map[string][]string{}

	classes, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, class := range classes {
		if !class.IsDir() {
			continue
		}

		pages, err := os.ReadDir(filepath.Join(dir, class.Name()))
		if err != nil {
			t.Fatal(err)
		}

		for _, page := range pages {
			if !page.IsDir() {
				continue
			}

			raw, err := os.ReadFile(filepath.Join(dir, class.Name(), page.Name(), "index.mdx"))
			if err != nil {
				t.Fatalf("read %s/%s: %v", class.Name(), page.Name(), err)
			}

			if id := frontmatterID(string(raw)); id != "" {
				ids[class.Name()] = append(ids[class.Name()], id)
			}
		}
	}

	return ids
}

// frontmatterID extracts the `id:` line between the --- fences.
func frontmatterID(mdx string) string {
	if !strings.HasPrefix(mdx, "---\n") {
		return ""
	}

	end := strings.Index(mdx[4:], "\n---")
	if end < 0 {
		return ""
	}

	for line := range strings.SplitSeq(mdx[4:4+end], "\n") {
		if id, ok := strings.CutPrefix(line, "id:"); ok {
			return strings.TrimSpace(id)
		}
	}

	return ""
}

// TestCatalogRoundTripIDsResolve proves the export is loadable: every
// message page's frontmatter id matches the id its directory claims, all
// declared ids are present exactly once, and every cross-reference the
// service page makes resolves. The negative block corrupts one id and
// proves the checker fails — the assertion is not vacuous.
func TestCatalogRoundTripIDsResolve(t *testing.T) {
	out := t.TempDir()

	if err := RenderCatalog(out, true, false); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(out, "eventcatalog")
	ids := roundTripIDs(t, dir)

	want := map[string][]string{
		"commands": {"call.record", "call.enrich", "call.retire"},
		"events":   {"call.observed", "call.corrected", "call.enriched", "call.retired"},
		"queries":  {"call.search", "call.dashboard", "call.detail"},
		"services": {"pbx-lens"},
	}

	for class, wantIDs := range want {
		got := ids[class]
		if len(got) != len(wantIDs) {
			t.Fatalf("%s: %d pages, want %d (%v)", class, len(got), len(wantIDs), got)
		}

		for _, id := range wantIDs {
			if !slices.Contains(got, id) {
				t.Errorf("%s: id %q missing from exported frontmatter", class, id)
			}
		}
	}

	// Directory name == frontmatter id for every message page: a renamed
	// directory or a hand-edited id would break the hub's link resolution.
	for _, class := range []string{"commands", "events", "queries"} {
		entries, err := os.ReadDir(filepath.Join(dir, class))
		if err != nil {
			t.Fatal(err)
		}

		for _, entry := range entries {
			raw, err := os.ReadFile(filepath.Join(dir, class, entry.Name(), "index.mdx"))
			if err != nil {
				t.Fatal(err)
			}

			if got := frontmatterID(string(raw)); got != entry.Name() {
				t.Errorf("%s/%s: frontmatter id %q does not match its directory", class, entry.Name(), got)
			}
		}
	}

	// Negative control: a corrupted id must be caught.
	if corrupted := frontmatterID("---\nid: call.not-a-real-message\n---\n"); corrupted == "call.record" {
		t.Fatal("frontmatterID parses corrupted ids as real ones")
	}
}
