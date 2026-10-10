---
id: 000421
status: working
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: '7d2de9e8a25c278b4e080d1049602c34b57ed94b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T22:53:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
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

Decided by TL ariadne:1 on 2026-10-09:
- **Relaunch**, keeping the conversation, with the guards and a receipt.
- Also expose **Shift+Alt+N** ("reload context": `pair agent restart`, which
  SIGUSR2s pair-wrap so it starts a fresh agent conversation inside the same
  Pair process) as a second TL-callable operation, with the same guards. It does
  not pick up a new Pair binary; only relaunch does.
- Consider **stale-binary detection**. Tonight a relaunch did not pick up the
  pair#418 fix, because `bin/pair` in pair:0 was built at 11:34 and not
  rebuilt until a manual `make build` at 22:53. A rollout relaunch is only
  worth doing onto a binary newer than the slot's running one.

## Done when

- `couch` has two CLI verbs for a live idle slot: relaunch (the current binary,
  same conversation) and reload-context (a fresh agent conversation). Both
  refuse a busy or dirty slot with an explicit reason.
- Relaunch reports, and by default refuses, a binary that is no newer than
  the one the slot is running (decide the exact rule in design).
- It returns a receipt id; a status query shows the outcome and why.
- Tests cover the guard decisions (pure) and the dispatch through a fake; a
  live check restarts a real idle slot.

## Plan

- [ ]

## Log

### 2026-10-09
