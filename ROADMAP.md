<!-- ROADMAP.md — long-term direction and raw ideas. No bounded tasks here
     (those live in TODO_LIST.md); no status indicators; no completed items. -->

# Roadmap

> Long-term direction and raw ideas for pbx-lens. Items here are NOT
> actionable tasks. When an idea is refined into bounded work, it moves to
> TODO_LIST.md. Phase numbering follows the SUPERB plan
> (`pbx-artmann/docs/planning/2026-10-09_20-28_SUPERB-pbx-lens-call-index-and-woolly-golive.md`).
>
> Rebuilt 2026-10-12 (docs-health AUDIT): every raw idea from the original
> four themes that has working code (detail page, FTS5 search, dashboards,
> insight summaries, poller + freshness, NixOS module, backup staging, cost
> column, UI auth, retention purge, `?pii=hash` export, webphone deep-link
> target) graduated to FEATURES.md and was removed here.

## Themes

### 1. The complete consumer surface (plan phase 2 — closing gaps)

One index is only worth building if every question about a call is answerable
from it. Shipped: list, detail, search, dashboard, export, cost, insights,
MOS. Everything this theme's raw ideas named landed 2026-10-12 (partial
number search, date-range picker, duration histogram, streaming export) —
the remaining follow-on idea:

- Cost panel on the dashboard (the data is in `CostMicros`; the histogram
  card pattern is there to copy)

### 2. Autonomy and operations (plan phase 3 — hardening)

Shipped: in-process poller, freshness banner, sd_notify, NixOS module,
nightly backup staging, purge timers — and 2026-10-12: htpasswd SIGHUP
reload, per-IP failure throttling, the `/api/ops` counters endpoint,
structured slog records, the restore-drill test. Remaining raw ideas:

- Backup rotation/shipping policy (host backup tool owns it today; a lens
  opinion would be a new decision)
- v0.1.0 release + the production `-max-age` / `-recording-timeout` module
  values are owner-blocked rows in TODO_LIST (not repeated here)

HTTP-stack follow-ons (2026-10-12 adoption wrap; each parked on its own
revisit trigger — not commitments):

- gzip CPU cost check at production index size (revisit when real export
  volumes are known; 4–6x ratios proven only on the synthetic probe)
- errorpages / security middleware revisit when templ-components publishes
  its errorpage or go-appkit v0.9 ships the `security/`/`errorpages/`
  packages (both absent in the pinned v0.8.0 / v1.21.0)
- data-fingerprint fast-304 (mtime+count keyed) if the index grows ~100x
- static assets with hashed filenames + long max-age (measure the actual
  cache-miss rate first; YAGNI-lean while htmx.min.js is the only asset)
- `go-etag/entitytag` as a direct dep only if handler-set tags ever land
- consolidate `timing.go`/`caching.go` when a fourth middleware arrives
- etag/gzip ratios surfaced as `appkit_*` metrics (needs appkit metric
  plumbing; `/api/ops` hook wiring is the TODO_LIST near-term)
- CSP `report-uri` collector endpoint (2026-10-15): the CSP_REPORT_ONLY dry
  run currently surfaces violations console-side only; a collector endpoint
  would aggregate browser reports during the enforcement evaluation
- Screenshot corpus compression (2026-10-15): 41 PNGs per sweep → webp or
  on-demand capture, for eyeball-pass ergonomics

### 3. Ecosystem integration (plan phase 4)

The index is a consumer; the ecosystem reads from it.

- Webphone read-only deep link INTO `/calls/{kind}/{id}` — the lens side is
  done; the link itself lands in the sibling repo (never a data dependency)
- Templ codegen staleness semantics (2026-10-15): the probe inputHash work
  found templ skips rewriting unchanged outputs (preserving stale mtimes) —
  a possible upstream finding; evaluate via verify-before-filing before any
  filing

### 4. Trust and safety (plan phase 4)

The index concentrates call metadata; it must age gracefully and respect PII.
Raw-at-rest is now explicit doctrine (owner decision 2026-10-12, R21): the
index stores raw numbers because real CDRs carry garbled parties, and the
EXPORT gate (`?pii=hash` + the FEATURES coverage matrix) is the PII boundary.
Remaining raw idea:

- Direction auto-detection for inbound AI calls (DID interception path);
  today direction is caller-declared because Telnyx conversation metadata
  exposes none. Design note: `docs/did-direction-design.md` (2026-10-12,
  R22) — the mapping rule and its ingestion plug-point are sketched there
  as a TODO_LIST row

UI long-tail raw ideas (2026-10-15 UI-contract plan M11/M17/M19 harvest;
each parked on its own trigger — the rows in TODO_LIST carry the actionable
versions):

- SegmentBar / FilterBar adoption re-evals on the next templ-components
  bump (absent at v1.21.0; the bump watchlist in
  docs/templ-components-adoption.md is the trigger)
- Mobile card-list for the calls table vs the current horizontal scroll
  (owner taste call gates the sketch)
- Dashboard SectionHeading trial and hit-card hover consistency — one
  decide-once pass, not open-ended polish
- Date/number vocabulary (US vs EU) — decided in the owner decision pack
  (D1); the change itself is one central helper, all surfaces
- i18n note (2026-10-15): the UI copy is English-only; the D1 EU-format
  decision may reopen the copy-language question for the operator audience

## Non-goals

- **Writing to any source.** Read-only over FreeSWITCH, Telnyx, and the
  receiver webhook log, forever. The index DB is disposable and re-derivable.
- **Owning ingestion inside the PBX runtime** (data-mesh placement decision,
  2026-10-09): sources stay domain-owned; the sibling repo ships only a thin
  deploy module.
- **Becoming a webphone component.** webphone links in; it never depends on
  this repo's data.
- **Dependency sprawl.** stdlib + go-cqrs-lite + go-error-family + the UI
  trio (go-appkit, templ-components, a-h/templ) + `x/crypto`/`x/time` where a
  module-graph rider made them free; no testify, no cobra, no ORM. Every
  addition is a deliberate AGENTS.md policy edit (the 2026-10-11 frontend
  plan superseded the original stdlib-only guardrail for UI work).

## Open questions (owner decisions, not tasks)

- ~~**Public or private on GitHub?**~~ **Resolved 2026-10-11:** public, at
  `github.com/LarsArtmann/pbx-lens`; `main` pushed, repo metadata set. The
  zero-secrets scan (gitleaks, full history) and synthetic-only fixtures made
  public visibility safe.
- **Keep the flat `internal/calls` package or split it?** Documented policy is
  flat (one aggregate, one projection); the natural revisit trigger is T9
  landing a second concern. (report 2026-10-10_06-57 §g.3, cqrs-lint E003)
- ~~**Dashboard day buckets: UTC or Europe/Berlin?**~~ **Resolved 2026-10-12
  (owner, R2): Europe/Berlin.** `DashboardStatsQuery`, `todayHref`, and bare
  date filters all bucket in the operator's zone; DST-stable via zone logic;
  the zone is injectable and UTC coverage is tested.
- ~~**Descoped dashboard micro-features:**~~ **Resolved 2026-10-12 (owner,
  R8): both DELIVERED** — date-range picker (M20.1) and duration histogram
  (M20.2) shipped with tests.
- ~~**Release gate for v0.1.0:**~~ **Resolved 2026-10-12 (owner, R4): the
  release goes once the tree gates green at load < 40** — the single clean
  full pass IS the gate; the sibling `-max-age` defaults to forever (owner,
  R5), which unblocks the module option too.
- ~~**Live SSE / auto-refresh**~~ **Resolved 2026-10-12 (owner, R9): htmx
  polling on dashboard + staleness banner**, self-hosted runtime; SSE
  upgrade path stays open if poll volume ever matters.
- **RetireCallCmd decision** (m27.9): the `call.retired` tombstone exists;
  an owner-facing `RetireCallCmd` (single-call retire via CLI/API) has no
  operator demand yet — recorded here so it does not fossilize in a plan.
- **Status-report format policy** (m27.9): reports are hand-written .md
  today; HTML default came from tooling, not from the owner. Standing
  default: .md + mermaid unless the owner says otherwise.
- **Machine-hygiene norm** (m27.9): the full gate only runs at 15-min load
  < 40; below that, casualties verify individually
  (docs/buildflow-dev-gate.md). Recorded as the standing norm so future
  sessions do not re-derive it.
- **Multi-agent coordination for this repo** (2026-10-12): two concurrent
  agent sessions collided twice in one day (shared-tree gate runs burned on
  foreign edits; one commit shipped another session's unstaged deletion and
  went CI-red). Candidate conventions: a session-claim line in AGENTS.md, or
  branch-per-agent with explicit merges. Until the owner decides, the
  standing norm is concurrency pre-flight (check `git status` for foreign
  entries + fresh mtimes before any gate/commit) and ship-state verification
  (test what HEAD contains, e.g. `git stash -u` the foreign delta or probe a
  clean worktree of the commit). (docs/status/2026-10-12_13-49 §e/§g.3)
- **Multi-match transcript navigation** (2026-10-12, from the ?q= jump
  follow-ups): the deep link lands on the FIRST matching turn; when several
  turns match, prev/next match buttons (or a match counter) inside the
  transcript card would let an operator walk the evidence. Needs the
  ScrollbackLine hook (templ-components#51) or the hand-rolled log growing
  nav affordances; goldens pin today's shape first.
- **Scroll-margin on fragment targets** (2026-10-12): `#turn-N` lands the
  matched line at the viewport's very top edge; `scroll-margin-top` (a few
  rem, sticky-header aware) would give the jump breathing room. Pure CSS,
  one line — candidate for the next UI pass, eyeball via
  scripts/screenshot-probe.js.
- **Search "filtered for" chip** (2026-10-12): the list page URL carries
  filters (direction/AI/dates) but nothing on the page names the active
  filter set once you scroll or share a link; a dismissible chip row
  ("filtered: inbound · AI · last 7 days — clear") would make the state
  visible and one-click clearable. Pairs with the existing EmptyState
  "Clear filters" CTA.
- **Trailing-slash canonical form** (2026-10-15, 05-40 M11): `/dashboard/`
  currently 301s to `/dashboard` (`redirectTrailingSlash`, friendlier to
  stale bookmarks, one registration per page) vs exact-pattern 404 (simpler
  route table, honest "path does not exist"). Two sessions fought the test
  assertion to a draw; it is deliberately UNPINNED until one owner taste
  call, then the test gets re-pinned (or the redirect deleted) to match.
  (docs/status/2026-10-15_16-32_superb-0540-m11-m19-execution-multi-session.md §g.1)
- **Pixel-diff baseline strategy** (2026-10-15): commit a light-theme
  baseline corpus (~1-2 MB PNGs — cross-session/cross-machine alerts at
  repo-weight cost) or keep baselines machine-local in /tmp (repo lean;
  baseline dies with the machine). Tooling itself is live and verified;
  only the storage trade is open. (same report §g.3)
