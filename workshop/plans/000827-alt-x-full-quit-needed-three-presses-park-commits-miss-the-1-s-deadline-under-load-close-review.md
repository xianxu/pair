# Boundary Review — pair#827 (whole-issue close)

| field | value |
|-------|-------|
| issue | 827 — Alt+x full quit needed three presses: park commits miss the 1 s deadline under load |
| repo | pair |
| issue file | workshop/issues/000827-alt-x-full-quit-needed-three-presses-park-commits-miss-the-1-s-deadline-under-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | ce5146c9cc9dfaea20e25b74888052fbbeecd862..81b47b3744ce3a937a181df9ddf84f3d4a3b25f3 |
| command | sdlc close --issue 827 |
| reviewer | claude (anthropic; author anthropic,openai) |
| isolation | fenced |
| timestamp | 2026-10-23T02:56:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the Leave/park diff; now checking console handling of skipped threads and the tests.

Targeted and race tests pass. Now reverting the store-lock fix to confirm the fleet test catches it.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The change does what the issue asks. One confirmed Alt+x now owns the whole park leave. A pure retry policy (`decideLeaveParkRetry`) allows a fresh drive only when a rollback is proven and the source is the same (that proof is `parkRolledBackError`). Existing transactions retry through `recoverJoined` and keep their nonce. Each drive waits for its exact submitted future. When the quit is incomplete, the user gets one typed summary (`LeaveParkIncompleteError`) and Couch stays open. The commit threshold moved from 1s to 10s on measured evidence (709ms worst case at 30 slots under load 64). The relaunch observation timeout still has its own 1s value.

The store-lock root cause is real and has a regression test. I reverted the polled wait in `park_recovery_store.go`. Four tests went red: the three `TestForegroundStoreWrite*` tests failed in all three runs, and `TestLeaveParkRetriesProvenRollbackWithinOneQuit` failed in one of three runs. They pass again with the fix restored. The targeted cohort passes, and the leave-park, AwaitJoined, decide and store-lock tests pass 3× under `-race`.

What stops a clean SHIP is one mismatch between the plan and the code, on skipped (unknown-incarnation) threads. The plan's state table says these get a partial summary and Couch stays open. The code still treats a skip-only full quit as success, and Couch exits. There are also a few smaller reporting and envelope nits.

1. **Strengths**
   - `leave_park.go:104` `decideLeaveParkRetry` is pure. `TestDecideLeaveParkRetryInvariants` checks all 2^6 combinations of evidence flags, at every drive count, against invariants written separately from the implementation.
   - `parkworker.go:600` `AwaitJoined` is an additive Leave-only adapter. Public `Park`/`Recover` keep `Await` because they go through `parkAwaiting`/`recoverAwaiting` with a typed `parkAwait`. This matches the PQ-1 contract.
   - `park_recovery.go:543` attaches the positive no-publication proof as a type, not by matching error text, and the error text stays the same.
   - `storelock_unix.go:663` makes coordinated foreground writes poll for the lock within a bound. Maintenance (`archive_gc.go:59`, wait 0) still gives up immediately. The fleet test with 2.5s modeled commits (`alternatingClock`) covers the Done-when for 15 and 30 slots. It also counts drives, publications and peak fan-out.
   - `TestLeaveParkBudgetEndNamesEveryUnfinishedThread` checks that when the budget ends, every thread is accounted for exactly once, including threads that never started.

2. **Critical findings:** none.

3. **Important findings**
   - **Skipped threads let a full quit count as complete (ARCH-PURPOSE; plan/code contradiction).** The plan's State-and-recovery row "Unsafe/unreadable/replaced source, unknown incarnation … One partial summary … Couch stays open" doesn't hold.
     - `leaveParkOne` (`leave_park.go:160`) returns `leaveSkipped, nil`, so `Leave` (`park.go:368`) returns no error.
     - The console (`console.go:2130`) then prints "left X occupied" and calls `c.Stop()`.
     - `FormatLeaveParkSummary` formats skipped threads, but it only runs on the error path, so a skip-only quit never reaches it.
     - No test covers LeavePark with a skipped thread.
     - Fix: either make a skipped thread under `LeavePark` an incomplete result (error plus summary), or add a `## Revisions` entry saying skipped threads keep the old report-and-exit behaviour. Add a test either way.

4. **Minor findings**
   - The recovery advice is repeated in each unresolved summary line. `leaveParkOne` already wraps the error with `withParkRecoveryAdvice`. `leaveParkUnresolved` then computes `ParkRecoveryAdvice` again for `Next`, so the line reads "…; <advice>; next: <advice>".
   - `park.go:343-353` calls `leaveParkUnresolved` one target at a time after `wg.Wait()`. Each call can take up to `recoveryObservationTimeout` (5s), outside the 60s budget. With many open transactions this adds a lot of unbudgeted exit time (ARCH-CONSTRAINTS).
   - `park_recovery_store.go:56`: the comment "It never waits on a nested busy store" is now stale, because `withUpdateLockContext` polls for up to `storeReadLockWait`.
   - `TestRelaunchObservationTimeoutIndependentOfCommitThreshold` only compares constants. It doesn't exercise relaunch's read timing.
   - `leaveParkOne` repeats `leaveOne`'s preamble: hold, get, not-found, active check and the single-live check (ARCH-DRY). A shared helper would keep the two in step.
   - `onStoreLockContended` is a package-level mutable hook. That is safe today because the tests aren't parallel, but it is fragile.

5. **Test coverage notes:** The pure decision, the joined future, the store-lock wait, the 15/30 fleet run, the budget end, unreadable records, the open-transaction retry through Recover, and the console notice are all covered. Gaps:
   - LeavePark with skipped threads.
   - A late-completing joined drive after the budget has expired, checked end to end through `Leave` (it is covered only at the future level).
   - The relaunch observation behaviour (see the constant-only test above).

   The native 4/8-slot runs are evidence in `native-full-quit.txt`. I did not re-run them: they need the PTY and Zellij harness.

6. **Architectural notes**
   - ARCH-DRY: flag (Minor; repeated preamble and repeated advice computation).
   - ARCH-PURE: pass. The decision is pure and IO stays in `leaveParkOne`.
   - ARCH-PURPOSE: flag (skipped threads, Important above).
   - ARCH-MOCK: pass. It uses the stateful `pairlifecycletest` model and a real private store, and the native harness provides conformance.
   - ARCH-CONSTRAINTS: mostly pass. The commit threshold is measured, the budget and drive cap are enforced, and fan-out is unchanged. Flag (Minor): the sequential advice pass runs outside the budget.
   - ARCH-SECURE: pass. Unreadable records are reported, not trusted, and there are no secrets.
   - ARCH-ORDER: pass, with a note. The evidence struct holds 6 booleans, but every combination is tested. Persisted Park state stays the authority, and the exact-future join prevents releasing the thread while effects are unsettled.
   - ARCH-FUNERAL: pass. Retry state dies with the call, and no new durable family is created.
   - ARCH-EVOLVE: pass. No persisted format changes. Only the Diagnostic text value differs ("10s").
   - #580 (wall-time cutoff vs committed-slow) is still open, correctly. A future change that distinguishes the two should reuse `ParkCommitOverrunError`.

7. **Plan revision recommendations**
   - If the skipped-exit behaviour is kept, add a `## Revisions` entry: "Skipped (unknown-incarnation) threads under LeavePark keep the pre-#827 report-and-exit behaviour (named in the exit report). The State table row for 'unknown incarnation' applies only to unresolved, not skipped, targets."

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: plan-state-table-contract-drift
    title: |
      Skipped (unknown-incarnation) threads let a full quit return success and Couch exits, contrary to the plan's state table
    detail: |
      leaveParkOne returns leaveSkipped with nil error (leave_park.go:160), so Leave(LeavePark) returns no error and console.go:2130 reports and calls Stop. The plan's State table says unknown incarnation yields one partial summary and Couch stays open. FormatLeaveParkSummary lists skipped threads but only runs on the error path. Either treat skipped as incomplete under LeavePark (with a test) or record a plan Revision keeping the old exit behaviour.
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-derived-report-text
    title: |
      Unresolved summary lines repeat the recovery advice in both Reason and Next
    detail: |
      leaveParkOne wraps its error with withParkRecoveryAdvice, then leaveParkUnresolved computes ParkRecoveryAdvice again for Next, producing "...; <advice>; next: <advice>".
  - id: new
    severity: Minor
    kind: defect
    family: unbudgeted-post-budget-work
    title: |
      Recovery advice for unresolved targets runs one at a time after the 60s budget, up to 5s each
    detail: |
      park.go:343-353 calls leaveParkUnresolved per target after wg.Wait, each with recoveryObservationTimeout (5s) on a WithoutCancel context, adding unbudgeted exit latency with many open transactions.
  - id: new
    severity: Minor
    kind: defect
    family: stale-invariant-comment
    title: |
      recoveryBackend comment says it never waits on a nested busy store, now false
    detail: |
      park_recovery_store.go:56; withUpdateLockContext now polls within storeReadLockWait.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"36606122-ce8c-4669-b6b6-a1b1ed5ac82f","repository":"github.com/xianxu/pair","issue_id":"000827","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/03745bda5067b9fb","reviewed_head":"81b47b3744ce3a937a181df9ddf84f3d4a3b25f3","slot":"pair:6","requested_at":"2026-10-23T09:53:11.997873Z"},"phase":"finished","pid":16264,"started_at":"2026-10-23T09:53:12.099614Z","finished_at":"2026-10-23T09:56:12.191304Z","session_id":"8c6994ef-e25c-4dd1-b458-c489dfdeb385","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.988151125s
head: "81b47b3744ce3a937a181df9ddf84f3d4a3b25f3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
