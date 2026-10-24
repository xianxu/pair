---
id: 000851
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'e9e5ac80530f6ec61e159ffd6c2c4b7dd2c5be86' # card fields mirrored from issue-cards; edit via sdlc
---

# Switcher: single click selects, a click on the selected row or a double click switches

## Problem

## Spec

**Switcher clicks: select first, then switch (operator, 10-23).** In the Couch switcher, a single mouse click today
jumps straight to the clicked slot, and the operator sometimes isn't sure where they clicked. New behaviour:
- **Single click** selects the row (moves the highlight) without switching.
- **A click on the already-selected row** switches to it.
- **Double click** on any row switches to it directly.
Keyboard behaviour is unchanged (Enter still switches to the selected row). Other click targets in the switcher
(chips, +N groups, action buttons) keep their current behaviour unless they also jump on a single click, in which
case they follow the same rule.

## Done when

- Single click selects without switching; a second click on the selected row switches; a double click switches
  (tests driving the real mouse event path, including the double-click timing window).
- Enter and the other switcher keys behave as before (existing tests pass).

## Plan

- [ ]

## Log

### 2026-10-23
