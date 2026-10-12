---
id: 000464
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'db60391a312465f1798ff7e7d0a741ea244ee705' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:15:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# couch: messages to an idle slot expire 'waiting for picker to close' when no picker is open

## Problem

2026-10-12 00:0x–00:5x: three `couch --send-to pair:3` messages from ops:0 expired with `delivery deadline elapsed: waiting for picker to close (1m30s)` (last probe: 6a381c3b-90c1-473f-a6c4-d32a3684f0b4). pair:3 (Claude Code 2.1.296) was idle at its prompt with no picker visible in `couch --peek`. Its prompt showed Claude Code's dim prompt suggestion (ghost text: "The operator reviewed the format; proceed to close and merge"), and earlier a background shell with the footer "1 shell · ↓ to manage". Other slots received messages normally in the same period. The hub couldn't reach the slot at all; the operator had to type into it by hand.

## Spec

Find what Couch's delivery reads as an open picker on that screen (the dim prompt suggestion, the background-task footer, or stale state from when one was shown) and fix the detection at the root: a picker is an actual selection UI, not ghost text or a footer hint. A message that can't be delivered says why in terms the sender can act on. Add a screen fixture from this case to the tests.

## Done when

- A fixture of an idle Claude Code prompt with a dim suggestion (and one with the background-task footer) reads as deliverable; real pickers still block.
- A live check: a message to an idle slot showing a prompt suggestion is submitted.

## Plan

Root cause (measured, below): `pickerActive` is a latch that only a plain Enter
through the wrapper clears. A picker that closes any other way (Claude Code's
unattended auto-deny, Esc, a digit key) leaves it armed for good, and peer
delivery waits "for picker to close" until someone presses Enter in the pane.

- [ ] Profile flag `overlayHidesComposer` (claude): its pickers replace the
      composer. While the latch is armed, each terminal feed reads the
      recognizer; once the composer has been seen hidden, its return means the
      picker closed, and the latch clears (same consume as Enter: tails reset).
- [ ] Tests: the recorded pair:3 screens (permission prompt with auto-deny
      countdown → idle composer with a faint suggestion) clear the latch and
      read deliverable; a picker still on screen stays armed; a composer that
      never went hidden (OSC ahead of the paint) doesn't clear; agents without
      the flag keep the Enter-only rule.
- [ ] Live check: a peer message to an idle slot showing a suggestion submits.

## Log

### 2026-10-12
- Measured on pair:3's recording (`scrollback-1-pair-7-claude.raw`): one
  `777;notify;Claude Code;Claude needs your permission` at 23:45 for a Bash
  prompt ("Dangerous rm operation…"), whose screen read "Claude Code will
  automatically deny this request in 1:59, to avoid blocking progress on an
  unattended session". It auto-denied with no Enter, so the latch stayed
  armed. The dim suggestion was a red herring: `peerPlaceholder` already reads
  faint ghost text as an empty composer.
