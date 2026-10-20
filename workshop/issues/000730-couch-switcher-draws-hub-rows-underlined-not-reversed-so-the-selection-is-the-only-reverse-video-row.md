---
id: 000730
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6ec219185c4a9032f308606d8f9e4d19e3672ecb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T10:08:52-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "694c8393", done: "f65fe438"}
actual_hours: 0.63
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
- [x] Run targeted tests and queue the recorded suite; defer staging/live look per the TL revision below. A recorded pass remains mandatory before landing.

## Log

- Close round 1 found BR-1: stale README claimed reversed switcher hubs. Swept README and atlas hub/reverse/inversion descriptions; corrected the single stale README paragraph and recorded the prevention rule in lessons. No code changed; recorded suite remains queued.

### 2026-10-20
- 2026-10-20: closed — Cell-level TestHub/TestRenderMenu passed for hub underline, selected reverse, glyph reset and blocked color; existing multi-hub tests retain dimming. BR-1 README/atlas style sweep corrected stale README; no code changed. Recorded suite queued and required before landing. TL defers live look until after landing; no stage.; review verdict: SHIP

- Quick-flow design: reuse the existing reset-restoring decorator, underline all hub rows and add reverse only for selection. Tab-chip inversion is separate and stays unchanged. No new state, IO, persistence or growth driver.
- Automatic start-plan weave refresh produced unrelated CI workflow edits; preserved in stash named pair730 automatic weave workflow refresh preserved.
- Red/green: terminal-cell regression failed on an unselected reversed hub, then TestHub/TestRenderMenu passed after the decorator fix. Existing multi-hub render test still pins foreign-hub dimming. Atlas now distinguishes tab inversion and switcher underline.

## Revisions

### 2026-10-20 10:12 PDT — TL defers live look

- Reason: pair:0 remains leased to #722 for a live drain check waiting on ariadne#501; replacing its stage costs more than this cosmetic check warrants.
- Delta: skip staging and allow landing after the cell-level tests, recorded suite and close review. The operator checks the switcher after landing at the next Couch restart. Original acceptance required the two-hub live look before completion.

- Stage attempt refused the existing #722 lease; no stage was replaced. TL explicitly directed finish suite, close, and merge, deferring the live look as above.

### 2026-10-20 — drain checkpoint

- PR: https://github.com/xianxu/pair/pull/488. Implementation and atlas change are committed; targeted `go test ./cmd/internal/couchtty -run 'TestHub|TestRenderMenu' -count=1` passed.
- Recorded suite on code head 82ebdf3b9a1411e139592d3c43a218bb9e5f71b4 never entered the test slot. It waited behind Ariadne suites, then the operator reboot drain paused admission. Canceled only this checkout’s queued suite using `sdlc local-check cancel --name suite`; no suite pass exists.
- Resume: run `sdlc state`, run recorded suite on the pushed branch head, tick remaining verification plan item, close with measured actuals and fresh-context review, then merge on SHIP. Do not stage over #722; TL deferred the live look until next Couch restart (Revisions above).
- Unrelated automatic weave workflow changes remain preserved in stash named `pair730 automatic weave workflow refresh preserved`; exclude them from this issue.

### 2026-10-20 — resumed verification

- Recorded suite queued on pushed checkpoint 196495090a7892b4e0495b0cf65035a32f1a262b; no test execution yet. Per tl-worker’s explicit overlap rule, run close review during the queue wait; landing still requires a recorded pass, and any code fix requires fresh suite verification.
