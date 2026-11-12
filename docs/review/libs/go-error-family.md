# go-error-family — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pin:** `github.com/larsartmann/go-error-family v0.12.0` (go.mod:22) · **Status:** USED
**Adoption score:** **82/100 (high)** — 9 of 11 applicable capabilities used appropriately (fully = 1, partial = 0.5 per the deep-dive rubric); the four boundary misses (exit codes, HTTP mapping, structured logging, test assertions) all closed 2026-11-04. Registries/retry/tone surface graded Not Applicable with reasons below.
> **Refreshed 2026-11-12:** pin `go-error-family v0.12.0` unchanged and still the latest tag (tags fetched).
> Census spot-checks pass at today's HEAD: `HandleError` (cmd/pbx-lens/main.go:103), `writeIndexError` (internal/web/errors.go:25), `errorfamilytest` sites intact (poller_test.go:481, depth_trio_test.go). **Score stands at 82/100.**
> **2026-11-11 identity adoption expanded the census** (score-neutral): `internal/web/userauth.go` is a new import file with 10 `errorfamily.NewRejection` construction sites (the cqrs-htmx identity bootstrap classifies its config/store failures at origin — exactly the audit's recommended posture), taking the importing-file count from 23 to 29 (22 prod, 7 test). No capability row moved: construction-at-origin was already Fully Leveraged; the breadth gaps (single wrap variant, fluent JSON/Summary, `Classify` in tests only) are unchanged.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## Verdict

pbx-lens uses go-error-family well **at the construction site** and almost not at all **at the boundaries**. Errors are classified at origin with a coherent `lens.*` code taxonomy across 23 files, the protocol is consumed cross-library (cqrs-lite version-conflict → `Classify` = Conflict, pinned by tests), and the fail-open Transient semantics are explicitly reasoned in test comments. But the library's three boundary conventions — CLI exit codes + What/Why/Fix messaging, HTTP status mapping, structured logging — are all hand-rolled instead, and the test-assertion sub-package `errorfamilytest` is unused. The gaps are cheap to close and would delete real duplication.

## Capability inventory vs use

Ground truth: `go doc -all github.com/larsartmann/go-error-family` at the v0.12.0 module-cache pin (≈141 exported identifiers, 38 capability rows).

| Capability                                                                                               | Used?                                                                             | Verdict                                                      |
| -------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Family constructors (`NewRejection/NewTransient/NewCorruption`, generic `New`)                           | ✅ ~60 sites                                                                      | Fully leveraged                                              |
| Family consts (`Rejection/Conflict/Transient/…`)                                                         | ✅ `Rejection`,`Transient`,`Conflict`                                             | Partially used (`Corruption` reachable only via constructor) |
| `Wrap` family (12 variants)                                                                              | ✅ `WrapInfrastructure` only (`internal/cli/catalog.go:33,49`)                    | Partially used                                               |
| `Code(err)` / `Coded` machine codes                                                                      | ✅ `internal/web/export.go:157,162`                                               | Fully leveraged                                              |
| `Classify` + protocol (cross-library)                                                                    | ✅ tests: `internal/cli/poller_test.go:478`, `internal/app/depth_trio_test.go:83` | Partially used                                               |
| `ExitCode` / `HandleError*` / message templates (CLI boundary)                                           | ❌                                                                                | **Missed opportunity**                                       |
| `HTTPStatus(err)` / `HTTPHandler` (web boundary)                                                         | ❌                                                                                | **Missed opportunity**                                       |
| `LogError` / `LogErrorContext` (structured logging)                                                      | ❌                                                                                | **Missed opportunity**                                       |
| `Error` fluent API (`WithContext`, `JSON`, `Summary`)                                                    | ❌ (accessors only)                                                               | Partially used                                               |
| `errorfamilytest` assertions                                                                             | ❌                                                                                | **Missed opportunity**                                       |
| Registries/classifiers, `IsRetryable`, `RetryPolicy`, tone/audience/parse surface, `WrapOnce`, `Compose` | ❌                                                                                | Not applicable (see reasons)                                 |

N/A reasons (claim, not escape hatch): registries stay deliberately unregistered — "carriers stay unclassified until a consumer needs the family" is a documented, test-commented decision (`internal/cli/poller_test.go:458-469`); `IsRetryable`/`RetryPolicy` — the poller re-ticks on a fixed schedule, there is no retry loop; tone/audience/presentation — errors surface as CLI one-liners and HTML pages, not tone-driven text; `WrapOnce` — classification happens once at origin; `Compose` — no `errors.Join` sites.

## Usage census

29 importing files (22 prod, 7 test — 23/18/5 at the audit; the 2026-11-11 identity adoption added `internal/web/userauth.go` with 10 bootstrap-time `NewRejection` sites). Distinct identifiers used: 14 of ≈141 exported (10%); capability-level: 11 of 38 (~29%). Construction at origin in domain (`internal/calls/*.go`), ingest parsers (`internal/ingest/cdr`, `internal/ingest/telnyx`), CLI (`internal/cli/*`), and web (`internal/web/*` — now including the identity bootstrap). Consumption is confined to export error markers (`internal/web/export.go:153-169`) and 3 test assertions.

## Top findings

1. **CLI exit codes + messaging hand-rolled.** `cmd/pbx-lens/main.go:89-92` — `fatal()` always `os.Exit(1)` and prints the raw error. `cmd/pbx-lens/ops.go:99` and `internal/cli/system.go:58` construct `Transient` errors (`lens.cli.projection_timeout`) that the library maps to exit **75** (EX_TEMPFAIL) via `ExitCode(err)`; `HandleError(err)` would add What/Why/Fix for free. _Impact:_ operators can script around transient failures; ~10 lines of messaging logic deleted.
2. **Web layer duplicates `Family.HTTPStatus`/`HTTPHandler` wholesale.** Inline mappings: 400 for Rejection (`internal/web/server.go:182`, `internal/web/export.go:176,192`, `internal/web/search.go:67`), 500 (`server.go:193,354,500`, `internal/web/segment_page.go:31`, `internal/web/freshness.go:31,43`), 503 (`server.go:622`, `internal/web/recording.go:111,235`), 404 (`server.go:507,555`). All write `err.Error()` inline — a leak risk the library's `writeHTTPError` deliberately avoids (it gates the message on classification). `errorfamily.HTTPHandler(fn)` replaces the pattern across ~14 sites.
3. **Infrastructure-shaped errors left unclassified.** `internal/web/search.go:50-55` ("needs a sqlite index") and `internal/web/server.go:618-626` ("no ops counters wired") are bare `http.Error` 503 strings with no family; `server.go:632` (json.Marshal failure) is unclassified Orchestration. `WrapInfrastructure` + `HTTPStatus` would classify and map them in one step (contrast: `internal/cli/catalog.go:33` already does it right).
4. **Export error marker hand-rolls `Error.JSON()`.** `internal/web/export.go:153-169` invents `exportError{ExportIncomplete, Code, Message}`; `errorfamily.WithContext("call_id", …)` + `JSON()` makes the marker canonical while keeping `Code(err)` consumers working.
5. **Test assertions duplicate `errorfamilytest`.** `internal/ingest/telnyx/telnyx_test.go:238-249`, `internal/cli/poller_test.go:478`, `internal/app/depth_trio_test.go:83-85` re-implement `AssertFamily`/`AssertCode`. The sub-package rides the same module — promotion adds no new module.

## Version currency

Pin v0.12.0 **is the latest release** (checkout `/home/lars/projects/go-error-family`: newest root tag v0.12.0; `[Unreleased]` CHANGELOG empty; last feature commit `a5ce32c` "nil-receiver-safe Unwrap" is in v0.12.0). Library `go 1.27`, zero third-party deps — riding it is free.

## Risks

- **Pre-1.0 API churn** (v0.x; upstream modularization proposal exists) — keep the pin exact.
- **Fail-open defaults**: `Classify`/`ParseFamily` fall back to Transient — safe today (documented), a footgun once `ExitCode`/`IsRetryable` adoption lands (everything unknown exits 75 / looks retryable). Adopt boundary mapping and keep the poller's explicit reasoning comments.
- `(*Error).Is` matches on code AND family — never use library errors as sentinels.

## Recommendations (ranked)

1. Route the CLI through `errorfamily.HandleError`/`ExitCode` (findings 1; easiest, highest operator value).
2. Swap the ~14 inline web status mappings for `HTTPHandler`/`HTTPStatus` (finding 2) — also closes the message-leak surface.
3. Classify the three unclassified Infrastructure sites (finding 3).
4. Adopt `errorfamilytest` in the 3 assertion sites (finding 5) — zero new modules.
5. Canonicalize the export marker with `WithContext`/`JSON` (finding 4).

## How to verify

```console
rg -c "errorfamily\." --type go -g '!*_templ.go' internal cmd   # 23 files
go doc github.com/larsartmann/go-error-family ExitCode           # the missed CLI mapping
rg -nF "os.Exit(1)" cmd/pbx-lens/main.go                         # finding 1
rg -n "http.Error" internal/web                                  # finding 2/3 sites
git -C /home/lars/projects/go-error-family tag --sort=-v:refname | head -3
```

## Delta appendix — 2026-11-04 closeout (plan T04–T08)

All five recommendations executed the day of the audit:

1. **CLI boundary (T04).** `fatalCode` in `cmd/pbx-lens/main.go` splits the
   boundary: classified errors go through `errorfamily.HandleError` (so the
   projection timeout now exits 75 EX_TEMPFAIL with What/Why/Fix), plain
   errors keep the legacy one-liner and exit 1 — deliberately NOT routed
   through `ExitCode`, because `Classify` fail-opens unclassified errors to
   Transient and a plain bug must never advertise "retry later". Pinned by
   `TestFatalCodeFamilyMapping` (6 cases).
2. **Web boundary (T05/T06).** One family-aware writer,
   `internal/web/errors.go writeIndexError`, replaced the hand mappings:
   classified errors take `Family.HTTPStatus()` and the body carries ONLY
   crafted safe text (never `err.Error()`) in either the text or the
   `{error, family, code}` JSON shape; unclassified keep the caller's
   fallback status. The three bare 503s were classified at source
   (`NewInfrastructure` / `WrapInfrastructure`). One deliberate exception:
   the recording proxy's metadata failure keeps 502, because proxy
   semantics beat the family's status — it still logs through
   `LogErrorContext`.
3. **Structured logging (T08).** A `logFuncHandler` slog adapter feeds
   `LogErrorContext`, so family/code/retryable/exit_code/context.* ride
   every index-read failure; the export stream uses it too.
4. **`errorfamilytest` adopted (T07)** in all three assertion sites —
   `poller_test.go`, `depth_trio_test.go`, `telnyx_test.go` (the last keeps
   an explicit contains-check, with a comment, because a fail-open
   `Classify` makes `AssertFamily` pass for unclassified errors).
5. **Marker canonicalized (T08).** `exportError` gained a `Context` map fed
   by `errorfamily.Contextual` (`AsType`), so a truncated download says
   WHERE it stopped; `Code(err)` consumers stay unchanged.

Score movement: **45/100 → 82/100** (4 of the 5 ❌ rows became ✅:
`HandleError`, `HTTPStatus`, `LogErrorContext`, `errorfamilytest`). The
residual gaps that keep it under 100 are breadth, not convention: only
`WrapInfrastructure` of the 12 wrap variants is used, the fluent
`JSON`/`Summary` surface is still hand-rolled (the marker keeps its own
struct), and `Classify` appears only in tests. `WithContext` is now
CONSUMED (`Contextual` read at the export marker) but never SET by
production code — attaching `call_id` at the source is the natural next
step, and the largest remaining win.
