# skill/docs vs the v1.21.0 pin: FilterBar, SegmentedControl, utils/format presented as current but absent from the released module

Consumers following the skill hit "undefined: display.FilterBar" against the published tag. Ask: version-stamp the skill, or mark unreleased components.

## Symptom

The consumer skill (`skill/SKILL.md`, shipped in this repo) documents components as current API. Consumers pinned to v1.21.0 (latest tag) compile against the module cache and find they do not exist:

| Claim in `skill/SKILL.md`              | v1.21.0 (latest tag)                  | master                                    |
| -------------------------------------- | ------------------------------------- | ----------------------------------------- |
| `FilterBar` (`:44`, `:128`)            | absent (`display/filter_bar.templ` no such file) | absent (only `docs/recipes/horizontal-filter-bar.md`, a forms.Form pattern) |
| `SegmentedControl` (`:44`, `:130`)     | absent (`display/segmented_control.templ` no such file) | absent                    |
| `utils/format` (`:250`)                | absent (`utils@v1.21.0` has no `format/`) | present (`utils/format/` on master, unreleased) |
| `SegmentBar` (referenced in docs)      | absent                                | present (`display/segment_bar.templ`, unreleased) |

Verified against the module cache at `templ-components@v1.21.0` / `utils@v1.21.0` (2026-10-12) and `gh api repos/larsartmann/templ-components/contents/...` for master.

The drift bit this consumer concretely: pbx-lens adopted components per the skill, hit missing symbols at the pin, and now carries a hand-rolled SegmentBar replacement plus an "adoption map" table whose only job is to record which skill claims are false at the pin (`AGENTS.md`, templ-components v1.21.0 adoption map).

## Root cause (source)

The skill is written against the working tree, not any tag: components get documented when they land on master (or when planned), and the skill has no notion of "since vX" or "unreleased". Consumers have no way to tell current-release API from main-branch API — the two failure classes are "component exists only on master" (SegmentBar, utils/format) and "component exists only as a plan/recipe" (FilterBar, SegmentedControl).

## Design

- [ ] Stamp the skill with a version ("documents v1.21.0" or "documents master as of 2026-10-..") so consumers can tell which contract they are reading.
- [ ] Mark unreleased components inline (`SegmentedControl — unreleased`, `utils/format — since v1.22.0-unreleased`) or move them to a separate "coming" section.
- [ ] Same treatment for the module docs under `docs/` that reference unreleased components (recipes naming components that no tag ships).

## To verify

- [ ] A consumer on the latest tag reading the skill can predict, for every named component, whether `go get`-ing the pin gives them the symbol — spot-check FilterBar (no), SegmentBar (no), Table (yes).
- [ ] Next release: the "since" annotations flip mechanically with the tag.

--
💘 Generated with Crush
