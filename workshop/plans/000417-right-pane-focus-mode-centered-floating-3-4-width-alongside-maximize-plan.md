# Right Pane Focus Mode Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** shift+alt+return cycles the right pane normal → **focus** (centered floating
pane, ~75% wide, agent pane dimmed) → **maximize** (existing tiled fullscreen) → normal.

**Architecture:** `layoutcmd` observes zellij (panes plus the tab's swap-layout name),
derives the current right-pane mode, and plans an ordered list of steps for the next
transition. One executor runs the steps and stops at the first failure. The phase
machine `FullscreenTransition` is retired into that list, so one owner orders all three
transitions. Dimming lives in `pair wrap`. On SIGWINCH it asks zellij whether a right
terminal is floating, and while one is, it adds faint (SGR 2) to the agent's output. The
agent redraws on resize, so its redraw comes out dimmed.

**Tech Stack:** Go, zellij 0.45.1 CLI actions, `charmbracelet/x/vt` (already in wrap).

---

## Probe findings (live zellij 0.45.1, 2026-10-09)

Probe: `cmd/internal/layoutcmd/focus_probe_live_test.go` (temporary, deleted in Task 6).
It runs the real `zellij/layouts/main-3.kdl` with receiver commands.

1. `toggle-pane-embed-or-floating --pane-id T` floats the right terminal. The floating
   layer is **not shown** (`are-floating-panes-visible` → false) until
   `show-floating-panes` runs.
2. `change-floating-pane-coordinates --pane-id T -x 12% -y 5% --width 75% --height 90%`
   centers it. Percentages resolve against the display.
3. Re-embedding does **not** restore the slot. zellij splits the focused tiled pane
   instead (the terminal lands bottom-right under the draft).
4. `current-tab-info --json` reports `active_swap_layout_name` (`BASE`, `minimized`,
   `third`, `*-split`) and `is_swap_layout_dirty`. Running `next-swap-layout` until the
   name equals the pre-focus name and the layout is not dirty restores the geometry
   exactly on every rung: base, minimized, third, with the agent or the draft focused.
   Plain next-then-prev does **not** work, because floating disturbs the swap index.
5. In a split (two right halves), the name is lost while floating (`BASE`), and
   after the restore cycle the halves can come back in swapped order. One `move-pane up`
   or `move-pane down` on the re-embedded half fixes the order.
6. `focus-pane-id <tiled pane>` hides the floating layer. Pinning
   (`--pinned true`) is meant to keep a pinned pane on top regardless; this is checked
   **visually in the smoke test**, because the CLI cannot observe the composed screen.
7. `toggle-fullscreen` on a floating pane fullscreens it. We never produce that state,
   and focus→maximize re-embeds first.
8. `list-panes --json` and `current-tab-info --json` each take about 15ms on 0.45.1
   (the 590ms in #220 was 0.44.3).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `RightPaneMode` | `cmd/internal/layoutcmd/rightpane.go` | new |
| `ObserveRightPaneMode` | `cmd/internal/layoutcmd/rightpane.go` | new |
| `ExpandRecord` | `cmd/internal/layoutcmd/rightpane.go` | new |
| `RightPanePlan` / `PlanRightPane` | `cmd/internal/layoutcmd/rightpane.go` | new |
| `FullscreenPlan` / `PlanFullscreen` | `cmd/internal/layoutcmd/fullscreen.go` | modified (selection only; feeds PlanRightPane) |
| `FullscreenTransition` (phase machine) | `cmd/internal/layoutcmd/fullscreen.go` | deleted |
| `FocusModeActive` | `cmd/internal/layoutcmd/rightpane.go` | new |
| `sgrDimmer` | `cmd/internal/wrapcmd/dim.go` | new |
| `ClassifyLiveLayout` | `cmd/internal/launcher/layoutflow.go` | modified |

- **RightPaneMode** — `Normal | Focus | Maximize`, derived from observation only. A right
  terminal with `IsFloating` means Focus. A tiled right terminal with
  `*IsFullscreen` means Maximize. Anything else is Normal. More than one floating or
  fullscreen right terminal is an error, as today.
  - **Relationships:** 1 per tab observation; zellij owns the state, so nothing persists it.
  - **DRY rationale:** wrap (dim) and layoutcmd (cycle) read the same predicate
    (`FocusModeActive` = mode is Focus), so they cannot disagree about "focus is on".
- **ExpandRecord** — `{Return, Swap string; Order []string}` encoded on the single line the
  existing `FullscreenReturnStore` holds: `"<return> swap=<name> order=<id>,<id>"`. A bare
  `"<id>"`, which is the record existing sessions have, decodes as `{Return: id}`.
  - **Relationships:** written on leaving Normal, cleared on returning to Normal; 1 per tag.
  - **DRY rationale:** reuses the fullscreen-return artifact family, its lock and its
    bounded reader, instead of a second family (ARCH-FUNERAL: no new residue; the existing
    family's retention covers it).
  - **Future extensions:** more fields append as `key=value` tokens.
- **PlanRightPane(obs) → RightPanePlan{From, To Mode; Steps []Step}** — pure. Steps per
  transition:
  - Normal→Focus: `SaveRecord`, `Float(T)`, `Place(T)` (coords + pinned),
    `ShowFloating`, `FocusPane(T)`, `NudgeWrap`.
  - Focus→Maximize: `Embed(T)`, `RestoreTiling`, `ToggleFullscreen(T)`, `NudgeWrap`. The
    record keeps `Return` and drops `Swap`/`Order`.
  - Maximize→Normal: `ToggleFullscreen(T)`, `FocusPane(Return)` (skipped when Return==T),
    `ClearRecord`. This is today's collapse, unchanged.
  - Normal with no right terminal: no steps (no-op, as today).
  - T is chosen by today's `PlanFullscreen` selection rules (caller half, else recorded
    half, else focused, else first). In Focus/Maximize, T is the observed
    floating/fullscreen pane.
- **sgrDimmer** — a stream transformer. `Feed(chunk, on bool) []byte` emits `ESC[2m` when
  dimming turns on, `ESC[22m` when it turns off, and while on appends `ESC[2m` after every
  complete SGR sequence (CSI, no private marker, no intermediates, final `m`). It carries
  an incomplete CSI tail across chunks, bounded at 256 bytes (an over-long tail is flushed
  raw). No parameter parsing: appending faint after each SGR keeps colors, bold and resets
  intact and only re-asserts faint.
- **ClassifyLiveLayout** — a floating right terminal alone (focus mode) is Layout3. The
  condition becomes `tiledTerminal || floatingTerminal`, which also covers the
  pre-pivot `filler && floatingTerminal` signature.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `layoutcmd.Runtime` | `cmd/internal/layoutcmd/layoutcmd.go` | modified (+`CurrentTabJSON`, +`NudgeWrap`) | `zellij action`, `kill(2)` |
| `RunToggleFocused` executor | `cmd/internal/layoutcmd/rightpane.go` | modified (moved from fullscreen.go) | Runtime + FullscreenStore |
| `restoreTiling` | `cmd/internal/layoutcmd/rightpane.go` | new | Runtime (cycle swap layouts, fix half order) |
| wrap focus observer | `cmd/internal/wrapcmd/dim.go` | new | `zellij action list-panes --json` |

- **Runtime.CurrentTabJSON** — `zellij action current-tab-info --json`.
  **Runtime.NudgeWrap** — reads the scoped `PairWrapPID` (the reader pattern from
  `agentcmd/restart.go`) and sends SIGWINCH. It is best effort: an error is logged to
  the fullscreen diagnostics and never fails the transition. It needs a `manifest.go`
  ResolvedConsumer row for `layoutcmd`.
  - **Injected into:** the executor. Tests use the existing stateful fake runtime in
    `fullscreen_test.go`, extended to model floating, the swap name and order.
- **restoreTiling** — cycles `next-swap-layout` at most 4 times until
  `active_swap_layout_name == record.Swap && !dirty`. Then, if `record.Order` has 2 ids
  and the observed tiled order differs, it runs `focus-pane-id T` and
  `move-pane up|down`. If `record.Swap` is empty or unreachable, it stops after the
  cycle bound and logs a diagnostic, leaving zellij's tiling as is (fail-soft: the
  pane is embedded and usable). For a split, the pre-focus swap name is a `*-split`
  name, and after the re-embed the 4-pane cycle can reach it.
- **wrap focus observer** — `observeFocus func() (bool, error)`, defaulting to
  list-panes + `layoutcmd.FocusModeActive(panes, registered)`, with the registry from
  `workbenchshortcut.LiveTerminalPaneIDsFromEnv`. It is called in the SIGWINCH handler
  **before** `setWinsize`, so the flag is set before the child sees the resize and
  redraws. If the flag changed, wrap also sends the child SIGWINCH after `setWinsize`,
  for the split case where the agent pane does not resize. On error the flag stays
  as it was.

## Tasks

### Task 1: ExpandRecord encode/decode

**Files:** Create `cmd/internal/layoutcmd/rightpane.go`; Test `cmd/internal/layoutcmd/rightpane_test.go`

- [ ] Write table tests: `"3"` → `{Return:"3"}`; `"3 swap=third-split order=2,5"` round-trips;
      unknown `key=v` tokens are ignored; an empty Return is an error; Encode output passes
      `FullscreenReturnStore`'s validation (≤256 bytes, no newline).
- [ ] Run `go test ./cmd/internal/layoutcmd -run TestExpandRecord` → FAIL (undefined).
- [ ] Implement `ExpandRecord`, `EncodeExpandRecord`, `DecodeExpandRecord` (strings.Fields).
- [ ] Re-run → PASS. Commit `#417: layoutcmd: expand record carries swap layout + split order`.

### Task 2: Mode observation + PlanRightPane, retire the phase machine

**Files:** Modify `fullscreen.go`, `rightpane.go`, `fullscreen_test.go`, `fullscreen_adversarial_test.go`

- [ ] Write tests for `ObserveRightPaneMode` (tiled → Normal, floating → Focus, tiled+fs
      → Maximize, two floating → error, nil fullscreen → error) and for `FocusModeActive`.
- [ ] Write `PlanRightPane` tests asserting the exact step lists above for each
      transition, including the split cases (T = caller half; T = observed floating half),
      `Return == T` skipping FocusPane, and no right terminal → no steps.
- [ ] Convert `TestFullscreenTransition*` ordering tables into expected step lists for
      Maximize→Normal (kept behavior), and delete the phase/event types they covered.
      Keep `TestPlanFullscreenGeneratedInventorySafety` pointed at selection.
- [ ] Implement; `go test ./cmd/internal/layoutcmd` → PASS. Commit.

### Task 3: Executor + restoreTiling against the fake

**Files:** Modify `rightpane.go`, `layoutcmd.go` (Runtime + OSRuntime), `fullscreen_test.go` fake

- [ ] Extend the fake runtime with this state: pane floating flags, fullscreen, swap
      name, dirty, tiled order. Model the probe-observed behaviors: embed disturbs
      order/dirty; `next-swap-layout` advances through a configured cycle; `move-pane`
      swaps order.
- [ ] Tests: the full three-press cycle returns to the original fake state (geometry
      tokens, swap name, order, focus on the invoking pane, record cleared); a step failure
      stops the run and logs; the restore bound is hit when the target name is absent;
      NudgeWrap failure doesn't fail the press; the lock is busy → no-op (as today).
- [ ] Implement the executor (one loop over steps; stop on the first error) and
      `restoreTiling`. Add `CurrentTabJSON`/`NudgeWrap` to `OSRuntime` and the
      manifest consumer row; run `go test ./cmd/internal/artifactpath ./cmd/internal/layoutcmd`.
- [ ] Commit.

### Task 4: Layout classifier accepts focus mode

**Files:** `cmd/internal/launcher/layoutflow.go`, its test

- [ ] Test: agent + draft + floating right terminal (no filler) → Layout3; agent + draft
      only → Layout2; the old filler+floating signature → Layout3.
- [ ] Simplify the condition; run `go test ./cmd/internal/launcher` → PASS. Commit.

### Task 5: Wrap dimming

**Files:** Create `cmd/internal/wrapcmd/dim.go`, `dim_test.go`; Modify `wrap.go` (handleChunk passthrough, SIGWINCH handler)

- [ ] sgrDimmer tests: off is the identity; the on-transition prefix; `ESC[31m` →
      `ESC[31mESC[2m`; `ESC[0m`/`ESC[m` get faint re-asserted; `ESC[>4;2m` and `ESC[?25h`
      are untouched; an SGR split across two Feeds; the off-transition emits `ESC[22m`; the
      tail bound flushes raw; non-SGR OSC passes through byte-identically.
- [ ] Proxy test with an injected observer: the SIGWINCH path sets dim before
      setWinsize, and a changed flag with unchanged size signals the child (seam:
      the existing `setPTYWinsize`/`getWinsize` injection plus a `signalChild` func).
      Scrollback log and terminal model still receive raw bytes.
- [ ] Implement: `p.dim atomic.Bool`, a dimmer applied to `event.Passthrough` in
      `handleChunk`, and the observer default in `dim.go`.
- [ ] `go test ./cmd/internal/wrapcmd` → PASS. Commit.

### Task 6: Live conformance, help text, atlas

**Files:** `cmd/internal/layoutcmd/rightpane_conformance_live_test.go` (replaces the probe),
`cmd/internal/workbenchshortcut/shortcut.go:180-181` help, `zellij/layouts/main-3.kdl:83`
comment, `atlas/architecture.md` (~436-450, 828)

- [ ] The live test runs real `main-3.kdl` with receivers. On each rung, unsplit and
      split, it runs the three-press cycle via `RunToggleFocused` and asserts: Focus
      shows a floating, visible, centered T (≈75%×90%); Maximize shows a tiled
      fullscreen T; Normal restores the exact geometry, swap name, half order and focus
      on the invoker. After a split round trip, `next-swap-layout` keeps the half order.
      Delete `focus_probe_live_test.go`.
- [ ] `PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run 'Conformance' -count=1`
      (unsandboxed) → PASS; the existing fullscreen conformance is adapted to the cycle.
- [ ] Help: "cycle right terminal: normal → focus (centered) → maximize". Atlas: describe
      the modes, the record, restoreTiling, and wrap dimming.
- [ ] Commit.

### Task 7: Verify + smoke

- [ ] Full suite per memory: `make -k test`, scratchpad-TMPDIR test-changelog, `go test ./...`.
- [ ] Move the branch to pair:0 (`sdlc move :0`), `make build`, and ask the operator to
      smoke test in a fresh thread: the cycle on each rung and split; whether a pinned
      pane stays on top after Alt+k / clicking the agent; whether the dim covers Claude's
      whole redraw and how faint looks; whether the redraw flicker is acceptable.
      Record the findings in `## Log`.

## Risks

- The agent redraws only part of the screen on resize → partial dim. The fallback (out
  of scope unless the smoke test shows it) is to repaint from wrap's vt snapshot.
- Rung changes (Alt+Up/Down) while in focus are undone on exit (the restore
  targets the pre-focus rung). This is acceptable and is documented in the atlas.
- Manual resizes (dirty swap layout) are not restored. That matches what a rung
  change already does.

## Revisions

### 2026-10-09 — operator review
- Focus geometry: **75% width × 100% height**, centered horizontally
  (`-x 12% -y 0 --width 75% --height 100%`), replacing 75% × 90%. Task 6's
  conformance test asserts full height.
- Dropped the "manual resizes are not restored" risk. Pair's zellij config clears all
  default keybinds (`zellij/config.kdl:96`) and disables mouse resize (`:37`), so the
  operator cannot resize panes manually. The only source of a dirty swap layout is
  Pair's own actions (e.g. the Alt+Shift+d split), and restoreTiling's cycle covers
  those.
- Partial-dim risk acknowledged by the operator; the fallback stays out of scope unless
  the smoke test shows it.
