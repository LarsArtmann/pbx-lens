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
| `gen-tc-classes.js`     | Regenerates the library class inventory (`templ-components-classes.txt`) used by the palette gate and the version-bump watchlist                                  | On templ-components version bumps                         |

Both browser scripts are self-contained: they build the binary, seed the
corpus (smoke + one agent transcript so the `?q=` match log has turns),
serve it on `--port`, and clean up. Port defaults: 18099 (screenshots),
18107 (verify). A stale serve on the port fails loudly instead of capturing
against a dead binary.

What only these scripts prove (the Go tests pin the server-side rules —
`internal/web/search_jump_test.go`, `copy_button_test.go`): click-time label
swap, real fragment scroll into the viewport, and pixel truth (dark/mobile
rendering). `node --check scripts/*.js` parses without a browser.

Editor note (plan T5.3, adjudicated 2026-10-12): vtsls may report
"Declaration or statement expected" parse errors on these files — they are
false positives (repo has no jsconfig/tsconfig, and the shebang +
CommonJS + async-IIFE combo trips its project inference). The arbiter is
`node --check`, which passes on every script here; do not "fix" the JS to
silence the editor.
