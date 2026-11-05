# go-codec — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-codec` v0.3.3 (pin == latest; in pbx-lens' graph **INDIRECT only**, go.mod:79) · **Status:** NOT USED DIRECTLY
**Verdict:** **REJECT direct adoption — keep indirect.** It is already exercised where it belongs: inside go-cqrs-lite's storage/event/signing stack.
**Fleet report:** [index.html](index.html)

## What the library offers

- `Codec` abstraction over CBOR/JSON/Raw; `BufferEncoder` hot path; `ObservableCodec`; `AutoDetect`; `TranscodeToJSON`; `WrapEncode`/`Size`; `DeterministicCodec` for signing; CBOR `toarray`/`keyasint` struct tags.

## Why pbx-lens should not import it directly

1. **It is the event-payload serialization contract of the cqrs-lite stack.** `go mod why` traces it: `cmd/pbx-lens → go-cqrs-lite/metaengine/v4 → go-codec`. Every event pbx-lens writes is already CBOR-encoded through it (`event/v4` typed payloads) — pbx-lens benefits from it without touching it, exactly as designed.
2. **pbx-lens' hand-rolled decode layer solves a different problem.** `internal/wire/flextime.go:17-45` (FlexTime + shared decode helpers) handles _external_ JSONL dialects — Telnyx timestamps in ns/µs/offset-less variants — a multi-layout tolerant-decode problem go-codec has no API for. Replacing wire with go-codec would be a worse fit, not a dedup.
3. Direct import would add a version-pinning obligation (hermetic nix, vendorHash) for zero new capability.

## When this verdict flips

If a hand-rolled binary encoding need appears outside event storage (e.g. a custom cache file format wanting CBOR), use it then — it's the house codec.

## How to verify

```console
go mod why github.com/larsartmann/go-codec   # → metaengine/v4 → go-codec
rg -n "go-codec" internal cmd --type go      # zero direct imports
rg -n "FlexTime" internal/wire/flextime.go | head -3
```
