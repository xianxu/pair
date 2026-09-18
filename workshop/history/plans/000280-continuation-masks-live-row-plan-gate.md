---
gate: plan-quality
issue: 280
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-17T22:02:18-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Plan keeps the Failed row's action set because the guard refuses lifecycle ops, but park and detach are not guarded
          detail: 'continuationGuard gates only relaunch.go:90, cold resume resume.go:453, switchagent.go:128 and start claims threadstore.go:500-503. park.go never reads Continuation, and detach.go reads it only for archive. TestActionOfferedImpliesPermitted never sets Continuation and its permission checks see only state and reason. Decide detach and park on how they are actually admitted: warm reattach through validateContinuationWarm may refuse a target session, which would be a real reason to hide detach. Fix the rationale, and pin the choice in a menuActionItems test. The issue Log names this action set as part of the problem it is fixing.'
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-2
          severity: Important
          title: The retry fix has no switcher-side regression test on the production executor
          detail: The red and green test is a couchcore test with hand-copied switcher arguments, and TestContinuationFailedRowKeepsAnExplicitRetry still uses a fake LiveOwner, so a change that puts ref back in dispatchMenuOperation stays green. Switch that existing couchtty test to CouchLiveOwnerExecutor on a temp-dir store; operation_queue_test.go already does this. The plan does it for dismiss but not for retry.
          family: test-crosses-production-seam
          round: 1
        - id: PQ-3
          severity: Minor
          title: DismissFailedContinuation takes no revision, and stale-revision handling is unstated
          detail: 'updateExistingThread needs an expected revision (threadstore.go:293,313). Follow the sibling pattern: a store method that takes a revision, with requestRecord (continuation.go:140-152) and advanceContinuation''s retry-on-stale loop in the caller. Reuse the existing no-request and obsolete-request messages instead of writing new wording.'
          family: reuse-existing-transition-pattern
          round: 1
        - id: PQ-4
          severity: Minor
          title: The ARCH-FUNERAL line leaves out couch's own checkpoint copy in the store's continuation directory
          detail: continuation_store.go:92 writes store-root/continuation/scope/tag.md. It survives a dismissal and is removed only at archive (threadstore.go:1211-1214) or overwritten by the next continuation. It is bounded at one per thread; the plan should say so.
          family: artifact-lifecycle-unnamed
          round: 1
        - id: PQ-5
          severity: Minor
          title: The checkpoint package will still read as if retry were the only exit from Failed
          detail: 'Dismissal lives only in couchcore, while checkpoint/request.go Advance defines the request lifecycle. Add a predicate or a comment there. Also cover a new publish after a dismissal: PublishContinuation''s generation check only runs when an old request exists, so the only remaining guard against a stale publish is sameContinuationSource (continuation.go:105).'
          family: state-machine-legibility
          round: 1
        - id: PQ-6
          severity: Minor
          title: Optional bootstrap ref makes CLI retry with no argument fail after the supervisor lease, not at argument binding
          detail: bindArgs used to return exit 2 before anything else ran. Now dispatch fails with "thread reference is required" after operationOwnsLive has taken the lease (run.go:277-284). Accept this knowingly or pin it with a test.
          family: cli-error-ordering
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T22:06:33-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Park/detach kept on real grounds; verified a detached Failed row lands in the Recovery branch (recovery.go:109-116, menu.go:1212) which offers retry + dismiss.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: TestContinuationFailedRowKeepsAnExplicitRetry moves onto DispatchOperation + CouchLiveOwnerExecutor on a temp store; putting ref back turns it red.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Store method takes expectedRevision; caller uses requestRecord (empty id = retained) and the stale-retry loop; messages reused.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Checkpoint copy named, bounded at one per thread, overwritten by next publish, removed at archive.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Comment in checkpoint Advance plus a stale-publish-after-dismiss test against sameContinuationSource.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Accepted knowingly with the reason stated in the Fix-retry task.
          round: 2
      findings:
        - id: PQ-7
          severity: Minor
          title: Failed-row action composition filters only continuationGuard ops, but archive has its own continuation admission (archiveContinuationVacant)
          detail: '2nd finding in this family. Composing "state actions minus ContinuationRefuses" newly offers archive on unusable Failed rows with no Recovery (menu.go:1255-1258), which archiveContinuationVacant (detach.go:430-450) may refuse, and TestContinuationFailedRowKeepsAnExplicitRetry (console_continuation_test.go:21), the test reused for PQ-2, asserts archive is absent there. Rule: an action is offered on a row with a retained request only if EVERY admission that reads record.Continuation allows it (continuationGuard sites, archiveContinuationVacant, validateContinuationWarm). The ContinuationRefuses table test should loop over every declared RowAction op, not a hand list, or composition should apply to live rows only.'
          family: unbacked-existing-behavior-claim
          round: 2
      blocked: false
content_hash: c401576861dd2500d5898b614fdaeae34126b4bc038e6d406b260869f0cf21c5
---

# Gate ledger — pair#280 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T22:02:18-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unbacked-existing-behavior-claim` Plan keeps the Failed row's action set because the guard refuses lifecycle ops, but park and detach are not guarded
  continuationGuard gates only relaunch.go:90, cold resume resume.go:453, switchagent.go:128 and start claims threadstore.go:500-503. park.go never reads Continuation, and detach.go reads it only for archive. TestActionOfferedImpliesPermitted never sets Continuation and its permission checks see only state and reason. Decide detach and park on how they are actually admitted: warm reattach through validateContinuationWarm may refuse a target session, which would be a real reason to hide detach. Fix the rationale, and pin the choice in a menuActionItems test. The issue Log names this action set as part of the problem it is fixing.
- **PQ-2** [Important] `test-crosses-production-seam` The retry fix has no switcher-side regression test on the production executor
  The red and green test is a couchcore test with hand-copied switcher arguments, and TestContinuationFailedRowKeepsAnExplicitRetry still uses a fake LiveOwner, so a change that puts ref back in dispatchMenuOperation stays green. Switch that existing couchtty test to CouchLiveOwnerExecutor on a temp-dir store; operation_queue_test.go already does this. The plan does it for dismiss but not for retry.
- **PQ-3** [Minor] `reuse-existing-transition-pattern` DismissFailedContinuation takes no revision, and stale-revision handling is unstated
  updateExistingThread needs an expected revision (threadstore.go:293,313). Follow the sibling pattern: a store method that takes a revision, with requestRecord (continuation.go:140-152) and advanceContinuation's retry-on-stale loop in the caller. Reuse the existing no-request and obsolete-request messages instead of writing new wording.
- **PQ-4** [Minor] `artifact-lifecycle-unnamed` The ARCH-FUNERAL line leaves out couch's own checkpoint copy in the store's continuation directory
  continuation_store.go:92 writes store-root/continuation/scope/tag.md. It survives a dismissal and is removed only at archive (threadstore.go:1211-1214) or overwritten by the next continuation. It is bounded at one per thread; the plan should say so.
- **PQ-5** [Minor] `state-machine-legibility` The checkpoint package will still read as if retry were the only exit from Failed
  Dismissal lives only in couchcore, while checkpoint/request.go Advance defines the request lifecycle. Add a predicate or a comment there. Also cover a new publish after a dismissal: PublishContinuation's generation check only runs when an old request exists, so the only remaining guard against a stale publish is sameContinuationSource (continuation.go:105).
- **PQ-6** [Minor] `cli-error-ordering` Optional bootstrap ref makes CLI retry with no argument fail after the supervisor lease, not at argument binding
  bindArgs used to return exit 2 before anything else ran. Now dispatch fails with "thread reference is required" after operationOwnsLive has taken the lease (run.go:277-284). Accept this knowingly or pin it with a test.

## Round 2 — 2026-09-17T22:06:33-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Park/detach kept on real grounds; verified a detached Failed row lands in the Recovery branch (recovery.go:109-116, menu.go:1212) which offers retry + dismiss.
- PQ-2 — addressed — TestContinuationFailedRowKeepsAnExplicitRetry moves onto DispatchOperation + CouchLiveOwnerExecutor on a temp store; putting ref back turns it red.
- PQ-3 — addressed — Store method takes expectedRevision; caller uses requestRecord (empty id = retained) and the stale-retry loop; messages reused.
- PQ-4 — addressed — Checkpoint copy named, bounded at one per thread, overwritten by next publish, removed at archive.
- PQ-5 — addressed — Comment in checkpoint Advance plus a stale-publish-after-dismiss test against sameContinuationSource.
- PQ-6 — addressed — Accepted knowingly with the reason stated in the Fix-retry task.

### Raised

- **PQ-7** [Minor] `unbacked-existing-behavior-claim` Failed-row action composition filters only continuationGuard ops, but archive has its own continuation admission (archiveContinuationVacant)
  2nd finding in this family. Composing "state actions minus ContinuationRefuses" newly offers archive on unusable Failed rows with no Recovery (menu.go:1255-1258), which archiveContinuationVacant (detach.go:430-450) may refuse, and TestContinuationFailedRowKeepsAnExplicitRetry (console_continuation_test.go:21), the test reused for PQ-2, asserts archive is absent there. Rule: an action is offered on a row with a retained request only if EVERY admission that reads record.Continuation allows it (continuationGuard sites, archiveContinuationVacant, validateContinuationWarm). The ContinuationRefuses table test should loop over every declared RowAction op, not a hand list, or composition should apply to live rows only.

## Open findings

- **PQ-7** [Minor] `unbacked-existing-behavior-claim` Failed-row action composition filters only continuationGuard ops, but archive has its own continuation admission (archiveContinuationVacant)
