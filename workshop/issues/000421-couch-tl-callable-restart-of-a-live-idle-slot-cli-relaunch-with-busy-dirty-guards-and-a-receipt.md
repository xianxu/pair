---
id: 000421
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: '1c91d0bddee890a184daaf7011ff244ec1455a2c' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt)

## Problem

Requested by TL ariadne:1 (operator-approved) on 2026-10-09. After a Pair
rollout, nothing scriptable can restart an idle worker slot onto the new binary.
`couch --reboot repo:N` refuses a live slot ("not-offered: ... is live"), so the
operator pressed Alt+n by hand in 7 slots.

Alt+n is **relaunch** (`couchcore` relaunch; the console's `onRelaunchHotkey`).
It parks the thread and cold-resumes it, which replaces the Pair process with
the current binary and keeps the agent conversation. Reboot, by contrast,
archives the conversation and starts a fresh agent. No CLI exposes relaunch.

## Spec

A TL-callable restart of a live slot, which a dispatcher can script across a
fleet:
- Refuses when the slot is busy: the agent is mid-turn, or its composer is
  occupied (a draft, a suggestion being edited, a dialog).
- Refuses when the slot's checkout is dirty.
- Returns a receipt the caller can verify afterwards, in the shape of
  `--message-status`.

Open decision (sent to TL ariadne:1): keep the conversation (relaunch, which is
what Alt+n did tonight), or archive it like reboot (fresh agent)? The TL's ask
says "archives the conversation like reboot". Relaunch is the recommendation,
because it reproduces what the operator did for the rollout without losing
context.

## Done when

- `couch` has a CLI verb that restarts a live idle slot on the current binary
  and refuses a busy or dirty slot with an explicit reason.
- It returns a receipt id; a status query shows the outcome and why.
- Tests cover the guard decisions (pure) and the dispatch through a fake; a
  live check restarts a real idle slot.

## Plan

- [ ]

## Log

### 2026-10-09
