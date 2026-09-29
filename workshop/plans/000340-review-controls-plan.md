# Review controls implementation plan

> For agentic workers: follow AGENTS.md §3. The operator approved implementation
> of the existing issue Spec. This records the expanded integration design once
> Couch routing and generated help made the change exceed the small-diff shell.

**Goal:** make review controls discoverable and usable in Pair and Couch.
**Architecture:** retain Neovim buffer-local actions; derive help from their
mapping descriptions. Reuse Couch's existing focus observation for a positively
identified review role. Return to the agent by command identity and pane ID.
**Tech Stack:** Lua/Neovim, Go, shell integration tests, Zellij.

## Core concepts

| Name | Lives in | Status |
|---|---|---|
| PaneRole | cmd/internal/workbenchshortcut/shortcut.go | modified |
| Review help catalog | cmd/internal/keyhelp/review.go | new |
| Keymap description parser | cmd/internal/keyhelp/parse.go | modified |

PaneRole adds review to the existing command-derived roles (ARCH-PURPOSE).
The review catalog owns display/grouping, while descriptions stay with the Lua
mappings (ARCH-DRY); a drift test rejects unmapped or undescribed bindings.
No new persistent artifacts, caches or background processes are created.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Review control maps | nvim/review.lua | modified | Neovim modes, windows and statusline |
| Agent return | nvim/pair_poke.lua | modified | Zellij pane discovery, hide and focus |
| Shortcut focus probe | cmd/internal/couchcmd/shortcut_focus.go | modified | exact client's focused pane |
| Console input dispatch | cmd/internal/couchtty/console.go | modified | raw key bytes and relaunch handler |

Agent return reuses the poke's agent resolver, with positive `pair wrap` command
identity instead of non-draft titles. The stateful Zellij test host keeps floating
visibility and focused pane; a right terminal before the agent must not change
which pane receives focus. Focus failures must not silently focus another pane.
Couch passes Alt+n to review and both relaunch chords to the right terminal;
Ctrl+Alt+n still relaunches from review, and draft/agent/switcher keep relaunch.

## Tasks

- [x] Write a failing headless test in tests/review-controls-test.sh for idle and
  awaiting hints, real map invocation, wrapped marker jumps, normal Esc return,
  insert/visual Escape, and popup-first dismissal. Run with bash.
- [x] Add buffer-local Esc/M-c/M-n/M-N and status hints in nvim/review.lua.
  Keep existing diagnostic popup behavior, adding focused-popup Esc dismissal.
- [x] Add positive agent discovery and return in nvim/pair_poke.lua; prove the
  reordered-terminal regression fails before changing identity selection.
- [x] Extend keyhelp parsing/catalog with review descriptions, including visual
  Alt+q and explicit definition alias classification; test missing and stale maps.
- [x] Write failing actual-focus-probe and Console.Run byte routing tests; extend
  the existing probe return type to PaneRole and preserve all other routing.
- [x] Update README and atlas/review-workbench.md with modes and host exceptions.
- [x] Run `make test-review` and affected Go packages with race detection;
  regenerate embedded runtime via `make build` before keyhelp drift checks.
- [x] Commit the implementation and build pair:0; verify branch/HEAD unchanged.
- [x] Operator smoke in both standalone Pair and restarted Couch: hints, Alt+h,
  Esc agent return, insert/visual Esc, diagnostic popup Esc, and marker
  next/previous wrapping. In a disposable conversation, verify Alt+Shift+N
  from the draft still restarts the agent.

## Review

Ad-hoc fresh-context review found the ambiguous pre-existing agent resolver;
fixed with command identity and retained the failing reordered-pane fixture.
SDLC close remains the final review boundary after operator smoke; do not close
or land solely from these implementation checks.

## Revisions

- 2026-09-28 — operator smoke feedback changes return destination to draft,
  adds Alt+Return submission to the review statusline, and adds Alt+c review
  to the draft bar (including active review). Share draft identity discovery
  with workbench_route; keep agent poke identity separate. Updated controls,
  toggle and draft-status regressions pass. Earlier agent-return statements
  describe the initial implementation, superseded by this revision.

- 2026-09-28 — viewer exit integration: register a shared VimLeave callback
  for interactive scrollback/changelog to hide all floating panes and focus
  draft after annotate's VimLeavePre sidecar emission. Headless unit sessions
  do not manipulate the live host. Add a stateful stacked-overlay regression
  covering both real viewer initializations and preserved annotation ordering.

- 2026-09-28 — operator confirmed revised smoke worked and authorized close/land.
