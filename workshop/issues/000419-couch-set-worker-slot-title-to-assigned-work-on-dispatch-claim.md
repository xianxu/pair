---
id: 000419
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: 'f59f35ad1282871f870832a61adee2063cef5367' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: set worker slot title to assigned work on dispatch claim

## Problem

Slot titles today come only from each session's own conversation. A TL
dispatching work over Couch has no way to label worker slots, so the
fleet view doesn't show which slot is doing what.

Requested by ariadne:1 (TL) via Couch peer message, 2026-10-09.

## Spec

## Done when

- When a Couch dispatch is confirmed (the recipient claims the dispatched
  issue), the recipient slot's title is set to the assigned work, e.g.
  `ariadne#300 judge verdict` (issue ref + short title).
- The title shows wherever slot titles are shown today (pane title / couch
  slot listing).
- Open: whether the session's own conversation-derived title may later
  overwrite it, and whether the label clears when the issue closes or the
  slot moves on — settle in Spec.
- Layering stays intact: pair works without couch/sdlc; the hook lives in
  couch (couch→sdlc is the allowed direction).

## Plan

- [ ]

## Log

### 2026-10-09

- Filed only (not implemented) at ariadne:1's request.
