---
id: 000118
status: open
deps: [117]
github_issue:
created: 2026-07-24
updated: 2026-07-24
estimate_hours:
---

# Rename terminal tabs in the pane frame

## Problem

Alt+r currently calls a raw prompt rendered into the active child terminal's
content area. It feels acceptable at an idle shell prompt but visually corrupts
or competes with full-screen applications such as Neovim. Zellij's native
rename-tab editor cannot be reused because the right-side tabs are Pair's own
PTY multiplexer tabs inside one Zellij floating pane.

## Spec

- Alt+r in `pair term` enters a Pair-owned rename mode for the active internal
  terminal tab. It never forwards the chord or editing bytes to the child PTY.
- Rename mode edits the existing tab name, initially placing the cursor at the
  end. The Zellij pane frame is the editor surface: on each edit its title shows
  the tab inventory with the active entry rendered as an explicit rename field
  and cursor marker. The child application's viewport is not cleared, scrolled,
  or written to.
- Printable UTF-8 inserts at the cursor. Left/Right move one rune;
  Home/End move to the boundary; Backspace/Delete remove one rune on the
  corresponding side.
- Enter trims surrounding whitespace and commits a non-empty name. An empty
  result retains the previous name. Escape cancels and restores the previous
  name. Either exit restores the ordinary tab-inventory frame title.
- While rename mode is active, unrelated bytes and Pair shortcuts are consumed
  rather than reaching the child. Terminal resize and child output continue to
  be processed normally; child output must not erase the rename title.
- The editor transition is a pure rune/cursor state machine (`ARCH-PURE`).
  Terminal input decoding owns sequence buffering, and the existing
  `renamePane`/Zellij runtime is the single title-output boundary
  (`ARCH-DRY`). No Zellij-native tab or pane rename mode is entered.
- Confirmation-global focus routing remains owned by #117 and is not coupled
  to rename mode.

## Done when

- Alt+r edits the active Pair terminal tab name in the pane frame while a shell,
  Neovim, or another full-screen child has focus.
- Enter commits and Escape cancels without any prompt or edit bytes appearing
  in the child application.
- Unicode insertion, cursor movement, Backspace/Delete, empty-name handling,
  and fragmented escape sequences have table-driven tests.
- Existing tab create/close/switch behavior and pane-title inventory remain
  unchanged outside rename mode.

## Plan

- [ ] Write and approve a durable implementation plan.
- [ ] Implement the pure rename editor and terminal input mode test-first.
- [ ] Verify the terminal integration suite and live Neovim-child behavior.

## Log

### 2026-07-24

- Confirmed the right-side tabs are Pair-owned PTYs, so Zellij's native
  rename-tab editor targets the wrong abstraction. Selected a Pair-owned
  frame-title editor that works independently of the foreground child.
