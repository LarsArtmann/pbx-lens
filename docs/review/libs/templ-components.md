# templ-components — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pin:** `github.com/larsartmann/templ-components v1.26.0` (+ `/htmx`, `/icons`, `/utils` all v1.26.0; go.mod:26-29) · **Status:** USED
**Adoption score:** **95/100 (very high)** — 26.5 of 28 applicable capabilities (25 fully ×1.0 + 3 partially ×0.5; 10 of 38 inventory rows N/A with reasons). The #52 fold-in (`RelativeTimeProps.EmptyText` replacing the local "never" guard) landed 2026-11-04, taking the audit-day 93 to 95.
**Prior reports:** [2026-10-24_templ-components-deep-dive.html](../../research/2026-10-24_templ-components-deep-dive.html) (91/100 at v1.23.0; 38 capabilities) and the delta appendix of its own 2026-10-25 re-run. This report supersedes their version-currency rows (pin now v1.26.0, current), their #52-blocked status, and the "six page surfaces" census (seven as of `segment.templ`). Per-component truth map: `docs/templ-components-adoption.md`.

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

Since v1.26.0 (upstream `[Unreleased]`): canonical CSP dismiss scripts, `utils.Coalesce/Lookup`, and the **templ gate** — v1.27.0 will require a-h/templ ≥ v0.3.1100 (pbx-lens pins v0.3.1070; manual bumps only, dependabot banned). #48/#49/#51/#60 workarounds: **still open upstream, correctly still present locally**.

## Gap analysis

| Finding                                                                                                                                                                                    | Status                                                                                                                                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **D1 — ops zero-time guard now replaceable** (`ops.go:296-298`, `ops.templ:205-206`, tests `ops_page_test.go:172-189`, `relative_time_test.go:173`)                                        | **Missed opportunity (the only one).** Swap to `RelativeTimeProps.EmptyText: "never"`, delete the guard, keep one contract test. `EmptyText` verified at pin (`relative_time.templ:26,34,61,72` — renders "never" incl. datetime suppression + Title honor). ~30 min incl. docs (`docs/upstream-filings.md` is stale: #52 closed 2026-10-29 unrecorded). |
| BarChart `!items-stretch` ×3, Sparkline fixed-size contract, Scrollback `?q=` match path                                                                                                   | Partially used — all upstream-blocked (#48/#60/#51 unfixed), not local negligence                                                                                                                                                                                                                                                                        |
| RelativeTime (until D1 lands)                                                                                                                                                              | Partially used (workaround where a prop now exists)                                                                                                                                                                                                                                                                                                      |
| v1.26.0 new surface (LoadingDialog, HoverCard, Autocomplete)                                                                                                                               | N/A — no blocking-operation/hover/autocomplete use case in a read-only dashboard                                                                                                                                                                                                                                                                         |
| SegmentBar/FilterBar/SegmentedControl, Modal/Tabs/Drawer/CommandPalette/etc., charts/echarts, datastar, errorpage, navigation.Pagination (hand-rolled keyset by design, `calls.templ:156`) | N/A — standing decisions (adoption map rows carry the reasons)                                                                                                                                                                                                                                                                                           |

## Version currency

**Current across the board:** templ-components v1.26.0 (2026-10-31) == latest release; a-h/templ v0.3.1070 == the correct pairing for ≤ v1.26.0 consumers; goldens + class corpus regenerated at this pin (commits `37e30ae`, `d86ca9f`). The bump watchlist (`docs/templ-components-adoption.md`) was executed properly at this bump except the filings-ledger state check (D1's docs step).

## Risks

1. **Coordinated templ gate ahead:** the next templ-components release (≥ v1.27.0) requires templ ≥ v0.3.1100 — a two-module bump with golden/codegen re-baseline. Watch upstream `scripts/check-templ-gate.sh`.
2. **Amber re-skin is semantic:** any assertion hardcoding yellow-* warning classes is now wrong — goldens were regenerated; keep the map row current.
3. `docs/upstream-filings.md` predates all three releases (#52 closure unrecorded) — refresh in the D1 commit.

## Recommendations

1. Execute D1 (fold in `EmptyText`, delete the ops guard, refresh the filings ledger + adoption map row) — takes the score to 95/100.
2. Keep the three remaining workarounds exactly until their upstream issues close (they are pinned by guard tests).

## How to verify

```console
rg -n "EmptyText" ~/go/pkg/mod/github.com/larsartmann/templ-components@v1.26.0/display/relative_time.templ
rg -n "templ-components" --type-add 'templ:*.templ' -ttempl internal/web | wc -l
rg -c "format\." internal/web/*.templ | head   # utils/format adoption
sed -n '294,300p' internal/web/ops.go          # the D1 guard
```
