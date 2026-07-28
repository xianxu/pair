---
id: 000123
status: working
deps: []
github_issue:
created: 2026-07-27
updated: 2026-07-27
estimate_hours: 0.8
started: 2026-07-27T17:29:35-07:00
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
  creates a Zellij-native horizontal split inside that right terminal area.
- The existing terminal pane remains above; the newly created terminal pane is
  below and receives focus.
- The new pane runs the same right-terminal command shape as the original
  terminal pane, so it is a real Pair terminal process and remains inside
  Pair/Zellij lifecycle management.
- The split uses Zellij panes, not `pair term` internal tabs, so Zellij owns the
  pane boundary and mouse drag resizing works where Zellij supports it.
- The shortcut is terminal-local: left-stack draft/agent/review behavior must not
  be hijacked by the new binding.
- Mouse focus behavior remains unchanged. Only mouse behavior needed for Zellij
  boundary drag resizing may be enabled.
- `ARCH-DRY`: reuse the existing terminal-local shortcut routing patterns and
  layout command strings; do not introduce an unrelated split subsystem.
- `ARCH-PURPOSE`: deliver the actual pane split and draggable boundary, not just
  another `pair term` tab.

## Done when

- `Alt+Shift+d` in the right terminal creates a top/bottom Zellij split and
  focuses the new lower pane.
- The boundary between the two right terminal panes can be resized with the
  mouse under Zellij's normal pane-resize behavior.
- Existing terminal shortcuts (`Alt+t`, `Alt+w`, `Alt+r`, tab switching,
  geometry toggle) still behave as before.
- Tests cover the shortcut routing/action shape and prove left/review contexts do
  not claim `Alt+Shift+d` unexpectedly.

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

- [ ] Add failing shortcut/config tests for right-terminal `Alt+Shift+d`.
- [ ] Implement the minimal Zellij action routing and config updates.
- [ ] Update docs/atlas for the new keybinding.
- [ ] Run focused and full verification.

## Log

### 2026-07-27
- Claimed locally. Broadcast failed because this checkout has no `main` worktree.
- Design approved: Zellij-native top/bottom split in the right terminal area,
  focus the new bottom pane, preserve mouse boundary resizing.
- Plan-quality found the split action command shape under-specified. Refined the
  durable plan to pin `new-pane --direction down --name terminal -- sh -c ...`,
  require the layout-3 `pair term` shell command shape, keep pane borders, and
  avoid disabling Zellij mouse mode while preserving `focus_follows_mouse false`.
