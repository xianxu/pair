# Boundary Review — pair#476 (whole-issue close)

| field | value |
|-------|-------|
| issue | 476 — tests: two more flaky pair tests on real-time windows (couchcore retention nested locks; couchcmd capture PTY boundaries) |
| repo | pair |
| issue file | workshop/issues/000476-tests-two-more-flaky-pair-tests-on-real-time-windows-couchcore-retention-nested-locks-couchcmd-capture-pty-boundaries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8382f9756d36cd7221d353d89c5edf70d9d5c0e4..dafc05aada3cce72a18e5ca6b58de78685b1991e |
| command | sdlc close --issue 476 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T05:35:03-07:00 |
| verdict | SHIP |

## Review

Diff read; now checking the retention test's context deadline to see if the new wait-on-done still detects a blocking acquire.

Code looks sound; running the touched packages' tests to confirm they pass at head.

```verdict
verdict: SHIP
confidence: high
```

Both flakes are fixed at the root, not by raising timeouts. The retention test no longer times the whole call. It now waits for the call to return while the test holds the nested lock, so a blocking acquire can never return. The capture flake came from the production 2 s drain bound that the tests inherited. That bound is now a config seam (`terminalcapture.Config.CloseTimeout`, carried through `consoleTraceConfig.captureCloseTimeout`). Production keeps 2 s because `tracesForRuntime` leaves the field at zero. A negative value is refused before anything is created, and that refusal is tested. I checked the Close-result family: every test in the tree that asserts a clean or specific `Close`/`CloseCapture` result now uses `drainOptions()`/`drainConfig`/`waitForDrain`. The exceptions are the two tests whose subject is the timeout, and they keep their own bounds. The callers left alone (couchtty `capture_status_test.go:21,103`, couchcmd `capture_test.go:122,140,320`, `run_test.go`) ignore the Close result or never enable capture. terminalcapture (full package) and the couchcore retention test pass under `-race` at head. The couchcmd package would not build in this review checkout because the generated `runtimebundle` assets are missing, so I didn't run the couchcmd tests myself; that's a checkout/environment issue, not a code issue. Nothing blocks SHIP.

1. **Strengths**
   - `recorder.go:105-110`: the seam is minimal. Zero means default, a negative value is refused before any file is opened (pinned at `recorder_test.go:419`), and both the default and the override are pinned (`recorder_test.go:425-440`).
   - `run.go:652`: the override reaches `terminalcapture.Open(path, captureConfig)` at `run.go:673`. The production path cannot set it.
   - `retention_test.go:596-603`: the new reasoning holds. The lock is held until `done`, so any return proves the call yielded rather than blocking, and the error check still requires busy or deadline.
   - The sweep went past the two named tests to the whole family, with the rest filed as #479, #480 and #481. This satisfies ARCH-PURPOSE.
   - `time.NewTimer(MaxInt64)` is safe: Go's runtime caps timer overflow.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **The retention test no longer limits how long the call may take** (`retention_test.go:603`). Before, a code path that polled for a long but bounded time before returning busy failed at 150 ms. Now it passes; only an unbounded block is caught, at the test binary's timeout. This is accepted by design. If the bound matters, pin the poll bound as a value rather than with wall-clock time.
   - **ARCH-DRY:** `waitForDrain` is defined twice, at `couchcmd/capture_test.go:87` and `terminalcapture/recorder_test.go:36`. They are test-only constants in different packages, so this is tolerable. An exported `terminalcapture` sentinel would give it one home.
   - **Unwrapped line:** `atlas/couch-live-capture.md:109` breaks the file's line-wrap convention.

5. **Test coverage notes:** The new config field is covered for zero, a positive value and a negative value. Production keeps the 2 s default; I confirmed that by reading the code, since the default path (zero settings) is the same code `tracesForRuntime` uses. couchcmd could not be built in this checkout because the generated assets are missing.

6. **Architectural notes**
   - ARCH-DRY: pass (see the Minor note above).
   - ARCH-PURE: pass. Config is validated in `Open` before any IO, and the timer stays at the Close edge.
   - ARCH-PURPOSE: pass. The family was swept, and the remaining classes are filed, separable follow-ups.

7. **Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: test-subject-bound-unpinned
    title: |
      Retention nested-lock test no longer pins the magnitude of the yield bound
    detail: |
      retention_test.go:603 waits on done with no window. A regression that keeps polling for a long but bounded time before returning ErrCoordinatorBusy now passes; only an unbounded block is caught, at the binary timeout. If the bound matters, pin the poll-bound constant directly with no wall clock. Only instance in this window.
  - id: new
    severity: Minor
    family: duplicated-test-helper
    title: |
      waitForDrain constant is defined in two test packages
    detail: |
      couchcmd/capture_test.go:87 and terminalcapture/recorder_test.go:36 (ARCH-DRY). An exported terminalcapture sentinel (for example NoCloseTimeout) would give it one home. These are the only two instances.
  - id: new
    severity: Minor
    family: doc-line-wrap
    title: |
      Unwrapped long line in atlas/couch-live-capture.md:109
    detail: |
      The added parenthetical breaks the file's line-wrap convention. This is the only instance.
```
