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

## Open findings

- **BR-5** [Minor] `absence-read-as-ownership` startStillOwnsThread returns true for a record with zero incarnations
- **BR-7** [Minor] `cleanup-scope-wider-than-own-effect` Expected-exit marks have first-come ownership; a late busy refusal can still delete a park's re-mark
- **BR-8** [Minor] `doc-comment-detached-by-insertion` withoutDead's doc comment now heads registry() at couch.go:1282
