---
id: 000485
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '7a889136d09a0bc657116abc3eb66e0052854e96' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: wrapcmd harness reap timeouts (50/100ms) and couchtty/wrapcmd ordering asserts fail under loaded -race

## Problem

Found by the #479 soak (200 `-race` runs per touched package under parallel load). These are outside #479's late-event family:

**Real product timeouts set short by the test:**
- wrapcmd `TestHarnessTTYCaptureBoundsRetainedOutput` (harness_tty_live_test.go:495): "captureHarnessTTY: child reap exceeded 50ms".
- wrapcmd `TestHarnessTTYCaptureAnswersCursorPositionQuery` (harness_tty_live_test.go:333): "child reap exceeded 100ms".

**Ordering or state assertions that lose under load:**
- couchtty `TestCouchParentCaptureStableAcrossChildMouseModes` (console_mouse_test.go:438, 4 of ~128 runs): "child changed parent capture". The asserted host write holds the full first paint, so the snapshot boundary isn't what the test assumes.
- couchtty `TestCouchProductionSoak` (terminal_soak_test.go:341, 2 of ~128): "console exit1: couch: terminal: terminal: presenter unavailable" after the assertions completed, during shutdown.
- wrapcmd `TestOrientationSerializedInputScheduler/overlay-after-paste` (orientation_test.go:82, 1 of 200): wanted cancel, got submitted; an overlay admitted after the paste raced the submit.
- couchcore `TestRecoveryObservationDeadlineCancelsBlockingObserver/30ms` (recovery_execute_test.go:485): "blocked observer did not stop exactly once: context deadline exceeded calls0" in a loaded parallel run after #479's merge of main; passes alone.

**Pre-existing `-race`-only failures, reproduced on origin/main 299d9f14e during #479's close (`make test` runs without `-race`, so they're unseen there):**
- terminalqualify `TestResourceCasesObserveBoundedBackendStorage` (resource_cases_test.go:12): "parser-memory-bound variant 0: context deadline exceeded" every run.
- couchcmd `TestCapturePTYBothBoundaries` (capture_test.go:236/281): "marker never reached host" after its 30 s guard; startup ingress short (3.3 MB of 3.9 MB).
- wrapcmd `TestCodexWorkingNotificationReachesCouchStatusAndSwitcher`: data race.
- wrapcmd `TestHarnessTTYFixtureConformance`: ~9 min under `-race`, past a 9 m binary timeout.

**Sweep the untouched packages (from #479's close review):** #479 converted every positive `case <-time.After(N): t.Fatal…` guard in the packages it touched to `testwait.After(t)`. Packages it didn't touch (broadcast, webcast, termcmd, couchmessage, and others; `grep -rn "case <-time.After" --include='*_test.go' cmd/internal`) still guard events with real-time windows. The same goes for polling deadlines (`deadline := time.Now().Add(N)` plus Fatal) in untouched files, which become `time.Now().Add(testwait.Budget(t))`: couchcmd message_deliveries:49, message_lifecycle:170, message_service:257/339/378/652, wait_notice:272/324/465; couchcore provision_lock:117, ptyrunner:17; couchsingleton subprocess:64; wrapcmd launch_watcher:139. And one stopwatch ceiling: launcher session_reap_test:175 (assert the outcome that proves the bound).

Same family as #480: a test races a timer it doesn't own, or asserts an order the product doesn't guarantee.

## Spec

## Done when

- Each listed test either gives the product its timer through a seam (as #480 did) or waits for the event its assertion depends on; no constant is just raised.
- Before each change, a temporary product delay shows the race.
- The touched packages pass 200 `-race` runs under parallel load.

## Plan

- [ ] Harness reap: inject the reap timer, or assert the reap outcome rather than its latency.
- [ ] Mouse capture: find the boundary the assertion means, and wait for it.
- [ ] Soak "presenter unavailable" at shutdown: root-cause (it may be a product exit-path bug).
- [ ] Orientation overlay-after-paste: order the overlay admission against the submit.
- [ ] Sweep: convert the remaining positive `time.After` guards in untouched packages to `testwait.After(t)`; leave absence windows (empty case bodies) alone.
- [ ] Soak.

## Log

### 2026-10-12
