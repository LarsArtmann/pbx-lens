# Follow-on sketches (plan 05-40 M18, F56–F58) — 2026-10-15

## F56 — dashboard cost panel (sketch; data does not support it yet)

The dashboard renders per-window volume/duration/AI share. A cost panel
needs per-call cost, and today NO source carries an ingested cost value:
Telnyx conversation objects expose no price (cost lives in Detail
Records — see the MOS poller design, same 2027-01 window), and FreeSWITCH
CDRs carry no billing columns in the vanilla templates this repo parses.

**Sketch for when Detail Records land:** a fourth StatCard ("Est. cost,
window") fed by `sum(cost)` over windowed calls, nullable exactly like
MOS (honest placeholder until the first score arrives); CSV/JSONL exports
gain the cost column; the PII read-boundary treatment does not apply
(cost is not personal data). Effort after the window: ~half a day. No
code now — the data dependency is the gate, and the MOS design doc
already carries the wire shape (ScoreRecord grows a Cost field).

## F57 — backup rotation (opinion: host-tool boundary)

The repo's backup contract ends at "verified nightly snapshot lands in a
host directory" (VACUUM INTO → fsync → verify → rename; systemd timer).
Rotation/off-host shipping deliberately stays with the host backup tool
(README: "retention and off-host shipping stay with the host backup
tool").

**Opinion (owner asked for one):** keep that boundary. Reasons: (1) the
host tool already owns encryption + dedup + off-site for EVERYTHING ELSE
on the machine — a second rotation implementation inside pbx-lens would
be worse at all three; (2) the snapshot is a single immutable file, the
exact shape host tools ingest best; (3) retention INSIDE the index is
already solved (`internal/retention` retire + purge timers). The only
pbx-lens-side obligation: keep snapshots crash-consistent and verified —
done, tested (`restore_drill_test.go`). If the host tool ever needs
help: expose `-out` (exists) and a post-snapshot hook (NOT planned — no
consumer).

## F58 — direction auto-detection: design note closed

The scope slice in `docs/did-direction-design.md` ("mapper option + flag
+ fixture tests") SHIPPED 2026-10-12 (R22: `-ai-did` +
`telnyx.DIDDirectionResolver` on ingest-telnyx/serve; CHANGELOG
Unreleased) and the sibling module option landed (`aiDids`, e7c0f8b).
The rejected alternative (post-ingest correction wave) stays rejected —
doctrine: corrections are for SOURCE-owned changes. Nothing open remains
in that note; it is now a design record, not a plan.
