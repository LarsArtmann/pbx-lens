# Owner Decision — internal/web package policy (2026-11-20, FM#11/T11)

**Decision: RATIFIED — execute the structural assessment's P1 moves.**
`internal/web` is four packages wearing one name (7.4k prod LOC, 69% of
recent Go churn, the identity bounded context with no code boundary).
The split proceeds in the assessment's order, as MOVE-ONLY refactors:

1. **Identity first** — `userauth.go` (+ its test files) →
   `internal/identity`. Smallest, independently landable, and it gives
   the documented identity bounded context the code boundary its
   durability guarantees (backup, purge-untouchability) have been
   resting on convention for.
2. **Read side second** — queries, dashboard, quality, segments,
   anomaly, coverage, freshness (≈2.9k LOC) → `internal/query` with
   exported view models.
3. **Serve composition third** — `cmd/pbx-lens/serve.go` + the ops
   payload move out of the binary into an internal package (the
   documented convention already puts the CLI command layer in
   `internal/cli`); the command stays a thin verb.

What REMAINS in `internal/web` is a coherent serving package: routing,
templ pages, export, recording proxy, ops page, caching.

**Ratification provenance:** executed under the owner's 2026-11-20
"GET SHIT DONE — the WHOLE TODO LIST" order (Pareto plan T11 →
T16/T17/T18); the plan's own gate treated this answer as the
pre-agreed unblock. Reversibility: all three moves are
package-relocation refactors with zero signature changes — any can be
reverted file-wise without touching behavior. The module-boundary HOLD
(2026-10-11, single go.mod) is unaffected: these are package seams,
not modules.

**Policy rule going forward:** a bounded context that docs/DOMAIN_LANGUAGE.md
names gets its own package; a package whose prod LOC passes ~3k or
that owns two doc-named contexts owes a split proposal in its next
review.
