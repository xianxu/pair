# Split Right Terminal Pane Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `Alt+Shift+d` as a terminal-local layout-3 shortcut that creates a Zellij top/bottom split in the right terminal area and focuses the new lower pane.

**Architecture:** Reuse the existing terminal-local shortcut pipeline (`workbenchshortcut` → `pair term` stdin pump → injected `Runtime.RunZellijAction`) so the behavior stays inside the current shortcut ownership model (`ARCH-DRY`). Use Zellij-native panes for the split, not Pair's internal terminal-tab mux, so Zellij owns mouse boundary resizing (`ARCH-PURPOSE`). The split action must create the same Pair terminal command shape as `zellij/layouts/main-3.kdl`: `zellij action new-pane --direction down --name terminal -- sh -c 'zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term'`.

**Tech Stack:** Go terminal command routing, generated workbench shortcut registry, Zellij KDL config/layouts, shell integration tests.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `ChordAltShiftD` | `cmd/internal/workbenchshortcut/shortcut.go` | new |

- **ChordAltShiftD** — canonical representation of the `Alt+Shift+d` byte sequence.
  - **Relationships:** N:1 with terminal routing tests and generated Neovim action metadata.
  - **DRY rationale:** Keeps shortcut bytes in the existing registry instead of duplicating raw escape sequences in each consumer.
  - **Future extensions:** Other right-pane management shortcuts should join this registry.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `TerminalSplitDownAction` | `cmd/internal/termcmd/run.go` | new | `zellij action new-pane` |
| `ZellijMouseResizeConfig` | `zellij/config.kdl` | modified | Zellij mouse pane resize behavior |

- **TerminalSplitDownAction** — terminal-local handler that invokes Zellij to split the focused right terminal pane downward.
  - **Injected into:** `pumpStdinWithTimer` through the existing `Runtime` fake.
  - **Action contract:** `RunZellijAction("new-pane", "--direction", "down", "--name", "terminal", "--", "sh", "-c", rightTerminalPaneShell)`.
  - **Command contract:** `rightTerminalPaneShell` is the same shell string used by layout 3's right terminal pane: `zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`.
  - **Mouse contract:** do not pass `--borderless true`; Zellij pane borders must remain available for mouse boundary dragging.
  - **Future extensions:** Adjacent right-pane management actions such as close split or move focus between split panes.
- **ZellijMouseResizeConfig** — the minimal config needed so real Zellij pane boundaries remain mouse-draggable.
  - **Injected into:** Zellij at session start through the existing config file.
  - **Config contract:** do not set `mouse_mode false`; keep `focus_follows_mouse false`. `advanced_mouse_actions false` may remain because the default config documents it as hover/grouping behavior, not basic pane-boundary resizing.
  - **Future extensions:** Only widen if a live smoke proves boundary drag still cannot work.

## Task 1: Pin Shortcut Routing

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Modify: `cmd/internal/workbenchshortcut/shortcut_test.go`
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Write failing registry/routing tests**

Add coverage that `Alt+Shift+d` decodes through the shared shortcut registry and that `pair term` maps it to `RunZellijAction("new-pane", "--direction", "down", "--name", "terminal", "--", "sh", "-c", rightTerminalPaneShell)`.

- [x] **Step 2: Verify RED**

Run:

```bash
go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1
bash tests/term-pane-shortcuts-test.sh
```

Expected: fail because the chord/action is not registered or routed.

- [x] **Step 3: Implement minimal routing**

Add the chord to the registry, define the right-terminal shell command once in `cmd/internal/termcmd/run.go`, and route it in `handleTerminalChord` to the injected Zellij runtime action.

- [x] **Step 4: Verify GREEN**

Run the same commands. Expected: pass.

## Task 2: Preserve Layout And Mouse Resize Behavior

**Files:**
- Modify: `zellij/config.kdl`
- Modify: `zellij/layouts/main-3.kdl` if the split action needs a command-compatible terminal pane shape.
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Write failing config/layout assertions**

Add shell assertions that `Alt+Shift+d` is terminal-local, the split action creates a named `pair term` pane rather than a raw shell, no split pane is borderless, and the config leaves Zellij pane boundary dragging enabled without enabling focus-follows-mouse.

- [x] **Step 2: Verify RED**

Run:

```bash
bash tests/term-pane-shortcuts-test.sh
zellij --config-dir zellij setup --check
```

Expected: fail until the config/action is updated.

- [x] **Step 3: Implement minimal config/layout changes**

Prefer Zellij's normal pane splitting and mouse boundary resizing. Keep `focus_follows_mouse false`, keep the new pane bordered, and avoid setting `mouse_mode false`.

- [x] **Step 4: Verify GREEN**

Run the same commands. Expected: pass.

## Task 3: Docs And Final Verification

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000123-split-the-right-terminal-pane-with-alt-shift-d.md`

- [x] **Step 1: Document the keybinding**

Add the new `Alt+Shift+d` right-terminal split behavior to README and atlas.

- [x] **Step 2: Run complete verification**

Run:

```bash
go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1
go test ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
zellij --config-dir zellij setup --check
git diff --check
```

Expected: all pass.

- [x] **Step 3: Record evidence and commit**

Update the issue log with test evidence and commit the implementation.

## Revisions

### 2026-07-27 — tiled right terminal pivot

**Reason:** Live dogfooding showed the floating split panes can be dragged off
position by their frames. Zellij 0.44.3 source-verified: floating frame-drag
move has no config gate, and every mitigation so far traded away something
(`mouse_mode false` → focus-lockout rescue lost; borderless → no divider/title).
User rejected the residual drag exposure and chose the structural fix: move the
right terminal from the floating layer into the tiled tree. Tiled panes have no
mouse-move operation, so drag-immunity comes from the architecture, keeping
frames (divider, #118 tab title, scroll indicator) and full mouse support
(`ARCH-PURPOSE`: this satisfies the original "Done when: dragging does not move
the layout" line instead of the logged relaxation).

**Delta:** Chunk 2 below replaces the floating split/expand machinery:
- `main-3.kdl`: right terminal becomes a real tiled pane; `terminal-filler` and
  the `floating_panes` block are deleted (removes the filler focus-trap class
  entirely); swap layouts gain 4-pane variants for the split state.
- `splitTerminalDown`: explicit floating-geometry two-step → native tiled
  `new-pane --direction down` (the action contract Chunk 1 Task 1 originally
  pinned).
- `Alt+Shift+Enter` expand: floating overlay → tiled boundary resize toggling
  50/50 ↔ 1/3 left, 2/3 right (user decision 2026-07-27). Left stack reflows
  while expanded; no floating pane ever exists.
- `layoutcmd`: `FocusRightTerminal`/`pickRightTerminal` select tiled right
  terminals; `floatingTerminalCoordinates` and `terminalFillerX` are deleted.
- Estimate delta: +0.8h (layout pivot 0.3, resize planner 0.2, focus/split
  rework 0.15, docs/tests/smoke 0.15) → issue `estimate_hours: 1.6`.

### 2026-07-28 — live-smoke findings (Chunk 2, Task 8)

The isolated-session smoke (real client keystrokes via a PTY-attached driven
client — CLI `action write` proved untrustworthy, see finding 4) surfaced four
issues; all fixed and re-verified live:

1. **Split halves were invisible to every classifier.** zellij 0.44.3 omits
   `terminal_command` for panes created via `action new-pane --direction`, and
   the #118 tab-strip title (`[terminal 1]`, tabs user-renamable) defeats
   RoleForPane's title fallback — so the new half's own chords passed through
   to its shell and the draft-side picker couldn't target it. Fix: a
   pid-verified **TerminalPaneRegistry** (`terminal-panes-<tag>` sidecar; each
   `pair term` self-registers pane id + pid at startup; readers filter by
   liveness) overlaid via `RoleForPaneWith`, threaded through termcmd,
   layoutcmd, and the toggle. Additionally, `focusedWorkbenchPanes` resolves
   pair term's OWN pane (`ZELLIJ_PANE_ID`) ahead of the `is_focused` scan —
   zellij reports several panes focused at once, and the draft was winning by
   list order.
2. **`--near-current-pane` is harmful for tiled splits**: it makes zellij
   create the pane invisibly (process spawned, pane absent from the layout).
   Reverted to plain `--direction down` — correct because the chord can only
   arrive when the invoking terminal holds the real client focus.
3. **Recorded-half must outrank zellij focus in the picker**: zellij's
   `is_focused` on right-side panes is stale memory (it pointed at the top
   half right after the user left the bottom one). `pickRightTerminal` now
   prefers the pair-authored record, then zellij focus, then pane order.
4. **Rung ladder lost the 12-row rung with a split present** (the base layout
   doesn't join the 4-pane swap cycle) — fixed by the planned `small-split`
   fallback, placed LAST so the 4-pane cycle order
   `[minimized-split, third-split, small-split]` reproduces the 3-pane
   next/prev semantics; the nvim clamp needed no changes. Also: the toggle's
   resize loop needed an 80ms settle pause (zellij applies resizes
   asynchronously; without it the no-progress guard stops short), and a live
   conformance probe (`live_classify_probe_test.go`, env-gated) checks
   `ClassifyLiveLayout` against a real pane dump.

## Chunk 2: Tiled Right Terminal (Revision)

**Goal:** The right terminal (and its Alt+Shift+d split) live in the tiled
layout tree, making every workbench pane immune to mouse drag-move while
keeping frames, and Alt+Shift+Enter toggles the terminal column between 50%
and 2/3 width by tiled resize.

**Architecture:** Delete the floating layer instead of guarding it
(`Root Cause`). The layout's tiled tree gains the terminal as a first-class
pane; `pair term` keeps owning terminal-local chords through the existing
`workbenchshortcut` → `Runtime.RunZellijAction` seam (`ARCH-DRY`). The expand
toggle becomes a pure resize-step planner (geometry in → next zellij action
out) executed by the existing thin runtime loop (`ARCH-PURE`). All zellij
interactions stay behind the existing `Runtime`/fake seam used by termcmd and
layoutcmd tests (`ARCH-MOCK`).

### Core concepts — delta

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `terminalResizeStep` | `cmd/internal/layoutcmd/resizeplan.go` | new |
| `floatingTerminalCoordinates` | `cmd/internal/layoutcmd/layoutcmd.go` | deleted |
| `terminalFillerX` | `cmd/internal/layoutcmd/layoutcmd.go` | deleted |

- **terminalResizeStep** — given the right terminal column's current width, the
  tiled screen width, and the toggle target (expanded = 2/3 screen, normal =
  1/2 screen), returns the next `zellij action resize` argv and a `done` flag.
  Executed in a re-read-geometry loop (cap 12 iterations) because zellij's
  resize step size is a runtime detail we never hardcode.
  - **Relationships:** 1:1 with the toggle executor in `RunToggleFocused`; N:1
    with `zellijpane.Pane` geometry.
  - **DRY rationale:** replaces `floatingTerminalCoordinates` as the single
    "what geometry should the terminal have" fact; reuses the existing ≥60%
    width threshold as the expanded-state detector.
  - **Future extensions:** more width rungs (e.g. 3/4) become new targets in
    one function.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `TerminalSplitDownAction` | `cmd/internal/termcmd/run.go` | modified | `zellij action new-pane --direction down` |
| `ToggleFocusedLayout` | `cmd/internal/layoutcmd/layoutcmd.go` | modified | `zellij action resize` loop |
| `Layout3TiledTerminal` | `zellij/layouts/main-3.kdl` | modified | zellij tiled tree + swap layouts |
| `ClassifyLiveLayout` | `cmd/internal/launcher/layoutflow.go` | modified | live `list-panes` layout probe |
| `AlignFloatingTerminal` | `cmd/internal/layoutcmd/layoutcmd.go` | deleted | `change-floating-pane-coordinates` at `pair term` startup |

- **TerminalSplitDownAction** — new contract:
  `RunZellijActionQuiet("new-pane", "--direction", "down", "--name", "terminal", "--", "sh", "-c", rightTerminalPaneShell)`.
  No geometry math, no `--floating`/`--pinned`, no pre-shrink step. Quiet
  because `new-pane` prints the created pane id (kept from Chunk 1).
  - **Injected into:** `pumpStdinWithTimer` via the existing `Runtime` fake.
  - **Live risk:** `--direction down` follows client focus; the invoking
    terminal holds it in the tiled tree (the chord arrived on its stdin). If
    the live smoke shows the split landing outside the right column, add
    `--near-current-pane` (known-good from Chunk 1's intermediate state).
- **ToggleFocusedLayout** — `RunToggleFocused` keeps its list-panes read but
  loops: read geometry → `terminalResizeStep` → run action → re-read, until
  done/cap. Resize actions target the focused terminal pane's left edge
  (`resize increase left` to grow, `resize decrease left` to shrink).
  - **Live risk:** with a split present, resizing one stacked pane's left edge
    must move the shared column boundary, not L-shape the tree; smoke-verify
    both split and unsplit states.
- **ClassifyLiveLayout** — currently identifies Layout3 by
  `filler && floatingTerminal`; after the pivot a live layout-3 session would
  match the Layout2 signature and corrupt attach/resume records. Re-keyed to
  discriminate Layout3 by the tiled right `pair term` terminal pane (Task 6).
- **AlignFloatingTerminal** — startup alignment of the floating terminal
  against the filler boundary; dead with the floating layer, deleted along
  with its `termcmd/run.go` call site (Task 7).
- **Last-used split half** — the floating layer's layer-focus used to
  remember which split half `Alt+k`/focus-terminal returns to; a tiled tree
  reports no focused right terminal while focus sits in the left stack, so
  the last-used right-terminal pane id is recorded explicitly following the
  `RecordLastLeftPaneID` precedent (Task 6).
- **Layout3TiledTerminal** — tiled tree becomes: left column (agent, draft) at
  50%, right `pane name="terminal"` at 50% running the same sh -c command the
  floating pane ran (`rightTerminalPaneShell` shape). `floating_panes` block
  and `terminal-filler` deleted. Swap layouts: keep `minimized`/`third` at
  `exact_panes=3` with `terminal` replacing `terminal-filler`; add
  `minimized-split`/`third-split` at `exact_panes=4` (right column split into
  two stacked `terminal` leaves) immediately after their 3-pane twins so
  zellij's constraint matching preserves rung adjacency in both states.
  - **Live risk:** rung ladder (Alt+Up/Down) with a split present, and "back
    to default" from a rung with 4 tiled panes; if the base layout can't
    re-tile 4 panes sanely, add explicit `small`/`small-split` swap layouts
    and adjust the nvim clamp state machine.

### Task 4: Layout pivot — tiled terminal, no floating layer

**Files:**
- Modify: `zellij/layouts/main-3.kdl`
- Modify: `tests/term-pane-shortcuts-test.sh` (layout assertions)

- [x] **Step 1: Write failing layout assertions**

In `tests/term-pane-shortcuts-test.sh`, assert `main-3.kdl` has no
`floating_panes` block and no `terminal-filler`, has a tiled
`pane name="terminal"` whose sh -c command matches `rightTerminalPaneShell`
(rename-pane + `exec pair term`), and defines swap layouts
`minimized`, `minimized-split`, `third`, `third-split` with
`exact_panes=3/4/3/4` respectively.

- [x] **Step 2: Verify RED**

Run: `bash tests/term-pane-shortcuts-test.sh` — expect the new assertions fail.

- [x] **Step 3: Rewrite the layout**

Right pane tiled at 50% running the terminal command; delete floating block and
filler; update swap layouts per the delta table (keep draft `borderless=true`
and agent framed in every variant; keep `size=12`/`size=1`/`size="33%"` rungs).
Update the file's comment block: the "permanently floating" rationale is
replaced by the tiled drag-immunity rationale.

- [x] **Step 4: Verify GREEN + config check**

Run: `bash tests/term-pane-shortcuts-test.sh && zellij --config-dir zellij setup --check` — expect pass.

- [x] **Step 5: Commit**

`#123: move right terminal into the tiled tree`

### Task 5: Native tiled split action

**Files:**
- Modify: `cmd/internal/termcmd/run.go` (`splitTerminalDown`)
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Update expected op strings to the new contract (RED)**

`run_test.go` split cases expect exactly:
`quiet new-pane --direction down --name terminal -- sh -c <rightTerminalPaneShell>`
with no preceding `change-floating-pane-coordinates` op. Run
`go test ./cmd/internal/termcmd -count=1` — expect FAIL.

- [x] **Step 2: Reimplement `splitTerminalDown` (GREEN)**

Drop geometry lookup/shrink; keep a minimal guard that a right terminal exists
via `currentRightTerminalPane` (no change needed there — it selects via
`RoleForPane`, which matches on command/title and is floating-agnostic). Run
the same test — expect PASS.

- [x] **Step 3: Full package + shell suite**

Run: `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1 && bash tests/term-pane-shortcuts-test.sh` — expect pass.

- [x] **Step 4: Commit**

`#123: split right terminal as a native tiled split`

### Task 6: Tiled focus routing + layout classification

**Files:**
- Modify: `cmd/internal/layoutcmd/layoutcmd.go`
- Modify: `cmd/internal/layoutcmd/layoutcmd_test.go`
- Modify: `cmd/internal/termcmd/run.go` (pane pick keys on `IsFloating && IsFocused`; last-used-half recording)
- Modify: `cmd/internal/launcher/layoutflow.go` (`ClassifyLiveLayout`)
- Modify: `cmd/internal/launcher/layoutflow_test.go`, `cmd/internal/launcher/osruntime_test.go` (fixtures)
- Modify: `nvim/init.lua` (filler-trap rationale comments, ~3617)

- [x] **Step 1: Write failing selection tests**

`pickRightTerminal`/`focusedRightTerminal` must select *tiled* right terminals
(`IsFloating` no longer required; keep `isRightTerminal` position/command
predicate), preferring the last-used split half, falling back to the first.
The floating layer's layer-focus no longer persists this preference in a tiled
tree (when focus sits in the left stack, no right terminal reports
`is_focused`), so record the last-used right-terminal pane id explicitly,
following the existing `RecordLastLeftPaneID` precedent in the same decision
path — that recorded id is what `pickRightTerminal` and termcmd's pane pick
prefer. Keep the layout2 relative-move fallback when no right terminal exists.
Run `go test ./cmd/internal/layoutcmd ./cmd/internal/termcmd -count=1` —
expect FAIL.

- [x] **Step 2: Write failing layout-classification tests**

`ClassifyLiveLayout` currently identifies Layout3 by `filler && floatingTerminal`
and Layout2 by their absence — after Task 4 a live layout-3 session matches the
Layout2 signature, so `ProbeLiveLayout` would write a wrong layout record and
corrupt attach/resume. Update `layoutflow_test.go` / `osruntime_test.go`
fixtures: Layout3 = agent + draft + a *tiled* right `pair term` terminal pane;
Layout2 = agent + draft, no right terminal. Run
`go test ./cmd/internal/launcher -count=1` — expect FAIL.

- [x] **Step 3: Implement (GREEN)**

Focus stays id-based via `focus-pane-id` (the "never relative move-focus for
this jump" rule still holds — cheap insurance and unchanged consumers in draft
nvim / pair wrap / pair term). Rework `ClassifyLiveLayout` to discriminate
Layout3 by the tiled right terminal pane. Update `FocusRightTerminal`'s doc
comment: the filler-trap rationale is gone; the id-based rule stays for
robustness. Same for the filler-trap comments in `nvim/init.lua` (~3617) —
update to the tiled rationale unconditionally, not only if clamps change.

- [x] **Step 4: Cross-package verification**

Run: `go test ./cmd/internal/layoutcmd ./cmd/internal/termcmd ./cmd/internal/wrapcmd ./cmd/internal/dispatcher ./cmd/internal/launcher -count=1` — expect pass.

- [x] **Step 5: Commit**

`#123: tiled focus routing and layout classification`

### Task 7: Expand toggle as tiled resize

**Files:**
- Create: `cmd/internal/layoutcmd/resizeplan.go`
- Create: `cmd/internal/layoutcmd/resizeplan_test.go`
- Modify: `cmd/internal/layoutcmd/layoutcmd.go` (`RunToggleFocused`; delete `floatingTerminalCoordinates`, `terminalFillerX`, `AlignFloatingTerminal`)
- Modify: `cmd/internal/layoutcmd/layoutcmd_test.go` (delete `TestAlignFloatingTerminalUsesFillerBoundaryAtStartup`)
- Modify: `cmd/internal/termcmd/run.go` (remove the `AlignFloatingTerminal` startup call, ~line 199)

- [x] **Step 1: Write failing planner tests**

`terminalResizeStep` cases: at 50% → target 2/3 (grow via
`resize increase left`, not done); at ≥60% → target 1/2 (shrink via
`resize decrease left`); within tolerance (±2 columns) → done, no action;
zero/absurd geometry → done (refuse). Run
`go test ./cmd/internal/layoutcmd -count=1` — expect FAIL.

- [x] **Step 2: Implement planner + executor loop (GREEN)**

Pure planner in `resizeplan.go` (no IO); `RunToggleFocused` loops
read-geometry → step → act, cap 12. The loop test's `Runtime` fake must be
*stateful* (`ARCH-MOCK`): each `resize` action mutates the geometry the next
`ListPanesJSON` read reports (crude ±N columns per step is fine) — otherwise
convergence is untestable and only the cap path executes. Live smoke item 4 is
the conformance check for that modeled behavior.

- [x] **Step 3: Delete dead floating helpers**

Remove `floatingTerminalCoordinates`, `terminalFillerX`,
`AlignFloatingTerminal` (dead once the floating pane is gone — today it runs
`change-floating-pane-coordinates` on every `pair term` startup), the
`AlignFloatingTerminal` call in `termcmd/run.go`, and their tests (incl.
`TestAlignFloatingTerminalUsesFillerBoundaryAtStartup`). Shadow-sweep
(`ARCH-PURPOSE`):
`grep -rn "change-floating-pane-coordinates\|terminal-filler\|AlignFloatingTerminal\|IsFloating" cmd/ zellij/ tests/ nvim/`
must come back empty except `zellijpane`'s parser field (keep the field —
it's how tests assert no floating panes exist) and any hit must be justified
in the commit body.

- [x] **Step 4: Verify GREEN**

Run: `go test ./cmd/internal/layoutcmd -count=1 && go test ./... -count=1` — expect pass.

- [x] **Step 5: Commit**

`#123: expand terminal by tiled resize, delete floating layer helpers`

### Task 8: Docs, full verification, live smoke

**Files:**
- Modify: `README.md`, `atlas/architecture.md`
- Modify: `workshop/issues/000123-split-the-right-terminal-pane-with-alt-shift-d.md`
- Modify: `nvim/init.lua` (only if the live rung smoke demands clamp changes)

- [x] **Step 1: Update docs**

README keybinding notes (`Alt+Shift+Enter` now re-tiles 50/50 ↔ 1/3–2/3;
panes cannot be mouse-dragged) and atlas architecture description of layout 3
(tiled right terminal, no floating layer, no filler).

- [x] **Step 2: Full automated verification**

Run the FULL suite (repo lesson: partial verification gets caught as REWORK),
scrubbing the pair-session env leak:

```bash
env -u PAIR_SESSION_ID -u PAIR_TAG make test
zellij --config-dir zellij setup --check
git diff --check
```

Expected: all pass (known pre-existing failure allowed: parley's
`parley_harness_golden` 7/7).

- [x] **Step 3: Rebuild before the live smoke**

`main-3.kdl` and `nvim/init.lua` ship inside the embedded runtime bundle;
edits do not reach a live session until regeneration + rebuild (Chunk 1 was
bitten by exactly this — see its stale-binary inode check in the Log). Run
`make build` (regenerates the runtime bundle), then confirm the live `pair`
processes will use the fresh `bin/pair` (compare inode/mtime as in Chunk 1).

- [x] **Step 4: Live smoke in an isolated zellij session (driven via `zellij action write`)**

Checklist (each risky item has a named fallback):
1. Session starts: agent/draft left, tiled terminal right, no filler pane in
   `list-panes`.
2. `Alt+Shift+d` in the terminal → split lands as two stacked right panes
   (fallback: `--near-current-pane`).
3. Draft↔terminal focus round trip lands on the terminal, preferring the
   last-used split half via the recorded pane id from Task 6 (fallback: accept
   first/top half and log the relaxation).
4. `Alt+Shift+Enter` toggles 50% ↔ ~2/3 width, both unsplit and split
   (fallback: resize direction/edge rework).
5. Rung ladder `Alt+Up`/`Alt+Down` works unsplit AND with a split present;
   returning to the default rung re-tiles sanely (fallback: explicit
   `small`/`small-split` swap layouts + nvim clamp update).
6. Mouse: frame drag on any *workbench* pane (agent, draft, terminals) moves
   nothing; boundary drag at most resizes; click-to-focus, scroll,
   copy-on-select still work. (Transient floating overlays — review,
   scrollback — remain draggable by design; outside this issue's Done-when.)
7. `Alt+t`/`Alt+w`/`Alt+r` terminal tabs still behave (#118 surface).
8. Launcher attach/resume probe classifies the session as layout 3
   (`ClassifyLiveLayout` via a fresh `pair` attach or its test-hook).

- [x] **Step 5: Record evidence, update issue log, commit**

`#123: docs + smoke for tiled right terminal`
