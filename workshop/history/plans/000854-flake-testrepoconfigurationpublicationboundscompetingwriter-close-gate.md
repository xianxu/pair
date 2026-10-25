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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: bce650d8-6dd6-4572-9f9f-23f179c09ebc
            repository: github.com/xianxu/pair
            issue_id: "000854"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/7565fbdf6fca53d1
            reviewed_head: 68c48a465da53014a25f71e274deea6ce927dace
            slot: pair:6
            requested_at: "2026-10-25T05:11:03.490959Z"
          phase: finished
          pid: 79969
          started_at: "2026-10-25T05:11:03.647318Z"
          finished_at: "2026-10-25T05:20:13.777745Z"
          session_id: a48812a5-96a1-43b5-af92-d004e9b47332
          outcome: returned
      "n": 3
      timestamp: "2026-10-24T22:20:15-07:00"
      dispatched: "2026-10-24T22:10:59-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'The #827 comment now sits above withStoreLockBounded (threadstore.go:202-207) and names storeLockBound; withStoreLockChecked has its own one-line comment.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'Mutation-checked here: reverting park_recovery_store.go:27 fails the coordinated subtest, :30 fails the plain subtest, and restoring the old join at repopreferences_store.go:102 fails TestRepoPreferencePublicationJoinsBusyAtACapOnlyDeadline.'
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: retention.go:343 still returns busy on a coincident expiry while ctx.Err can be nil (threadstore_preview.go:82); scoped out in atlas and Revisions, stays a Minor note.
          round: 3
      findings:
        - kind: defect
          id: BR-4
          severity: Minor
          title: Cap-only publication wiring test hangs instead of failing when the park-recovery cap regresses
          detail: 'With storeLockBound reverted at park_recovery_store.go:30, TestRepoPreferencePublicationJoinsBusyAtACapOnlyDeadline (store_lock_deadline_wiring_test.go:68) loops forever, because the nil Done of deadlineOnlyContext never fires waitForStoreLockContext (storelock_unix.go:85). Observed: the run hung past 400s. Only instance in the window (the other deadlineOnly tests are single calls). Fix: close Done at deadline plus a grace period, or guard publishSecond with a test-level timer.'
          family: regression-test-fails-bounded
          round: 3
        - kind: unverifiable
          id: BR-5
          severity: Minor
          title: Empty-startup deadline tests could not run in the review sandbox
          detail: 'Ran: go test ./cmd/internal/couchcore -run TestEmptyStartupLockWait -count=1 (also with TMPDIR moved into the checkout). Both tests fail at slotrecovery_test.go:124 with "git init refused ... outside the reviewer''s directories (sdlc #339)". Evidence that would answer it: a host run showing both pass at 68c48a465 and that reverting startup_creation.go:62-63 to the fixed wait turns TestEmptyStartupLockWaitEndsAtACapOnlyDeadline red.'
          family: regression-test-pins-consumer-wiring
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: 68c48a465da53014a25f71e274deea6ce927dace
      blocked: false
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

## Round 3 — 2026-10-24T22:20:15-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — The #827 comment now sits above withStoreLockBounded (threadstore.go:202-207) and names storeLockBound; withStoreLockChecked has its own one-line comment.
- BR-2 — addressed — Mutation-checked here: reverting park_recovery_store.go:27 fails the coordinated subtest, :30 fails the plain subtest, and restoring the old join at repopreferences_store.go:102 fails TestRepoPreferencePublicationJoinsBusyAtACapOnlyDeadline.
- BR-3 — not-addressed — retention.go:343 still returns busy on a coincident expiry while ctx.Err can be nil (threadstore_preview.go:82); scoped out in atlas and Revisions, stays a Minor note.

### Raised

- **BR-4** [Minor] `regression-test-fails-bounded` (kind: defect) Cap-only publication wiring test hangs instead of failing when the park-recovery cap regresses
  With storeLockBound reverted at park_recovery_store.go:30, TestRepoPreferencePublicationJoinsBusyAtACapOnlyDeadline (store_lock_deadline_wiring_test.go:68) loops forever, because the nil Done of deadlineOnlyContext never fires waitForStoreLockContext (storelock_unix.go:85). Observed: the run hung past 400s. Only instance in the window (the other deadlineOnly tests are single calls). Fix: close Done at deadline plus a grace period, or guard publishSecond with a test-level timer.
- **BR-5** [Minor] `regression-test-pins-consumer-wiring` (kind: unverifiable) Empty-startup deadline tests could not run in the review sandbox
  Ran: go test ./cmd/internal/couchcore -run TestEmptyStartupLockWait -count=1 (also with TMPDIR moved into the checkout). Both tests fail at slotrecovery_test.go:124 with "git init refused ... outside the reviewer's directories (sdlc #339)". Evidence that would answer it: a host run showing both pass at 68c48a465 and that reverting startup_creation.go:62-63 to the fixed wait turns TestEmptyStartupLockWaitEndsAtACapOnlyDeadline red.

## Open findings

- **BR-3** [Minor] `store-lock-wait-bounded-by-caller-deadline` Preview reads (retentionReadLock) keep a fixed storeReadLockWait independent of the caller deadline
- **BR-4** [Minor] `regression-test-fails-bounded` (kind: defect) Cap-only publication wiring test hangs instead of failing when the park-recovery cap regresses
- **BR-5** [Minor] `regression-test-pins-consumer-wiring` (kind: unverifiable) Empty-startup deadline tests could not run in the review sandbox
