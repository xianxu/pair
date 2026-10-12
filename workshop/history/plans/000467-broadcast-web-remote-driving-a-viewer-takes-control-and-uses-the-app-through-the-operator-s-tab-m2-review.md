# Boundary Review — pair#467 (milestone M2)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c80750739cf50e363172c9f28c5aa46dbbb49b9f..7594bb856d1f049e84cab9cfcd9b2e74cab9985a |
| command | sdlc milestone-close --issue 467 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:25:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M2 diff now: I've read the plan and the code changes, and I'm checking drive.js and viewer.js behaviour against model.js, annotate.js and the server's limits.

Confirmed two defects with a quick probe: long pastes produce batches over the server's 8 KB limit, and a drag past the frame's edge sends negative coordinates. Now running the JS and Go test suites.

I couldn't run the live test here: headless Chrome crashes on startup in this environment, and the old `TestLiveViewerPage` fails the same way. I'm writing up the verdict now.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what Tasks 8 and 9 promise. `drive.js` has its pure parts (`batcher`, `gestures`, `keyAct`, `driveView`, `joinSeat`, `seatPoster`) plus a thin `mountDrive` shell. `viewer.js` takes a seat before it opens the EventSource and streams with `?viewer=<seat>`. On 404 it streams with no drive UI. `annotate.js` adopts the seat id as the note author, the server embeds and serves `drive.js`/`drive.css`, and the atlas describes the new surface. `node --test tests/webcast/*.test.mjs` passes (90/90), and the Go drive and server tests pass. I could not run `TestDriveLiveTwoViewers`: Chrome exits during `Emulation.setDeviceMetricsOverride` (cdp read: EOF), and the existing `TestLiveViewerPage` fails the same way, so the cause is this environment, not M2.

Two shell defects are real and I reproduced both with a probe against `drive.js`. In both, the shell sends a batch that `drive.ParseBatch` / `admitBody` will reject. Neither is a crash, and both show the server's message to the viewer, so they don't block the gate. They should be fixed before M3 builds on the shell.

**Strengths**
- `drive.js:461-501`: the batcher's one-in-flight, ordered, coalescing queue is pinned by a seeded property test over 200 seeds with a fake server that answers late and refuses some requests (`tests/webcast/drive.test.mjs`). Failing seeds can be replayed.
- `gestures` is pure, with injected `toPage`/`nodeAt`/`emit`, and is tested on replayed traces. The traces include the malformed cases the plan asks for: a cancel mid-drag, a late second finger, and an up/move for a finger that was never down.
- Who drives comes only from the stream's `ctl` records (`driveView`, `mountDrive`'s `onRecord`). The page never guesses, which matches #466's single source of truth. Names and server messages are inserted as `textContent` only (ARCH-SECURE pass).
- The seat comes before the stream (`viewer.js:812-829`), and `couchViewer.ready` / `onRecord` wait for `started`, so a listener registers before the first record arrives.
- The M1 advisory is addressed with a real regression: `TestDriveSeatTableSweepsIdleKeys` drives the injected clock (`server_drive.go` `d.now`).

**Critical findings**
None.

**Important findings**
1. **Batches can exceed the server's 8 KB body limit.** (ARCH-CONSTRAINTS, `drive.js:467,488-490`)
   - `batcher` caps a batch at 64 acts and splits text into 1000-rune pieces, but it never caps the serialized size. `drive.MaxBody` is `8 << 10`.
   - Probe: pasting 9000 ASCII characters produced one 9,198-byte batch. 3000 CJK characters produced a 9,078-byte batch (UTF-8).
   - `admitBody` answers 413 "request too large", and the batcher drops the whole paste.
   - Fix: export the byte budget beside `MAX_TEXT`. In `go()`, take acts while the running `JSON.stringify` UTF-8 size stays under it, and split text by bytes as well as runes. Add a batcher test where every sent body is ≤ `MaxBody`.
2. **A drag past the frame's left or top edge sends negative coordinates.** (`drive.js:553-561,577`; `viewer.js` `toPage` doesn't clamp)
   - The layer captures the pointer, so moves outside `#box` keep arriving. `toPage` turns them into negative page coordinates.
   - `ParseBatch`'s `coord()` requires ≥ 0, so the batch gets a 400. The queue is dropped, and the closing `up` (also negative) is refused too.
   - Result: the cast tab saw a `down` and never sees the `up`. Probe output: `down (50,50) → move (-50,50) → up (-60,50)`.
   - Fix: clamp the page point to ≥ 0, and to the page box, in `gestures`' `pt()`/`at()`. Add a gesture trace test that leaves the frame.
3. **The live test uses a spy instead of `FakeInjector`, and the plan doesn't say so.** (`viewer_drive_live_test.go:163-180`)
   - Task 9 specifies a "`MemController` over `FakeInjector` fed by the hub". The test uses `spyInjector`, which accepts every batch.
   - So the injector's rules never run end to end against the real shell: the segment check, `Confine`, and the stale check. A wrong `seg` from `v.model.seg` would still pass.
   - This is the 2nd finding in family `plan-drift-unrecorded`. Don't just fix this one. The rule: each Task's named files, fixtures and test doubles are part of the plan's contract, and any deviation gets a `## Revisions` line in the milestone's closing commit.
   - Sweep M2 against that rule. Known deviations:
     - spy instead of `FakeInjector`;
     - button label "Cancel request", where the plan says "Waiting… (cancel)";
     - 1024×700 desktop metrics with touch emulation, where the plan says "tablet width".
   - Then either swap in `FakeInjector` over the hub's reduced state (preferred: it is the stateful fake ARCH-MOCK asks for), or record why the spy is enough.

**Minor findings**
- **Rejoin offer is lost** (`drive.js:674-682`): the next `render()` (any `ctl`/`seg`/`stop`) overwrites `offerRejoin`'s button text and handler, so after a 401 the offer disappears on the next record.
- **Repeated helpers** (ARCH-DRY): the toast (`drive.js:646-655`) is a copy of `annotate.js:293-302`, both using the same `.annotate-msg` class, so the two can overlap. `joinSeat`/`seatPoster` repeat `collab.poster`'s request options; adding a `headers` parameter to `poster` would cover both.
- **Keyup lost for chords** (`keyAct`): if the user releases the modifier before the letter, the letter's keyup is a non-chord printable and returns null, so the cast tab never gets the key-up.
- **Hover while a button is held**: a press on "nothing" (`press = null`) with the button still down sends hover moves without a button.
- **Pause logic outside the pure part**: the paused note and layer gating (`render`, `drive.js:680-685`) could move into `driveView(ctl, me, live)` so a table test covers them (ARCH-PURE).
- **No client pacing**: the batcher doesn't pace to the server's input budget of 30/s, burst 30. Fast key auto-repeat on a low-latency link could hit a 429 and drop the queue.
- **`drive.css` untested**: no served-asset test covers it. `TestViewerImportsAreServed` reaches `drive.js` through the import walk, but CSS links aren't walked.

**Test coverage notes**
- The JS unit tests are strong for ordering and gesture shapes. They don't cover the server's wire contract: no test checks that `batcher`'s output parses under `ParseBatch`'s limits (byte size, coordinate ≥ 0). That is exactly where Important 1 and 2 live.
  - A cheap backstop: a Go test that feeds batches serialized from `drive.js` into `ParseBatch`, or a JS fixture that mirrors `MaxBody`/`coord` and asserts against it.
- I couldn't verify the live test in this environment (Chrome EOF). The main agent should confirm `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run DriveLive` passes locally.

**Architecture**
- **ARCH-DRY:** flagged (Minor: the toast and poster copies).
- **ARCH-PURE:** passes. Pure helpers are exported and tested; `mountDrive` is glue. Minor note on the paused-view logic above.
- **ARCH-PURPOSE:** passes. Every M2 deliverable is present. Couch and the fixture belong to M3/M4.
- **ARCH-MOCK:** flagged under Important 3, where a stateless spy stands in for the stateful `FakeInjector`.
- **ARCH-CONSTRAINTS:** flagged (Important 1). The client's batch budget isn't derived from `drive.MaxBody`.
- **ARCH-SECURE:** passes. `joinSeat` checks the seat's shape, names and messages go in as `textContent`, the seat key stays in memory, and requests omit credentials.
- **ARCH-ORDER:** passes, with a note. The shell's state is the `driveView` state plus `batch`, and `batch`'s lifetime follows `view.state`. Gating by live state is spread across `render`.
- **ARCH-FUNERAL:** passes. Nothing durable is created: the DOM nodes and listeners mount once per page, and the seat dies with the stream.

**Architectural notes for upcoming work**
- M3/M4 will drive the same routes. Make `drive.js` constants derive from, or be checked against, `drive/act.go` (`MaxBody`, `MaxActs`, `MaxText`, `coord`) so the client and server can't drift apart.

**Plan revision recommendations**
- `## Revisions` entry for M2:
  - the live test's injector (spy or `FakeInjector`, with the reason);
  - the button labels as built ("Ask to drive" / "Cancel request" / "Give back control");
  - desktop metrics plus touch emulation instead of "tablet width";
  - the 401 path offers Rejoin (a reload) rather than re-joining automatically.

```findings
findings:
  - id: new
    severity: Important
    family: client-honors-wire-contract
    title: |
      batcher caps acts and text runes but not body bytes; a paste over about 8 KB is refused (413) whole
    detail: |
      drive.MaxBody is 8 KB. A probe of drive.js batcher: 9000 ASCII chars gave a 9198-byte batch and 3000 CJK chars 9078 bytes; admitBody answers 413 and the queue is dropped. Bound go() by serialized UTF-8 size (and split text by bytes), with a test that every sent body is at most MaxBody.
  - id: new
    severity: Important
    family: client-honors-wire-contract
    title: |
      a captured drag leaving the frame left or top emits negative page coords; ParseBatch 400s and the up is never delivered
    detail: |
      toPage does not clamp and the pointer is captured; coord() requires >= 0. Probe: down(50,50), move(-50,50), up(-60,50). The cast tab keeps a pressed button. Clamp in gestures' pt/at and add a trace test that leaves the frame.
  - id: new
    severity: Important
    family: plan-drift-unrecorded
    title: |
      Task 9 live test uses a stateless spyInjector, not MemController over FakeInjector, and the plan has no Revisions entry
    detail: |
      2nd in family. Rule: a Task's named files, fixtures and doubles are contract; any deviation gets a Revisions line in the milestone's closing commit. Sweep M2: the spy, the "Cancel request" label, 1024x700 instead of tablet width, Rejoin by reload. The spy skips the seg, Confine and stale rules, so a wrong seg from the shell would pass.
  - id: new
    severity: Minor
    family: shell-state-overwritten
    title: |
      offerRejoin's button is overwritten by the next render(), so the 401 Rejoin offer disappears
  - id: new
    severity: Minor
    family: shared-helper-reuse
    title: |
      drive.js copies annotate's toast (same .annotate-msg class, can overlap) and collab.poster's request options
  - id: new
    severity: Minor
    family: input-pairing
    title: |
      chord keyup is dropped when the modifier is released first; hover moves are sent while a button is held after a press on nothing
  - id: new
    severity: Minor
    family: client-honors-wire-contract
    title: |
      batcher does not pace to the server's 30/s input budget; fast key auto-repeat can 429 and drop the queue
  - id: new
    severity: Minor
    family: served-asset-coverage
    title: |
      drive.css is served but no test fetches it; the import-walk test covers only JS modules
```

---

## Re-review — 2026-10-12T07:41:32-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c80750739cf50e363172c9f28c5aa46dbbb49b9f..00f524eb0d9dbd3114de17a35ffdc585de7d81b2 |
| command | sdlc milestone-close --issue 467 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:41:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the server's wire limits (drive package) against the client's batcher.

Verifying the plan's Revisions entry, then running the tests.

Running the node and Go test suites.

Reverting fixes in the disposable checkout to confirm the regression tests go red.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what it says. `drive.js` puts the pure logic (`batcher`, `gestures`, `keyTracker`, `keyAct`, `driveView`) behind injected clocks and transport. All 94 node tests pass, and so does `go test ./cmd/internal/webcast/... ./cmd/internal/broadcast/...`. The round-1 fixes for BR-7, BR-8, BR-12 and BR-13 hold up: I reverted each one in the disposable checkout and each revert turned `drive.test.mjs` red. Two new Important findings remain, both about the sequence of events rather than message shape:

- **Wrong segment on queued acts:** the batcher stamps `seg` when it sends a batch, not when the act happened. Acts queued before a new segment arrives can therefore land on a page the driver never saw.
- **Inputs left held:** a press or key the driver started is never released when control ends or a refusal empties the queue.

I could not run `TestDriveLiveTwoViewers` here. Headless Chrome fails with `cdp: read: EOF`, and the existing `TestLiveViewerPage` fails the same way, so the cause is this machine, not the diff.

1. **Strengths**
   - `batcher` is the one gate to the wire (`drive.js:103-157`). Byte-aware `take()` plus `wireAct` and `splitText` are checked by a seeded property test (`drive.test.mjs:103`) that covers body ≤ MAX_BODY, coordinate range and pacing. Reverting any one of these turns it red.
   - `gestures` is tested with replayed traces, including malformed ones (cancel mid-drag, a late second finger, a finger that was never down, a drag that leaves the frame).
   - `keyTracker` sends an up for each down, even when Meta is released first. It is table-tested.
   - `sayer` and `poster(headers)` in `collab.js` replace two hand-copied helpers (ARCH-DRY), and annotate now uses them.
   - The live test now runs `MemController` over `FakeInjector`, the production rules. It asserts an attributed `Click` in the stream (`viewer_drive_live_test.go:169-177`), and the plan's Revisions entry records each deviation.

2. **Critical**: none.

3. **Important**
   - **`seg` is stamped when the batch is sent (`drive.js:107`, `seg: () => v.model.seg` at `drive.js:364`).**
     - The server's `ErrNotLive` exists to refuse "a batch aimed at a segment that has gone" (`injector.go:35-36`). Stamping at send time defeats it: text and key acts queued before a `seg` record arrives go out with the new segment and are typed into the new page.
     - Text and key acts carry no element id, so the stale check cannot catch them. A resume during a pause has the same effect: the batch survives the pause, and the hidden keys textarea still queues input.
     - Fix: stamp `seg` on each act in `push()`, and have `take()` end a batch where the stamp changes. Add property-test cases where the `seg` value changes between push and send.
   - **Inputs left held when control ends or the queue is dropped.**
     - `stop()` (`drive.js:154`) and a refusal (`drive.js:128`) throw away queued `up` and key-up acts.
     - `gestures`' `press` is never reset when control ends, so the next time this viewer drives it sends left-button moves with no down.
     - The server sends no release on revoke or take.

4. **Minor**
   - `lost` and "paused while driving" are handled with flags in `render()` instead of inside `driveView`. As a result, the shell's possible states can't be read off the code (ARCH-ORDER/PURE).
   - The atlas doesn't describe the wire gate's clamping, splitting and pacing.
   - When the cast's hello budget (2/s, burst 8) is exceeded, a viewer silently streams with no drive UI.
   - `ime` acts are not cut to `MAX_TEXT`.

5. **Test coverage notes**
   - `mountDrive` has no DOM harness. Rejoin, pause handling and stop-on-revoke are covered only by the live test, which this environment can't run.
   - No test changes `seg` while acts are queued.

6. **Architectural notes for upcoming work**
   - M3 (Couch) consumes the same `ctl` stream. Settle "held input on control change" on the server side now, because neither shell can do it alone.
   - Move the shell's state into `driveView`: `none / asking / driving-live / driving-paused / other / lost`.

7. **Plan revision recommendations**
   - Add a Revisions line: batches carry the segment of their acts, not the segment at send time. This replaces the claim in the `batcher` doc comment.
   - Record the release-on-control-change rule once it is implemented.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      take() bounds by serialized UTF-8 size and splitText cuts by bytes; reverting the MAX_BODY break turns the property test at drive.test.mjs:103 red.
  - id: BR-8
    disposition: addressed
    note: |
      wireAct clamps x/y into [0, MAX_COORD]; removing the clamp fails 3 tests, including the leaves-the-frame trace at drive.test.mjs:272.
  - id: BR-9
    disposition: addressed
    note: |
      Live test runs MemController over FakeInjector (spy only records) and asserts an attributed Click in the stream; Revisions entry covers the spy, labels, viewport and Rejoin. Could not execute here: Chrome CDP EOF also fails the pre-existing TestLiveViewerPage.
  - id: BR-10
    disposition: not-addressed
    note: |
      The lost flag and early return in render() look correct, but no test fails without it: mountDrive has no harness. Fold lost into driveView (pure) and table-test it.
  - id: BR-11
    disposition: addressed
    note: |
      collab.js sayer (one shared .annotate-msg element) and poster(headers) are used by both drive.js and annotate.js; the copies are gone.
  - id: BR-12
    disposition: addressed
    note: |
      keyTracker test covers the up after the modifier is released; reverting the !e.buttons hover guard fails the gestures test.
  - id: BR-13
    disposition: addressed
    note: |
      PACE_MS=40 is enforced in go(); forcing wait=0 fails the pacing assertion in the property test.
  - id: BR-14
    disposition: addressed
    note: |
      TestViewerImportsAreServed now walks the page's stylesheet links and requires drive.css to be served as text/css.
findings:
  - id: new
    severity: Important
    family: client-honors-wire-contract
    title: |
      batcher stamps seg at send time, so acts queued before a new seg record are injected into a page the driver never saw
    detail: |
      This is the 4th finding in family client-honors-wire-contract. Rule covering all of them: every field of a request is fixed at the moment of the user act it describes, and the gate only gathers and bounds, never re-derives. drive.js:107 reads deps.seg() inside take(), but the server's seg check (injector.go:35) is meant to refuse acts aimed at a segment that has gone. Text and key acts have no id, so the stale check cannot catch them. A resume after a pause does the same, because the batch survives the pause and the keys textarea still queues input. Fix: stamp seg in push(), have take() end a batch where the stamp changes, and add property-test cases where the seg value changes between push and send.
  - id: new
    severity: Important
    family: input-pairing
    title: |
      a press or key held when control ends, or when a refusal drops the queue, is never released in the cast tab
    detail: |
      This is the 2nd finding in family input-pairing. Rule: every down the driver starts is ended on every exit path (refusal, revoke, take, seg change, seat loss), and that duty belongs to the side that outlives the exit, the server. batcher.stop() and the refusal path drop queued up and key-up acts. gestures' press is never reset, so the next drive sends left-button moves with no down. Nothing server-side releases held buttons or keys on a control change. Fix: on a control change, have TabInjector/WithTab dispatch mouseReleased and key-up for whatever the previous driver left down; add a reset() to gestures and keyTracker and call it from render() when the batch stops.
  - id: new
    severity: Minor
    family: state-scope-unmodelled
    title: |
      shell states (lost, driving-but-paused) live as flags in render() rather than in driveView
    detail: |
      This is the 2nd finding in family state-scope-unmodelled. driveView should return the whole state set (none, asking, driving-live, driving-paused, other, lost) so render() only paints it, and every state can be table-tested without a DOM.
```

---

## Re-review — 2026-10-12T07:50:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c80750739cf50e363172c9f28c5aa46dbbb49b9f..f9f608000ffebbb5c3cab27803f17ce69a03124c |
| command | sdlc milestone-close --issue 467 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:50:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on the last commit (round 2 fixes) for disposition.

Tests pass; checking key-repeat handling and the release ordering before writing up.

The review is done; writing up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The round-2 commit (f9f608000) fixes all four open findings, and both behavior fixes come with regression tests that fail without them. I checked this by reverting each fix in the disposable checkout:
- With the `Releaser` call turned off in `controller_mem.go`, `TestControllerLetsGoOnRevoke` fails ("the press outlived the revoke").
- With the per-segment cut removed from `take()`, the batcher's "never mixes pages" test fails.

All Go tests pass (`go test ./cmd/internal/webcast/...`), and all 96 node tests pass (`node --test tests/webcast/*.test.mjs`). One check does fail: `go vet` reports unreachable code at `drive/tab.go:164`. Nothing here blocks: there is one Important docs gap (the atlas) and three Minors. One Minor matters for design: `Release` is not tied to the driver whose hold it ends.

**1. Strengths**
- `drive/tab.go:35-120`: The server now owns letting go of held input, as the BR-16 rule asked. `TabInjector` records what is held per driver and segment. It releases on four paths: a different driver, a different page, a refused batch, and a control change. Every exit path the client can't cover is covered on the side that outlives it.
- `controller_mem.go:162-170`: The release runs outside the lock with a time limit, so a revoke never waits on a stuck tab. The extent is bounded at 10 s. `cast_drive_test.go` still proves a revoke goes through while a batch is stuck.
- `drive.js:139-152`: The segment is stamped in `push()`, and `take()` ends a batch where the segment changes. Moves and wheels don't coalesce across pages. This is the "fix fields at act time; the gate only gathers" rule from BR-15, applied as a rule rather than one patched site.
- `driveView` (`drive.js:304-322`) now returns the whole shell state set, each with its action and capture flag. `render()` only paints it, and the table test covers all eight rows.
- `release_test.go` checks the order of releases (button first, then keys last-pressed first), where the up lands, and that nothing is released twice. Real logic is under test against the stateful `FakeInjector`.

**2. Critical findings:** none.

**3. Important findings**
- **The atlas doesn't describe the new let-go behavior** (`atlas/webcast.md:197-215`). It doesn't mention that the cast tab releases what a driver held, the new `drive.Releaser` that `MemController` calls on every control change, or that acts carry their segment. Line 211 still describes the batcher only as "drops its queue at a refusal". Fix: add a short **Let go** bullet listing the four release paths, and say that a batch holds acts from one segment only.

**4. Minor findings**
- **`Release` doesn't check whose hold it ends** (`tab.go:70`, `controller_mem.go:162`). This is ARCH-ORDER, and the 3rd finding in family `input-pairing`.
  - The release runs on a goroutine that was started at control change. It then ends whatever `held` contains when it reaches the tab.
  - If the next driver's first batch reaches the tab before the queued release, the release drops that driver's button. FIFO `ask` ordering and human-speed grants make this very unlikely, but nothing prevents it.
  - The rule that covers the family: every release names the hold it ends (driver and segment) and does nothing when the current hold belongs to someone else.
  - Fix: pass the outgoing driver in `Release(ctx, by)` and skip when `held.by != by`. To test it, revoke Ana, grant Bo, let Bo press down, then call `Release(ctx, "v1")`. Bo's button must stay down.
- **Duplicate `return nil`** at `tab.go:163-164`; `go vet` flags the unreachable code.
- **Held keys grow with auto-repeat** (`tab.go:107`). `keyTracker.down` doesn't filter `e.repeat`, so every repeated keydown is appended to `held.keys`, and a release then sends N keyups for one key. Fix: in `note`, skip a key that is already held, or filter repeats in the client.

**5. Test coverage notes**
- BR-15 is covered by `drive.test.mjs` ("an act carries the page…"), which goes red when reverted.
- BR-16 on the server is covered by `TestTabInjectorLetsGo`, which tests four paths, and by `TestControllerLetsGoOnRevoke`, which goes red when reverted. On the client, `gestures.reset` has its own test.
- Not covered: the order between the release goroutine and the next driver's batch (see the Minor above). The fake serializes `WithTab`, so a test can force that order by calling `Release` after Bo's batch.
- The `lost` and `paused` rendering is tested only through the pure `driveView`. That's acceptable, since `render()` now only paints.

**6. Architectural notes**
- **ARCH-DRY: pass.** Toast and request options come from collab.js, as fixed in BR-11.
- **ARCH-PURE: pass.** `batcher`, `gestures`, `keyTracker` and `driveView` are pure and injected. The held-input bookkeeping in `TabInjector` is plain logic over the `Tab` seam.
- **ARCH-PURPOSE: pass.** The fixes apply the stated family rules rather than patching single sites.
- **ARCH-MOCK: pass.** `FakeInjector` is stateful, now records what it dispatched, and sits behind the same `TabRunner` seam the recorder uses.
- **ARCH-CONSTRAINTS: pass.** Release is bounded at 10 s, and the batcher keeps its pacing.
- **ARCH-SECURE: pass.** No new untrusted input; the segment still comes from the client and is checked by the server.
- **ARCH-ORDER: flag (Minor).** The release has no identity guard (see above). The held state is a small struct whose changes all go through `note` and `release`, which is acceptable.
- **ARCH-FUNERAL: pass.** Nothing durable is created; the release goroutine ends within its 10 s limit.

**7. Plan revision recommendations:** none needed. The plan's Revisions entry for 2026-10-12 round 2 matches the code.

```findings
dispose:
  - id: BR-10
    disposition: addressed
    note: |
      lost is now a driveView state (action rejoin) painted by every render(); table-tested in drive.test.mjs.
  - id: BR-15
    disposition: addressed
    note: |
      seg stamped in push(), take() cuts at seg change, no coalescing across segs; batcher test goes red with the cut reverted.
  - id: BR-16
    disposition: addressed
    note: |
      TabInjector tracks held button/keys, releases on other driver, other seg, refusal and control change (Releaser); gestures.reset and tracked.clear on end/refusal; TestControllerLetsGoOnRevoke goes red with Release disabled.
  - id: BR-17
    disposition: addressed
    note: |
      driveView returns none/asking/driving/paused/other/lost with action and capture; render only paints; table test covers all rows.
findings:
  - id: new
    severity: Important
    family: atlas-tracks-surface
    title: |
      atlas/webcast.md does not describe the cast tab's let-go (Releaser on control change, refusal, driver or page change) or per-act segments
    detail: |
      Remote driving section lines 197-215 still says only that the batcher drops its queue at a refusal; add a Let go bullet and the one-segment-per-batch rule.
  - id: new
    severity: Minor
    family: input-pairing
    title: |
      Release ends whatever is held when it reaches the tab, not the outgoing driver's hold
    detail: |
      3rd in family input-pairing. Rule: every release names the hold it ends (driver, seg) and is a no-op otherwise. The async release from changeLocked (controller_mem.go:162) could otherwise drop the next driver's press if that driver's batch runs first; pass by into Release and test revoke, grant, press, then late Release.
  - id: new
    severity: Minor
    family: dead-code
    title: |
      duplicate return nil at drive/tab.go:164; go vet reports unreachable code
  - id: new
    severity: Minor
    family: residue-bound-on-writer
    title: |
      held.keys appends every auto-repeat keydown, so a release sends N keyups for one key
    detail: |
      2nd in family. Rule: a server-held collection fed per input act is keyed by identity at its writer. Fix by deduping by key in TabInjector.note (tab.go:107) or filtering e.repeat in keyTracker.down.
```

---

## Re-review — 2026-10-12T07:54:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c80750739cf50e363172c9f28c5aa46dbbb49b9f..515639975d462d080c71d73aed630188d099ff34 |
| command | sdlc milestone-close --issue 467 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:54:25-07:00 |
| verdict | SHIP |

## Review

Both round-3 fixes go red when I revert them. Next I'm running the JS tests and checking the plan's revisions, then I'll write the verdict.

```verdict
verdict: SHIP
confidence: high
```

All four open findings are fixed, and the two behaviour fixes each have a regression test that fails without them. I ran the stat and name-status recipes and looked at the round-3 patch in full. In this checkout, `go vet ./cmd/internal/webcast/...` is clean and `go test ./cmd/internal/webcast/...` passes. The two JS suites `tests/webcast/drive.test.mjs` and `inject.test.mjs` pass 24 of 24. To check the fixes, I reverted the `by` check in `TabInjector.Release` and the test failed at `release_test.go:112` with "Ana's late release ended Bo's hold: [up keyup End]". I reverted the key dedup in `note` and it failed at `release_test.go:116` with "[up keyup End keyup End]". The checkout was restored to the pinned head afterwards. I found nothing new at or above Important.

1. **Strengths**
   - `drive/tab.go:75-85`: `Release(ctx, by)` checks who holds the press inside the same `WithTab` run that sends the ups. The ownership check and the release therefore happen in stream order, so a release can't end a press it doesn't own.
   - `drive/controller_mem.go:162-171`: there is no release when nobody was driving (`f.control.Driving() != ""`). The outgoing driver is captured before `f.control` changes, and the goroutine stops after `releaseWait`.
   - `drive/tab.go:122`: the writer dedups held keys by key, which follows the family rule ("keyed by identity at its writer"). It doesn't rely on the client filtering `e.repeat`.
   - The new atlas text (`atlas/webcast.md:211-228`, `atlas/webcast-format.md:601-603`) lists all four let-go cases and the one-segment-per-batch rule.

2. **Critical**: none.

3. **Important**: none.

4. **Minor**
   - A release is matched by driver only, not driver plus segment as BR-19 suggested. If the same viewer is revoked, granted again and presses before the earlier release reaches the tab, that release would end the new press. That needs a request and a grant inside the release's 10 s window while the tab is stalled, which is unlikely, so I'm not raising it. A per-hold sequence number would close it if this family comes back.
   - The plan has no round-3 Revisions line for the `Releaser` signature change. I'm not raising it either, because the plan names `drive.Releaser` without a signature, so nothing in it is contradicted.

5. **Test coverage**: `TestTabInjectorLetsGo` now covers revoke, double release, another driver's first batch, a late release from a previous driver, an auto-repeated key, a refused batch and a page change. `TestControllerLetsGoOnRevoke` covers the controller path end to end. The late-release case puts the events in order by hand, so it doesn't depend on goroutine timing.

6. **Architecture**
   - ARCH-DRY: pass. One `release` and one `note` serve every let-go path.
   - ARCH-PURE: pass. The let-go rules sit behind the `Tab` seam and are tested through the fake.
   - ARCH-PURPOSE: pass. The fix covers the whole input-pairing family as one rule (every release names its hold), not just the instance BR-19 named.
   - ARCH-MOCK: pass. The fake and the real recorder share `TabInjector`.
   - ARCH-CONSTRAINTS: pass. Each release has a time limit.
   - ARCH-SECURE: pass. The `by` check stops one viewer's release from ending another viewer's input.
   - ARCH-ORDER: pass. The ordering that BR-19 named is now settled under the tab's serialization and tested. The same-viewer case is left as the Minor above.
   - ARCH-FUNERAL: pass. Nothing durable is created, and the release goroutine is time-limited.

7. **Plan revisions**: none needed.

```findings
dispose:
  - id: BR-18
    disposition: addressed
    note: |
      atlas/webcast.md:211-228 adds the Batches rules (one segment per batch) and a Letting go section listing four cases with Releaser naming the outgoing driver; webcast-format.md:601-603 matches tab.go.
  - id: BR-19
    disposition: addressed
    note: |
      Release(ctx, by) is a no-op unless held.by==by (tab.go:75-85), and controller_mem.go:165 passes the outgoing driver. With the check reverted, release_test.go:112 fails.
  - id: BR-20
    disposition: addressed
    note: |
      The duplicate return is gone (tab.go:176), and go vet ./cmd/internal/webcast/... is clean.
  - id: BR-21
    disposition: addressed
    note: |
      note() dedups held keys by Key at the writer (tab.go:122). With the dedup reverted, release_test.go:116 fails with two keyups.
```
