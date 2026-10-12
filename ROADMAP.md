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

### 3. Ecosystem integration (plan phase 4)

The index is a consumer; the ecosystem reads from it.

- Webphone read-only deep link INTO `/calls/{kind}/{id}` — the lens side is
  done; the link itself lands in the sibling repo (never a data dependency)

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
