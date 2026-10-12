---
id: 000503
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '0aad4715728e7b4899c904be9eb0c2fd0c0df409' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:12:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "9e39e630", done: "bde13a86"}
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

- [x] Remove the control: `broadcast.ControlLabel`, `RenderedStatusRow.Remote`, `Console.statusRemote`, `onRemoteClick` and its click branch; `StatusGuardCols` shrinks to `LIVE ⏸ 👆` (12 → 9 columns).
- [x] Tests: the exact-row, span and guard tests read `LIVE ⏸ 👆`; the 👽 click test goes; the shared pin-spot table (`tests/broadcast-viewer/pinspots.json`) and `TestNoteVisible` follow the narrower guard.
- [x] Docs: README, atlas/broadcast.md, atlas/webcast.md; #407's Log notes the removal.
- [ ] Merge main (#495 touches the same row), full suite, close.

## Log

### 2026-10-12
- The guard width reaches the web viewer over the wire (`event: notes` carries `guard`), so the viewer needs no change; its `.mjs` fixtures carry their own guard. Only the shared pin-spot table pins the number, and Go checks it equals `StatusGuardCols`.
- `TestNoteVisible` now derives its columns from `StatusGuardCols` rather than hard-coding a column inside the old guard.
