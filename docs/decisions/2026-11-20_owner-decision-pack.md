# Owner Decision Pack — 2026-11-20 (four calls from the review wave)

- **Created:** 2026-11-20 (Pareto execution plan T26, F70/F71). Four open
  policy calls the 20-skill review wave surfaced; each gates a small
  executable tail.
- **How to answer:** write the choice ID (or free-form) after
  **→ ANSWER:**. On arrival the gated rows in TODO_LIST.md flip to
  executable and this file records the outcome.
- **Code truth:** docs/DOMAIN_LANGUAGE.md (PII primitives, ASR variant),
  `internal/web/server.go` CSP wiring, `cmd/pbx-lens/catalog.go`,
  docs/DOMAIN_LANGUAGE.md § "Transcript truth (2026-10-23)".

---

## 1. PII boundary — when does `?pii=hash` become the default? → ANSWER:

**Current truth:** `E164`/`ParticipantHash` exist as primitives and the
export honors `?pii=hash`, but the DEFAULT export and every UI surface
still render raw numbers. The domain doc says PII enforcement is planned,
not shipped; nobody has defined the trigger.

**Options:**

- **A — keep raw default forever** (single-operator lens; the operator
  owns the PBX and the numbers). Cost: any future screen-share/export
  leak is a raw-number leak.
- **B — flip export default to hash, keep UI raw.** One flag flip in the
  export route + docs; the forensic-dump contract changes (hash-only
  rows unless `?pii=raw`).
- **C — full hash mode (UI + export).** Largest change: number columns
  render as stable short hashes with a reveal-on-demand affordance.

**→ ANSWER:**

## 2. CSP enforce flip — `Content-Security-Policy` report-only → enforcing? → ANSWER:

**Current truth:** the lens ships an enforcing CSP shape for its own
routes (nonce-threaded forms/scripts, the tc validity bridge needs the
nonce on every `forms.Form`) but the policy enforcement status for the
whole document surface was left report-only when the identity stack
landed; the login page scripts are the newest unreviewed surface.

**Options:**

- **A — flip now.** Requires one browser-lane pass over login + register
  - OIDC callback with the report-uri quiet; any third-party inline
    script (none known) would break loudly.
- **B — hold report-only** until the first real deploy receipts show a
  quiet report stream, then flip. Cost: the door stays open for an
  injected inline script the whole time.
- **C — flip per-page** (app pages enforce, auth pages report). Splits
  the policy into two headers; most code, most precise.

**→ ANSWER:**

## 3. Catalog push — publish the eventcatalog tree anywhere? → ANSWER:

**Current truth:** `pbx-lens catalog` renders the eventcatalog tree +
`-architecture` HTML locally (the DB-free exception command). The hub
(nix-international-telephony) links nothing; there is no published URL.

**Options:**

- **A — keep local-only.** The catalog is a developer mirror; CI
  proves it renders, nobody else reads it.
- **B — ship it on the docs site** (website-launch pattern) as a static
  artifact of CI. Cost: a publish lane + the catalog becomes a public
  API surface (domain terms leak).
- **C — push to the telephony hub repo** as a git artifact (no site).
  Cheap; the hub renders or links it.

**→ ANSWER:**

## 4. Divergence dossier — keep or fold the 2026-10-23 forensic deltas? → ANSWER:

**Current truth:** the transcript-truth doctrine records measured
divergences (created_at vs sent_at rewrite of every Telnyx stream; the
29.2 s tail vs 44.3 s speech disagreement). They are doctrine prose, not
a structured dossier; the refresh detector's `additive-only` class is
the only live watch on such drift.

**Options:**

- **A — fold into doctrine** (current state is enough; the
  2026-10-23 section IS the dossier). No action.
- **B — structured dossier:** a docs/ table of every measured
  source-vs-lens divergence with its detection mechanism, so the next
  forensic call starts from a checklist instead of archaeology.
- **C — automate:** teach the refresh classifier to flag the known
  divergence shapes (sent_at drift recurrences) as a named class.

**→ ANSWER:**

---

_(Answered choices get their outcome recorded inline and the matching
TODO_LIST rows flipped — same ritual as the 2026-11-12 identity sheet.)_
