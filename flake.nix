{
  description = "pbx-lens — unified call-history index for the pbx.artmann.tech deployment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    # go-standard (flakeModules.go-standard) composes treefmt-nix + systems
    # internally — this repo does not declare them as direct inputs.
    go-nix-helpers = {
      url = "github:LarsArtmann/go-nix-helpers";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.go-nix-helpers.flakeModules.go-standard
        inputs.git-hooks.flakeModule
      ];

      go-standard = {
        pname = "pbx-lens";
        description = "Unified call-history index: read-only ingest of FreeSWITCH CDRs, Telnyx conversations, recordings and the webhook log into one sqlite read model";
        goPkgAttr = "go_1_27";
        goExperiment = "jsonv2";
        vendorHash = import ./vendorHash.nix;
        # The binary lives in cmd/pbx-lens. The module default builds the
        # root directory, which holds no Go package — the build then
        # "succeeds" with an empty result and no bin/pbx-lens.
        subPackages = [ "./cmd/pbx-lens" ];
        # UI codegen: *_templ.go and static/styles.css are build artifacts
        # (gitignored) regenerated inside the package build — templ via the
        # module's preBuild policy, tailwind via our preBuild hook below.
        enableTempl = true;
        templGenerationPolicy = "preBuild";
        extraBuildAttrs = pkgs: {
          nativeBuildInputs = [ pkgs.tailwindcss_4 ];
          preBuild = ''
            templ generate ./internal/web/
            tailwindcss -i internal/web/static/app.css -o internal/web/static/styles.css --minify
          '';
        };
        # BuildFlow owns Go formatting in this repo (oxfmt); treefmt Go
        # formatters would be a second source of truth. Nix files stay with
        # treefmt (go-output's pattern: nixfmt + deadnix + statix only).
        enableGofumpt = false;
        enableGoimports = false;
        enableNixfmt = true;
        extraMeta = {
          homepage = "https://github.com/larsartmann/pbx-lens";
          mainProgram = "pbx-lens";
          platforms = inputs.nixpkgs.lib.platforms.unix;
        };
      };

      perSystem =
        {
          config,
          pkgs,
          ...
        }:
        let
          inherit (pkgs) lib;
          go = pkgs.go_1_27;

          # cqrs-lint is its own module (…/cmd/cqrs-lint/v4) inside the
          # go-cqrs-lite monorepo, so the build roots there. Pinned to the
          # cmd/cqrs-lint/v4.16.0 release tag (the first tag carrying the
          # P015 journal-readall rule; 2026-10-12).
          cqrs-lint = pkgs.buildGoModule.override { inherit go; } {
            pname = "cqrs-lint";
            version = "4.16.0";
            src = pkgs.fetchFromGitHub {
              owner = "LarsArtmann";
              repo = "go-cqrs-lite";
              rev = "5a08b0ead55cd923b3b08cd45aba07d425aa193e";
              hash = "sha256-BH3IGPqOie1RLaxATrWtnsbCpXVv3iEgKLHRR5Tlv+Y=";
            };
            sourceRoot = "source/cmd/cqrs-lint";
            # cqrs-lint's own tests run fixture projects through `go`, which
            # fetches modules from the proxy — impossible in the hermetic
            # sandbox. The tool compiles clean; it is exercised for real
            # every run against this repo (see AGENTS.md findings policy).
            doCheck = false;
            # The monorepo root carries a go.work; without this, `go mod
            # vendor` in the modules FOD runs in workspace mode and dies.
            env.GOWORK = "off";
            vendorHash = "sha256-jTwZn/TZJZ22iSwmLRHG/Hc9lE6L9O6fncmKxkiriNE=";
          };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              deadnix.enable = true;
              statix.enable = true;
            };
            # treefmt-nix's templ program wraps templ with pkgs.go — the
            # nixpkgs DEFAULT go (1.26.8 at this lock), older than this
            # repo's go.mod floor (1.27.1). templ fmt shells out to that go
            # for import processing, so the sandboxed pre-commit check
            # tries a forbidden toolchain download and the format fails.
            # Wrap with OUR resolved go instead. Upstream fix pending in
            # go-nix-helpers: the templ formatter should use the module's
            # goPkg when enableTempl.
            settings.formatter.templ.command = lib.mkForce (
              pkgs.writeShellApplication {
                name = "templ";
                runtimeInputs = [
                  go
                  pkgs.templ
                ];
                text = ''exec templ "$@"'';
              }
            );
          };

          pre-commit.settings = {
            hooks = {
              treefmt.enable = true;
            };
          };

          # mkForce composes with the pre-commit hook installation (the
          # module's default shell has no shellHook).
          devShells.default = lib.mkForce (
            pkgs.mkShellNoCC {
              name = "pbx-lens";

              packages =
                builtins.attrValues {
                  inherit go;
                  inherit (pkgs)
                    dprint
                    golangci-lint
                    gopls
                    govulncheck
                    templ
                    tailwindcss_4
                    ;
                }
                ++ [
                  cqrs-lint
                ];

              GOWORK = "off";
              GOTOOLCHAIN = "local";
              GOEXPERIMENT = "jsonv2";

              shellHook = config.pre-commit.shellHook;
            }
          );

          apps.update-vendor-hash = {
            type = "app";
            # nix flake check warns about apps without meta.description on
            # every gate run — the module-generated apps carry one, so the
            # hand-written app must too.
            meta.description = "Repair vendorHash.nix after a go.mod/go.sum change: probe the fixed-output derivation hash with a fake value, write the real one, and verify with a green build";
            program = lib.getExe (
              pkgs.writeShellApplication {
                name = "update-vendor-hash";
                runtimeInputs = with pkgs; [
                  nix
                  git
                  gnused
                ];
                text = ''
                  set -euo pipefail
                  root="$(git rev-parse --show-toplevel)"
                  hash_file="$root/vendorHash.nix"
                  cp "$hash_file" "$hash_file.bak"
                  restore() { mv "$hash_file.bak" "$hash_file"; }
                  trap restore ERR
                  fake="sha256-Z7yp53LzffljsqsLBL9mV+Ah44X2bIhgyR9s5fCbSns="
                  printf '"%s"\n' "$fake" > "$hash_file"
                  out="$(nix build "$root#default" --no-link 2>&1 || true)"
                  got="$(printf '%s' "$out" | grep -oP 'got:\s+\Ksha256-[A-Za-z0-9+/=]+' | head -1 || true)"
                  if [ -z "$got" ]; then
                    echo "ERROR: expected a vendorHash mismatch but none surfaced. Build output:" >&2
                    printf '%s\n' "$out" >&2
                    exit 1
                  fi
                  printf '"%s"\n' "$got" > "$hash_file"
                  nix build "$root#default" --no-link
                  rm -f "$hash_file.bak"
                  trap - ERR
                  echo "vendorHash.nix updated to $got and verified with a green build."
                '';
              }
            );
          };
        };
    };
}
