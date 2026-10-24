---
id: 000850
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours: 2.156
collaboration-mode: collaboration
card_mirror: 'cd28fe8f93b05acb923e67310accaa15128d8b97' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:27:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Cast notes: sticky-note style, minimize to a draggable button, draggable panel, visible add-note on sketches

## Problem

## Spec

**Cast notes: make the floating notes usable on the page (echo operator, 10-23).** In the Chrome cast's floating
notes (broadcast-web), the operator asked for:
1. **Sticky-note style:** the same look as the authoring experience (light yellow, like a sticky note).
2. **Minimize:** collapse the notes into a draggable button; the expanded panel draggable too. Today the notes cover
   the page and block clicking a note to edit it.
3. **Fixed height with a vertical scrollbar** (keep).
4. **A visible way to open or add notes on a sketch** (the operator couldn't find it).
Evidence: ops-echo `session.side/2026-10-23-03-echo-23-board-feedback.md` and its screenshots.

## Done when

- Notes render in the sticky-note style; minimize collapses them to a draggable button and restores them; the
  expanded panel can be dragged; its height is fixed with a scrollbar (tests for each behaviour).
- A sketch shows a visible control to open or add its notes (test), plus a brief operator look.

- The approved shared position, keyboard/cancel/resize behavior stays local to the mounted context; moving/minimizing never mutates anchored feedback, the selection, its stamps or label/witness authority. Unfinished sketches use existing Add note; completed labels open read-only. Guard regressions and the staged operator look cover both.

## Plan

Durable full-flow design: [Cast notes panel plan](../plans/000850-cast-notes-panel-plan.md).

- [ ] Implement one session-local floating-panel owner and sticky, fixed-height scrolling annotation list, preserving existing note rows/editor authority.
- [ ] Add visible per-sketch Add note / read-only View note controls through existing actions and accepted metadata, coordinating shared seams with #849.
- [ ] Verify pointer/keyboard/resize lifetimes, unchanged submission guards, headless layout, pre-close check and staged operator acceptance; close and land through SDLC.


## Log

### 2026-10-23

### 2026-10-23 — claimed; evidence and coordinated design

#845 landed PR568, integrated suite12m28s and final CI green; cleanup EXIT0. Hub and ariadne:4 notified with submitted receipts. Claimed850; start-plan created its independent branch from main. New fleet policy: local-check run --pre-close selects affected for Pair; never start an unnecessary parallel suite.

Read the echo operator's original side document and screenshot locally. Obstructing widget is the white .annotation-notes all-open list, separate from dark Pending feedback. No source evidence images copied. Existing annotation editor already has #fff59d sticky styling, while list rows are white; no panel header/minimize/drag owner exists. Sketch toolbar's Note is shown only while active, pending-label buttons are vague, and completed labels are final at recording/sketch_state.go. Proposed completed View note is read-only; unfinished Add note reuses existing label commands.

Pair:6 messages15fa9f8e and06588197 confirm #849 root cause is still under operator diagnosis. Preserve Shown acknowledgment, exact fresh comparison/freeze, board/delivered stamps and completed visibility witnesses; do not leave paint busy or change label commit/witness flow. Annotation-list presentation is independent of the scene paint selection; all new sketch changes remain affordance-only. They will report shared-file changes before implementation. Full design records explicit panel state, local-lifetime cleanup, counted costs, direct tests and stage look. Pending TL approval on panel geometry and final-label view semantics; no code, no estimate or change-code yet.


### 2026-10-23 — design gate passed; panel integration

Full-flow change-code round3 passed plan quality and estimate quality, with an advisory INFO only. Test-authoring cost is inside the two UI items and metadata mapping; the review/native/operator overhead item covers execution, not all test development. The UX allowance covers expected polish within the approved constraints, not new product choices. Queue wait remains excluded.

Floating-panel Task1 tests observed semantic RED before implementation, then8/8 GREEN including3,000 generated transitions and N/2N constant-cost movement. Annotation integration regression first failed because the scrolling body did not exist; after wrapping the existing keyed list,63 focused Node tests pass. Minimize preserves anchored feedback and note state; later data updates preserve local mode. Existing guard tests remain green. Added shared module to viewer embedding and isolated operator assembly; assembly initially refused the unmapped import, then both focused assembly tests passed after wiring. Panel resource disposal is exposed and registered on pagehide for both mount contexts; no changes to feedback/preview owners. Sketch controls and native layout proof are the remaining implementation tasks; no recorded check, merge runner or stage owned here.

## Revisions

### 2026-10-23 — TL approved design (19f42a37)

Approved yellow300x320 within viewport, fixed header/scroll body, draggable header and minimized Notes(N) sharing one local position, keyboard move and cancel/resize cleanup. Anchored feedback/selection/stamps and849 invariants stay protected. Approved existing unfinished-label Add note and read-only completed View note; finished edits only if the operator later asks. Proceed through the full-flow design gate; live operator look at end. Added explicit Done-when coverage of these accepted semantics. No new permission pending.

### 2026-10-23 — plan-quality PQ-1

First full-flow gate refused before implementation solely for prose test-case inventories. Compressed all task test descriptions to named production-function strategies with adversarial input classes and mechanical invariants. Commands, independent native/browser proof and cost bounds remain. No product/design change or estimate yet; rerun gate next.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is marked stale, so these numbers are provisional. The approved plan resolves choices: apply0.2 design discount and15% design buffer. Implementation values use the prescribed0.40 scale exactly once, familiarity1.0 for established browser/DOM/Chrome fixture patterns. Standard Pointer Events and existing modules cover the seams; no novel-stack library discovery or new external API is assumed.

Two screen/state-machine extensions: floating panel and sketch note controls each raw design1.0/impl1.0 -> design0.20/impl0.40. Compact metadata mapping uses smaller-module raw0.20/0.40 ->0.04/0.16. Shared embedding/import wiring uses cross-cutting-refactor raw0.40/0.30 ->0.08/0.12. Docs raw0.10/0.15 ->0.02/0.06. Atomic review/native/operator verification chunk raw0.10/0.50 ->0.02/0.20. One planned operator UX iteration raw0.40/0.20 ->0.08/0.08. Design subtotal0.64 ×1.15=0.736; implementation1.42; total2.156h. Queue wait is excluded; unresolved #849 diagnosis is peer work, not charged as implementation here.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: tui-screen design=0.20 impl=0.40
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.04 impl=0.16
item: cross-cutting-refactor design=0.08 impl=0.12
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
item: ux-rename-iteration design=0.08 impl=0.08
design-buffer: 0.15
total: 2.156
```
