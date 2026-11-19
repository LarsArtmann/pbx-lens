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
| [2026-11-19_templ-components-deep-dive.html](2026-11-19_templ-components-deep-dive.html)                                         | templ-components v1.27.0 pin, 51-capability re-audit                                                                  | 89/100 (very high) | v1.27.1 (2026-11-19) is the retirement wave — the bump + adoption-map watchlist steps retire the #48/#49/#51/#55/#63 workarounds in one act (templ stays v0.3.1100); supersedes the fleet refresh's "v1.27.0 == latest" |
| [2026-11-19_cqrs-htmx-deep-dive.html](2026-11-19_cqrs-htmx-deep-dive.html)                                                       | cqrs-htmx usermgmt v4.20.0 + loginpage v4.18.0 identity surface (PATH B)                                              | 83/100 (high)     | loginpage NoRegistration/AccentColor/ThemeFromRequest unset; logical identity export unused; ADR-0062 spotless; v5 exposure = /v4→/v5 renumber only; supersedes fleet 78 (OIDC lane + OAuth limits + audit snapshot + logout landed since) |
| [2026-11-20_go-cqrs-lite-deep-dive.html](2026-11-20_go-cqrs-lite-deep-dive.html)                                                 | go-cqrs-lite runtime core (14 direct modules at pin)                                                                  | 91/100 (very high) | 5 partials (ScanPage keyset, stricter-than-HealthCheck readiness, Reset prod run owner-gated, schema v1, record envelope); 4 pins behind (system/metaengine/sqliteengine/catalog — wave verified zero used-surface change); supersedes the runtime-core half of docs/review/libs/go-cqrs-lite.md |
| [2026-11-20_go-appkit-deep-dive.html](2026-11-20_go-appkit-deep-dive.html)                                                       | go-appkit v0.12.0 + systemd v0.1.0 (serve shell)                                                                      | 100/100 (very high) | fully adopted; fleet-73 was the hooks-as-applicable convention (hooks now N/A-by-reason); ReadyCheck wired; systemd v0.2.0 rides the next deps train; supersedes docs/review/libs/go-appkit.md + the appkit half of 2026-10-25 |

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

**Fleet-wide audit (2026-11-04; re-scored 2026-11-05; refreshed 2026-11-11 — cqrs-htmx flip — and 2026-11-12 — full re-verification pass):** [docs/review/libs/index.html](../review/libs/index.html) scores ALL 22 first-party libraries against pbx-lens — 8 used (cqrs-lite core 92 pooled ≈81, templ-components 95, go-etag 95, go-error-family 82, httputil 80, cqrs-htmx 78 adopted surface, appkit 73, server_timing 67; mean 83, every finding closed) and 14 not imported directly, each with a documented reason or a named flip-trigger (REJECT ×11, DEFER ×2 — go-retry/go-atomic-write; both ride the graph indirectly with zero pbx-lens imports, PATTERN-ONLY ×1: go-output). Scores in the index are POST-closeout; the audit-day numbers (mean 71) stay in the derivation paragraph so the delta is visible. Per-lib detail lives beside that index as 22 markdown reports. Refresh a lib's report on its pin bump or when its flip-trigger fires.

Refresh policy: a deep-dive is a snapshot. Re-run the census + score after
major actions on its subject (the templ-components report's delta appendix
records its 2026-10-25 re-run) or on the next pin bump; append deltas
non-destructively to the existing report. The mechanical half arrived
2026-11-12: every console block in the 22 markdown reports re-derives via
`scripts/replay-report-verify.sh` (the verify-all `--replay` lane), so a
report's commands stay executable claims, not prose.

Report screenshot policy (decided 2026-11-05): the fleet index's proof render
(`docs/review/libs/index-screenshot.png`) is COMMITTED uncompressed — ≈2.3 MB
at 1440×full-page (re-shot 2026-11-12; the 2026-11-05 shot was ≈1.6 MB at 1280×11026). It is the only visual evidence for a report that nobody can
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
