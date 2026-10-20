---
gate: boundary-review
issue: 712
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 494fdc62-28cc-4f1f-b196-f39c16281080
            repository: github.com/xianxu/pair
            issue_id: "000712"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9f2475c8d40546c8
            reviewed_head: 4f846ccf2248cd84b6dca6324925c8af56b4e526
            slot: pair:3
            requested_at: "2026-10-19T21:37:08.460559Z"
          phase: finished
          pid: 76143
          started_at: "2026-10-19T21:37:08.778602Z"
          finished_at: "2026-10-19T21:39:22.024624Z"
          session_id: 01a15618-d100-79e0-ab04-fc1d3144413c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4f846ccf2248cd84b6dca6324925c8af56b4e526
        tracker_head: 180f9292e2092f6a03d69e91836186c844c49c74
        branch_point: a040b13bab7015c399ff6c9a81c1b4c02d7d7b24
        since: "2026-10-19T14:08:10-07:00"
        until: "2026-10-19T14:36:23-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T14:08:10-07:00"
              end: "2026-10-19T14:36:23-07:00"
              began: "2026-10-19T14:08:10-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: f1a808a8cf2badd0de0b724d29e2fe1a34f590a6da932f3d23c25e46f999e157
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 114069098
              windows:
                - start: "2026-10-19T14:08:10-07:00"
                  end: "2026-10-19T14:36:23-07:00"
            - harness: codex
              id: 01a155ff-314d-71f3-a112-16b0145ba63c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 6cbb3203f3ac384bbf131d51200dd9d253b2ba20ccd8c3f0604eb7056cafc497
              started: "2026-10-19T21:09:10.353Z"
              spawned: true
              bytes: 555160
              windows:
                - start: "2026-10-19T21:09:10.353Z"
                  end: "2026-10-19T14:36:23-07:00"
            - harness: codex
              id: 01a15605-a09f-7443-9874-02a444fc8897
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 1573d22646773221c23a2bd1ee526cb010e7eb9fb48cc708e801b5ea6fafe7a2
              started: "2026-10-19T21:16:12.657Z"
              spawned: false
              bytes: 760126
              windows:
                - start: "2026-10-19T21:16:12.657Z"
                  end: "2026-10-19T14:36:23-07:00"
            - harness: codex
              id: 01a15607-0a47-7402-914b-31174d48250f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d29845f3c714dab4df233efa17c447e01fb7c1e011396410195e129fcac76158
              started: "2026-10-19T21:17:45.313Z"
              spawned: false
              bytes: 408856
              windows:
                - start: "2026-10-19T21:17:45.313Z"
                  end: "2026-10-19T14:36:23-07:00"
            - harness: codex
              id: 01a15608-506a-71c2-b037-42686d90bdf5
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 83764c44c4aadf81f9e548968b32e02b99fdb466ed9dfea44bf851055098ccb0
              started: "2026-10-19T21:19:08.87Z"
              spawned: false
              bytes: 411232
              windows:
                - start: "2026-10-19T21:19:08.87Z"
                  end: "2026-10-19T14:36:23-07:00"
            - harness: codex
              id: 01a15608-a15d-7c61-ae43-a9bf7ccf8f1a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d93a989b3a85a9dfcf680c08986f9aeb829c3d405a7ed4e8d7aa1b66c3f77d8f
              started: "2026-10-19T21:19:29.506Z"
              spawned: false
              bytes: 379423
              windows:
                - start: "2026-10-19T21:19:29.506Z"
                  end: "2026-10-19T14:36:23-07:00"
            - harness: codex
              id: 01a15609-5573-7bf1-9aca-51904d272b9a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: d00054311e33386b0434354e97f26ad96a61dca0615f2b9f6d4fbed3b8a47b0c
              started: "2026-10-19T21:20:14.986Z"
              spawned: true
              bytes: 911162
              windows:
                - start: "2026-10-19T21:20:14.986Z"
                  end: "2026-10-19T14:36:23-07:00"
      "n": 1
      timestamp: "2026-10-19T14:39:23-07:00"
      dispatched: "2026-10-19T14:37:05-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Thread-store reads can block beyond the session cleanup deadline
          detail: 'ARCH-CONSTRAINTS / ARCH-PURPOSE: cmd/internal/couchcore/couch.go:1004 calls quiesceThreadSession, whose recovery_execute.go:86 GetThread uses context.Background and blocking store locking (threadstore.go:299,182; storelock_unix.go:24-28). The new timeout branch at couch.go:1001 performs another blocking GetThread after expiration. A concurrent store-lock holder can therefore prevent this session phase from returning indefinitely. Carry the shared deadline through all session-phase metadata reads, retain the exact session identity without rereading after expiration, and add contention regressions proving return before the held lock is released.'
          family: cleanup-budget-covers-all-io
          round: 1
        - kind: unverifiable
          id: BR-2
          severity: Important
          title: Composed cleanup and loaded completion could not be verified
          detail: 'ARCH-MOCK / ARCH-CONSTRAINTS: Running go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run ''Test(UnknownCleanup|SessionCleanup|PersistentSessionCleanup|PostAckSessionCleanup|DecideStartCleanup|SpawnCompos|RegistrationFixture|RegistrationTerminalFixture|LifecycleFailureReceipt)'' -count=1 -timeout=180s failed both ordinary/fresh-slot composed families before readiness with “couch session repository scope changed.” Review TMPDIR is inside the checkout, while testpair.Environment clears GIT_CEILING_DIRECTORIES, allowing discovery of the outer repository. Provide pinned-head host evidence that these cases reach cleanup and pass 20/20 under the documented owned load, or make the fixture work within review confinement and rerun.'
          family: boundary-verification-reaches-target
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a3e2ab98-5f8f-4cce-a744-351173dd134d
            repository: github.com/xianxu/pair
            issue_id: "000712"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/2965270817b1d2d8
            reviewed_head: 328f187bc9ca6e3aad6a2b6b6e0465e0c0b5debd
            slot: pair:3
            requested_at: "2026-10-19T22:13:06.041356Z"
          phase: finished
          pid: 50639
          started_at: "2026-10-19T22:13:06.238772Z"
          finished_at: "2026-10-19T22:16:21.529288Z"
          session_id: 01a15639-bc18-7de2-987d-d0fa17a9335b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 328f187bc9ca6e3aad6a2b6b6e0465e0c0b5debd
        tracker_head: 180f9292e2092f6a03d69e91836186c844c49c74
        branch_point: a040b13bab7015c399ff6c9a81c1b4c02d7d7b24
        since: "2026-10-19T14:08:10-07:00"
        until: "2026-10-19T15:12:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T14:08:10-07:00"
              end: "2026-10-19T15:12:26-07:00"
              began: "2026-10-19T14:08:10-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 12f8a3c19ee1dccbccef65c70de3e8e98951b467342b2bc2a22781ccbfbf829d
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 115443039
              windows:
                - start: "2026-10-19T14:08:10-07:00"
                  end: "2026-10-19T15:12:26-07:00"
            - harness: codex
              id: 01a155ff-314d-71f3-a112-16b0145ba63c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 6cbb3203f3ac384bbf131d51200dd9d253b2ba20ccd8c3f0604eb7056cafc497
              started: "2026-10-19T21:09:10.353Z"
              spawned: true
              bytes: 555160
              windows:
                - start: "2026-10-19T21:09:10.353Z"
                  end: "2026-10-19T15:12:26-07:00"
            - harness: codex
              id: 01a15605-a09f-7443-9874-02a444fc8897
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 1573d22646773221c23a2bd1ee526cb010e7eb9fb48cc708e801b5ea6fafe7a2
              started: "2026-10-19T21:16:12.657Z"
              spawned: false
              bytes: 760126
              windows:
                - start: "2026-10-19T21:16:12.657Z"
                  end: "2026-10-19T15:12:26-07:00"
            - harness: codex
              id: 01a15607-0a47-7402-914b-31174d48250f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d29845f3c714dab4df233efa17c447e01fb7c1e011396410195e129fcac76158
              started: "2026-10-19T21:17:45.313Z"
              spawned: false
              bytes: 408856
              windows:
                - start: "2026-10-19T21:17:45.313Z"
                  end: "2026-10-19T15:12:26-07:00"
            - harness: codex
              id: 01a15608-506a-71c2-b037-42686d90bdf5
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 83764c44c4aadf81f9e548968b32e02b99fdb466ed9dfea44bf851055098ccb0
              started: "2026-10-19T21:19:08.87Z"
              spawned: false
              bytes: 411232
              windows:
                - start: "2026-10-19T21:19:08.87Z"
                  end: "2026-10-19T15:12:26-07:00"
            - harness: codex
              id: 01a15608-a15d-7c61-ae43-a9bf7ccf8f1a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d93a989b3a85a9dfcf680c08986f9aeb829c3d405a7ed4e8d7aa1b66c3f77d8f
              started: "2026-10-19T21:19:29.506Z"
              spawned: false
              bytes: 379423
              windows:
                - start: "2026-10-19T21:19:29.506Z"
                  end: "2026-10-19T15:12:26-07:00"
            - harness: codex
              id: 01a15609-5573-7bf1-9aca-51904d272b9a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: da9a60f9c7ed7c967bd3cb2d093c536846e146f37d6523eb034b3c8fbf2106c9
              started: "2026-10-19T21:20:14.986Z"
              spawned: true
              bytes: 1352070
              windows:
                - start: "2026-10-19T21:20:14.986Z"
                  end: "2026-10-19T15:12:26-07:00"
        cuts:
            - dispatched: "2026-10-19T14:37:05-07:00"
              finalized: "2026-10-19T14:39:23-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-19T15:16:22-07:00"
      dispatched: "2026-10-19T15:13:03-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Contextual metadata reads, cached expiry identity, legacy-index propagation, and bounded disposition persistence are covered by contention regressions. Removing PreviewThreadContext through a temporary Go overlay makes TestSessionCleanupMetadataReadHonorsContextWhileStoreHeld fail after waiting for lock release; the pinned implementation passes.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The original focused command passes within checkout-nested TMPDIR. The pinned-head composed family passes 20/20 with 20 owned CPU workers in 106.456 seconds; all workers were killed and reaped. Removing the explicit Git ceiling makes the nested-checkout regression fail with repository-scope changes.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Minor
          title: Exact live-to-unknown transition logic is duplicated
          detail: 'ARCH-DRY: cmd/internal/couchcore/couch.go:960-995 repeats the identity matching, live-to-unknown mutation and revision-conflict loop in threadstore.go:1034. Extract a shared transition helper or contextual store operation while preserving the callers’ different already-unknown behavior.'
          family: single-incarnation-transition-authority
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 328f187bc9ca6e3aad6a2b6b6e0465e0c0b5debd
      blocked: false
---

# Gate ledger — pair#712 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T14:39:23-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `cleanup-budget-covers-all-io` (kind: defect) Thread-store reads can block beyond the session cleanup deadline
  ARCH-CONSTRAINTS / ARCH-PURPOSE: cmd/internal/couchcore/couch.go:1004 calls quiesceThreadSession, whose recovery_execute.go:86 GetThread uses context.Background and blocking store locking (threadstore.go:299,182; storelock_unix.go:24-28). The new timeout branch at couch.go:1001 performs another blocking GetThread after expiration. A concurrent store-lock holder can therefore prevent this session phase from returning indefinitely. Carry the shared deadline through all session-phase metadata reads, retain the exact session identity without rereading after expiration, and add contention regressions proving return before the held lock is released.
- **BR-2** [Important] `boundary-verification-reaches-target` (kind: unverifiable) Composed cleanup and loaded completion could not be verified
  ARCH-MOCK / ARCH-CONSTRAINTS: Running go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(UnknownCleanup|SessionCleanup|PersistentSessionCleanup|PostAckSessionCleanup|DecideStartCleanup|SpawnCompos|RegistrationFixture|RegistrationTerminalFixture|LifecycleFailureReceipt)' -count=1 -timeout=180s failed both ordinary/fresh-slot composed families before readiness with “couch session repository scope changed.” Review TMPDIR is inside the checkout, while testpair.Environment clears GIT_CEILING_DIRECTORIES, allowing discovery of the outer repository. Provide pinned-head host evidence that these cases reach cleanup and pass 20/20 under the documented owned load, or make the fixture work within review confinement and rerun.

## Round 2 — 2026-10-19T15:16:22-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Contextual metadata reads, cached expiry identity, legacy-index propagation, and bounded disposition persistence are covered by contention regressions. Removing PreviewThreadContext through a temporary Go overlay makes TestSessionCleanupMetadataReadHonorsContextWhileStoreHeld fail after waiting for lock release; the pinned implementation passes.
- BR-2 — addressed — The original focused command passes within checkout-nested TMPDIR. The pinned-head composed family passes 20/20 with 20 owned CPU workers in 106.456 seconds; all workers were killed and reaped. Removing the explicit Git ceiling makes the nested-checkout regression fail with repository-scope changes.

### Raised

- **BR-3** [Minor] `single-incarnation-transition-authority` (kind: defect) Exact live-to-unknown transition logic is duplicated
  ARCH-DRY: cmd/internal/couchcore/couch.go:960-995 repeats the identity matching, live-to-unknown mutation and revision-conflict loop in threadstore.go:1034. Extract a shared transition helper or contextual store operation while preserving the callers’ different already-unknown behavior.

## Open findings

- **BR-3** [Minor] `single-incarnation-transition-authority` (kind: defect) Exact live-to-unknown transition logic is duplicated
