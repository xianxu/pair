---
id: '000244'
status: wontfix
started: 2026-09-13T11:29:36-07:00
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 0.35
---

# draft keymap for M-S-Left/Right is spelled <S-M-Left> but nvim decodes Alt+Shift+arrow as <M-S-Left>, so it never fires

## Problem

The from-anywhere `M-S-Left`/`M-S-Right` never worked from the DRAFT nvim — the
operator confirmed it stays broken after restarting couch and pair, so it is
not a stale process. The regression fix #243 (deliver the global chord) and
#227 before it fixed the DELIVERY side; the draft's keymap firing was never the
thing under test.

Root cause, measured with a bare `nvim --clean` under a pty:

    \x1b[1;4D (Alt+Shift+Left) -> <S-M-Left> g:a=0   <M-S-Left> g:b=1

nvim decodes the terminal's Alt+Shift+Left as **`<M-S-Left>`** (modifier order
M-before-S), but the generated `nvim/workbench_actions.lua` spells the keymap
**`<S-M-Left>`** (from `GlobalBinding.NvimKey`). The two are NOT equivalent for
arrow keys — unlike `<M-t>`/`<M-T>` where shift folds into the letter case — so
the keymap never matches and `PairTermPrevTab` is never invoked. `M-S-t` works
because `<M-T>` is the correct spelling for a letter.

Why every test missed it: #216/#227/#243 tested the delivery (`--test-shortcut`,
`RunSwitchTerminalTab`, the pump), never that the draft's GENERATED keymap fires
on the bytes the terminal sends.
