---
id: 000116
status: working
deps: []
github_issue:
created: 2026-07-23
updated: 2026-07-23
estimate_hours:
started: 2026-07-23T16:16:22-07:00
---

# Three-panel Pair layout with user terminal

## Problem

Pair's current session layout is optimized around two surfaces: an agent output
pane and a Neovim draft/input pane. That leaves the user's own shell workflow
outside the main Pair workspace, even though real pairing often needs a terminal
for commands, ad hoc inspection, or opening a full Neovim instance while the
agent continues working.

The desired workbench shape is a three-panel session: preserve the familiar
agent/draft split, but add a first-class user terminal surface where the user
can run a shell or launch `nvim` without stealing the agent/draft panes.

## Spec

- Pair's main layout should become a three-panel workbench.
- The left side preserves the current Pair split and remains Pair-owned:
  - left top: agent pane;
  - left bottom: draft pane.
- The right side is a user-owned terminal pane.
- The user terminal starts as an ordinary interactive shell.
- From that terminal, the user can either stay in the shell or open `nvim`
  normally.
- Zellij should remain fully usable from the terminal panel, including creating
  tabs, splitting panes, moving focus, resizing, and other normal zellij
  operations.
- Pair's added workbench shortcuts are pane-local, not raw global zellij
  shortcuts:
  - `Alt+j` moves vertically between the agent and draft panes when focus is in
    the left Pair stack, and has no effect in the right terminal.
  - `Alt+k` moves horizontally between the left Pair stack and the right
    terminal.
  - `Alt+t` creates a zellij tab only when focus is in the right terminal.
  - `Alt+w` closes the active zellij tab only when focus is in the right
    terminal.
  - `Alt+r` renames the active zellij tab only when focus is in the right
    terminal. This must not steal review-pane `Alt+r` reject behavior.
  - `Alt+Shift+C` / `Ctrl+Alt+c` compaction and `Alt+/` scrollback viewer work
    only in the left Pair stack.
- Existing agent/draft behaviors should continue to work: draft send, prompt
  history/future queue, copy-on-select into the draft, scrollback viewer,
  restart/quit flows, and pane/frame metadata.
- The design should be explicit about which pane owns Pair-specific automation
  and which pane is deliberately user-owned terminal space. ARCH-PURPOSE:
  right-terminal shortcuts must be unavailable from the left Pair panes, and
  left-Pair shortcuts must be unavailable from the right terminal.
- Because zellij KDL keybinds are global, pane-local shortcut behavior should
  be implemented at a Pair-owned pane boundary, for example a transparent
  terminal wrapper around the right shell plus existing left-pane handlers,
  rather than by binding every shortcut directly in zellij config. ARCH-DRY:
  reuse the shared `zellijpane` parser for focused-pane classification rather
  than re-open-coding `list-panes` JSON walks.

## Done when

- A normal Pair session opens with the agent pane above the draft pane on the
  left and a user terminal panel available as the other main panel.
- The terminal panel starts in an interactive shell and can launch `nvim`
  without breaking Pair's agent/draft workflow.
- Standard zellij tab and pane operations work from the terminal panel.
- `Alt+t`, `Alt+w`, and `Alt+r` affect zellij tabs from the right terminal and
  do nothing from the agent, draft, scrollback, changelog, or review panes.
- `Alt+j` moves between agent and draft from the left stack and does nothing
  from the right terminal.
- `Alt+k` moves between the left Pair stack and the right terminal.
- `Alt+Shift+C` / `Ctrl+Alt+c` and `Alt+/` work from the left Pair stack and do
  nothing from the right terminal.
- Existing Pair key flows still work from their expected panes.
- Automated layout/config checks cover the changed zellij assets, and manual
  smoke steps record the terminal, `nvim`, and zellij-tab behavior.

## Plan

- [ ] Inspect the existing zellij layout/config ownership and document the
      current agent/draft assumptions.
- [ ] Design the three-panel geometry and focus/keybinding behavior.
- [ ] Update the zellij layout/config and any pane metadata assumptions.
- [ ] Add or update tests/checks for the layout/config assets.
- [ ] Smoke a live Pair session: shell in terminal panel, `nvim` from terminal,
      normal agent/draft send, and zellij tab/pane operations.

## Log

### 2026-07-23

- Created after checking active and punted issues: no existing ticket tracks the
  requested three-panel workbench layout. #82 is only a punted percentage-only
  two-pane layout experiment, and #113 is unrelated.
- Claimed #116 and entered planning. The agreed design is left Pair stack plus
  right user terminal, with pane-gated shortcuts: left-only Pair flows
  (`Alt+j`, `Alt+Shift+C`, `Alt+/`), right-only tab helpers (`Alt+t`, `Alt+w`,
  `Alt+r`), and `Alt+k` as the horizontal bridge. ARCH-PURPOSE rules out global
  zellij binds that fire in the wrong pane; ARCH-DRY points to reusing
  `cmd/internal/zellijpane` for pane classification.
