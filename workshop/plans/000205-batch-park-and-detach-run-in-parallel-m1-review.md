# Boundary Review — pair#205 (milestone M1)

| field | value |
|-------|-------|
| issue | 205 — batch park and detach run in parallel |
| repo | pair |
| issue file | workshop/issues/000205-batch-park-and-detach-run-in-parallel.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1640a14915fdc9e6ae4896b6337a23995195f765..3670c67597feb2c5b0efc8e487b3e70c92f5649d |
| command | sdlc milestone-close --issue 205 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-07T00:26:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers what it set out to do. There is an in-memory `ThreadGate` with token-keyed re-entry. It is held at every lifecycle entry the plan lists. Composites re-enter through their context. The drains (`Leave`, `RecoverActiveParks`, `AbortStarted`) wait instead of refusing. Busy refusals reach the console as `MenuEvent.Busy`, and the atlas and Spec are updated.

The diff is consistent with the plan and its Revisions. I ran couchcore and couchtty under `-race`. The only failures were the PTY/zellij tests this environment blocks; nothing gate-related failed.

Four Important findings block SHIP:
- the open-transaction bypass in `Couch.Park` can start a new park without holding the gate;
- moving the abort onto its own goroutine adds an unlocked writer to the actor registry before M2's registry mutex exists;
- a busy continuation clears expected-exit marks that other operations set;
- two of the three wait-mode drains have no test showing they wait.

None of these is reachable often under M1's single worker, but M2 removes the serialization that hides them.

1. **Strengths**
   - `threadgate.go:59-68`: the admission rule is a pure function (`gateDecision`), and re-entry is keyed on a per-hold token rather than the address. This closes the stale-context bypass, and the table test pins it.
   - `park.go` `leaveOne`: it waits on the gate first, then re-reads with `GetThread`, then decides. The decision is never made from the snapshot row, and `TestLeaveDecidesFromTheRecordAfterWaiting` pins this.
   - `couch.go:858` `startStillOwnsThread`: the address-scoped quiesce is gated on identity, and the handle cleanup runs exactly once per path. `TestAMismatchedAbortStillClosesItsOwnHandle` is meaningful because `StartSpawn` owns the session, and `abort_started_test.go:35` asserts the matching case does quiesce.
   - `TestEveryLifecycleEntryRefusesAHeldThread` enumerates the whole class, not one instance (ARCH-PURPOSE).
   - `parkworker.go:79-85` now frees the address before signalling done. This is a real ordering fix and has a regression guard.

2. **Critical:** none.

3. **Important**
   - **`Couch.Park`'s open-transaction bypass has a check-then-act gap in `normal` mode** (`park.go` `Couch.Park`, ARCH-ORDER).
     - It reads `current.Park != nil` and skips the gate. `PairLifecycleController.Park` then re-reads the record.
     - If the transaction closed in between, it mints a new nonce and starts a fresh park without a hold. That can happen when, for example, a relaunch has finished its park phase and is now resuming while still holding the gate. This is the #214 race class.
     - `retry`, `recover` and `abandon` re-check `Park == nil` and fail safely; only `normal` can begin a new transaction.
     - Fix: on the bypass path, route `normal` to `Retry`, so the bypass can only join an existing transaction. Add a test that holds the gate, closes the transaction between the read and the submit, and asserts busy or "no active park".
   - **`AbortStarted` now writes `c.reg` from a `GoTracked` goroutine without a lock** (`couchcmd/run.go:765`, `couch.go:860`, ARCH-ORDER).
     - Before this change, it was serialized with `Forget` (`couch.go:1173`, which runs on the console goroutine) and with other aborts. Now two failed attaches, or an abort and a `Forget`, can lose each other's registry update both in memory and in `Store.Save`.
     - The registry mutex is deferred to M2 Task 8b, but M1 introduces the new writer.
     - Fix: bring `regMu` forward to cover at least `AbortStarted` and `Forget`, or record in the plan Revisions that M1 widens a known race until 8b lands.
   - **A busy `continue-thread` clears expected-exit marks that other operations set** (`console_continuation.go:256-268`).
     - It deletes `expectedExits` for every pane on the thread. But `retry-continuation` sets marks when it is enqueued (`console.go:1631`), and park/detach set them when they succeed (`console.go:1887`).
     - If the busy completion lands between those marks and the deliberate child exit, that exit is reported as unexpected. This is rare under one worker and likely under M2.
     - Fix: when enqueueing, record on the watch which pane IDs this `continue-thread` marked, and drop only those.
   - **The drains' wait paths are untested.** The plan named `TestLeaveAndParkRecoveryWaitForAHolder`, but only `Leave` got a test.
     - Missing: a test that `RecoverActiveParks` waits for a holder and then recovers.
     - Missing: a test that `AbortStarted` waits for a holder, then quiesces when the identity still matches.
     - Missing: a test of the cancelled-wait path, where only the helper and terminal are ended and the record is left untouched.
     - Done-when asks for a test for every interleaving cell.

4. **Minor**
   - `startStillOwnsThread` returns true when the record has zero incarnations. That treats "no evidence" as ownership. Require at least one matching incarnation.
   - `TestACancelledParkStillRefusesOtherLifecycleOperations` asserts only `err != nil`. It should check the refusal cause (open park transaction, or "another park transaction").
   - The gate op name is `reconcile-continuation`, but the plan table says `continuation-status`. This is harmless, but the busy message the operator sees differs from the documented one.
   - The Task 5 full verification (unsandboxed `make -k test` and `go test ./...`) is not yet recorded in the Log. Record it at close.

5. **Test coverage notes**
   - The gate, the entry table, both relaunch orderings, different-thread independence, `Leave` waiting, the stale snapshot, a mismatched abort, composite re-entry, the busy reattach skip, the busy notice, the continuation re-arm, and the `GoTracked` join are all covered.
   - Gaps are listed under Important above. The tests use blocking fakes rather than sleeps, apart from a 150 ms window in the `Leave` test.

6. **Architecture**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass. `Couch.Park` absorbed the executor's mode switch, and the `Leave` body became `leaveOne`. |
   | ARCH-PURE | Pass. `gateDecision` is pure, and the gate is a thin shell around it. |
   | ARCH-PURPOSE | Pass. All entries are enumerated and tested. |
   | ARCH-MOCK | Pass. The gate is in memory; tests use the existing stateful fakes. |
   | ARCH-CONSTRAINTS | Pass for M1. The bound is deferred to M2 by design. |
   | ARCH-SECURE | Pass. There is no new trust boundary. |
   | ARCH-ORDER | Flag: the three findings above (the park bypass gap, the unlocked registry writer, the expected-exit clearing). |
   | ARCH-FUNERAL | Pass. Holds are released by deferred calls, the map is bounded by the thread count, and `GoTracked` work is joined at shutdown. |

   For M2: Task 8b must cover the `AbortStarted` goroutine. The `Couch.Park` bypass should be revisited before the queue gets more than one worker.

7. **Plan revision recommendations**
   - Add a Revisions entry saying that M1 moves `AbortStarted` off the console goroutine ahead of the registry mutex (or that `regMu` was pulled into M1).
   - Narrow the `Couch.Park` revision text to "joins an existing transaction; never begins one unheld".

```findings
findings:
  - id: new
    severity: Important
    family: gate-bypass-check-then-act
    title: |
      Couch.Park's open-transaction bypass can begin a fresh park without holding the gate
    detail: |
      Couch.Park reads Park!=nil and skips the gate. If the transaction closes before PairLifecycle.Park re-reads the record, normal mode mints a new nonce and parks while another holder, such as a relaunch now resuming, holds the thread. Route normal to Retry on the bypass path and add a test.
  - id: new
    severity: Important
    family: shared-state-writer-without-lock
    title: |
      AbortStarted now writes c.reg from a GoTracked goroutine with no registry lock
    detail: |
      Before this change it was serialized with Forget (on the console goroutine) and with other aborts. Now concurrent aborts, or an abort and a Forget, can lose a registry update in memory and on disk. regMu (Task 8b) is deferred to M2. Pull it forward, or record that M1 widens the race.
  - id: new
    severity: Important
    family: cleanup-scope-wider-than-own-effect
    title: |
      A busy continue-thread clears expected-exit marks that other operations set on the thread's panes
    detail: |
      console_continuation.go deletes expectedExits for every pane on the thread. retry-continuation (console.go:1631) and park/detach success (console.go:1887) also set these marks, so a deliberate exit gets reported as unexpected. Track the IDs this enqueue marked and drop only those.
  - id: new
    severity: Important
    family: interleaving-cell-untested
    title: |
      The wait paths of RecoverActiveParks and AbortStarted have no test
    detail: |
      The plan named TestLeaveAndParkRecoveryWaitForAHolder; only Leave is tested. Add tests that RecoverActiveParks waits and then recovers, that AbortStarted waits and then quiesces on an identity match, and that a cancelled abort wait ends only the helper and terminal.
  - id: new
    severity: Minor
    family: absence-read-as-ownership
    title: |
      startStillOwnsThread returns true for a record with zero incarnations
    detail: |
      No incarnation is not evidence that this start still owns the thread. Require at least one matching incarnation before the address-scoped quiesce.
  - id: new
    severity: Minor
    family: test-asserts-any-error
    title: |
      TestACancelledParkStillRefusesOtherLifecycleOperations asserts only err != nil
    detail: |
      Assert the refusal cause (open park transaction, or "another park transaction") so an unrelated failure cannot pass the test.
```

---

## Re-review — 2026-10-07T01:07:28-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 205 — batch park and detach run in parallel |
| repo | pair |
| issue file | workshop/issues/000205-batch-park-and-detach-run-in-parallel.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1640a14915fdc9e6ae4896b6337a23995195f765..88314b720fbaeaf72c3fa0d07cc6de90e3e9a588 |
| command | sdlc milestone-close --issue 205 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-07T01:07:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

Round 1 is substantively closed. All four Important findings (BR-1..BR-4) are fixed in `75a8d3dc`, and every fix except BR-1's call site has a test that would go red if the fix were reverted. BR-6 is fixed too. Nothing blocks. What remains is three Minors: BR-5's code change has no regression test, one doc comment got orphaned, and the BR-3 expected-exit marks still have a narrow reverse-order gap. On verification: the targeted tests pass under `-race` (`TestConcurrentRegistryWritersLoseNoUpdate`, `TestParkJoinModeNeverBegins`, the wait/cancel tests, `TestABusyContinuationIsRetriedSilently`, `TestGoTracked…`), and `go vet` is clean. The full `couchcore` and `couchtty` runs fail only in tests that start a pty child, all with `ptychild: start …: operation not permitted`. The review shell can't spawn ptys, so that is the environment, not this diff. The same cause explains the 35s timeout in `TestSpawnComposesProductionPairRegistrationBoundary` and the `couchcmd` launcher-helper failures. Rerun them in a normal shell before close.

1. **Strengths**
   - `mutateRegistry` (`cmd/internal/couchcore/couch.go:1300-1317`) is one read-modify-write-save critical section, and the in-memory registry only updates after the save succeeds. Every `c.reg` / `c.names` access now goes through `registry()` or `mutateRegistry`; a grep finds no stray access. `Registry.Insert` / `RemoveActor` really are copy-on-write (`registry.go:57-85`), so readers can safely use their copy after the lock is released.
   - The BR-1 fix is behavior-preserving outside the race window. `PairLifecycleController.Park` already sends an open transaction to `retry` under the same nonce (`park.go:378-381`), so forcing normal→retry on the join path only changes the stale-read case. In that case `Retry` now refuses ("thread has no active park transaction") instead of minting a new transaction.
   - BR-3's `markThreadExitsLocked` returns only the marks this operation added, and both enqueue sites use it (`console.go:1627`, `console_continuation.go:223`).
   - The BR-4 tests check real effects: no quit or quiesce while the thread is held, then a closed and verified park, or exactly one quiesce, after release. The cancel test checks that the helper is dead, nothing was quiesced, and the registry is empty.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Orphaned doc comment:** inserting the registry helpers at `couch.go:1282-1292` left `withoutDead`'s doc comment as the header of `registry()`. Move the comment back above `withoutDead` (now at `couch.go:1319`).
   - **BR-5 has no test:** `startStillOwnsThread` now refuses a record with zero incarnations, but nothing checks it. Add a cheap test where `AbortStarted` runs against a record whose incarnations were cleared, and assert there are no quiesces.
   - **Expected-exit marks, reverse order:** a pane counts as "added" only if it wasn't already marked, so ownership is still first-come. Suppose a continuation marks pane X, a park later succeeds and re-marks X (`console.go:1883`), and the continuation's busy refusal is processed after that. The refusal deletes the park's mark. This is unlikely, because a busy refusal is normally posted before the holder releases, but it is the same rule broken from the other side. See the new finding below.

5. **Test coverage notes**
   - `TestParkJoinModeNeverBegins` pins the mapping but not the call site: removing `mode = parkJoinMode(mode)` at `park.go:235` keeps everything green. There is no seam between the two `GetThread` reads, so I accept the pure test plus the equivalence argument above. An injectable thread-read hook would close the gap (ARCH-ORDER).
   - The wait tests detect "still blocked" with a fixed 150ms sleep. That's fine for proving the wait happens, but it only observes one interleaving.

6. **Architectural notes for upcoming work**
   - **ARCH-DRY:** pass. The two hand-rolled mark loops collapsed into `markThreadExitsLocked`. The park/detach bridge loop at `console.go:1879` is a near-sibling that M2 could fold in.
   - **ARCH-PURE:** pass. `parkJoinMode` is pure; `mutateRegistry` takes a pure mutator.
   - **ARCH-PURPOSE:** pass. BR-2 swept every registry reader, not just the one named site.
   - **ARCH-MOCK:** pass. The tests use the existing stateful fakes (`pairlifecycletest`, `FakeThreadArtifactCollisionChecker`).
   - **ARCH-CONSTRAINTS:** pass. `mutateRegistry` documents that no zellij or other external call may run inside it. `withoutDead`'s liveness probe does run under the lock; that is bounded and documented.
   - **ARCH-SECURE:** N/A. No new untrusted input or secrets.
   - **ARCH-ORDER:** flag (Minor). Expected-exit marks are still a shared boolean set with no recorded owner; see the new finding.
   - **ARCH-FUNERAL:** pass. `watch.marked` is cleared on every completion, and nothing new is persisted.

7. **Plan revision recommendations:** none required. Optionally, add a Revisions note that the registry lock planned for M2 (Task 8b, `regMu`) moved into M1 because of BR-2.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      park.go:235 maps begin-capable modes to retry on the join path; controller Park already retries an open txn, so only the stale window changes (now refuses). Pure mapping test only; call site has no seam.
  - id: BR-2
    disposition: addressed
    note: |
      All c.reg/c.names access routed via registry()/mutateRegistry (grep-verified); TestConcurrentRegistryWritersLoseNoUpdate passes under -race.
  - id: BR-3
    disposition: addressed
    note: |
      watch.marked records own marks; TestABusyContinuationIsRetriedSilently goes red if the finish path reverts to delete-all.
  - id: BR-4
    disposition: addressed
    note: |
      TestParkRecoveryWaitsForAHolderThenRecovers, TestAbortStartedWaitsForAHolderThenQuiesces, TestACancelledAbortWaitEndsOnlyItsOwnHelper added and passing.
  - id: BR-5
    disposition: not-addressed
    note: |
      Code fix at couch.go startStillOwnsThread is correct, but no regression test covers the zero-incarnation record.
  - id: BR-6
    disposition: addressed
    note: |
      Asserts ResumeDiagnosticOf, "open park transaction", and "another park transaction" causes.
findings:
  - id: new
    severity: Minor
    family: cleanup-scope-wider-than-own-effect
    title: |
      Expected-exit marks have first-come ownership; a late busy refusal can still delete a park's re-mark
    detail: |
      This is the 2nd finding in family cleanup-scope-wider-than-own-effect. Rule: a shared expectedExits mark must record its owners (pane id -> set of operation ids, or a refcount), and an undo removes only its own ownership. Current state: markThreadExitsLocked skips already-marked panes, but park/detach success (console.go:1883) re-marks without recording an owner, so a continuation refusal processed after it deletes the park's mark. Only that ordering is affected, and it is unlikely.
  - id: new
    severity: Minor
    family: doc-comment-detached-by-insertion
    title: |
      withoutDead's doc comment now heads registry() at couch.go:1282
    detail: |
      The registry helpers were inserted between the comment and its function; move the comment back above withoutDead.
```
