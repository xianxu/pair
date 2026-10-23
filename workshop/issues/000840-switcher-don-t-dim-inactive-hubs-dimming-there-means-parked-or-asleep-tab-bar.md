---
id: 000840
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '11d2bbf1b2e8d5e95fad644592978d8f08a3701d' # card fields mirrored from issue-cards; edit via sdlc
---

# Switcher: don't dim inactive hubs (dimming there means parked or asleep); tab bar keeps it

## Problem

## Spec

**Don't dim inactive hubs in the switcher (operator, 10-23).** The tab bar dims an inactive hub (50% background
blend, operator decision 10-22) to show where work is happening, and that stays. The switcher applies the same dimming,
which is confusing there: the switcher already uses dimming to mean a parked (or asleep) thread. Remove the
inactive-hub dimming from the switcher only; parked/asleep dimming there is unchanged.

## Done when

- In the switcher, an inactive but live hub renders like any other live row; a parked or asleep row is still dimmed
  (render test).
- The tab bar still dims inactive hubs (existing test passes).

## Plan

- [ ]

## Log

### 2026-10-23
