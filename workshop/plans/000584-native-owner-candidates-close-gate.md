---
gate: boundary-review
issue: 584
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 06df5d05-36f0-43a1-bcf1-389b396d4207
            repository: github.com/xianxu/pair
            issue_id: "000584"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/5020d8595e076223
            reviewed_head: 935fd8c6820b93349468454644e83ebbe693ab39
            slot: pair:2
            requested_at: "2026-10-16T00:30:42.69503Z"
          phase: finished
          pid: 83515
          started_at: "2026-10-16T00:30:42.868494Z"
          finished_at: "2026-10-16T00:33:02.679394Z"
          session_id: 01a1421e-47f2-7be3-8a5f-e9c14906927a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 935fd8c6820b93349468454644e83ebbe693ab39
        tracker_head: 6f59b1d966560e05c0bb3887995ce3fb23c436e5
        branch_point: 3c7f6cb43d05f889aaf852cb61f61ec4fb005051
        since: "2026-10-15T15:43:33-07:00"
        until: "2026-10-15T17:29:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T15:43:33-07:00"
              end: "2026-10-15T17:29:55-07:00"
              began: "2026-10-15T15:43:33-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 223a0b7d67f74e7c3d8b1ae76a8fd7ee274887be8d731274e09a8f6673e4cd3f
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 89822751
              windows:
                - start: "2026-10-15T15:43:33-07:00"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141c7-6682-75a3-907d-387a99328853
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: d50d61b699e0e9dbbf3be36b3555d6ef2c88b66683d7ac0e22f93df95f230ac6
              started: "2026-10-15T22:55:49.661Z"
              spawned: true
              bytes: 741748
              windows:
                - start: "2026-10-15T22:55:49.661Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141c7-b508-7400-b690-e5b2e3ff92b9
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d0641b29881f1f44dfe321c47b6bea44959e4d79cb826ccd01cba2456691cc7
              started: "2026-10-15T22:56:09.768Z"
              spawned: false
              bytes: 573817
              windows:
                - start: "2026-10-15T22:56:09.768Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141c9-7802-7800-93dc-5d1d24eb28ee
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 95457948e8582be118c68ecc83bfa371e35916d20334ac284831170647e6d252
              started: "2026-10-15T22:58:05.424Z"
              spawned: false
              bytes: 433329
              windows:
                - start: "2026-10-15T22:58:05.424Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141ca-1302-7900-8980-37ac52de6fc6
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 6a581a61953be83decd0f022b0591bc30e3eb39190a701ca684064e43ed9f48d
              started: "2026-10-15T22:58:44.756Z"
              spawned: true
              bytes: 818265
              windows:
                - start: "2026-10-15T22:58:44.756Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141ca-f8c0-7d12-993e-38793b4752b0
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a7d1b9d02b1dd7dfc8c9457e3e014cc48fc7ea22ca0b23c4577e9f9e209cfd39
              started: "2026-10-15T22:59:44.122Z"
              spawned: false
              bytes: 422299
              windows:
                - start: "2026-10-15T22:59:44.122Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141cb-5dab-7260-9824-228d126ea623
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 10b83f12dd2ec7a039291bb25c683eacbc0478ac6fc76cec60b37bfa7c389ae1
              started: "2026-10-15T23:00:09.526Z"
              spawned: false
              bytes: 401586
              windows:
                - start: "2026-10-15T23:00:09.526Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141e2-9277-7521-a6de-104056107b04
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 11ed964791c3925a76f1e0470cffcec097eaee038de2bb54091356a93abdf567
              started: "2026-10-15T23:25:30.288Z"
              spawned: true
              bytes: 844381
              windows:
                - start: "2026-10-15T23:25:30.288Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a141e8-7837-72d0-9a15-e012125d5244
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 00f816a54b6f3cd87f27d0f6991169293c65c57475c2f5ecbbbeed79a77ce50a
              started: "2026-10-15T23:31:56.736Z"
              spawned: true
              bytes: 489128
              windows:
                - start: "2026-10-15T23:31:56.736Z"
                  end: "2026-10-15T17:29:55-07:00"
            - harness: codex
              id: 01a14205-e8bd-73b1-bebc-a98907c012b7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 07c554d32e797cbd82b0240ab4c3e23e1543ceb95f8d849a4b5d6f7e1a92440f
              started: "2026-10-16T00:04:06.091Z"
              spawned: true
              bytes: 675091
              windows:
                - start: "2026-10-16T00:04:06.091Z"
                  end: "2026-10-15T17:29:55-07:00"
      "n": 1
      timestamp: "2026-10-15T17:33:11-07:00"
      dispatched: "2026-10-15T17:30:40-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Core concepts classifies IO-driven retry and ownership proof as PURE
          detail: workshop/plans/000584-native-owner-candidates-plan.md:15–16 lists observation retry and legacy source proof as pure entities. Their implementations invoke injected observation, metadata reads, and held-file probes (recovery_native_owner.go:74 and recovery_native_legacy.go:44); their tests require those fakes. ARCH-PURE and the Core concepts gate require promoting these orchestration entities to INTEGRATION and naming the actual deterministic helpers separately. Add a Revisions entry correcting the table.
          family: pure-integration-classification
          round: 1
        - id: BR-2
          severity: Important
          title: Writable-file ownership proof has no repeatable live lsof conformance check
          detail: 'cmd/internal/procutil/procutil.go:105 introduces an admission-critical external contract, but lsof_writable_test.go:14 replaces lsof with scripted output throughout its tests. No committed live check or recurring invocation exercises this contract against real descriptors; the issue''s manual observations do not provide ongoing drift detection. ARCH-MOCK: add an opt-in conformance test using temporary files opened read-only, writable, and then closed, call the production LsofWritableNames boundary, and connect it to the live-check workflow with a documented cadence.'
          family: external-dependency-conformance
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7889d3ca-4b86-4aea-b34b-2440c876d92f
            repository: github.com/xianxu/pair
            issue_id: "000584"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ac00987f608f6cd7
            reviewed_head: 44b10b71dd206c8f694724df6137f54b8c9f7cbc
            slot: pair:2
            requested_at: "2026-10-16T00:38:35.514066Z"
          phase: finished
          pid: 39249
          started_at: "2026-10-16T00:38:35.697515Z"
          finished_at: "2026-10-16T00:40:33.636236Z"
          session_id: 01a14225-7ee2-7aa1-973e-3577d784d6b8
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 44b10b71dd206c8f694724df6137f54b8c9f7cbc
        tracker_head: 6f59b1d966560e05c0bb3887995ce3fb23c436e5
        branch_point: 3c7f6cb43d05f889aaf852cb61f61ec4fb005051
        since: "2026-10-15T15:43:33-07:00"
        until: "2026-10-15T17:37:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T15:43:33-07:00"
              end: "2026-10-15T17:37:28-07:00"
              began: "2026-10-15T15:43:33-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 494fe0e4e9bc469b9c6959d81e68fbefe5c40b78fcaeb09b11cd0b142493af6e
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 90291533
              windows:
                - start: "2026-10-15T15:43:33-07:00"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141c7-6682-75a3-907d-387a99328853
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: d50d61b699e0e9dbbf3be36b3555d6ef2c88b66683d7ac0e22f93df95f230ac6
              started: "2026-10-15T22:55:49.661Z"
              spawned: true
              bytes: 741748
              windows:
                - start: "2026-10-15T22:55:49.661Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141c7-b508-7400-b690-e5b2e3ff92b9
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d0641b29881f1f44dfe321c47b6bea44959e4d79cb826ccd01cba2456691cc7
              started: "2026-10-15T22:56:09.768Z"
              spawned: false
              bytes: 573817
              windows:
                - start: "2026-10-15T22:56:09.768Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141c9-7802-7800-93dc-5d1d24eb28ee
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 95457948e8582be118c68ecc83bfa371e35916d20334ac284831170647e6d252
              started: "2026-10-15T22:58:05.424Z"
              spawned: false
              bytes: 433329
              windows:
                - start: "2026-10-15T22:58:05.424Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141ca-1302-7900-8980-37ac52de6fc6
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 6a581a61953be83decd0f022b0591bc30e3eb39190a701ca684064e43ed9f48d
              started: "2026-10-15T22:58:44.756Z"
              spawned: true
              bytes: 818265
              windows:
                - start: "2026-10-15T22:58:44.756Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141ca-f8c0-7d12-993e-38793b4752b0
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a7d1b9d02b1dd7dfc8c9457e3e014cc48fc7ea22ca0b23c4577e9f9e209cfd39
              started: "2026-10-15T22:59:44.122Z"
              spawned: false
              bytes: 422299
              windows:
                - start: "2026-10-15T22:59:44.122Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141cb-5dab-7260-9824-228d126ea623
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 10b83f12dd2ec7a039291bb25c683eacbc0478ac6fc76cec60b37bfa7c389ae1
              started: "2026-10-15T23:00:09.526Z"
              spawned: false
              bytes: 401586
              windows:
                - start: "2026-10-15T23:00:09.526Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141e2-9277-7521-a6de-104056107b04
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 11ed964791c3925a76f1e0470cffcec097eaee038de2bb54091356a93abdf567
              started: "2026-10-15T23:25:30.288Z"
              spawned: true
              bytes: 844381
              windows:
                - start: "2026-10-15T23:25:30.288Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a141e8-7837-72d0-9a15-e012125d5244
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 00f816a54b6f3cd87f27d0f6991169293c65c57475c2f5ecbbbeed79a77ce50a
              started: "2026-10-15T23:31:56.736Z"
              spawned: true
              bytes: 489128
              windows:
                - start: "2026-10-15T23:31:56.736Z"
                  end: "2026-10-15T17:37:28-07:00"
            - harness: codex
              id: 01a14205-e8bd-73b1-bebc-a98907c012b7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 07c554d32e797cbd82b0240ab4c3e23e1543ceb95f8d849a4b5d6f7e1a92440f
              started: "2026-10-16T00:04:06.091Z"
              spawned: true
              bytes: 675091
              windows:
                - start: "2026-10-16T00:04:06.091Z"
                  end: "2026-10-15T17:37:28-07:00"
        cuts:
            - dispatched: "2026-10-15T17:30:40-07:00"
              finalized: "2026-10-15T17:33:11-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T17:40:35-07:00"
      dispatched: "2026-10-15T17:38:32-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Inspected the pinned before/after table correction. Plan lines 15–29 now name deterministic helpers separately from IO orchestration; the Revisions passage at line 90 explains the correction. The referenced implementations support those classifications.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestLsofWritableNamesLiveConformance exercises production LsofWritableNames with writable, read-only, closed and reopened descriptors; it passed against real lsof. Makefile.local:136 connects it to the existing weekly/manual macOS workflow, and plan line 92 documents cadence. This adds conformance coverage without changing production behavior.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 44b10b71dd206c8f694724df6137f54b8c9f7cbc
      blocked: false
---

# Gate ledger — pair#584 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T17:33:11-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `pure-integration-classification` Core concepts classifies IO-driven retry and ownership proof as PURE
  workshop/plans/000584-native-owner-candidates-plan.md:15–16 lists observation retry and legacy source proof as pure entities. Their implementations invoke injected observation, metadata reads, and held-file probes (recovery_native_owner.go:74 and recovery_native_legacy.go:44); their tests require those fakes. ARCH-PURE and the Core concepts gate require promoting these orchestration entities to INTEGRATION and naming the actual deterministic helpers separately. Add a Revisions entry correcting the table.
- **BR-2** [Important] `external-dependency-conformance` Writable-file ownership proof has no repeatable live lsof conformance check
  cmd/internal/procutil/procutil.go:105 introduces an admission-critical external contract, but lsof_writable_test.go:14 replaces lsof with scripted output throughout its tests. No committed live check or recurring invocation exercises this contract against real descriptors; the issue's manual observations do not provide ongoing drift detection. ARCH-MOCK: add an opt-in conformance test using temporary files opened read-only, writable, and then closed, call the production LsofWritableNames boundary, and connect it to the live-check workflow with a documented cadence.

## Round 2 — 2026-10-15T17:40:35-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Inspected the pinned before/after table correction. Plan lines 15–29 now name deterministic helpers separately from IO orchestration; the Revisions passage at line 90 explains the correction. The referenced implementations support those classifications.
- BR-2 — addressed — TestLsofWritableNamesLiveConformance exercises production LsofWritableNames with writable, read-only, closed and reopened descriptors; it passed against real lsof. Makefile.local:136 connects it to the existing weekly/manual macOS workflow, and plan line 92 documents cadence. This adds conformance coverage without changing production behavior.

## Open findings

(none — every finding has been disposed)
