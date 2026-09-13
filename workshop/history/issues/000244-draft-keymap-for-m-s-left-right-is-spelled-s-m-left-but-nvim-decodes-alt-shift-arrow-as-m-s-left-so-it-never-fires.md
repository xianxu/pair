---
id: 000244
status: wontfix
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 0.35
started: 2026-09-13T11:29:36-07:00
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

## Spec

- `GlobalBinding.NvimKey` for `ChordAltShiftLeft`/`ChordAltShiftRight` becomes
  `<M-S-Left>`/`<M-S-Right>` (the spelling nvim actually decodes). Regenerate
  `nvim/workbench_actions.lua` and the embedded bundle.
- `keyhelp` catalog `Key` for those rows matches the new NvimKey (so Help still
  derives).
- **A probe/test that the GENERATED keymap actually fires**, closing the gap
  that let this ship: a bare nvim loads the real keymap spelling and confirms
  `\x1b[1;4D`/`\x1b[1;4C` trigger it (mirrors the `<M-T>` bare-nvim check that
  DID prove #243's letter chord). Commit it as `probes/` so it re-runs.

## Done when

- The generated keymap uses `<M-S-Left>`/`<M-S-Right>`; a bare nvim fires them
  on `\x1b[1;4D`/`\x1b[1;4C`.
- `M-S-t` (`<M-T>`) still fires (unchanged; already proven).
- Full make test green incl the regenerated bundle guards.
- Live: from the draft, `M-S-Left`/`M-S-Right` switch the right terminal's tabs.

## Plan

- [ ] `NvimKey` -> `<M-S-Left>`/`<M-S-Right>` in `globalBindings`; regenerate keymap
- [ ] `keyhelp` catalog `Key` -> `<M-S-Left>`/`<M-S-Right>`
- [ ] Regenerate the embedded bundle; `TestEmbeddedSourcesMatchTree` green
- [ ] Probe: a bare nvim fires the GENERATED keymaps on the Alt+Shift+arrow bytes
- [ ] Full make test; live check from the draft

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Diagnosed spelling fix + a verifying probe.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.02 impl=0.10
item: lua-neovim         design=0.02 impl=0.10
item: milestone-review   design=0.00 impl=0.10
design-buffer: 0.15
total: 0.35
```

## Log

### 2026-09-13

- Diagnosed with a bare-nvim pty probe after the operator reported M-S-Left/
  Right still dead from the draft post-restart. The probe showed `\x1b[1;4D`
  fires `<M-S-Left>`, not `<S-M-Left>`. This is the draft-keymap half that
  #216/#227/#243 never verified — they tested delivery only.

## Log

### 2026-09-13

## Revisions

### 2026-09-13 — reject the modifier-order diagnosis

Operator confirms that Alt+Shift+Left/Right and Alt+Shift+T work from the
left pane to control right-pane tabs regardless of the app in the target tab,
and requests wontfix. The proposed spelling change is withdrawn.

The existing plan review (PQ-1) already refuted the claimed root cause:
Neovim canonicalizes both arrow spellings to the same mapping, so installing
both overwrites the first. Its `a=0, b=1` probe result was an overwrite
artifact, not proof that the first spelling cannot fire. Rechecked today with
`nvim --headless -u NONE -l /tmp/pair-244-keymap-check.lua`: both canonicalize
to `<M-S-Left>` and the second mapping replaces the first.

Close as wontfix with no production changes. Preserve the rejected plan and
its review findings as evidence; its implementation checklist is intentionally
unexecuted. No new diagnosis is inferred for the historical report.

### 2026-09-13 — disposition log

Marked wontfix at the operator's request after live acceptance and local
confirmation of the mapping-overwrite explanation. Current code verification
remains the passing shell/editor checks and final all-Go run from #242 in this
session; this disposition changes only issue/review records.
