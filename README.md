# pbx-lens

One sqlite database and one list page answering, for every call the PBX ever
saw: who, when, how long, AI or human, and where the transcript lives.

pbx-lens is READ-ONLY over every source: it parses FreeSWITCH CDRs and JSONL
logs and calls Telnyx read APIs. It never writes back to the PBX runtime and
never mutates receiver logs. Its own database is disposable: every row is
re-derivable from the sources, so it can be deleted and rebuilt at any time.

Built on [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) (system
composition root + metaengine sqlite engine).

## Status

Scaffold (T1+T2): domain model, composition root, sqlite end-to-end smoke.

```console
go run ./cmd/pbx-lens smoke        # in-memory pipeline, prints one CallView
go test ./...
```
