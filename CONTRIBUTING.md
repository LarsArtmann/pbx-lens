# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run the quality gate (below)
5. Submit a pull request

## Development Setup

The repo is a Go 1.27 module with a Nix flake; quality automation is owned by
BuildFlow — run the gate, not the individual tools:

```console
GOEXPERIMENT=jsonv2 go test ./...   # quick loop while coding
buildflow --build-mode dev          # the quality gate (lint + format + test)
nix build                           # package build — BROKEN (empty result), see TODO_LIST.md
```

If a step fails, `buildflow --fix` runs detect → repair → verify; drill into
one tool with `buildflow -s <tool> -v`. Do not hand-run formatters or
golangci-lint — BuildFlow owns them and hand-runs bypass its verification.

Project conventions (stack constraints, domain rules, deliberate lint
non-fixes) live in [AGENTS.md](AGENTS.md); the vocabulary in
[docs/DOMAIN_LANGUAGE.md](docs/DOMAIN_LANGUAGE.md).

## Reporting Issues

Please use GitHub Issues to report bugs or request features.

## Reporting Vulnerabilities

See [SECURITY.md](SECURITY.md) — do not open public issues for vulnerabilities.
