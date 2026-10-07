# Boundary Review — pair#205 (milestone M2)

| field | value |
|-------|-------|
| issue | 205 — batch park and detach run in parallel |
| repo | pair |
| issue file | workshop/issues/000205-batch-park-and-detach-run-in-parallel.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 493bd8934fafc54d3344e0e66adda3e41139cb4b..9c729384bf744d8013eb9abdcfa96d30a5178db0 |
| command | sdlc milestone-close --issue 205 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T12:48:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what the plan asks for: one shared bound, `LifecycleParallelism = max(1, NumCPU/2)`, at `threadgate.go:17`, used by `Leave`'s fan-out, the park worker and the reattach pass. The console runs `Limit+1` queue workers, and every bound makes callers wait instead of refusing. The core code is correct. `Leave` collects outcomes by index and reports them in snapshot order. The park worker re-checks the duplicate address before capacity on every loop, and a close-and-replace channel wakes the waiters. The reattach reducer is still pure: it clones the pass before changing its in-flight map. I found no correctness bug. What keeps this from SHIP is test coverage. `couchtty`'s new `TestMain` pins `LifecycleParallelism = 1` for the whole package, so no console-level test ever runs more than one pass attempt or more than two queue workers. Two tests that plan Tasks 8 and 10 named were never written, and nothing in this window tests what happens when `Leave` is cancelled partway through its fan-out. I did not run the test suites: the review was read-only, and the issue Log records `couchcore` taking 553 s.

**1. Strengths**
- `park.go:193-232`: the fan-out uses outcome and error slices indexed by record. Reporting in snapshot order is then free (`TestLeaveResultIsInSnapshotOrder` pins it), and `errors.Join` keeps the failures of all sibling threads (D6, `TestLeaveFinishesSiblingsWhenOneFails` checks that the failed thread's tag appears).
- `parkworker.go:68-88`: waiting for capacity rechecks "another park transaction owns this address" on every pass, before the capacity check, so a waiter cannot slip past a same-address conflict. A waiter with the same nonce joins the existing future.
- `menu_reattach.go:224-264`: the reducer stays pure (it clones before writing). It ends the pass exactly when the queue and the in-flight set are both empty, and the 2^64 counter guard is kept. `TestReattachBoundedPassGeneratedSequences` checks independent invariants (in-flight ≤ limit, each thread attempted at most once, Done ⇔ empty) over 200 seeds.
- Placeholder chips are sorted by attempt number (`loadingAttempts`), and a 20-frame test checks the order is stable.
- `FakeProcOps` is now safe for concurrent use. Its `OnSignal` hook runs outside the fake's lock, which gives tests a way to control ordering without sleeps on the gate.

**2. Critical findings**
None.

**3. Important findings**
- **Concurrent paths are not tested above a bound of 1** (`reattach_testmain_test.go:14`, `parkworker_test.go`, `leave_test.go`). This is the 2nd finding in family `interleaving-cell-untested`, so the fix should apply a rule, not patch this one spot.
  - **Rule:** each concurrency policy M2 declares must have a test that runs at bound > 1 and controls the ordering. Pinning the global bound to 1 for a package does not cover any policy that only exists above 1.
  - **The cases this leaves uncovered:**
    - (a) `TestLeaveParkAlongsideAnotherParkNeverFails` (plan Task 8) is missing. It is the test for the reason capacity was changed to wait rather than refuse.
    - (b) Cancelling `Leave` partway through ("stops starting, started finish") has no fan-out test. `park_test.go:416` only covers cancellation before any work starts.
    - (c) Plan Task 10 Step 3 asked for console-path tests under N workers: a burst of N reattaches, and a late abort racing a relaunch. All `couchtty` console tests now run with 1 pass attempt and 2 workers.
    - (d) Quit in the middle of a pass with `Limit > 1` is covered only at the reducer level.
  - **Fix:** add (a) and (b) in `couchcore` using `withParallelism`. Add one console fixture test that sets the bound above 1 and runs a pass with several attempts plus an operator operation. Alternatively, record in `## Revisions` why the reducer-level coverage is enough for (c) and (d).

**4. Minor findings**
- `console.go:180-183` and `atlas/couch.md` claim "an operator's gesture never waits behind its attempts". Remote and continuation jobs share the spare worker, so this holds only when no remote or continuation work is queued.
- `max(1, …)` is applied twice: in `New` (`queueWorkers`) and again in `Run`.
- `errors.Join` in `Leave` can produce N copies of "context canceled" with no thread tag when the gate wait (`holdWait`) is cancelled.
- `LifecycleParallelism` is a mutable package `var`. Tests change it globally (`TestMain`, `withParallelism`), which would race with any future `t.Parallel()` test.
- `TestParkWorkerCapacityWaitHonoursContext` and `TestParkWorkerBoundsAndCoalesces` assert absence with 20–30 ms sleeps. If `cancel` fires before `Submit` blocks, the capacity test passes without exercising the wait. The plan says "don't assert sleeps".

**5. Test coverage notes**
`couchcore` covers the `Leave` bound (peak equals the bound), sibling failure, ordering and the counted SIGTERM-once check. The park worker covers wait and cancel. `couchtty` covers the reducer thoroughly and the queue with multiple workers. Integration through the console at bound > 1 is not covered (see the Important finding).

**6. Architectural notes**

| Principle | Result | Notes |
|---|---|---|
| ARCH-DRY | pass | One bound serves all three consumers. |
| ARCH-PURE | pass | `Limit` is fixed at arm time, so the reducer reads no global. |
| ARCH-PURPOSE | pass | Every Done-when bullet is either delivered or revised by the operator, with a Revisions entry. |
| ARCH-MOCK | pass | The fake is stateful and now concurrency-safe. |
| ARCH-CONSTRAINTS | pass | The bound and its reason are stated, and every bound waits rather than refuses. The measurement was waived by the operator and recorded. |
| ARCH-SECURE | N/A | No new input boundary. |
| ARCH-ORDER | flag | The policy table is written down, but the cells that only exist above bound 1 are untested (see the Important finding). |
| ARCH-FUNERAL | pass | The in-flight map is bounded by `Limit` and emptied by `finishReattach`. The park worker's `freed` channel is replaced, not accumulated. Nothing durable is created. |

The park worker has no FIFO fairness. A submitter can lose the race for a freed slot more than once. This is bounded and acceptable at this scale, but worth knowing if starvation ever shows up.

**7. Plan revision recommendations**
- Add a `## Revisions` entry for Task 8's `TestLeaveParkAlongsideAnotherParkNeverFails` and Task 10 Step 3's console-path tests under N workers: either they were written, or they were replaced by named tests and the reason is given.
- Revise Task 11's "#204 counting suite / O(N) store writes" to the SIGTERM-once count. The issue already records this; the plan should too.

```findings
findings:
  - id: new
    severity: Important
    family: interleaving-cell-untested
    title: |
      Concurrency cells that exist only above bound 1 are untested; couchtty pins LifecycleParallelism=1 package-wide
    detail: |
      2nd finding in family interleaving-cell-untested. Rule: each declared M2 concurrency policy needs a test that runs at bound > 1 with controlled ordering; pinning the global bound to 1 covers none of them. Missing: Task 8 TestLeaveParkAlongsideAnotherParkNeverFails; Leave cancellation partway through the fan-out (stops starting, started finish); Task 10 Step 3 console-path tests under N workers (burst of reattaches, late abort vs relaunch); quit mid-pass at Limit>1 covered only in the reducer. Fix: add the couchcore tests with withParallelism and one console test with the bound raised, or record a Revisions entry saying why reducer coverage suffices.
  - id: new
    severity: Minor
    family: doc-claim-exceeds-mechanism
    title: |
      queueWorkers comment and atlas say an operator gesture never waits behind pass attempts; remote and continuation jobs share the spare worker
    detail: |
      console.go:180-183 and atlas/couch.md. Narrow the wording to: the pass alone cannot occupy every worker.
  - id: new
    severity: Minor
    family: test-asserts-absence-by-sleep
    title: |
      Park worker capacity tests assert absence and wait with 20-30ms sleeps
    detail: |
      TestParkWorkerCapacityWaitHonoursContext passes without exercising the wait if cancel fires before Submit blocks. Expose a waiting hook or signal so the test synchronizes instead of sleeping.
  - id: new
    severity: Minor
    family: joined-error-loses-subject
    title: |
      Leave joins one untagged context error per cancelled thread
    detail: |
      When holdWait is cancelled, the joined error repeats "context canceled" N times without naming the threads. Wrap each error with the thread tag in the fan-out goroutine.
```

---

## Re-review — 2026-10-07T12:58:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 205 — batch park and detach run in parallel |
| repo | pair |
| issue file | workshop/issues/000205-batch-park-and-detach-run-in-parallel.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 493bd8934fafc54d3344e0e66adda3e41139cb4b..f878a542fe07f27c551ba14f731f3bc21ca63972 |
| command | sdlc milestone-close --issue 205 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T12:58:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

All four prior findings are handled, and the code checks out. BR-9 got two real tests above bound 1:
- `TestStopMidPassCancelsEveryParallelAttemptAndStartsNoMore` runs end to end through the console worker pool at bound 3.
- `TestLeaveCancelledMidFanOutFinishesStartedAndStartsNoMore` runs at bound 2.

Writing the second test turned up a real dispatch race: `select` picks at random when the context is done and a unit is free. It also exposed a wrong policy sentence. Both are fixed. The cells left untested have written reasons in the issue Log, and the reasons hold: adoption runs on the console goroutine; the park-alongside-park and late-abort cells are covered at the `parkWorker` / gate level. BR-10 and BR-11 are fixed directly. BR-12 is fixed for the case it named (cancelled `holdWait`). Two untagged error sites remain, which is minor.

I ran these and they all passed:
- `go test -race -count=20 -run 'TestLeave|TestParkWorker' ./cmd/internal/couchcore` passed.
- The couchtty reattach and queue tests passed at `-count=5 -race`.
- The new Stop test passed 10 of 10 under `-race`.

Nothing blocks SHIP.

**1. Strengths**
- `park.go:211-228`: checks `ctx.Err()` both before and after `select`, and returns the token it just took. That closes the random-choice race properly. `lessons.md` records the general rule, not just this case.
- `park.go:196-200`: the cancellation policy was corrected from "started finish" to "stop at their own safe points", and the text points back to how the serial version behaved.
- `parkworker.go:55-57, 83-87`: the `onWait` hook is read under `mu`. The capacity tests now wait on a real "blocked on capacity" signal, and the `default:` absence check is deterministic because work cannot run before `Submit` returns.
- `menu_reattach.go:224-265`: the reducer stays pure. It clones before writing, the in-flight set is keyed by attempt, and placeholders are sorted by attempt so the chips don't jitter.
- `console_reattach_parallel_test.go`: Stop cancels every parallel attempt and exactly 3 resumes start. That is a direct, bound-above-1 check of the quit-mid-pass cell.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- `leave_test.go:177`: the test name still says "FinishesStarted". That is the claim this round corrected, so rename it to something like `…StopsStartingMore`. See the findings block for the family rule.
- `park.go:332-338, 341, 360`: three `leaveOne` errors still lack the thread tag: the non-NotFound `GetThread` error and both "controller unavailable" errors. The generic fix is to wrap once in the fan-out goroutine and drop the per-site tags.
- The `Leave` dispatch-check fix is an equivalent mutant at the new test's level: `Detach` refuses a cancelled ctx before signalling. The only red-without-fix evidence is the roughly 50% flake in `TestLeaveCanceledDoesNotRetireDeadIncarnation`. The issue Log already acknowledges this.

**5. Test coverage**
- The concurrency cells above bound 1 are now: Stop mid-pass (console), Leave cancelled mid-fan-out, the signal-exactly-once count, and the park-worker capacity wait and context cancel.
- The couchtty `TestMain` pin to 1 is safe with `raiseBound`, because neither package uses `t.Parallel()`.

**6. Architecture**
- **ARCH-DRY:** pass. The fan-out uses one units channel, and the reattach helpers are shared.
- **ARCH-PURE:** pass. The reducer reads `Limit` from state, not from the global.
- **ARCH-PURPOSE:** pass, apart from the BR-12 leftover noted above.
- **ARCH-MOCK:** pass. The proc and lifecycle fakes sit behind the existing seams.
- **ARCH-CONSTRAINTS:** pass. The bound is `max(1, NumCPU/2)`, capacity waits instead of refusing, and the queue has `Limit+1` workers.
- **ARCH-SECURE:** N/A. No new untrusted input and no secrets.
- **ARCH-ORDER:** pass. The in-flight state is a keyed map handled by the reducer, and the cancellation ordering is now tested. The only weak spot is that the dispatch check is enforced by a flaky oracle rather than a deterministic one.
- **ARCH-FUNERAL:** pass. Nothing durable is created, and goroutines are joined through `wg` and `c.workers`.

**7. Plan revisions recommended**
- Add a `## Revisions` entry to the plan (not only the issue Log) saying that Task 8's `TestLeaveParkAlongsideAnotherParkNeverFails` and Task 10 Step 3's console burst and late-abort tests were replaced by mechanism-level coverage, with the Log's reasons. As it stands, the plan still names tests that don't exist.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      Two bound>1 tests added (console Stop mid-pass at bound 3; Leave cancel mid-fan-out at bound 2), both pass under -race; remaining cells justified in issue Log (should also be a plan Revisions entry).
  - id: BR-10
    disposition: addressed
    note: |
      console.go:180-183 and atlas/couch.md now say "the pass alone" and name the shared spare worker.
  - id: BR-11
    disposition: addressed
    note: |
      parkWorker.onWait hook (read under mu) replaces the 20-30ms sleeps; absence check is now deterministic.
  - id: BR-12
    disposition: addressed
    note: |
      holdWait cancellation now names the thread; GetThread and controller-unavailable errors in leaveOne remain untagged (minor; wrap once in the fan-out goroutine).
findings:
  - id: new
    severity: Minor
    family: doc-claim-exceeds-mechanism
    title: |
      Leave cancellation test name still claims started threads finish
    detail: |
      2nd in family. Rule: a cancellation-policy claim in a name, comment or atlas line must state what the test asserts. Corrected in park.go:196-200 but not in the leave_test.go:177 name. Rename it, and grep this issue's tests and comments for "finish" against cancellation.
```
