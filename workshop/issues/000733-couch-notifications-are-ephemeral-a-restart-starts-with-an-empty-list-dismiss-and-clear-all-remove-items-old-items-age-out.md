---
id: 000733
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b18cbba52e7300ae9ce0e3073e21a54be18b5954' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: notifications are ephemeral: a restart starts with an empty list, dismiss and clear-all remove items, old items age out

## Problem

Couch's "Waiting on you" list keeps every `couch --notify` forever, across restarts: after the 10-20 reboot it showed
11 items going back to 21:03 the night before, all long handled (operator, 10-20, screenshot in ops's session file).
A list that only grows stops being read, which defeats notifications.

## Spec

- Notifications are **ephemeral**: a Couch restart starts with an empty list (they are a nudge, not a record; the
  durable record is the hub's Needs-you section).
- Opening or dismissing an item removes it; a "clear all" control empties the list.
- While Couch runs, items older than a bound (e.g. 12 h) drop off on their own.

## Done when

- Tests pin: restart starts empty; dismiss removes one; clear-all empties; age-out drops an old item.
- Operator live look: after the next restart the list is empty.

## Plan

- [ ]

## Log

### 2026-10-20
