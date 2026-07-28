---
id: 000123
status: codecomplete
deps: []
github_issue:
created: 2026-07-27
updated: 2026-07-27
estimate_hours: 0.8
started: 2026-07-27T17:29:35-07:00
actual_hours: 1.23
---

# Split the right terminal pane with Alt+Shift+d

## Problem

Layout 3 has a Pair-owned right-side terminal area, but it only supports
terminal tabs inside one floating pane. When two live terminal views are needed
side-by-side in the right area, the user has to leave Pair's keybinding model and
manually invoke Zellij splitting. The desired workflow is one terminal shortcut:
`Alt+Shift+d` creates a top/bottom split in the right-side terminal area.

## Spec

- In layout 3, `Alt+Shift+d` while focus is in the right-side terminal context
  creates a top/bottom split inside that right terminal area.
- The existing terminal pane remains above; the newly created terminal pane is
  below and receives focus.
- The new pane runs the same right-terminal command shape as the original
  terminal pane, so it is a real Pair terminal process and remains inside
  Pair/Zellij lifecycle management.
- The split uses Zellij panes, not `pair term` internal tabs, but Pair keeps the
  panes fixed in place rather than allowing mouse drag movement/resizing.
- The shortcut is terminal-local: left-stack draft/agent/review behavior must not
  be hijacked by the new binding.
- Mouse focus behavior remains unchanged. Zellij mouse layout manipulation stays
  disabled so dragging does not move or resize workbench panes.
- `ARCH-DRY`: reuse the existing terminal-local shortcut routing patterns and
  layout command strings; do not introduce an unrelated split subsystem.
- `ARCH-PURPOSE`: deliver the actual pane split and draggable boundary, not just
  another `pair term` tab.

## Done when

- `Alt+Shift+d` in the right terminal creates a top/bottom Zellij split and
  focuses the new lower pane.
- Dragging the right split panes or left workbench boundaries does not move or
  resize Pair's fixed layout.
- Existing terminal shortcuts (`Alt+t`, `Alt+w`, `Alt+r`, tab switching,
  geometry toggle) still behave as before.
- Tests cover the shortcut routing/action shape and prove left/review contexts do
  not claim `Alt+Shift+d` unexpectedly.

## Revisions

- 2026-07-27: Live testing showed native `new-pane --direction down` followed
  ambiguous Zellij client focus in the pinned floating layout. The implementation
  was revised to split the invoking right terminal's explicit floating geometry
  and create a pinned lower floating terminal pane, preserving visible bordered
  Zellij panes without depending on focused-client routing.
- 2026-07-27: Live testing showed bordered floating panes could be dragged out
  of place, and left workbench boundaries still resized by mouse drag. The
  implementation was revised again to create borderless split terminal panes and
  disable Zellij mouse layout manipulation for the fixed workbench.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.08 impl=0.04
item: smaller-go-module design=0.27 impl=0.25
item: atlas-docs design=0.05 impl=0.05
design-buffer: 0.15
total: 0.80
```

## Plan

- [x] Add failing shortcut/config tests for right-terminal `Alt+Shift+d`.
- [x] Implement the minimal Zellij action routing and config updates.
- [x] Update docs/atlas for the new keybinding.
- [x] Run focused and full verification.

## Log

### 2026-07-27
- 2026-07-27: closed — Implemented Alt+Shift+d as a right-terminal-local Zellij downward split running the layout3 pair term shell, with bordered panes for mouse boundary resizing. Verified with RED focused failures before implementation; then go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1; go test ./... -count=1; make test-lua; bash tests/term-pane-shortcuts-test.sh; bash tests/review-toggle-test.sh; zellij --config-dir zellij setup --check; git diff --check.; review verdict: FIX-THEN-SHIP
- Claimed locally. Broadcast failed because this checkout has no `main` worktree.
- Design approved: Zellij-native top/bottom split in the right terminal area,
  focus the new bottom pane, preserve mouse boundary resizing.
- Plan-quality found the split action command shape under-specified. Refined the
  durable plan to pin `new-pane --direction down --name terminal -- sh -c ...`,
  require the layout-3 `pair term` shell command shape, keep pane borders, and
  avoid disabling Zellij mouse mode while preserving `focus_follows_mouse false`.
- Implemented `Alt+Shift+d` as a right-terminal-local split action. Verified RED
  before code with focused Go compile failures for missing chord/action symbols
  and shell failure `term: unknown shortcut "Alt+Shift+d"`.
- Verified GREEN/full: `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `go test ./... -count=1`; `make test-lua`; `bash tests/term-pane-shortcuts-test.sh`;
  `bash tests/review-toggle-test.sh`; `zellij --config-dir zellij setup --check`;
  `git diff --check`.
- Close review verdict was FIX-THEN-SHIP: the terminal split's new KKP forwarding
  needed proof it did not regress the existing review-pane `Shift+Alt+d`
  definition shortcut. Added a review-pane raw `ESC[68;4u` visual mapping alias
  and smoke coverage. Verified with `bash tests/review-window-test.sh`;
  `bash tests/review-toggle-test.sh`; `make test-lua`;
  `bash tests/term-pane-shortcuts-test.sh`; `git diff --check`.
- Live test showed `zellij action new-pane` prints the created pane id
  (`terminal_N`) to stdout, leaking into the terminal after `Alt+Shift+d`.
  Root cause: `OSRuntime.RunZellijAction` streams Zellij stdout to the focused
  pane, and `new-pane` documents the pane id as its return value. Added a quiet
  Zellij action path for the split only, rebuilt `bin/pair`, and verified with
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `bash tests/term-pane-shortcuts-test.sh`; `git diff --check`.
- Live test then showed the split could appear in the left stack. Root cause:
  `new-pane --direction down` follows Zellij's client focus, which can differ
  from the right `pair term` process that received the forwarded key bytes in
  the pinned floating layout. Added `--near-current-pane` so Zellij splits near
  the invoking terminal pane, rebuilt `bin/pair`, and verified with
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `bash tests/term-pane-shortcuts-test.sh`; `bin/pair term --test-shortcut Alt+Shift+d`;
  `git diff --check`.
- Live log/state check after another `Alt+Shift+d` press showed no stale binary:
  live `pair term` processes mapped to the current rebuilt `bin/pair` inode, and
  Zellij pane inventory still showed one right floating terminal plus the left
  stack. Replaced native split routing with explicit pane-id/geometry actions:
  shrink the invoking right terminal to the upper half, then create a pinned
  bordered floating terminal in the lower half while discarding `new-pane`
  stdout. Added ambiguous-focus regression coverage and `pane_y` parsing.
  Rebuilt `bin/pair` and verified with
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `bash tests/term-pane-shortcuts-test.sh`; `git diff --check`.
- Live test confirmed `Alt+Shift+d` now creates the right split, but the split
  panes could be dragged around and the left panel boundary could still be
  resized with the mouse. Root cause: in Zellij, bordered floating panes are
  mouse-movable, and global mouse mode enables mouse layout manipulation for
  tiled boundaries. Switched the split panes to `--borderless true` and set
  `mouse_mode false` in the Pair Zellij config so the workbench stays fixed.
