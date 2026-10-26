<!-- docs/decisions/2026-10-26_pii-and-export-compression-owner-message.md
     Plan T8 (2026-10-26): the P17 pair as ONE owner message — the PII
     boundary (the 2026-10-15 one-pager's answer, still open) and the export
     streaming compression question (15-39 §f27). Choice IDs; answer by
     letter. Nothing here blocks a session; both rows stay parked until the
     answer sheet carries the letters. -->

# Owner message — PII boundary + export compression (2-minute read, 2026-10-26)

Two parked decisions, both yours, neither blocking session work. Answer by
letter (e.g. "PII: A, compression: B") into the answer sheet
(docs/decisions/2026-10-17_owner-answer-sheet.md).

## 1. PII enforcement boundary — pick A or B

Full analysis: `2026-10-15_pii-enforcement-one-pager.md` (unchanged since;
the primitives `calls.E164`/`ParticipantHash` remain built, tested, and
unenforced). The short form:

- **A — at INGEST** (store normalized + hashed; raw only in a sidecar):
  strongest leak posture, but the index stops storing what the source
  said, and the ContentHash v2 re-ingest wave becomes the mandatory
  migration moment.
- **B — at READ** (store raw forever; normalize/hash at the edge per
  consumer): source truth preserved (the R21 export doctrine already
  chose read for exports); every new read surface must remember to gate —
  the discipline cost recurs per surface instead of once.

The 2026-10-15 recommendation stands: **B (read boundary)** — it matches
what the repo already does for exports and keeps the index an honest
projection.

## 2. Export/recording stream compression — pick A, B, or C

The export streams and the recording proxy are UNBOUNDED and skip the ETag
cache (docs/http-stack.md). On a LAN this is a non-question; over WAN or
mobile it is the difference between a 40 MB export and an ~8 MB one.

- **A — none (today's shape):** zero moving parts; big exports stay big.
- **B — q-negotiated gzip on the streaming routes** (`Accept-Encoding`
  honored, `Content-Encoding: gzip` on export + recording): one
  middleware-shaped change, CPU per stream is trivial at personal-PBX
  volume, browser/curl decompress transparently.
- **C — defer until a real consumer complains:** cheapest now; the cost is
  a future session re-deriving this exact analysis.

Recommendation: **B** — small, reversible, and the recording WAVs (the
biggest payloads) compress poorly but the CDR/JSON exports (the common
case) compress ~5×.

## If you answer

- PII answer → closes the TODO_LIST Blocked row (enforcement + policy flag
  - docs become a normal session task).
- Compression answer → B becomes a small plan row; A or C closes it.
