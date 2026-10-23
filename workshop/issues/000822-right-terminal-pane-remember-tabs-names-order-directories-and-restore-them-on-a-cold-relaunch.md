---
id: 000822
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f7647859f46262df4d051a528c1fa448b9a36ae6' # card fields mirrored from issue-cards; edit via sdlc
---

# Right terminal pane: remember tabs (names, order, directories) and restore them on a cold relaunch

## Problem

## Spec

Operator request (10-22): pair should remember what's in the **right terminal pane** and recover it on a full cold
restart (relaunch), as the agent pane (resumed conversation) and the nvim draft already do. **At minimum: the number of
right-pane tabs and their names.**

- Persist per slot (per worktree): the right pane's tab list in order (names as renamed with Alt+r), which tab was
  active, and each tab's working directory; on a cold relaunch, recreate the tabs with those names and directories,
  active tab restored. Running processes are not resurrected (a fresh shell in the same directory), unless a later
  issue adds opt-in command replay.
- Owned by pair (the standalone wrapper), stored with the slot's existing per-slot state; Couch needs no new state.
- A missing or corrupt record falls back to today's single default tab, with no error dialog.

## Done when

- Cold relaunch of a slot with three renamed right-pane tabs restores three tabs with the same names, order, directories
  and active tab (test).
- A corrupt or missing record starts the default single tab (test).
- README documents what is and isn't restored.

## Plan

- [ ]

## Log

### 2026-10-22
