---
id: 000117
status: working
deps: []
github_issue:
created: 2026-07-24
updated: 2026-07-24
estimate_hours:
started: 2026-07-24T13:52:38-07:00
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

## Spec

- A **global Pair hotkey** is an action intended to work from every primary
  workbench pane regardless of whether the agent, draft, or right terminal has
  focus.
- Audit and classify every Pair-defined shortcut:
  - draft-routed globals: Alt+d, Alt+x, Alt+n/Ctrl+Alt+n, Shift+Alt+N,
    Alt+Up, Alt+Down, and Alt+c;
  - direct global surfaces: Alt+h and Alt+l, which open independent floating
    panes and do not execute inside draft Neovim;
  - pane-local actions: Alt+j/k/t/w/r, Alt+/, compaction, and
    Alt+Shift+Return; these remain scoped to their owning pane.
- Zellij bindings for draft-routed globals must only forward a distinctive key
  sequence to the focused pane. They must not combine directional focus changes
  with writes.
- The agent wrapper and right-terminal wrapper must decode those sequences
  through the shared `workbenchshortcut` model and route the corresponding Lua
  function to the draft pane by stable pane id.
- Draft Neovim must invoke the functions locally. Pair-owned Neovim overlays
  that expose global workbench keys must route them to the draft rather than
  executing a draft-only command in the overlay.
- One shared action-to-draft-function mapping must drive the Go wrappers
  (`ARCH-DRY`). Shortcut classification remains pure; Zellij pane discovery and
  writes stay in thin injected runtime seams (`ARCH-PURE`).
- The audit is complete only when every Pair-defined shortcut is explicitly
  classified and every draft-routed global is covered from agent, draft, and
  right-terminal focus (`ARCH-PURPOSE`).
- Zellij's independent `focus_follows_mouse` option is not enabled by this
  issue. Hover focus is not needed for reliable hotkey delivery and would
  change focus merely by moving the pointer.

## Done when

- Alt+n and its Ctrl+Alt+n alias open `PairConfirmRestart()` when the right
  terminal is focused; no control bytes or Lua text reach the shell.
- Every draft-routed global works from agent, draft, and right-terminal focus.
- Global routing uses stable draft-pane discovery and contains no
  `MoveFocus`-then-write KDL choreography.
- Direct-global and pane-local shortcuts retain their existing behavior.
- Automated tests cover the shortcut inventory, decoding, draft destination,
  and absence of leaked bytes.

## Plan

- [ ] Write and approve a durable implementation plan.
- [ ] Implement the audited routing with test-first regressions.
- [ ] Update README/atlas shortcut ownership and verify the full workbench
      integration suite.

## Log

### 2026-07-24

- Reproduced the failure shape from the reported literal
  `^\\^N:lua PairConfirmRestart()` shell input. The KDL binding mixes focus and
  write actions while the already-working Alt+x path delegates routing to the
  focused process. Selected the latter as the common invariant.
- Confirmed Zellij supports `focus_follows_mouse true` with a default of false;
  Pair does not currently override it. Kept that independent behavior out of
  the routing fix.
