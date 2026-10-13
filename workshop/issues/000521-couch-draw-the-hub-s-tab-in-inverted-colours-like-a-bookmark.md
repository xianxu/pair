---
id: 000521
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '66e18e5f9723acff649886787e52aecbcab99561' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: draw the hub's tab in inverted colours, like a bookmark

## Problem

Couch's tab bar leads with the hub (pair#491), but it looks like every other tab, so the eye has to read its label to find it. The operator (2026-10-12) wants the hub marked like a bookmark.

## Spec

Draw the hub's tab in inverted (reverse-video) colours of its normal style, so it stands out as a bookmark wherever the bar is. The other state cues (red, bell, focused) stay readable on it: decide how each combines with the inversion and show them in the issue (a small table), then the operator approves on a live look. Same treatment for the hub's row in the switcher, if it reads well. No new keys.

## Done when

- The hub's tab is inverted in the tab bar at every width (including when other tabs fold into +N); red, belled and focused hub tabs stay distinguishable; tests pin the rendering; the operator approves the look.

## Plan

- [ ]

## Log

### 2026-10-12
