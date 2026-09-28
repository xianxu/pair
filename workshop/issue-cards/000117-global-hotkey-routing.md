---
id: '000117'
status: done
started: 2026-07-24T13:52:38-07:00
created: 2026-07-24
updated: 2026-07-25
estimate_hours: 4.98
actual_hours: 0.84
---

# Route global hotkeys through draft pane

## Problem

Pair declares several workbench-wide shortcuts in Zellij as a sequence of
`MoveFocus` and `Write`/`WriteChars` actions intended to target the draft
Neovim pane. Zellij does not guarantee sequential execution within a binding,
and a focused floating right terminal does not reliably leave its floating
layer through directional focus actions. Consequently Alt+n can type
`^\\^N:lua PairConfirmRestart()` into the user's shell instead of opening the
draft confirmation.

Alt+x already demonstrates the reliable shape: Zellij forwards a distinctive
sequence to the focused Pair-owned process, and that process locates the draft
pane by stable id before sending the Lua invocation. The remaining global
shortcuts need the same routing invariant rather than ad hoc focus movement.
