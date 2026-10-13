# Boundary Review — pair#505 (whole-issue close)

| field | value |
|-------|-------|
| issue | 505 — broadcast-web: remote driving fails live — every drive input returns 500 'Something went wrong' (unmapped injector error, nothing logged) |
| repo | pair |
| issue file | workshop/issues/000505-broadcast-web-remote-driving-fails-live-every-drive-input-returns-500-something-went-wrong-unmapped-injector-error-nothing-logged.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5a533d455b12f31e879dc526e6ac832d779c3be3..965e02fba53408d28a2bbd1e1fd7607d785576e6 |
| command | sdlc close --issue 505 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T17:30:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking how CDP reply errors reach the hit test, then I'll run the tests.

Tests pass. Now checking gofmt at base and reverting the fixes to confirm the regression tests go red.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two root causes are fixed correctly and each has a regression test that fails without the fix. I checked this in this checkout. With the old scroll-subtracting hit test, `TestCdpTabHitTestsPageCoordinates` fails with "hit-tested at [40 120], want (40, 420)". With the old `e.detail` click count, the node test fails 1 of 16. With `logOnce` stubbed out, `TestDriveRoutesLogWhatTheViewerCantFix` fails with "operator's log []". In the unmodified checkout, `go test ./cmd/internal/webcast/...` and `node --test tests/webcast/drive.test.mjs` both pass.

Nothing blocks shipping. Three of the new error mappings have no test, though, and the Done-when says "tests pin both", so those gaps should be closed cheaply. The unmapped 500 the operator saw live was not reproduced; the issue says so, and the third Done-when item allows for it ("or the logged cause names what's left").

## 1. Strengths
- **Hit-test fix rests on measurement.** `recorder/drive.go:161-186` uses page px for `DOM.getNodeForLocation`. The Log records page vs. viewport probes at scroll 236. "No node found" is now matched as a typed `*cdp.Error` prefix, and other failures become `ErrInput` rather than "stale". `cdp.decodeReply` returns `*cdp.Error` unwrapped, so `errors.As` reaches it.
- **Error sentinels stay out of the request handler.** `drive.Answer` stays a pure mapping, and `ErrInput` keeps the cause in its text (`%v`) while showing the viewer a fixed sentence. Nothing from the request is echoed back.
- **The operator log is bounded.** `server_drive.go` `logOnce` keeps at most 64 entries in memory, then resets, as the plan states.
- **Double-click counting has no IO.** `gestures` counts double clicks from time and place, a drag clears `lastClick`, and the node test covers the 1, 2, 1 sequence plus the too-slow and moved cases.
- **There is a real-Chrome test for the whole chain.** `TestDriveRealCast` is gated by an env var, runs through the production path, and wheels again after the page has scrolled, which is what reproduces the operator's report.

## 2. Critical
None.

## 3. Important
- **Three new error mappings have no test** (`recorder/drive.go:36-38`, `recorder/drive.go:189-195`, `cast_drive.go:27`):
  - `Recorder.WithTab` turning `errClosed` into `drive.ErrClosed` (410).
  - `cdpTab.Dispatch` wrapping CDP input failures as `ErrInput` while passing `ErrDialog` through unchanged.
  - `Chrome.WithTab` with no recorder returning `drive.ErrNotLive`.
  
  The route test injects a ready-made `ErrInput` through `failingInjector`, so it never runs the code that produces one. Reverting any of these three stays green.
  
  **Fix:** in `recorder/drive_test.go`, make `driveTab` fail `Input.dispatchMouseEvent` and assert `ErrInput`. Make it open a dialog and assert `ErrDialog`, not `ErrInput`. Close the recorder and assert `ErrClosed`. Add a nil-`rec` case for `Chrome.WithTab`.

## 4. Minor
- gofmt reports `cmd/internal/webcast/live_drive_real_test.go` (the `errSpy` fields) and `cmd/internal/webcast/recorder/drive_test.go` (the new `driveTab` field comments) as unformatted. The base version of `drive_test.go` was clean. I found no CI gofmt check.
- `gestures.reset()` (`drive.js:263`) doesn't clear `lastClick`. If control ends and is granted again within 500 ms, a click at the same spot counts as a double click. This is a small state leak (ARCH-ORDER).
- A viewer disconnecting mid-request (`context.Canceled`) is logged to the operator as "503". The once-per-message rule limits it, but it is noise rather than a fault.

## 5. Test coverage notes
- Reverting the fixes confirmed the regression tests for the hit-test point, the double click and the operator log (see the summary).
- The "No node found" → stale vs. other failure → `ErrInput` table in `TestCdpTabHitTestsPageCoordinates` tests real logic.
- `TestDriveRealCast` doesn't run by default. Its windowed run is recorded in the Log, but the review can't re-check it.

## 6. Architecture
- **ARCH-DRY: pass.** No duplication. `answerDrive` became the method `d.answer`, and every call site was moved over.
- **ARCH-PURE: pass.** `Answer` and `gestures` are pure. Logging lives only in the route shell.
- **ARCH-PURPOSE: pass.** Every 5xx goes through `d.answer` and gets logged. The one unmapped error the plan named (`ErrNotRecording` from `Chrome.WithTab`) is now `ErrNotLive`. The live 500 was not reproduced, and the Log says so honestly.
- **ARCH-MOCK: pass.** The `driveTab` fake gained `hitErr` and `hitAt`. `TestDriveRealCast` serves as the live conformance check.
- **ARCH-CONSTRAINTS: pass.** `DOUBLE_MS` 500 matches the platform default, and log volume is bounded.
- **ARCH-SECURE: pass.** Logged causes are CDP or Go error text and contain no request content. The viewer gets fixed sentences only.
- **ARCH-ORDER: pass, with the minor `reset()` / `lastClick` note above.**
- **ARCH-FUNERAL: pass.** The `logged` map is capped at 64 and reset after that. Nothing durable is created.

## 7. Plan revisions
None. The plan matches the code. The Tests item could also list the three mapping tests above once they are added.

```findings
findings:
  - id: new
    severity: Important
    family: behavior-change-without-regression-test
    title: |
      Recorder errClosed→ErrClosed, cdpTab.Dispatch→ErrInput (ErrDialog pass-through), Chrome.WithTab nil→ErrNotLive have no test
    detail: |
      Reverting any of recorder/drive.go:36-38, recorder/drive.go:189-195 or cast_drive.go:27 leaves the suite green; the route test injects a pre-built ErrInput via failingInjector. Add driveTab cases: Input.* failure → ErrInput, dialog → ErrDialog, closed recorder → ErrClosed; and a nil-rec Chrome.WithTab → ErrNotLive.
  - id: new
    severity: Minor
    family: gofmt-clean
    title: |
      live_drive_real_test.go and recorder/drive_test.go are not gofmt-clean
    detail: |
      gofmt -l lists both; the errSpy fields and the new driveTab field comments are misaligned. Base drive_test.go was clean.
  - id: new
    severity: Minor
    family: state-reset-completeness
    title: |
      gestures.reset() does not clear lastClick (ARCH-ORDER)
    detail: |
      A click before control ends and one after it is granted again, within DOUBLE_MS and DRAG_PX, is sent as a double click; reset() should also set lastClick = null.
```

---

## Re-review — 2026-10-12T17:43:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 505 — broadcast-web: remote driving fails live — every drive input returns 500 'Something went wrong' (unmapped injector error, nothing logged) |
| repo | pair |
| issue file | workshop/issues/000505-broadcast-web-remote-driving-fails-live-every-drive-input-returns-500-something-went-wrong-unmapped-injector-error-nothing-logged.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5a533d455b12f31e879dc526e6ac832d779c3be3..9ea549903dfc09362cffd0168ba595e27cf9b832 |
| command | sdlc close --issue 505 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T17:43:03-07:00 |
| verdict | SHIP |

## Review

Tests pass and gofmt is clean on the touched files. Next I'm mutation-checking the claimed fixes.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** All three findings from the earlier round are fixed, and I checked each fix by removing it again and running its test. The code matches the issue's Spec and Plan:
- **Hit test:** the hit test now uses page coordinates, with "No node found" read as stale and any other failure as `ErrInput`.
- **Double click:** drive.js counts a double click from time and place, because Chrome's pointer events carry `detail` 0.
- **Error mapping:** every 5xx answer has its own sentence and is logged for the operator once per distinct message, in a map that holds at most 64 entries.

`go test ./cmd/internal/webcast/...` and `node --test tests/webcast/drive.test.mjs` both pass at the head commit. The atlas covers the new answers, the operator log, page-coordinate hit tests and the double-click rule. Nothing blocks.

**1. Strengths**
- `recorder/drive.go:210-224`: `At` sorts errors into stale, dialog and `ErrInput` with a typed `errors.As(*cdp.Error)` check on the CDP message prefix. A real failure is no longer reported as "the page changed".
- `server_drive.go:385-400`: `logOnce` keeps a bounded set, forgets it when full (ARCH-FUNERAL), and calls `logf` outside the lock.
- `drive.go:590-593`, `drive.go:607`: a double click resets after its second click, and a drag (`:581`) or `reset()` clears `lastClick`. The node test covers a slow pair, a pair that moved, a third click and a reset.
- `drive_test.go:307-339`: the fake tab records where each hit test landed (`hitAt`), so the test checks the coordinates directly, not just a downstream result.
- `answer_test.go`: the new cases cover wrapped errors (`fmt.Errorf("ask: %w", …)`).

**2. Critical:** none.

**3. Important:** none.

**4. Minor:** none new. `gofmt -l` still lists `query/events.go` and `server.go`, but this window doesn't touch either file.

**5. Test coverage notes.** I removed each fix in turn and ran its test; all five went red, and the tree was restored afterwards:
- the `errClosed` → `ErrClosed` mapping → `TestWithTabFailuresAreDriveErrors` FAIL
- `Dispatch` returning the raw error → FAIL
- `Dispatch` wrapping `ErrDialog` too → `TestWithTabGivesUpOnADialog` FAIL
- `Chrome.WithTab` returning `ErrNotRecording` again → `TestUnrecordedChromeWithTabIsNotLive` FAIL
- `reset()` without `lastClick = null` → the node gestures test FAIL

`TestDriveRealCast` only runs when `WEBCAST_LIVE_CHROME` is set, so I didn't run it here. The issue log records it reproducing the bug before the fix and passing one approved windowed run after it.

**6. Architecture**
- **ARCH-DRY: pass.** One `Answer` table, and one `d.answer` replaces `answerDrive` at every call site.
- **ARCH-PURE: pass.** `Answer` is pure. Logging sits in the thin route layer. `gestures` stays pure, with injected deps and timestamps.
- **ARCH-PURPOSE: pass.** I checked every source of injector errors (`At`, `Dispatch`, `ask` closed, nil recorder, context): each now maps to a sentence or is logged. Nothing is deferred.
- **ARCH-MOCK: pass.** The fake CDP tab now models hit-test and input failures. The env-gated real-Chrome test is the conformance check against the real browser.
- **ARCH-CONSTRAINTS: pass.** The log is bounded at 64 entries and logs once per message, so a burst of failed inputs can't flood Couch's notice line.
- **ARCH-SECURE: pass.** The viewer gets fixed sentences only; causes go to the operator log alone, and the message text comes from the server's own error values.
- **ARCH-ORDER: pass.** `lastClick` is one nullable field with a written rule for clearing it: second click, drag, and reset. The rule is tested through timestamps the test supplies.
- **ARCH-FUNERAL: pass.** The `logged` map is in memory, capped, cleared when full, and dies with the server.

For upcoming work: a 503 caused by a client disconnecting (`context.Canceled`) is also logged. That is bounded, but it may add noise to the notice line if viewers drop connections often.

**7. Plan revision recommendations:** none. The Plan and Log match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestWithTabFailuresAreDriveErrors, TestWithTabGivesUpOnADialog and TestUnrecordedChromeWithTabIsNotLive each fail when the matching fix is reverted (mutation-checked).
  - id: BR-2
    disposition: addressed
    note: |
      gofmt -l on cmd/internal/webcast lists only query/events.go and server.go, neither touched in this window; both files named in BR-2 are clean.
  - id: BR-3
    disposition: addressed
    note: |
      drive.js:607 reset() clears lastClick; the node test for a click across a reset fails without it (mutation-checked).
```
