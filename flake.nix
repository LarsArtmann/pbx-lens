{
  description = "pbx-lens — unified call-history index for the pbx.artmann.tech deployment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    go-nix-helpers = {
      url = "github:LarsArtmann/go-nix-helpers";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ inputs.go-nix-helpers.flakeModules.go-standard ];

      go-standard = {
        pname = "pbx-lens";
        description = "Unified call-history index: read-only ingest of FreeSWITCH CDRs, Telnyx conversations, recordings and the webhook log into one sqlite read model";
        goPkgAttr = "go_1_27";
        goExperiment = "jsonv2";
        vendorHash = import ./vendorHash.nix;
        extraMeta = {
          homepage = "https://github.com/larsartmann/pbx-lens";
          mainProgram = "pbx-lens";
          platforms = inputs.nixpkgs.lib.platforms.unix;
        };
      };
    };
}
