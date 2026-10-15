# scripts/

Browser-level UI instruments (dev tooling only — puppeteer-core installs
into a scratch prefix under /tmp on first use; nothing enters the Go module
graph, the flake, or package.json). Shared bootstrap lives in
`lib/ui-probe.js` (chromium discovery, scratch install, probe-binary build,
corpus seeding, serve lifecycle).

| Script                  | What it proves / captures                                                                                                                                         | When to run                                               |
| ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| `screenshot-probe.js`   | 24 PNGs (light/dark × mobile/desktop × calls/search/dashboard/2 details/?q= match-log detail) into `--out` for eyeball passes                                     | After any visual change; before tagging                   |
| `verify-ui-contract.js` | Browser-level contract check: 3 copy buttons + label swap (T20); search hit `?q=` → 302 → `#turn-N` → amber mark → real fragment scroll (T21). Exit 1 on any FAIL | After changes to detail/search/copy paths; before tagging |
| `verify-style-contract.js` | Browser-level style contracts: `--ds-brand-rgb` resolves teal (light `15, 118, 110`, dark `45, 212, 191`, never violet) + a rendered heatmap cell computes to `rgb(15, 118, 110)`; the All-option of both filter selects is `value=""` + selected, and `?direction=inbound` selects Inbound. Exit 1 on any FAIL | After any CSS/theme or filter-form change; before tagging |
| `verify-staleness-live.js` | Live staleness watch (M3/F11): a 48 h-old index renders the banner, forcing the htmx poll keeps it while stale, and a fresh ingest + poll makes it disappear on screen WITHOUT a reload. Exit 1 on any FAIL | After changes to the staleness banner / PolledRegion / freshness logic |
| `gen-tc-classes.js`     | Regenerates the library class inventory (`templ-components-classes.txt`) used by the palette gate and the version-bump watchlist                                  | On templ-components version bumps                         |

Both browser scripts are self-contained: they build the binary, seed the
corpus (smoke + one agent transcript so the `?q=` match log has turns),
serve it on `--port`, and clean up. `--port` is OPTIONAL since M4 (F13):
when omitted, every probe asks the OS for a free port (`lib.findFreePort`),
which removes the stale-port stumble entirely; the pinned defaults (18099
screenshots, 18107 verify, 18110 style, 18116 staleness) remain for
reproducibility. A stale serve on a chosen port still fails loudly instead
of capturing against a dead binary.

Codegen bootstrap (M4/F14): every probe runs `lib.ensureCodegen()` first —
it regenerates `*_templ.go` when a `.templ` is newer (go.mod-pinned
`go tool templ`, NEVER the devShell templ — drift exists) and rebuilds
`styles.css` when missing (flake-pinned tailwindcss via `nix develop -c`),
so a clean checkout can't probe against empty or stale assets. `go:embed`
is compile-time, so this runs BEFORE the binary build.

Smoke seeding flags (M3): `smoke -age 48h` backdates the seeded call
(staleness surfaces) and `smoke -with-recording ref` sets the recording
reference (audio card). Both are probe fixtures; defaults unchanged.

Golden review helper (M4/F12):

```console
bash scripts/golden-pretty-diff.sh   # readable word-diff of the golden HTML
```

What only these scripts prove (the Go tests pin the server-side rules —
`internal/web/search_jump_test.go`, `copy_button_test.go`, and the built-CSS
gate `css_gate_test.go`): click-time label
swap, real fragment scroll into the viewport, computed pixel truth (the
browser resolves the CSS vars the gate can only read as text), and
dark/mobile rendering. `node --check scripts/*.js` parses without a browser.

CI lane decision (plan M1/F4, 2026-10-15): the browser probes stay
LOCAL/dev-tooling only — the CI runner has no chromium and the flake
deliberately does not add one; the Go-side twin of the style contract
(`TestCSSGateBridgesHeatmapBrandToTeal` over the embedded styles.css) DOES
run in the existing `go test` lane, so the silent-rot class fails CI without
a browser. Run all three probes by hand before tagging:

```console
node scripts/verify-ui-contract.js --port 18107
node scripts/verify-style-contract.js --port 18110
```

Editor note (plan T5.3, adjudicated 2026-10-12): vtsls may report
"Declaration or statement expected" parse errors on these files — they are
false positives (repo has no jsconfig/tsconfig, and the shebang +
CommonJS + async-IIFE combo trips its project inference). The arbiter is
`node --check`, which passes on every script here; do not "fix" the JS to
silence the editor.
