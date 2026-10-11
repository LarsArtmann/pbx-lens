# Security Policy

pbx-lens indexes call metadata (numbers, transcripts, recordings references)
for a live PBX — treat the index database as sensitive. The repo itself is
public, but call data never is: the index database, secrets, and any real
call content stay local and are never committed (fixtures use synthetic
numbers only).

## Reporting a vulnerability

Contact the owner directly: lars@artmann.tech (or GitHub private
vulnerability reporting on this repository). Do not open public issues.

Response time: single-maintainer project; reports are triaged as fast as
practical, typically within a few days.

## Supported Versions

Single live version: the `main` branch. No tagged releases, no maintained
branches.

## Security Practices

- READ-ONLY over every source: pbx-lens never writes back to the PBX
  runtime, receiver logs, or telephony service units.
- The Telnyx API key is read from `~/.pbx-prod-secrets/telnyx_api_key`
  (0600) and never logged; no secret ever reaches the index database.
- The index DB is disposable: delete and rebuild from sources at any time.
- Participant hashes (sha256-truncated) exist for the future PII policy;
  raw numbers are stored locally only, never exported.
