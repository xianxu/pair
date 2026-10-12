---
id: 000476
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'b5106c2590966ec149ca96045aecd87e3a10f647' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T03:57:00-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "c15c62bb", done: "c9efc146"}
---

# tests: two more flaky pair tests on real-time windows (couchcore retention nested locks; couchcmd capture PTY boundaries)

## Problem

While fixing pair#473 (PR #265), pair:5 saw two other tests fail once each under load, untouched by that change: `couchcore` `TestRetentionNestedStoreLocksYieldAndReleaseRoot/forget` (a 150 ms real-time window, retention_test.go:604) and `couchcmd` `TestCapturePTYBothBoundaries/regular` (a drain timeout, capture_test.go:238). Same family as #473: assertions on real-time windows that a loaded machine misses.

## Spec

As in #473: reproduce by widening the window or loading the machine, fix the root cause (an injectable clock, waiting on the event, deterministic ordering), never a longer timeout or a retry; fix product code if the race is there. Then sweep the suite for the same pattern (fixed real-time windows asserted under load) and list any others in the Log, fixing the cheap ones and filing the rest.

## Done when

- Both tests pass 200 runs under -race and load, and in repeated full-suite runs; causes in the Log.
- The sweep's findings are recorded (fixed or filed).

## Plan

- [x] Reproduce each flake by widening its window; record the cause.
- [x] couchcore retention: drop the 150 ms window, since the held lock makes any return the proof.
- [x] couchcmd capture: `terminalcapture.Config.CloseTimeout` plus `consoleTraceConfig.captureCloseTimeout`; content tests wait out the drain.
- [x] Sweep: fix the terminalcapture `Close` family (same cause); file the rest as #479, #480, #481.
- [x] Proof: 200 `-race` runs of each under parallel load; full-suite runs.

## Log

### 2026-10-12
- 2026-10-12: closed — Both flakes reproduced by widening (200ms sleep in forget fails retention_test.go:604; 2.5s worker sleep fails capture_test.go:238 with drain timed out); with the fixes both pass with the widening in place; blocking-lock mutation hangs the retention test into the binary timeout. 200 -race runs each under 4x parallel load: 0 fails (targets, terminalcapture pkg, couchcmd ^TestCapture). go test ./... x3 and make -k test: exit 0, no failures. Sweep filed as #479/#480/#481.; review verdict: SHIP
- **couchcore cause (measured).** Nothing waits on the nested lock: every `forget`/`detach`/... path takes the store lock nonblocking (`retentionReadLock` polls `LOCK_NB` within its bound) and returns `ErrCoordinatorBusy`. The 150 ms `time.After` timed the whole call, including its file I/O, and the call ran slow under the full `-race` suite. A 200 ms sleep at the top of `ForgetStoreArchiveReceipt` fails `/forget` at `retention_test.go:604` with "maintenance blocked on nested Couch lock past its deadline", although nothing blocks. Fix: the test holds the nested lock until the call returns, so a return at all proves it yielded. It now waits on `done` with no window, and passes 3/3 with the 200 ms sleep in place. Mutation check: a blocking `acquireThreadStoreLockMode(..., false)` added to forget hangs the test into the binary timeout (20 s in the check), and the goroutine dump shows the flock.
- **couchcmd cause (measured).** `capture_test.go:238` is `con.CloseCapture()` returning `ErrCloseTimeout`: `Recorder.Close` waits a fixed production 2 s (`defaultOptions().closeTimeout`) for the worker to write the end record, and the test inherited that bound. A 2.5 s sleep before the worker's `out.Close` fails both subtests at `:238` with "terminal capture drain timed out". The 2 s bound is right for the product (a diagnostic wait after terminal restore, documented as such), so the fix is a seam: `terminalcapture.Config.CloseTimeout` (zero = 2 s, negative refused before anything is created), threaded through `consoleTraceConfig.captureCloseTimeout`. Tests whose subject is content set `waitForDrain` (no bound), so the drain is their sync point. With the 2.5 s sleep in place both subtests pass 2/2.
- **Sweep.** About 40 at-risk tests (an Explore pass over `cmd/**/_test.go`; ranks are by reading, not measured). Fixed here, same cause as couchcmd: nine terminalcapture/couchcmd tests that assert a clean or specific `Close` result under the default 2 s drain (`TestPrivateSessionsRoundTripAndClose`, `TestConcurrentAdmissionsRemainOrdered`, `TestConcurrentRecordAndClosePreserveFinalAdmissionOrder` (its 1 s select also went, since it was tighter than the drain), `TestBlockedStartupBurstSurvives` (~4,000 records), `TestStatusNilAndConfiguration`, `TestUsageNotificationsOnlyCrossPercentages`, `TestCaptureStorageOpenKeepsCompletedReservation`, `TestCaptureHostWritePreservesPartialFailure`, `TestCaptureConfiguredLimitAppliesAtComposition`), plus the queue/file-limit tests that expect a specific error. They use `drainOptions()`/`drainConfig`. The two that test the timeout keep their own 20 ms/100 ms bounds. Filed the rest by kind of fix: #479 short hang guards (couchtty `waitUpTo` ~47 sites and siblings), #480 tests racing a real timer or sleeping as sync (need injected clocks or sync points), #481 wall-clock perf budgets in the default suite (gate behind `PAIR_PERF_BUDGETS`).
- **Proof.** 200 `-race` runs of each target test, 4 processes per test, both packages at once: 0 failures. Then on the final code, 200 `-race` runs of the whole terminalcapture package and of couchcmd `^TestCapture`, 4 processes each: 0 failures. Full suite: `go test ./... -count=1` three times (retention/data-dir env scrubbed, sandbox off) and `make -k test`: all four exited 0, and no test failed in any run.
