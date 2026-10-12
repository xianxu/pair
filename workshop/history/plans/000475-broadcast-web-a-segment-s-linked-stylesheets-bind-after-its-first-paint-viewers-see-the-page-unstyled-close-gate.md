---
gate: boundary-review
issue: 475
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T06:08:07-07:00"
      dispatched: "2026-10-12T06:03:45-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: The held frame's z-index:1 covers the shell's pointer overlay, replay marks and Paused label during a hold
          detail: '.box creates no stacking context, and .overlay, .replay-layer and .paused-label are positioned with z-index:auto, so the held iframe (render.js HELD=''1'') paints over them for up to HOLD_MS on every navigation or resync. Fix: give the overlay layers a z-index above HELD, or isolate the frames in their own stacking context.'
          family: overlay-stacking-order
          round: 1
        - id: BR-2
          severity: Important
          title: 'ARCH-ORDER: hold state spread over holding/topDrawn/holdTimer/pair.owner/pair.current/style.zIndex; renderer hand-off untested'
          detail: The legal states are not written down, and an inline style serves as the "held" flag. The global setTimeout gives tests no clock seam (the live test sleeps 700ms), and no test covers a second renderer or a blankFrames seek taking over mid-hold while the old timer fires. Collapse into one {idle|holding(owner,deadline)} state on the pair, inject the timer, and add a hand-off test.
          family: state-machine-implicit
          round: 1
        - id: BR-3
          severity: Minor
          title: The replay page's renderer (viewer.js shell.renderer via mountReplay) never wires onDirty
          detail: On a paused replay page, a release or stylesheet load does not re-flush, so a scroll clamped during the hold stays clamped. This is the same class the log fixed for the harness/inBrowser (ARCH-PURPOSE).
          family: ondirty-wiring-sweep
          round: 1
        - id: BR-4
          severity: Minor
          title: An errored stylesheet still blocks until HOLD_MS; the error listener's checkHold cannot release it
          detail: An errored link's sheet stays null, so blockingSheets().every(l => l.sheet) is false. A browser stops blocking on error.
          family: render-blocking-semantics
          round: 1
        - id: BR-5
          severity: Minor
          title: harness releaseAt reaches into Replayer internals (recs/next/feed/renderer) and duplicates its stepping
          family: replayer-encapsulation
          round: 1
        - id: BR-6
          severity: Minor
          title: Renderer.bound() waits on this.loads with no deadline; a stalled proxy fetch hangs replayAt
          family: unbounded-wait
          round: 1
        - id: BR-7
          severity: Minor
          title: fidelity_live_test ignores knownGap.most, so a fidelity drift above the ceiling goes unreported
          family: known-gap-ceiling
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T06:18:58-07:00"
      dispatched: "2026-10-12T06:15:37-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: z-index removed; render.js:455 moves the drawing frame before the held one, and the hand-off test probes the overlay corner at every step.
          round: 2
        - id: BR-2
          disposition: addressed
          note: pair.hold is null or {owner,drawn,cancel} (render.js:442-464); timers injected (render.js:329); TestLiveRenderHoldHandOff drives takeover and a stale deadline.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Default onDirty now flushes on rAF (render.js:381-384), which re-clamps scroll on the replay page; the marks redraw sibling is raised as a new Minor.
          round: 2
        - id: BR-4
          disposition: addressed
          note: sheetFailed WeakSet (render.js:419-420, 549-550); the hand-off test's t.css failed step would still show S without it.
          round: 2
        - id: BR-5
          disposition: addressed
          note: releaseAt moved into Replayer (replayer.js:102-121); the harness is a one-line call (harness.js:41).
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: Deadline race present (render.js:491-493) but no test fails without it; fake timers plus a never-settling load would pin it.
          round: 2
        - id: BR-7
          disposition: addressed
          note: 'Correct by design: fidelity compares at the frame''s own time, where the hold shows the old paint; the ceiling is enforced at the release in replay_test.go:119 (comment at fidelity_live_test.go:48-50).'
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: replaySource never routes onDirty to its draw pipeline, so a paused replay's marks stay at the pre-release layout
          detail: 'This is the 2nd finding in family ondirty-wiring-sweep. Rule: every shell source that owns a renderer routes onDirty to its full paint pipeline (flush plus deps.frame), not flush alone. Enumeration: connect (viewer.js:64, done), replaySource (replay.js:319, missing; wrap deps.renderer to set r.onDirty = draw), inBrowser/harness (default flush is enough).'
          family: ondirty-wiring-sweep
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 1bf1ad67f346172322c50295bf11c216959094ff
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T06:24:43-07:00"
      dispatched: "2026-10-12T06:23:52-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: not-addressed
          note: render.js not in this window; the HOLD_MS race in bound() is present but still no fake-timer test with a never-settling load proves it.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: 'replay.js:322 wraps deps.renderer so r.onDirty = draw (correct, and it closes the sweep: connect, replaySource), but no regression test; a replaySource test in replay.test.mjs calling r.onDirty() and asserting frame() runs would pin it.'
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: f01d0fd8b550d801211c739f1faf1e68ecd43637
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T06:41:02-07:00"
      dispatched: "2026-10-12T06:40:25-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: not-addressed
          note: No change in this window; still no fake-timer test with a never-settling load proving bound() releases at HOLD_MS.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: No change in this window; replay.js still sets r.onDirty = draw, but no replay.test.mjs test calls r.onDirty() and asserts frame() runs.
          round: 4
      recipe: fix-review
      isolation: fenced
      reviewed: accf2497abaa2d6aa292ec0b0ce5508d4fd4f3e4
      blocked: false
---

# Gate ledger — pair#475 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T06:08:07-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `overlay-stacking-order` The held frame's z-index:1 covers the shell's pointer overlay, replay marks and Paused label during a hold
  .box creates no stacking context, and .overlay, .replay-layer and .paused-label are positioned with z-index:auto, so the held iframe (render.js HELD='1') paints over them for up to HOLD_MS on every navigation or resync. Fix: give the overlay layers a z-index above HELD, or isolate the frames in their own stacking context.
- **BR-2** [Important] `state-machine-implicit` ARCH-ORDER: hold state spread over holding/topDrawn/holdTimer/pair.owner/pair.current/style.zIndex; renderer hand-off untested
  The legal states are not written down, and an inline style serves as the "held" flag. The global setTimeout gives tests no clock seam (the live test sleeps 700ms), and no test covers a second renderer or a blankFrames seek taking over mid-hold while the old timer fires. Collapse into one {idle|holding(owner,deadline)} state on the pair, inject the timer, and add a hand-off test.
- **BR-3** [Minor] `ondirty-wiring-sweep` The replay page's renderer (viewer.js shell.renderer via mountReplay) never wires onDirty
  On a paused replay page, a release or stylesheet load does not re-flush, so a scroll clamped during the hold stays clamped. This is the same class the log fixed for the harness/inBrowser (ARCH-PURPOSE).
- **BR-4** [Minor] `render-blocking-semantics` An errored stylesheet still blocks until HOLD_MS; the error listener's checkHold cannot release it
  An errored link's sheet stays null, so blockingSheets().every(l => l.sheet) is false. A browser stops blocking on error.
- **BR-5** [Minor] `replayer-encapsulation` harness releaseAt reaches into Replayer internals (recs/next/feed/renderer) and duplicates its stepping
- **BR-6** [Minor] `unbounded-wait` Renderer.bound() waits on this.loads with no deadline; a stalled proxy fetch hangs replayAt
- **BR-7** [Minor] `known-gap-ceiling` fidelity_live_test ignores knownGap.most, so a fidelity drift above the ceiling goes unreported

## Round 2 — 2026-10-12T06:18:58-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — z-index removed; render.js:455 moves the drawing frame before the held one, and the hand-off test probes the overlay corner at every step.
- BR-2 — addressed — pair.hold is null or {owner,drawn,cancel} (render.js:442-464); timers injected (render.js:329); TestLiveRenderHoldHandOff drives takeover and a stale deadline.
- BR-3 — addressed — Default onDirty now flushes on rAF (render.js:381-384), which re-clamps scroll on the replay page; the marks redraw sibling is raised as a new Minor.
- BR-4 — addressed — sheetFailed WeakSet (render.js:419-420, 549-550); the hand-off test's t.css failed step would still show S without it.
- BR-5 — addressed — releaseAt moved into Replayer (replayer.js:102-121); the harness is a one-line call (harness.js:41).
- BR-6 — not-addressed — Deadline race present (render.js:491-493) but no test fails without it; fake timers plus a never-settling load would pin it.
- BR-7 — addressed — Correct by design: fidelity compares at the frame's own time, where the hold shows the old paint; the ceiling is enforced at the release in replay_test.go:119 (comment at fidelity_live_test.go:48-50).

### Raised

- **BR-8** [Minor] `ondirty-wiring-sweep` replaySource never routes onDirty to its draw pipeline, so a paused replay's marks stay at the pre-release layout
  This is the 2nd finding in family ondirty-wiring-sweep. Rule: every shell source that owns a renderer routes onDirty to its full paint pipeline (flush plus deps.frame), not flush alone. Enumeration: connect (viewer.js:64, done), replaySource (replay.js:319, missing; wrap deps.renderer to set r.onDirty = draw), inBrowser/harness (default flush is enough).

## Round 3 — 2026-10-12T06:24:43-07:00 (claude) — passed

### Disposed

- BR-6 — not-addressed — render.js not in this window; the HOLD_MS race in bound() is present but still no fake-timer test with a never-settling load proves it.
- BR-8 — not-addressed — replay.js:322 wraps deps.renderer so r.onDirty = draw (correct, and it closes the sweep: connect, replaySource), but no regression test; a replaySource test in replay.test.mjs calling r.onDirty() and asserting frame() runs would pin it.

## Round 4 — 2026-10-12T06:41:02-07:00 (claude) — passed

### Disposed

- BR-6 — not-addressed — No change in this window; still no fake-timer test with a never-settling load proving bound() releases at HOLD_MS.
- BR-8 — not-addressed — No change in this window; replay.js still sets r.onDirty = draw, but no replay.test.mjs test calls r.onDirty() and asserts frame() runs.

## Open findings

- **BR-6** [Minor] `unbounded-wait` Renderer.bound() waits on this.loads with no deadline; a stalled proxy fetch hangs replayAt
- **BR-8** [Minor] `ondirty-wiring-sweep` replaySource never routes onDirty to its draw pipeline, so a paused replay's marks stay at the pre-release layout
