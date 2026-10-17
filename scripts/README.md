# scripts/

Browser-level UI instruments (dev tooling only — puppeteer-core installs
into a scratch prefix under /tmp on first use; nothing enters the Go module
graph, the flake, or package.json). Shared bootstrap lives in
`lib/ui-probe.js` (chromium discovery, scratch install, probe-binary build,
corpus seeding, serve lifecycle).

| Script                     | What it proves / captures                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | When to run                                                                  |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| `screenshot-probe.js`      | 24 PNGs (light/dark × mobile/desktop × calls/search/dashboard/2 details/?q= match-log detail) into `--out` for eyeball passes; `--only <name[,name…]>` captures a subset (surface labels: calls, search, dashboard, detail-a/b/q, notfound, detail-rec, print, empty-*, long-excerpt); `--save-baseline <dir>` copies the corpus out; `--diff <dir>` compares alert-only (never fails) with a 128-px time-text budget + 0.05% tolerance — the practiced noise floor (2026-10-17: the only same-session churn is the footer "Updated" stamp and RelativeTime chips, ≤81 px). Aged baselines flag age-text drift — that is the refresh-baseline signal, not a regression; save fresh baselines to `~/lens-baselines/<gen>/` (machine-local; the in-repo strategy stays the ROADMAP owner question). `LENS_DIFF_BASELINE=<dir> bash scripts/verify-all.sh` wires the diff into the lane | After any visual change; before tagging; `--diff` whenever a baseline exists |
| `verify-ui-contract.js`    | Browser-level contract check: 3 copy buttons + label swap (T20); search hit `?q=` → 302 → `#turn-N` → amber mark → real fragment scroll (T21). Exit 1 on any FAIL                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | After changes to detail/search/copy paths; before tagging                    |
| `verify-style-contract.js` | Browser-level style contracts: `--ds-brand-rgb` resolves teal (light `15, 118, 110`, dark `45, 212, 191`, never violet) + a rendered heatmap cell computes to `rgb(15, 118, 110)`; the All-option of both filter selects is `value=""` + selected, and `?direction=inbound` selects Inbound. Exit 1 on any FAIL                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | After any CSS/theme or filter-form change; before tagging                    |
| `verify-staleness-live.js` | Live staleness watch (M3/F11): a 48 h-old index renders the banner, forcing the htmx poll keeps it while stale, and a fresh ingest + poll makes it disappear on screen WITHOUT a reload. Exit 1 on any FAIL                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | After changes to the staleness banner / PolledRegion / freshness logic       |
| `csp-sweep.js`             | CSP report-only dry run (P1-T7): boots the chain with `CSP_NONCE=1 CSP_REPORT_ONLY=1`, drives six surfaces + theme/copy clicks, collects `securitypolicyviolation` events. Exit 1 on any violation; zero = the nonce policy is enforce-clean (verdict history in docs/http-stack.md)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 | After any new inline script, templ-components bump, or CSP policy edit       |
| `gen-tc-classes.js`        | Regenerates the library class inventory (`templ-components-classes.txt`) used by the palette gate and the version-bump watchlist                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | On templ-components version bumps                                            |
| `verify-all.sh`            | The one-command lane: runs the three contract probes + the screenshot sweep sequentially, each on its own free port; first red wins. Opens with the statix no-op tripwire — statix exits 0 on a config parse error and lints nothing (the five-day silent outage, docs/gotchas.md), so the lane greps stderr for "config error" and fails the lane on it (lint verdicts stay BuildFlow's; skips when statix is off PATH). Extra args pass through to screenshot-probe (`--only`, `--themes`, `--out`); `--baseline <dir>` is the LENS_DIFF_BASELINE hook as a lane flag (wins over the env var)  | Before tagging; after any UI or cache-affecting change                       |
| `cut-release.sh`           | The release runbook: clean-tree quiesce (BuildFlow#55 guard) → CHANGELOG sweep → nix build → full dev gate → browser probes → remote-CI-green check (the 2026-10-15 tag-red lesson) → annotated tag + push → `go list -m` module-proxy check. Refuses dirty trees, load over policy, and unpushed HEADs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Cutting a release (replaces the hand-run receipt checklist)                  |
| `check-agents-budget.sh`   | AGENTS.md ≤219 `wc -l` check (the lint tool counts a phantom +1; rationale in docs/buildflow-dev-gate.md). Exit 1 over budget. `.githooks/pre-commit` calls it — activate with `git config core.hooksPath .githooks` (config stays owner-set)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | Standalone, in the pre-commit hook, or wired into CI on demand               |

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

Codegen bootstrap (M4/F14, hash-primary since 05-40 M12/F36): every probe
runs `lib.ensureCodegen()` first — it hashes the `.templ` inputs + app.css
into a ledger file (`.codegen-inputs.sha256`, gitignored) and regenerates
whenever the hash moves (branch switches with lying mtimes) OR the per-file
mtime fallback fires (each `*_templ.go` vs its own `.templ`; a missing
generated file counts as stale). The old global-newest mtime comparison
never converged — templ skips rewriting unchanged outputs, preserving their
older mtimes — so every probe paid a no-op `templ generate`. The ledger and
regeneration run before the binary build (`go:embed` is compile-time).

Probe-merge decision (05-40 M12/F35, 2026-10-15): the four probes stay
SEPARATE scripts behind the single `verify-all.sh` entry point. The
contract probes assert different failure domains (DOM copy behavior,
computed style truth, timed staleness transitions) with different browser
lifecycles; merging them would couple unrelated failures and bloat each
run with the others' setup. Shared plumbing (chromium discovery, scratch
puppeteer, build/seed/serve) already lives in `lib/ui-probe.js`, so the
duplication cost merging would have removed is already paid once.

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
