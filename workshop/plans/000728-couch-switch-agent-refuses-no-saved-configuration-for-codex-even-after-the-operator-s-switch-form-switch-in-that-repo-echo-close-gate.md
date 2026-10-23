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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ee782184-8afa-4c5c-8938-2abe6a21428b
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/436e173051c20c61
            reviewed_head: e7de5ca161079b427e7fe4726d3782ac03d35b95
            slot: pair:3
            requested_at: "2026-10-22T20:47:20.680067Z"
          phase: finished
          pid: 24060
          started_at: "2026-10-22T20:47:20.931459Z"
          finished_at: "2026-10-22T20:58:50.412782Z"
          session_id: 01a1655e-4ef6-78f0-a656-9b40d4658bc1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e7de5ca161079b427e7fe4726d3782ac03d35b95
        tracker_head: 1894b074f451dc7d3e26cfaec04d75907cc5ff03
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T13:45:39-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T13:45:39-07:00"
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
                  end: "2026-10-22T13:45:39-07:00"
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
                  end: "2026-10-22T13:45:39-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 72d878899199ecde9f8b40a6b3e37fe1be9fee0baf3ab0e02cd62ff6aa536188
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 8638632
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T13:45:39-07:00"
            - harness: codex
              id: 01a1654e-ca09-7bc1-9b4f-462d9ffb80be
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: d480d4b86ba0f8e2d840ecb293c790a866a49dfa6533dbaa636df79348e189ef
              started: "2026-10-22T20:30:24.984Z"
              spawned: true
              bytes: 929026
              windows:
                - start: "2026-10-22T20:30:24.984Z"
                  end: "2026-10-22T13:45:39-07:00"
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
        excluded_brain_minutes: 19.17791666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:18:35-07:00"
              finalized: "2026-10-22T13:28:07-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-22T13:58:52-07:00"
      dispatched: "2026-10-22T13:47:17-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-4
          disposition: addressed
          note: repopreferences_adoption.go:56–62 shares one deadline across migration and publication; repopreferences_store.go:87–94 bounds writer admission. The competing-store/coordinator regressions and console input/render/remote-completion regression pass. Restoring blocking writer admission in a disposable checkout makes both contention cases fail at the deadline assertion. The temporary checkout was removed.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: e7de5ca161079b427e7fe4726d3782ac03d35b95
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 0d7930a8-d074-46f8-a5fb-44c448735f85
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/49cb243041941ffe
            reviewed_head: 165ab40890ba9c506ecc26b3f93d41018a5db8f4
            slot: pair:3
            requested_at: "2026-10-22T23:17:27.172147Z"
          phase: finished
          pid: 64008
          started_at: "2026-10-22T23:17:27.273012Z"
          finished_at: "2026-10-22T23:22:26.688002Z"
          session_id: 01a165e7-b97b-7580-b74a-b55cd19b1a94
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 165ab40890ba9c506ecc26b3f93d41018a5db8f4
        tracker_head: 04475a196ee13c8108b09ac2b770fc7fafb8b684
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T16:15:53-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T16:15:53-07:00"
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
                  end: "2026-10-22T16:15:53-07:00"
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
                  end: "2026-10-22T16:15:53-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 3ed3f4c440b6d17dc27a131501a1b76450ab5cf1aa28323df1940524ae17f4f3
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 23446800
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T16:15:53-07:00"
            - harness: codex
              id: 01a1654e-ca09-7bc1-9b4f-462d9ffb80be
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: b2402ca7fb910ba96be44f2518ed663ed7af11913d0822316c91af59e7414c4f
              started: "2026-10-22T20:30:24.984Z"
              spawned: true
              bytes: 6642176
              windows:
                - start: "2026-10-22T20:30:24.984Z"
                  end: "2026-10-22T16:15:53-07:00"
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
        excluded_brain_minutes: 79.82053333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:18:35-07:00"
              finalized: "2026-10-22T13:28:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:47:17-07:00"
              finalized: "2026-10-22T13:58:52-07:00"
              finalizing: true
      "n": 4
      timestamp: "2026-10-22T16:22:28-07:00"
      dispatched: "2026-10-22T16:17:25-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-5
          severity: Critical
          title: Preserve the local interrupt phase's source token through the final park guard
          detail: cmd/internal/couchcmd/switch_interrupt.go:49 pins the current wrapper token only in a local variable, while InterruptBeforeAgentSwitch returns only error. Its caller at cmd/internal/couchcore/switchagent.go:443 therefore retains the form's empty request.Source. If the wrapper reconnects after the interrupt phase returns but before the final guard, an unchanged thread record produces the same fingerprint, and inspectSwitchLiveSourceWithOptions accepts the new connection because source is empty. This violates the plan's exact-generation/reconnect refusal contract (ARCH-ORDER, ARCH-SECURE). Carry pinned source evidence across the whole operation and require it at the final guard; sweep ordinary, cooperative and force local/remote paths. Add a controlled reconnect between phase completion and final confirmation and assert zero park/start effects.
          family: switch-source-generation-fencing
          round: 4
        - id: BR-6
          severity: Important
          title: Copy the prepared preview before clearing its force offer
          detail: cmd/internal/couchtty/menu_switchagent.go:23 and :108 assign through frame.SwitchPrepared.Force, but cloneMenuState at cmd/internal/couchtty/menu.go:1968 copies frames without copying SwitchPrepared. Reducing Escape or Cancel therefore changes the input MenuState and existing snapshots. showMenuPublishing renders such a snapshot after releasing c.mu, so cancellation can also race its Force read at menu_switchagent.go:255 (ARCH-PURE, ARCH-ORDER). Copy the PreparedAgentSwitch value before replacing its Force field, retaining the same one-shot offer object where needed. Test both cancellation routes for input immutability and concurrent snapshot rendering.
          family: immutable-reducer-state
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3edf6480-43b0-4c97-8800-d0d9f8545b2a
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/92632de98c05926e
            reviewed_head: 052e2ff49d5dbb4db398a8f00610379394a8b2d6
            slot: pair:3
            requested_at: "2026-10-22T23:28:40.631091Z"
          phase: finished
          pid: 12133
          started_at: "2026-10-22T23:28:40.727385Z"
          finished_at: "2026-10-22T23:33:21.289164Z"
          session_id: 01a165f2-0029-7441-9545-ed3cc8a470db
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 052e2ff49d5dbb4db398a8f00610379394a8b2d6
        tracker_head: 2e5078a826b8b2250ae046a3369393db7e0f704d
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T16:27:46-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T16:27:46-07:00"
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
                  end: "2026-10-22T16:27:46-07:00"
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
                  end: "2026-10-22T16:27:46-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 22f65544c5ba76142f7c4a6106ff9b3aedaa3f6aa4b64e45dc444c63e7152f2a
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 24459328
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T16:27:46-07:00"
            - harness: codex
              id: 01a1654e-ca09-7bc1-9b4f-462d9ffb80be
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: b2402ca7fb910ba96be44f2518ed663ed7af11913d0822316c91af59e7414c4f
              started: "2026-10-22T20:30:24.984Z"
              spawned: true
              bytes: 6642176
              windows:
                - start: "2026-10-22T20:30:24.984Z"
                  end: "2026-10-22T16:27:46-07:00"
            - harness: codex
              id: 01a165ec-bc82-7f53-a756-f1b85f5140d7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: 9193a465a2dd0eb34b692dd7c6fc6cd80edfca660402ddaa777bf2da514149c2
              started: "2026-10-22T23:22:56.094Z"
              spawned: true
              bytes: 719616
              windows:
                - start: "2026-10-22T23:22:56.094Z"
                  end: "2026-10-22T16:27:46-07:00"
        queue_waits:
            - attempt_id: d21fcedfda5b439e8007eef7615195fc
              slot: pair:3
              record_digest: a4f27f0124de149527f21bf57279c7755a778bf7484740693f934df8be8a58d8
              start: "2026-10-22T20:46:00.409747Z"
              end: "2026-10-22T23:14:19.613041Z"
              queue_seconds: 8899.203294
              raw_seconds: 8899.203294
              exact: true
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
        excluded_brain_minutes: 79.82053333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:18:35-07:00"
              finalized: "2026-10-22T13:28:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:47:17-07:00"
              finalized: "2026-10-22T13:58:52-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-22T16:17:25-07:00"
              finalized: "2026-10-22T16:22:28-07:00"
              finalizing: false
      "n": 5
      timestamp: "2026-10-22T16:33:22-07:00"
      dispatched: "2026-10-22T16:28:38-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-5
          disposition: addressed
          note: switchagent.go:414 and :450 retain the verified token through final confirmation. TestSwitchAgentPinsLiveSourceAcrossFinalGuard covers local/remote ordinary/cooperative reconnects with zero effects. Removing token retention caused both local paths to park/start. Force retains its consent-bound token and has replacement/refusal coverage.
          round: 5
        - id: BR-6
          disposition: addressed
          note: menu_switchagent.go:118 copies PreparedAgentSwitch before clearing Force. Both cancellation routes preserve input and render snapshots. Reintroducing shared mutation failed both immutability tests and produced rendering data races.
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 052e2ff49d5dbb4db398a8f00610379394a8b2d6
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 30d68429-2cf4-444d-984a-1e208fee159b
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/da10127af18cdd3b
            reviewed_head: c6967052e028bd896a40daa9f7751a4066984556
            slot: pair:3
            requested_at: "2026-10-23T00:32:03.72371Z"
          phase: finished
          pid: 74961
          started_at: "2026-10-23T00:32:03.869703Z"
          finished_at: "2026-10-23T00:34:30.081379Z"
          session_id: 01a1662c-08bf-7232-ad96-c8183f0f63dc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: c6967052e028bd896a40daa9f7751a4066984556
        tracker_head: 6cf5839e4dc62ad06ff195febd934a56854eec01
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T17:30:52-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T17:30:52-07:00"
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
                  end: "2026-10-22T17:30:52-07:00"
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
                  end: "2026-10-22T17:30:52-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 9ae1175fc2d51b6d778c9f07a65d1c2409933287949f1c68b5c10d754b72be3c
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 30022124
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T17:30:52-07:00"
            - harness: codex
              id: 01a1654e-ca09-7bc1-9b4f-462d9ffb80be
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: 1809db34f6e0bccc8a29a317e807513ca094d410a39fa06a27ca456e63100870
              started: "2026-10-22T20:30:24.984Z"
              spawned: true
              bytes: 7844340
              windows:
                - start: "2026-10-22T20:30:24.984Z"
                  end: "2026-10-22T17:30:52-07:00"
            - harness: codex
              id: 01a165ec-bc82-7f53-a756-f1b85f5140d7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: d1f1e83fa1a6a738f706ad165c45d9ebf3c5182bb8fa88d355deb81f6470bc19
              started: "2026-10-22T23:22:56.094Z"
              spawned: true
              bytes: 3363440
              windows:
                - start: "2026-10-22T23:22:56.094Z"
                  end: "2026-10-22T17:30:52-07:00"
        queue_waits:
            - attempt_id: d21fcedfda5b439e8007eef7615195fc
              slot: pair:3
              record_digest: a4f27f0124de149527f21bf57279c7755a778bf7484740693f934df8be8a58d8
              start: "2026-10-22T20:46:00.409747Z"
              end: "2026-10-22T23:14:19.613041Z"
              queue_seconds: 8899.203294
              raw_seconds: 8899.203294
              exact: true
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
              end: "2026-10-22T02:59:28-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T03:01:39-07:00"
              end: "2026-10-22T10:17:54.82Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:00:58.434Z"
              end: "2026-10-22T16:19:20.991Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T17:57:12.75Z"
              end: "2026-10-22T18:12:47.687992Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T19:43:59.38Z"
              end: "2026-10-22T21:08:37.884151Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T20:30:26.259Z"
              end: "2026-10-22T20:49:13.472Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:25:59.742942Z"
              end: "2026-10-22T23:37:22.441Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:25:59.742942Z"
              end: "2026-10-22T23:36:11.884Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:28:19.001Z"
              end: "2026-10-22T23:35:10.51Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 125.3493
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:18:35-07:00"
              finalized: "2026-10-22T13:28:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:47:17-07:00"
              finalized: "2026-10-22T13:58:52-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-22T16:17:25-07:00"
              finalized: "2026-10-22T16:22:28-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-22T16:28:38-07:00"
              finalized: "2026-10-22T16:33:22-07:00"
              finalizing: true
      "n": 6
      timestamp: "2026-10-22T17:34:31-07:00"
      dispatched: "2026-10-22T17:32:00-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      reviewed: c6967052e028bd896a40daa9f7751a4066984556
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b6766e61-ff9e-4b18-8101-6212703f30cd
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 7
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/6fa35332b55fb2e9
            reviewed_head: 6a854ad5a7a774534b7312d85872310e6d42d19c
            slot: pair:3
            requested_at: "2026-10-23T04:58:18.497677Z"
          phase: finished
          pid: 4617
          started_at: "2026-10-23T04:58:18.694389Z"
          finished_at: "2026-10-23T05:04:22.104385Z"
          session_id: 01a1671f-cabd-7ba1-a3ef-ec58e4c94643
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 6a854ad5a7a774534b7312d85872310e6d42d19c
        tracker_head: bdf81dc5a3b5fc5c77ffbc68e65a8aff13b55439
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T21:55:39-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T21:55:39-07:00"
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
                  end: "2026-10-22T21:55:39-07:00"
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
                  end: "2026-10-22T21:55:39-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 0ed6a6eebf2f276951f2f008b1c4826aa99885e4190fc8bdc830517dfc1159de
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 35285327
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T21:55:39-07:00"
            - harness: codex
              id: 01a1654e-ca09-7bc1-9b4f-462d9ffb80be
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: 1809db34f6e0bccc8a29a317e807513ca094d410a39fa06a27ca456e63100870
              started: "2026-10-22T20:30:24.984Z"
              spawned: true
              bytes: 7844340
              windows:
                - start: "2026-10-22T20:30:24.984Z"
                  end: "2026-10-22T21:55:39-07:00"
            - harness: codex
              id: 01a165ec-bc82-7f53-a756-f1b85f5140d7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: d1f1e83fa1a6a738f706ad165c45d9ebf3c5182bb8fa88d355deb81f6470bc19
              started: "2026-10-22T23:22:56.094Z"
              spawned: true
              bytes: 3363440
              windows:
                - start: "2026-10-22T23:22:56.094Z"
                  end: "2026-10-22T21:55:39-07:00"
        queue_waits:
            - attempt_id: d21fcedfda5b439e8007eef7615195fc
              slot: pair:3
              record_digest: a4f27f0124de149527f21bf57279c7755a778bf7484740693f934df8be8a58d8
              start: "2026-10-22T20:46:00.409747Z"
              end: "2026-10-22T23:14:19.613041Z"
              queue_seconds: 8899.203294
              raw_seconds: 8899.203294
              exact: true
            - attempt_id: 4b4a41391cbd3d540d4a3b3a382d3255
              slot: pair:3
              record_digest: 1e41c504b84c577a67a530e0d88f66aebc946d29a95713efe91d97407e99af6e
              start: "2026-10-23T01:05:00.49232Z"
              end: "2026-10-23T02:26:12.619906Z"
              queue_seconds: 4872.127586
              raw_seconds: 4872.127586
              exact: true
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
              end: "2026-10-22T10:17:54.82Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:00:58.434Z"
              end: "2026-10-22T16:19:20.991Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T17:57:12.75Z"
              end: "2026-10-22T18:12:47.687992Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T19:43:59.38Z"
              end: "2026-10-22T21:08:37.884151Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T20:30:26.259Z"
              end: "2026-10-22T20:49:13.472Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:25:59.742942Z"
              end: "2026-10-22T16:37:33-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:40:05-07:00"
              end: "2026-10-22T23:43:20.969895Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:25:59.742942Z"
              end: "2026-10-22T16:37:33-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:40:05-07:00"
              end: "2026-10-22T23:43:20.969895Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:28:19.001Z"
              end: "2026-10-22T16:37:33-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:40:05-07:00"
              end: "2026-10-22T23:43:20.969895Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-23T01:11:17.546337Z"
              end: "2026-10-23T01:22:44.827Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 241.22538333333335
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:18:35-07:00"
              finalized: "2026-10-22T13:28:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:47:17-07:00"
              finalized: "2026-10-22T13:58:52-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-22T16:17:25-07:00"
              finalized: "2026-10-22T16:22:28-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-22T16:28:38-07:00"
              finalized: "2026-10-22T16:33:22-07:00"
              finalizing: true
            - boundary: M3
              dispatched: "2026-10-22T17:32:00-07:00"
              finalized: "2026-10-22T17:34:31-07:00"
              finalizing: true
      "n": 7
      timestamp: "2026-10-22T22:04:24-07:00"
      dispatched: "2026-10-22T21:58:14-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: menu_switchagent.go renders borrowed donor, conflict, own and absent provenance; TestSwitchAgentConfigurationProvenanceRequiresExplicitAcceptance passed and asserts those distinctions.
          round: 7
        - id: BR-2
          disposition: addressed
          note: README.md:1359 describes the Couch repository document and independent Pair defaults; README.md:1380 describes final-attachment publication. These match repopreferences_store.go, launcher/createflow.go and couchcmd/run.go:1029.
          round: 7
        - id: BR-3
          disposition: addressed
          note: TestSpawnPreparedRefusesDriftByFingerprint and TestNestedRepositoryStorageRemainsIndependentOfOuterSlot now exercise repository preferences and independently passed together.
          round: 7
        - id: BR-4
          disposition: addressed
          note: repopreferences_adoption.go:56 shares the migration/publication deadline. Deadline and competing-writer regressions passed, as did TestRepositoryPublicationKeepsConsoleResponsiveAndRemotePending.
          round: 7
        - id: BR-5
          disposition: addressed
          note: SwitchAgent retains returned source tokens through the final guard. TestSwitchAgentPinsLiveSourceAcrossFinalGuard passed and requires refusal before effects for local/remote ordinary/cooperative reconnection.
          round: 7
        - id: BR-6
          disposition: addressed
          note: cancelSwitchForceConfirmation copies the prepared preview before clearing Force. Input/snapshot preservation and concurrent-render cancellation regressions passed under the race detector.
          round: 7
      findings:
        - kind: defect
          id: BR-7
          severity: Critical
          title: Switch-form preparation can replay a pending journal before confirmation
          detail: cmd/internal/couchcore/switchagent.go:260 calls GetThread from the common preparation path used by PrepareAgentSwitch, PrepareKnownAgentSwitch and PrepareForcedAgentSwitch. On the production writable store, GetThread reaches withLock, which registers retention and invokes recoverStoreJournalLockedChecked; that applies journal images and removes journal.json before the form is confirmed. This contradicts the explicit no-initialization/no-retention/no-journal-recovery contract at workshop/plans/000728-shared-agent-startup-preferences-plan.md:230 (ARCH-PURPOSE, ARCH-SECURE, ARCH-ORDER). Use PreviewThreadContext for preparation and audit its transitive reads across all three entry points. Add full preparation/form regressions with pending journals in global and routed slot backends, asserting refusal and byte-identical storage through cancellation; the current migration-preview test exercises only previewRepositoryPreferences.
          family: read-only-preview-boundary
          round: 7
        - kind: defect
          id: BR-8
          severity: Important
          title: Edited-restart acceptance setup races the already-running console
          detail: 'cmd/internal/couchcmd/editedrestart_console_acceptance_test.go:71 starts newColdResumeRig, which starts Console.Run, before editedRestartConsoleSource performs its claim/register/park mutations at lines 30–50. Only the actionable provider is gated. The focused multi-package review run failed TestEditedRestartConsoleRemoteAdoption/codex/resume/attach-refused at line 123 with “lock thread store: resource temporarily unavailable” during fixture construction. The isolated case passed once and across ten repetitions, and a subsequent service selection passed, demonstrating schedule-dependent coverage rather than a stable failure oracle (ARCH-ORDER, ARCH-MOCK). Finish source setup before starting console workers, or explicitly fence every competing store observer; exercise the competing ordering deterministically.'
          family: test-fixture-initialization-ordering
          round: 7
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: eca3b47d-21b4-4a54-99a0-198688742aa7
            repository: github.com/xianxu/pair
            issue_id: "000728"
            recipe: milestone-review
            round: 8
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e9ca90b6c3fbc862
            reviewed_head: fba4fd7863e67c7f0df87005aa967fa5bc18040f
            slot: pair:3
            requested_at: "2026-10-23T05:15:51.561317Z"
          phase: finished
          pid: 27332
          started_at: "2026-10-23T05:15:51.726834Z"
          finished_at: "2026-10-23T05:19:32.843328Z"
          session_id: 01a1672f-dbcf-7520-8064-24ff80e44858
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: fba4fd7863e67c7f0df87005aa967fa5bc18040f
        tracker_head: 5718f53c13b34f241fb224c2810a67f7b08a1445
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-20T23:31:35-07:00"
        until: "2026-10-22T22:14:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T23:31:35-07:00"
              end: "2026-10-22T22:14:28-07:00"
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
                  end: "2026-10-22T22:14:28-07:00"
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
                  end: "2026-10-22T22:14:28-07:00"
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d9d7bbb3fae35af23167a10efad6432c771476d01c90a736476b8d30091af4b9
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 37097545
              windows:
                - start: "2026-10-21T19:51:15.615Z"
                  end: "2026-10-22T22:14:28-07:00"
            - harness: codex
              id: 01a1654e-ca09-7bc1-9b4f-462d9ffb80be
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: 1809db34f6e0bccc8a29a317e807513ca094d410a39fa06a27ca456e63100870
              started: "2026-10-22T20:30:24.984Z"
              spawned: true
              bytes: 7844340
              windows:
                - start: "2026-10-22T20:30:24.984Z"
                  end: "2026-10-22T22:14:28-07:00"
            - harness: codex
              id: 01a165ec-bc82-7f53-a756-f1b85f5140d7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: d1f1e83fa1a6a738f706ad165c45d9ebf3c5182bb8fa88d355deb81f6470bc19
              started: "2026-10-22T23:22:56.094Z"
              spawned: true
              bytes: 3363440
              windows:
                - start: "2026-10-22T23:22:56.094Z"
                  end: "2026-10-22T22:14:28-07:00"
            - harness: codex
              id: 01a16728-6d39-7f52-b72f-3e872ce84f9a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              digest: abfaa1c54de445256e67e0f4ec46439dcf9ab73678e460e269c7d0695eb2efd0
              started: "2026-10-23T05:07:45.281Z"
              spawned: true
              bytes: 594345
              windows:
                - start: "2026-10-23T05:07:45.281Z"
                  end: "2026-10-22T22:14:28-07:00"
        queue_waits:
            - attempt_id: d21fcedfda5b439e8007eef7615195fc
              slot: pair:3
              record_digest: a4f27f0124de149527f21bf57279c7755a778bf7484740693f934df8be8a58d8
              start: "2026-10-22T20:46:00.409747Z"
              end: "2026-10-22T23:14:19.613041Z"
              queue_seconds: 8899.203294
              raw_seconds: 8899.203294
              exact: true
            - attempt_id: 4b4a41391cbd3d540d4a3b3a382d3255
              slot: pair:3
              record_digest: 1e41c504b84c577a67a530e0d88f66aebc946d29a95713efe91d97407e99af6e
              start: "2026-10-23T01:05:00.49232Z"
              end: "2026-10-23T02:26:12.619906Z"
              queue_seconds: 4872.127586
              raw_seconds: 4872.127586
              exact: true
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
              end: "2026-10-22T10:17:54.82Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:00:58.434Z"
              end: "2026-10-22T16:19:20.991Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T17:57:12.75Z"
              end: "2026-10-22T18:12:47.687992Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T19:43:59.38Z"
              end: "2026-10-22T21:08:37.884151Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T20:30:26.259Z"
              end: "2026-10-22T20:49:13.472Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:25:59.742942Z"
              end: "2026-10-22T16:37:33-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:40:05-07:00"
              end: "2026-10-22T23:43:20.969895Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:25:59.742942Z"
              end: "2026-10-22T16:37:33-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:40:05-07:00"
              end: "2026-10-22T23:43:20.969895Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T23:28:19.001Z"
              end: "2026-10-22T16:37:33-07:00"
            - issue: "743"
              slot: pair:3
              start: "2026-10-22T16:40:05-07:00"
              end: "2026-10-22T23:43:20.969895Z"
            - issue: "743"
              slot: pair:3
              start: "2026-10-23T01:11:17.546337Z"
              end: "2026-10-23T01:22:44.827Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 256.6123
        cuts:
            - boundary: M1
              dispatched: "2026-10-22T13:03:45-07:00"
              finalized: "2026-10-22T13:10:03-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:18:35-07:00"
              finalized: "2026-10-22T13:28:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-22T13:47:17-07:00"
              finalized: "2026-10-22T13:58:52-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-22T16:17:25-07:00"
              finalized: "2026-10-22T16:22:28-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-22T16:28:38-07:00"
              finalized: "2026-10-22T16:33:22-07:00"
              finalizing: true
            - boundary: M3
              dispatched: "2026-10-22T17:32:00-07:00"
              finalized: "2026-10-22T17:34:31-07:00"
              finalizing: true
            - dispatched: "2026-10-22T21:58:14-07:00"
              finalized: "2026-10-22T22:04:24-07:00"
              finalizing: false
      "n": 8
      timestamp: "2026-10-22T22:19:34-07:00"
      dispatched: "2026-10-22T22:15:48-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-7
          disposition: addressed
          note: switchagent.go:263 uses PreviewThreadContext across ordinary, known and forced preparation. Transitive storage reads remain observational. Both preparation regressions pass; restoring GetThread makes all nine journal cases and all three initialization/retention cases fail.
          round: 8
        - id: BR-8
          disposition: addressed
          note: editedrestart_console_acceptance_test.go:66 completes source setup through the pre-Run hook. The deterministic independent observer and actual Console continuation observer pass. Moving setup after construction makes the regression fail for both agents; the remote-adoption matrix also passes under race.
          round: 8
      recipe: milestone-review
      isolation: fenced
      reviewed: fba4fd7863e67c7f0df87005aa967fa5bc18040f
      blocked: false
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

## Round 3 — 2026-10-22T13:58:52-07:00 (codex) — passed

### Disposed

- BR-4 — addressed — repopreferences_adoption.go:56–62 shares one deadline across migration and publication; repopreferences_store.go:87–94 bounds writer admission. The competing-store/coordinator regressions and console input/render/remote-completion regression pass. Restoring blocking writer admission in a disposable checkout makes both contention cases fail at the deadline assertion. The temporary checkout was removed.

## Round 4 — 2026-10-22T16:22:28-07:00 (codex) — BLOCKED

### Raised

- **BR-5** [Critical] `switch-source-generation-fencing` Preserve the local interrupt phase's source token through the final park guard
  cmd/internal/couchcmd/switch_interrupt.go:49 pins the current wrapper token only in a local variable, while InterruptBeforeAgentSwitch returns only error. Its caller at cmd/internal/couchcore/switchagent.go:443 therefore retains the form's empty request.Source. If the wrapper reconnects after the interrupt phase returns but before the final guard, an unchanged thread record produces the same fingerprint, and inspectSwitchLiveSourceWithOptions accepts the new connection because source is empty. This violates the plan's exact-generation/reconnect refusal contract (ARCH-ORDER, ARCH-SECURE). Carry pinned source evidence across the whole operation and require it at the final guard; sweep ordinary, cooperative and force local/remote paths. Add a controlled reconnect between phase completion and final confirmation and assert zero park/start effects.
- **BR-6** [Important] `immutable-reducer-state` Copy the prepared preview before clearing its force offer
  cmd/internal/couchtty/menu_switchagent.go:23 and :108 assign through frame.SwitchPrepared.Force, but cloneMenuState at cmd/internal/couchtty/menu.go:1968 copies frames without copying SwitchPrepared. Reducing Escape or Cancel therefore changes the input MenuState and existing snapshots. showMenuPublishing renders such a snapshot after releasing c.mu, so cancellation can also race its Force read at menu_switchagent.go:255 (ARCH-PURE, ARCH-ORDER). Copy the PreparedAgentSwitch value before replacing its Force field, retaining the same one-shot offer object where needed. Test both cancellation routes for input immutability and concurrent snapshot rendering.

## Round 5 — 2026-10-22T16:33:22-07:00 (codex) — passed

### Disposed

- BR-5 — addressed — switchagent.go:414 and :450 retain the verified token through final confirmation. TestSwitchAgentPinsLiveSourceAcrossFinalGuard covers local/remote ordinary/cooperative reconnects with zero effects. Removing token retention caused both local paths to park/start. Force retains its consent-bound token and has replacement/refusal coverage.
- BR-6 — addressed — menu_switchagent.go:118 copies PreparedAgentSwitch before clearing Force. Both cancellation routes preserve input and render snapshots. Reintroducing shared mutation failed both immutability tests and produced rendering data races.

## Round 6 — 2026-10-22T17:34:31-07:00 (codex) — passed

## Round 7 — 2026-10-22T22:04:24-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — menu_switchagent.go renders borrowed donor, conflict, own and absent provenance; TestSwitchAgentConfigurationProvenanceRequiresExplicitAcceptance passed and asserts those distinctions.
- BR-2 — addressed — README.md:1359 describes the Couch repository document and independent Pair defaults; README.md:1380 describes final-attachment publication. These match repopreferences_store.go, launcher/createflow.go and couchcmd/run.go:1029.
- BR-3 — addressed — TestSpawnPreparedRefusesDriftByFingerprint and TestNestedRepositoryStorageRemainsIndependentOfOuterSlot now exercise repository preferences and independently passed together.
- BR-4 — addressed — repopreferences_adoption.go:56 shares the migration/publication deadline. Deadline and competing-writer regressions passed, as did TestRepositoryPublicationKeepsConsoleResponsiveAndRemotePending.
- BR-5 — addressed — SwitchAgent retains returned source tokens through the final guard. TestSwitchAgentPinsLiveSourceAcrossFinalGuard passed and requires refusal before effects for local/remote ordinary/cooperative reconnection.
- BR-6 — addressed — cancelSwitchForceConfirmation copies the prepared preview before clearing Force. Input/snapshot preservation and concurrent-render cancellation regressions passed under the race detector.

### Raised

- **BR-7** [Critical] `read-only-preview-boundary` (kind: defect) Switch-form preparation can replay a pending journal before confirmation
  cmd/internal/couchcore/switchagent.go:260 calls GetThread from the common preparation path used by PrepareAgentSwitch, PrepareKnownAgentSwitch and PrepareForcedAgentSwitch. On the production writable store, GetThread reaches withLock, which registers retention and invokes recoverStoreJournalLockedChecked; that applies journal images and removes journal.json before the form is confirmed. This contradicts the explicit no-initialization/no-retention/no-journal-recovery contract at workshop/plans/000728-shared-agent-startup-preferences-plan.md:230 (ARCH-PURPOSE, ARCH-SECURE, ARCH-ORDER). Use PreviewThreadContext for preparation and audit its transitive reads across all three entry points. Add full preparation/form regressions with pending journals in global and routed slot backends, asserting refusal and byte-identical storage through cancellation; the current migration-preview test exercises only previewRepositoryPreferences.
- **BR-8** [Important] `test-fixture-initialization-ordering` (kind: defect) Edited-restart acceptance setup races the already-running console
  cmd/internal/couchcmd/editedrestart_console_acceptance_test.go:71 starts newColdResumeRig, which starts Console.Run, before editedRestartConsoleSource performs its claim/register/park mutations at lines 30–50. Only the actionable provider is gated. The focused multi-package review run failed TestEditedRestartConsoleRemoteAdoption/codex/resume/attach-refused at line 123 with “lock thread store: resource temporarily unavailable” during fixture construction. The isolated case passed once and across ten repetitions, and a subsequent service selection passed, demonstrating schedule-dependent coverage rather than a stable failure oracle (ARCH-ORDER, ARCH-MOCK). Finish source setup before starting console workers, or explicitly fence every competing store observer; exercise the competing ordering deterministically.

## Round 8 — 2026-10-22T22:19:34-07:00 (codex) — passed

### Disposed

- BR-7 — addressed — switchagent.go:263 uses PreviewThreadContext across ordinary, known and forced preparation. Transitive storage reads remain observational. Both preparation regressions pass; restoring GetThread makes all nine journal cases and all three initialization/retention cases fail.
- BR-8 — addressed — editedrestart_console_acceptance_test.go:66 completes source setup through the pre-Run hook. The deterministic independent observer and actual Console continuation observer pass. Moving setup after construction makes the regression fail for both agents; the remote-adoption matrix also passes under race.

## Open findings

(none — every finding has been disposed)
