# pbx-lens: Session Status Report — 2026-10-11 04:59 CEST

_Written by the resuming agent at the "What did you forget?" interruption
point. Scope: the "GET SHIT DONE, WHOLE TODO LIST" mandate session
(04:10-04:59), which continued the backend-first execution of the 00-09
Pareto plan. Tree state at writing: auto-committed by the daemon, working
tree clean, all tests green, lint gate at ZERO findings._

## What this session completed (a)

### Quality gates — from 382 findings to zero

- **golangci-lint: 382 → 0 findings.** Real code fixes (errcheck on hash
  writers and file closes, err113 sentinel `ErrUnparseableTimestamp` in
  FlexTime, magic numbers → named constants across telnyx/cdr/projection,
  CSV field templates de-globaled into `fieldTemplates()`, `strconv.Atoi`
  replacing hand-rolled digit math, makezero, omitzero on `TranscriptTurn.At`)
  plus curated policy in `.golangci.yml` — tagliatelle `json: snake`
  (project wire convention, keeps the linter load-bearing), wrapcheck globs
  for own module + the cqrs stack (errorfamily errors are terminal
  classifications; wrapping buries the family code), fleet-precedent
  test-file exclusions (paralleltest/lll/unparam/varnamelen, per go-output),
  and `^cmd/` forbidigo (the CLI's product IS terminal output; internal/
  stays print-free).
- **cqrs-lint: 0 errors**, 5 warnings all documented decisions in AGENTS.md
  (two new: the E005 false-positive class now also covers `EnrichCallCmd`;
  the ignored `Close()` in `OpenSystem`'s cleanup is deliberate for one-shot
  CLI processes).
- Build + full test suite green and STABLE (5 consecutive clean runs after
  the drain fix below).

### internal/cli package (C7 core)

- `ResolveKey`, `ParseSince`, `ParseDirection` — tested flag parsing.
- `Ingest` — the shared dispatch loop: Recorded/Corrected/Rejected stats
  with per-observation reject reasons. Tested end-to-end: first pass
  records, identical replay is a no-op, changed snapshot classifies
  Corrected, invalid observations reject.
- `FetchTelnyxObservations` — the fetch/map/correlate loop shared by the
  CLI and the future serve poller.
- `OpenSystem` + **`DrainProjections`** — see the hard lesson below.
- `cmd/pbx-lens/main.go` rewritten thin: cyclop 15 → ~7, all mnd → consts,
  errcheck clean. `ingest-webhook`, `ingest-agent`, `ingest-cdr -ai-dest`
  all wired through the shared loop.

### The hard lesson (~1h paid, now AGENTS.md gotchas)

Tests using `cli.Ingest` + drains "never converged" in ways that looked
like engine flakiness (phantom theories chased: negative caching, Get-miss
poisoning, dispatch-count dependence, test-harness differences, machine
load average 22). Root causes, both now documented:

1. **Projection record keys carry the `StreamMarker:` prefix.**
   `cmd.StreamID().String()` is `"StreamMarker:telnyx-conversation:x"` and
   THAT is the call_views record key. My tests polled unprefixed keys.
2. **The projection host is a one-shot batch drainer.** Start refuses
   `already_started`; Stop CANCELS the run ctx (can kill a drain before it
   processes anything); WorkerIdle right after Start means "not drained
   yet", not done. The original fire-and-forget `go host.Start(ctx)` was
   genuinely racy for CLI one-shots. Correct synchronous drain = idempotent
   Stop → Start → poll `host.Status()` until every worker is WorkerStopped
   — that is `cli.DrainProjections`, now used by every command. Memory
   engine NEVER projects without a host drain (the sqlite e2e test only
   passed by coincidence of timing).

Process lesson recorded: when a lookup "never finds", PRINT the actual key
and the actually-stored keys BEFORE theorizing about engines and caches.

### Third source family + cross-source join (B1, B4, B5) — all e2e-verified

- **B1 webhook ingest**: `internal/ingest/webhook` parses the receiver's
  `inbound.jsonl`, AGGREGATES per `call_control_id` (order-independent →
  stable ContentHash), maps direction (incoming/outgoing → inbound/outbound),
  duration from hangup−initiated truncated to whole seconds; garbage lines
  are Skipped, not fatal (log-only source tolerance). 4 tests. The
  `ingest-webhook` CLI e2e: two runs against a sqlite DB — second run
  replays idempotently.
- **B4 agent transcripts**: `internal/ingest/agent` parses
  `{uuid}.jsonl` (start/turn/end lines; caller/agent → user/assistant;
  per-turn At stays zero because the writer stamps none),
  `cli.FetchAgentObservations` joins the freeswitch-cdr view by call uuid
  (parties/direction/duration overlay; transcripts stand alone without a
  join — validation relaxes for agent-transcript kind). 4 tests. The e2e
  run is the doctrine proof: agent ingest → CDR ingest → agent RE-ingest
  prints "1 corrected" — the join filled the numbers and superseded the
  first snapshot via `call.corrected`.
- **B5 dialplan badge**: CDR rows badge IsAI when destination_number hits
  the AI-destination set (`-ai-dest`, default "9100" from
  nix-international-telephony's agent.extension); custom sets tested.

### Honest-demo hygiene (A3)

README: `go install @latest` line dropped (no tag exists), ingest-cdr
output format corrected, the changed-row doctrine line fixed (supersession,
not conflict — it had drifted from the domain redesign). CHANGELOG: entries
for the lint zero-drive, the cli package + drain pattern, the domain
redesign, the +1555 smoke number. Smoke demo number is now +15555550123.

### Verified-parallel work (not mine, artifact-checked by me)

The parallel session landed: the nix package fix (verified:
`result/bin/pbx-lens version` → `pbx-lens 0.1.0`), vendorHash repair app,
`.github/workflows/ci.yml` + dependabot ecosystems (push still gated on
owner question), cqrs-lint in the devShell.

## In progress when interrupted (b) — F3/F4 frontend adoption ~70%

- Deps landed: go-appkit v0.8.0, templ-components v1.21.0, a-h/templ
  v0.3.1020; go directive → 1.27.1.
- **Discovery: v1.21.0 (latest PUBLISHED) lacks FilterBar, SegmentedControl,
  FilterChips, utils/format, and the errorpage package** — they exist only
  in the unpublished working tree the 01-32 plan researched. Decision: stay
  on published v1.21.0; hand-roll a thin GET filter form + local format
  helpers. No local-replace hackery (it would break hermetic nix builds).
- CSS scaffold in `internal/web/static/` (app.css/custom.css/theme copied;
  the class corpus `templ-components-classes.txt` GENERATED myself — 8821
  deduped candidate tokens from v1.21.0 sources — because the shipped
  corpus file also postdates the release).
- `internal/web` WRITTEN AND COMPILING for the list half:
  `queries.go` (ListCalls via metaengine Scan — direction/AI/date filters,
  started_at desc, keyset `before` cursor, 50-row cap; CallRow view model;
  CallDetail), `page.templ` (shell), `calls.templ` (list page, filter form,
  typed Table rows with clickable-row Hrefs, stale banner, empty state,
  local format helpers), `server.go` (routes `/`, `/calls/{kind}/{id}`,
  `/app.css`; parseFilters with 400-class rejections; staleness check).
- `detail.templ` JUST written and KNOWN-BROKEN in 4 spots: it calls a
  nonexistent `view.Href()`; `DefinitionList` takes `Items []DefinitionItem`
  (not children); Split props unverified; `trimZero` undefined.

## Not started (c) — the remaining honest backlog

Serve command wiring (appkit Service, flags, graceful drain), flake devShell
templ+tailwindcss_4, .gitignore `*_templ.go` + `static/styles.css`, CSS
build + BuildFlow templ-generate/tailwind-build verification, web render
tests, recording proxy (B2), transcript search (B3), poller + /api/freshness
(C2), insights + cost probes (C5/C6), backup command (C4), systemd/NixOS
draft (C3), htpasswd auth (D1), retention/PII purge (D2), dashboard (D3),
export API (D5), C009 upstream (D6), §30 strike + daemon-diff review (C9),
webphone deep link (D4), final docs sync, production DB re-ingest (hash
v2), full-mode gate.

## What went wrong or could be better (d)

1. The StreamMarker debugging detour (~1h): I theorized before
   instrumenting. The debug logging that solved it should have been the
   FIRST step. (Fixed the process: print keys before theories.)
2. The auto-commit daemon ate TWO of my edits mid-session (an AGENTS.md
   gotcha insertion; a CHANGELOG bullet that vanished in a failed
   multiedit) — both recovered because I rg-verified after writing.
   Recovery habit now standard.
3. I wrote templ templates against the SKILL DOC's API (which documents the
   library's working tree) before checking the pinned v1.21.0 — burned
   cycles on FilterBar/SegmentedControl/utils-format/errorpage mismatches.
   Lesson: verify library APIs against the PINNED version's source first.
4. `?to=YYYY-MM-DD` parses midnight START-of-day, so the To filter excludes
   that day's calls. Known wart, unfixed, untested.
5. `staleness()` runs a second full ListCalls (duplicate scan) — minor
   inefficiency, noted.
6. Dependency-policy edit to AGENTS.md (guard #7: same commit as go.mod)
   is OVERDUE — the go.mod change is already daemon-committed without it.

## Process improvements for next session (e)

1. Instrument before theorizing (print the actual key/ids/values).
2. Verify pinned-version APIs from the module cache before writing code
   against doc tables.
3. rg-verify every doc edit immediately after (daemon races).
4. Add the drain/await pattern as a shared test helper instead of
   re-deriving it per test file.
5. Batch the remaining templ fixes (detail.templ has 4 known issues) into
   one generate-build-test cycle.

## Owner questions (cannot self-answer)

1. ~~**O1**: GitHub visibility public/private + may I create the remote and~~ done — answered: public at github.com/LarsArtmann/pbx-lens (2026-10-11)
   ~~push? (Gates CI activation, `go install @latest` someday.)~~
2. ~~**Doctrine ratification**: `call.corrected` supersession replaced the~~ done — supersession doctrine documented (AGENTS.md gotchas; total fold intact)
   ~~documented "different hash on live stream = conflict". Keep as-is, or~~
   ~~keep conflicts for freeswitch-cdr kind only (immutable CDRs)?~~
3. ~~**§30 strike confirmed?** Cross-repo edit marking pbx-lens rows in~~ **Won't implement — owner-gated cross-repo edit; pbx-lens TODO_LIST is the canonical backlog.**
   ~~`~/projects/pbx-artmann/TODO_LIST.md` as moved-to (the blanket mandate~~
   ~~arguably covers it; explicit yes preferred).~~

## Next 40 (highest-value first)

1. ~~Fix detail.templ (Href → static, DefinitionList Items, Split props,~~ done (docs-health pass 2026-10-12)
   ~~trimZero) → generate → build.~~
2. ~~Web render tests: list empty/rows, bad filter → 400, detail found/404.~~ done (docs-health pass 2026-10-12)
3. ~~`serve` command: appkit Service{Addr}, web.Register, graceful drain.~~ done (docs-health pass 2026-10-12)
4. ~~flake devShell: templ + tailwindcss_4; `.gitignore` `*_templ.go` +~~ done (docs-health pass 2026-10-12)
   ~~`static/styles.css`.~~
5. ~~Build CSS; verify BuildFlow templ-generate + tailwind-build trigger.~~ done (docs-health pass 2026-10-12)
6. AGENTS.md dependency-policy edit + templ-components adoption table **→ ROADMAP theme 1 (templ-components upgrade watch)**
   (OVERDUE — guard #7).
7. ~~e2e httptest: seed 3-source fixture → list → filter → detail.~~ done (docs-health pass 2026-10-12)
8. ~~B2 recording proxy (signed-URL refetch server-side + local~~ done (docs-health pass 2026-10-12)
   ~~`{uuid}_{dest}.wav` Range); key-never-in-browser audit test.~~
9. ~~B3 search route + FilterInput (decision locked: in-memory index).~~ done (docs-health pass 2026-10-12)
10. ~~C2 serve poller (ticker → Fetch + Ingest + Drain) + /api/freshness.~~ done (docs-health pass 2026-10-12)
11. C5 insights live probe (GET-only) → Enrichment wiring. **→ TODO_LIST blocked row (needs a real API key)**
12. ~~C6 Detail Records cost probe → Enrichment.CostMicros.~~ done (docs-health pass 2026-10-12)
13. ~~C4 backup command (VACUUM INTO) + staging note.~~ done (docs-health pass 2026-10-12)
14. ~~C3 systemd unit + NixOS module draft + sibling handoff.~~ done (docs-health pass 2026-10-12)
15. ~~D1 htpasswd auth ({PLAIN} shared file; x/crypto dep + policy edit).~~ done (docs-health pass 2026-10-12)
16. ~~D2 retention/PII purge job.~~ done (docs-health pass 2026-10-12)
17. ~~D3 dashboard page (MOS honest-limitation note).~~ done (docs-health pass 2026-10-12)
18. ~~D5 export API (JSONL/CSV).~~ done (docs-health pass 2026-10-12)
19. ~~D6 C009 verify-at-source + upstream issue.~~ done (docs-health pass 2026-10-12)
20. ~~C9 §30 strike + daemon-diff review.~~ **Won't implement — owner-gated cross-repo edit; pbx-lens TODO_LIST is the canonical backlog.**
21. ~~Fix `?to=` end-of-day semantics + test.~~ done (docs-health pass 2026-10-12)
22. ~~staleness(): reuse the fetched page (drop second scan).~~ done (docs-health pass 2026-10-12)
23. ~~mnd audit on new web code (e.g. `1_000_000` in costText).~~ done (docs-health pass 2026-10-12)
24. ~~cqrs-lint + golangci pass over the new packages.~~ done (docs-health pass 2026-10-12)
25. ~~httptest e2e for the ingest-telnyx loop (C7/m53 residual).~~ done (docs-health pass 2026-10-12)
26. ~~/api/freshness JSON + PolledRegion banner wiring.~~ done (docs-health pass 2026-10-12)
27. serve `-stale-after` flag. **→ ROADMAP theme 2**
28. ListNote Total once a count query exists. **→ ROADMAP theme 1**
29. Consider upstream templ-components issue: publish v1.22 with **→ ROADMAP theme 1 (templ-components upgrade watch)**
    FilterBar/format/errorpage (fleet consumers hit the same gap).
30. ~~FEATURES.md/README UI rows when the list page ships.~~ done (docs-health pass 2026-10-12)
31. ~~DOMAIN_LANGUAGE: TranscriptTurn, Enrichment, excerpt semantics,~~ done (docs-health pass 2026-10-12)
    ~~corrected doctrine.~~
32. ~~TODO_LIST row flips for everything in section (a).~~ done (docs-health pass 2026-10-12)
33. Production DB re-ingest runbook (hash v2) — needs access-path answer. **→ TODO_LIST blocked row (schema v2 gate)**
34. ~~Full-mode buildflow gate.~~ done (docs-health pass 2026-10-12)
35. ~~Verify parallel session's ci.yml matches fleet pattern (review pass).~~ done (docs-health pass 2026-10-12)
36. ~~Detail-page cost/insight absent-state tests.~~ done (docs-health pass 2026-10-12)
37. ~~Filter-form a11y pass (labels, noscript Apply affordance).~~ done (docs-health pass 2026-10-12)
38. ~~Recording proxy degradation tests (upstream down → honest error).~~ done (docs-health pass 2026-10-12)
39. ~~Consider Pagination vs LoadMore once count exists.~~ done (docs-health pass 2026-10-12)
40. ~~Status-report refresh at the next interruption point.~~ done (docs-health pass 2026-10-12)

_ARte in Aeternum — the index is honest, the gates are green, and the
remaining work is enumerated._
