# Owner Decision Pack — UI direction questions (plan M2/F6)

- **Created:** 2026-10-15 (SUPERB UI-contract plan, M2)
- **Answers gate:** the date-render change (touches every surface), the
  dashboard control layout, and two upstream-filing decisions. Nothing here
  blocks M1/M3/M4, which landed first.
- **How to answer:** write the choice ID (e.g. "D1: B") next to each
  question, or free-form. Outcomes get recorded in AGENTS.md/ROADMAP
  (plan F7) once answered.

---

## D1 — Date/number vocabulary (status f-15, question g-1)

Every rendered date is US-style ("Oct 9, 2025 22:31"); the operator context
is German (Europe/Berlin buckets, German PBX). Surfaces touched by a switch:
RelativeTime text, search hits, detail facts, chart tooltips.

| Choice                                                  | What it means                                | Tradeoff                                                                                         |
| ------------------------------------------------------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| A. Keep US format                                       | zero code, zero risk                         | operator reads US dates in a German context forever                                              |
| B. EU format (09.10.2025 22:31)                         | one central time-format helper, all surfaces | a bounded change with goldens + screenshots; must pick "de-DE" vs "en-DE hybrid" for month names |
| C. Hybrid: EU dates, keep relative text ("3 hours ago") | most machine-sortable-looking tables         | two formats coexist; needs a rule for when each applies                                          |

**Recommendation:** B — the whole UI already thinks in Europe/Berlin; the
format should match the operator's calendar, not the server's locale
default.

## D2 — Dashboard control rows (status f-5, question g-2)

Presets (7/30-day) and the from/to range form are two stacked rows.

| Choice                       | What it means                                       | Tradeoff                                                                                                                   |
| ---------------------------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| A. Keep two rows (current)   | separation of navigation vs filtering stays visible | taller header on small screens                                                                                             |
| B. Merge into one denser row | one row on desktop                                  | the two controls have different scopes (window vs filter); merging reads as if the range participates in the preset toggle |

**Recommendation:** A (this session's call) — different mental models should
not share a row. Overrule freely; it is a one-templ change either way.

## D3 — Volume chart during quiet weeks (status f-16, question g-3)

A 30-day chart with one spike renders 29 invisible zero-days: honest but
axis-less; at a glance it can read as "broken".

| Choice                                                                   | What it means                                                                        | Tradeoff                                                                 |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| A. Spike-only reading is fine                                            | no action; the tooltip carries exact counts                                          | occasional "is this broken?" moments                                     |
| B. File the axis/baseline request upstream (BarChart baseline/axis prop) | converts a waiting item into an upstream conversation (verify-before-filing applies) | upstream may not want it; local workaround stays until a bump            |
| C. Local workaround: always draw a baseline rule under the bars          | visible "chart is alive" signal in all cases                                         | one more hand-rolled piece to re-check on every version bump (watchlist) |

**Recommendation:** B, with C only if the operator actually looks at the
chart during quiet stretches. (Note: the M3 empty-index pass already made
the truly-empty case honest with a message; D3 is about the one-spike case.)

## D4 — Export honesty note (status f-16 item 16)

The export section carries a multi-line honesty note (keyset pagination,
PII caveats). Wrapping it in a `CollapsibleSection` cuts page noise but
hides the caveat behind a click.

| Choice                                           | What it means                     | Tradeoff                                      |
| ------------------------------------------------ | --------------------------------- | --------------------------------------------- |
| A. Keep visible (current)                        | honesty stays unmissable          | permanent page noise                          |
| B. Collapse with an always-open-by-default state | quieter page; one click to verify | collapsed-on-arrival would defeat the purpose |

**Recommendation:** A for now — the note is short enough that collapsing it
saves less than it risks.

## D5 — AI-share card tone (M3 eyeball finding)

The AI-share StatCard is the only non-signal tone on the dashboard
(library `StatTonePurple`, a violet accent). The signal palette doctrine
re-skins every blue but leaves purple untouched.

| Choice                          | What it means                        | Tradeoff                                             |
| ------------------------------- | ------------------------------------ | ---------------------------------------------------- |
| A. Keep purple                  | "AI" keeps a distinct color identity | one violet accent in an otherwise teal UI            |
| B. Switch to the teal/info tone | full signal-palette uniformity       | AI loses its color differentiator vs the other cards |

**Recommendation:** A — the purple is a deliberate differentiator, not a
leak; it is recorded in docs/templ-components-adoption.md as a
known-off-token component so the version-bump watchlist covers it.
