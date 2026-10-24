---
id: 000850
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c7f2c4075a5ab9bbf0f13f206d698eaf7c778046' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:27:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
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

## Revisions

### 2026-10-23 — TL approved design (19f42a37)

Approved yellow300x320 within viewport, fixed header/scroll body, draggable header and minimized Notes(N) sharing one local position, keyboard move and cancel/resize cleanup. Anchored feedback/selection/stamps and849 invariants stay protected. Approved existing unfinished-label Add note and read-only completed View note; finished edits only if the operator later asks. Proceed through the full-flow design gate; live operator look at end. Added explicit Done-when coverage of these accepted semantics. No new permission pending.
