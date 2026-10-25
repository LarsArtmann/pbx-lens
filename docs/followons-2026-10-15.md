# Follow-on sketches (plan 05-40 M18, F56–F58) — 2026-10-15

> Audited 2026-10-25: F56 SHIPPED (see its entry), F57 stands as the opinion
> record (sibling-module fold is a TODO row), F58 was already closed. The file
> stays as the reasoning record; nothing here is open work except F57's
> sibling wording.

## F56 — dashboard cost panel — **EXECUTED 2026-10-25**

~~The dashboard renders per-window volume/duration/AI share. A cost panel
needs per-call cost, and today NO source carries an ingested cost value.~~
**Superseded:** the Detail-Records enrichment pass writes
`Enrichment.CostMicros`/`CostCurrency` onto already-observed calls
(`internal/cli/enrich.go`), so the data dependency this sketch waited for
is satisfied by the poller, not by the MOS window.

Shipped: the dashboard's "Spend" card (`internal/web/dashboard.go`
`DashboardStats.Cost*`, `internal/web/dashboard.templ` `spendCard`) renders
window spend, cost per billed minute, the week-over-week delta, and a daily
spend sparkline. Every figure carries its own denominator ("N of M calls
priced"), a window with no priced calls renders the honest empty state, and
a mixed-currency window refuses to name one currency. Exports already carry
the cost column (`CostMicros`/`CostCurrency` in `internal/web/export.go`).
The sketch's "fourth StatCard" became a card in the aside (the headline grid
is a fixed four-up).

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

- fixture tests") SHIPPED 2026-10-12 (R22: `-ai-did` +
  `telnyx.DIDDirectionResolver` on ingest-telnyx/serve; CHANGELOG
  Unreleased) and the sibling module option landed (`aiDids`, e7c0f8b).
  The rejected alternative (post-ingest correction wave) stays rejected —
  doctrine: corrections are for SOURCE-owned changes. Nothing open remains
  in that note; it is now a design record, not a plan.
