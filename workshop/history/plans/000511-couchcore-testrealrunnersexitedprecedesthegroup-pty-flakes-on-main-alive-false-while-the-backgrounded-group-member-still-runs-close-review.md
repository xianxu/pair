# Boundary Review — pair#511 (whole-issue close)

| field | value |
|-------|-------|
| issue | 511 — couchcore: TestRealRunnersExitedPrecedesTheGroup/pty flakes on main (Alive = false while the backgrounded group member still runs) |
| repo | pair |
| issue file | workshop/issues/000511-couchcore-testrealrunnersexitedprecedesthegroup-pty-flakes-on-main-alive-false-while-the-backgrounded-group-member-still-runs.md |
| boundary | whole-issue close |
| milestone | — |
| window | 407f578d0e1ee78b3b75aeacb26e1f726cec230b..b79d23de4adcb6410c9485d3dfec5c8a00c93c46 |
| command | sdlc close --issue 511 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T19:44:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This documentation-only closure points to an implementation already present in the pinned base. The handshake preserves both real-runner conformance checks, and independent repeated tests pass. One minor evidence-count error does not block shipping.

1. **Strengths**
   - Confirmed merge `eb63b610` is an ancestor of the pinned base.
   - `cmd/internal/couchcore/ptyrunner_test.go:162–186` establishes readiness after installing SIGHUP ignore and checks successful parent exit.
   - Retained qualification logs contain 50 successful survivor repetitions, including both runner variants.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings**
   - Issue #511, line 54: the cited baseline log records **3/5 failures**, not 2/5. Correct the count. This is the only instance of this discrepancy introduced in the window.

5. **Test coverage notes**
   - Independently passed survivor conformance with `-count=50` and a race-enabled run.
   - Inspected retained baseline, qualification, and full-suite logs; did not independently rerun the full suite or recreate parallel host load.
   - Working tree remains unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** closure references the landed fix without duplicating implementation.
   - **ARCH-PURE — pass:** no executable changes or altered IO boundaries.
   - **ARCH-PURPOSE — pass:** the referenced implementation addresses fixture readiness while preserving both runner checks.
   - No new surface requires atlas or README updates.

7. **Plan revision recommendations:** None; correct the Log count only.

```findings
findings:
  - id: new
    severity: Minor
    family: evidence-count-accuracy
    title: |
      Baseline failure count disagrees with its cited log
    detail: |
      workshop/issues/000511-couchcore-testrealrunnersexitedprecedesthegroup-pty-flakes-on-main-alive-false-while-the-backgrounded-group-member-still-runs.md:54 says 2 of 5 repetitions failed, but /tmp/pair542-descendant-baseline.log contains three failing PTY repetitions and two passing repetitions. Change the count to 3 of 5. This is the sole instance introduced in this window.
```
