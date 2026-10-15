# Upstream filings (D6, 2026-10-11 — all source-verified, voice-checked, review boxes left UNCHECKED for Lars)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit).
Update these notes when upstream lands.

**State check 2026-10-12 (read-only, gh api): all five filings still OPEN
upstream — no upstream movement to fold in yet.**

- **go-cqrs-lite#62**: sqliteengine binds filter values raw (`filter_clause.go:44`,
  `planned_filter.go:77`) — time.Time miscompares vs RFC3339 JSON (Le/Lt leak the day,
  Ge/Gt drop it, Eq never). In-repo fix stays: bind `rfc3339Nano` strings.
- **go-cqrs-lite#63**: memory engine resolves filter/sort/cursor columns by Go FIELD name
  (`typed_reader_scan.go:271` → `reflect.go:307`) vs sqlite's JSON keys.
- **go-cqrs-lite#64**: cqrs-lint C009 `len(name) > 4` flags the bare `must` idiom;
  E005 doesn't recognize `system.RegisterCommand` (ADR-0123 root API).
- **go-nix-helpers#7**: go-standard's templ formatter inherits treefmt-nix's `pkgs.go`
  wrap (`go-standard.nix:1297`) instead of the module's resolved goPkg (line 665).
- **numtide/treefmt-nix#545**: already filed upstream 2026-10-02 — corroborating comment
  posted with our independent repro (nixpkgs go 1.26.8 vs floor 1.27.1).
- **BuildFlow#45** (filed 2026-10-12): go-mod-update misreports a
  load-starved compile gate (`tool.timeout` kill) as "dependency bump
  introduced compile errors" and silently reverts go.mod/go.sum;
  source-verified chain + draft in `docs/drafts/`.
- **BuildFlow#50** (filed 2026-10-12, plan T15): the general gate-level
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
- **go-appkit#26** (filed 2026-10-12): testkit cleanup runs the full
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
- **BuildFlow#55** (filed 2026-10-15, plan 05-40 M5/F11): `nix-hash-fix`
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
- Re-verified 2026-10-15 (plan M10/F32-F33): #48/#49/#51 all OPEN, zero
  comments; the #48 mechanism is still present at the pin —
  `display/bar_chart_templ.go:140` (outer `flex items-end` + inline height)
  and `:286` (hard-coded middle `flex w-full flex-1 items-end`, no definite
  height) mean percentage bar heights still resolve against auto → 0px
  without the `!items-stretch` workaround.
- Drafts live in `docs/drafts/` (never /tmp).
