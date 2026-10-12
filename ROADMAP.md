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
MOS. The remaining raw ideas:

- Substring phone-number search (FTS5 tokenizes `+1728…` as ONE token; a
  trigram/prefix index over E.164 would lift the documented limitation)
- Dashboard enrichment: date-range picker, duration histogram, cost panel
  (both M20.1/M20.2 were descoped in 2026-10-11 without an owner decision —
  see open questions)
- Export streaming beyond the `maxExportRows` cap (keyset pagination loop)

### 2. Autonomy and operations (plan phase 3 — hardening)

Shipped: in-process poller, freshness banner, sd_notify, NixOS module,
nightly backup staging, purge timers. Raw ideas:

- htpasswd reload on SIGHUP/mtime (restart required today)
- Per-IP rate limiting (the shared bucket is global)
- Poller ops endpoint: per-source rejection counters (log lines only today)
- Structured logging (`log/slog`) for poller/serve passes
- Production `-max-age` window in the sibling NixOS module (owner picks the
  value; also `-recording-timeout` exposure there)
- v0.1.0 first release tag once the owner ratifies the release gate

### 3. Ecosystem integration (plan phase 4)

The index is a consumer; the ecosystem reads from it.

- Webphone read-only deep link INTO `/calls/{kind}/{id}` — the lens side is
  done; the link itself lands in the sibling repo (never a data dependency)

### 4. Trust and safety (plan phase 4)

The index concentrates call metadata; it must age gracefully and respect PII.

- PII policy on stored rows: `E164`/`ParticipantHash` exist and are tested
  but nothing enforces them on storage (the hash gate is export-boundary only)
- Direction auto-detection for inbound AI calls (DID interception path);
  today direction is caller-declared because Telnyx conversation metadata
  exposes none
- Production re-ingest for ContentHash v2 (the production DB must be
  re-ingested once; needs the prod deploy path — owner)

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
- **Dashboard day buckets: UTC or Europe/Berlin?** "Calls today" and the
  30-day chart are UTC (documented); for a German PBX a 23:30 CET call counts
  toward the next day's bucket. Asked three times (2026-10-11), still open —
  changes `DashboardStatsQuery` window math and its tests.
- **Descoped dashboard micro-features:** the plan's date-range picker
  (M20.1) and duration histogram (M20.2) were shipped neither nor struck —
  decide deliver-vs-strike.
- **Release gate for v0.1.0:** step-level green gate evidence exists; is that
  sufficient, or must the single clean full pass + the sibling `-max-age`
  decision land first?
- **Live SSE / auto-refresh** for the dashboard (FEATURES PLANNED row) —
  needs the same owner decision as the static-banner replacement.
