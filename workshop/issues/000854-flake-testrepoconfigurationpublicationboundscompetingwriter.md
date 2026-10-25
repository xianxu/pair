---
id: 000854
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
card_mirror: '7c0214e034bb2182eb415109876a259ad8b63305' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T05:56:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestRepoConfigurationPublicationBoundsCompetingWriter]
flow: {kind: quick, provenance: inferred, spec: "9d9e8613", done: "79b590e7"}
actual_hours: N/A
---

# flake: TestRepoConfigurationPublicationBoundsCompetingWriter

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestRepoConfigurationPublicationBoundsCompetingWriter: Same-exact-head fail then focused pass on 954e3caea7078371c91c7523e86a70bdad751658. Recorded affected attempt01392f0a6d0a73ce07338d4fa6c68104 failed only this whole test/store at repopreferences_adoption_test.go417: expected deadline cause but warning was thread store busy;retry (store5.38s). Parser proven ancestor is whole top-level; no subtest-only registration. Exact detached HEAD verified by git, unchanged tree, standard runtimebundle preparation; go test -json ./cmd/internal/couchcore -run ^TestRepoConfigurationPublicationBoundsCompetingWriter$ -count=1 -timeout=90s then PASS10.562s, store5.07s and coordinator5.22s. JSON log /Volumes/TestScratch/pair841-leave-repro-gub_n4rh/841-repoprefs-repro.jsonl. Original failed log SHA prefix37b0a76723f5. #841 is bell display only and does not change repopreferences/store code. Focused diagnostic proves intermittency only, not a passing affected status; script groups did not run after original Go failure. Root cause needs independent repair, no timeout inflation or retry fix.

## Spec

Root cause (pair:6, from the code): #827 made `withUpdateLockContext` (couchcore/park_recovery_store.go) poll
`store.lock` for a fixed `storeReadLockWait` (1 s) through `acquireThreadStoreLockWithin`, independent of the
caller's context deadline. Each poll runs `check()` (the context check), then tests its own wait deadline. When the
caller has about as much budget left as the wait (the test leaves ~1 s of the migration budget), the two deadlines
coincide. A deschedule between `check()` and the wait-deadline test then returns `ErrThreadStoreBusy` ("thread store
busy; retry") instead of the deadline cause. The coordinator lock waits on `ctx.Done()` and is not affected.

Fix: a store-lock wait never outlives the caller's deadline. `storeLockBound(ctx, wait)` caps the wait at the time
the context has left. When that cap ends a busy wait, the store path reports `context.DeadlineExceeded`, not busy.
A caller with more budget than the wait still gets a genuine busy after 1 s. No retry, and no longer timeout.

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- A store-lock wait is bounded by the caller's deadline and reports the deadline cause, even when the context check
  cannot see it yet (deterministic test). A caller with more budget than the wait still gets busy (test).

## Plan

- [x] `storeLockBound` in couchcore. `withUpdateLockContext` computes the bound inside the coordinator callback,
  maps a capped busy (or a cap already at zero) to `context.DeadlineExceeded`, and leaves an uncapped busy as it is.
- [x] Tests: a held `store.lock` with a check blind to the deadline returns `DeadlineExceeded` at the caller's
  deadline, not busy at 1 s (it shows busy on the old composition); with more budget than the wait, busy as before;
  `TestRepoConfigurationPublicationBoundsCompetingWriter` repeated under load.

## Log



- 2026-10-24: closed — Re-close after integrating main (9c97f28cf; #873, #877 landed) and the TL-approved scope add: both empty-startup acquisitions take storeLockBound(ctx, storeReadLockWait) (dc92f440a). Review BR-2 fixed in 68c48a465: store_lock_deadline_wiring_test.go drives a deadline-only context (Err stays nil) through park recovery (coordinated and plain), empty startup and the #877 publication join; mutation-checked: reverting park_recovery_store.go or startup_creation.go to the fixed wait fails with busy after 1 s, and the old ctx.Err join condition fails the publication test. BR-1 doc comment moved to withStoreLockBounded; BR-3 (preview reads) out of scope. make test-affected-inventory clean. Pre-close: sdlc-local/affected pass on 68c48a46 (25 packages + lua/review/shell, 9m24s). --no-actual under TL standing authorization (ariadne#603): "measurement branch proof: measurement snapshot is 135933 bytes, over the version 4 bound of 131072 (107 sources, 1 branch records, 642 branch intervals)". (done-when: contract moved since change-code; judged by the close review); review verdict: FIX-THEN-SHIP
- 2026-10-24: closed — Actual not measured (--no-actual per the TL fleet notice for ariadne#593): close refused with "measurement snapshot does not serialize: ... measurement exceeds 65536 bytes". Root cause fixed, no retry or timeout bump: withUpdateLockContext polled store.lock for a fixed storeReadLockWait independent of the caller deadline, so with ~1 s of budget left a deschedule between the context check and the wait-deadline test reported busy. storeLockBound caps the wait at the remaining budget and a wait the cap ends reports context.DeadlineExceeded. TestStoreLockWaitEndsAtTheCallerDeadline (deadline-blind check: fixed wait busy, bounded wait DeadlineExceeded; mutation of the cap fails with the issue symptom), TestStoreLockWaitWithinBudgetStaysBusy; TestRepoConfigurationPublicationBoundsCompetingWriter 5/5 under load; full couchcore passes (333 s); sdlc-local/affected pass on c4d4ed487 (25 packages).; review verdict: SHIP
### 2026-10-23

- Implemented in c088041de. `storeWait{d, deadline}` carries the deadline mark into `acquireThreadStoreLockWithin`, so a genuine busy from inside the locked work (a nested preview lock) is never relabelled. `withStoreLockChecked` keeps its signature for the literal-0 callers; `withUpdateLockContext` uses `withStoreLockBounded`, computing the bound inside the coordinator callback, after the coordinator wait has spent budget.
- Tests: `TestStoreLockWaitEndsAtTheCallerDeadline`. With a check blind to the deadline (the race), the fixed wait reports busy; the bounded wait reports `DeadlineExceeded` within the caller's budget, and a spent budget still reports the deadline. Mutation check: never capping fails it with "thread store busy; retry", the issue's symptom. `TestStoreLockWaitWithinBudgetStaysBusy` covers a caller with more budget. `TestRepoConfigurationPublicationBoundsCompetingWriter` passed 5/5 under load (the full couchcore package running beside it). The full couchcore package passes alone in 333 s. Two concurrent full runs each passed go test's default 10-minute timeout; the dump showed a test that had started 0 s before, not a hang.

## Revisions

- 2026-10-24: scope add approved by TL ops:0 and agreed with pair:7. After #873 (PR 594) landed, main was merged in (9c97f28cf; atlas/couch.md conflict resolved by keeping both notes) and #873's two empty-startup acquisitions (startup_creation.go) now take storeLockBound(ctx, storeReadLockWait). Measured: this is behavior-equivalent there, because startup's check already returns ctx.Err while polling; a mutation back to the uncapped wait still passes TestEmptyStartupLockWaitEndsAtTheCallerDeadline. The test pins the deadline contract; the wiring keeps one wait policy. Also hardened #877's publication loop to join the last busy when a capped wait returns DeadlineExceeded before ctx.Err is set.
- 2026-10-24: re-close review BR-2 (Important): no test pinned the storeLockBound wiring at its call sites. Added store_lock_deadline_wiring_test.go with a context whose deadline is near but whose Err stays nil, so only a capped wait can end at it: park recovery (coordinated and plain), empty startup, and the #877 publication join. Mutation-checked: reverting park_recovery_store.go or startup_creation.go to the fixed wait turns the matching test red with busy after 1 s; restoring the old ctx.Err join condition turns the publication test red. BR-1 (Minor): the #827 lock doc comment moved to withStoreLockBounded. BR-3 (Minor, preview reads keep a fixed wait) is outside #854's coordinated-write scope and left as a note.
- 2026-10-24: third close round FIX-THEN-SHIP, no blocking. BR-4 fixed: the cap-only publication test now runs under a 5 s bound and fails with "publication never ended" when the park-recovery cap is reverted (mutation-checked) instead of hanging. BR-5 is the reviewer's sandbox refusing git init; the host run and mutation results above answer it. BR-3 stays out of scope.
