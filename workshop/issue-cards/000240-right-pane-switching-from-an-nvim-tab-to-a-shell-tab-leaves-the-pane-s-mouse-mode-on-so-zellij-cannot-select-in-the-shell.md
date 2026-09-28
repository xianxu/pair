---
id: '000240'
status: done
started: 2026-09-12T19:05:00-07:00
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.47
actual_hours: 1.47
---

# right pane: switching from an nvim tab to a shell tab leaves the pane's mouse mode on, so zellij cannot select in the shell

## Problem

Operator report, 2026-09-12: "I can't select at all in a terminal in the
right pane." The operator keeps nvim (parley) in one `pair term` tab and a
shell in another.

Reproduced with a probe that runs the real `pair term` under a pty and
records every mouse DECSET/DECRST it writes to the outer terminal:

| step | bytes written outward |
|---|---|
| tab 1: `nvim --clean` starts | `?1002h`, `?1006h` |
| Alt+t → tab 2 (shell) | **nothing** |
| Alt+Left → tab 1 (nvim) | `?1002h`, `?1006h` (nvim's own bytes, replayed) |
| Alt+Right → tab 2 (shell) | **nothing** |
| `:q!` in tab 1 | `?1002l`, `?1006l` |

Identical on today's build and on a control build from before #234, so this
is not a regression from today's stdin-loop change; it surfaced because the
operator recently started keeping nvim in a right-pane tab (#227, #234).

Mechanism: `pair term` is the pane's only application from zellij's point of
view. The active child's output passes through to zellij live, so nvim's
`?1002h;?1006h` sets the PANE's mouse mode in zellij. A tab switch
(`applyTakeover`, `run.go:993`) composes `HomeAndClear` + the incoming
child's replay and asserts nothing else — `hostty.repaint` says so
deliberately, because couch owns its host's mouse mode itself. The shell
child never said anything about the mouse, so its replay carries no
`?1002l`, and zellij keeps believing the pane wants mouse reports. It then
forwards every click and drag to `pair term` → the shell, instead of doing
its own selection. Selection is dead in that tab for as long as nvim lives
in the other.

The reverse direction works today only by accident: switching back to nvim
re-enables the mode because nvim's startup `?1002h` happens to still be in
its replay ring. A long-running nvim whose ring has dropped it would come
back with mouse OFF after a round trip.
