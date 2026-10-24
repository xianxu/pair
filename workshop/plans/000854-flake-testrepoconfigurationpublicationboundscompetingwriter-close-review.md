# Boundary Review — pair#854 (whole-issue close)

| field | value |
|-------|-------|
| issue | 854 — flake: TestRepoConfigurationPublicationBoundsCompetingWriter |
| repo | pair |
| issue file | workshop/issues/000854-flake-testrepoconfigurationpublicationboundscompetingwriter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6b04d05d7c294d9091ae966e48f05c29cac96a6b..c4d4ed487292cadb4bb629d42381ae25406a8468 |
| command | sdlc close --issue 854 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-24T07:53:10-07:00 |
| verdict | SHIP |

## Review

Diff and callers read; now running the new tests and a revert-mutation check.

```verdict
verdict: SHIP
confidence: high
```

The fix goes after the root cause the Spec describes, and the tests prove it. `storeLockBound` (`cmd/internal/couchcore/threadstore.go:199`) caps the `store.lock` wait at the time the caller has left. A wait ended by that cap now returns `context.DeadlineExceeded` instead of `ErrThreadStoreBusy` (`storelock_unix.go:39-41`; `threadstore.go:225-226` for a cap of zero). There is no retry loop and no longer timeout, as Done-when requires. Both halves of the Done-when test clause are covered: the deadline-bounded case and the more-budget-than-the-wait busy case. I checked the fix with a mutation in this checkout: when `storeLockBound` never caps the wait, `TestStoreLockWaitEndsAtTheCallerDeadline` fails with `lock thread store: thread store busy; retry`, which is the issue's symptom. With the fix restored, the two new tests and the flaky test pass 3 times in a row. Nothing blocks SHIP.

1. **Strengths**
   - The deadline flag is decided by the caller and passed in `storeWait`. A real busy from a nested lock inside the locked work is never relabelled as a deadline (`threadstore.go:190-193`).
   - The bound is computed inside the coordinator callback (`park_recovery_store.go:26-27`), so time spent waiting for the coordinator is already subtracted. That matches the "remaining migration budget" check in `repopreferences_adoption_test.go:401-403`.
   - The test uses a blind `check` to reproduce the deschedule race deterministically, without depending on scheduler timing (`storelock_test.go:65-86`). It also shows the old fixed-wait behaviour on the same fixture.
   - The change stays small. `withStoreLockChecked` keeps its signature, so the four literal-0 callers are untouched (`archive_gc.go:59`, `threadstore.go:176`, and two tests). `withUpdateLockContext` is the only caller with a nonzero wait, and it moved to the bounded path.
   - The atlas is updated (`atlas/couch.md:3601-3603`), and what it says matches the code.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Doc comment left on the wrapper.** The #827 comment at `threadstore.go:179-183` (check selects polled locking, wait semantics) still sits on `withStoreLockChecked`, which is now a one-line wrapper. `withStoreLockBounded` (`threadstore.go:208`) holds that logic and has no doc. This is the only instance in this window.

5. **Test coverage notes**
   - The deadline-cut path, the spent-budget path (cap of zero), and the within-budget busy path are all tested.
   - The coordinator branch of `withUpdateLockContext` is covered by the `coordinator` subtest of the flaky test. The new unit tests don't cover it on their own.

6. **Architecture notes**
   - **ARCH-DRY:** pass. No duplicated logic. The two paths through `withUpdateLockContext` share `storeLockBound`.
   - **ARCH-PURE:** pass. `storeLockBound` is a small pure decision, apart from reading the clock through `time.Until`. The IO stays in the lock-acquisition shell.
   - **ARCH-PURPOSE:** pass. The root cause is fixed, not hidden. I swept every `acquireThreadStoreLockWithin` and `withStoreLockChecked` caller: only `withUpdateLockContext` passes a nonzero wait, and both of its branches now derive from `storeLockBound`.
   - **ARCH-EVOLVE:** pass. No persisted format changes.

7. **Plan revision recommendations:** none. The Plan and Log match the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: doc-comment-follows-logic
    title: |
      The #827 lock-semantics doc comment stays on the withStoreLockChecked wrapper; withStoreLockBounded has none
    detail: |
      threadstore.go:179-183 describes the check and wait semantics now carried out in withStoreLockBounded (threadstore.go:208), which has no doc comment. Move or extend the comment so it covers the new function and the storeWait deadline flag. This is the only instance in the window.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f6945bc9-1a53-42a9-9bdd-290251b7e2e0","repository":"github.com/xianxu/pair","issue_id":"000854","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/cf2eb91c631f1824","reviewed_head":"c4d4ed487292cadb4bb629d42381ae25406a8468","slot":"pair:6","requested_at":"2026-10-24T14:51:13.063152Z"},"phase":"finished","pid":77672,"started_at":"2026-10-24T14:51:13.213048Z","finished_at":"2026-10-24T14:53:04.489674Z","session_id":"014b125f-6f0e-4a34-a30a-a4fc54cc14aa","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.343986791s
head: "c4d4ed487292cadb4bb629d42381ae25406a8468"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
