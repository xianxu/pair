---
id: 000116
status: working
deps: []
github_issue:
created: 2026-07-23
updated: 2026-07-23
estimate_hours: 4.53
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
- The right terminal's intended zellij affordance is the small Pair-provided
  tab vocabulary below. This issue does not restore zellij's mode-switch
  defaults or promise every stock zellij pane/resize binding; those remain
  governed by Pair's existing quiet-zellij config.
- Pair's added workbench shortcuts are pane-local, not raw global zellij
  shortcuts:
  - `Alt+j` moves vertically between the agent and draft panes when focus is in
    the left Pair stack, and has no effect in the right terminal.
  - `Alt+k` moves horizontally between the left Pair stack and the right
    terminal. Returning from the right terminal focuses the last left Pair pane
    that had focus; if no left focus has been recorded yet, it falls back to the
    draft pane.
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
- The right terminal remains an ordinary shell, so users can run `nvim` or any
  other terminal program there.
- `Alt+t`, `Alt+w`, and `Alt+r` affect zellij tabs from the right terminal and
  do nothing from the agent, draft, scrollback, changelog, or review panes.
- `Alt+j` moves between agent and draft from the left stack and does nothing
  from the right terminal.
- `Alt+k` moves from the focused left Pair pane to the right terminal, then
  returns from the right terminal to the same left pane; before any recorded
  left focus exists, it returns to the draft pane.
- `Alt+Shift+C` / `Ctrl+Alt+c` and `Alt+/` work from the left Pair stack and do
  nothing from the right terminal.
- Existing Pair key flows still work from their expected panes.
- Automated layout/config checks cover the changed zellij assets, and manual
  smoke steps record the terminal, `nvim`, and zellij-tab behavior.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 0.80
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.30 impl=0.28
item: greenfield-go-module design=0.30 impl=0.28
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: lua-neovim design=0.20 impl=0.40
item: tui-screen design=0.40 impl=0.40
item: api-integration design=0.40 impl=0.40
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.08 impl=0.12
total: 4.53
```

## Plan

- [x] Inspect the existing zellij layout/config ownership and document the
      current agent/draft assumptions.
- [x] Design the three-panel geometry and focus/keybinding behavior.
- [x] Update the zellij layout/config and any pane metadata assumptions.
- [x] Add or update tests/checks for the layout/config assets.
- [ ] Smoke a live Pair session: shell in terminal panel, `nvim` from terminal,
      normal agent/draft send, and right-terminal zellij tab helpers.

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
- Spec review found two Important clarity gaps: `Alt+k` needed a concrete return
  target, and "normal zellij operations" over-promised against Pair's existing
  locked-normal zellij config. Resolved by specifying last-left-pane return
  semantics for `Alt+k` and narrowing the right terminal contract to the
  explicit Pair tab helpers.

### 2026-07-24

- Implemented the three-pane workbench shape: zellij layout now keeps
  agent/draft as the left stack and starts a right-side `pair term` user
  terminal. Swap layouts preserve all three panes while resizing only the draft
  rung.
- Moved pane-local shortcuts out of global zellij KDL. `pair wrap` owns
  agent-pane `Alt+j`/`Alt+k`/`Alt+/`/compaction and swallows right-tab helpers;
  `nvim/init.lua` owns the draft equivalents and records the last left pane;
  `pair term` owns right-terminal `Alt+t`/`Alt+w`/`Alt+r` and right-to-left
  `Alt+k` return. `nvim/review.lua` keeps pane-local `Alt+r` reject.
- Added pure workbench shortcut decisions and sidecar helpers in
  `cmd/internal/workbenchshortcut`, the right-terminal wrapper in
  `cmd/internal/termcmd`, wrapper regression coverage for no-remap and split
  escape chunks, and `tests/term-pane-shortcuts-test.sh` for fake-zellij pane
  gating.
- Verification passed: focused Go packages
  (`workbenchshortcut`, `termcmd`, `wrapcmd`, `dispatcher`, `pair-go`,
  `zellijpane`), `bash tests/term-pane-shortcuts-test.sh`,
  `bash tests/copy-on-select-test.sh`, `bash tests/review-toggle-test.sh`,
  `zellij --config-dir zellij setup --check`,
  `zellij setup --dump-layout zellij/layouts/main.kdl`,
  `make runtimebundle-drift-check`, `make test-lua`, `git diff --check`, and
  `go test ./... -count=1`.
- Live nested Pair smoke is still unchecked in this noninteractive run; the
  automated fake-zellij test covers the shortcut decisions and zellij asset
  validation covers the static layout/config.
