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

## Open findings

- **BR-1** [Critical] `routing-answer-escalation` resizeLayout is a fourth no-endpoint refusal: untyped, and a terminal resize still exits couch
- **BR-2** [Critical] `routing-answer-escalation` pair term's repaint path still stops the mux on the already-typed ErrNoDestination
- **BR-3** [Important] `fix-without-red-oracle` The paintNow classification and the panel-check half of Task 3 have no test that fails without them
- **BR-4** [Important] `unsignalled-degradation` Every new non-fatal path is silent, and the state it degrades in is durable rather than transient
- **BR-5** [Minor] `dead-return-value` deliverPresenterInput's error return has no consumers and misreports fatal errors as nil
- **BR-6** [Minor] `stale-enumeration-claim` atlas/couch.md and atlas/terminal.md both assert "three sites answer with it"
- **BR-7** [Minor] `fix-without-red-oracle` presenter_test.go covers Input and UpdateChrome but not mouseInput's converted refusal
