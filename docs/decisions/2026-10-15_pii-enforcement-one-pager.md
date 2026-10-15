# PII enforcement — the one-pager (owner decision pack, F13)

- **Created:** 2026-10-15 (plan 05-40 M4). Decision gate = F14 (owner).
  This page exists so the decision is a 2-minute read, not a code dive.
- **Status:** AWAITING OWNER BOUNDARY ANSWER (g-3). Nothing below is
  implemented; the primitives exist and are tested but unenforced.

## What exists today (verified)

- `calls.E164` (normalized +49… numbers) and `calls.ParticipantHash`
  (salted hash) — both tested primitives, NEITHER enforced on stored
  fields (AGENTS.md Domain).
- Stored numbers are RAW as observed (real CDRs carry unknown/empty
  parties — normalization would corrupt the source truth).
- The EXPORT gate already chose READ (owner doctrine R21): exports answer
  `X-PII-Mode`, FEATURES.md carries the field-coverage matrix, raw-at-rest
  is explicit owner doctrine.

## The one decision: where does enforcement live?

### Option A — at INGEST (storage normalized)

Numbers stored as E164 (+ participant hashes) from day one; raw text kept
in a sidecar only if the owner wants an audit trail.

- **For:** strongest posture (a DB leak exposes hashes, not numbers);
  search/dedupe gets exact normalized keys; exports leak-proof by
  construction.
- **Against:** breaks the read-only/source-truth doctrine at the storage
  layer (the index rewrites what the source said); ContentHash v2 (the
  stored hash changes → one-time re-ingest wave); raw CDR oddities
  (short codes, SIP URIs, extension numbers) don't all normalize —
  either the E164 type grows lossy branches or ingest rejects rows.

### Option B — at READ (storage stays raw, every surface renders/enforces)

Storage keeps raw as observed; E164/ParticipantHash compute at the
render/export/search boundaries. The existing export gate is the
precedent and the pattern.

- **For:** zero migration (no ContentHash v2 wave); source-truth doctrine
  intact (AGENTS: read-only over every source); the boundary is a
  presentation/authorization concern — where PII policy actually lives;
  enforcement can tighten without re-ingest.
- **Against:** a DB leak exposes raw numbers (mitigated by the deploy
  posture: the DB lives behind the host, backups are the owner's
  existing encrypted host-tool chain); every new surface must remember
  the boundary (mitigated by making `calls.E164` the ONLY rendering path
  — a lint/golden net like the palette gate can pin it).

### Recommendation (assistant's view, not a decision)

**Option B.** The index's whole design premise is "disposable, always
re-derivable, never mutates source truth" — ingest-side normalization is
the one change class that breaks re-derivability. The leak vector the
strong posture answers is better answered by the host's disk/backup
encryption, which is already the owner's chain. Enforcement-at-read with
a pinned rendering path gives 90% of A's value for 10% of its cost.

## What implementation looks like (either answer, bounded)

1. `calls.E164` becomes the single number-rendering path for the UI
   (list/detail/copy buttons) — raw stays available under an explicit
   "as observed" disclosure where the owner wants it.
2. `ParticipantHash` wires into search's digits shadow for hash-mode
   lookups (only if the owner picks hash-at-rest ambitions later).
3. A palette-gate-style test: no `.templ` renders `FromNumber`/`ToNumber`
   except through the E164 helper (fails the gate on new surfaces).
4. Docs: FEATURES matrix + SECURITY.md posture line.

Effort: L (a focused day). Blocked on: F14 answer (and for A only: the
ContentHash v2 gate).
