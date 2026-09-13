---
id: 000243
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 1.00
started: 2026-09-13T10:35:43-07:00
---

# regression: from-anywhere M-S-left/right stops switching the right terminal's tab when it shows a full-screen app (#227)

## Problem

Two things, one root:

1. **Regression from #227.** The from-anywhere `M-S-left`/`M-S-right` (switch the
   right terminal's tab from any pane, #216) stopped switching when the right
   pane shows a **full-screen app** (nvim). Confirmed at the pump: the delivery
   sends the ROLE-SCOPED `Alt+Left`/`Alt+Right` bytes to the right terminal
   (`TabChordFor` -> `ChordAltLeft`/`ChordAltRight`), and #227 now passes exactly
   those through to the full-screen child instead of switching:

       Alt+Left  (old delivery) under fullscreen -> ops=[write:ESC[1;3D]  (eaten by nvim)
       Alt+Shift+Left (global)  under fullscreen -> ops=[prev-tab]        (switches)

   The delivery reused the role-scoped chord; #227 made that chord passthrough-
   eligible. The fix is to deliver the GLOBAL chord, which pair term always
   handles and never passes through.

2. **Feature the operator asked for.** Make the from-anywhere right-terminal
   control a set of **three** global chords -- `M-S-left`, `M-S-right`, and a new
   `M-S-t` (create a tab) -- so the operator can drive the right pane from the
   draft or agent even while a full-screen app owns it. `M-S-t` is also the
   deliberate "escape chord" #227 left out: create a tab without leaving nvim.

## Spec

The from-anywhere right-terminal control set is delivered as GLOBAL chords, so
each survives #227's passthrough and pair term always acts on it:

- `TabChordFor` returns the GLOBAL chord for each action: `ActionTerminalPrevTab`
  -> `ChordAltShiftLeft`, `ActionTerminalNextTab` -> `ChordAltShiftRight` (the
  fix), and a new `ActionTerminalNewTab` -> `ChordAltShiftT`.
- New chord `ChordAltShiftT` (`ESC T`, `ESC[84;4u`), action
  `ActionTerminalNewTab`, and a `globalBindings` row: `ChordAltShiftT` ->
  `PairTermNewTab`, `<M-T>`, `HandledInPane`. The nvim keymap
  (`workbench_actions.lua`) is REGENERATED from `globalBindings`.
- pair term's `handleTerminalChord` gains `ChordAltShiftT` -> `newTab`.
- The agent pane (`wrapcmd`) and the CLI (`RunSwitchTerminalTab`, `new`
  direction) gain the new action; init.lua gains `PairTermNewTab() =
  pair_switch_terminal_tab('new')` -- the existing route/argv plumbing carries
  it.
- `M-k` and the operator TYPING `Alt+Left`/`Alt+Right` inside nvim are
  UNCHANGED: those role-scoped chords still pass through under a full-screen app
  (that is #227's point). Only the from-anywhere GLOBAL delivery uses globals.

## Done when

- With a full-screen nvim in the right pane, `M-S-left`/`M-S-right` from the
  draft switch its tabs, and `M-S-t` opens a new tab -- asserted at the pump
  (the global chords produce prev-tab/next-tab/new-tab under `ownsScreen=true`,
  no passthrough).
- `TabChordFor` returns a chord for which `IsGlobalChord` is true for every
  action (a test ties the two, so a future role-scoped delivery cannot regress
  #227 again).
- Operator TYPING `Alt+Left`/`Alt+Right` in nvim still reaches nvim (unchanged
  #227).
- `pair keys` / help, atlas, and README show the three from-anywhere chords.
- Live: from the draft with nvim in the right pane, `M-S-left/right/t` drive it.

## Plan

Durable plan: `workshop/plans/000243-from-anywhere-right-terminal-control-set-plan.md`.

- [x] `ChordAltShiftT` (`\x1b[84;4u` (KKP only)) + `ActionTerminalNewTab` + `ChordName`
- [x] `TabChordFor` returns the GLOBAL chord for prev/next/new; test pins `IsGlobalChord`
- [x] `globalBindings` row for `M-S-t`; regenerate `nvim/workbench_actions.lua`
- [x] `handleTerminalChord` `ChordAltShiftT` -> `newTab`; pump test (three globals switch/create under fullscreen, no passthrough)
- [x] `wrap.go` + `RunSwitchTerminalTab` (`new` direction) gain the action
- [x] `init.lua` `PairTermNewTab`; keyhelp + atlas + README; full make test; live check

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Design at x0.2 (the plan resolves the delivery-chord fix, the new chord/action/binding, and the wiring); impl at 40% of v2; +15% buffer.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.05 impl=0.16
item: smaller-go-module  design=0.04 impl=0.16
item: lua-neovim         design=0.02 impl=0.10
item: smaller-go-module  design=0.03 impl=0.16
item: atlas-docs         design=0.03 impl=0.08
item: milestone-review   design=0.00 impl=0.14
design-buffer: 0.15
total: 1.00
```

- new chord + action + ChordName + TabChordFor fix/new -- 0.05 / 0.16
- handleTerminalChord + wrap.go + RunSwitchTerminalTab -- 0.04 / 0.16
- init.lua PairTermNewTab + regenerate workbench_actions.lua -- 0.02 / 0.10
- pump/global tests + TabChordFor-is-global guard -- 0.03 / 0.16
- keyhelp + atlas + README -- 0.03 / 0.08
- close review -- 0.00 / 0.14

## Log

### 2026-09-13

- Filed after #227 shipped: the operator found `M-S-left/right` no longer switch
  tabs when nvim (full-screen) is in the right pane, and asked for a third
  from-anywhere chord `M-S-t`. Reproduced the regression at the pump (`Alt+Left`
  passes through under `ownsScreen=true`). Traced the delivery: `TabChordFor` ->
  role-scoped `ChordAltLeft`/`Right`, which #227 now forwards. Wiring mapped:
  globalBindings -> `RenderLuaGlobalMaps` -> workbench_actions.lua -> init.lua
  `pair_switch_terminal_tab` -> workbench_route argv -> dispatcher `layout
  switch-terminal-tab` -> `RunSwitchTerminalTab` -> `TabChordFor` ->
  `SwitchRightTerminalTab` (delivers bytes to the right terminal).

### 2026-09-13 (close)

- **Live encoding verified (BR-2).** A bare `nvim --clean` under a pty (no pair
  term) with `nnoremap <M-T>` set, fed `\x1b[84;4u` (what zellij's `Alt T` bind
  delivers), fires the map: `get(g:, 'mt', 0) == 1`. So the generated
  `["<M-T>"]` keymap in the draft receives the operator's `M-S-t`. This is the
  `<M-N>` precedent (`\x1b[78;4u` -> `<M-N>`) confirmed for `<M-T>`. The
  remaining operator step is the in-workbench check that `M-S-Left/Right/t`
  from the draft drive a full-screen right pane, which needs a fresh `pair term`.
- BR-1: `RunSwitchTerminalTab` now has per-direction delivery-byte tests
  (prev/next/new deliver the global bytes). BR-3: the two `newTab` cases in
  `handleTerminalChord` are merged. BR-4: this duplicate heading removed; the
  atlas sentence reworded.
