---
id: 000417
status: codecomplete
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: 'b9b45d2c87a1fa85ca798656397f9182ebcc0bbb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T09:42:52-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:0
    worktree: /Users/xianxu/workspace/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: N/A
---

# Right pane focus mode: centered floating ~3/4 width, alongside maximize

## Problem

The right pane has one enlarged view, **maximize** (shift+alt+return, zellij
tiled fullscreen, `cmd/internal/layoutcmd/fullscreen.go`). Full width is often
too wide to read comfortably. A better view for many cases is the right pane
centered on screen at about 3/4 width, but not full width.

## Spec

Name the two right-pane modes **maximize** (existing) and **focus** (new).

Focus mode, sketched with zellij 0.45.1 primitives:
- `zellij action toggle-pane-embed-or-floating --pane-id <right>` floats the
  right terminal pane;
- `zellij action change-floating-pane-coordinates --pane-id <right> -x 12% -y 5%
  --width 75% --height 90%` centers it (percent sizes adapt to the window);
- toggling again re-embeds it.

Decided (operator, 2026-10-09):
- **Keybinding:** shift+alt+return cycles normal → focus → maximize → normal.
  No separate shortcut.
- **Agent pane during focus:** stays visible around the edges, **dimmed**.
  zellij 0.45.1 has no per-pane dim action, so pair-wrap dims its own output.
  Dimming is a nice-to-have: if it turns out hard, ship focus without it.

Dimming approach (pair-wrap, feasibility-checked 2026-10-09):
- Agent stdout already passes through a rewrite stage in pair-wrap
  (`notificationRewriter.Feed`, `cmd/internal/wrapcmd/wrap.go:3248`). Add a
  dim stage there that, while focus is on, forces faint (SGR 2) into every SGR
  the agent emits, dropping bold/normal-intensity (SGR 1/22) codes that would
  cancel it.
- No repaint machinery is needed: entering focus resizes the agent pane to
  full width, and the agent redraws its whole screen, so that redraw comes out
  dimmed. Exiting focus resizes it back, and the redraw comes out normal.
- Needed: a way to tell pair-wrap focus is on (layoutcmd → wrap signal).
- Risks to check live: an agent that redraws only part of the screen on
  resize leaves the rest undimmed until it repaints; how the terminal renders
  faint over truecolor. Fallback if the redraw is unreliable: repaint from
  pair-wrap's existing vt emulator snapshot (`terminal_model.go`) with dimmed
  styles.

Open questions for the design pass:
- Agent pane reflows to full width underneath while focus is on (the agent
  TUI redraws on enter and exit).
- Does re-embedding restore the original split ratio? Measure live; if not,
  record the ratio and resize after re-embed.
- `cmd/internal/launcher/layoutflow.go:73` reads a floating terminal as the
  legacy pre-pivot layout signature; a focused right pane matches neither
  branch, so relaunch/reattach during focus must be guarded (as fullscreen
  state is).
- The cycle passes focus → maximize directly: re-embed and fullscreen in one
  step must converge to one well-defined state (no stray floating or
  fullscreen pane), including when zellij state drifts from Pair's record.

## Done when

- shift+alt+return cycles normal → focus (centered, 75% width × full height,
  agent pane dimmed) → maximize → normal, restoring the split on return to normal.
- **Maximize** behaves as before; switching between focus and maximize
  converges without leaving a stray floating or fullscreen pane.
- Relaunch/reattach while in focus mode doesn't misclassify the layout.
- Tests cover the plan logic (enter/exit/switch, layout classification);
  the operator smoke-tests live in pair:0.

## Plan

Durable plan: `workshop/plans/000417-right-pane-focus-mode-centered-floating-3-4-width-alongside-maximize-plan.md`.
Single pass (no Mx), so one review runs at close.

- [x] Task 1: ExpandRecord encode/decode
- [x] Task 2: mode observation + PlanRightPane; retire the FullscreenTransition phase machine
- [x] Task 3: executor + restoreTiling against the stateful fake
- [x] Task 4: layout classifier accepts focus mode
- [x] Task 5: wrap dimming (sgrDimmer + SIGWINCH observer)
- [x] Task 6: live conformance test, help text, atlas
- [x] Task 7: full verification + operator smoke test in pair:0

## Log

### 2026-10-09
- 2026-10-09: closed — Operator smoke-tested in pair:0 on a fresh thread (177707a5): cycle normal→focus→maximize→normal, restore, pinned centered pane, agent dim all work. Live zellij (unsandboxed): TestFullscreenChordZellijLive (real Shift+Alt+Return via real pair panes), TestFullscreenZellijConformance, TestRightPaneRungsZellijConformance (main-3.kdl every rung, split+dirty split, half order survives rung change); restoreTiling no-op mutation fails live and fake tests. Round-1 fixes (ec51c477): dimmer frames in both modes, never delays bytes, transitions at the first sequence boundary (chunk-invariance test over every split point incl. ST-terminated OSC 52), 1s-bounded zellij observer, handleWinch ordering test via signalChild seam, README synced; wrapcmd/layoutcmd/termcmd/keyhelp green, launcher green under clean env. make -k test green except test-changelog (passes with scratchpad TMPDIR) and test-pair-embedded-runtime (passes with full env scrub). go test ./...: artifactpath list identical to merge base, gcruntime pre-existing, couchcore passes standalone on branch and base (only the 10m default timeout), terminal lacks node module @xterm/addon-unicode11. Actual N/A: session ran from the slot-1 worktree cwd, no transcript events for this window.; review verdict: SHIP
- 2026-10-09: flow upgraded quick → full — 723 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Live probes on zellij 0.45.1 (see the plan's "Probe findings"):
  - Re-embed doesn't restore the slot. Cycling `next-swap-layout` to the recorded
    `active_swap_layout_name` restores every rung exactly, and `move-pane` fixes the
    order of split halves.
  - A floated pane needs `show-floating-panes`.
  - Focusing a tiled pane hides the floating layer. Pinning should prevent that; the
    smoke test checks it.
- Design: no new artifact family. The restore info rides the existing
  fullscreen-return record (ARCH-FUNERAL), and wrap derives the dim state from
  zellij on SIGWINCH (zellij owns mode state, as with fullscreen today).

- Implementation done (Tasks 1–6). Live, unsandboxed:
  - `TestFullscreenChordZellijLive`: the real Shift+Alt+Return through real Pair
    panes (draft, agent, both split halves, an nvim child). Three presses each,
    with exact geometry, focus and record restored.
  - `TestFullscreenZellijConformance`: passes.
  - `TestRightPaneRungsZellijConformance`: every rung, the dirty split, the top
    half, and half order surviving a later rung change.
- Suite: `make -k test` failures were `test-term-pane-shortcuts`, a stale
  expectation that is now fixed; `test-changelog`, which passes with the
  scratchpad TMPDIR; and `test-pair-embedded-runtime`, which passes with the
  full env scrub (the Couch-slot leak).
- Operator smoke test in pair:0 on a fresh thread at 177707a5: "works well". The
  cycle, the restore, the pinned centered pane and the dim all hold up live.

## Revisions

### 2026-10-09
- Focus geometry is 75% width × 100% height (operator), not 75% × 90% as the
  Spec's sketch had it. Done-when updated.
