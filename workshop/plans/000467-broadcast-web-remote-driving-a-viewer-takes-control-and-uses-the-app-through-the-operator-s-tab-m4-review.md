# Boundary Review — pair#467 (milestone M4)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 1cce37e495ba359b76284ca128ee9d5bc580903a..787cf7ad92395e7f5f91260bf6a7038d1ae7146a |
| command | sdlc milestone-close --issue 467 --milestone M4 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:42:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now the code diff (non-fixture, non-docs).

Checking the mirror's `by` validation and the new tests next.

Running the test suites now.

Live Chrome can't start in this sandbox (GPU process crash), which is an environment problem. Checking the mirror's `by` refusal path now.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M4 does what it set out to do. The hand-written `driven` fixture is gone and `rec-driven`, a session recorded through `MemController` → `liveGate` → `TabInjector` → `Recorder.WithTab`, replaces it. All four planned consumers now derive from the fixture's own records, and the coverage test names `rec-driven`. The recording also exposed a real gap: a driver's click that follows a link was lost, because the take cannot reach a document that has gone. The fix is now pinned in the data: a click by `v1` at 4114 lands before the navigate segment at 4127. `go test ./cmd/internal/webcast/...` and `node --test tests/webcast/*.test.mjs` (96/96) are green. I could not run the `WEBCAST_LIVE_CHROME=1` suites here: Chrome's GPU process fails to start in this sandbox (`GPU process isn't usable`), which is an environment problem, not a code one.

The fix does not block the gate, but it changes the attribution protocol in two ways that should be settled first:
- A driven report whose `by` loses a race with a revoke is now **dropped entirely**, so a real click disappears from the recording. The plan's own ARCH-ORDER rationale rejected exactly this outcome.
- The recorder's fake-CDP double still models the old protocol, where the take returns items. It no longer matches `observer.js`.

### 1. Strengths
- **Consumers derive from the data.** `query/events_test.go:115`, `replay_live_test.go:179`, `replay_server_test.go:159` and `tests/webcast/replay.test.mjs:108` compute spans, acts, gaps and duration from `rec-driven`'s records. Each has a non-vacuity guard (`want.length >= 2`, all four attribution flags).
- **The scenario uses the production stack.** `live_record_driven_test.go:56` drives `MemController(liveGate{c}, NewTabInjector(c))` with a real grace period, so `leave` really comes from a detach plus the 1 s grace. The fixture confirms it: request at 11546, leave at 12948.
- **Structural quoting.** `recorder/drive.go:104` passes the argument through `json.Marshal` into the evaluated expression rather than concatenating it raw.
- **`by` trust holds.** The binding exists only in the `couchcast` isolated world (`chrome.go:442`, `executionContextName`), so page script can't forge a `by`. The reducer also refuses any `by` that isn't the driver (pinned by `drive_test.go:255`).
- **Atlas and plan revisions** record the rename, the hold/take change and its trade (`atlas/webcast-format.md:600-609`, plan `## Revisions` 2026-10-12 M4).

### 2. Critical findings
None.

### 3. Important findings

**I-1: a report whose `by` loses a race with a revoke is dropped instead of recorded without attribution** (ARCH-ORDER)
- **Where:** `recorder/recorder.go:1092`, `recorder/mirror.go:155`
- **What happens:** driven reports now reach the loop as later `bindingCalled` events in one shared arrival-ordered queue. A revoke that the controller enqueues during a batch can land before them; the controller drops its lock before `Inject`, so this is the normal path. `mirror.emit` then refuses the whole record. A click the operator's page really received is erased from the recording and only counted under `Dropped`.
- **Why it matters:** the plan's "Why hold/take" paragraph names this exact result ("drops a real click") as the reason for the design. The plan's ARCH note promised a test showing taken items still validate across a revoke, and that test now only holds against the fake.
- **Fix:**
  - In `binding()`/`observeItems`, when `msg.By` is not the mirror's current driver, record the items with `by=""`. That puts them in the residual class the atlas already describes (an expired hold's items) instead of losing them.
  - Add a fake-CDP test: Emit a revoke, then send a tagged binding click, and assert the click is written unattributed.
  - Update the atlas "Take" bullet and the plan revision.
  - If the drop is deliberate, the plan's ARCH note on revoke ordering needs a revision entry instead, plus a test asserting the drop.

**I-2: the recorder's fake still models the old observer protocol** (ARCH-MOCK; family `live-conformance-unasserted`, 2nd)
- **Where:** `recorder/drive_test.go:39-45`; the dead path is `recorder/drive.go:52`.
- **What happens:** `driveTab.more` answers `__couchcastTake` with the `taken` items. The real `observer.js:231` now always returns `'[]'` and sends through the binding. So `TestWithTabAttributesExactly` exercises an ordering production never produces, and the Go branch that observes take-returned items with `by` is dead in production.
- **Rule covering the family:** a fake's answers for an in-page or CDP protocol must change in the same commit as the real script. One way to enforce this is a live conformance test that asserts the shape of hold and take returns in real Chrome (take returns `[]`; held items arrive as tagged `bindingCalled`).
- **Fix:**
  - Change `driveTab` so the take returns `[]` and the driver's items arrive as `bindingCalled` with `by` after the dispatch.
  - Either delete the take-items branch in `withTab`, or keep it explicitly as a backward-compat path for an older observer and say so.
  - Add the live assertion next to `TestLiveDriveNavigation`.

### 4. Minor findings
- `recording/internal/fixturegen/main.go:5` still documents the removed `-driven` flag. This is the 2nd finding in family `dead-code`. Rule: removing a flag also removes its usage and doc lines; grep the flag name before committing.
- `recording/testdata/fixtures/README.md:14` says "21 s, 9 frames"; `frames/index.jsonl` has 8 entries.
- Plan Task 11 says "v1 posts a note through the board". As built, the note is a fixturegen spec note (`notes.go:47`). The M4 revision doesn't record this. This is the 4th finding in family `plan-drift-unrecorded`. Rule: at milestone close, walk each checklist line of the milestone and add every mechanism change to the "as built" revision. Doing this as one step, rather than fixing instances one at a time, covers all four findings.
- The coverage check "the operator acting while a viewer drives" (`fixture_test.go:105`) only checks `kinds[click] > by[click]`, not that the click fell inside a drive span. The query test covers this properly, so it is low risk.
- The span-derivation loop is written three times: in Go replay, in JS replay and in the query test. That is acceptable for tests (ARCH-DRY note only).

### 5. Test coverage notes
- `TestHeldDocumentSendsAttributedItems` pins the Go side of tagged binding reports: accepted for the driver, refused for `v9`.
- The `observer.js` change (tagging during the hold; the take sending through the binding) is covered only by the env-gated live tests (`TestLiveDriveNavigation` and the `driven` scenario) and by the committed fixture. I couldn't run those here.
- No test covers a revoke arriving between a batch and its tagged reports (see I-1).

### 6. Architectural notes
- **ARCH-DRY:** pass (minor test duplication noted above).
- **ARCH-PURE:** pass. The changes are in the IO shell and the observer, and the reducer stays the oracle.
- **ARCH-PURPOSE:** pass. Every listed consumer derives from `rec-driven`.
- **ARCH-MOCK:** flag (I-2).
- **ARCH-CONSTRAINTS:** pass. Batch bounds are unchanged, and tagged sends add no fan-out.
- **ARCH-SECURE:** pass. The binding is isolated-world only, `by` is checked by the reducer, and the argument is JSON-quoted.
- **ARCH-ORDER:** flag (I-1). Attribution moved from synchronous (inside the queue input) to arrival-ordered, and the uncertain case is collapsed into absence.
- **ARCH-FUNERAL:** pass. The fixture was replaced rather than added (8 small PNGs) and the budget test is green.

### 7. Plan revision recommendations
- Under `## Revisions`, M4 as built, add:
  - "The driver's note in `rec-driven` comes from fixturegen's spec, not the board."
  - "The ARCH-notes test 'ctl revoke between hold and take, taken items still validate' no longer describes production. Reports arrive after the queue input; a revoke ordered before them [drops them | records them unattributed, per I-1]."
- In the Integration points `Recorder.WithTab` bullet, step (3) take still says it "returns what the observer queued". Revise it: the take flushes through the binding and returns nothing.

```findings
findings:
  - id: new
    severity: Important
    family: uncertain-attribution-degrades
    title: |
      A driven report whose by loses the race to a revoke is dropped whole, erasing a real click
    detail: |
      recorder.go:1092 passes msg.By into observeItems; mirror.emit refuses and drops the record when by is no longer the driver. Reports now arrive as later bindingCalled events, so a revoke enqueued mid-batch (controller drops its lock before Inject) erases acts the page really received, the outcome the plan's hold/take rationale rejected. Record them with by="" (the expired-hold residual class), add a fake-CDP test (Emit revoke, then a tagged binding click, assert it is written unattributed), and update the atlas Take bullet and plan.
  - id: new
    severity: Important
    family: live-conformance-unasserted
    title: |
      The recorder's fake driveTab still returns items from __couchcastTake; the real observer.js returns '[]'
    detail: |
      2nd in family. drive_test.go:39-45 models the old protocol, so TestWithTabAttributesExactly tests an ordering production never produces, and drive.go:52's take-items branch is dead in production. Rule: a fake's protocol answers change in the same commit as the real script, enforced by a live check of hold/take return shapes. Re-model driveTab (take returns [], driver items arrive as tagged bindingCalled), and delete the dead branch or label it as backward compatibility.
  - id: new
    severity: Minor
    family: dead-code
    title: |
      fixturegen main.go:5 still documents the removed -driven flag
    detail: |
      2nd in family. Rule: removing a flag also removes its usage and doc lines; grep the flag name before committing.
  - id: new
    severity: Minor
    family: plan-drift-unrecorded
    title: |
      Driver's note comes from the fixturegen spec, not the board as Task 11 states; the M4 revision omits it, and WithTab step 3 still describes take returning items
    detail: |
      4th in family. Rule: at milestone close, walk each checklist line and Integration-point bullet of the milestone and record every mechanism change in the as-built revision, as one step.
  - id: new
    severity: Minor
    family: fixture-doc-matches-data
    title: |
      fixtures README says rec-driven has 9 frames; frames/index.jsonl has 8
```

---

## Re-review — 2026-10-12T09:46:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 1cce37e495ba359b76284ca128ee9d5bc580903a..f95c0c3302d253f8eae9f45f0ad1b546d23aac19 |
| command | sdlc milestone-close --issue 467 --milestone M4 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:46:21-07:00 |
| verdict | SHIP |

## Review

Prior-round fixes check out in the code and all webcast tests pass. Next I'm reverting the BR-30 fix to confirm its regression test fails without it.

```verdict
verdict: SHIP
confidence: high
```

All five findings from round 1 are fixed in the pinned head. BR-30 has a real regression test: I turned the fix off in `observeItems` and `TestHeldDocumentSendsAttributedItems` failed (`1 clicks, by "v1"`). With the fix restored it passes, and the checkout is clean. The fake tab in BR-31 now follows the same protocol as `observer.js`: the take returns nothing and sends the driver's items as a tagged `bindingCalled`. The dead take-items branch in `drive.go` is gone. The doc and plan corrections (BR-32/33/34) match the code and data they describe. `go test ./cmd/internal/webcast/...` passes in full. I found one new issue, a Minor stale comment. Nothing blocks the M4 close.

1. **Strengths**
   - `recorder.go:1099-1104`: when a tagged report arrives after control has moved, the recorder keeps the act and drops only the attribution. It no longer refuses the whole record, so the page's real input still reaches the stream.
   - `observer.js`: while held, the observer sends what it collects at once with the tag. A click that follows a link is therefore recorded before its document unloads, and every driven item reaches the recorder through the binding in order.
   - `drive_test.go:16-58`: the fake `driveTab` sends tagged binding events from the take. `TestWithTabAttributesExactly` now tests an ordering the production code can actually produce.
   - The hand-written `driven` fixture is replaced by `rec-driven`, a recorded session. The consumers (query goldens, replay tests) read its own records.
   - ARCH-SECURE: the binding lives in the isolated `couchcast` world (`recorder.go:1168-1172`), so page scripts can't fake a `by`.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `recorder.go:1084-1086`: the comment on `msg.By` still says "The reducer refuses a by that isn't the driver." Since BR-30, `observeItems` removes such a `by` before the reducer sees it.

5. **Test coverage notes**
   - Reverting the BR-30 fix makes its regression test fail, as it should.
   - The take-returns-`[]` protocol is checked against real Chrome only by the end-to-end live tests (`TestLiveDrive`, `TestLiveDriveNavigation`, the recorded fixture). There is no direct check of the return shape. That's acceptable for now.
   - The fake sends all of a dispatch's items at take time, which is one valid interleaving. The real observer's immediate send during a hold is covered by `TestHeldDocumentSendsAttributedItems`.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Hold and take share `observerCall`.
   - **ARCH-PURE: pass.** Attribution is decided in `observeItems` from reducer state.
   - **ARCH-PURPOSE: pass.** The recorded fixture now drives every consumer.
   - **ARCH-MOCK: pass.** The fake follows the real script's protocol, and live Chrome tests cover the real script.
   - **ARCH-CONSTRAINTS: pass.** `HOLD_MAX` bounds how long a hold can last.
   - **ARCH-SECURE: pass.** The binding is in an isolated world, and a `by` from a stale driver is downgraded, not trusted.
   - **ARCH-ORDER: pass.** Driven reports now come only as queue-ordered binding events, and the late-report-after-revoke ordering is tested.
   - **ARCH-FUNERAL: pass.** The fixture files replace the old ones and no new growing artifact is added.

7. **Plan revision recommendations:** none. The M4 round 1 revision already records BR-30, BR-31, Task 4 step 3 and Task 11 as built.

```findings
dispose:
  - id: BR-30
    disposition: addressed
    note: |
      recorder.go:1099 downgrades by to "" when it is no longer the driver; TestHeldDocumentSendsAttributedItems goes red with the guard disabled (verified), green with it.
  - id: BR-31
    disposition: addressed
    note: |
      driveTab take now returns nothing and sends a tagged bindingCalled (drive_test.go:44-58); the take-items branch in drive.go is removed; live suite covers real shapes per plan revision.
  - id: BR-32
    disposition: addressed
    note: |
      fixturegen main.go usage line and flag removed; grep finds no remaining -driven reference.
  - id: BR-33
    disposition: addressed
    note: |
      Plan revision "M4 review round 1" records Task 4 step 3 (take sends tagged via binding) and Task 11 (note from fixturegen spec, author v1).
  - id: BR-34
    disposition: addressed
    note: |
      README now says 8 frames, matching frames/index.jsonl (8 lines).
findings:
  - id: new
    severity: Minor
    family: plan-drift-unrecorded
    title: |
      recorder.go:1084-1086 comment says the reducer refuses a non-driver by; observeItems now downgrades it first
    detail: |
      Stale after BR-30. Rule (5th in family): a change in behavior updates every comment or doc that describes that behavior. Grep the old wording before committing.
```
