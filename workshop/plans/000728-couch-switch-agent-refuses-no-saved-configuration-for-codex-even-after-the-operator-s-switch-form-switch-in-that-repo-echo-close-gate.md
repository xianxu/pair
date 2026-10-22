---
gate: boundary-review
issue: 728
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b3dfcf92-2230-45b7-a6a1-8918c2d5e14f
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/f9c922cf1d8d9e56
            reviewed_head: 5a0fb9f504e8672b8491d6637b1828fe0161ee73
            slot: pair:3
            requested_at: "2026-10-22T20:03:48.742598Z"
          phase: finished
          pid: 30769
          started_at: "2026-10-22T20:03:48.955302Z"
          finished_at: "2026-10-22T20:10:00.874922Z"
          session_id: 01a16536-7203-78e0-8f10-ad72537d7922
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 5a0fb9f504e8672b8491d6637b1828fe0161ee73
        tracker_head: 7ba6cc0eb5f81693864fc39260d0866794d87cab
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T12:59:17-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T12:59:17-07:00"
              began: "2026-10-20T23:31:35-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a15d17-8eb2-79b2-82d4-848985b4d8a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d450dc513b7acfdf5eba26e9524045df7b2a2851450e9d20f99abe9eb9fc44a9
              started: "2026-10-21T06:13:07.817Z"
              spawned: false
              bytes: 16235780
              windows:
                - start: "2026-10-20T23:31:35-07:00"
                  end: "2026-10-22T12:59:17-07:00"
            - harness: codex
              id: 01a15d29-153f-7452-97d3-7d2c1d094ad7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15d17-8eb2-79b2-82d4-848985b4d8a3
              digest: 799759a2ccd0bfe075a8730a654affc1c9c2887404b6a43c0a34678a426dbf24
              started: "2026-10-21T06:32:16.07Z"
              spawned: true
              bytes: 1797356
              windows:
                - start: "2026-10-21T06:32:16.07Z"
                  end: "2026-10-22T12:59:17-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: ac72325c3ec8be9d4d0d138e575b33c46ad1dd84307e965d0c05413f3f182d7a
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 6244265
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T12:59:17-07:00"
        protected:
            - issue: "743"
              slot: pair:3
              start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-21T07:00:58.259899Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-21T06:31:47.587Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:32:10.287Z"
              end: "2026-10-21T06:37:10.662Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:32:18.563Z"
              end: "2026-10-21T07:00:58.259899Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:40:45.831Z"
              end: "2026-10-21T06:41:59.932Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:45:37.509Z"
              end: "2026-10-21T06:49:22.955Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T09:44:41.27Z"
              end: "2026-10-21T02:59:39-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T02:59:39-07:00"
              end: "2026-10-21T03:13:35-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T03:17:33-07:00"
              end: "2026-10-21T03:18:54-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T03:18:54-07:00"
              end: "2026-10-21T10:28:52.200322Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T15:44:53.21Z"
              end: "2026-10-21T16:43:59.107Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T15:57:07.598Z"
              end: "2026-10-21T16:16:35.787Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T15:57:40.597Z"
              end: "2026-10-21T15:59:23.781Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:02:11.33Z"
              end: "2026-10-21T16:02:45.641Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:05:48.585Z"
              end: "2026-10-21T16:06:19.403Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:06:25.195Z"
              end: "2026-10-21T16:07:05.27Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:30:49.095Z"
              end: "2026-10-21T19:51:08.104Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:35:21.95Z"
              end: "2026-10-21T17:53:57.829Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:35:53.432Z"
              end: "2026-10-21T18:01:27.06Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:36:36.55Z"
              end: "2026-10-21T17:48:56.775Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T19:51:16.84Z"
              end: "2026-10-21T20:03:08.016343Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T20:03:08.376521Z"
              end: "2026-10-21T20:16:56.720381Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T23:30:07.401Z"
              end: "2026-10-21T16:30:28-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:34:05-07:00"
              end: "2026-10-21T23:50:36.112Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T23:53:15.351Z"
              end: "2026-10-22T00:08:26.944Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T01:58:45.649Z"
              end: "2026-10-22T02:14:00.132Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T02:56:58.521Z"
              end: "2026-10-22T03:12:27.833Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T04:33:50.839Z"
              end: "2026-10-22T04:35:16.088562Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T09:59:10.637Z"
              end: "2026-10-22T09:59:26.878Z"
      "n": 1
      timestamp: "2026-10-22T13:10:03-07:00"
      dispatched: "2026-10-22T13:03:45-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Borrowed and conflicting arguments reach the form without their required provenance
          detail: cmd/internal/couchcore/switchagent.go:274 now prefills another repository's arguments and returns Borrowed/Conflict metadata at line 309, but cmd/internal/couchtty/menu_switchagent.go:181 renders neither field. The existing reducer displays and accepts these arguments as ordinary parameters; a migration conflict likewise appears as an unexplained empty field. This violates M1's marked-borrowing and explicit-conflict contracts (ARCH-PURPOSE). Display the donor and “Borrowed — not saved for this repository”, expose conflicts, and add reducer/render/acceptance regressions covering own, borrowed, absent and conflicting configurations.
          family: configuration-acceptance-provenance
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: README still directs users to the retired preference authority
          detail: README.md:1331 describes per-path argument precedence, line 1344 identifies numbered .couch/preferences.json as current settings, and line 1355 instructs users to change Couch defaults by launching Pair directly. The implementation now uses Couch's global repository document and removes Pair-default reads, so that documented repair procedure cannot work. README.md is unchanged in the pinned range. Update these passages and the CLI description at line 676 for the delivered M1 behavior; atlas changes alone do not satisfy the README gate.
          family: user-documentation-traceability
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: Two existing regressions were left on the retired path-preference contract
          detail: 'The full couchcore package and an isolated rerun fail TestSpawnPreparedRefusesDriftByFingerprint at startfingerprint_test.go:62 and TestNestedRepositoryStorageRemainsIndependentOfOuterSlot at nested_repository_storage_test.go:97. The former now returns “read repository arguments: no such file or directory”; the latter reads through the obsolete routed helper retained at repopreferences_fixture_support_test.go:58 and finds no preference. Migrate these tests to the production repository-preference seam, retaining their stale-preview/no-effects and nested-repository isolation assertions, then rerun the package.'
          family: regression-suite-migration
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 642e58ec-c8de-4d74-8fa4-8c319eb6dfe6
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/ad3a4abaf30575fb
            reviewed_head: 32ababba112e3519b0009afb904b3df84acaf724
            slot: pair:3
            requested_at: "2026-10-22T20:18:37.80375Z"
          phase: finished
          pid: 91108
          started_at: "2026-10-22T20:18:37.915697Z"
          finished_at: "2026-10-22T20:28:05.507033Z"
          session_id: 01a16544-02de-72e0-8b78-e0bf30018dc7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 32ababba112e3519b0009afb904b3df84acaf724
        tracker_head: 6c9fa74e5fb39cdc62b44b53b742911cf6fc2224
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T13:17:34-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T13:17:34-07:00"
              began: "2026-10-20T23:31:35-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a15d17-8eb2-79b2-82d4-848985b4d8a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d450dc513b7acfdf5eba26e9524045df7b2a2851450e9d20f99abe9eb9fc44a9
              started: "2026-10-21T06:13:07.817Z"
              spawned: false
              bytes: 16235780
              windows:
                - start: "2026-10-20T23:31:35-07:00"
                  end: "2026-10-22T13:17:34-07:00"
            - harness: codex
              id: 01a15d29-153f-7452-97d3-7d2c1d094ad7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15d17-8eb2-79b2-82d4-848985b4d8a3
              digest: 799759a2ccd0bfe075a8730a654affc1c9c2887404b6a43c0a34678a426dbf24
              started: "2026-10-21T06:32:16.07Z"
              spawned: true
              bytes: 1797356
              windows:
                - start: "2026-10-21T06:32:16.07Z"
                  end: "2026-10-22T13:17:34-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 0221d37ae90d49ca669e6dce0f4274cece8301ca44d9cc574692b1aa740fa8a4
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 7183103
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T13:17:34-07:00"
        protected:
            - issue: "743"
              slot: pair:3
              start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-21T07:00:58.259899Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-21T06:31:47.587Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:32:10.287Z"
              end: "2026-10-21T06:37:10.662Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:32:18.563Z"
              end: "2026-10-21T07:00:58.259899Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:40:45.831Z"
              end: "2026-10-21T06:41:59.932Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T06:45:37.509Z"
              end: "2026-10-21T06:49:22.955Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T09:44:41.27Z"
              end: "2026-10-21T02:59:39-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T02:59:39-07:00"
              end: "2026-10-21T03:13:35-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T03:17:33-07:00"
              end: "2026-10-21T03:18:54-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T03:18:54-07:00"
              end: "2026-10-21T10:28:52.200322Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T15:44:53.21Z"
              end: "2026-10-21T16:43:59.107Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T15:57:07.598Z"
              end: "2026-10-21T16:16:35.787Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T15:57:40.597Z"
              end: "2026-10-21T15:59:23.781Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:02:11.33Z"
              end: "2026-10-21T16:02:45.641Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:05:48.585Z"
              end: "2026-10-21T16:06:19.403Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:06:25.195Z"
              end: "2026-10-21T16:07:05.27Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:30:49.095Z"
              end: "2026-10-21T19:51:08.104Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:35:21.95Z"
              end: "2026-10-21T17:53:57.829Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:35:53.432Z"
              end: "2026-10-21T18:01:27.06Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T17:36:36.55Z"
              end: "2026-10-21T17:48:56.775Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T19:51:16.84Z"
              end: "2026-10-21T20:03:08.016343Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T20:03:08.376521Z"
              end: "2026-10-21T20:16:56.720381Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T23:30:07.401Z"
              end: "2026-10-21T16:30:28-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T16:34:05-07:00"
              end: "2026-10-21T23:50:36.112Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-21T23:53:15.351Z"
              end: "2026-10-22T00:08:26.944Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T01:58:45.649Z"
              end: "2026-10-22T02:14:00.132Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T02:56:58.521Z"
              end: "2026-10-22T03:12:27.833Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T04:33:50.839Z"
              end: "2026-10-22T04:35:16.088562Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T09:59:10.637Z"
              end: "2026-10-22T09:59:26.878Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 7.477116666666666
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-22T13:28:07-07:00"
      dispatched: "2026-10-22T13:18:35-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: menu_switchagent.go:184-199 renders own, borrowed/donor, absent and conflict states. The reducer/render/cancel/acceptance regression passes at head; removing the rendering correction through a temporary overlay makes all eight agent/state cases fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md:676-682 and 1328-1364 now describe Couch repository arguments, legacy migration inputs, independent Pair defaults and post-attachment publication/repair. These passages match repopreferences_store.go:28, run.go:1025-1030 and launcher's trusted-profile guards.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Both tests now use canonical repository preferences and pass in isolation. Restoring their original fixtures through a temporary overlay reproduces both reported failures. The complete couchcore rerun failed an unchanged live PTY teardown test, which passed on isolated rerun.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Critical
          title: Optional preference publication can block console input and rendering indefinitely
          detail: 'couchcmd/run.go:1029 synchronously calls PublishAdoptedConfiguration on the console dispatcher. repopreferences_adoption.go:62 then enters repopreferences_store.go:57''s blocking withLock without passing the migration deadline; this ultimately uses blocking flock. A controlled competing-writer check remained blocked after 5.303 seconds and completed only when the lock was released, with no warning. ARCH-CONSTRAINTS: keep optional publication off the console event path, bound coordinator/store lock acquisition, and report timeout through the existing started-but-not-remembered warning without stopping the adopted child. Add a controlled contention regression.'
          family: optional-console-work-bounds
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#728 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T13:10:03-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `configuration-acceptance-provenance` (kind: defect) Borrowed and conflicting arguments reach the form without their required provenance
  cmd/internal/couchcore/switchagent.go:274 now prefills another repository's arguments and returns Borrowed/Conflict metadata at line 309, but cmd/internal/couchtty/menu_switchagent.go:181 renders neither field. The existing reducer displays and accepts these arguments as ordinary parameters; a migration conflict likewise appears as an unexplained empty field. This violates M1's marked-borrowing and explicit-conflict contracts (ARCH-PURPOSE). Display the donor and “Borrowed — not saved for this repository”, expose conflicts, and add reducer/render/acceptance regressions covering own, borrowed, absent and conflicting configurations.
- **BR-2** [Important] `user-documentation-traceability` (kind: defect) README still directs users to the retired preference authority
  README.md:1331 describes per-path argument precedence, line 1344 identifies numbered .couch/preferences.json as current settings, and line 1355 instructs users to change Couch defaults by launching Pair directly. The implementation now uses Couch's global repository document and removes Pair-default reads, so that documented repair procedure cannot work. README.md is unchanged in the pinned range. Update these passages and the CLI description at line 676 for the delivered M1 behavior; atlas changes alone do not satisfy the README gate.
- **BR-3** [Important] `regression-suite-migration` (kind: defect) Two existing regressions were left on the retired path-preference contract
  The full couchcore package and an isolated rerun fail TestSpawnPreparedRefusesDriftByFingerprint at startfingerprint_test.go:62 and TestNestedRepositoryStorageRemainsIndependentOfOuterSlot at nested_repository_storage_test.go:97. The former now returns “read repository arguments: no such file or directory”; the latter reads through the obsolete routed helper retained at repopreferences_fixture_support_test.go:58 and finds no preference. Migrate these tests to the production repository-preference seam, retaining their stale-preview/no-effects and nested-repository isolation assertions, then rerun the package.

## Round 2 — 2026-10-22T13:28:07-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — menu_switchagent.go:184-199 renders own, borrowed/donor, absent and conflict states. The reducer/render/cancel/acceptance regression passes at head; removing the rendering correction through a temporary overlay makes all eight agent/state cases fail.
- BR-2 — addressed — README.md:676-682 and 1328-1364 now describe Couch repository arguments, legacy migration inputs, independent Pair defaults and post-attachment publication/repair. These passages match repopreferences_store.go:28, run.go:1025-1030 and launcher's trusted-profile guards.
- BR-3 — addressed — Both tests now use canonical repository preferences and pass in isolation. Restoring their original fixtures through a temporary overlay reproduces both reported failures. The complete couchcore rerun failed an unchanged live PTY teardown test, which passed on isolated rerun.

### Raised

- **BR-4** [Critical] `optional-console-work-bounds` (kind: defect) Optional preference publication can block console input and rendering indefinitely
  couchcmd/run.go:1029 synchronously calls PublishAdoptedConfiguration on the console dispatcher. repopreferences_adoption.go:62 then enters repopreferences_store.go:57's blocking withLock without passing the migration deadline; this ultimately uses blocking flock. A controlled competing-writer check remained blocked after 5.303 seconds and completed only when the lock was released, with no warning. ARCH-CONSTRAINTS: keep optional publication off the console event path, bound coordinator/store lock acquisition, and report timeout through the existing started-but-not-remembered warning without stopping the adopted child. Add a controlled contention regression.

## Open findings

- **BR-4** [Critical] `optional-console-work-bounds` (kind: defect) Optional preference publication can block console input and rendering indefinitely
