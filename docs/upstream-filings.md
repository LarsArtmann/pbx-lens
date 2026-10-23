# Upstream filings (D6, 2026-10-11 — all source-verified, voice-checked, review boxes left UNCHECKED for Lars)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit).
Update these notes when upstream lands.

**State check 2026-10-12 (read-only, gh api): all five filings still OPEN
upstream — no upstream movement to fold in yet.**

**State check 2026-10-17 (gh, docs-health pass): CLOSED upstream —
BuildFlow#45 (2026-10-12), #50 (2026-10-12), #55 (2026-10-16), #56
(2026-10-16), go-cqrs-lite#62/#63/#64, go-appkit#26. Still OPEN:
go-etag#2, go-nix-helpers#7, numtide/treefmt-nix#545,
templ-components#48/#49/#50/#51/#55, dependabot-core#16538.**

**State check 2026-10-23 (gh, docs-health audit): unchanged — same CLOSED
set; go-appkit#26 confirmed CLOSED (fold-in still pending), and go-etag#2,
go-nix-helpers#7, numtide/treefmt-nix#545 plus templ-components
#48/#49/#50/#51/#55 re-confirmed OPEN. Two new BuildFlow candidates were
diagnosed this day and are staged, not filed (report 2026-10-23_16-21 §b3).**

- **go-cqrs-lite#62** (CLOSED upstream 2026-10-17; fold-in = the 2027-01 watch): sqliteengine binds filter values raw (`filter_clause.go:44`,
  `planned_filter.go:77`) — time.Time miscompares vs RFC3339 JSON (Le/Lt leak the day,
  Ge/Gt drop it, Eq never). In-repo fix stays: bind `rfc3339Nano` strings.
- **go-cqrs-lite#63** (CLOSED upstream 2026-10-17): memory engine resolves filter/sort/cursor columns by Go FIELD name
  (`typed_reader_scan.go:271` → `reflect.go:307`) vs sqlite's JSON keys.
- **go-cqrs-lite#64** (CLOSED upstream 2026-10-17): cqrs-lint C009 `len(name) > 4` flags the bare `must` idiom;
  E005 doesn't recognize `system.RegisterCommand` (ADR-0123 root API).
- **go-nix-helpers#7**: go-standard's templ formatter inherits treefmt-nix's `pkgs.go`
  wrap (`go-standard.nix:1297`) instead of the module's resolved goPkg (line 665).
- **numtide/treefmt-nix#545**: already filed upstream 2026-10-02 — corroborating comment
  posted with our independent repro (nixpkgs go 1.26.8 vs floor 1.27.1).
- **BuildFlow#45** (filed 2026-10-12; CLOSED upstream 2026-10-12): go-mod-update misreports a
  load-starved compile gate (`tool.timeout` kill) as "dependency bump
  introduced compile errors" and silently reverts go.mod/go.sum;
  source-verified chain + draft in `docs/drafts/`.
- **BuildFlow#50** (filed 2026-10-12, plan T15; CLOSED upstream 2026-10-12): the general gate-level
  behavior under #45 — spawn starvation (`errorfamily.Transient` +
  `tool.timeout`, `cmdrun_streaming.go:86`) still exits error with a
  `step_failure` verdict; the error registry's fix advice ("increase
  the timeout") is wrong for the starvation flavor. Ask: one retry for
  no-output spawn kills + an environment exit class for all-transient
  runs + load average in the summary. Draft:
  `docs/drafts/buildflow-spawn-starvation-gate.md`.
- C009 verification note (2026-10-12): the `len(name) > 4` guard is
  confirmed at source (`cmd/cqrs-lint/pkg/rules/correctness/c009.go:95`,
  same shape for `New` at line 99) and IS covered by go-cqrs-lite#64 — no
  separate filing.
- R12 pass (2026-10-12, owner: annotate-and-keep): all six filed drafts in
  `docs/drafts/` now carry a FILED banner naming their issue; they stay as
  historical write-ups. `go-atomic-write-commit-primitive.md` remains the
  one LIVE draft (unfiled; owner review box unchecked by design).
- **dependabot-core#16538** (filed 2026-10-12, owner-approved R24): gomod
  bump of an unrelated module deleted all four templ-components requires
  (evidence: this repo's PR #1 diff + commit `40e1a49`); mechanism class of
  closed #3617. Draft + repro shape in `docs/drafts/2026-10-12_dependabot-gomod-unrelated-bump-deletes-requires.md`.
- **go-etag#2** (filed 2026-10-12): server middleware — a handler-initiated
  `Flush`/`Hijack` abandons ETag generation silently; `OnBufferOverflow`
  explicitly excludes flushes (v0.6.1 `etag.go:83-88,382-419`), no hook or
  log on the path. Real consumer proof: templ's per-section flush gave
  every pbx-lens HTML page zero ETags. Ask: `OnFlushed` hook or debug log.
  Draft: `docs/drafts/go-etag-server-flush-no-observability.md`.
- **go-appkit#26** (filed 2026-10-12; CLOSED upstream 2026-10-17 — fold-in ready, TODO_LIST row): testkit cleanup runs the full
  production drain, so every `Serve` test pays the 5s default `DrainDelay`
  (v0.8.0 `testkit.go:80,179-181`, `config.go:18`) and the
  `Serve(tb, *Service)` shape gives tests no way to avoid it. Measured:
  5.0s of a 5.0s package. Ask: `ServeConfig` constructor with
  `NoDrainDelay` default or a doc line. Draft:
  `docs/drafts/go-appkit-testkit-drain-delay-cost.md`.
- **templ-components#48** (filed 2026-10-12, T24): vertical BarChart — the
  documented `Height` field ("Essential for vertical charts",
  v1.21.0 `display/bar_chart.go:93-96`) is necessary but NOT sufficient:
  the container's `items-end` (`bar_chart.templ:77`) stops columns
  stretching into the definite height, so bars compute 0px. Live
  before/after: Height-only = 0/0/0px; + `Class: "!items-stretch"` =
  46/108/15px. Local workaround stays until the fix lands. Draft:
  `docs/drafts/templ-components-barchart-vertical-zero-height.md`.
- **templ-components#49** (filed 2026-10-12, T24): clickable table rows
  are focusable + Enter-activatable (`table.templ:281-289`,
  `shared.go:217-224`) but ship no focus-visible styling — keyboard users
  can't see which row is active (WCAG 2.4.7). Local ring in
  `static/app.css` (`tr[role="link"]:focus-visible`). Draft:
  `docs/drafts/templ-components-tablerow-focus-indicator.md`.
- **templ-components#50** (filed 2026-10-12, T24): the consumer skill
  documents `FilterBar`/`SegmentedControl`/`utils/format` as current API;
  all three are absent at the v1.21.0 pin (utils/format exists on master,
  unreleased; FilterBar/SegmentedControl exist only as a recipe + plan).
  Ask: version-stamp the skill / mark unreleased components. Draft:
  `docs/drafts/templ-components-skill-docs-drift-v1.21.0.md`.
- **templ-components#51** (filed 2026-10-12, plan T2): `ScrollbackLine`
  has only Timestamp/Tag/Text/Tone (v1.21.0 `scrollback.templ:25-30`,
  same on master 2026-10-12) — no per-line id/class hook, so anchor links
  and highlight marks on individual lines are impossible; the container's
  `BaseProps` (lines 67-78) doesn't help. This is the gap that forced
  pbx-lens's hand-rolled `transcriptMatchLog` (T21); fold-in retires it
  (owner-gated, TODO_LIST). Draft:
  `docs/drafts/templ-components-scrollback-line-hook.md`.
- **templ-components#55** (filed 2026-10-15, plan M8/F28 + M10): `EnsureID`
  mints a crypto/rand id per render when `BaseProps.ID` is empty
  (`utils/validate.go:32-45`; caller `navigation/nav_templ.go:80`), so every
  page body is unique and ETag/`no-cache` revalidation can never 304 —
  measured on pbx-lens: three consecutive `/` fetches carried three
  different ETags, `If-None-Match` always 200; after pinning the nav id the
  same probe 304s across `/`, `/app.css`, and polled fragments. Local fix:
  `BaseProps.ID: "pbx-lens-nav"` in the shell (`internal/web/calls.templ`),
  pinned by `TestNavIDIsDeterministic`. Mechanism re-verified at the pin
  2026-10-15. Draft:
  `docs/drafts/2026-10-15_templ-components_nav-id-etag-volatility.md`.
- **BuildFlow#55** (filed 2026-10-15, plan 05-40 M5/F11; CLOSED upstream 2026-10-16 — the binary on this machine may still predate the fix, answer-sheet §9): `nix-hash-fix`
  matches the stale hash by VALUE only, first lexical hit wins
  (`modules/nix-checker/stale_hash.go:53-63` at `d114f28ce`), so when the
  hash text exists on a second line (here: flake.nix's cqrs-lint
  `fetchFromGitHub` hash, pre-churned by the 2026-10-12 fixes) the
  corrected hash lands in the WRONG attribute — live corruption a61ba8c
  wrote the pbx-lens vendor hash into the input hash, restore 48a63df;
  the fixer also has no dirty-tree guard, and the auto-commit daemon
  swept the corruption in within minutes. Ask: attribute-aware matching +
  loud refusal on multi-line matches. Draft:
  `docs/drafts/2026-10-15_buildflow_nix-hash-fix_wrong-attribute.md`.
  Until upstream lands: never run the hash fixer on a dirty tree
  (AGENTS.md, vendorHash.nix provenance comment).
  **RECURRENCE 2026-10-15 18:33** (stale binary `518739c`, tree dirty with
  doc edits): `buildflow -s nix-hash-fix --fix` re-committed the exact
  wrong-attribute write (real vendor hash → cqrs-lint `hash`, placeholder →
  cqrs-lint `vendorHash` AND vendorHash.nix, corruption 62bcca6), and the
  step still self-reported ✔ verify + "1 fixed". Restored by hand from the
  probe log's `got:` value; `nix build .` + `nix develop -c true` green.
  Hardened repo rule: until the binary carries the fix, hash repairs go
  through `nix run .#update-vendor-hash` on a clean tree, never the step.
- Re-verified 2026-10-15 (plan M10/F32-F33): #48/#49/#51 all OPEN, zero
  comments; the #48 mechanism is still present at the pin —
  `display/bar_chart_templ.go:140` (outer `flex items-end` + inline height)
  and `:286` (hard-coded middle `flex w-full flex-1 items-end`, no definite
  height) mean percentage bar heights still resolve against auto → 0px
  without the `!items-stretch` workaround.
- **BuildFlow#56** (filed 2026-10-15, 05-40 M6/F17 aftermath): the gate's
  managed `go`/`govalid` tools execute binaries compiled into BuildFlow's
  own nix closure (go 1.27.0), so a module toolchain-floor bump (go-appkit
  v0.9.0 → go.mod 1.27.2) failed those two steps while every repo-side
  environment was already on 1.27.2 — doctor even prints "go1.27.2 is
  supported" in the same run. Individual verification is the fallback;
  root fix = rebuild BuildFlow from a clean checkout. Recorded in
  docs/buildflow-dev-gate.md.
- Drafts live in `docs/drafts/` (never /tmp).

## Fold-in watch (quarterly — plan 05-40 M14/F44)

Next scheduled check: **2027-01** (rides the Telnyx live-API re-verify
window; same session, one pass over the list below). For each filing:
re-open the issue, check state + maintainer comments, and if it landed
upstream, execute the local fold-in and delete the workaround + TODO row.

| Filing                          | Fold-in action when it lands                                                                                                                                                                                          |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| go-cqrs-lite#62/#63             | CLOSED 2026-10-17 — re-evaluate the in-repo RFC3339 binding / field-name workarounds for deletion                                                                                                                     |
| go-cqrs-lite#64                 | CLOSED 2026-10-17 — drop any cqrs-lint C009/E005 suppressions this repo carries (docs/cqrs-lint-policy.md)                                                                                                            |
| go-nix-helpers#7                | bump the flake input; drop the local templ-formatter override if any                                                                                                                                                  |
| numtide/treefmt-nix#545         | confirm the fix, remove the corroborating-comment note                                                                                                                                                                |
| BuildFlow#45/#50/#55            | #45/#50/#55 all CLOSED (10-12/10-16) — bump BuildFlow; re-run the gate on a load-starved machine (#50); retry a dirty-tree hash fix in a scratch repo (#55, only after the local binary carries the fix)              |
| dependabot-core#16538           | owner decides gomod pinning; re-check the dependabot PR behavior                                                                                                                                                      |
| go-etag#2                       | bump; consider wiring OnFlushed when it exists                                                                                                                                                                        |
| go-appkit#26                    | CLOSED 2026-10-17 — bump; adopt the no-drain-delay testkit constructor for tests                                                                                                                                      |
| templ-components#48/#49/#51/#55 | bump → run the adoption-map watchlist (docs/templ-components-adoption.md); retire `!items-stretch`, the focus ring CSS, hand-rolled match-log classes, and the pinned nav id + TestNavIDIsDeterministic as each lands |
