---
id: 000464
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '7ab052f689a3764c7aeeb550e0ac28d866ce5535' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
