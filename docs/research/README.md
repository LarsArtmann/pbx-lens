# docs/research/ — dependency and stack deep-dive reports

Point-in-time HTML audits of the libraries and surfaces pbx-lens builds on.
Method for each: capability ground truth read from the module cache at the
exact pin (not drifting skill docs), usage census from this repo's code,
upstream state from `gh` — every claim cited `file:line`. Scores are
fractions of verified capabilities, not vibes; each report carries its own
denominator and a how-to-verify appendix.

| Report                                                                                                                           | Subject                                                                                                                 | Score             | Session verdict                                                                                                                                                                                                        |
| -------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [2026-10-12_http-stack-adoption-audit.html](2026-10-12_http-stack-adoption-audit.html)                                           | the HTTP stack trio (`httputil`, `server_timing`, `go-etag/server`) + appkit/appkit family errorpages-security-realtime | adoption audit    | adopted the trio (middleware chain: docs/http-stack.md); rejected appkit errorpages/security/realtime + templ-components errorpage + export rate-limiting + SSE-hub (docs/drafts/ + AGENTS.md record the dispositions) |
| [2026-10-24_go-cqrs-lite-catalog-testing-modules-deep-dive.html](2026-10-24_go-cqrs-lite-catalog-testing-modules-deep-dive.html) | go-cqrs-lite catalog + testing modules                                                                                  | 58/100 (moderate) | verdict recorded in the report; fed the 2026-10-25 wave/systemscenario/catalog plan (docs/planning/)                                                                                                                   |
| [2026-10-24_templ-components-deep-dive.html](2026-10-24_templ-components-deep-dive.html)                                         | templ-components (UI trio member) at the v1.23.0 pin, 38 capabilities                                                   | 91/100            | actions executed 2026-10-24/25: utils/format adopted, pin bumped to v1.24.0 (ARIA fixes), ops zero-time guard — see docs/templ-components-adoption.md for the per-component truth                                      |

Refresh policy: a deep-dive is a snapshot. Re-run the census + score after
major actions on its subject (the templ-components report's delta appendix
records its 2026-10-25 re-run) or on the next pin bump; append deltas
non-destructively to the existing report.

CHANGELOG policy for docs-only sessions: no CHANGELOG entries — CHANGELOG
records product/user-facing changes; docs-only sessions ship their reports
here and in docs/status/ + docs/planning/, and this index plus the status
README are their discovery surface.
