> **FILED as [templ-components#55](https://github.com/LarsArtmann/templ-components/issues/55) (2026-10-15, plan M8/F28 + M10). Kept as the historical write-up; upstream state lives in `docs/upstream-filings.md` (re-verified OPEN 2026-10-17).**

> [!NOTE]
> This filing was drafted by GLM via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

**TL;DR:** `utils.EnsureID` mints a fresh crypto/rand id on every render when the caller leaves `ID` empty, so any component that renders a `Nav` gets a byte-different body per request and ETag/`Cache-Control: no-cache` revalidation can never 304 — every navigation and htmx poll pays a full 200. Ask: make component ids deterministic by default (caller-supplied or content-derived), or document the caching consequence so consumers know they must pin every id.

## Problem

Consumer shell renders `navigation.SimpleNav` without `BaseProps.ID` (the zero value feels safe — "no id needed"). v1.21.0 `navigation/nav.templ.go:80` then calls:

```go
menuID := utils.EnsureID("mobile-menu", props.ID)
```

and `utils/validate.go:32-45` fills the empty case from `crypto/rand`:

```go
var b [8]byte
rand.Read(b[:])
return fmt.Sprintf("tc-%s-%x", prefix, b[:])
```

One id, but it lands three times in the markup (`aria-controls`, `data-mobile-menu-toggle`, `id`/`data-mobile-menu-id` on the menu div), so the whole page body differs on every render.

Measured against a real consumer (`pbx-lens`, go-etag `DefaultETagConfig` + `Cache-Control: no-cache` on all HTML): three consecutive `GET /` responses carried three different ETags — `"543e3db0f6633ac3"`, `"d962559d065392f7"`, `"40a2431927b89bef"` — and `If-None-Match` returned 200 every time. After the consumer pinned `BaseProps.ID`, the same probe returned 304 across `/`, `/app.css`, and polled fragments. The htmx poll lanes (30 s/60 s) were silently re-downloading every pass.

## Root cause

`EnsureID` treats "no caller id" as "invent one", and the invented one is render-entropy, not content. Any body-fragmenting cache (ETag, CDN, goldens) sees a new document each time. Goldens have to normalize the id away (`tc-mobile-menu-NORMALIZED`), which hides the effect from page tests.

## Suggested direction

Either:

- derive the fallback deterministically (prefix + content hash / caller-supplied seed), or
- keep randomness but document loudly on `Nav`/`BaseProps.ID` that leaving it empty defeats HTTP caching for every page the component renders on.

Deterministic-by-default would have saved the consumer debugging: the failure has no error surface, it just quietly multiplies traffic.

## To verify

- [ ] `EnsureID("x", "")` twice in one process → different values (current behavior, `utils/validate.go:37-45`)
- [ ] consumer page rendered twice with empty `BaseProps.ID` → different bytes; with a pinned id → identical bytes + ETag 304 on re-request

💘 Generated with Crush
