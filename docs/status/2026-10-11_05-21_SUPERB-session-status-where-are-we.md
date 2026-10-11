# Session status — where are we right now?

**Date/time:** 2026-10-11 05:21 CEST
**Author:** the working session (auto-daemon commits throughout)
**Supersedes:** `docs/status/2026-10-11_04-59_SUPERB-session-status-what-did-you-forget.md`
**Plans in force:** `docs/planning/2026-10-11_00-09_SUPERB-pareto-remaining-work.md` (backend A–D) and `docs/planning/2026-10-11_01-32_SUPERB-pbx-lens-frontend-templ-appkit.md` (frontend F1–F25, supersedes the stdlib-html guardrail).

---

## 1. What got DONE since the last report (04:59 → 05:21)

Frontend tier + first two B-tier features, all test-verified. 9/9 packages green (`go test ./... -count=1`), `nix build` green with full CSS embedded.

| What                                                           | Where                                                         | Proof                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| -------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **detail.templ un-broken** (4 known issues fixed)              | `internal/web/detail.templ`                                   | DefinitionList takes typed `Items []DefinitionItem{Term, Detail}` (children were the parse bug); `display.Split` does NOT exist in v1.21.0 → plain grid div; `costText` via `strconv.FormatFloat` + `microsPerUnit` const                                                                                                                                                                                                                                                                                           |
| **render helper + web tests**                                  | `internal/web/render.go`, `internal/web/server_test.go`       | 7 tests: list empty/rows, filter 400s (direction/ai/date), direction filter, detail facts+transcript, detail 404, styles-unbuilt 404. 404 path sets Content-Type BEFORE WriteHeader                                                                                                                                                                                                                                                                                                                                 |
| **serve verb (F5/M18)**                                        | `cmd/pbx-lens/main.go` (`serve`)                              | appkit `NewService(ServiceConfig{Addr})` + `svc.Run(ctx)` (appkit owns SIGINT/SIGTERM + graceful drain). Setup ctx bounded 5m; serve lifetime = `context.Background()` (a timeout there would kill the server). One startup drain catches unprojected events. **E2e verified** on sqlite: list/detail render, graceful shutdown, DB intact                                                                                                                                                                          |
| **go-appkit dependency re-landed**                             | `go.mod`                                                      | `go mod tidy` had dropped it (nothing imported it yet); `go get github.com/larsartmann/go-appkit@v0.8.0` + tidy                                                                                                                                                                                                                                                                                                                                                                                                     |
| **Nix UI codegen pipeline**                                    | `flake.nix`, `vendorHash.nix`                                 | `enableTempl = true` + `templGenerationPolicy = "preBuild"` + `extraBuildAttrs` preBuild running `templ generate ./internal/web/` + `tailwindcss -i app.css -o styles.css --minify`. `nix run .#update-vendor-hash` landed `sha256-In32XfpeiCKo2jSIOHgMkZqQInxUTgtABxCUqpIouMI=`; `nix build` green; served `/app.css` carries the full 4.3.3 stylesheet. devShell += `templ`, `tailwindcss_4`                                                                                                                      |
| **Generated files untracked**                                  | `.gitignore`, git index                                       | `*_templ.go` + `internal/web/static/styles.css` ignored; `git rm --cached` the 3 templ files the daemon had committed pre-gitignore                                                                                                                                                                                                                                                                                                                                                                                 |
| **AGENTS.md dependency policy + adoption table (was OVERDUE)** | `AGENTS.md` "Stack decisions"                                 | UI trio (go-appkit v0.8.0 / templ-components v1.21.0 / templ v0.3.1020) with rationale; per-component adoption map (what's used, what's ABSENT in v1.21.0 — FilterBar/SegmentedControl/utils/format/errorpage); flake bullet updated with preBuild codegen                                                                                                                                                                                                                                                          |
| **B2 recording proxy (m24-m25)**                               | `internal/web/recording.go`, `recording_test.go`, serve flags | Route `GET /calls/{kind}/{id}/recording`. Telnyx path: refetch signed WAV URL **server-side**, stream through, Range passthrough (206 verified), `Cache-Control: no-store`, never a redirect. Local path: `http.ServeFile` over `{ref                                                                                                                                                                                                                                                                               |
| **B3 search, package layer DONE**                              | `internal/search/` (new)                                      | FTS5 virtual table `lens_search` in the same sqlite file; `Rebuild` = full recompute (DROP+CREATE+INSERT, one tx) from views; `Query` = MATCH + bm25 rank + `snippet()` with `\x01/\x02` markers parsed into `[]Part{Text, Highlight}`; `MatchExpression` double-quotes every term (FTS5 injection/parse-error proof), rejects control chars + >256 runes. 4 tests on a REAL sqlite temp file: match+highlight, rank (double-mention first), party-number search, no-match, rebuild-is-recompute, expression safety |
| **B3 web layer DONE**                                          | `internal/web/search.go`, `search.templ`, `server.go`         | `GET /search?q=`, page with input, ranked hits, `<mark>` highlighting, deep links. Register refactored to `web.Options{Recordings, Search}` + `Searcher` interface; all call sites updated                                                                                                                                                                                                                                                                                                                          |

## 2. What is CURRENTLY WORKING (verified this session)

- `GOEXPERIMENT=jsonv2 go build ./...` — clean.
- `GOEXPERIMENT=jsonv2 go test ./... -count=1` — **9/9 packages ok** (app, calls, cli, ingest/{agent,cdr,telnyx,webhook}, search, web).
- `nix build && ./result/bin/pbx-lens version` — green, `usage` lists all 6 verbs incl. `serve`.
- E2e serve on real sqlite: ingest-webhook fixture twice (idempotent: "1 recorded" both runs, 0 rejected) → `serve -db` → `/` renders table + staleness banner; `/calls/webhook-log/e2e-1` renders facts + empty states; graceful shutdown leaves the DB intact.
- Recording proxy unit tests incl. Range 206, 503-with-reason, 404s, and the key audit.
- Search package tests incl. FTS5 highlight/rank against real sqlite.

## 3. What is PARTIALLY DONE / In Flight

**B3 search is ~90%: the missing 10% is wiring.**

1. `cli.RebuildSearchIndex(ctx, sys, dsn)` helper NOT YET WRITTEN — reads all views via `reader.Scan(ctx, metaengine.WithLimit(0))` (0 = unlimited, verified in v4.17.0 scan_options.go:94) → `search.Open(dsn)` → `Rebuild`. No-op when dsn == "" (memory).
2. The four ingest verbs in `cmd/pbx-lens/main.go` don't call it yet — call after every `cli.DrainProjections`.
3. `serve` currently passes `web.Options{Recordings: recordings}` only → **`/search?q=x` answers 503 on every deploy right now** (form page renders, query doesn't). Fix: open `search.Open(*db)` when dsn != "", pass as `Options.Search`, defer Close, plus one startup rebuild.

**Lint gate not re-run since the new code landed** — `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding` was 0 findings BEFORE B2/B3; new files (recording.go, search.go, search.templ Go block, serve) are unmeasured. LSP shows warnings but LSP is unreliable here (stale templ diagnostics + it ignores the `^cmd/` forbidigo exclusion; only buildflow counts).

## 4. What is NOT STARTED / Pending (backend plan order)

| Item                         | Notes                                                                                                           |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------- |
| C2 poller + `/api/freshness` | keeps the index fresh; staleness banner already exists (6h threshold)                                           |
| C5/C6 live probes            | Telnyx key at `~/.pbx-prod-secrets/telnyx_api_key`, GET-only                                                    |
| C4 backup                    | sqlite file backup story                                                                                        |
| C3 systemd module            | thin NixOS module in nix-international-telephony (sibling repo)                                                 |
| D1 auth (T24)                | htpasswd on serve                                                                                               |
| D2 purge (T23)               | retention/PII policy display-side first                                                                         |
| D3 dashboard                 | nav link exists, route doesn't                                                                                  |
| D5 export                    | CSV/JSON of the index                                                                                           |
| D6 upstream filing           | templ-components gaps (DefinitionList children? FilterBar backport) — needs verify-before-filing                |
| C9 §30 strike                | blocked on owner Q3                                                                                             |
| D4 final                     | prod re-ingest (hash v2), full-mode gate, docs sync, README/CHANGELOG entries for serve/search                  |
| Warts                        | `?to=YYYY-MM-DD` = start-of-day (excludes that day); `staleness()` duplicate full scan (fold into list handler) |

## 5. Top 3 blockers / challenges

1. **No hard blockers.** The B3 wiring (§3) is mechanical: one helper + 5 call sites.
2. **Owner questions still unanswered** (resurface, don't act): **Q1 (O1)** GitHub public/private + may I create the remote and push (gates CI activation; a parallel session already landed `.github/workflows/ci.yml`)? **Q2** doctrine ratification: `call.corrected` supersession vs documented conflict-rejection — keep as-is or freeswitch-cdr-only conflicts? **Q3** §30 strike in `~/projects/pbx-artmann/TODO_LIST.md` confirmed?
3. **Environment friction** (ongoing, not new): auto-commit daemon rewrites files mid-edit (re-View before every edit; rg-verify after every write); LSP diagnostics stale/unreliable (trust `templ generate` + `go build` + buildflow only); shared machine had a parallel session land CI/devShell commits — check `git log` before editing shared docs.

## 6. Top 5 next tasks by impact (pareto)

1. **Finish B3 wiring** (~20 min): `cli.RebuildSearchIndex` + call in 4 ingest verbs + serve passes `Options.Search` with startup rebuild. This flips `/search` from 503 to working — the single biggest user-visible gap.
2. **Run the full quality gate** (~10 min): `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding`, `/tmp/cqrs-lint .`, then `buildflow --build-mode dev`. Fix findings in new files. Re-run `nix flake check` (flake.nix changed since last check... it passed at 05:1x WITH the codegen config).
3. **C2 poller + /api/freshness** (~60 min): loop `ingest-telnyx` (and CDR/webhook when files change) every N minutes + tiny JSON endpoint; makes the staleness banner honest forever.
4. **D1 auth** (~45 min): single-user htpasswd middleware in front of serve (appkit ExtraMiddlewares), key from file. serve is currently UNAUTHENTICATED on localhost — fine for dev, gate before any exposure.
5. **Fix the two known warts** (~15 min): `?to=` end-of-day semantics (`AddDate(0,0,1).Add(-time.Second)` or FilterLt next-day) + kill the staleness double scan by computing staleness from the unfiltered list page the handler already fetched.

## 7. What I'd TOTALLY FORGET if shutdown now

- **`metaengine.WithLimit(0)` = UNLIMITED** (no SQL LIMIT) — needed for the rebuild read; default is 100. scan_options.go:94 in v4.17.0.
- **modernc.org/sqlite ships FTS5** — verified by the passing search tests; no build tag needed. `search.Open` uses `db.SetMaxOpenConns(1)` (sqlite write safety) — keep.
- **`snippet()` markers are `\x01`/`\x02`** control chars — they can never collide with content and never reach HTML (templ escapes the parts; markers are stripped in `parseSnippet`).
- **The audit test pattern**: fake Telnyx = one httptest server serving BOTH `/v2/recordings/{id}` JSON and the WAV; `telnyx.NewClient(key, baseURL)` second arg overrides the base — that's the seam for all future Telnyx tests.
- **appkit `svc.Run(ctx)` already handles signals** — do NOT wrap serve in signal.NotifyContext yourself, and never give Run a timeout context (I almost shipped a 5-minute server).
- **`git rm --cached` was needed** because the daemon committed `*_templ.go` before the gitignore landed; if the daemon re-adds them, repeat `git rm --cached internal/web/*_templ.go`.
- **templ-components v1.21.0 adoption table now lives in AGENTS.md** — consult it before touching any component (DefinitionList=Items, SelectOption.Selected per-option, NO Split/FilterBar/format/errorpage).
- The e2e serve check pattern: `printf` webhook JSONL (envelope = `{"received_at","remote","body":{"data":{"event_type","occurred_at","payload":{...}}}}` — the OUTER envelope tripped me once) → ingest twice → serve on 8317/8318 → fetch.

## 8. Anything TOTALLY FUCKED UP?

- **One self-inflicted wound, fixed in place:** I wrote `serveTimeout = 5 * time.Minute` and passed that ctx into `svc.Run` — the server would have died after 5 minutes in production. Caught it before committing the behavior; now setup is bounded, serve lifetime is signal-owned background context. Regression risk: none (pattern documented above).
- **A `defer` typo** (`Close()()` missing a brace) in recording.go — compiler caught it immediately.
- **`/search` is wired-but-dead** (503 on query) until §3.1–3.3 land — the ONLY knowingly-shipped incomplete behavior right now.
- No data was lost; no source was written to; index DB stayed disposable (the FTS table is DROP+CREATE rebuilt every ingest — drift is structurally impossible).

## 9. Verification commands (copy-paste)

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./... -count=1
/run/current-system/sw/bin/templ generate ./internal/web/
BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding
/tmp/cqrs-lint .
nix build && ./result/bin/pbx-lens version
printf '%s\n' '{"received_at":"2026-10-10T09:00:00Z","remote":"198.51.100.7","body":{"data":{"event_type":"call.initiated","occurred_at":"2026-10-10T09:00:00Z","payload":{"call_control_id":"e2e-1","direction":"incoming","from":"+17287289311","to":"+15555550123"}}}}' '{"received_at":"2026-10-10T09:00:26Z","remote":"198.51.100.7","body":{"data":{"event_type":"call.hangup","occurred_at":"2026-10-10T09:00:26Z","payload":{"call_control_id":"e2e-1","hangup_cause":"normal_clearing"}}}}' > /tmp/lens-e2e.jsonl
go run ./cmd/pbx-lens ingest-webhook -db /tmp/lens-e2e.db /tmp/lens-e2e.jsonl   # run twice, second = idempotent
go run ./cmd/pbx-lens serve -db /tmp/lens-e2e.db -addr 127.0.0.1:8317          # then hit /, /calls/webhook-log/e2e-1, /app.css
```
