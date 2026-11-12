# templ-components — utilization audit (pbx-lens)

**Date:** 2026-11-04 (refreshed **2026-11-11** — the coordinated templ gate bump landed) · **Pin:** `github.com/larsartmann/templ-components v1.27.0` (+ `/htmx`, `/icons`, `/utils` all v1.27.0; go.mod:31-34; == proxy latest) with the paired `a-h/templ v0.3.1100` (go.mod:6; == proxy latest) · **Status:** USED
**Adoption score:** **95/100 (very high)** — 26.5 of 28 applicable capabilities (25 fully ×1.0 + 3 partially ×0.5; 10 of 38 inventory rows N/A with reasons). The #52 fold-in (`RelativeTimeProps.EmptyText` replacing the local "never" guard) landed 2026-11-04, taking the audit-day 93 to 95.
**Prior reports:** [2026-10-24_templ-components-deep-dive.html](../../research/2026-10-24_templ-components-deep-dive.html) (91/100 at v1.23.0; 38 capabilities) and the delta appendix of its own 2026-10-25 re-run. This report supersedes their version-currency rows (pin now v1.26.0, current), their #52-blocked status, and the "six page surfaces" census (seven as of `segment.templ`). Per-component truth map: `docs/templ-components-adoption.md`.

> **Refreshed 2026-11-12:** follow-up audit pass on the bump the 2026-11-11 refresh recorded. **Filings closure wave:** all six tracked filings (#48/#49/#50/#51/#55/#63) are **CLOSED (2026-11-06/07) — AFTER the v1.27.0 tag (2026-11-04)**, so none of their fixes are in the pin (re-verified in the v1.27.0 module cache: `ScrollbackLine` still has no ID/Class — #51; no CopyButton page-level mount — #63; nav `EnsureID("mobile-menu", props.ID)` semantics per navigation/nav.templ — #55). The local workarounds therefore all stay (`!items-stretch`, focus-visible CSS, hand-rolled transcriptMatchLog, pinned nav id, no-CopyButton-in-htmx-fragment rule) and form the retirement queue for the **next** release, which already carries the fixes — master is 88 commits ahead of the tag.
> **Three new pin behaviors the bump absorbed:** (a) templ v0.3.1100 codegen emits an icon→text separator space inside anchors — `pagination_test.go`'s `extractHref` (which matched on a leading `>`) was fixed for it; (b) v1.27.0 renders a new CSP-safe validity-bridge singleton script (`tcValidityAttached`, `forms/form.templ:266`) **without a nonce** — under enforcing CSP the browser blocks it and native form validation silently dies; pbx-lens threaded `Nonce` into `forms.FormProps` at all three call sites (`calls.templ:215`, `dashboard.templ:101`, `search.templ:29`); (c) codegen goldens show a **duplicated `</script>`** close tag after the tc toast singleton script (one per page golden, all 9 of them; upstream's own `feedback/testdata/toast_container.golden` at the pin has the same double) — verified against master: **already fixed upstream** (toast scripts now render via `utils.ScriptComponent`, which also owns the omit-empty nonce rule; master golden is 1:1), so no new filing needed — it arrives with the next release.
> **Adoption score:** **95/100 (very high) — unchanged.** Nothing in the v1.27.0 pin adds adoptable capability (all six filings closed post-tag), so the census cannot move; the three behaviors above are a test-helper fix, a nonce prop pass-through, and an upstream markup wart — none touch adoption.
> **Version currency re-verified:** v1.27.0 **is** the latest release (checked `gh release list -R larsartmann/templ-components` and local `git tag --sort=-v:refname`) with a-h/templ v0.3.1100 as its required pairing — matching the header pins (go.mod:6, 31-34).

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

Since v1.26.0, **v1.27.0 released 2026-11-04 and pbx-lens took the coordinated bump** (templ-components v1.26.0→v1.27.0 + a-h/templ v0.3.1070→v0.3.1100, one act — the two-module risk below executed cleanly; class corpus + goldens regenerated with it, `TestTailwindCorpusMatchesPinnedVersion`/`…RenderedClasses` green at HEAD). v1.27.0 itself is a docs/lint/tooling release for pbx-lens' surface (docs-site chrome bar, dark-mode scan extension to `ring-offset-*`/`shadow-*`, `AccordionProps.Exclusive`, Slider track fill, autofill tint, count-table codegen) — no component pbx-lens renders changed classes or props. Upstream `[Unreleased]` (the likely v1.28.0) is Datastar-heavy: typed Datastar reactive builders, the CSP nonce mode (`layout.DatastarNonceAttrs`), `Stacked` form-field wrapping (#53), `CopyButtonScript` for fragment-mounted buttons (#63), and the responsive Sparkline — watch the responsive-Sparkline fold-in, it retires the fixed-size workaround below. #48/#49/#51 workarounds: **still open upstream, correctly still present locally**. [2026-11-12: superseded — #48/#49/#51 (+#50/#55/#63) all CLOSED 2026-11-06/07, but post-tag: the fixes are not in v1.27.0, so the workarounds remain correct; retirement queued for the next release.]

## Gap analysis

| Finding                                                                                                                                                                                    | Status                                                                                                                                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~D1~~ — **FOLDED 2026-11-04** (ops.templ:204 passes `EmptyText: "never"`, the guard is gone, the contract test remains at ops_page_test.go:173)                                                                 | Resolved — the score's last missed opportunity is banked; no open findings below it                                                        |

| BarChart `!items-stretch` ×3, Sparkline fixed-size contract, Scrollback `?q=` match path                                                                                                   | Partially used — all upstream-blocked (#48/#60/#51 unfixed at the 2026-11-04 audit), not local negligence [2026-11-12: #48/#51 closed post-tag — their fixes land in the next release; #60 not re-checked this pass]; the Sparkline item has an upstream fix pending in `[Unreleased]`                                                                                                                                                                                                       |
| RelativeTime                                                                                                                                                                              | Fully leveraged since the D1 fold (prop-driven, no workaround)                                                                                                                                                                                                                                                                                            |
| v1.26.0 new surface (LoadingDialog, HoverCard, Autocomplete)                                                                                                                               | N/A — no blocking-operation/hover/autocomplete use case in a read-only dashboard                                                                                                                                                                                                                                                                         |
| SegmentBar/FilterBar/SegmentedControl, Modal/Tabs/Drawer/CommandPalette/etc., charts/echarts, datastar, errorpage, navigation.Pagination (hand-rolled keyset by design, `calls.templ:156`) | N/A — standing decisions (adoption map rows carry the reasons)                                                                                                                                                                                                                                                                                           |

## Version currency

**Current across the board (2026-11-11):** templ-components v1.27.0 (2026-11-04) == latest release, paired with a-h/templ v0.3.1100 == latest — the coordinated gate bump the audit flagged as risk #1 is EXECUTED (one act, corpus + goldens regenerated, corpus-freshness tests green at HEAD). The v1.27.0 delta touched no component pbx-lens renders (docs-site chrome, dark-mode lint scan, Accordion/Slider/autofill, countgen tooling). The bump watchlist (`docs/templ-components-adoption.md`) step-8 corpus sweep ran with the bump (corpus commits `7b0b584`/`539a136`).

**Refreshed 2026-11-12:** still current — v1.27.0 == latest release (re-verified via `gh release list` + local tags), a-h/templ v0.3.1100 (go.mod:6) its pairing, all four templ-components modules at v1.27.0 (go.mod:31-34). Master is **88 commits ahead** of the v1.27.0 tag and already carries the six closed filings' fixes (#48/#49/#50/#51/#55/#63, closed 2026-11-06/07) plus the duplicated-`</script>` toast-script fix — the next release is the workaround-retirement wave.

## Risks

1. ~~Coordinated templ gate ahead~~ **EXECUTED 2026-11-07/11-11 (templ-components v1.27.0 + templ v0.3.1100, corpus re-baselined).** The next gate to watch is the reverse direction: upstream `[Unreleased]` carries the Datastar builders + responsive Sparkline; neither requires a templ move.
2. **Amber re-skin is semantic:** any assertion hardcoding yellow-* warning classes is now wrong — goldens were regenerated; keep the map row current.
3. `docs/upstream-filings.md` #52 closure — recorded at the D1 fold (2026-11-04); verify it stays current when the `[Unreleased]` wave lands.

## Recommendations

1. ~~Execute D1~~ **DONE (2026-11-04, scored).** No open adoption findings remain; the three upstream-blocked partials stay until their issues close.
2. Keep the three remaining workarounds exactly until their upstream issues close (they are pinned by guard tests). **→ Updated 2026-11-12:** the tracked issues (#48/#49/#50/#51/#55/#63) all closed post-pin (2026-11-06/07) — keep the workarounds until the **next** release ships their fixes, then fold in as one batch.
3. On the next bump, run the adoption-map corpus sweep (`docs/templ-components-adoption.md` step 8) and fold the `[Unreleased]` Sparkline responsive fix when it ships — it retires the fixed-size workaround.

## How to verify

```console
rg -n "EmptyText" internal/web/ops.templ                      # the D1 fold, live
rg -n 'templ-components' go.mod                               # v1.27.0 x4
rg -n "templ-components" --type-add 'templ:*.templ' -ttempl internal/web | wc -l
rg -c "format\." internal/web/*.templ | head   # utils/format adoption
GOEXPERIMENT=jsonv2 go test ./internal/web/ -run TestTailwindCorpus -count=1   # corpus fresh at pin
```

**Refreshed verify (2026-11-12, at the v1.27.0 pin):**

```console
rg -n "tcValidityAttached" ~/.cache/go-mod/github.com/larsartmann/templ-components@v1.27.0/forms/form.templ   # validity bridge at the pin
rg -o "</script></script>" internal/web/testdata/*.golden | wc -l   # duplicated toast-script close tag — fixed on master, not in the pin
rg -n "Nonce" internal/web/calls.templ internal/web/dashboard.templ internal/web/search.templ   # nonce threaded at the three forms.Form call sites
rg -n "type ScrollbackLine" -A 6 ~/.cache/go-mod/github.com/larsartmann/templ-components@v1.27.0/display/scrollback.templ   # ID/Class still absent (#51 fix is post-pin)
git -C ~/projects/templ-components rev-list --count v1.27.0..master   # master ahead of the pin
```
