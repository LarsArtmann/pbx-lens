# Dedup to zero, pass 2 — the suppressed bucket was hiding the real clones

Session 2026-10-24 ~12:40–13:30. Owner instruction: "deduplicate as much as
possible to zero," continuing `2026-10-24_12-33_dedup-to-zero-baseline-gate.md`
(that run's §g questions are answered in §e below).

## §a What the first pass missed

The 12:33 run audited the 24 groups the default report SHOWS and concluded
"zero harmful clones." That conclusion was wrong for an interesting reason:
**art-dupl's default report hides two buckets** (13 "non-actionable" + 21
"filtered suppressed" = 34 of 58 groups), and `check` gates only on what it
shows. Auditing the hidden bucket (`--show-suppressed`, the deferred Q3)
found **four real production clone groups** in `internal/web`, all invisible
to the just-shipped gate:

- ~~`serveStyles`/`serveHtmx`/`serveFavicon` — 3×~10-line static-file handlers~~ done — extracted (§b)
- ~~`list` vs `stalenessFragment` — 2×19-line bodies differing only in render~~ done — extracted (§b)
- ~~`dashboard` vs `dashboardFragment` — 2×15-line bodies differing only in render~~ done — extracted (§b)
- ~~`exportJSONL` vs `exportCSV` — 2×7-line format preamble (the one visible~~ done — extracted (§b)
  ~~group, previously accepted as "extraction loses")~~

Everything else hidden was test scaffolding (spot-checked: poller_test's
28-line pair is fixture+bootstrap setup, the accepted self-contained-test
family).

## §b Extractions (behavior-identical; error strings, timing labels, header order preserved)

| Was                                    | Now                                                                                                                                  |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| ~~static handler trio~~ done (docs-health pass 2026-10-26) | ~~`staticAsset` struct + one `serve` method + `staticRoutes` map (declarative table, one `//nolint:gochecknoglobals`, house precedent)~~ |
| ~~`list`/`stalenessFragment` bodies~~ done (docs-health pass 2026-10-26) | ~~`queryListStaleness` returning typed `listStaleness` (struct — the two trailing bools cannot transpose at call sites)~~ |
| ~~`dashboard`/`dashboardFragment` bodies~~ done (docs-health pass 2026-10-26) | ~~`queryDashboardStats` (stats/fragURL/ok); handlers only render~~ |
| ~~export format preamble~~ done (docs-health pass 2026-10-26) | ~~`beginExportResponse(w, r, contentType, disposition)` absorbs both headers after the PII gate~~ |

Full suite green after each step; `internal/web` re-tested per extraction.

## §c Gate recalibration: 23 pinned groups → ZERO

The 23 baseline entries were all 1-line signature/type-expression matches
(`(v string)`, `templ.Component`, two-field structs) — shape coincidence
between functions with different bodies, not duplication. `--min-lines 2`
removes that noise class from the detection universe, so the regenerated
baseline records **zero accepted groups**: the gate contract is now "no
clones accepted at all; any new multi-line clone group goes red."

- ~~Green proof: clean tree → `check` exit 0.~~ done (docs-health pass 2026-10-26)
- ~~Red proof: seeded 12-statement cross-package clone (wire+backup probes)~~ done (docs-health pass 2026-10-26)
  ~~→ exit 2 with both files named; probes trashed; exit 0 again.~~
- ~~Plain vs type-aware modes agree at this calibration (0 shown both).~~ done (docs-health pass 2026-10-26)

## §d Verification receipts

- ~~`go build ./...` + full `go test ./...` green (twice, incl. after lint fixes).~~ done (docs-health pass 2026-10-26)
- ~~`buildflow --build-mode dev`: **exit 0, 60 success, 0 failed** (final run~~ done (docs-health pass 2026-10-26)
  ~~13:14). Casualties en route, both fixed: go-structure-linter AGENTS.md size~~
  ~~(236→219 lines; the 12:33 session had left it over the 220 limit), and a~~
  ~~cold-cache nix-build step deadline (re-ran green; `go clean -cache` earlier~~
  ~~freed a 100%-full /mnt/buildcache — 143G — which also explains the run's~~
  ~~across-the-board timing regressions).~~
- ~~`golangci-lint run ./internal/web/...` (repo config): my findings cleared~~ done — the three pre-existing findings were cleared or dispositioned by the 2026-10-25 lint pass and the 2026-10-26 review
  ~~(gochecknoglobals ×3, wsl ×2). Three PRE-EXISTING findings remain in files~~
  ~~this session never touched — `dashboard.go:376` cyclop (view, 14>12),~~
  ~~`queries.go:97` gocognit (ListCalls, 37>25), `segments.go:60`~~
  ~~nonamedreturns — not relitigated here.~~
- ~~ci.yml: valid YAML; dprint-format green via buildflow.~~ done (docs-health pass 2026-10-26)

## §e The 12:33 report's three owner questions — resolved autonomously

- ~~**Q1 keep the gate red-on-new-clone?** Yes — and strengthened: zero-group~~ done — answered by this session (zero-group)
  ~~contract instead of 23-pinned-groups.~~
- ~~**Q2 pin art-dupl into the flake devShell?** Not adopted now: the binary~~ done — answered; the 2026-10-26 stale-binary incident enforced pin parity via check-dedup-gate.sh instead
  ~~resolves system-wide from the nix profile (`0.9.0-93432fd`), CI pins~~
  ~~published `@v0.9.0`, and a devShell input would churn flake.lock for a~~
  ~~convenience already present. Revisit only if profile availability flakes.~~
- ~~**Q3 audit the suppressed bucket?** Done — it was the actual bug (§a).~~ done — answered in §a

## §f Follow-ups (none blocking)

- ~~Watch the first CI run of the recalibrated gate (pin-parity receipt:~~ done — CI green from its first run through 2026-10-26
  ~~local `0.9.0-93432fd` vs published `@v0.9.0`; if hashes mismatch,~~
  ~~regenerate per docs/art-dupl-baseline.md).~~
- ~~Optional: after any art-dupl pin bump, run the `--show-suppressed` audit~~ done — documented as a duty in docs/art-dupl-baseline.md; ran with the v1.24.0 sweep
  ~~(now documented as a duty in docs/art-dupl-baseline.md).~~
- ~~The three pre-existing golangci findings (§d) predate this work; route via~~ routed — dispositioned by the 2026-10-25 lint pass (13-44) and the 2026-10-26 review (advisory-only residue documented)
  ~~TODO_LIST if wanted.~~
