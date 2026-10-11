---
id: 000449
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'c642f8643943a7cf52d7553735882e03643cb986' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T12:41:18-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "aa93a095", done: "584677b8"}
---

# Broadcast viewer: pointer color swatches as a vertical overlay on the screen's right edge

## Problem

pair#446 put the pointer color swatches in a band below the terminal screen. The operator (10-11, after a live smoke test that worked) wants them overlaid on the terminal screen itself, at the right edge, vertically centered, in a vertical column.

## Spec

- The six swatches render as a vertical column overlaid on the terminal stage (not the page), at its right edge, centered vertically on the stage's height; they track the stage as it resizes.
- Keep it unobtrusive: small swatches, a translucent backing, and it never takes pointer events outside the swatches themselves (a point or drag over the rest of the screen still reaches the pointer).
- The selected swatch stays marked; the pick is still remembered per browser.
- Phone width: still usable (the column can shrink); no horizontal page scroll.
- Leave room for pair#447's mode icon (pointer vs comment) in the same column.

## Done when

- On a live broadcast the column sits at the stage's right edge, vertically centered, at desktop and phone widths (operator smoke test; headless check at 390px).
- Pointing over the rest of the screen works as before (viewer test).

## Plan

- [ ] Move the swatch band into a stage overlay; CSS for position, size and translucency; viewer tests; headless width checks.

## Log

### 2026-10-11
