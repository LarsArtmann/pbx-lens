> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

go-standard's treefmt templ formatter inherits treefmt-nix's `pkgs.go` wrap — hermetic checks break when the module's resolved go is newer than nixpkgs' default go.

## Problem

`modules/go-standard.nix:1297` (master `ad423c8`) enables the templ formatter bare:

```nix
templ.enable = cfg.enableTempl;
```

treefmt-nix's templ program wraps the templ CLI with nixpkgs' default `pkgs.go` (their numtide/treefmt-nix#545). Meanwhile the module resolves the consumer's actual Go itself — `goPkg = pure.goBaseFrom { ... }` at `modules/go-standard.nix:665` — and uses it for builds and devShells, but never feeds it to the formatter.

`templ fmt` shells out to `go` for import processing. When the consumer's go.mod floor is newer than nixpkgs' go, that inner go tries a toolchain download — forbidden inside the hermetic `hermeticTreefmtCheck` sandbox — and the format check dies with a network error, not a formatting error.

## Consumer evidence

pbx-lens (go-standard consumer): go.mod floor `go 1.27.1`, nixpkgs go at our lock `1.26.8`. The sandboxed pre-commit check failed on the toolchain download until the flake force-overrode `settings.formatter.templ.command` with a `writeShellApplication` carrying the module's own go — exactly the wrap this module could own.

## Suggested fix

When `enableTempl` is on, default `settings.formatter.templ.command` to a wrapper that puts the module's resolved `goPkg` (line 665) on PATH — the same resolution the build already trusts. Consumers keep the `mkForce` escape hatch.

Root fix belongs in treefmt-nix (#545, accepting a go package option); this is the consumer-side defense so go-standard templates stay hermetic on either side of that fix.

💘 Generated with Crush
