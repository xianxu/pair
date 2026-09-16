---
id: 000262
status: open
deps: [pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Diagnose input-triggered screen flicker

## Problem

The screen sometimes flickers subtly during fast Neovim input, without visible
corruption. Holding Delete and deleting one character at a time can trigger it;
running a program or producing heavy output in the right pane does not show the
same symptom.

## Spec

Instrument and reproduce the flicker at the terminal/input and compositor
boundaries. Compare rapid editor input with right-pane output, identify whether
extra redraws, synchronization, cursor updates, or scheduling cause the flash,
and fix the root cause without reducing input fidelity.

## Done when

- A bounded reproduction captures the input sequence and redraw/flush timing.
- The cause is corrected with no output corruption or lost input.
- Coverage distinguishes rapid editor input from high-volume right-pane output.
- Operator smoke testing confirms fast typing and held Delete no longer flicker.

## Plan

- [ ] Build a focused reproduction and lightweight redraw/input timing trace.
- [ ] Compare editor input, held Delete, and high-volume right-pane output.
- [ ] Correct the responsible path and add regression/performance coverage.
- [ ] Run terminal/compositor tests and obtain operator smoke confirmation.

## Log

### 2026-09-15

Filed as a follow-up to #255 from operator smoke testing. Flicker is subtle and
non-corrupting, appears during rapid input, and has not appeared during similar
right-pane output.
