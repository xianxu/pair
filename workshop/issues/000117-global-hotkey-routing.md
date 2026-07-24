---
id: 000117
status: working
deps: []
github_issue:
created: 2026-07-24
updated: 2026-07-24
estimate_hours: 4.34
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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The calibration source is currently marked
stale, so the estimate is provisional.

```estimate
model: estimate-logic-v3.1
familiarity: 0.90
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.30 impl=0.28
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: lua-neovim design=0.20 impl=0.40
item: api-integration design=0.40 impl=0.40
item: tui-screen design=0.40 impl=0.40
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.08 impl=0.12
total: 4.34
```

## Plan

- [x] Write and approve a durable implementation plan.
- [x] Implement the audited routing with test-first regressions.
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
- Wrote the durable implementation plan at
  `workshop/plans/000117-global-hotkey-routing-plan.md`. It extends the existing
  pure registry, uses pane-id-addressed IO shells, and sequences every behavior
  change test-first.
- Implemented the shared global registry and one `draftroute` Go IO helper for
  both production PTY consumers. Table-driven stream tests cover every
  distinctive sequence, all pane roles, missing-draft failures, and the
  no-byte-leak invariant.
- Replaced KDL focus/write choreography with one forwarded sequence per global
  chord. Added `nvim/workbench_route.lua`; draft executes locally while review,
  scrollback, and change-log initializers discover and address draft by pane id.
- Verification passed: `go test ./... -count=1`, `make test-lua`, both affected
  shell integration suites, runtime-bundle drift check, Zellij config and both
  layout parsers, and `git diff --check`.
- Fresh smoke exposed a performance defect: routing from the right pane waited
  about one second before draft showed the action. Timed the synchronous
  `list-panes --json --command --state` boundary at 0.62–0.84 seconds, making
  pane discovery—not key decoding or confirmation rendering—the root cause.
- Added a validated startup-published draft locator fast path; tests assert a
  valid record performs zero pane-list calls, while stale sessions and dead
  processes fall back safely. The full Go/Lua/shell/runtime/KDL/layout
  verification matrix passes. Enabled and pinned `focus_follows_mouse true` at
  the operator's request.

## Revisions

### 2026-07-24 — Reconcile plan review

The first plan review found that overlay discovery requires Zellij command and
state metadata, and that the wrapper did not yet expose the injected runtime
assumed by the draft plan. Add a shared `draftroute` integration helper used by
both Go wrappers, explicit non-fatal error reporters, the exact `list-panes
--json --command --state` invocation, and separate pure Lua versus process-fake
Neovim tests. The added greenfield helper and wrapper adapter expand the
estimate from 3.20 to 4.02 hours. A second review identified changelog as an
independent Neovim initializer rather than a scrollback bootstrap consumer; the
plan now names and tests `nvim/changelog.lua` directly. The final arithmetic
review restored the calibrated 0.12-hour close-review implementation primitive
and made the distinct Ctrl+Alt+n encoding an explicit mapping/test case in
every Neovim consumer.

The deterministic estimate-reconciliation gate applies the v3.1 formula
`Σdesign×1.30 + Σimpl×familiarity`; with familiarity 0.90, the same primitive
rows produce 4.34 hours rather than their 4.02 raw sum. Correct the declared
total and frontmatter without changing scope.

### 2026-07-24 — Enable pointer-followed focus

The operator reversed the earlier decision to leave Zellij hover focus out of
scope. Set `focus_follows_mouse true` and pin it in the workbench configuration
regression. This is independent of the pane-id routing invariant: global
hotkeys remain deterministic regardless of which pane the pointer focuses.

### 2026-07-24 — Expand the right terminal to three-quarters

Change the Alt+Shift+Return expanded geometry from two-thirds to three-quarters
of the workbench. Preserve the exact one-action resize and filler-anchored
half-width collapse behavior.

### 2026-07-24 — Revert pointer-followed focus

Live smoke showed Zellij hover focus crosses from the tiled left layer into the
pinned floating terminal but does not cross back symmetrically. Revert to
explicit `focus_follows_mouse false`; click and Pair's pane-focus shortcuts
remain predictable in both directions.

### 2026-07-24 — Focus draft for confirmation globals

Pane-id routing guarantees delivery but leaves confirmation dialogs hidden when
another pane is focused. Before invoking `PairConfirmDetach`,
`PairConfirmQuit`, `PairConfirmRestart`, or `PairConfirmAgentRestart`, focus the
validated draft pane by id. Layout growth/shrink and review toggle remain
focus-preserving. The shared shortcut decision must carry this policy so agent,
terminal, and Neovim-overlay consumers cannot drift.
