---
id: '000326'
status: done
started: 2026-09-24T22:18:43-07:00
created: 2026-09-24
updated: 2026-09-24
actual_hours: 0.16
---

# Right-pane drag selection broken since clickable tabs (#311)

## Problem

Since #311 (clickable right-pane tab strip), mouse drag-to-select text in the
right pane (`pair term`) no longer works — operator report, 2026-09-24.

Likely cause (from reading the #311 diff, not yet reproduced): `1c5ed349`
switched `newTerminalMux` from `terminal.ChildRequested` to
`terminal.AnyMotion` (`cmd/internal/termcmd/presentation.go`). Before, a plain
shell tab (child tracking 0) meant the parent requested no mouse reports, so
zellij/the host terminal did native text selection. Now the parent always
emits `\x1b[?1003h`, so the host forwards every press/drag to `pair term`
instead of selecting. In `Presenter.mouseInput`
(`cmd/internal/terminal/presenter.go`), a press over a child with
`Tracking == 0` becomes `ParentPressMouse`, which is swallowed — and the
parent has no selection implementation of its own. Net: the drag goes
nowhere.
