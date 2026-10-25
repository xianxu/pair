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

---

## Re-review — 2026-10-24T20:47:46-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 854 — flake: TestRepoConfigurationPublicationBoundsCompetingWriter |
| repo | pair |
| issue file | workshop/issues/000854-flake-testrepoconfigurationpublicationboundscompetingwriter.md |
| boundary | whole-issue close |
| milestone | — |
| window | dc3031079bddd3c160cd9481fdef94f1ff29c83b..dc92f440ace7f8b6546de1eb8ba51f4cfd2a3f67 |
| command | sdlc close --issue 854 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-24T20:47:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checked the diff and the flaky test. Next I'll check the non-unix lock variant, then run the new tests and a mutation test.

Found a sibling fixed-wait in the preview read path; checking whether the atlas claim covers it.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The root-cause fix is right. `storeLockBound` (`threadstore.go:199`) caps the store-lock wait at the caller's remaining budget, and `storeWait.deadline` carries the reason down to `acquireThreadStoreLockWithin`. A wait that the cap cuts short now returns `context.DeadlineExceeded` instead of busy. A real busy from nested locked work is never relabelled, because the relabel happens only at the outer wait's deadline test. The new primitive tests pass here, and so does the flaky test. One thing stands between this and SHIP: the fix at the actual root-cause site isn't pinned by any deterministic test. I swapped both `withUpdateLockContext` call sites in `park_recovery_store.go` back to the old `withStoreLockChecked(..., storeReadLockWait)`, and every deterministic test still passed: `TestStoreLockWait*`, `TestParkRecovery*`, and `TestRepoConfigurationPublicationBoundsCompetingWriter`. The tests cover the primitive, not the consumer the issue is about. The prior finding BR-1 is also still open at head.

Done-when: current. Both clauses (deadline-bound wait reporting the deadline cause; more budget than the wait still gives busy) cover what this diff builds. The startup wiring added in Revisions behaves the same as before, and the publication-loop hardening falls under the deadline-cause clause. Its missing test is captured in the Important finding below.

1. **Strengths**
   - `storelock_unix.go:39`: the deadline is reported only at the wait's own deadline test, so busy returned from inside the locked work (a nested preview) is never turned into a deadline error.
   - `threadstore.go:199-206`: `storeLockBound` is pure, clamps a spent budget to 0, and leaves callers that have no deadline or more budget unchanged. `TestStoreLockWaitWithinBudgetStaysBusy` checks the struct directly.
   - `storelock_test.go:67`: the test uses a check that can't see the deadline, which reproduces the race deterministically, and it shows the old composition gives busy on the same setup.
   - `park_recovery_store.go:26`: the bound is computed inside the coordinator callback, after the coordinator wait has used part of the budget.
   - The atlas is updated in the same range (`atlas/couch.md:3619`).

2. **Critical:** none.

3. **Important**
   - The consumer wiring has no regression test (mutation confirmed). Instances of this family in the window:
     - `park_recovery_store.go:27` (coordinator path) and `:30` (plain path), the root-cause site.
     - `repopreferences_store.go:102`, the new "capped DeadlineExceeded before `ctx.Err` is set" join. Nothing exercises it.
     - `startup_creation.go:62-63`. Revisions already admit the startup test passes under mutation.
   - Fix sketch: a test-only context whose `Deadline()` returns about now+150ms while `Err()` returns nil. This makes the check blind at the consumer level. Then:
     - With `store.lock` held, assert `withUpdateLockContext` returns `DeadlineExceeded` well before `storeReadLockWait`. The uncapped version gives busy after 1s.
     - Use the same context in `rememberRepoConfigurationContext` (with `Done` taken from a real context) to pin the "store stayed busy" join.

4. **Minor**
   - BR-1 is still open: `withStoreLockBounded` at `threadstore.go:208` has no doc comment. The #827 semantics comment is still on the wrapper (`:179-183`).
   - ARCH-PURPOSE sibling: `retentionReadLock` (`retention.go:330-345`), used by `withPreviewLockContext`, still waits a fixed `storeReadLockWait` regardless of the caller's deadline. Its `errors.Join(err, ctx.Err())` (`threadstore_preview.go:82`) has the same race: when the two deadlines nearly coincide, it reports busy alone. This is the only other fixed-wait instance in the tree. The atlas scopes the guarantee to coordinated writes, so this is a note, not a contract break.

5. **Test coverage notes**
   - Passed here: `TestStoreLockWaitEndsAtTheCallerDeadline`, `TestStoreLockWaitWithinBudgetStaysBusy`, `TestRepoConfigurationPublicationBoundsCompetingWriter`.
   - The `TestEmptyStartupLock*` tests, including the new one, couldn't run in this sandbox: the fixture's `git init` was refused (sdlc #339). This is environmental and affects the pre-existing startup tests too.
   - The boundary edge where remaining budget equals the wait exactly isn't capped. That's harmless, because the wait then ends no later than the deadline.

6. **Architectural notes**
   - ARCH-DRY: pass. `withStoreLockChecked` is now a thin adapter over `withStoreLockBounded`, so there's one implementation. The preview path's separate poll loop (`retentionReadLock`) is older duplication of `acquireThreadStoreLockWithin` that predates this window. If the Minor sibling is fixed, both could share one `storeWait`-aware poller.
   - ARCH-PURE: pass. `storeLockBound` is a pure decision and the IO stays in the lock poller.
   - ARCH-PURPOSE: the issue's purpose is delivered, with the sibling above noted (Minor).
   - ARCH-EVOLVE: pass, no persisted format changes.
   - No README surface is involved.

7. **Plan revision recommendations**
   - Add a Revisions note that the publication-loop join (`repopreferences_store.go:102`) is changed behaviour and say which test pins it, once one is added.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      At head, withStoreLockBounded (threadstore.go:208) still has no doc comment; the #827 comment remains on the wrapper at 179-183.
findings:
  - id: new
    severity: Important
    family: regression-test-pins-consumer-wiring
    title: |
      No deterministic test pins the storeLockBound wiring at withUpdateLockContext, the root-cause site
    detail: |
      Mutation run: reverting park_recovery_store.go:27 and :30 to withStoreLockChecked(run, ..., storeReadLockWait) leaves TestStoreLockWait*, TestParkRecovery* and TestRepoConfigurationPublicationBoundsCompetingWriter all passing; the unit tests call withStoreLockBounded directly. Family instances in window: park_recovery_store.go:27, :30; repopreferences_store.go:102 (capped DeadlineExceeded before ctx.Err join, untested); startup_creation.go:62-63 (Revisions admit mutation-passing). Fix: a test ctx whose Deadline is near but Err returns nil, driving withUpdateLockContext (expect DeadlineExceeded before storeReadLockWait) and rememberRepoConfigurationContext (expect the store-stayed-busy join).
  - id: new
    severity: Minor
    family: store-lock-wait-bounded-by-caller-deadline
    title: |
      Preview reads (retentionReadLock) keep a fixed storeReadLockWait independent of the caller deadline
    detail: |
      retention.go:330-345 polls for a fixed wait and returns ErrThreadStoreBusy; threadstore_preview.go:82 joins ctx.Err, which may not yet be set when the deadlines coincide, the same race this issue fixed for writes. Only remaining fixed-wait site in the tree; atlas scopes the guarantee to coordinated writes, so this is a note.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2aad0b9e-9e8d-4fd7-94ec-b1a1391f88e7","repository":"github.com/xianxu/pair","issue_id":"000854","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/68b7b7a9fc6fc50e","reviewed_head":"dc92f440ace7f8b6546de1eb8ba51f4cfd2a3f67","slot":"pair:6","requested_at":"2026-10-25T03:44:51.823093Z"},"phase":"finished","pid":34418,"started_at":"2026-10-25T03:44:51.970027Z","finished_at":"2026-10-25T03:47:43.312101Z","session_id":"16e8cf29-02b0-4f1d-894f-d144a5bea0e4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 8.704021583s
head: "dc92f440ace7f8b6546de1eb8ba51f4cfd2a3f67"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
