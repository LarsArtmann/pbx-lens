# docs/assets — evidence convention

In-repo evidence PNGs only. The convention (plan T6, 2026-10-12):

- **What lives here:** evidence that is expensive to regenerate or pins a
  one-time comparison — upstream-filing before/after captures, action
  proofs (a click's effect, a redirect's landing), and the two README
  screenshots. Referenced by docs and README, so the numbers in
  `docs/templ-components-adoption.md` and filing drafts link to real
  artifacts instead of volatile /tmp paths.
- **What stays out:** the screenshot probe's 24-PNG eyeball sets — those
  are regenerable any time via `node scripts/screenshot-probe.js`
  (self-contained build + seed + serve) and would rot here; keep them in
  /tmp. Eyeball verdicts get recorded as text in the plan/status docs
  that commissioned them.
- **Sizing:** the probe captures full-page PNGs (~50-100 KB each); commit
  sparingly and delete assets whose owning filing is closed upstream
  (the issue text carries its own captures once filed).
- **Naming:** `<surface>-<what>.png` (e.g. `detail-search-jump.png`), no
  dates — git history is the timeline.

Current contents:

| File                          | What it proves                                                                                               | Commissioned by      |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------ | -------------------- |
| `detail-copy-button.png`      | CopyButton on the detail page: "Copy ID" + both numbers, label swapped to "Copied!" after a real click (T20) | templ-UI plan T20/T4 |
| `detail-search-jump.png`      | The search deep link landed: URL `?q=thermostat&at=0#turn-0`, matched turns amber-marked (T21)               | templ-UI plan T21/T4 |
| `dashboard-light.png`         | README screenshot: the quality dashboard (desktop, light)                                                    | plan T6.3            |
| `detail-match-log-mobile.png` | README screenshot: mobile detail with copy buttons + amber match log                                         | plan T6.3            |

Regeneration: the two evidence captures come from the verify flow in
`scripts/verify-ui-contract.js` (the contracts it asserts are these
captures' content); the README shots come from
`node scripts/screenshot-probe.js` output selection.
