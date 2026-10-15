# Status — 2026-10-15 05:18 CEST — SUPERB UI-contract plan execution (M1–M11 + tail)

Session ran the 02-24 SUPERB plan (`docs/planning/2026-10-15_02-24_…`, now
annotated with its execution record). 66 fine tasks commissioned; **52
executed and verified**, 14 remain as harvested TODO_LIST rows (blocked or
owner-gated by design). One unplanned **production bug found and fixed**
(nav-id ETag volatility), one upstream issue filed (#55), one stale-vendor
hash incident triaged to root cause.

## a) FULLY DONE (verified green)

1. **M1 contract probes:** `scripts/verify-style-contract.js` (S1 ds-brand
   teal light+dark, S2 heatmap cell computed `rgb(15,118,110)`, S3
   All-option truth) — ALL PASS; `TestCSSGateBridgesHeatmapBrandToTeal`
   Go twin over the embedded stylesheet; scripts/README wired; CI-lane
   decision recorded (browser probes stay local; the Go gate covers CI).
2. **M2/F6 owner decision pack:** D1–D5 written with options, tradeoffs,
   recommendations (`docs/decisions/2026-10-15_ui-owner-decision-pack.md`).
   F7 (collecting answers) = owner-blocked.
3. **M3 unseen states:** screenshot probe grew 404 + audio-card +
   empty-index captures (39 PNGs total, eyeballed); empty-index exposed
   **collapsed-0px-bar chart cards → fixed** (volume/duration return nil,
   trend explains "No calls in the window yet."); `verify-staleness-live.js`
   proves the banner disappears live without reload.
4. **M4 review tooling:** `golden-pretty-diff.sh` (used for real, twice);
   `findFreePort` + optional `--port` in every probe; `ensureCodegen()`
   bootstrap (go.mod-pinned templ + flake tailwind on clean checkouts);
   port/flag docs in scripts/README.
5. **M5 accessibility:** Contract D keyboard walk (row focus ring is teal,
   copy affordances reachable — no fix needed, ring already ships); print
   check (chrome drops, facts + badges render; probe-asserted + PNG);
   badge-in-`<dd>` reasoned AT-safe (visible text carries meaning).
6. **M6 gate hygiene:** `go mod tidy` clean, `govulncheck` 0 affected,
   `golangci-lint ./...` 0 issues, full test suite 14/14, `nix build` 0;
   gate 60/70 with the single failure being the **documented** nix-build
   OOM casualty (standalone green, doctrine honored).
7. **M7 repo-doc hygiene:** templ fmt-only drift note in flake.nix;
   AGENTS.md back under the 220-line budget (218); adoption map gained the
   Heatmap ds-brand bridge row; gotchas gained the eyeball-after-screenshot
   lesson (F27).
8. **M8 HTTP-stack — THE session find:** live ETag probe proved the
   documented no-cache contract **silently broken**: the library's
   `EnsureID` mints a random mobile-menu id per render → every page body
   unique → ETags never 304. Fixed by pinning `BaseProps.ID` (one line +
   `TestNavIDIsDeterministic`); all surfaces now 304; fragment
   Server-Timing measured (0.4–1.2 ms ≪ poll budgets); full write-up in
   docs/http-stack.md.
9. **M10 upstream:** #48/#49/#51 re-verified OPEN, zero comments, #48
   mechanism re-confirmed at the pin (source lines cited in filings doc);
   **templ-components#55 FILED** (nav-id volatility, checker-clean draft
   kept in docs/drafts/), fold-in watch extended.
10. **M9 harvest:** TODO_LIST eleventh pass (decision pack, MOS poller,
    PII, mobile cards, pixel-diff, ops surface, retention, simplifications,
    dark sweep command); ROADMAP long-tail stubs; #51+#55 joined the
    fold-in row.
11. **M11 spot-pass:** dark `theme-color` fixed (#1e1b4b indigo → #111827;
    the "not prop-overridable" comment was stale — `DarkThemeColor` IS a
    prop); hit-card hover aligned to table rows (100/800); aria-checked
    verified surviving htmx swaps; SectionHeading trial rejected with
    rationale (recorded as D6).
12. **Tail:** M18 TZ vocabulary table in docs/DOMAIN_LANGUAGE.md (F60);
    long-excerpt 200-rune eyeball — wraps cleanly, no fix (F61); T23
    BarChart-vs-LineChart evidence PNG committed per convention (F63);
    `--themes` flag on the screenshot probe = single-command dark sweep
    (F64); hasFilters/aiValue evaluated → already minimal, rewrite
    rejected (F59).

## b) PARTIALLY DONE

1. **F65 (Telnyx 2027-01 calendar confirm):** no calendar tooling/ics in
   this environment — the entry lives in the owner's external calendar;
   repo-side truth (quarterly note in docs/telnyx-live-api.md) stands.
2. **M6 buildflow dev gate:** everything green EXCEPT the in-gate nix-build
   step (OOM under load 24–45); individually verified green per doctrine —
   a quiet-machine full-gate rerun is still owed for a 70/70 stamp.
3. **Plan §Step-1 P2 outcomes:** memo done, answers not (F7) — D1 alone
   gates an all-surfaces date-format change.

## c) NOT STARTED (deliberately — harvested, not dropped)

M12 MOS poller (XL), M13 PII enforcement (L), M14 alert-only pixel-diff,
M15 /api/ops surface, M16 retention surfacing, M17 mobile card-list,
M22 #51 fold-in (upstream-gated), F7 owner-answer collection. All live as
TODO_LIST rows with evidence pointers.

## d) TOTALLY FUCKED UP (and what it cost)

1. **buildflow `nix-hash-fix --fix` corrupted flake.nix** — it wrote the
   pbx-lens vendor hash into `cqrs-lint`'s `fetchFromGitHub` hash and left
   that input's vendorHash as the AAAA placeholder; the auto-commit daemon
   then **committed the corrupted flake** mid-churn. Cost ~25 min of
   confusion (gate failures pointing at a "source.drv hash mismatch");
   root-caused via git history (only one auto-commit carried the wrong
   hash) and reverted. Lesson candidate: never run the hash fixer while the
   tree is dirty/daemon-active.
2. **Stale `.git/index.lock` from the crashed daemon** blocked `git
   restore` for several minutes — removed the (verified-stale, empty, 10-min
   old) lock after confirming no git process existed.
3. **First M3 empty-phase captures were garbage** (captured against the
   dying old serve — `stopProbe` didn't wait for exit). Caught by the
   connection-refused error, fixed by making `stopProbe` resolve on exit
   and re-run; the bad PNGs were overwritten.
4. **Two verification-method errors, both self-corrected:** the keyboard
   walk looked for copy buttons on the list page (they live on detail), and
   F37's first pass injected `.dark` manually instead of clicking the
   toggle (the library only re-syncs on render/click). Both re-run
   correctly; both now encode the correct method permanently.
5. Minor: `templ` field-name slip in the T23 spike (`Label` vs `Name`) and
   a missing Tailwind link in the standalone evidence page — each cost one
   rebuild.

## e) WHAT WE SHOULD IMPROVE

1. The hash-fixer must refuse to run (or refuse to write) when the tree is
   dirty or a daemon commit lands mid-run — file upstream (BuildFlow) with
   the flake.nix corruption evidence.
2. `nix-build-verify` has failed 12/12 and in-gate `nix-build` OOMs under
   load ≥ 30 — the gate should cap its nix concurrency (or the fleet
   should document `--max-concurrency` as the default for this box).
3. `ensureCodegen` regenerates only on mtime; a content-hash check would
   survive branch switches that reset mtimes.
4. The screenshot probe re-captures all 39 PNGs even when one surface
   changed — a `--only <name>` filter would cut eyeball-loop latency.
5. The gate's nix lane and the standalone `nix build` hash differently
   (snapshot vs worktree) — one shared source-pinning mechanism would
   remove the whole "whose hash?" class.

## f) NEXT 50 (ordered by impact; the first ~10 are the real queue)

1. Answer D1–D5 (owner) → date-format change lands everywhere.
2. Quiet-machine full-gate rerun for the 70/70 stamp.
3. File the hash-fixer dirty-tree corruption upstream (BuildFlow).
4. M12 F39–F45: MOS poller when the 2027-01 window opens (design now,
   execute then).
5. M13 F46–F49: PII enforcement point decision + E164/ParticipantHash.
6. M17 mobile card-list: decide card vs scroll, implement.
7. M14 alert-only pixel-diff over the 39-PNG corpus.
8. M15 /api/ops human surface (card vs page).
9. M16 retention surfacing (last purge, retired counts).
10. Watch templ-components#48/#49/#51/#55; fold in on landing (retire
    workarounds + goldens).
11. v0.2.0 cut (CHANGELOG Unreleased is full).
12. Production re-ingest ContentHash v2 (deploy-path blocked).
13. Host deploys with current module options (deploy-path blocked).
14. Dependabot confirmation for newly-direct deps.
15. >1MiB transcript policy decision.
16. E2E cache-semantics hardening (Vary/HEAD/query variants).
17. Coverage pass on caching/serve hot paths.
18. Concurrent-poll hash-cost probe.
19. `/api/ops` stale-pass warning.
20. Cost panel on the dashboard (CostMicros data exists).
21. Backup rotation/shipping opinion.
22. Direction auto-detection design follow-through (did-direction doc).
23. SegmentBar/FilterBar re-eval on next templ-components bump.
24. Search hit-card keyboard focus parity with table rows (focus ring on
    the `<a>` hit card — verify once).
25. `--only <surface>` filter for screenshot-probe (e5.4).
26. Content-hash `ensureCodegen` (e5.3).
27. Shared snapshot hashing for gate-vs-standalone nix (e5.5).
28. Long-transcript `?q=` performance sanity (amber marks over 500 turns).
29. Export CSV header i18n check once D1 answers.
30. Golden normalization audit: is `tc-mobile-menu-NORMALIZED` still needed
    anywhere after the pinned nav id? (probably dead — remove).
31. `verify-ui-contract` + `verify-staleness-live` merge? Two serves of the
    same shape; one runner with subcommands could cut probe time.
32. Add the four probes to a local `just`-free runner script (single
    `scripts/verify-all.sh`) — convenience only.
33. Phone-number formatting pass (E.164 display grouping) behind D1.
34. Table column width audit on 200-rune excerpts at 768px (tablet gap in
    the eyeball matrix).
35. Print stylesheet: add transcript page-break rules (long logs split
    mid-turn today).
36. `favicon.svg` dark-variant via `prefers-color-scheme` media attr.
37. robots.txt + noindex meta (public LAN tool hygiene).
38. Security headers pass (CSP report-only dry run on / and detail).
39. Rate-limit the recording proxy explicitly (auth exists; throttle
    documented as not honored via XFF).
40. CHANGELOG entry for the nav-id fix + empty-index ladder (Unreleased).
41. AGENTS.md: retire the "38-PNG" count drift risk (say "the probe set").
42. docs/README screenshots refresh after the theme-color/hover fixes.
43. Spotlight the heat-map ring in the README quality section (new teal
    evidence PNG exists).
44. Benchmark: `ListCalls` page latency at 10k seeded rows (index sanity).
45. Fuzz-ish ingest pass: malformed JSONL lines never panic (probably
    covered; one property test would pin it).
46. `smoke -age`/`-with-recording` go doc comment + `--help` wording pass.
47. Move `/tmp/long-excerpt-capture.js` logic into screenshot-probe as a
    fixture option or delete the file.
48. Retire `TestSearchHintProse`'s chip-absence assertion if the chip never
    comes back (or keep as a tripwire — decide once).
49. Add `vendorHash.nix` provenance comment (which commit set it, which
    command refreshes it) — the placeholder incident showed the gap.
50. Quarterly Telnyx re-verify (2027-01, owner calendar F65).

## g) Questions only the owner can answer

1. **D1 date/number vocabulary:** EU (09.10.2025), US (current), or hybrid?
   Recommendation stands: EU. This gates the single largest cross-surface
   change.
2. **D3 quiet-week volume chart:** do you actually look at the 30-day chart
   during quiet stretches — is an upstream axis/baseline ask (#48 follow-up
   or a local baseline rule) worth it, or is spike-reading fine?
3. **M13 PII boundary:** enforce E164/ParticipantHash at INGEST (storage is
   normalized; raw-at-rest doctrine bends) or at READ (raw stays, every
   surface gates)? The export gate already chose READ; should storage match
   eventually, or is export-time-only the permanent answer?
