---
id: 000116
status: working
deps: []
github_issue:
created: 2026-07-23
updated: 2026-07-24
estimate_hours: 7.02
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
- The right terminal provides Pair-owned local terminal tabs inside that pane.
  Outer Zellij tabs are deliberately not used because they replace the whole
  workbench rather than tabbing only the user-terminal surface. This issue does
  not restore Zellij's mode-switch defaults or promise every stock Zellij
  pane/resize binding; those remain governed by Pair's quiet config.
- Pair's added workbench shortcuts are pane-local, not raw global zellij
  shortcuts:
  - `Alt+j` moves vertically between the agent and draft panes when focus is in
    the left Pair stack, and has no effect in the right terminal.
  - `Alt+k` moves horizontally between the left Pair stack and the right
    terminal. Returning from the right terminal focuses the last left Pair pane
    that had focus; if no left focus has been recorded yet, it falls back to the
    draft pane.
  - `Alt+t` creates a local terminal tab only when focus is in the right terminal.
  - `Alt+w` closes the active local terminal tab only when focus is in the right
    terminal.
  - `Alt+r` renames the active local terminal tab only when focus is in the right
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
- Pair supports both workbench topologies:
  - `pair` and `pair --layout2` select the original agent/draft-only topology
    when a tag has no recorded layout;
  - `pair --layout3` selects the three-pane topology described above;
  - layout flags are Pair-owned, may appear before or after the agent name but
    before `--`, and are never forwarded to the coding agent;
  - when no layout flag is typed, Pair reuses the tag's recorded topology;
  - an explicit flag overrides the recorded topology. If that tag is live and
    the topology differs, Pair must confirm before restarting the whole
    workbench because arbitrary user-terminal state cannot be recovered.

## Done when

- A new unrecorded Pair tag without a layout flag opens the original two-pane
  agent/draft workbench.
- `pair --layout3` opens the agent/draft stack on the left with a user terminal
  panel on the right, and a recorded layout-3 tag resumes that topology when no
  layout flag is supplied.
- The terminal panel starts in an interactive shell and can launch `nvim`
  without breaking Pair's agent/draft workflow.
- The right terminal remains an ordinary shell, so users can run `nvim` or any
  other terminal program there.
- `Alt+t`, `Alt+w`, and `Alt+r` affect Pair-owned local terminal tabs from the
  right terminal and do nothing from the agent, draft, scrollback, changelog,
  or review panes.
- `Alt+j` moves between agent and draft from the left stack and does nothing
  from the right terminal.
- `Alt+k` moves from the focused left Pair pane to the right terminal, then
  returns from the right terminal to the same left pane; before any recorded
  left focus exists, it returns to the draft pane.
- `Alt+Shift+C` / `Ctrl+Alt+c` and `Alt+/` work from the left Pair stack and do
  nothing from the right terminal.
- Existing Pair key flows still work from their expected panes.
- A new unrecorded tag defaults to the two-pane topology; a recorded tag
  resumes its recorded topology; an explicit conflicting layout changes a live
  tag only after operator confirmation.
- `pair claude --layout2 -- --other-claude-flags other-claude-params` consumes
  `--layout2` itself and forwards only the tokens after `--` to Claude.
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
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.20 impl=0.28
item: api-integration design=0.20 impl=0.24
item: cross-cutting-refactor design=0.20 impl=0.20
item: tui-screen design=0.20 impl=0.30
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.08 impl=0.12
total: 7.02
```

## Plan

- [x] Inspect the existing zellij layout/config ownership and document the
      current agent/draft assumptions.
- [x] Design the three-panel geometry and focus/keybinding behavior.
- [x] Update the zellij layout/config and any pane metadata assumptions.
- [x] Add or update tests/checks for the layout/config assets.
- [x] Add pure layout parsing/precedence plus tag-scoped topology persistence.
- [x] Split the original and layered workbenches into validated layout 2 and
      layout 3 assets, with layout 2 as the unrecorded default.
- [x] Implement confirmed live-layout override and preserve topology across
      attach, quit, restart, compaction, and rename.
- [x] Update embedded assets and architecture consumers for both modes.
- [x] Smoke a live Pair session: shell in terminal panel, `nvim` from terminal,
      normal agent/draft send, right-terminal local tab helpers, two-pane
      default, recorded resume, and explicit live override.

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

- Extended the approved design to support both topologies. The original
  two-pane workbench is the default for unrecorded tags; the three-pane
  workbench is opt-in via `--layout3`. Omitted flags reuse a tag-scoped
  `workbench-layout` record, while an explicit conflicting flag requires a
  confirmed whole-workbench restart. The record is deliberately distinct from
  the existing `layout-mode` draft-height diagnostic (`ARCH-DRY`,
  `ARCH-PURPOSE`).
- Recalibrated the estimate from 4.53h to 7.02h after the topology-selection
  revision added launcher parsing, durable persistence, live Zellij probing,
  confirmed relaunch, dual embedded assets, and a second operator smoke pass.
  The added primitives use the same v3.1 Method A calibration as the original
  estimate; its source currently reports stale pending the fleet recalibration
  tracked in ariadne#127.
- Implemented the approved dual-topology contract. `LayoutMode` is a pure
  selection model; `workbench-layout-<tag>` is the durable topology record;
  Pair-owned flags are consumed before `--`; and live explicit conflicts take
  a confirmed nonterminal teardown/relaunch path. Missing rollout records are
  classified from a session-scoped Zellij pane report. This keeps policy in the
  pure core and Zellij/filesystem/prompt effects at injected boundaries
  (`ARCH-PURE`, `ARCH-DRY`).
- Split the validated static products into `main-2.kdl` and `main-3.kdl`,
  requiring both in source and embedded-runtime asset roots. Structural tests
  compare their shared agent/draft commands while preserving their deliberate
  topology difference (`ARCH-PURPOSE` shadow sweep).
- Fresh automated verification passed: `make runtimebundle-drift-check`,
  `go test ./... -count=1`, `make test-lua`,
  `bash tests/term-pane-shortcuts-test.sh`,
  `bash tests/pair-embedded-runtime-test.sh`, `bash tests/pair-rename.sh`, and
  `git diff --check`. Both KDL layouts and the Zellij config also passed the
  installed Zellij parser/checker earlier in this implementation window.
- Rebuilt `bin/pair` for live testing. `~/.local/bin/pair` is a symlink to that
  exact workspace binary, so `make install` correctly rebuilt the live target
  but macOS `install` then reported its same-file guard; `command -v pair`
  resolves to `/Users/xianxu/workspace/pair/bin/pair`, whose help shows the new
  layout flags.
- Operator smoke confirmed the installed dual-topology behavior works and
  approved landing.
- First close review returned REWORK. It found one production bug in the
  terminal byte pump: a shortcut or SGR mouse event coalesced with following
  shell bytes was forwarded as one opaque chunk. Added red/green regressions
  and a prefix-consuming stream loop that handles the event while preserving
  the remaining payload. Reconciled the durable plan's stale pre-revision
  outer-Zellij-tab paths with the delivered local PTY tabs, and documented the
  public layout flags/topology-specific keys in README. The review's
  clean-`HEAD` embedded-asset and unchecked-plan observations came from the
  pre-landing commit state; the generated assets and completed plan are present
  in the working transaction that will become the reviewed landing commit.

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
- Live dogfood found that right-terminal `Alt+t` used bare
  `zellij action new-tab`, which could instantiate the session's Pair layout
  and make the agent panes look restarted. Fixed `Alt+t` to create a new tab
  whose initial command is `pair term`, so it opens another plain terminal tab
  with the same right-terminal shortcut wrapper instead of another Pair
  workbench. Regression coverage updated in `cmd/internal/termcmd` and
  `tests/term-pane-shortcuts-test.sh`.
- Second dogfood pass showed the running zellij session still had the old
  two-pane `new_tab_template`; source KDL changes do not mutate an already
  running zellij session. Also hardened future sessions by making zellij
  forward workbench chords (`Alt+j/k/t/w/r//`, `Alt+Shift+C`, `Ctrl+Alt+c`) as
  literal Meta bytes to the focused pane process instead of relying on default
  unbinds. That keeps the behavior pane-local while overriding zellij's own
  shared defaults such as `Alt+j`/`Alt+k`.
- Follow-up dogfood from a fresh `pair-dev codex -- --sandbox
  danger-full-access` session showed `Alt+t` could still produce a new Pair-like
  workbench tab. The stale-session explanation was incomplete. Reproduced the
  zellij boundary in a detached probe session and verified that
  `new-tab --layout-string` with a one-pane `pair term` layout creates exactly
  one terminal pane. Updated right-terminal `Alt+t` to use that explicit layout
  string so it cannot inherit the session's tab template and respawn
  agent/draft panes.
- Additional dogfood found two right-terminal regressions. `Alt+x` still used
  the old two-pane `MoveFocus Down` route and wrote the nvim quit command into
  the terminal; zellij's nvim-routed global binds now move left before moving
  down so the draft receives the command from either side of the three-pane
  workbench. `Alt+r` no longer prompts inside the terminal wrapper; it opens a
  small floating `pair term rename-tab-prompt` pane that reads the new name and
  calls `zellij action rename-tab`, keeping shell input separate from Pair's tab
  helper.
- Follow-up dogfood clarified that right-terminal `Alt+t` must stay inside the
  right pane instead of creating a whole-session zellij tab. Changed `Alt+t` to
  create a stacked terminal pane near the current right terminal, and changed
  `Alt+w` to close the focused terminal pane. Also fixed the rename prompt
  launch to call top-level `zellij run`; `run` is not a `zellij action`
  subcommand in zellij 0.44.3.
- Final right-pane tab correction: outer zellij cannot provide true tabs inside
  only the right pane; its tabs replace the whole workbench, and stacked panes
  are not the requested tab model. `pair term` now runs an inner zellij session
  in the right pane with a minimal terminal config. The outer wrapper still owns
  `Alt+k`/`Alt+j` pane boundaries, while `Alt+t`, `Alt+w`, and `Alt+r` pass
  through to the inner zellij, whose config binds them to native tab create,
  close, and rename.
- Dogfood screenshot showed the inner zellij default layout was visually noisy
  inside Pair's right pane. Added a minimal inner layout with only a shell pane
  and changed `pair term` to attach to an existing inner session or create a new
  one with that layout, avoiding nested default tab/status/plugin chrome.
- Follow-up screenshot still showed nested chrome after creating right-pane
  tabs. Fixed the inner terminal config to disable pane frames and set
  `default_layout "main"`, so native `Alt+t` tabs also use Pair's quiet inner
  layout instead of zellij's built-in default layout.
- Another dogfood pass still showed the old nested chrome because `pair term`
  reattached to an already-created inner terminal session. Zellij applies
  `pane_frames` and `default_layout` at session creation, so the existing
  `pair-<tag>-terminal` sessions masked the config fix. Bumped the inner
  terminal session suffix to `-terminal-v2` so new Pair runs create a fresh
  quiet inner session without manual cleanup.
- Follow-up screenshots showed nested zellij remained visually unacceptable
  even with quiet config. Abandoned the inner-zellij approach. `pair term` now
  runs shell PTYs directly again and owns local `Alt+t`/`Alt+w`/`Alt+r` tab
  actions in-process, leaving the right pane visually as a normal Pair terminal.
- Added right-terminal tab navigation on top of the local tab layer:
  `Alt+Left`/`Alt+Right` switch to the previous/next local terminal tab, and
  `pair term` draws a one-line tab strip with SGR mouse tracking so clicking a
  tab label on that first row switches the active PTY.
- Follow-up dogfood showed the SGR mouse mode needed for tab-strip clicks also
  captured wheel events before zellij could use them for shell scrollback.
  `pair term` now maps wheel-up/wheel-down mouse events to zellij
  `scroll-up`/`scroll-down`, while keeping non-wheel shell-row mouse events as
  PTY input. ARCH-PURE: the mouse parser stays unit-tested separately from the
  IO action dispatch.
- Follow-up dogfood showed that unconditional wheel-to-zellij routing broke
  mouse scroll inside nvim running in the right terminal. `pair term` now tracks
  child DEC private mouse mode enable/disable sequences per local tab; wheel
  events scroll zellij only for plain shells, and pass through when the active
  PTY app has requested mouse input.
- Added `Alt+Shift+Return` as a focused-side width toggle for the three-panel
  workbench. zellij forwards a distinct KKP sequence (`ESC[13;4u`) so the
  chord no longer collapses to Return in the right terminal; Pair then applies
  a geometry-derived override layout where the focused side toggles between the
  default split and 67% width, preserving the current draft height rung.
- Corrected the width toggle after dogfood: retiling the whole workbench shrank
  the left panes, which violated the desired "right terminal floats over the
  left stack" behavior. `Alt+Shift+Return` is now right-terminal-only: embedded
  terminal panes become a pinned floating overlay at `x=33%`, `width=67%`,
  `height=100%`; pressing it again embeds the terminal back without resizing
  the left panes. Draft nvim keeps its prior `Alt+Shift+Return` send-without-
  submit behavior.
- Follow-up dogfood showed zellij did not restore the tiled terminal to the
  original half-width split after embedding it back from the floating overlay.
  The collapse half now embeds the terminal and then applies the balanced
  three-pane layout, making `Alt+Shift+Return` a real expand/collapse toggle.
- Follow-up dogfood showed the collapse chord still did not fire from the
  floating terminal because the shared workbench classifier deliberately treats
  floating panes as non-workbench panes. `pair term` now handles
  `Alt+Shift+Return` before that classifier, so the right-terminal process can
  collapse its own floating pane. The overlay is also borderless to avoid a
  double frame around `pair term`'s local tab strip.
- Follow-up dogfood showed two more right-terminal edge cases. `Alt+x` still
  used zellij-side focus moves plus `WriteChars ":lua PairConfirmQuit()"`, so a
  focused right terminal could receive the literal Lua command in the shell.
  zellij now forwards `Alt+x` as `ESC[120;3u`; `pair term` routes that chord to
  the draft nvim quit prompt, `pair wrap` handles the same chord from the agent
  pane, and draft nvim maps `<M-x>` directly. The expanded terminal toggle also
  now recognizes tab-strip titles like `terminal 1 [work]` and uses integer
  screen geometry when zellij reports it, avoiding percentage rounding at the
  right edge.
- Verification for this pass: red tests first for missing `Alt+x` decode,
  terminal routing, wrap translation, and geometry rounding; then green
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd
  ./cmd/internal/layoutcmd ./cmd/internal/wrapcmd -count=1`,
  `bash tests/term-pane-shortcuts-test.sh`,
  `bash tests/review-toggle-test.sh`,
  `zellij --config-dir zellij setup --check`,
  `go test ./... -count=1`, `bash tests/queue-send-test.sh`,
  `bash tests/review-window-test.sh`, `bash tests/scrollback-open-test.sh`,
  `make runtimebundle-drift-check`, and `git diff --check`.
- Further live smoke identified the width-toggle root causes: expansion
  explicitly requested a borderless floating pane, and collapse used
  `override-layout`, allowing Zellij to reconstruct the tab instead of
  preserving its processes. Replaced the collapse re-layout with localized,
  pane-id-targeted resize reconciliation after embedding the same terminal
  pane, and retained the floating pane frame. ARCH-PURE: width reconciliation
  derives only from reported pane geometry; ARCH-PURPOSE: the toggle changes
  geometry without changing pane identity or process ownership.
- The next smoke pass exposed a deeper Zellij constraint: embedding the same
  pane preserves its process but not its original position in the split tree.
  Zellij reinserted the terminal beside the agent, leaving Neovim spanning the
  bottom. Removed floating/embed operations entirely. Both toggle directions
  now reconcile the tiled terminal's left boundary in place—balanced to 2/3
  when expanding and 1/2 when collapsing—so topology, pane IDs, processes, and
  frames remain unchanged.
- Follow-up smoke showed collapse stopping with a visibly larger right pane.
  Root cause was the 5% target tolerance, which accepted the split one discrete
  Zellij resize step early. Tightened the tolerance to 1% and added closest-step
  rollback when a resize overshoots. Regression coverage includes the observed
  early-stop geometry and an expansion overshoot.
- Replaced the checkpointed tiled-resize experiment with the operator's
  two-layer model. The tiled base is a half-width agent/draft stack plus an
  inert borderless filler; a permanently floating terminal exactly covers the
  filler at 50% and overlays the right third of the left stack at 67%.
  Alt+Shift+Return now performs one precise coordinate update, with no resize
  loop, embed, retiling, or process replacement. Floating terminal role
  classification is explicit while unrelated floating review panes remain
  outside the workbench role set. ARCH-PURPOSE: the filler exists only to keep
  the left stack laid out at its visible width beneath the overlay.
- Live smoke exposed two real-report details absent from the first fixture:
  Zellij marked both draft and floating terminal focused, and on an odd-width
  screen `x=50%` rounded one column left of the filler's boundary. Terminal
  discovery now scans past other focused panes; terminal startup and collapse
  anchor normal geometry to the filler's reported `pane_x`. A live expand/
  collapse probe produced `(x=57,width=114)` then `(x=86,width=85)` in one
  action each, preserving the left frame.
- Added agent-only context refresh for the now-long-lived workbench.
  Alt+Shift+N confirms in the draft and runs `pair agent restart`, which signals
  the stable `pair wrap` supervisor. The wrapper terminates only its agent PTY
  child, reconstructs arguments through the launcher's canonical persistence
  transform (dropping resume/session bindings), refreshes Claude or async
  Codex/agy session tracking, and `exec`s itself in the same Zellij pane.
  Alt+N remains the explicit whole-workbench reload. Process-level coverage
  verifies SIGUSR2 reaches a replacement wrapper invocation while preserving
  user-authored flags and removing the prior session ID.
- Removed the redundant in-pane terminal tab strip now that local-tab state is
  carried by the floating pane frame title. The strip had reserved one PTY row
  and globally enabled SGR mouse reporting; after resize, a device-attributes
  response combined with mouse press/release bytes could bypass the single-
  event parser and appear as literal shell input. Child PTYs now use the full
  pane height, no synthetic mouse mode is enabled, and non-wheel mouse input is
  passed directly to applications that request it.
- Disabled Zellij's startup-tip popup with its supported
  `show_startup_tips false` option. The right terminal intentionally remains
  pinned; Zellij exposes no deterministic arbitrary z-order between a pinned
  pane and startup popup, so suppressing the redundant tip is the direct fix.
