> [!NOTE]
> **FILED upstream 2026-10-12 (R12): numtide/treefmt-nix#545 (comment posted 2026-10-11 with our repro). Draft historical.**
>
Independent confirmation of this from another go-standard consumer: pbx-lens pins go.mod to `go 1.27.1` while our nixpkgs lock ships go 1.26.8. The sandboxed treefmt check died on `templ fmt` trying a toolchain download (no network in the sandbox), not on any formatting diff. Worked around by force-overriding `settings.formatter.templ.command` with a wrapper that puts our module's go on PATH — the fix proposed here (a `go` package option) would remove that override.
