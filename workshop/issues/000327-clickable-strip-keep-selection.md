---
id: 000327
status: open
deps: []
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
---

# Clickable right-pane tab strip without losing drag selection

## Problem

#311 made the right-pane tab strip clickable over a plain shell by switching
`pair term` to the `AnyMotion` mouse policy, which took drag selection away
from zellij (#326 reverted to `ChildRequested`). Today the strip is clickable
only while the child requests mouse tracking. Want both: clickable strip and
native-feeling selection. Options and the parent-side-selection cost breakdown
are in #326's Spec/Log (zellij click hook, parent-side selection + OSC 52,
Shift+drag).

## Spec

## Done when

-

## Plan

- [ ]

## Log

### 2026-09-24
