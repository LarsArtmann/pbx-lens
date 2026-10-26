<!-- docs/drafts/README.md — one row per draft: file, upstream issue, state.
     Filed drafts stay as historical write-ups (FILED banner on top); LIVE
     drafts wait on owner review. Upstream state detail: docs/upstream-filings.md.
     Created 2026-10-12 (templ-UI Pareto plan T26). -->

# Filing drafts

| Draft                                                             | Upstream issue          | State                                                                                                               |
| ----------------------------------------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `2026-10-12_buildflow-gomod-update-tool-timeout.md`               | BuildFlow#45            | FILED — CLOSED upstream 2026-10-12 (fixed)                                                                          |
| `buildflow-spawn-starvation-gate.md`                              | BuildFlow#50            | FILED — CLOSED upstream 2026-10-12                                                                                  |
| `2026-10-15_buildflow_nix-hash-fix_wrong-attribute.md`            | BuildFlow#55            | FILED — CLOSED upstream 2026-10-16 (local binary may predate fix)                                                   |
| `2026-10-12_dependabot-gomod-unrelated-bump-deletes-requires.md`  | dependabot-core#16538   | FILED — OPEN upstream (owner watches for maintainer questions)                                                      |
| `2026-10-12_v0.1.0-release-notes.md`                              | —                       | local artifact — consumed at the v0.1.0 tag (2026-10-12)                                                            |
| `2026-10-12_v0.2.0-release-notes.md`                              | —                       | local artifact — EXECUTED: v0.2.0 tagged 2026-10-15, gh release cut                                                 |
| `go-appkit-testkit-drain-delay-cost.md`                           | go-appkit#26            | FILED — CLOSED upstream 2026-10-17 (fold-in ready, TODO_LIST row)                                                   |
| `statix-exit0-config-error-footgun.md`                            | —                       | LIVE — unfiled, owner review box unchecked by design (T9, 2026-10-18; dupe search clean: molybdenumsoftware/statix) |
| `templ-components-card-titletag-h3-level-skip.md`                 | —                       | LIVE — unfiled, owner review box unchecked by design (T8, 2026-10-18; dupe search clean)                            |
| `templ-components-copybutton-empty-nonce-unconditioned-script.md` | —                       | LIVE — unfiled, owner review box unchecked by design (T7, 2026-10-18; cross-refs #63/#24)                           |
| `go-atomic-write-commit-primitive.md`                             | —                       | LIVE — unfiled, owner review box unchecked by design                                                                |
| `go-cqrs-lite-cqrs-lint-c009-e005.md`                             | go-cqrs-lite#64         | FILED — CLOSED upstream 2026-10-17 (C009 half; E005 superseded by ADR-0123 note)                                    |
| `go-cqrs-lite-memory-engine-column-resolution.md`                 | go-cqrs-lite#63         | FILED — CLOSED upstream 2026-10-17                                                                                  |
| `go-cqrs-lite-sqlite-time-filter-bind.md`                         | go-cqrs-lite#62         | FILED — CLOSED upstream 2026-10-17 (in-repo fix stays)                                                              |
| `go-etag-server-flush-no-observability.md`                        | go-etag#2               | FILED — OPEN upstream                                                                                               |
| `go-nix-helpers-templ-formatter-gopkg.md`                         | go-nix-helpers#7        | FILED — OPEN upstream                                                                                               |
| `templ-components-barchart-vertical-zero-height.md`               | templ-components#48     | FILED — OPEN upstream (own repo; re-verified 2026-10-17)                                                            |
| `templ-components-scrollback-line-hook.md`                        | templ-components#51     | FILED — OPEN upstream (own repo; re-verified 2026-10-17)                                                            |
| `templ-components-skill-docs-drift-v1.21.0.md`                    | templ-components#50     | FILED — OPEN upstream (own repo; re-verified 2026-10-17)                                                            |
| `templ-components-tablerow-focus-indicator.md`                    | templ-components#49     | FILED — OPEN upstream (own repo; re-verified 2026-10-17)                                                            |
| `2026-10-15_templ-components_nav-id-etag-volatility.md`           | templ-components#55     | FILED — OPEN upstream (own repo; re-verified 2026-10-17)                                                            |
| `treefmt-nix-545-comment.md`                                      | numtide/treefmt-nix#545 | FILED — filed upstream 2026-10-02; OPEN (re-verified 2026-10-17)                                                    |
| `2026-10-25_comment-52-ops-use-case.md`                           | templ-components#52     | FILED — upstream comment drafted (registered 2026-10-26; was missing from this table)                               |
| `2026-10-25_release-bodies-v0.2.1-vs-v0.3.0.md`                   | —                       | local artifact — RESOLVED: v0.3.0 cut 2026-10-26 (registered 2026-10-26; was missing from this table)               |
| `2026-10-26_v0.3.0-release-notes-shipped.md`                      | —                       | LIVE — tag shipped 2026-10-26; gh release object awaits the owner token (registered 2026-10-26)                     |
| `2026-10-26_v0.3.0-release-notes-paste-ready.md`                  | —                       | LIVE — the shipped notes with the meta header stripped; the file `gh release create --notes-file` consumes (T3)     |
| `2026-10-26_v031-vs-wait-one-pager.md`                            | —                       | LIVE — owner decision: repair the misprinted v0.3.0 tag with v0.3.1, or wait for v0.4.0 (T3; default: wait)         |
| `2026-10-26_tag-ci-first-push-watch-checklist.md`                 | —                       | LIVE — what to watch when the first `v*` tag proves the `cf6e393` trigger live (T3; cutting probe tags stays §g1)   |
| `2026-10-26_advisory-triage-jsonv2-inventory.md`                  | —                       | LIVE — T13 dispositions: branching/file-size ACCEPTED, the jsonv2 flip surface ordered P0-P3 (the standing flip order) |
