> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

A handler-initiated `Flush` switches the response to untagged streaming with no hook and no log — `OnBufferOverflow` explicitly excludes it, so a consumer whose handlers flush (templ, SSE, chunked writes) silently loses every 304 and finds out only by diffing response headers. Ask: an `OnFlushed` hook (or debug log) on the flush/hijack path.

## Problem

`server/etag.go` at v0.6.1 (re-verified at master `039f2f1`, same shape). The one observability hook the middleware has is scoped to overflow only:

```go
// OnBufferOverflow is called when a response body exceeds MaxBufferSize
// and the writer switches to streaming without an ETag. The argument is
// the configured limit that was exceeded. It fires at most once per
// response; a handler-initiated Flush does not trigger it. If nil, no
// action is taken.
OnBufferOverflow func(int) // etag.go:83-88
```

The flush/hijack path takes the same "streaming without an ETag" exit and reports nothing:

```go
func (w *etagWriter) Flush() { // etag.go:382
	if w.flushed {
		w.responseWrapper.Flush()

		return
	}

	w.markFlushed()
```

`Hijack` does the same (`etag.go:408`), and `markFlushed` (`etag.go:416`) only sets the unexported `flushed` flag.

## Real consumer impact

Serving templ pages through `etag.New(etag.DefaultETagConfig())`: templ's sectioned writer calls `Flush` between sections, so **every HTML response streamed untagged** — the middleware was configured, tests asserted only the static-asset path, and the first signal was a live header probe showing no `ETag` on pages while `/app.css` had one. Nothing in the library would have told me; I read `etag.go` to learn the mode even existed.

## Suggested fix

Either an `OnFlushed func()` hook fired from `markFlushed` on the flush/hijack path (overflow keeps its own, richer hook — no double-fire), or a debug-level log when the flag flips. The behavior itself is right — buffering past a flush would corrupt streaming responses — it is just invisible today, and invisible degradation is the one failure mode a conditional-GET middleware cannot afford.

Checked open and closed issues for "flush" — no existing report.

💘 Generated with Crush
