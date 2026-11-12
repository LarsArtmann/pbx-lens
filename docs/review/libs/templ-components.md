# templ-components — utilization audit (pbx-lens)

**Date:** 2026-11-04 (refreshed **2026-11-11** — the coordinated templ gate bump landed) · **Pin:** `github.com/larsartmann/templ-components v1.27.0` (+ `/htmx`, `/icons`, `/utils` all v1.27.0; go.mod:26-29; == proxy latest) with the paired `a-h/templ v0.3.1100` (go.mod:9; == proxy latest) · **Status:** USED
**Adoption score:** **95/100 (very high)** — 26.5 of 28 applicable capabilities (25 fully ×1.0 + 3 partially ×0.5; 10 of 38 inventory rows N/A with reasons). The #52 fold-in (`RelativeTimeProps.EmptyText` replacing the local "never" guard) landed 2026-11-04, taking the audit-day 93 to 95.
**Prior reports:** [2026-10-24_templ-components-deep-dive.html](../../research/2026-10-24_templ-components-deep-dive.html) (91/100 at v1.23.0; 38 capabilities) and the delta appendix of its own 2026-10-25 re-run. This report supersedes their version-currency rows (pin now v1.26.0, current), their #52-blocked status, and the "six page surfaces" census (seven as of `segment.templ`). Per-component truth map: `docs/templ-components-adoption.md`.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## Verdict

This is the fleet's best-adopted dependency. All 8 applicable packages are imported (display, forms, feedback, layout, navigation, htmx, icons, utils — plus `utils/golden` in tests), all 7 `.templ` pages are built on the library with zero hand-rolled parallel components, the newest page (`segment.templ`) reused the dashboard's shared cards wholesale — the adoption discipline generalized to a new surface untouched. utils/format is fully adopted (26 sites). The three releases since the last audit (v1.25.0/1.25.1/1.26.0) were absorbed almost entirely for free; exactly one local workaround became obsolete upstream and awaits its fold-in.

## Usage census (deltas since the prior reports)

- **Pages:** 7 `.templ` files all on the library — `page.templ` (layout.Base, ToastContainer, GlobalErrorHandling, Footer), `calls.templ` (Table+forms+Alert, self-hosted `HTMXSrc`), `detail.templ` (PolledRegion, Breadcrumbs, CopyButton ×2, DefinitionList, Scrollback, Badge), `dashboard.templ` (PolledRegion 30s, Card ×9, BarChart ×4, Heatmap, Sparkline ×2, StatCard ×4, PieChart Donut, CircularProgress MOS gauge), `ops.templ`, `search.templ` (icons), `segment.templ` (new; reuses `statCards`/`volumeCard`/`qualityCard`).
- **utils/format:** 26 call sites across dashboard/ops/detail/calls (`*_templ.go` cited in the adoption map); `Bytes`/`CompactCount` unused — N/A (no byte/count-heavy surface). All 6 exported funcs verified at pin (`utils@v1.26.0/format/format.go:38-156`).
- **Theme:** token-chain override intact; the one off-chain component (Heatmap) still re-bridged in app.css; contract re-verified at pin (`display/heatmap.templ:91`, `heatmap.go:67` unchanged).

## Delta census (v1.24.0 → v1.26.0)

| Release | Change                                                                                                                                                       | pbx-lens impact                                                                                                             |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------- |
| v1.25.0 | **`display.RelativeTimeProps.EmptyText`** — closes upstream #52 (changelog names pbx-lens as a consumer who can delete its local zero-guard)                 | **Fold-in pending — finding D1**                                                                                            |
| v1.25.1 | Five chart rendering fixes (Sparkline X-overflow, BarChart rune-safe truncation, PieChart ShowLabels, Line/Area out-of-domain ticks, smooth-fill detach)     | Absorbed free; goldens re-pinned                                                                                            |
| v1.26.0 | **Warning hue → amber** (Alert/Badge/StatusDot), `LoadingDialog`, `InputProps.Autocomplete`, top-layer Toasts, native `<progress>`, modal scroll containment | Amber touched Alert (6 sites) + BadgeWarning — handled by the goldens/corpus regeneration commits; Heatmap bridge unchanged |

Since v1.26.0, **v1.27.0 released 2026-11-04 and pbx-lens took the coordinated bump** (templ-components v1.26.0→v1.27.0 + a-h/templ v0.3.1070→v0.3.1100, one act — the two-module risk below executed cleanly; class corpus + goldens regenerated with it, `TestTailwindCorpusMatchesPinnedVersion`/`…RenderedClasses` green at HEAD). v1.27.0 itself is a docs/lint/tooling release for pbx-lens' surface (docs-site chrome bar, dark-mode scan extension to `ring-offset-*`/`shadow-*`, `AccordionProps.Exclusive`, Slider track fill, autofill tint, count-table codegen) — no component pbx-lens renders changed classes or props. Upstream `[Unreleased]` (the likely v1.28.0) is Datastar-heavy: typed Datastar reactive builders, the CSP nonce mode (`layout.DatastarNonceAttrs`), `Stacked` form-field wrapping (#53), `CopyButtonScript` for fragment-mounted buttons (#63), and the responsive Sparkline — watch the responsive-Sparkline fold-in, it retires the fixed-size workaround below. #48/#49/#51 workarounds: **still open upstream, correctly still present locally**.

## Gap analysis

| Finding                                                                                                                                                                                    | Status                                                                                                                                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~D1~~ — **FOLDED 2026-11-04** (ops.templ:204 passes `EmptyText: "never"`, the guard is gone, the contract test remains at ops_page_test.go:173)                                                                 | Resolved — the score's last missed opportunity is banked; no open findings below it                                                        |

| BarChart `!items-stretch` ×3, Sparkline fixed-size contract, Scrollback `?q=` match path                                                                                                   | Partially used — all upstream-blocked (#48/#60/#51 unfixed), not local negligence; the Sparkline item has an upstream fix pending in `[Unreleased]`                                                                                                                                                                                                       |
| RelativeTime                                                                                                                                                                              | Fully leveraged since the D1 fold (prop-driven, no workaround)                                                                                                                                                                                                                                                                                            |
| v1.26.0 new surface (LoadingDialog, HoverCard, Autocomplete)                                                                                                                               | N/A — no blocking-operation/hover/autocomplete use case in a read-only dashboard                                                                                                                                                                                                                                                                         |
| SegmentBar/FilterBar/SegmentedControl, Modal/Tabs/Drawer/CommandPalette/etc., charts/echarts, datastar, errorpage, navigation.Pagination (hand-rolled keyset by design, `calls.templ:156`) | N/A — standing decisions (adoption map rows carry the reasons)                                                                                                                                                                                                                                                                                           |

## Version currency

**Current across the board (2026-11-11):** templ-components v1.27.0 (2026-11-04) == latest release, paired with a-h/templ v0.3.1100 == latest — the coordinated gate bump the audit flagged as risk #1 is EXECUTED (one act, corpus + goldens regenerated, corpus-freshness tests green at HEAD). The v1.27.0 delta touched no component pbx-lens renders (docs-site chrome, dark-mode lint scan, Accordion/Slider/autofill, countgen tooling). The bump watchlist (`docs/templ-components-adoption.md`) step-8 corpus sweep ran with the bump (corpus commits `7b0b584`/`539a136`).

## Risks

1. ~~Coordinated templ gate ahead~~ **EXECUTED 2026-11-07/11-11 (templ-components v1.27.0 + templ v0.3.1100, corpus re-baselined).** The next gate to watch is the reverse direction: upstream `[Unreleased]` carries the Datastar builders + responsive Sparkline; neither requires a templ move.
2. **Amber re-skin is semantic:** any assertion hardcoding yellow-* warning classes is now wrong — goldens were regenerated; keep the map row current.
3. `docs/upstream-filings.md` #52 closure — recorded at the D1 fold (2026-11-04); verify it stays current when the `[Unreleased]` wave lands.

## Recommendations

1. ~~Execute D1~~ **DONE (2026-11-04, scored).** No open adoption findings remain; the three upstream-blocked partials stay until their issues close.
2. Keep the three remaining workarounds exactly until their upstream issues close (they are pinned by guard tests).
3. On the next bump, run the adoption-map corpus sweep (`docs/templ-components-adoption.md` step 8) and fold the `[Unreleased]` Sparkline responsive fix when it ships — it retires the fixed-size workaround.

## How to verify

```console
rg -n "EmptyText" internal/web/ops.templ                      # the D1 fold, live
rg -n 'templ-components' go.mod                               # v1.27.0 x4
rg -n "templ-components" --type-add 'templ:*.templ' -ttempl internal/web | wc -l
rg -c "format\." internal/web/*.templ | head   # utils/format adoption
GOEXPERIMENT=jsonv2 go test ./internal/web/ -run TestTailwindCorpus -count=1   # corpus fresh at pin
```
