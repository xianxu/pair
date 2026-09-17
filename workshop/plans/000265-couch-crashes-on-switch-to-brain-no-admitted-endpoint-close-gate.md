---
gate: boundary-review
issue: 265
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-16T16:36:48-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: 'resizeLayout is a fourth no-endpoint refusal: untyped, and a terminal resize still exits couch'
          detail: |-
            presenter.go:634 returns a plain errors.New("terminal: resize requires endpoint") for the
            same p.selected == nil condition the three converted sites report, and Console.onResize
            (console.go:1222-1228) hands it to c.terminalError. Reproduced with a scratch test from the
            durable (focus=actor, selected=nil) state that installObservedThreadActor (console.go:419)
            creates when c.active == "" on a foreground attach: "resize latched a terminal failure:
            terminal: resize requires endpoint". termcmd.inheritSize (presentation.go:394-398) has the
            same escalation. Convert to noDestination and classify in both consumers; the plan's own
            Task 2 acceptance says a fourth site belongs there, and both atlas files assert "three sites".
          family: routing-answer-escalation
          round: 1
        - id: BR-2
          severity: Critical
          title: pair term's repaint path still stops the mux on the already-typed ErrNoDestination
          detail: |-
            Task 5 swept only writeEvents. terminalMux.paintStripLocked (termcmd/presentation.go:353-363)
            guards on activeTabLocked() -- pair term's own tab model, the exact disagreement the shipped
            TestRoutingAnswerDoesNotStopTheMux encodes -- then feeds any UpdateChrome error to stopLocked.
            Reproduced on the identical fixture that test builds: "a repaint latched a mux failure:
            terminal: no destination for input: chrome requires selected endpoint". The plan names both a
            by-input and a by-repaint enumeration and applied the second to couchtty only.
          family: routing-answer-escalation
          round: 1
        - id: BR-3
          severity: Important
          title: The paintNow classification and the panel-check half of Task 3 have no test that fails without them
          detail: |-
            Measured and restored. Deleting the errors.Is block at console.go:1116-1123 leaves the couchtty
            package green apart from the two known environmental failures. Restoring the #255 bypass
            (key release / focus / blur calling deliverPresenterInput directly, no panel check) keeps all
            16 new assertions green AND passes the AST guard, because the door is the allowlisted callee.
            So only the classification half is pinned; the routing half named in the issue title is not.
          family: fix-without-red-oracle
          round: 1
        - id: BR-4
          severity: Important
          title: Every new non-fatal path is silent, and the state it degrades in is durable rather than transient
          detail: |-
            paintNow returns, deliverChildInput drops, deliverPresenterInput discards -- no trace, no log.
            Because (focus=actor, selected=nil) persists until the operator switches (console.go:419 never
            selects and finishOperation only forceSwitches for resume/recover), an operator sits with a
            blank viewport, no chrome and dead keys with zero signal -- the pair#273 symptom, now
            guaranteed silent. c.traceEvent is already the non-repainting channel the plan's notice
            analysis was looking for.
          family: unsignalled-degradation
          round: 1
        - id: BR-5
          severity: Minor
          title: deliverPresenterInput's error return has no consumers and misreports fatal errors as nil
          detail: |-
            All three call sites discard it, and it returns nil after terminalError has latched a genuine
            failure, so the signature advertises a distinction nothing uses.
          family: dead-return-value
          round: 1
        - id: BR-6
          severity: Minor
          title: atlas/couch.md and atlas/terminal.md both assert "three sites answer with it"
          detail: That count is already wrong given resizeLayout, and will need updating with the C1 fix.
          family: stale-enumeration-claim
          round: 1
        - id: BR-7
          severity: Minor
          title: presenter_test.go covers Input and UpdateChrome but not mouseInput's converted refusal
          detail: |-
            mouseInput's noDestination line is exercised only indirectly through couchtty's mouse
            subtests; a third table row in TestPresenterRefusalsWithoutADestinationAreClassifiable is
            one line.
          family: fix-without-red-oracle
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-16T16:58:54-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'resizeLayout typed; both consumers classify. Mutation-verified: reverting the typing reddens presenter_test resize-layout and termcmd resize; reverting onResize''s classification reddens TestResizeWithNoEndpointDoesNotStopTheConsole.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: paintStripLocked and inheritSize both classify; reverting each independently reddens exactly its row in TestRoutingAnswerDoesNotStopTheMuxOnRepaintOrResize.
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Both halves now have a red oracle. Measured: deleting paintNow''s errors.Is block reddens TestChromeRepaint...; restoring the pair#255 bypass reddens TestPanelDropsChildOnlyEventsWithoutAskingThePresenter via the trace detail, which the AST guard cannot see.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: traceNoDestination added on all four drop paths, via the non-repainting channel this finding named; reachability confirmed by the two mutations and asserted for the panel arm. See new finding on the atlas trace-event list and the unrecorded pair#273 lead.
          round: 2
        - id: BR-5
          disposition: addressed
          note: deliverPresenterInput is void; no caller can misread a latched failure as nil.
          round: 2
        - id: BR-6
          disposition: addressed
          note: Both atlas files now say four and name all four sites, with the by-answer enumeration rule spelled out. The same claim in the plan and issue is still stale -- raised as the class, not re-raised here.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Absence is now measured and documented at presenter_test.go:828-837; I confirmed Ready is ViewState's zero value (view.go:8), so an idle presenter cannot reach mouseInput's guard.
          round: 2
      findings:
        - id: BR-8
          severity: Important
          title: 'Third in family: the enumeration sweep stopped at atlas; five restatements stale, one cross-artifact claim false'
          detail: |-
            Do NOT fix these sites one at a time. Rule: when the ErrNoDestination
            producer/consumer enumeration changes, every durable artifact restating it is
            regenerated in the same commit, and a claim about another artifact is verified
            against that artifact before it is written. Measured prevalence, 7
            restatements / 2 swept / 5 stale: atlas/couch.md:1105-1113 omits the new
            no-destination trace event from the canonical COUCH_TRACE list; plan:57 Core
            concepts table lists 3 of 4 producers and none of onResize,
            paintStripLocked, inheritSize, traceDropped, traceNoDestination; plan:206
            Task 2 still says "both refusal sites" and the plan has no "## Revisions"
            section at all despite two gates changing its scope (AGENTS.md section 1);
            issue:105 Plan row still says "three, not two"; issue:384 claims the BR-4
            lead "is recorded there" in pair#273 -- grepped, it is not. Same rule in
            code: termcmd/presentation.go:208 and :361 assert a tab can exist while
            nothing is admitted, which I could not substantiate (admitTab selects
            atomically, removeTab reselects first, Retire refuses while selected,
            nothing calls Panel) -- keep the guards, correct the claim to "defensive,
            unreachable today".
          family: stale-enumeration-claim
          round: 2
        - id: BR-9
          severity: Important
          title: 'Third in family: the AST guard pins 1 of the 4 answer-producing methods, and termcmd has no guard at all'
          detail: |-
            Do NOT add a seventh classification site and stop. Rule: every console/mux
            call to a presenter method that can answer ErrNoDestination must classify
            it, enforced on the method set rather than on Input alone.
            input_door_guard_test.go matches only <x>.presenter.Input, so UpdateChrome,
            Resize and ResizeLayout -- three quarters of the answer surface, and the
            exact two that round 1 caught -- have no mechanical pin. A new
            c.presenter.UpdateChrome forwarding to terminalError compiles, passes the
            guard, and reproduces the crash with no input involved. Cheap encoding:
            extend the existing AST walk to flag a call to any of {Input, UpdateChrome,
            Resize, ResizeLayout} on *.presenter whose enclosing function does not
            reference terminal.ErrNoDestination. I verified this is satisfiable today in
            both packages with no code movement.
          family: routing-answer-escalation
          round: 2
        - id: BR-10
          severity: Minor
          title: traceEvent's doc comment is now orphaned onto traceDropped
          detail: |-
            trace.go:172-176 -- the inserted traceDropped sits between traceEvent's
            comment and traceEvent, so traceDropped carries a two-paragraph comment
            whose first half describes a different function and traceEvent is
            undocumented. Move traceDropped below traceEvent.
          family: orphaned-doc-comment
          round: 2
        - id: BR-11
          severity: Minor
          title: lessons.md ships the same lesson twice in the same section
          detail: |-
            workshop/lessons.md:5119 "Enumerate the answer, not the call site" and
            :5145 "Enumerate the ANSWER, not the callers" are the same rule, recorded
            once from the plan gate and once from BR-1. lessons.md is read at session
            start; a duplicated rule reads as two rules (ARCH-DRY). Merge, keeping
            BR-1's measurement that the same mistake escaped two gates.
          family: duplicated-lesson
          round: 2
        - id: BR-12
          severity: Minor
          title: traceDropped writes the error text where the sibling helper documents "a code, never the error's text"
          detail: |-
            Substantively safe -- the payload is a static reason plus %q-quoted endpoint
            IDs, which the trace format permits as addresses, and %q escapes tab and
            newline so the TSV framing cannot break. But reattachDoneDetail's comment
            states the opposite convention and neither cites the other. One sentence at
            atlas/couch.md:1115 settling which applies.
          family: stale-enumeration-claim
          round: 2
      blocked: true
---

# Gate ledger — pair#265 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-16T16:36:48-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `routing-answer-escalation` resizeLayout is a fourth no-endpoint refusal: untyped, and a terminal resize still exits couch
  presenter.go:634 returns a plain errors.New("terminal: resize requires endpoint") for the
  same p.selected == nil condition the three converted sites report, and Console.onResize
  (console.go:1222-1228) hands it to c.terminalError. Reproduced with a scratch test from the
  durable (focus=actor, selected=nil) state that installObservedThreadActor (console.go:419)
  creates when c.active == "" on a foreground attach: "resize latched a terminal failure:
  terminal: resize requires endpoint". termcmd.inheritSize (presentation.go:394-398) has the
  same escalation. Convert to noDestination and classify in both consumers; the plan's own
  Task 2 acceptance says a fourth site belongs there, and both atlas files assert "three sites".
- **BR-2** [Critical] `routing-answer-escalation` pair term's repaint path still stops the mux on the already-typed ErrNoDestination
  Task 5 swept only writeEvents. terminalMux.paintStripLocked (termcmd/presentation.go:353-363)
  guards on activeTabLocked() -- pair term's own tab model, the exact disagreement the shipped
  TestRoutingAnswerDoesNotStopTheMux encodes -- then feeds any UpdateChrome error to stopLocked.
  Reproduced on the identical fixture that test builds: "a repaint latched a mux failure:
  terminal: no destination for input: chrome requires selected endpoint". The plan names both a
  by-input and a by-repaint enumeration and applied the second to couchtty only.
- **BR-3** [Important] `fix-without-red-oracle` The paintNow classification and the panel-check half of Task 3 have no test that fails without them
  Measured and restored. Deleting the errors.Is block at console.go:1116-1123 leaves the couchtty
  package green apart from the two known environmental failures. Restoring the #255 bypass
  (key release / focus / blur calling deliverPresenterInput directly, no panel check) keeps all
  16 new assertions green AND passes the AST guard, because the door is the allowlisted callee.
  So only the classification half is pinned; the routing half named in the issue title is not.
- **BR-4** [Important] `unsignalled-degradation` Every new non-fatal path is silent, and the state it degrades in is durable rather than transient
  paintNow returns, deliverChildInput drops, deliverPresenterInput discards -- no trace, no log.
  Because (focus=actor, selected=nil) persists until the operator switches (console.go:419 never
  selects and finishOperation only forceSwitches for resume/recover), an operator sits with a
  blank viewport, no chrome and dead keys with zero signal -- the pair#273 symptom, now
  guaranteed silent. c.traceEvent is already the non-repainting channel the plan's notice
  analysis was looking for.
- **BR-5** [Minor] `dead-return-value` deliverPresenterInput's error return has no consumers and misreports fatal errors as nil
  All three call sites discard it, and it returns nil after terminalError has latched a genuine
  failure, so the signature advertises a distinction nothing uses.
- **BR-6** [Minor] `stale-enumeration-claim` atlas/couch.md and atlas/terminal.md both assert "three sites answer with it"
  That count is already wrong given resizeLayout, and will need updating with the C1 fix.
- **BR-7** [Minor] `fix-without-red-oracle` presenter_test.go covers Input and UpdateChrome but not mouseInput's converted refusal
  mouseInput's noDestination line is exercised only indirectly through couchtty's mouse
  subtests; a third table row in TestPresenterRefusalsWithoutADestinationAreClassifiable is
  one line.

## Round 2 — 2026-09-16T16:58:54-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — resizeLayout typed; both consumers classify. Mutation-verified: reverting the typing reddens presenter_test resize-layout and termcmd resize; reverting onResize's classification reddens TestResizeWithNoEndpointDoesNotStopTheConsole.
- BR-2 — addressed — paintStripLocked and inheritSize both classify; reverting each independently reddens exactly its row in TestRoutingAnswerDoesNotStopTheMuxOnRepaintOrResize.
- BR-3 — addressed — Both halves now have a red oracle. Measured: deleting paintNow's errors.Is block reddens TestChromeRepaint...; restoring the pair#255 bypass reddens TestPanelDropsChildOnlyEventsWithoutAskingThePresenter via the trace detail, which the AST guard cannot see.
- BR-4 — addressed — traceNoDestination added on all four drop paths, via the non-repainting channel this finding named; reachability confirmed by the two mutations and asserted for the panel arm. See new finding on the atlas trace-event list and the unrecorded pair#273 lead.
- BR-5 — addressed — deliverPresenterInput is void; no caller can misread a latched failure as nil.
- BR-6 — addressed — Both atlas files now say four and name all four sites, with the by-answer enumeration rule spelled out. The same claim in the plan and issue is still stale -- raised as the class, not re-raised here.
- BR-7 — addressed — Absence is now measured and documented at presenter_test.go:828-837; I confirmed Ready is ViewState's zero value (view.go:8), so an idle presenter cannot reach mouseInput's guard.

### Raised

- **BR-8** [Important] `stale-enumeration-claim` Third in family: the enumeration sweep stopped at atlas; five restatements stale, one cross-artifact claim false
  Do NOT fix these sites one at a time. Rule: when the ErrNoDestination
  producer/consumer enumeration changes, every durable artifact restating it is
  regenerated in the same commit, and a claim about another artifact is verified
  against that artifact before it is written. Measured prevalence, 7
  restatements / 2 swept / 5 stale: atlas/couch.md:1105-1113 omits the new
  no-destination trace event from the canonical COUCH_TRACE list; plan:57 Core
  concepts table lists 3 of 4 producers and none of onResize,
  paintStripLocked, inheritSize, traceDropped, traceNoDestination; plan:206
  Task 2 still says "both refusal sites" and the plan has no "## Revisions"
  section at all despite two gates changing its scope (AGENTS.md section 1);
  issue:105 Plan row still says "three, not two"; issue:384 claims the BR-4
  lead "is recorded there" in pair#273 -- grepped, it is not. Same rule in
  code: termcmd/presentation.go:208 and :361 assert a tab can exist while
  nothing is admitted, which I could not substantiate (admitTab selects
  atomically, removeTab reselects first, Retire refuses while selected,
  nothing calls Panel) -- keep the guards, correct the claim to "defensive,
  unreachable today".
- **BR-9** [Important] `routing-answer-escalation` Third in family: the AST guard pins 1 of the 4 answer-producing methods, and termcmd has no guard at all
  Do NOT add a seventh classification site and stop. Rule: every console/mux
  call to a presenter method that can answer ErrNoDestination must classify
  it, enforced on the method set rather than on Input alone.
  input_door_guard_test.go matches only <x>.presenter.Input, so UpdateChrome,
  Resize and ResizeLayout -- three quarters of the answer surface, and the
  exact two that round 1 caught -- have no mechanical pin. A new
  c.presenter.UpdateChrome forwarding to terminalError compiles, passes the
  guard, and reproduces the crash with no input involved. Cheap encoding:
  extend the existing AST walk to flag a call to any of {Input, UpdateChrome,
  Resize, ResizeLayout} on *.presenter whose enclosing function does not
  reference terminal.ErrNoDestination. I verified this is satisfiable today in
  both packages with no code movement.
- **BR-10** [Minor] `orphaned-doc-comment` traceEvent's doc comment is now orphaned onto traceDropped
  trace.go:172-176 -- the inserted traceDropped sits between traceEvent's
  comment and traceEvent, so traceDropped carries a two-paragraph comment
  whose first half describes a different function and traceEvent is
  undocumented. Move traceDropped below traceEvent.
- **BR-11** [Minor] `duplicated-lesson` lessons.md ships the same lesson twice in the same section
  workshop/lessons.md:5119 "Enumerate the answer, not the call site" and
  :5145 "Enumerate the ANSWER, not the callers" are the same rule, recorded
  once from the plan gate and once from BR-1. lessons.md is read at session
  start; a duplicated rule reads as two rules (ARCH-DRY). Merge, keeping
  BR-1's measurement that the same mistake escaped two gates.
- **BR-12** [Minor] `stale-enumeration-claim` traceDropped writes the error text where the sibling helper documents "a code, never the error's text"
  Substantively safe -- the payload is a static reason plus %q-quoted endpoint
  IDs, which the trace format permits as addresses, and %q escapes tab and
  newline so the TSV framing cannot break. But reattachDoneDetail's comment
  states the opposite convention and neither cites the other. One sentence at
  atlas/couch.md:1115 settling which applies.

## Open findings

- **BR-8** [Important] `stale-enumeration-claim` Third in family: the enumeration sweep stopped at atlas; five restatements stale, one cross-artifact claim false
- **BR-9** [Important] `routing-answer-escalation` Third in family: the AST guard pins 1 of the 4 answer-producing methods, and termcmd has no guard at all
- **BR-10** [Minor] `orphaned-doc-comment` traceEvent's doc comment is now orphaned onto traceDropped
- **BR-11** [Minor] `duplicated-lesson` lessons.md ships the same lesson twice in the same section
- **BR-12** [Minor] `stale-enumeration-claim` traceDropped writes the error text where the sibling helper documents "a code, never the error's text"
