---
gate: boundary-review
issue: 854
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f6945bc9-1a53-42a9-9bdd-290251b7e2e0
            repository: github.com/xianxu/pair
            issue_id: "000854"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/cf2eb91c631f1824
            reviewed_head: c4d4ed487292cadb4bb629d42381ae25406a8468
            slot: pair:6
            requested_at: "2026-10-24T14:51:13.063152Z"
          phase: finished
          pid: 77672
          started_at: "2026-10-24T14:51:13.213048Z"
          finished_at: "2026-10-24T14:53:04.489674Z"
          session_id: 014b125f-6f0e-4a34-a30a-a4fc54cc14aa
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T07:53:10-07:00"
      dispatched: "2026-10-24T07:51:09-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: 'The #827 lock-semantics doc comment stays on the withStoreLockChecked wrapper; withStoreLockBounded has none'
          detail: threadstore.go:179-183 describes the check and wait semantics now carried out in withStoreLockBounded (threadstore.go:208), which has no doc comment. Move or extend the comment so it covers the new function and the storeWait deadline flag. This is the only instance in the window.
          family: doc-comment-follows-logic
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: c4d4ed487292cadb4bb629d42381ae25406a8468
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2aad0b9e-9e8d-4fd7-94ec-b1a1391f88e7
            repository: github.com/xianxu/pair
            issue_id: "000854"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/68b7b7a9fc6fc50e
            reviewed_head: dc92f440ace7f8b6546de1eb8ba51f4cfd2a3f67
            slot: pair:6
            requested_at: "2026-10-25T03:44:51.823093Z"
          phase: finished
          pid: 34418
          started_at: "2026-10-25T03:44:51.970027Z"
          finished_at: "2026-10-25T03:47:43.312101Z"
          session_id: 16e8cf29-02b0-4f1d-894f-d144a5bea0e4
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T20:47:46-07:00"
      dispatched: "2026-10-24T20:44:43-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'At head, withStoreLockBounded (threadstore.go:208) still has no doc comment; the #827 comment remains on the wrapper at 179-183.'
          round: 2
      findings:
        - id: BR-2
          severity: Important
          title: No deterministic test pins the storeLockBound wiring at withUpdateLockContext, the root-cause site
          detail: 'Mutation run: reverting park_recovery_store.go:27 and :30 to withStoreLockChecked(run, ..., storeReadLockWait) leaves TestStoreLockWait*, TestParkRecovery* and TestRepoConfigurationPublicationBoundsCompetingWriter all passing; the unit tests call withStoreLockBounded directly. Family instances in window: park_recovery_store.go:27, :30; repopreferences_store.go:102 (capped DeadlineExceeded before ctx.Err join, untested); startup_creation.go:62-63 (Revisions admit mutation-passing). Fix: a test ctx whose Deadline is near but Err returns nil, driving withUpdateLockContext (expect DeadlineExceeded before storeReadLockWait) and rememberRepoConfigurationContext (expect the store-stayed-busy join).'
          family: regression-test-pins-consumer-wiring
          round: 2
        - id: BR-3
          severity: Minor
          title: Preview reads (retentionReadLock) keep a fixed storeReadLockWait independent of the caller deadline
          detail: retention.go:330-345 polls for a fixed wait and returns ErrThreadStoreBusy; threadstore_preview.go:82 joins ctx.Err, which may not yet be set when the deadlines coincide, the same race this issue fixed for writes. Only remaining fixed-wait site in the tree; atlas scopes the guarantee to coordinated writes, so this is a note.
          family: store-lock-wait-bounded-by-caller-deadline
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#854 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T07:53:10-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `doc-comment-follows-logic` (kind: defect) The #827 lock-semantics doc comment stays on the withStoreLockChecked wrapper; withStoreLockBounded has none
  threadstore.go:179-183 describes the check and wait semantics now carried out in withStoreLockBounded (threadstore.go:208), which has no doc comment. Move or extend the comment so it covers the new function and the storeWait deadline flag. This is the only instance in the window.

## Round 2 — 2026-10-24T20:47:46-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — At head, withStoreLockBounded (threadstore.go:208) still has no doc comment; the #827 comment remains on the wrapper at 179-183.

### Raised

- **BR-2** [Important] `regression-test-pins-consumer-wiring` No deterministic test pins the storeLockBound wiring at withUpdateLockContext, the root-cause site
  Mutation run: reverting park_recovery_store.go:27 and :30 to withStoreLockChecked(run, ..., storeReadLockWait) leaves TestStoreLockWait*, TestParkRecovery* and TestRepoConfigurationPublicationBoundsCompetingWriter all passing; the unit tests call withStoreLockBounded directly. Family instances in window: park_recovery_store.go:27, :30; repopreferences_store.go:102 (capped DeadlineExceeded before ctx.Err join, untested); startup_creation.go:62-63 (Revisions admit mutation-passing). Fix: a test ctx whose Deadline is near but Err returns nil, driving withUpdateLockContext (expect DeadlineExceeded before storeReadLockWait) and rememberRepoConfigurationContext (expect the store-stayed-busy join).
- **BR-3** [Minor] `store-lock-wait-bounded-by-caller-deadline` Preview reads (retentionReadLock) keep a fixed storeReadLockWait independent of the caller deadline
  retention.go:330-345 polls for a fixed wait and returns ErrThreadStoreBusy; threadstore_preview.go:82 joins ctx.Err, which may not yet be set when the deadlines coincide, the same race this issue fixed for writes. Only remaining fixed-wait site in the tree; atlas scopes the guarantee to coordinated writes, so this is a note.

## Open findings

- **BR-1** [Minor] `doc-comment-follows-logic` (kind: defect) The #827 lock-semantics doc comment stays on the withStoreLockChecked wrapper; withStoreLockBounded has none
- **BR-2** [Important] `regression-test-pins-consumer-wiring` No deterministic test pins the storeLockBound wiring at withUpdateLockContext, the root-cause site
- **BR-3** [Minor] `store-lock-wait-bounded-by-caller-deadline` Preview reads (retentionReadLock) keep a fixed storeReadLockWait independent of the caller deadline
