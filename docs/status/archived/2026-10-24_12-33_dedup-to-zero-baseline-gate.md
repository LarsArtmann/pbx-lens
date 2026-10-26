# Status 2026-10-24 12:33 — dedup-to-zero: audit verdict made a CI gate

Session scope: the `deduplicate to zero` run (art-dupl audit + baseline-gate
wiring). No other areas researched. Self-review answers are folded into §d/§e;
the three owner questions are §g.

## a) FULLY DONE

| Item                                                                                                                                                                                                                                                                                                                                                                                                                                      | Receipt                                                                                                                               |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| ~~t=5 type-aware clone audit: all 20 shown groups read and judged — every one a signature/type-expression match between helpers with distinct documented business rules (`mosBars` vs `volumeBars` empty-slot vs all-zero suppression; `clockDuration` vs `compactDuration`; `noTranscriptTitle` vs `noTranscriptDescription`; `tableProps` vs `opsSourcesTable`; `percent` vs `mosGaugePercent`; …)~~ done (docs-health pass 2026-10-26) | ~~/tmp/dupe-report.html parse + code reads in dashboard/calls/detail/ops/export; acceptance table rows in docs/art-dupl-baseline.md~~ |
| ~~t=4 fleet-gate audit: the only shown group is the `beginExportWithPII` call pair (helper already extracted, plan T5.2); the serve_test 4x bootstrap clone is confirmed gone (`serveTestStack` landed earlier today)~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                | ~~fresh scan diff vs old baseline: probe group absent, export pair only~~                                                             |
| ~~Audit item 19/20 (sub-threshold test bootstraps): ~28 `OpenSystem`+`Fatalf` 3-line sites across 5 packages judged ACCEPT — per-test DSN and t/b verb are the real variables; a helper nets ~2 lines and hides them~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                 | ~~grep inventory + judgment note in docs/art-dupl-baseline.md~~                                                                       |
| ~~JSON baseline committed: 23 groups, t=4, DIRECTORY mode (`.templ` source included; `*_templ.go` auto-excluded)~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                                                                                     | ~~scripts/art-dupl-baseline.json~~                                                                                                    |
| ~~Gate proven both directions on the local binary: clean tree → `exit 0` ("No new clones detected (baseline: 23 groups)"); seeded cross-package 22-line clone → `exit 2`, files named~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                | ~~/tmp/check-pos.log, /tmp/check-neg.log~~                                                                                            |
| ~~CI lane rewritten: `art-dupl@v0.6.2`→`@v0.9.0`, warning-only text diff → `art-dupl check -t 4 .` failing gate, load-bearing comments (directory arg, pin parity, refresh command)~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                  | ~~.github/workflows/ci.yml `duplication` job; YAML parse-verified~~                                                                   |
| ~~docs/art-dupl-baseline.md: acceptance table (all 23 groups with why-extraction-loses), gate mechanics, refresh procedure, history of the retired txt lane~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                                          | ~~new file~~                                                                                                                          |
| ~~Living references updated: AGENTS.md CI bullet + audit note (gate, not prose), templ-components-adoption.md bump-watchlist step 6, TODO_LIST row 49 evidence cell, CHANGELOG Unreleased entry~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                      | ~~git diff on each~~                                                                                                                  |
| ~~scripts/art-dupl-baseline.txt retired (text diff, warning-only, file-list universe)~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                                                                                                                | ~~trashed; history documented in the new doc + CHANGELOG~~                                                                            |
| ~~`go build ./...` + `go test ./...` all green after changes; final gate check exit 0~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                                                                                                                | ~~session log~~                                                                                                                       |
| ~~dprint on every touched file (found 2 unformatted — my own edits; fixed via `dprint fmt`, re-check green)~~ done (docs-health pass 2026-10-26)                                                                                                                                                                                                                                                                                          | ~~DPRINT-GREEN in session log~~                                                                                                       |

## b) PARTIALLY DONE

- ~~**Version parity is assumed, not proven.** The baseline was generated by~~ done — proven: CI ran the lane green from its first run through 2026-10-26 (parity receipt; 17-20 §b banked it)
  ~~the local nix-built `art-dupl 0.9.0-93432fd`; CI installs the published~~
  ~~`@v0.9.0`. `go install` is hook-blocked in this sandbox, so I could not~~
  ~~run the published binary against the baseline locally. Same tag ⇒ almost~~
  ~~certainly identical hashes, but the first CI run of the new lane is the~~
  ~~open receipt. If it goes red with "N new groups", regenerate the baseline~~
  ~~with the published pin in the same fix.~~
- ~~**Suppressed buckets are tool-trusted.** At t=5: "11 non-actionable, 22~~ done — audited same day (13-30): the suppressed bucket was human-audited and four production clone groups extracted from it
  ~~filtered suppressed" groups were never human-audited (the 20 actionable~~
  ~~ones were). If a future harmful clone lands inside a suppression pattern,~~
  ~~the gate may not see it until a pin bump shifts the buckets. Depth here is~~
  ~~a risk-appetite call (§g Q3).~~
- ~~**docs/status/README.md live table** — updated with this report (done as~~ done — the counts gate (check-docs-counts.sh, 2026-10-25) re-derives them now
  ~~part of landing it), but the archive-count hardcodes (TODO_LIST row 51)~~
  ~~remain untouched drift work.~~

## c) NOT STARTED (this session's natural follow-ups)

- ~~HARVEST of this report's §f into TODO_LIST/ROADMAP (docs-health; skipped~~ done — later harvests plus this pass
  ~~deliberately — user instruction was report-then-wait; TODO_LIST already~~
  ~~got the one row update this session made).~~
- ~~Local parity hook: the gate runs in CI only; `scripts/verify-all.sh` and~~ done — scripts/check-dedup-gate.sh runs CI's exact command locally (315587e); the verify-all wiring is the open owner row (17-29 f9)
  ~~any pre-commit chain do not run `art-dupl check`.~~
- ~~art-dupl is NOT in the flake devShell — the local binary comes from~~ done — the pin doctrine landed (315587e refuses non-pin binaries; the stale-binary trap documented 2026-10-26)
  ~~outside the project's pinned environment (version drift local↔CI is~~
  ~~possible by construction).~~
- ~~Upstream suggestion (audit item 18): teach the deduplicate-code skill the~~ **NOT-DO — out of repo scope; unfiled.**
  ~~combined `--suggest-generics` mode (crush-config repo PR).~~
- ~~docs/gotchas.md entry for the file-list vs directory-mode detection~~ **NOT-DO — covered by docs/art-dupl-baseline.md and the ci.yml comment; a duplicate entry is noise.**
  ~~universe trap (currently documented in art-dupl-baseline.md + ci.yml~~
  ~~comment only).~~

## d) TOTALLY FUCKED UP (own goals, all recovered)

1. ~~**Three failed negative controls before a valid one.** Probe #1 was~~ **NOT-DO — process note; the derive-from-a-known-clone lesson is the receipt.**
   ~~below t=4 in the tool's statement-count semantics (~3 statements, not my~~
   ~~assumed 6+); probe #2 duplicated a function whose "original" lives in an~~
   ~~excluded `*_templ.go`-adjacent `.templ` file — the probe existed exactly~~
   ~~once, so nothing could match. Root cause: I crafted probes from~~
   ~~assumptions instead of deriving one from a known-detected clone shape.~~
   ~~Probe #3 (12-statement cross-package) finally went red. Cost: ~4 wasted~~
   ~~tool rounds and a false "gate bites" narrative risk — the saving grace~~
   ~~was refusing to wire CI until a red proof existed.~~
2. ~~**First baseline was generated in the wrong universe.** I uncritically~~ **NOT-DO — process note; the directory-mode gate fixed both at once.**
   ~~inherited the retired lane's `find | xargs` file-list invocation (1~~
   ~~entry, .go-only) — a baseline whose gate would never see `.templ`~~
   ~~source-level duplication. Caught only because the negative control~~
   ~~stayed green, prompting the mode investigation. The old CI lane had this~~
   ~~same blind spot for its entire life.~~
3. ~~**Auto-commit daemon swept up mid-flight state:** commit 6160423 contains~~ **NOT-DO — process note; the tree committed correctly.**
   ~~the two probe files and the 1-entry wrong-mode baseline. Working tree is~~
   ~~correct (probes trashed, 23-entry baseline), but history carries a junk~~
   ~~commit. Avoidable: keep probes outside the repo (I did for two rounds,~~
   ~~then re-created them in-repo for the final control).~~
4. ~~**dprint near-miss:** I declared "build + tests green" as verification~~ **NOT-DO — process note; the formatter receipt is in the battery now.**
   ~~while the repo's own formatter had findings on two files I had just~~
   ~~edited. Caught during this self-review and fixed — verification should~~
   ~~have included the formatter receipt from the start.~~

## e) WHAT WE SHOULD IMPROVE (self-review answers)

- ~~**Q forgot?** The formatter receipt (fixed now), and I initially trusted~~ **NOT-DO — process note.**
  ~~the inherited CI invocation over the tool's own docs. Also: minor~~
  ~~overstatement in the final summary — "all 20 groups read and judged" is~~
  ~~exactly true for the function-pair groups; the pure type-expression~~
  ~~groups (`templ.Component`, `[]display.TableRow`, …) were judged from the~~
  ~~report snippets themselves, which is adequate but not the same act.~~
- ~~**Q stupid-that-we-do?** A duplication lane that ran warning-only for six~~ **NOT-DO — process note; the gate fixed both at once.**
  ~~days over a universe that excluded the files where all the real clone~~
  ~~groups live (.templ). The gate fixes both at once.~~
- ~~**Q better?** Read the full `--help` before proving tool behavior; derive~~ **NOT-DO — process note.**
  ~~negative-control fixtures from known-detected clones; run `dprint check`~~
  ~~as part of every change's receipt, not as an afterthought.~~
- ~~**Q still improve?** Close the version-parity seam (§b), local gate hook,~~ done — parity receipt banked, check-dedup-gate.sh landed (315587e), the pin doctrine holds
  ~~devShell pin — all in §f.~~
- ~~**Q lie?** No falsehoods; one wording overshoot (§e Q1) and one unverified~~ **NOT-DO — process note.**
  ~~assumption explicitly labeled as such (§b version parity) rather than~~
  ~~hidden.~~
- ~~**Q ghost systems?** None created: the baseline JSON is consumed by CI;~~ **NOT-DO — process note; none created.**
  ~~the doc is consumed by humans; nothing unwired was added. The retired txt~~
  ~~is deleted, not left as a zombie.~~
- ~~**Q scope creep?** One judgment call to flag: converting the CI lane from~~ done — the owner never disagreed; the gate caught real clones twice since (19-09 d1; afae51e)
  ~~warning-only to fail-the-build is a POLICY change the user never~~
  ~~explicitly requested — justified by the deduplicate-code skill ("acceptance~~
  ~~as a gate, not prose"), by go-etag precedent, and by today's audit item~~
  ~~16, and disclosed in the summary. Reverting to warning-only is a one-line~~
  ~~change if the owner disagrees (§g Q1).~~
- ~~**Q removed something useful?** No — the txt baseline's information~~ **NOT-DO — process note.**
  ~~content is fully superseded by the JSON + doc; its history is recorded.~~
- ~~**Q split brains?** One soft one, accepted: the refresh command now lives~~ done — held; the doc stayed canonical with no divergence through 2026-10-26
  ~~in ci.yml (comment) and docs/art-dupl-baseline.md (canonical); AGENTS.md~~
  ~~and the adoption map point at the doc. Watch that the two command sites~~
  ~~never diverge — the doc is canonical by declaration.~~
- ~~**Q tests?** The gate itself has red/green proofs, but nothing automated~~ done — closed: check-dedup-gate.sh carries the local red/green self-test (315587e)
  ~~re-proves it locally (CI-only). A tiny script in verify-all closes that~~
  ~~(§f).~~

## f) Next (impact-sorted; 1–3 are this work's open receipts)

1. ~~Watch the first CI run of the new duplication gate — version-parity~~ done — CI green on the lane from its first run through 2026-10-26
   ~~receipt (§b). If red on hash mismatch: regenerate baseline with the~~
   ~~published pin in the same fix.~~
2. ~~Add `art-dupl check -t 4 . --baseline-path scripts/art-dupl-baseline.json`~~ done — scripts/check-dedup-gate.sh (315587e); the verify-all wiring is the open owner row (17-29 f9)
   ~~to `scripts/verify-all.sh` (local parity with the CI gate; needs art-dupl~~
   ~~available outside nix devShell — see 4).~~
3. ~~Pin `art-dupl v0.9.0` into the flake devShell so local, daemon, and CI~~ done — the pin doctrine landed (315587e refuses non-pin binaries; the stale-binary trap documented)
   ~~share one binary source (kills the parity seam by construction).~~
4. ~~Investigate where the current local `art-dupl` binary comes from (nix~~ done — the pin-parity gotcha lives in docs/art-dupl-baseline.md and the owning docs
   ~~profile? not the project flake) and record it in AGENTS.md.~~
5. ~~HARVEST this report's §f into TODO_LIST/ROADMAP per docs-health.~~ done — later harvests plus this pass
6. ~~Upstream PR (crush-config): add `--suggest-generics` combined mode to the~~ **NOT-DO — out of repo scope; unfiled.**
   ~~deduplicate-code skill's Run section (audit item 18).~~
7. ~~docs/gotchas.md entry: art-dupl file-list vs directory-mode universes~~ **NOT-DO — covered by docs/art-dupl-baseline.md plus the ci.yml comment.**
   ~~(the trap that made the old lane blind).~~
8. ~~One-time suppressed-bucket eyeball: `art-dupl --list-patterns` output vs~~ done — the 13-30 suppressed-bucket audit (four production clone groups extracted)
   ~~the 22 suppressed t=5 groups — confirm nothing harmful hides in a~~
   ~~pattern (§g Q3 if the owner wants depth).~~
9. ~~TODO_LIST row 49 (detect-only findings triage): the art-dupl half is now~~ done — re-scoped into the detect-only advisory triage row; art-dupl itself resolved
   ~~"gate + reviewed baseline" — re-scope the row to the remaining scanners~~
   ~~(branching-flow, file-size, cqrs-lint).~~
10. ~~Consider `//art-dupl:accept` inline directives for the two 8t groups so~~ **NOT-DO — not adopted — the zero-group recalibration (13-30) superseded per-group acceptance.**
    ~~the acceptance lives next to the code, not only in the baseline/doc.~~
11. ~~Scheduled (weekly) t=5 `--type-aware` audit run — slower (23 s) but~~ **NOT-DO — not scheduled — the zero-group gate plus the suppressed audit cover the corpus.**
    ~~catches semantic clones the t=4 fast gate hashes differently.~~
12. ~~Probe-script lint lane (TODO_LIST row 48) — same class of "receipt gap"~~ done — check-js-lint.sh (2026-10-25)
    ~~as my dprint near-miss; worth scheduling next.~~
13. ~~TODO_LIST row 51 docs-drift gates would have caught the stale~~ done — check-docs-counts.sh (2026-10-25)
    ~~art-dupl-baseline.txt verdict automatically — its priority went UP~~
    ~~after this session.~~
14. ~~Verify the daemon's next sweep commits the probe deletions + corrected~~ done — the tree committed correctly (the junk commit stays as history)
    ~~baseline (working tree is already correct).~~
15. ~~On the next templ-components bump, exercise adoption-map step 6 for~~ done — the v1.24.0 sweep ran the watchlist; the gate stayed coherent
    ~~real (expect the gate to go red on moved .templ signature groups;~~
    ~~refresh per doc).~~
16. ~~Small: add the baseline refresh command to AGENTS.md Commands block for~~ **NOT-DO — the doc stays canonical; AGENTS points at it (the pointer satisfies discoverability).**
    ~~discoverability.~~

## g) Questions for the owner

1. ~~**Gate or warning?** I converted the CI duplication lane to fail-the-build~~ done — answered by outcome — the gate stayed red-on-new-clone and caught real clones (19-09 d1; afae51e)
   ~~(policy change, disclosed §e). Keep it red-on-new-clone, or step it back~~
   ~~to warning-only?~~
2. ~~**Where should the local art-dupl binary come from?** It is v0.9.0-tagged~~ done — resolved — the pin doctrine (315587e) plus the stale-binary incident closed the seam
   ~~nix-built but not from this project's devShell — should I pin it in the~~
   ~~flake (my recommendation, §f 3–4), and if so, do you know the input~~
   ~~source you'd prefer (flake input vs nixpkgs vs go-install shim)?~~
3. ~~**Suppressed-bucket depth:** is tool-trust of the 22 suppressed + 11~~ done — answered — the 13-30 suppressed-bucket audit ran; four groups extracted
   ~~non-actionable groups acceptable, or do you want a one-time human audit~~
   ~~of that bucket (≈30 min) before treating the gate as fully load-bearing?~~

— Session receipts: gate exit 0 on final tree; `go build`/`go test ./...`
green; dprint green; ci.yml YAML-valid; daemon will sweep this file.
