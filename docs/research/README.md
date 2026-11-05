# docs/research/ — dependency and stack deep-dive reports

Point-in-time HTML audits of the libraries and surfaces pbx-lens builds on.
Method for each: capability ground truth read from the module cache at the
exact pin (not drifting skill docs), usage census from this repo's code,
upstream state from `gh` — every claim cited `file:line`. Scores are
fractions of verified capabilities, not vibes; each report carries its own
denominator and a how-to-verify appendix.

| Report                                                                                                                           | Subject                                                                                                                 | Score             | Session verdict                                                                                                                                                                                                                             |
| -------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [2026-10-12_http-stack-adoption-audit.html](2026-10-12_http-stack-adoption-audit.html)                                           | the HTTP stack trio (`httputil`, `server_timing`, `go-etag/server`) + appkit/appkit family errorpages-security-realtime | adoption audit    | adopted the trio (middleware chain: docs/http-stack.md); rejected appkit errorpages/security/realtime + templ-components errorpage + export rate-limiting + SSE-hub (docs/drafts/ + AGENTS.md record the dispositions)                      |
| [2026-10-24_go-cqrs-lite-catalog-testing-modules-deep-dive.html](2026-10-24_go-cqrs-lite-catalog-testing-modules-deep-dive.html) | go-cqrs-lite catalog + testing modules                                                                                  | 58/100 (moderate) | verdict recorded in the report; fed the 2026-10-25 wave/systemscenario/catalog plan (docs/planning/)                                                                                                                                        |
| [2026-10-24_templ-components-deep-dive.html](2026-10-24_templ-components-deep-dive.html)                                         | templ-components (UI trio member) at the v1.23.0 pin, 38 capabilities                                                   | 91/100            | actions executed 2026-10-24/25: utils/format adopted, pin bumped to v1.24.0 (ARIA fixes), ops zero-time guard — see docs/templ-components-adoption.md for the per-component truth                                                           |
| [2026-10-25_go-appkit-templ-deep-dive.html](2026-10-25_go-appkit-templ-deep-dive.html)                                           | go-appkit service shell (73/100) + a-h/templ codegen (86/100) — testkit adoption, unwired seams, the F1 test-drain fix  | 73 + 86 /100      | testkit.NewService adopted (3 sites); errorpages/security/realtime dispositions re-affirmed; fold-in watch: appkit v0.10.0 default Server-Timing + new testkit helpers (row added 2026-11-04 — the report had shipped without an index row) |

**Prior-report reconciliation (2026-11-05, plan T23).** All four earlier
deep-dives were re-read against the tree; their headline claims stand and the
fleet report needed no inherited-claim caveat retired, because the 2026-11-04
re-score had already replaced every second-hand note with a verified one:

| Report                       | Headline found in the file       | Reconciled against the tree                                                                                                                              |
| ---------------------------- | -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-10-12 http-stack        | adoption audit, no numeric score | carries its own ERRATA block (T03) correcting finding #2 at its own pin                                                                                  |
| 2026-10-24 cqrs-lite catalog | 58/100 (moderate)                | "Execution errata, 2026-10-25" records F1 closed and the eight-pin wave; the residual items live in that wave's plan, not as open findings               |
| 2026-10-24 templ-components  | 91/100 at the v1.23.0 pin        | "Delta appendix" records the 2026-10-25 re-run; the per-component truth moved to docs/templ-components-adoption.md, and the fleet score is 95 at v1.26.0 |
| 2026-10-25 appkit + templ    | 73/100 + 86/100                  | appkit still 73 (v0.10 watch open); the 86 is the templ CODEGEN score, not the components score — the index keeps them distinct                          |

**Fleet-wide audit (2026-11-04, re-scored 2026-11-05):** [docs/review/libs/index.html](../review/libs/index.html) scores ALL 19 first-party libraries against pbx-lens — 7 used (cqrs-lite core 92 pooled ≈81, templ-components 95, go-etag 95, go-error-family 82, httputil 80, appkit 73, server_timing 67; mean 83, every finding closed) and 12 not imported, each with a documented reason or a named flip-trigger (REJECT ×9, DEFER ×2, PATTERN-ONLY ×1, ADOPT-PARTIAL ×1). Scores in the index are POST-closeout; the audit-day numbers (mean 71) stay in the derivation paragraph so the delta is visible. Per-lib detail lives beside that index as 19 markdown reports. Refresh a lib's report on its pin bump or when its flip-trigger fires.

Refresh policy: a deep-dive is a snapshot. Re-run the census + score after
major actions on its subject (the templ-components report's delta appendix
records its 2026-10-25 re-run) or on the next pin bump; append deltas
non-destructively to the existing report.

Report screenshot policy (decided 2026-11-05): the fleet index's proof render
(`docs/review/libs/index-screenshot.png`) is COMMITTED uncompressed — ≈1.6 MB
at 1280×11026. It is the only visual evidence for a report that nobody can
eyeball from a diff, this host ships no PNG optimizer (`oxipng`, `pngquant`,
`optipng`, `zopflipng` all absent), and a lossy quantizer would degrade the
exact text the screenshot exists to prove. Rules: render AFTER the final
content edit (never before — a pre-edit render is a lie), one file per report
replaced in place, and revisit the policy if a render passes ~5 MB or a
lossless optimizer lands in the devShell.

CHANGELOG policy for docs-only sessions: no CHANGELOG entries — CHANGELOG
records product/user-facing changes; docs-only sessions ship their reports
here and in docs/status/ + docs/planning/, and this index plus the status
README are their discovery surface.
