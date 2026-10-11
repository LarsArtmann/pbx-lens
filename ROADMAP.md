<!-- ROADMAP.md — long-term direction and raw ideas. No bounded tasks here
     (those live in TODO_LIST.md); no status indicators; no completed items. -->

# Roadmap

> Long-term direction and raw ideas for pbx-lens. Items here are NOT
> actionable tasks. When an idea is refined into bounded work, it moves to
> TODO_LIST.md. Phase numbering follows the SUPERB plan
> (`pbx-artmann/docs/planning/2026-10-09_20-28_SUPERB-pbx-lens-call-index-and-woolly-golive.md`).

## Themes

### 1. The complete consumer surface (plan phase 2)

One index is only worth building if every question about a call is answerable
from it: who, when, how long, AI or human, what was said, what it cost, how it
sounded.

Raw ideas:

- Per-call detail page with recording playback beside a role-split transcript
- Full-text search across every transcript ever ingested (FTS5)
- Quality dashboards: call volume by day, AI share, MOS trends mined from CDR
  quality fields
- Insight summaries per AI call (Telnyx already generates
  `conversation_insights`; surface them instead of recomputing)

### 2. Autonomy and operations (plan phase 3)

Today every ingest is a CLI invocation. The index should keep itself fresh and
survive on its own.

Raw ideas:

- Poller timers plus a freshness self-check so a stale index is visible in the
  UI, not silently wrong
- Thin NixOS module in the sibling repo that deploys pbx-lens on the PBX host
- Index DB folded into the existing nightly backup-staging
- Cost column fed by a lag-tolerant Detail Records poller

### 3. Ecosystem integration (plan phase 4)

The index is a consumer; the ecosystem reads from it.

Raw ideas:

- webphone gets a read-only deep link into a call view — never a data
  dependency
- UI auth reusing the operator htpasswd pattern

### 4. Trust and safety (plan phase 4)

The index concentrates call metadata; it must age gracefully and respect PII.

Raw ideas:

- Retention and PII policy enforcement as a purge job; `E164` and
  `ParticipantHash` (already defined in `internal/calls/domain.go`) become
  load-bearing when this lands
- Direction auto-detection for inbound AI calls (DID interception path);
  today direction is caller-declared because Telnyx conversation metadata
  exposes none

## Non-goals

- **Writing to any source.** Read-only over FreeSWITCH, Telnyx, and the
  receiver webhook log, forever. The index DB is disposable and re-derivable.
- **Owning ingestion inside the PBX runtime** (data-mesh placement decision,
  2026-10-09): sources stay domain-owned; the sibling repo ships only a thin
  deploy module.
- **Becoming a webphone component.** webphone links in; it never depends on
  this repo's data.
- **Dependency sprawl.** stdlib + go-cqrs-lite + go-error-family only; no
  testify, no cobra, no ORM.

## Open questions (owner decisions, not tasks)

- ~~**Public or private on GitHub?**~~ **Resolved 2026-10-11:** public, at
  `github.com/LarsArtmann/pbx-lens`; `main` pushed, repo metadata set. The
  zero-secrets scan (gitleaks, full history) and synthetic-only fixtures made
  public visibility safe.
- **Keep the flat `internal/calls` package or split it?** Documented policy is
  flat (one aggregate, one projection); the natural revisit trigger is T9
  landing a second concern. (report 2026-10-10_06-57 §g.3, cqrs-lint E003)
