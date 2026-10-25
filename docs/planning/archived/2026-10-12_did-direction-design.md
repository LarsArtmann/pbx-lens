<!-- docs/did-direction-design.md — R22 design note (2026-10-12).
     Design only: no code here. The actionable slice lives in TODO_LIST.md. -->

# DID-interception direction auto-detect

## The lie today

`Direction` on an inbound AI call is wrong by construction. The sibling
dialplan routes inbound PSTN calls to the AI agent, but Telnyx conversation
metadata exposes no carrier-side direction, so the poller stamps every
polled conversation with the caller-declared `-poll-direction outbound`
(serve default). An inbound customer call to the AI therefore indexes as
`outbound` — the list filter `?direction=inbound` silently misses it, and
the dashboard's AI-share-by-direction split lies.

## The true signal: the DID path

What actually distinguishes the call is WHICH number was dialed. The
sibling's dialplan (nix-international-telephony) sends:

- **Human extensions** → FreeSWITCH endpoints directly (CDR source:
  `context` says so — `cdr.direction` already derives from context).
- **AI DIDs** (the numbers webphone/agent answer) → the AI runtime, whose
  Telnyx conversation carries `to_number = <AI DID>`.

So the deployable mapping rule is: **a Telnyx conversation whose To number
is a known AI DID is inbound, regardless of what the caller declared.** The
AI-DID set already exists on the CDR side as `-ai-dest` (`ParseAIDestinations`,
the same agent-extension concept at the PSTN layer).

## Where it plugs in

Two candidate plug-points, one recommended:

1. **Mapper-side (recommended):** `internal/ingest/telnyx/map.go` grows a
   `DirectionResolver` option: `func(from, to string) calls.Direction`.
   The CLI builds it from a new `-ai-did` flag (repeatable, or a file —
   same shape as `-ai-dest`). The resolver runs BEFORE the observation is
   built, so ContentHash, idempotent re-ingest, and the read model all see
   the corrected direction with zero migration. A serve flag
   (`-ai-did`, mirroring `-poll-direction`) feeds the poller path.
2. **Post-ingest correction:** a `call.corrected` wave that rewrites
   direction on already-indexed streams. Rejected: corrections are for
   SOURCE-owned snapshot changes; direction here would be a lens-side
   opinion injected into a source-shaped record — a doctrine breach.

## Consequences

- Deterministic: same conversation → same direction on every re-poll (the
  DID set is configuration, not state).
- The DID set is deployment config: the NixOS module gains the option when
  the flag lands (sibling-side follow-up).
- False positives are bounded: a customer calling FROM an AI DID number is
  not a real shape (DIDs do not originate calls).
- The CDR side stays untouched (context-derived direction is already true).

## Scope slice (when picked up)

Mapper option + flag + fixture tests (AI-DID to-number → inbound; non-DID
to-number → declared direction; empty DID set → exactly today's behavior)
is a bounded TODO_LIST row — see "Direction auto-detect" there.
