---
id: 000730
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cd62c2f81f7dbb904f85d23a21300154e1e53b42' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T10:08:52-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "694c8393", done: "f65fe438"}
---

# couch: switcher draws hub rows underlined, not reversed, so the selection is the only reverse-video row

## Problem

In Couch's thread switcher, a hub row (ops, ops-echo) is drawn in reverse video, the same treatment as the
selected row, so with two hubs and a selection the operator sees three highlighted rows and can't tell at a glance
which one is selected (operator, 10-20, screenshot in ops's session file).

## Spec

- In the switcher, draw hub rows **underlined, without reverse video** (bold may stay); the selected row keeps reverse
  video, and a selected hub row is reverse plus underline. The code: `cmd/internal/couchtty/menu_render.go`, the
  `if row.hub` block (`style := reverseSGR`).
- The tab bar's hub chip (pair#521, `reserve.go`) is out of scope unless it shares the style constant; say so if it
  does.

## Done when

- A render test pins: a hub row carries underline and no reverse; a selected non-hub row carries reverse; a selected
  hub row carries both; the other-hub dim treatment is unchanged.
- Operator live look is deferred until after landing, at the next Couch restart: the switcher with two hubs shows one reverse-video row, the selection. Per TL direction, cell-level render tests qualify this cosmetic change for landing without staging.

## Plan

- [x] Pin hub underline, selected reverse, and other-hub dimming at the terminal-cell boundary; change the existing hub decorator (ARCH-PURE, ARCH-DRY).
- [ ] Run targeted tests and recorded suite; stage build for the operator’s two-hub live check.

## Log

### 2026-10-20

- Quick-flow design: reuse the existing reset-restoring decorator, underline all hub rows and add reverse only for selection. Tab-chip inversion is separate and stays unchanged. No new state, IO, persistence or growth driver.
- Automatic start-plan weave refresh produced unrelated CI workflow edits; preserved in stash named pair730 automatic weave workflow refresh preserved.
- Red/green: terminal-cell regression failed on an unselected reversed hub, then TestHub/TestRenderMenu passed after the decorator fix. Existing multi-hub render test still pins foreign-hub dimming. Atlas now distinguishes tab inversion and switcher underline.

## Revisions

### 2026-10-20 10:12 PDT — TL defers live look

- Reason: pair:0 remains leased to #722 for a live drain check waiting on ariadne#501; replacing its stage costs more than this cosmetic check warrants.
- Delta: skip staging and allow landing after the cell-level tests, recorded suite and close review. The operator checks the switcher after landing at the next Couch restart. Original acceptance required the two-hub live look before completion.

- Stage attempt refused the existing #722 lease; no stage was replaced. TL explicitly directed finish suite, close, and merge, deferring the live look as above.
