---
id: 000268
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# filter Ctrl mouse-move escape sequences in pair

## Problem

`couchtty.stripWheelResizeModifier` (`cmd/internal/couchtty/mouse.go` — #213) strips the `Ctrl` bit from SGR mouse *wheel* reports (`button 64/65 + ModCtrl 16 = 80/81 → 64/65`) so `Ctrl+Wheel` scrolls instead of triggering Zellij's pane-resize. That fix lives in the `couch` host (the parent that owns the PTY and sees bytes before Zellij).

When running `pair` directly (`pair term` / standalone `pair` without `couch` in the path), the same Zellij resize still fires on `Ctrl+MouseMove`/`Ctrl+Drag`. Zellij maps `Ctrl` + pointer motion to "resize the pane under the cursor" (same family as `Ctrl+Wheel → resize`). `pair term` forwards raw SGR reports (`\x1b[<button;col;rowM`) to its child / to Zellij's scroll handling without stripping the modifier, so holding `Ctrl` while moving/dragging the mouse resizes Zellij panels instead of selecting/scrolling inside the terminal.

Reproduction: start `pair term` (or `pair` with the `pair term` right pane) inside Zellij 0.44/0.45, hold `Ctrl`, drag to select text or move the mouse over the pane — Zellij shows the resize overlay / changes panel size. With `couch` in front the wheel case is gone, but the move/drag case remains in the direct-`pair` path.
