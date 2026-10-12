package web

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The palette gate (plan T2): pbx-lens .templ files must never hardcode the
// library's blue palette. The theme bridge in static/app.css re-maps the
// library's blue shades onto the teal signal palette globally, so raw blue
// classes in OUR templates are either redundant (the shades the bridge
// covers) or drift risks (shades it does not), and either way they break the
// one-block re-skin contract in AGENTS.md. Library-internal components may
// emit blue classes; they are out of scope — this gate walks only this
// package's .templ sources.
//
// Gate form decision (subtask 2.1): a Go test, not a CI yaml grep step — it
// runs identically locally and in CI through the existing go test lane, and
// the red/green proof lives in the tests below (synthetic files) instead of
// a one-off manual plant in the working tree.

var rawBlueClass = regexp.MustCompile(`-blue-\d`)

// scanForRawBlue returns the offending line numbers and text of any raw blue
// palette class in a .templ source.
func scanForRawBlue(source string) []string {
	var findings []string

	for i, line := range strings.Split(source, "\n") {
		if loc := rawBlueClass.FindStringIndex(line); loc != nil {
			findings = append(findings, fmt.Sprintf("line %d: %s", i+1, strings.TrimSpace(line)))
		}
	}

	return findings
}

// TestPaletteGateScansSyntheticFiles proves the scanner red/green: a planted
// violation is caught, a clean template passes.
func TestPaletteGateScansSyntheticFiles(t *testing.T) {
	dirty := `<div class="hover:bg-blue-700 dark:bg-blue-600">x</div>`
	if got := scanForRawBlue(dirty); len(got) != 1 {
		t.Fatalf("scanner missed planted violation, got %v", got)
	}

	clean := `<div class="bg-tc-primary dark:bg-tc-primary-hover text-white">x</div>`
	if got := scanForRawBlue(clean); len(got) != 0 {
		t.Fatalf("scanner flagged clean template: %v", got)
	}
}

// TestNoRawPaletteClasses is the actual gate over the real tree.
func TestNoRawPaletteClasses(t *testing.T) {
	var violations []string

	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, f := range scanForRawBlue(string(src)) {
			violations = append(violations, path+": "+f)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	for _, v := range violations {
		t.Errorf("raw blue palette class (tokens only — see static/app.css theme bridge): %s", v)
	}
}
