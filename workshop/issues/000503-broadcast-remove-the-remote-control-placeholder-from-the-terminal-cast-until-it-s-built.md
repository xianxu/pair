---
id: 000503
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'edb1a7337e02686117bb0894292f5841789cb2ce' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:12:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# broadcast: remove the 👽 remote-control placeholder from the terminal cast until it's built

## Problem

The terminal broadcast's status row shows `LIVE ⏸ 👆 👽`. `👽` is a placeholder for remote control of the terminal (#407, not built): clicking it does nothing (`onRemoteClick` 'waits for #407'). The operator (10-12): remove it until remote control for the terminal is picked up. Remote driving exists for the web cast (#467) through its drive link.

## Spec

Remove the 👽 control from the terminal cast: the status row becomes `LIVE ⏸ 👆` (plus notes), the `ControlLabel`/`Remote` span, `onRemoteClick` and their layout and mark geometry (`statusRemote`, the pointer-check spans, broadcast indicator constants) go, with tests and docs updated. Leave #407 open and note in it that the control was removed and comes back with the feature. Coordinate with #495 (pair:4, cast codes) and #491 (tab bar), which touch the same status row: merge main before close.

## Done when

- The terminal cast shows no 👽; pointer and notes controls still work and their geometry is correct; tests pass; #407 notes the removal.
- The operator sees it at the next smoke test.

## Plan

- [ ]

## Log

### 2026-10-12
