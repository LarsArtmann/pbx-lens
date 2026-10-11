# pbx-lens NixOS module — DRAFT (T7/F23, M23.3)

# Handoff note: this draft moves to the sibling repo
# (nix-international-telephony) as its `services.pbx-lens` module; the
# sibling owns the deploy surface, pbx-lens owns the binary. Kept here so
# the option shape is decided next to the code that must satisfy it.

{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.pbx-lens;
in
{
  options.services.pbx-lens = {
    enable = lib.mkEnableOption "the pbx-lens call-history index UI";

    package = lib.mkPackageOption pkgs "pbx-lens" { };

    address = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1:8300";
      description = "Listen address. Keep loopback and front it with the reverse proxy.";
    };

    stateDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/pbx-lens";
      description = "Index DB directory (disposable: always re-derivable from sources).";
    };

    telnyxApiKeyFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = ''
        Telnyx API key file for the recording proxy. Loaded as a systemd
        credential; null disables the proxy path.
      '';
    };

    htpasswdFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = ''
        bcrypt htpasswd file (htpasswd -B). Loaded as a systemd credential;
        null leaves every route open — acceptable only on loopback dev.
      '';
    };

    recordingsDir = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = "/var/lib/telephony/recordings";
      description = "FreeSWITCH recordings directory served through the proxy.";
    };

    pollEvery = lib.mkOption {
      type = lib.types.nullOr lib.types.nonEmptyStr;
      default = "15m";
      description = "Poll cadence (systemd time span); null disables the poller.";
    };

    cdrFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "FreeSWITCH Master.csv re-ingested every poll.";
    };

    webhookLog = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "Receiver webhook JSONL re-ingested every poll.";
    };

    transcriptsDir = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "Agent transcripts dir re-scanned every poll.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.pbx-lens = {
      description = "pbx-lens — unified call-history index (read-only UI)";
      after = [ "network-online.target" ];
      wants = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        Type = "notify";
        WatchdogSec = "30";
        DynamicUser = true;
        StateDirectory = "pbx-lens";
        StateDirectoryMode = "0750";
        ExecStart = lib.concatStringsSep " " (
          [
            (lib.getExe cfg.package)
            "serve"
            "-db ${cfg.stateDir}/index.db"
            "-addr ${cfg.address}"
          ]
          ++ lib.optionals (cfg.telnyxApiKeyFile != null) [
            "-telnyx-key-file \${CREDENTIALS_DIRECTORY}/telnyx-api-key"
          ]
          ++ lib.optionals (cfg.htpasswdFile != null) [
            "-htpasswd \${CREDENTIALS_DIRECTORY}/htpasswd"
          ]
          ++ lib.optionals (cfg.recordingsDir != null) [
            "-recordings-dir ${cfg.recordingsDir}"
          ]
          ++ lib.optionals (cfg.pollEvery != null) [ "-poll-every ${cfg.pollEvery}" ]
          ++ lib.optionals (cfg.cdrFile != null) [ "-cdr-file ${cfg.cdrFile}" ]
          ++ lib.optionals (cfg.webhookLog != null) [ "-webhook-log ${cfg.webhookLog}" ]
          ++ lib.optionals (cfg.transcriptsDir != null) [
            "-transcripts-dir ${cfg.transcriptsDir}"
          ]
        );
        LoadCredential =
          lib.optional (cfg.telnyxApiKeyFile != null) "telnyx-api-key:${cfg.telnyxApiKeyFile}"
          ++ lib.optional (cfg.htpasswdFile != null) "htpasswd:${cfg.htpasswdFile}";
        ReadOnlyPaths = lib.optionals (cfg.recordingsDir != null) [ cfg.recordingsDir ];
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        RestrictAddressFamilies = [
          "AF_UNIX"
          "AF_INET"
          "AF_INET6"
        ];
        MemoryDenyWriteExecute = true;
      };
    };
  };
}
