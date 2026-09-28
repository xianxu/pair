---
id: '000089'
status: done
started: 2026-07-05T12:07:32-07:00
created: 2026-06-30
updated: 2026-07-05
estimate_hours: 11.66
actual_hours: 6.96
---

# review mode should disable edit while agent update the doc

## Problem

While the agent is producing a review round, the human may keep editing the doc.
A hard lock (make the buffer non-modifiable during the agent's turn) prevents the
agent's edits from landing on a doc that no longer matches what it reviewed — but
it's a workflow degradation: the human would rather keep working.

The record model already reconciles *non-overlapping* concurrent edits for free:
`apply.lua` resolves each record's `old`@occurrence against the **live** buffer
(`apply.lua:261`, `local base = buf_content(buf)`), so an agent edit to a region
the human didn't touch still anchors and applies. Two failure modes remain:

- **Overlap** — the human edited the exact span the agent targeted → `old` isn't
  found → the record is silently dropped (`'not found'`, WARN only). **This is
  what this issue addresses**: surface it as a reconcilable marker instead of a
  silent drop.
- **Occurrence-shift** — the human added/removed an earlier copy of the record's
  `old` text → "the Nth occurrence" now resolves to the *wrong* instance → a
  silent, *incorrect* edit. A distinct latent bug; **out of scope here** (this
  design keeps today's occurrence-against-live-buffer resolution). Noted so it
  isn't mistaken for fixed.

The lock exists only to preserve the invariant *what-the-agent-saw ==
what's-in-the-buffer*. We make that invariant explicit and reconcile against it
per-record instead of enforcing it by disabling the human.
