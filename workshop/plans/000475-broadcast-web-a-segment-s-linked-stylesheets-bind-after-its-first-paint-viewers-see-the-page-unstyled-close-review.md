# Boundary Review — pair#475 (whole-issue close)

| field | value |
|-------|-------|
| issue | 475 — broadcast-web: a segment's linked stylesheets bind after its first paint (viewers see the page unstyled) |
| repo | pair |
| issue file | workshop/issues/000475-broadcast-web-a-segment-s-linked-stylesheets-bind-after-its-first-paint-viewers-see-the-page-unstyled.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5d4d61469037a3d142d379583b4f506cb3016af..40d7bdb8bd5cdba1a13b7d8e6742af325bb6f1d5 |
| command | sdlc close --issue 475 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:08:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing #475's paint-hold diff; now checking the shell wiring and running the live tests.

Chrome dies on connect in this environment; checking whether that's environmental (pre-existing live tests too).

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The paint hold is built the way the spec decided. `render.js` draws each new top document (`head`/`seg`) into a second frame under the first. The first frame keeps the previous paint until the document's render-blocking `<link rel=stylesheet>`s are bound, or until `HOLD_MS` passes. `#page` always moves to the frame with the current document. The shell (`viewer.js`) and the shared `replayer.js` now look the frame up instead of keeping a reference. The known-gap ceiling and the release-time retry in `replay_test.go` match the amended Done-when. The atlas and lessons are updated. Nothing blocks the boundary, but two Important issues should be fixed:
- **Stacking bug.** The held frame's `z-index:1` paints over the shell's overlay: the pointer, the replay marks and the Paused label.
- **Implicit state.** The hold's state is spread over flags, a shared pair object and an inline style, and the hand-off between renderers has no test.

I could not run the live Chrome tests in this checkout. Every `WEBCAST_LIVE_CHROME=1` test fails at `Emulation.setDeviceMetricsOverride: cdp: read: EOF`, including the existing `TestLiveViewerPage`, so the environment is the cause, not this diff. So I have not independently checked the mutation claim ("a no-op `holdPaint` fails it") or the 3.55% figure. `go vet` and the non-live `go test` pass.

**1. Strengths**
- `render.js` `holdPaint`/`release`: the `pair.owner` guard means a renderer replaced mid-hold (a resync, a seek, a fresh `Replayer`) cannot release the new owner's hold. A `HOLD_MS` timer always lets the page through in the end, so a hold can never get stuck.
- `blockingSheets` (`render.js:~703`) follows the browser rule (not alternate, not disabled, media matches). The new live test covers the print-only case.
- The `openDoc` listener fix (`render.js:179-182`) is right: `doc.open()` erases listeners but keeps properties. Sending the redraw through `doc.__couchRenderer` lets the `doc.fonts` listener, added only once, follow whichever renderer draws the frame now.
- `knownGap{why, most}` (`replay_test.go:59-69`) makes the gap a guard: the test fails if the frame passes, and also if its best try goes over the 4% ceiling.
- `framePair` copies the iframe with `cloneNode(false)`, so the second frame keeps `sandbox="allow-same-origin"` and gets no `allow-scripts`. The sandbox stays the same.

**2. Critical findings**
None.

**3. Important findings**
- **The held frame covers the shell's overlay** (`render.js` `HELD = '1'`, `holdPaint`; `index.html:19-21`; `viewer.css:13-16`).
  - `.box` is `position:absolute` with no z-index, so it creates no stacking context.
  - Its siblings `.overlay` (the pointer), `.replay-layer` and `.paused-label` are positioned with `z-index:auto`.
  - So on every navigation, tab switch or resync, the held iframe paints over them for up to 500 ms. In a replay paused inside a hold it stays over them until the timer fires.
  - No test looks at the overlay during a hold.
  - Fix: give `.overlay` and `.paused-label` a z-index above `HELD` in `viewer.css`, or put the frames in their own stacking context (a wrapper with `isolation:isolate`).
- **ARCH-ORDER: the hold's state is implicit, and the hand-off is untested.**
  - The legal states are spread over `this.holding`, `this.topDrawn`, `this.holdTimer`, `pair.owner`, `pair.current`, and `other.style.zIndex === HELD`, which is an inline style used as the "a paint is held" flag.
  - The hand-off paths have no test: a second renderer starting `holdPaint` while a hold is in flight, `blankFrames` followed by the old renderer's timer, and a `seg` arriving while still holding.
  - The renderer calls the global `setTimeout`, so there is no clock seam (`Replayer` already injects `now`/`sleep`). The live test waits 700 ms of real time, so it sees one ordering only.
  - Fix: collapse the hold into one `{idle | holding(owner, deadline)}` state on the pair, with the zIndex/visibility styles applied as effects. Inject the timer, and add a hand-off test, e.g. a new `Replayer` or second renderer taking over mid-hold, then the old timer firing.

**4. Minor findings**
- **Replay page never wires `onDirty`** (`viewer.js` `shell.renderer`, used by `mountReplay`). On a paused replay page, a release or a stylesheet load does not re-flush, so a scroll clamped during the hold stays clamped. This is the same class the log fixed in the harness/`inBrowser`; the replay page was not covered in the same pass (ARCH-PURPOSE). This predates the diff, but the hold makes it more likely.
- **A stylesheet that errors still blocks for 500 ms.** `setLinkCSS` adds an `error` listener that calls `checkHold`, but an errored link's `sheet` is null, so it keeps blocking until `HOLD_MS`. A browser stops blocking on error.
- **`releaseAt` reaches into `Replayer` internals** (`harness.js`: `r.recs`, `r.next`, `r.feed`, `r.renderer`) and duplicates its stepping loop (ARCH-DRY). Consider a `Replayer` method for it.
- **`bound()` can wait indefinitely.** The `while (this.loads.size)` loop has no deadline; only the later sheet wait races `HOLD_MS`. A stalled proxy fetch would hang `replayAt` (ARCH-CONSTRAINTS).
- **The fidelity test ignores the ceiling.** `fidelity_live_test.go` reads `knownGap.why` but not `.most`, so a frame drifting above 4% under the fidelity check goes unreported.
- **A replaced renderer's `holdTimer` is never cleared.** It fires once and the `owner` guard makes it harmless, but nothing bounds how long it lives.

**5. Test coverage notes**
- `TestLiveRenderHoldsPaintUntilStylesheetsBind` covers holding, binding, a print-only link, the deadline, and the empty-srcset side fix. Its timing depends on the real clock.
- Untested: the hold hand-off between renderers, overlay stacking during a hold, and a stylesheet that errors.
- The release-time retry now applies to every failing frame of every fixture, not just rec-stress. That widens what the test accepts by up to `HOLD_MS` after a segment starts. It is acceptable by design, but worth knowing.
- I could not run the live tests here (Chrome dies on CDP connect); results are on trust from the log.

**6. Architectural notes**
- ARCH-DRY: Minor (`releaseAt` duplicates `Replayer` stepping).
- ARCH-PURE: the hold's logic is tied to the DOM and the global timer. Only `blockingSheets` is close to pure. Covered by the ARCH-ORDER finding.
- ARCH-PURPOSE: passes for the Done-when; Minor for the replay page's missing `onDirty`.
- ARCH-MOCK: pass. It runs in a real browser, with no new external dependency.
- ARCH-CONSTRAINTS: pass. `HOLD_MS` bounds the hold; the `loads` loop is the one unbounded wait (Minor).
- ARCH-SECURE: pass. The cloned frame keeps the sandbox, and nothing new runs in either frame.
- ARCH-ORDER: flagged (Important).
- ARCH-FUNERAL: pass. The second frame lives with the page, the held frame is emptied on release, and blobs are still revoked on reset.

**7. Plan revision recommendations**
None. The plan and the Revisions entry match the code. If the overlay fix changes where the frames sit in the stacking order, add a line to the spec's "paint holding" bullet.

```findings
findings:
  - id: new
    severity: Important
    family: overlay-stacking-order
    title: |
      The held frame's z-index:1 covers the shell's pointer overlay, replay marks and Paused label during a hold
    detail: |
      .box creates no stacking context, and .overlay, .replay-layer and .paused-label are positioned with z-index:auto, so the held iframe (render.js HELD='1') paints over them for up to HOLD_MS on every navigation or resync. Fix: give the overlay layers a z-index above HELD, or isolate the frames in their own stacking context.
  - id: new
    severity: Important
    family: state-machine-implicit
    title: |
      ARCH-ORDER: hold state spread over holding/topDrawn/holdTimer/pair.owner/pair.current/style.zIndex; renderer hand-off untested
    detail: |
      The legal states are not written down, and an inline style serves as the "held" flag. The global setTimeout gives tests no clock seam (the live test sleeps 700ms), and no test covers a second renderer or a blankFrames seek taking over mid-hold while the old timer fires. Collapse into one {idle|holding(owner,deadline)} state on the pair, inject the timer, and add a hand-off test.
  - id: new
    severity: Minor
    family: ondirty-wiring-sweep
    title: |
      The replay page's renderer (viewer.js shell.renderer via mountReplay) never wires onDirty
    detail: |
      On a paused replay page, a release or stylesheet load does not re-flush, so a scroll clamped during the hold stays clamped. This is the same class the log fixed for the harness/inBrowser (ARCH-PURPOSE).
  - id: new
    severity: Minor
    family: render-blocking-semantics
    title: |
      An errored stylesheet still blocks until HOLD_MS; the error listener's checkHold cannot release it
    detail: |
      An errored link's sheet stays null, so blockingSheets().every(l => l.sheet) is false. A browser stops blocking on error.
  - id: new
    severity: Minor
    family: replayer-encapsulation
    title: |
      harness releaseAt reaches into Replayer internals (recs/next/feed/renderer) and duplicates its stepping
  - id: new
    severity: Minor
    family: unbounded-wait
    title: |
      Renderer.bound() waits on this.loads with no deadline; a stalled proxy fetch hangs replayAt
  - id: new
    severity: Minor
    family: known-gap-ceiling
    title: |
      fidelity_live_test ignores knownGap.most, so a fidelity drift above the ceiling goes unreported
```

---

## Re-review — 2026-10-12T06:18:58-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 475 — broadcast-web: a segment's linked stylesheets bind after its first paint (viewers see the page unstyled) |
| repo | pair |
| issue file | workshop/issues/000475-broadcast-web-a-segment-s-linked-stylesheets-bind-after-its-first-paint-viewers-see-the-page-unstyled.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5d4d61469037a3d142d379583b4f506cb3016af..1bf1ad67f346172322c50295bf11c216959094ff |
| command | sdlc close --issue 475 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:18:58-07:00 |
| verdict | SHIP |

## Review

Prior-round findings: BR-1, BR-2 and BR-4 look fixed in the code. Next I'm checking BR-3 (onDirty wiring) and the rest, then I'll run the tests.

BR-5 and BR-6 are also fixed in the code. Now running the live paint-hold and replay tests in the checkout.

Chrome is crashing at startup (CDP read: EOF) inside the sandbox. I'll retry once outside the sandbox to get real results.

Chrome still crashes outside the sandbox. I'll check whether an older live test unrelated to this diff fails the same way, to rule out an environment problem.

The control test fails the same way, so Chrome can't run on this machine and none of the live tests can execute here. Checking for any non-Chrome JS tests, then a few static details.

```verdict
verdict: SHIP
confidence: medium
```

**Summary.** This diff does what #475 set out to do. The renderer now keeps the previous paint on screen until the new document's render-blocking `<link rel=stylesheet>`s are bound, have failed, or `HOLD_MS` has passed. All seven round-1 findings were re-checked against the code at head. BR-1, BR-2, BR-4, BR-5 and BR-7 are fixed. BR-3's stated symptom (a clamped scroll on a paused replay page) is fixed. BR-6's deadline is in the code, but there's no test that would fail without it, so under the claimed-fix rule it stays open as a Minor. Nothing blocks shipping; the remaining items are all Minor. Confidence is medium for one reason: Chrome exits at startup on this machine (`cdp: read: EOF`), both inside and outside the sandbox. The unrelated control test `TestLiveViewerPage` fails the same way, so the cause is the environment, not the diff. That means I could not run the four affected live tests (`TestLiveRenderHold*`, `TestLiveReplayFixtures`, `TestLiveFidelityFixtures`). The non-live suites (`go test ./cmd/internal/webcast/...`, including `TestNode`) and `go vet` pass.

Done-when: current — the amended items match the code: the synthetic hold test (`replay_test.go:495`), and `rec-stress` frame 0 kept as a known gap with a 0.04 ceiling, tried at `releaseAt` (`replay_test.go:69`, `replay_test.go:107-110`, `replay_test.go:119`).

### 1. Strengths
- **Stacking by document order, not z-index** (`render.js:454-455`). The frame being drawn moves before the held frame, so the shell's later siblings (`#overlay`, `.paused-label`, `.replay-layer` inside `#overlay`; see `index.html:20-22`) stay on top without any z-index. `TestLiveRenderHoldHandOff` checks this with the `corner overlay` probe.
- **The hold is one value**: `pair.hold = null | {owner, drawn, cancel}` (`render.js:442-464`). Ownership is a `getter` comparison (`render.js:467-470`), so a stale renderer's deadline does nothing, and the test checks exactly that (`fire(1)` → still holding).
- **Injected clock** (`render.js:328-333`). It lets the hand-off test drive takeover, a cancelled deadline, a new segment during the hold, the owner's deadline, and a failed sheet, one ordering at a time.
- **The `releaseAt` logic moved into `Replayer`** (`replayer.js:102-121`). The harness now calls it in one line instead of reaching into the replayer's internals.
- **The lesson** (`lessons.md:1216`) correctly identifies the `doc.open()` listener-erasure bug and the order-dependent numbers it caused.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **The replay page's pin/stroke marks go stale when a hold releases while paused** (`replay.js:279-306`). `replaySource` never sets `onDirty`, so a release only runs the default `flush()` (`render.js:381-384`). It does not run `deps.frame()`, which is where `drawMarks` and `layout` happen. Marks placed against the unstyled `#page` layout stay put after the stylesheet applies. Family `ondirty-wiring-sweep`, 2nd finding: see the findings block for the rule.
- **BR-6's deadline in `bound()`** (`render.js:491-493`) has no test that fails without it. A fake `timers` plus a never-settling entry in `loads` would cover it.
- `bound()`'s `settled` async block keeps running after the race times out. It's harmless, but its extent isn't bounded (ARCH-ORDER, cosmetic).
- `TestLiveRenderHoldsPaintUntilStylesheetsBind` still waits a real 700 ms (`replay_test.go:568`); it could use the same fake timers.

### 5. Test coverage notes
- The hand-off test covers takeover, a cancelled stale deadline, a segment during the hold, the owner's deadline, a failed sheet and overlay stacking. With the round-1 code, the `corner overlay` and `t.css failed` lines would read differently, so the test does catch the BR-1 and BR-4 regressions.
- No test covers `blankFrames` cancelling a hold in flight (`Replayer.reset` during a hold).
- The live tests were **not run** in this review (the Chrome problem above).

### 6. Architectural notes
- **ARCH-DRY: pass.** The `releaseAt` duplication is gone, and `blockingSheets`/`sheetSettled` are each defined once.
- **ARCH-PURE: pass.** Inherently DOM-bound; the clock seam is injected.
- **ARCH-PURPOSE: pass**, apart from the marks redraw above.
- **ARCH-MOCK: N/A.** No external service; the timers double is stateful (cleared ids are tracked).
- **ARCH-CONSTRAINTS: pass.** `HOLD_MS` 500 ms bounds both the hold and `bound()`.
- **ARCH-SECURE: pass.** The recorded `media` strings go to `matchMedia`, which degrades to "no match" on bad input.
- **ARCH-ORDER: pass.** The states are written down (`render.js:442-447`); `style.visibility` is now only derived by `hideHeld`.
- **ARCH-FUNERAL: pass.** There is one sibling frame per page frame, emptied on release; listeners are added `once` or per document.

### 7. Plan revision recommendations
None. The atlas (`atlas/webcast.md:189-196`) and `## Revisions` match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      z-index removed; render.js:455 moves the drawing frame before the held one, and the hand-off test probes the overlay corner at every step.
  - id: BR-2
    disposition: addressed
    note: |
      pair.hold is null or {owner,drawn,cancel} (render.js:442-464); timers injected (render.js:329); TestLiveRenderHoldHandOff drives takeover and a stale deadline.
  - id: BR-3
    disposition: addressed
    note: |
      Default onDirty now flushes on rAF (render.js:381-384), which re-clamps scroll on the replay page; the marks redraw sibling is raised as a new Minor.
  - id: BR-4
    disposition: addressed
    note: |
      sheetFailed WeakSet (render.js:419-420, 549-550); the hand-off test's t.css failed step would still show S without it.
  - id: BR-5
    disposition: addressed
    note: |
      releaseAt moved into Replayer (replayer.js:102-121); the harness is a one-line call (harness.js:41).
  - id: BR-6
    disposition: not-addressed
    note: |
      Deadline race present (render.js:491-493) but no test fails without it; fake timers plus a never-settling load would pin it.
  - id: BR-7
    disposition: addressed
    note: |
      Correct by design: fidelity compares at the frame's own time, where the hold shows the old paint; the ceiling is enforced at the release in replay_test.go:119 (comment at fidelity_live_test.go:48-50).
findings:
  - id: new
    severity: Minor
    family: ondirty-wiring-sweep
    title: |
      replaySource never routes onDirty to its draw pipeline, so a paused replay's marks stay at the pre-release layout
    detail: |
      This is the 2nd finding in family ondirty-wiring-sweep. Rule: every shell source that owns a renderer routes onDirty to its full paint pipeline (flush plus deps.frame), not flush alone. Enumeration: connect (viewer.js:64, done), replaySource (replay.js:319, missing; wrap deps.renderer to set r.onDirty = draw), inBrowser/harness (default flush is enough).
```

---

## Re-review — 2026-10-12T06:24:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 475 — broadcast-web: a segment's linked stylesheets bind after its first paint (viewers see the page unstyled) |
| repo | pair |
| issue file | workshop/issues/000475-broadcast-web-a-segment-s-linked-stylesheets-bind-after-its-first-paint-viewers-see-the-page-unstyled.md |
| boundary | whole-issue close |
| milestone | — |
| window | 919257f147d78b3d4f10ebfabe7a068bad4b320e..f01d0fd8b550d801211c739f1faf1e68ecd43637 |
| command | sdlc close --issue 475 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:24:43-07:00 |
| verdict | SHIP |

## Review

I'm checking the two open findings against the code now. BR-8's fix is in the window; next I'm checking whether BR-6's deadline was already at base.

```verdict
verdict: SHIP
confidence: high
```

This fix window has one code change: `replaySource` now sets each renderer's `onDirty` to `draw`. That finishes the `ondirty-wiring-sweep` list. `connect` was already wired at `viewer.js:65`, the replay source is now wired at `replay.js:322`, and the in-browser and harness renderers keep the default flush. The change is correct. `Player` builds its renderer through `deps.renderer(this.model)` (`replay.js:182`), and the wrapper passes the model through unchanged. `draw` is only scheduled, so it can't run before `player` and `ui` are assigned, and the `queued` guard merges repeated redraws into one. The rest of the window is not a scope change. The `lessons.md` hunk moves the #475 lesson with its text unchanged, which looks like the merge with main. The issue file changes only the status (codecomplete), the flow (quick → full), `actual_hours` and the log, and `## Spec` and `## Done when` are untouched. Both open findings are Minor and stay open only because neither has a regression test. Neither blocks the gate.

1. **Strengths**
   - `replay.js:319-323`: the fix wraps the injected renderer factory rather than reaching into `Player`. Every renderer `Player` builds after a seek or segment change gets `onDirty`, not just the first one. That covers the whole rule, not one site.
   - The comment says why the redraw takes the whole `draw` path: the marks and layout have to move too, as `connect()` routes it to its paint.
   - All 88 tests pass under `node --test tests/webcast/*.test.mjs`. The directory form, `node --test tests/webcast`, errors in this checkout, which is an invocation issue and not a test failure.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **BR-8:** the fix is right, but no test fails without it. `tests/webcast/replay.test.mjs:292` already builds `replaySource` with injected deps. A cheap test would give it a fake renderer and a recording `schedule`, call `r.onDirty()`, and assert that `frame` is called once more.
   - **BR-6:** `render.js` is not in this window. The `HOLD_MS` race in `bound()` (`render.js:516-524`) was there at base, but a stalled load still has no test proving the deadline.

5. **Test coverage:** neither change has a test that fails without it (see the two Minor items above).

6. **Architecture**
   - ARCH-DRY: pass. The one-line wrapper copies nothing.
   - ARCH-PURE: pass. The wiring is in the thin source shell, and `draw` is the existing injected pipeline.
   - ARCH-PURPOSE: pass. The `ondirty-wiring-sweep` list is now complete.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-6
    disposition: not-addressed
    note: |
      render.js not in this window; the HOLD_MS race in bound() is present but still no fake-timer test with a never-settling load proves it.
  - id: BR-8
    disposition: not-addressed
    note: |
      replay.js:322 wraps deps.renderer so r.onDirty = draw (correct, and it closes the sweep: connect, replaySource), but no regression test; a replaySource test in replay.test.mjs calling r.onDirty() and asserting frame() runs would pin it.
```
