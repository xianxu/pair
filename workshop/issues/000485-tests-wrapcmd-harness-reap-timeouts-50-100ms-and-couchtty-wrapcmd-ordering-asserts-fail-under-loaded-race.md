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
- [ ] Soak.

## Log

### 2026-10-12
