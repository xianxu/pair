---
id: 000417
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: '2682d075e2e09454b873270b8fae702d979009a6' # card fields mirrored from issue-cards; edit via sdlc
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

- shift+alt+return cycles normal → focus (centered, ~75% width, agent pane
  dimmed) → maximize → normal, restoring the split on return to normal.
- **Maximize** behaves as before; switching between focus and maximize
  converges without leaving a stray floating or fullscreen pane.
- Relaunch/reattach while in focus mode doesn't misclassify the layout.
- Tests cover the plan logic (enter/exit/switch, layout classification);
  the operator smoke-tests live in pair:0.

## Plan

- [ ]

## Log

### 2026-10-09
