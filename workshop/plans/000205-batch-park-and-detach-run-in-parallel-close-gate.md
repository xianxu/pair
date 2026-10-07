---
gate: boundary-review
issue: 205
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T00:26:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Couch.Park's open-transaction bypass can begin a fresh park without holding the gate
          detail: Couch.Park reads Park!=nil and skips the gate. If the transaction closes before PairLifecycle.Park re-reads the record, normal mode mints a new nonce and parks while another holder, such as a relaunch now resuming, holds the thread. Route normal to Retry on the bypass path and add a test.
          family: gate-bypass-check-then-act
          round: 1
        - id: BR-2
          severity: Important
          title: AbortStarted now writes c.reg from a GoTracked goroutine with no registry lock
          detail: Before this change it was serialized with Forget (on the console goroutine) and with other aborts. Now concurrent aborts, or an abort and a Forget, can lose a registry update in memory and on disk. regMu (Task 8b) is deferred to M2. Pull it forward, or record that M1 widens the race.
          family: shared-state-writer-without-lock
          round: 1
        - id: BR-3
          severity: Important
          title: A busy continue-thread clears expected-exit marks that other operations set on the thread's panes
          detail: console_continuation.go deletes expectedExits for every pane on the thread. retry-continuation (console.go:1631) and park/detach success (console.go:1887) also set these marks, so a deliberate exit gets reported as unexpected. Track the IDs this enqueue marked and drop only those.
          family: cleanup-scope-wider-than-own-effect
          round: 1
        - id: BR-4
          severity: Important
          title: The wait paths of RecoverActiveParks and AbortStarted have no test
          detail: The plan named TestLeaveAndParkRecoveryWaitForAHolder; only Leave is tested. Add tests that RecoverActiveParks waits and then recovers, that AbortStarted waits and then quiesces on an identity match, and that a cancelled abort wait ends only the helper and terminal.
          family: interleaving-cell-untested
          round: 1
        - id: BR-5
          severity: Minor
          title: startStillOwnsThread returns true for a record with zero incarnations
          detail: No incarnation is not evidence that this start still owns the thread. Require at least one matching incarnation before the address-scoped quiesce.
          family: absence-read-as-ownership
          round: 1
        - id: BR-6
          severity: Minor
          title: TestACancelledParkStillRefusesOtherLifecycleOperations asserts only err != nil
          detail: Assert the refusal cause (open park transaction, or "another park transaction") so an unrelated failure cannot pass the test.
          family: test-asserts-any-error
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T01:07:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: park.go:235 maps begin-capable modes to retry on the join path; controller Park already retries an open txn, so only the stale window changes (now refuses). Pure mapping test only; call site has no seam.
          round: 2
        - id: BR-2
          disposition: addressed
          note: All c.reg/c.names access routed via registry()/mutateRegistry (grep-verified); TestConcurrentRegistryWritersLoseNoUpdate passes under -race.
          round: 2
        - id: BR-3
          disposition: addressed
          note: watch.marked records own marks; TestABusyContinuationIsRetriedSilently goes red if the finish path reverts to delete-all.
          round: 2
        - id: BR-4
          disposition: addressed
          note: TestParkRecoveryWaitsForAHolderThenRecovers, TestAbortStartedWaitsForAHolderThenQuiesces, TestACancelledAbortWaitEndsOnlyItsOwnHelper added and passing.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: Code fix at couch.go startStillOwnsThread is correct, but no regression test covers the zero-incarnation record.
          round: 2
        - id: BR-6
          disposition: addressed
          note: Asserts ResumeDiagnosticOf, "open park transaction", and "another park transaction" causes.
          round: 2
      findings:
        - id: BR-7
          severity: Minor
          title: Expected-exit marks have first-come ownership; a late busy refusal can still delete a park's re-mark
          detail: 'This is the 2nd finding in family cleanup-scope-wider-than-own-effect. Rule: a shared expectedExits mark must record its owners (pane id -> set of operation ids, or a refcount), and an undo removes only its own ownership. Current state: markThreadExitsLocked skips already-marked panes, but park/detach success (console.go:1883) re-marks without recording an owner, so a continuation refusal processed after it deletes the park''s mark. Only that ordering is affected, and it is unlikely.'
          family: cleanup-scope-wider-than-own-effect
          round: 2
        - id: BR-8
          severity: Minor
          title: withoutDead's doc comment now heads registry() at couch.go:1282
          detail: The registry helpers were inserted between the comment and its function; move the comment back above withoutDead.
          family: doc-comment-detached-by-insertion
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-10-07T12:48:38-07:00"
      agent: claude
      findings:
        - id: BR-9
          severity: Important
          title: Concurrency cells that exist only above bound 1 are untested; couchtty pins LifecycleParallelism=1 package-wide
          detail: '2nd finding in family interleaving-cell-untested. Rule: each declared M2 concurrency policy needs a test that runs at bound > 1 with controlled ordering; pinning the global bound to 1 covers none of them. Missing: Task 8 TestLeaveParkAlongsideAnotherParkNeverFails; Leave cancellation partway through the fan-out (stops starting, started finish); Task 10 Step 3 console-path tests under N workers (burst of reattaches, late abort vs relaunch); quit mid-pass at Limit>1 covered only in the reducer. Fix: add the couchcore tests with withParallelism and one console test with the bound raised, or record a Revisions entry saying why reducer coverage suffices.'
          family: interleaving-cell-untested
          round: 3
        - id: BR-10
          severity: Minor
          title: queueWorkers comment and atlas say an operator gesture never waits behind pass attempts; remote and continuation jobs share the spare worker
          detail: 'console.go:180-183 and atlas/couch.md. Narrow the wording to: the pass alone cannot occupy every worker.'
          family: doc-claim-exceeds-mechanism
          round: 3
        - id: BR-11
          severity: Minor
          title: Park worker capacity tests assert absence and wait with 20-30ms sleeps
          detail: TestParkWorkerCapacityWaitHonoursContext passes without exercising the wait if cancel fires before Submit blocks. Expose a waiting hook or signal so the test synchronizes instead of sleeping.
          family: test-asserts-absence-by-sleep
          round: 3
        - id: BR-12
          severity: Minor
          title: Leave joins one untagged context error per cancelled thread
          detail: When holdWait is cancelled, the joined error repeats "context canceled" N times without naming the threads. Wrap each error with the thread tag in the fan-out goroutine.
          family: joined-error-loses-subject
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-07T12:58:10-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: Two bound>1 tests added (console Stop mid-pass at bound 3; Leave cancel mid-fan-out at bound 2), both pass under -race; remaining cells justified in issue Log (should also be a plan Revisions entry).
          round: 4
        - id: BR-10
          disposition: addressed
          note: console.go:180-183 and atlas/couch.md now say "the pass alone" and name the shared spare worker.
          round: 4
        - id: BR-11
          disposition: addressed
          note: parkWorker.onWait hook (read under mu) replaces the 20-30ms sleeps; absence check is now deterministic.
          round: 4
        - id: BR-12
          disposition: addressed
          note: holdWait cancellation now names the thread; GetThread and controller-unavailable errors in leaveOne remain untagged (minor; wrap once in the fan-out goroutine).
          round: 4
      findings:
        - id: BR-13
          severity: Minor
          title: Leave cancellation test name still claims started threads finish
          detail: '2nd in family. Rule: a cancellation-policy claim in a name, comment or atlas line must state what the test asserts. Corrected in park.go:196-200 but not in the leave_test.go:177 name. Rename it, and grep this issue''s tests and comments for "finish" against cancellation.'
          family: doc-claim-exceeds-mechanism
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-10-07T13:25:27-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: couch.go:880 returns false on zero incarnations; TestAnAbortOnARetiredThreadLeavesTheSessionAlone (threadgate_couch_test.go:408) reaches that branch.
          round: 5
        - id: BR-7
          disposition: addressed
          note: exitMarks is now a refcount map (console_continuation.go:37); park's re-mark at console.go:1898 increments; TestARefusalNeverRemovesAnotherOperationsExitMark covers both orderings.
          round: 5
        - id: BR-8
          disposition: addressed
          note: withoutDead's doc comment sits directly above it at couch.go:1316-1321; registry() has its own comment.
          round: 5
        - id: BR-13
          disposition: addressed
          note: Renamed TestLeaveCancelledMidFanOutStartsNoFurtherThread (leave_test.go:177); its comment says started threads stop at their own safe points, matching park.go:196-200.
          round: 5
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#205 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T00:26:37-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `gate-bypass-check-then-act` Couch.Park's open-transaction bypass can begin a fresh park without holding the gate
  Couch.Park reads Park!=nil and skips the gate. If the transaction closes before PairLifecycle.Park re-reads the record, normal mode mints a new nonce and parks while another holder, such as a relaunch now resuming, holds the thread. Route normal to Retry on the bypass path and add a test.
- **BR-2** [Important] `shared-state-writer-without-lock` AbortStarted now writes c.reg from a GoTracked goroutine with no registry lock
  Before this change it was serialized with Forget (on the console goroutine) and with other aborts. Now concurrent aborts, or an abort and a Forget, can lose a registry update in memory and on disk. regMu (Task 8b) is deferred to M2. Pull it forward, or record that M1 widens the race.
- **BR-3** [Important] `cleanup-scope-wider-than-own-effect` A busy continue-thread clears expected-exit marks that other operations set on the thread's panes
  console_continuation.go deletes expectedExits for every pane on the thread. retry-continuation (console.go:1631) and park/detach success (console.go:1887) also set these marks, so a deliberate exit gets reported as unexpected. Track the IDs this enqueue marked and drop only those.
- **BR-4** [Important] `interleaving-cell-untested` The wait paths of RecoverActiveParks and AbortStarted have no test
  The plan named TestLeaveAndParkRecoveryWaitForAHolder; only Leave is tested. Add tests that RecoverActiveParks waits and then recovers, that AbortStarted waits and then quiesces on an identity match, and that a cancelled abort wait ends only the helper and terminal.
- **BR-5** [Minor] `absence-read-as-ownership` startStillOwnsThread returns true for a record with zero incarnations
  No incarnation is not evidence that this start still owns the thread. Require at least one matching incarnation before the address-scoped quiesce.
- **BR-6** [Minor] `test-asserts-any-error` TestACancelledParkStillRefusesOtherLifecycleOperations asserts only err != nil
  Assert the refusal cause (open park transaction, or "another park transaction") so an unrelated failure cannot pass the test.

## Round 2 — 2026-10-07T01:07:28-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — park.go:235 maps begin-capable modes to retry on the join path; controller Park already retries an open txn, so only the stale window changes (now refuses). Pure mapping test only; call site has no seam.
- BR-2 — addressed — All c.reg/c.names access routed via registry()/mutateRegistry (grep-verified); TestConcurrentRegistryWritersLoseNoUpdate passes under -race.
- BR-3 — addressed — watch.marked records own marks; TestABusyContinuationIsRetriedSilently goes red if the finish path reverts to delete-all.
- BR-4 — addressed — TestParkRecoveryWaitsForAHolderThenRecovers, TestAbortStartedWaitsForAHolderThenQuiesces, TestACancelledAbortWaitEndsOnlyItsOwnHelper added and passing.
- BR-5 — not-addressed — Code fix at couch.go startStillOwnsThread is correct, but no regression test covers the zero-incarnation record.
- BR-6 — addressed — Asserts ResumeDiagnosticOf, "open park transaction", and "another park transaction" causes.

### Raised

- **BR-7** [Minor] `cleanup-scope-wider-than-own-effect` Expected-exit marks have first-come ownership; a late busy refusal can still delete a park's re-mark
  This is the 2nd finding in family cleanup-scope-wider-than-own-effect. Rule: a shared expectedExits mark must record its owners (pane id -> set of operation ids, or a refcount), and an undo removes only its own ownership. Current state: markThreadExitsLocked skips already-marked panes, but park/detach success (console.go:1883) re-marks without recording an owner, so a continuation refusal processed after it deletes the park's mark. Only that ordering is affected, and it is unlikely.
- **BR-8** [Minor] `doc-comment-detached-by-insertion` withoutDead's doc comment now heads registry() at couch.go:1282
  The registry helpers were inserted between the comment and its function; move the comment back above withoutDead.

## Round 3 — 2026-10-07T12:48:38-07:00 (claude) — BLOCKED

### Raised

- **BR-9** [Important] `interleaving-cell-untested` Concurrency cells that exist only above bound 1 are untested; couchtty pins LifecycleParallelism=1 package-wide
  2nd finding in family interleaving-cell-untested. Rule: each declared M2 concurrency policy needs a test that runs at bound > 1 with controlled ordering; pinning the global bound to 1 covers none of them. Missing: Task 8 TestLeaveParkAlongsideAnotherParkNeverFails; Leave cancellation partway through the fan-out (stops starting, started finish); Task 10 Step 3 console-path tests under N workers (burst of reattaches, late abort vs relaunch); quit mid-pass at Limit>1 covered only in the reducer. Fix: add the couchcore tests with withParallelism and one console test with the bound raised, or record a Revisions entry saying why reducer coverage suffices.
- **BR-10** [Minor] `doc-claim-exceeds-mechanism` queueWorkers comment and atlas say an operator gesture never waits behind pass attempts; remote and continuation jobs share the spare worker
  console.go:180-183 and atlas/couch.md. Narrow the wording to: the pass alone cannot occupy every worker.
- **BR-11** [Minor] `test-asserts-absence-by-sleep` Park worker capacity tests assert absence and wait with 20-30ms sleeps
  TestParkWorkerCapacityWaitHonoursContext passes without exercising the wait if cancel fires before Submit blocks. Expose a waiting hook or signal so the test synchronizes instead of sleeping.
- **BR-12** [Minor] `joined-error-loses-subject` Leave joins one untagged context error per cancelled thread
  When holdWait is cancelled, the joined error repeats "context canceled" N times without naming the threads. Wrap each error with the thread tag in the fan-out goroutine.

## Round 4 — 2026-10-07T12:58:10-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — Two bound>1 tests added (console Stop mid-pass at bound 3; Leave cancel mid-fan-out at bound 2), both pass under -race; remaining cells justified in issue Log (should also be a plan Revisions entry).
- BR-10 — addressed — console.go:180-183 and atlas/couch.md now say "the pass alone" and name the shared spare worker.
- BR-11 — addressed — parkWorker.onWait hook (read under mu) replaces the 20-30ms sleeps; absence check is now deterministic.
- BR-12 — addressed — holdWait cancellation now names the thread; GetThread and controller-unavailable errors in leaveOne remain untagged (minor; wrap once in the fan-out goroutine).

### Raised

- **BR-13** [Minor] `doc-claim-exceeds-mechanism` Leave cancellation test name still claims started threads finish
  2nd in family. Rule: a cancellation-policy claim in a name, comment or atlas line must state what the test asserts. Corrected in park.go:196-200 but not in the leave_test.go:177 name. Rename it, and grep this issue's tests and comments for "finish" against cancellation.

## Round 5 — 2026-10-07T13:25:27-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — couch.go:880 returns false on zero incarnations; TestAnAbortOnARetiredThreadLeavesTheSessionAlone (threadgate_couch_test.go:408) reaches that branch.
- BR-7 — addressed — exitMarks is now a refcount map (console_continuation.go:37); park's re-mark at console.go:1898 increments; TestARefusalNeverRemovesAnotherOperationsExitMark covers both orderings.
- BR-8 — addressed — withoutDead's doc comment sits directly above it at couch.go:1316-1321; registry() has its own comment.
- BR-13 — addressed — Renamed TestLeaveCancelledMidFanOutStartsNoFurtherThread (leave_test.go:177); its comment says started threads stop at their own safe points, matching park.go:196-200.

## Open findings

(none — every finding has been disposed)
