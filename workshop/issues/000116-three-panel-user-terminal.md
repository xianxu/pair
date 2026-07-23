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
- The left side preserves the current Pair split:
  - left top: agent pane;
  - left bottom: draft pane.
- The remaining large panel is a user terminal pane.
- The user terminal should start as an ordinary interactive shell.
- From that terminal, the user can either stay in the shell or open `nvim`
  normally.
- Zellij should remain fully usable from the terminal panel, including creating
  tabs, splitting panes, moving focus, resizing, and other normal zellij
  operations.
- Existing agent/draft behaviors should continue to work: draft send, prompt
  history/future queue, copy-on-select into the draft, scrollback viewer,
  restart/quit flows, and pane/frame metadata.
- The design should be explicit about which pane owns Pair-specific automation
  and which pane is deliberately user-owned terminal space.

## Done when

- A normal Pair session opens with the agent pane above the draft pane on the
  left and a user terminal panel available as the other main panel.
- The terminal panel starts in an interactive shell and can launch `nvim`
  without breaking Pair's agent/draft workflow.
- Standard zellij tab and pane operations work from the terminal panel.
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
