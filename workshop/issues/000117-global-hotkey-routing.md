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
- Audit every shortcut bound in `zellij/config.kdl` and classify it as
  draft-routed global, direct global, or focused-process/pane-local. Existing
  Neovim-only editor/viewer keymaps documented in README remain pane-local and
  are outside this routing change.
- The Zellij-defined classifications are:
  - draft-routed globals: Alt+d, Alt+x, Alt+n/Ctrl+Alt+n, Shift+Alt+N,
    Alt+Up, Alt+Down, and Alt+c;
  - direct global surfaces: Alt+h and Alt+l, which open independent floating
    panes and do not execute inside draft Neovim;
  - focused-process/pane-local actions: Alt+j/k/t/w/r, Alt+/,
    Alt+Shift+C/Ctrl+Alt+c compaction, and Alt+Shift+Return; these remain scoped
    to their owning pane.
- The draft-routed mapping is authoritative:

  | Chord | Shared action | Draft Lua target |
  |---|---|---|
  | Alt+d | confirm detach | `PairConfirmDetach` |
  | Alt+x | confirm quit | `PairConfirmQuit` |
  | Alt+n, Ctrl+Alt+n | reload Pair | `PairConfirmRestart` |
  | Shift+Alt+N | restart supervised agent | `PairConfirmAgentRestart` |
  | Alt+Up | grow draft | `PairLayoutBigger` |
  | Alt+Down | shrink draft | `PairLayoutSmaller` |
  | Alt+c | toggle review | `PairReviewToggle` |

  Aliases resolve to one shared action. `PairConfirmRestartNewSession` is not
  part of this mapping.
- Zellij bindings for draft-routed globals must only forward a distinctive key
  sequence to the focused pane. They must not combine directional focus changes
  with writes.
- The agent wrapper and right-terminal wrapper must decode those sequences
  through the shared `workbenchshortcut` model and route the corresponding Lua
  function to the draft pane by stable pane id.
- Draft Neovim invokes the functions locally. Pair-owned Neovim overlays—the
  review pane and the shared scrollback/change-log viewer runtime—route all
  seven actions to the draft. Alt+c is resolved by the draft's authoritative
  `PairReviewToggle`, including hiding a live review pane; the review pane does
  not retain a separate hide-self interpretation.
- Draft discovery uses current `zellij action list-panes` metadata and
  `RoleForPane`, never relative `MoveFocus` or saved coordinates. Every routed
  write is addressed to that discovered pane. If discovery or routing fails,
  the chord is swallowed and the failure is surfaced or logged; raw chord,
  control, or Lua bytes must never reach the focused application.
- One shared action-to-draft-function mapping must drive the Go wrappers
  (`ARCH-DRY`). Extend `workbenchshortcut`'s existing decoding, `RoleForPane`,
  and `Decide` model rather than creating a parallel registry. The pure core
  returns the classification, action, and Lua target; pane discovery and
  Zellij calls stay in thin injected runtime seams (`ARCH-PURE`).
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
- Review and scrollback/change-log Neovim overlays route the same global set to
  the draft rather than swallowing it or executing draft-only behavior locally.
- Global routing uses stable draft-pane discovery and contains no
  `MoveFocus`-then-write KDL choreography.
- Direct-global and pane-local shortcuts retain their existing behavior.
- Table-driven pure tests cover every `(global action, pane role)` decision.
- Injected-runtime tests feed each distinctive sequence through the production
  agent and right-terminal stream decoders, assert the child PTY receives no
  chord/Lua/control bytes, and assert writes target the discovered draft pane.
  Headless Neovim tests cover draft-local and overlay-routed behavior. Static
  KDL inventory tests supplement but do not replace these behavioral tests.

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
- Fresh spec review narrowed the mechanical audit to Zellij-defined bindings,
  made the chord-to-Lua mapping authoritative, named review and
  scrollback/change-log consumers, required safe failure behavior, and expanded
  production-path non-leakage coverage. ARCH-DRY: extend the existing shared
  shortcut model; ARCH-PURE: keep pane discovery/writes injected; ARCH-PURPOSE:
  cover every KDL consumer rather than only the reported Alt+n path.
