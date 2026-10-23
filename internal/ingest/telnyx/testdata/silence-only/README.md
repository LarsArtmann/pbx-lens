# `silence-only` fixture — a two-sided call the ASR never transcribed

Live evidence from the 2026-10-23 forensic investigation ("why did the lens say
the caller did not respond when the recording has a human greeting?"). The
report lives in the deployment repo:
`docs/status/2026-10-23_14-59_lens-transcript-deaf-asr-greeting-collision.md`.

## What was fetched

| File                | Live endpoint                                                            |
| ------------------- | ------------------------------------------------------------------------ |
| `conversation.json` | `GET /v2/ai/conversations/acea3e91-5464-42c1-9439-57296942687a`          |
| `messages.json`     | `GET /v2/ai/conversations/acea3e91-5464-42c1-9439-57296942687a/messages` |
| `recording.json`    | `GET /v2/recordings/2ffb2d8a-6b59-463e-951e-fed416b659ff`                |

Fetched 2026-10-23, unedited apart from the two trims noted in each file
(`system_prompt` removed, presigned download URLs blanked).

## Why this fixture exists

The message log carries **four rows and zero `user` turns**: two assistant
turns and two `system` rows whose text is `[long silence]`. The recording of
the same call is 45.06 s of **dual-channel** audio in which channel A carries a
human greeting at 0.3-3.2 s and a closing line at 39.9-44.3 s. So:

- the far end did speak (proven by per-channel RMS over the recording),
- Telnyx's recogniser produced no turn for it (two independent mechanisms: the
  greeting collided with the assistant's own greeting inside a protected
  window, and the closing line finished ~0.25 s before the call ended, short of
  the 1.0 s `on_no_punctuation_seconds` endpointing window),
- the lens then mapped the `system` markers away and rendered an
  assistant-only transcript, which reads as "the caller did not respond".

The fixture pins all three facts as data so the mapper, the page model and the
goldens can regress against reality instead of against a hand-written idea of
it.

## Deliberate properties

- `metadata.telnyx_agent_target` and `metadata.telnyx_end_user_target` are the
  SAME number (`+17287289311`): the call dialled the agent's own DID. The call
  page's self-dial signal keys on exactly this.
- `messages[].created_at` sits ~1 minute after `messages[].sent_at` for every
  row (persistence time vs dispatch time). `Message.SpokeAt()` prefers
  `sent_at`; the fixture is the regression that keeps the mapper honest about
  it.
- No audio file is committed. Reprocess one from the API if a future test needs
  the real waveform; the span table is quoted in the deployment report.
