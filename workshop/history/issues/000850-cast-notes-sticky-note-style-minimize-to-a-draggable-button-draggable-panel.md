---
id: 000850
status: done
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours: 2.156
collaboration-mode: collaboration
card_mirror: 'd6c09ab7412a95762adeb274fae008829678fdc2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:27:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.99
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

- [x] Implement one session-local floating-panel owner and sticky, fixed-height scrolling annotation list, preserving existing note rows/editor authority.
- [x] Add visible per-sketch Add note / read-only View note controls through existing actions and accepted metadata, coordinating shared seams with #849.
- [ ] Verify pointer/keyboard/resize lifetimes, unchanged submission guards, headless layout, pre-close check and staged operator acceptance; close and land through SDLC.


## Log

### 2026-10-23
- 2026-10-23: closed — BR5 TL-approved grouped Sketch notes(N) selector replaces all colliding controls; native test independently selects all24 overlapping/edge-clamped rows, exact labels and zero final-view commands. All webcast Node tests, focused Go checks and affected inventory PASS. Refreshed source-blob native panel/full operator log PASS7.392s. Operator panel acceptance TL a7dcd3bb and selector approval TL36312a99 recorded. --no-plan-check acknowledges only final recorded-check/close/landing orchestration; existing old-source suite is finishing normally, and this corrected source must obtain its own pre-close pass before merge. Full review required, no judge bypass.; review verdict: SHIP

### 2026-10-23 — claimed; evidence and coordinated design

#845 landed PR568, integrated suite12m28s and final CI green; cleanup EXIT0. Hub and ariadne:4 notified with submitted receipts. Claimed850; start-plan created its independent branch from main. New fleet policy: local-check run --pre-close selects affected for Pair; never start an unnecessary parallel suite.

Read the echo operator's original side document and screenshot locally. Obstructing widget is the white .annotation-notes all-open list, separate from dark Pending feedback. No source evidence images copied. Existing annotation editor already has #fff59d sticky styling, while list rows are white; no panel header/minimize/drag owner exists. Sketch toolbar's Note is shown only while active, pending-label buttons are vague, and completed labels are final at recording/sketch_state.go. Proposed completed View note is read-only; unfinished Add note reuses existing label commands.

Pair:6 messages15fa9f8e and06588197 confirm #849 root cause is still under operator diagnosis. Preserve Shown acknowledgment, exact fresh comparison/freeze, board/delivered stamps and completed visibility witnesses; do not leave paint busy or change label commit/witness flow. Annotation-list presentation is independent of the scene paint selection; all new sketch changes remain affordance-only. They will report shared-file changes before implementation. Full design records explicit panel state, local-lifetime cleanup, counted costs, direct tests and stage look. Pending TL approval on panel geometry and final-label view semantics; no code, no estimate or change-code yet.


### 2026-10-23 — design gate passed; panel integration

Full-flow change-code round3 passed plan quality and estimate quality, with an advisory INFO only. Test-authoring cost is inside the two UI items and metadata mapping; the review/native/operator overhead item covers execution, not all test development. The UX allowance covers expected polish within the approved constraints, not new product choices. Queue wait remains excluded.

Floating-panel Task1 tests observed semantic RED before implementation, then8/8 GREEN including3,000 generated transitions and N/2N constant-cost movement. Annotation integration regression first failed because the scrolling body did not exist; after wrapping the existing keyed list,63 focused Node tests pass. Minimize preserves anchored feedback and note state; later data updates preserve local mode. Existing guard tests remain green. Added shared module to viewer embedding and isolated operator assembly; assembly initially refused the unmapped import, then both focused assembly tests passed after wiring. Panel resource disposal is exposed and registered on pagehide for both mount contexts; no changes to feedback/preview owners. Sketch controls and native layout proof are the remaining implementation tasks; no recorded check, merge runner or stage owned here.


### 2026-10-23 — candidate implemented; native regression fixed

Sketch controls reuse the existing accepted placement pass and current policy slots; final labels open read-only and stale handlers re-read current entries. TDD observed missing-affordance RED; focused68 tests passed. Strengthened unique P/2P property-access/projection counts and generated malformed placement/metadata cases pass. Both modules are embedded and assembled for viewer and isolated operator.

The first complete native operator run exposed a real #850 regression: a View note button placed directly at the first ink point intercepted the next stroke. The board held1 sketch while the preview helper waited for2. Added native hit-test proof, which failed in1.85s, then moved the44px control beside ink with an8px gap (below at the top edge). Complete native scenario passed6.31s after correction. Pair:6 notified; this is not evidence of their separate #849 publisher defect. Preview/visibility/commit/witness owners remain unchanged.

Final verification: all Node webcast tests EXIT0; focused Go operator assembly/server checks EXIT0; WEBCAST_LIVE_CHROME=1 targeted panel plus full operator drawing checks passed non-skipped9.756s. Native panel proof includes1280x800,320x480 and shrinking mid-drag to180x130, actual capture/lost capture, wheel scrolling, keyboard, full chip hits and page reachability. The actual operator scenario retains completed-paint and stale/selected-version guards through masks, tab returns and SPA replacement. make test-affected-inventory EXIT0 after staging the new native test; no reader certificate repair requested. Remaining: pushed-head pre-close record, boundary close, hub-timed stage/operator look, and merge. No stage is held by this slot.


### 2026-10-23 — close round1 REWORK; approved correction

BR1 Critical: first-point controls scroll away despite later visible ink. BR2 Important: minimized count text can grow beyond the viewport. BR3 Important: direct mode assignment bypasses the transition owner. BR4 unverifiable: reviewer Chrome GPU startup failed before page execution; staged operator acceptance still pending. Native worker source previously passed9.756s; retain corrected-source raw evidence after fixes. No close finalized.

TL f0df15c2 approved painter-returned bounds in the existing bounded per-path ink cache, visible bounds in one P pass, measured/clamped complete controls once per update; local count geometry and reducer corrections. Plan Revisions records the mechanism/cost. Supported cancel lease-2921208569 completed EXIT0 while it was queued, before its suite began; old source is superseded by known review defects and cannot supply final evidence. No duplicate check. Pair:6 owns combined stage6f0478a3a and sent restart/ask; per TL do not restage. Next: implement/pin corrections, retain native log, push and pre-close corrected head, then re-close after operator result. Stage acceptance remains pending.


### 2026-10-23 — BR1–3 fixed and verified; exact-source browser evidence

Painter now returns pressure-expanded bounds during its existing projection/paint loop; existing per-path ink cache unions incremental bounds and resets under its existing transform/repaint owner. One path pass derives clipped visible per-sketch bounds, and complete controls are measured/clamped once per update. Tests first reproduced absent bounds and offscreen first-point/multipath/edge failure, then passed. New mounted scroll/cache regression confirms the first sample can leave the viewport while Add note stays reachable and unchanged refresh paints zero points. Counted unique P/2P cases reject repeated history projection and superlinear grouping. No new placement IO or cache owner.

BR2 count-growth and BR3 mode-sequence regressions observed RED, then passed. Minimized size-changing count updates retire tentative drag, render/measure/reclamp; unchanged counts do nothing. Mode changes route reducer → render → measured geometry. Existing O(1) movement counts stay green. All webcast Node tests EXIT0; focused Go assembly/server tests PASS0.907s; inventory next before re-close.

Corrected-source native pair passed non-skipped11.242s, complete operator scenario8.60s. Raw log workshop/evidence/000850/native-qualification.log and JSON manifest retain SHA256 plus exact tested source Git blobs, avoiding reliance on a prose claim after the review host GPU startup refusal. Extended native panel check uses the actual painter for scroll/multipath/all-edge controls, and real layout for count growth. These are no longer unverified browser scenarios; operator acceptance itself still awaits the combined staged look (TL explicitly says no restage).


### 2026-10-23 — operator accepted; retained suite survived restart

Authenticated TL receipt a7dcd3bb reports the combined-stage operator acceptance PASSED (about20:45, “works now”) and directs full re-close and merge. This completes the outstanding operator portion of BR4; corrected-source native evidence remains the retained11.242s pass with source-blob manifest. Per prior TL direction no restage was performed after the viewport/reducer corrections.

Polled retained terminal9613 after restart: lease-3061314793 survived, acquired the suite lock after2366s, and is now running make test from the independent bac53fc98 snapshot; no restart or duplicate check. Next: inventory and full re-close while this existing recorded suite finishes; finalize only after its pass, then sdlc merge on SHIP. The final orchestration Plan checkbox remains open until the recorded check and landing finish; review may overlap the already running check under tl-worker rules.


### 2026-10-23 — close round2 REWORK; BR5 re-plan approved

Full review disposed BR1–4 as addressed, including mutation-proved regressions, source-bound native evidence and authenticated acceptance. It raised BR5 Critical: two sketches with the same/overlapping/edge-clamped bounds produce controls that cover earlier notes. Stopped at the third viewport delta and proposed replacing geometric per-sketch controls rather than adding collision heuristics. TL36312a99 approved ONE clamped Sketch notes(N) button for all eligible on-view sketches, opening a bounded scrolling keyed list with individually labeled Add/View note rows. Existing actions/final read-only view stay unchanged; no per-sketch controls/packing. Plan Revisions records state, alternative, single-selection owner, cost and composition-level native oracle. Earlier operator acceptance covers the panel; report the button change for the hub to tell them. Running old-source suite continues, no new runner yet; corrected source requires its own pre-close proof and close.


### 2026-10-23 — BR5 selector implemented and composed native hits verified

Replaced all per-sketch geometric controls with exactly one measured/clamped Sketch notes(N) summary. Its bounded scrolling keyed list identifies every current sketch by index/id and optional final-text snippet; rows route existing Add note or final read-only View note, rereading current membership. No collision packing or point-history changes. State preserves list mode across refresh, closes on choice/empty membership, retires stale handlers and disposes all UI. TDD observed missing summary and128 per-row measurements RED, then71 focused tests GREEN; N/2N proves linear row updates and one summary measurement.

All webcast Node tests EXIT0. Updated real Chrome fixture opens the single selector and independently hit-tests/clicks every one of24 identical/partially overlapping/edge-clamped sketch rows (scrolling the list), verifying the exact chosen final label and zero commands. Updated complete operator test still draws at prior ink coordinates without the summary blocking it. Native pair PASS7.392s non-skipped, complete operator5.60s; raw evidence and exact source-blob manifest refreshed in workshop/evidence/000850. Prior panel acceptance remains recorded; hub must tell the operator about the approved summary-button change. Existing old-source suite still runs and will finish normally; this source needs fresh pre-close proof, then full close and merge.

## Revisions

### 2026-10-23 — TL approved design (19f42a37)

Approved yellow300x320 within viewport, fixed header/scroll body, draggable header and minimized Notes(N) sharing one local position, keyboard move and cancel/resize cleanup. Anchored feedback/selection/stamps and849 invariants stay protected. Approved existing unfinished-label Add note and read-only completed View note; finished edits only if the operator later asks. Proceed through the full-flow design gate; live operator look at end. Added explicit Done-when coverage of these accepted semantics. No new permission pending.

### 2026-10-23 — plan-quality PQ-1

First full-flow gate refused before implementation solely for prose test-case inventories. Compressed all task test descriptions to named production-function strategies with adversarial input classes and mechanical invariants. Commands, independent native/browser proof and cost bounds remain. No product/design change or estimate yet; rerun gate next.

### 2026-10-23 — BR5 selector replaces per-sketch controls (TL36312a99)

The visible sketch-note discovery requirement is fulfilled by one clamped Sketch notes(N) selector opening a bounded keyed list, one Add note/View note row per eligible on-view sketch. This supersedes individual overlaid controls, whose collisions hid notes. Existing unfinished-label and final read-only semantics remain. Earlier operator acceptance covers the unchanged annotation panel; hub will communicate this approved button change.

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
