---
gate: boundary-review
issue: 706
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 81cc1f91-ad07-4e84-9a23-cfb004d5152d
            repository: github.com/xianxu/pair
            issue_id: "000706"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/42c9bd609fad4fab
            reviewed_head: 0e07fa2cbad5092e837f181d802ac6500028fd48
            slot: pair:6
            requested_at: "2026-10-20T18:11:40.162716Z"
          phase: finished
          pid: 23639
          started_at: "2026-10-20T18:11:40.341101Z"
          finished_at: "2026-10-20T18:14:06.336751Z"
          session_id: 01a15a83-0e33-7871-9678-4ae718f246cb
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 0e07fa2cbad5092e837f181d802ac6500028fd48
        tracker_head: abe3eac4533e690b0342fcdd503b09e48bacb057
        branch_point: fc2932e416e5dfb4096e30c8f1ef8baaf355c780
        since: "2026-10-19T20:44:04-07:00"
        until: "2026-10-20T11:10:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T20:44:04-07:00"
              end: "2026-10-20T11:10:45-07:00"
              began: "2026-10-19T20:44:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: ea3b0c36cdf241339c37ef8f1673ef8ed525d93bb4927f7360e137386660d4f7
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 57948838
              windows:
                - start: "2026-10-19T20:44:04-07:00"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a1576f-cd4b-72f2-95dc-3dced4835dcb
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 23fc1ec80a2c7363b69d5856a50396aa7a45e9d207a1dd751a6c87528f3c9d27
              started: "2026-10-20T03:51:48.038Z"
              spawned: false
              bytes: 623919
              windows:
                - start: "2026-10-20T03:51:48.038Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15771-61e1-75d0-b25a-f0c092bce1c6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 215123b95afe9271ee0f3e822fad647307f194654c0d10b880870637810846a4
              started: "2026-10-20T03:53:31.517Z"
              spawned: false
              bytes: 426064
              windows:
                - start: "2026-10-20T03:53:31.517Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15772-a7f0-7a71-8179-40bb6f026d64
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c3ebdafbf2992b50f68053cf01ef42e285cdb4cdfa68df37001118898abdbf92
              started: "2026-10-20T03:54:54.909Z"
              spawned: false
              bytes: 452494
              windows:
                - start: "2026-10-20T03:54:54.909Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15773-30ba-7fd1-a14d-117b4947d2b0
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f48d52c254c920ccc35103a443c2bc82a899caf33c9b5aac72b685559225e762
              started: "2026-10-20T03:55:29.867Z"
              spawned: false
              bytes: 413603
              windows:
                - start: "2026-10-20T03:55:29.867Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15773-e28b-7e71-a28e-1412dd5a3698
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fddacb7f6bd07bb2cacf33e86e19ba1f8cd39d89187255d4a77109fd4a853290
              started: "2026-10-20T03:56:15.104Z"
              spawned: true
              bytes: 463283
              windows:
                - start: "2026-10-20T03:56:15.104Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15774-1140-78d0-a327-5734dca0eab7
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: c812386785b161e27ff3f425058819952060a75457251b20030e27f2e13acb44
              started: "2026-10-20T03:56:27.037Z"
              spawned: true
              bytes: 437362
              windows:
                - start: "2026-10-20T03:56:27.037Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15775-fbd5-7101-9ac2-8fbdc812ee2c
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 8071fc76f40bba1456151fe7bbb8316fd0951386a52f7e96fa838fb3cfd6c63d
              started: "2026-10-20T03:58:32.65Z"
              spawned: true
              bytes: 599209
              windows:
                - start: "2026-10-20T03:58:32.65Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15a7c-66b4-7112-a945-614b32b4bdf5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: e8d805037a9a57cdcb82217628ffe1d0fecfde0a2e07b5c4c764250e7f64a745
              started: "2026-10-20T18:04:24.804Z"
              spawned: true
              bytes: 558075
              windows:
                - start: "2026-10-20T18:04:24.804Z"
                  end: "2026-10-20T11:10:45-07:00"
            - harness: codex
              id: 01a15a7c-8cd8-75c1-9218-94721e444c0b
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: dbad3c695f735056d3f92a7ec1c67b9ec5af141ab5ac6d5e9525284c6682d610
              started: "2026-10-20T18:04:34.548Z"
              spawned: true
              bytes: 471235
              windows:
                - start: "2026-10-20T18:04:34.548Z"
                  end: "2026-10-20T11:10:45-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 86.39396666666667
      "n": 1
      timestamp: "2026-10-20T11:14:07-07:00"
      dispatched: "2026-10-20T11:11:37-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Year creation reuses stale root capacity and can exceed the listing limit
          detail: 'cmd/internal/historyarchive/drive_months.go:199-202 passes the original root inventory to every EnsureChild call without accounting for newly created years. With 99,999 root entries and pending bundles in two absent years, both mkdir attempts are admitted; the second creates entry 100,001, whose confirmation and subsequent root listings fail DecodeFlatInventory''s limit. ARCH-CONSTRAINTS and ARCH-ORDER: carry forward confirmed parent observations and reserve capacity for potentially completed creates. Add a coordinator regression covering successive year creations at capacity, including an uncertain first outcome.'
          family: remote-capacity-reservation
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: README update appears missing for month-folder uploads and per-folder batching
          detail: README.md:2009 still describes one upload batch, while M1 routes new pairs to YYYY/MM and runs a batch per active physical folder. README.md is unchanged in the reviewed range despite this new file-tree convention and the boundary's docs gate. Update the current upload description, legacy behavior, and shared byte budget now; document M2 recovery features when delivered.
          family: user-surface-documentation
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8f1d271f-256b-4829-9f93-e5f9102b6dd6
            repository: github.com/xianxu/pair
            issue_id: "000706"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a476782903c4fa4d
            reviewed_head: 686270450d4949ce1738cbaa91c806a7ad8d83ab
            slot: pair:6
            requested_at: "2026-10-20T18:16:42.654874Z"
          phase: finished
          pid: 62068
          started_at: "2026-10-20T18:16:42.771872Z"
          finished_at: "2026-10-20T18:19:29.296147Z"
          session_id: 01a15a87-ab35-75f3-a75a-3b74e1b321f9
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 686270450d4949ce1738cbaa91c806a7ad8d83ab
        tracker_head: 5fe89353e75f9d65fc3704476e3c65f58317981b
        branch_point: fc2932e416e5dfb4096e30c8f1ef8baaf355c780
        since: "2026-10-19T20:44:04-07:00"
        until: "2026-10-20T11:16:11-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T20:44:04-07:00"
              end: "2026-10-20T11:16:11-07:00"
              began: "2026-10-19T20:44:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 681dc7abde0209dbd553f142b14861b6b2a048839154bd6486eca81e1b4efc38
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 58403686
              windows:
                - start: "2026-10-19T20:44:04-07:00"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a1576f-cd4b-72f2-95dc-3dced4835dcb
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 23fc1ec80a2c7363b69d5856a50396aa7a45e9d207a1dd751a6c87528f3c9d27
              started: "2026-10-20T03:51:48.038Z"
              spawned: false
              bytes: 623919
              windows:
                - start: "2026-10-20T03:51:48.038Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15771-61e1-75d0-b25a-f0c092bce1c6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 215123b95afe9271ee0f3e822fad647307f194654c0d10b880870637810846a4
              started: "2026-10-20T03:53:31.517Z"
              spawned: false
              bytes: 426064
              windows:
                - start: "2026-10-20T03:53:31.517Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15772-a7f0-7a71-8179-40bb6f026d64
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c3ebdafbf2992b50f68053cf01ef42e285cdb4cdfa68df37001118898abdbf92
              started: "2026-10-20T03:54:54.909Z"
              spawned: false
              bytes: 452494
              windows:
                - start: "2026-10-20T03:54:54.909Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15773-30ba-7fd1-a14d-117b4947d2b0
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f48d52c254c920ccc35103a443c2bc82a899caf33c9b5aac72b685559225e762
              started: "2026-10-20T03:55:29.867Z"
              spawned: false
              bytes: 413603
              windows:
                - start: "2026-10-20T03:55:29.867Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15773-e28b-7e71-a28e-1412dd5a3698
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fddacb7f6bd07bb2cacf33e86e19ba1f8cd39d89187255d4a77109fd4a853290
              started: "2026-10-20T03:56:15.104Z"
              spawned: true
              bytes: 463283
              windows:
                - start: "2026-10-20T03:56:15.104Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15774-1140-78d0-a327-5734dca0eab7
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: c812386785b161e27ff3f425058819952060a75457251b20030e27f2e13acb44
              started: "2026-10-20T03:56:27.037Z"
              spawned: true
              bytes: 437362
              windows:
                - start: "2026-10-20T03:56:27.037Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15775-fbd5-7101-9ac2-8fbdc812ee2c
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 8071fc76f40bba1456151fe7bbb8316fd0951386a52f7e96fa838fb3cfd6c63d
              started: "2026-10-20T03:58:32.65Z"
              spawned: true
              bytes: 599209
              windows:
                - start: "2026-10-20T03:58:32.65Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15a7c-66b4-7112-a945-614b32b4bdf5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: e8d805037a9a57cdcb82217628ffe1d0fecfde0a2e07b5c4c764250e7f64a745
              started: "2026-10-20T18:04:24.804Z"
              spawned: true
              bytes: 558075
              windows:
                - start: "2026-10-20T18:04:24.804Z"
                  end: "2026-10-20T11:16:11-07:00"
            - harness: codex
              id: 01a15a7c-8cd8-75c1-9218-94721e444c0b
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: dbad3c695f735056d3f92a7ec1c67b9ec5af141ab5ac6d5e9525284c6682d610
              started: "2026-10-20T18:04:34.548Z"
              spawned: true
              bytes: 471235
              windows:
                - start: "2026-10-20T18:04:34.548Z"
                  end: "2026-10-20T11:16:11-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 86.39396666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T11:11:37-07:00"
              finalized: "2026-10-20T11:14:07-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-20T11:19:30-07:00"
      dispatched: "2026-10-20T11:16:40-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: rclone_folders.go:161-175 reserves capacity after confirmed or uncertain creates; drive_months.go:201,218 applies it to both hierarchy levels. The year-capacity and uncertain month-capacity regressions pass at head. Bypassing the helper in a disposable copy makes all three cases fail; the copy was removed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md:2009 now documents YYYY/MM uploads, retained legacy pairs, serial physical-folder batches, and the shared 4 GiB budget. The pinned before/after passage agrees with drive_months.go routing and runFolder execution.
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 686270450d4949ce1738cbaa91c806a7ad8d83ab
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c5c8b3a4-1e39-40f7-94fe-0f9472dce8cf
            repository: github.com/xianxu/pair
            issue_id: "000706"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/0a332015a3ee69e8
            reviewed_head: a659b2823d24ecdf08b1e9ac8bf7d7924da1d77b
            slot: pair:6
            requested_at: "2026-10-20T21:23:11.917034Z"
          phase: finished
          pid: 58234
          started_at: "2026-10-20T21:23:12.120335Z"
          finished_at: "2026-10-20T21:28:43.789417Z"
          session_id: 01a15b32-67fe-7151-87ce-9734136d5bab
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: a659b2823d24ecdf08b1e9ac8bf7d7924da1d77b
        tracker_head: 49f36c76eb883d391d0e8f77149356430d908251
        branch_point: 72a25ac7b601f005fa3199f2622299794a885da3
        since: "2026-10-19T20:44:04-07:00"
        until: "2026-10-20T14:22:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T20:44:04-07:00"
              end: "2026-10-20T14:22:04-07:00"
              began: "2026-10-19T20:44:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 47c5bdcf997ef9d69318821d388a77875be5ae0f515470baa2dcd2eb3f97ab96
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 61922457
              windows:
                - start: "2026-10-19T20:44:04-07:00"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a1576f-cd4b-72f2-95dc-3dced4835dcb
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 23fc1ec80a2c7363b69d5856a50396aa7a45e9d207a1dd751a6c87528f3c9d27
              started: "2026-10-20T03:51:48.038Z"
              spawned: false
              bytes: 623919
              windows:
                - start: "2026-10-20T03:51:48.038Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15771-61e1-75d0-b25a-f0c092bce1c6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 215123b95afe9271ee0f3e822fad647307f194654c0d10b880870637810846a4
              started: "2026-10-20T03:53:31.517Z"
              spawned: false
              bytes: 426064
              windows:
                - start: "2026-10-20T03:53:31.517Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15772-a7f0-7a71-8179-40bb6f026d64
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c3ebdafbf2992b50f68053cf01ef42e285cdb4cdfa68df37001118898abdbf92
              started: "2026-10-20T03:54:54.909Z"
              spawned: false
              bytes: 452494
              windows:
                - start: "2026-10-20T03:54:54.909Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15773-30ba-7fd1-a14d-117b4947d2b0
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f48d52c254c920ccc35103a443c2bc82a899caf33c9b5aac72b685559225e762
              started: "2026-10-20T03:55:29.867Z"
              spawned: false
              bytes: 413603
              windows:
                - start: "2026-10-20T03:55:29.867Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15773-e28b-7e71-a28e-1412dd5a3698
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fddacb7f6bd07bb2cacf33e86e19ba1f8cd39d89187255d4a77109fd4a853290
              started: "2026-10-20T03:56:15.104Z"
              spawned: true
              bytes: 463283
              windows:
                - start: "2026-10-20T03:56:15.104Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15774-1140-78d0-a327-5734dca0eab7
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: c812386785b161e27ff3f425058819952060a75457251b20030e27f2e13acb44
              started: "2026-10-20T03:56:27.037Z"
              spawned: true
              bytes: 437362
              windows:
                - start: "2026-10-20T03:56:27.037Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15775-fbd5-7101-9ac2-8fbdc812ee2c
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 8071fc76f40bba1456151fe7bbb8316fd0951386a52f7e96fa838fb3cfd6c63d
              started: "2026-10-20T03:58:32.65Z"
              spawned: true
              bytes: 599209
              windows:
                - start: "2026-10-20T03:58:32.65Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15a7c-66b4-7112-a945-614b32b4bdf5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: e8d805037a9a57cdcb82217628ffe1d0fecfde0a2e07b5c4c764250e7f64a745
              started: "2026-10-20T18:04:24.804Z"
              spawned: true
              bytes: 558075
              windows:
                - start: "2026-10-20T18:04:24.804Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15a7c-8cd8-75c1-9218-94721e444c0b
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 69a6f68e663659a26a052b19047f155a5445ee4ff4f09c76e74daa739b47d02d
              started: "2026-10-20T18:04:34.548Z"
              spawned: true
              bytes: 1123591
              windows:
                - start: "2026-10-20T18:04:34.548Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15a8a-dfb4-76d0-a7ab-68ee9dc3509e
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: bd5a3caaf31490873256ebbbec90ef53cae5a7d39948c9dd92f342c892abc475
              started: "2026-10-20T18:20:13.203Z"
              spawned: true
              bytes: 1122451
              windows:
                - start: "2026-10-20T18:20:13.203Z"
                  end: "2026-10-20T14:22:04-07:00"
            - harness: codex
              id: 01a15a8b-0713-7060-b1ea-6fcb20ec6af3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 8bbe2f18da277901b9162f79004f2fe3a6f29e968045cf5a93982e00990f7aca
              started: "2026-10-20T18:20:23.271Z"
              spawned: true
              bytes: 648291
              windows:
                - start: "2026-10-20T18:20:23.271Z"
                  end: "2026-10-20T14:22:04-07:00"
        queue_waits:
            - attempt_id: 9ec35740f7e8bf2624140cf43048b1ab
              slot: pair:6
              record_digest: 8eace18943ef8227448ce688741bf5630600c92f546f458998ab19fdd90788b9
              start: "2026-10-20T18:30:48.677885Z"
              end: "2026-10-20T19:23:09.815282Z"
              queue_seconds: 3141.137397
              raw_seconds: 3141.137397
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 86.39396666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T11:11:37-07:00"
              finalized: "2026-10-20T11:14:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T11:16:40-07:00"
              finalized: "2026-10-20T11:19:30-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-20T14:28:44-07:00"
      dispatched: "2026-10-20T14:23:09-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: a659b2823d24ecdf08b1e9ac8bf7d7924da1d77b
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ed9ef26c-a0cc-435d-96e6-95cb51b96af4
            repository: github.com/xianxu/pair
            issue_id: "000706"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/2bc08e3a7d57cd53
            reviewed_head: edad1fb9759600c2ad25aceb49e787dce7deffb7
            slot: pair:6
            requested_at: "2026-10-20T21:30:05.105607Z"
          phase: finished
          pid: 33524
          started_at: "2026-10-20T21:30:05.23475Z"
          finished_at: "2026-10-20T21:33:26.931922Z"
          session_id: 01a15b38-b579-7530-9670-537a4357ad0b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: edad1fb9759600c2ad25aceb49e787dce7deffb7
        tracker_head: 49f36c76eb883d391d0e8f77149356430d908251
        branch_point: 72a25ac7b601f005fa3199f2622299794a885da3
        since: "2026-10-19T20:44:04-07:00"
        until: "2026-10-20T14:28:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T20:44:04-07:00"
              end: "2026-10-20T14:28:45-07:00"
              began: "2026-10-19T20:44:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 812790db33854458a4db7ff6b58d68cd4eb8d970c43167a189a9d96baea5e079
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 62232323
              windows:
                - start: "2026-10-19T20:44:04-07:00"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a1576f-cd4b-72f2-95dc-3dced4835dcb
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 23fc1ec80a2c7363b69d5856a50396aa7a45e9d207a1dd751a6c87528f3c9d27
              started: "2026-10-20T03:51:48.038Z"
              spawned: false
              bytes: 623919
              windows:
                - start: "2026-10-20T03:51:48.038Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15771-61e1-75d0-b25a-f0c092bce1c6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 215123b95afe9271ee0f3e822fad647307f194654c0d10b880870637810846a4
              started: "2026-10-20T03:53:31.517Z"
              spawned: false
              bytes: 426064
              windows:
                - start: "2026-10-20T03:53:31.517Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15772-a7f0-7a71-8179-40bb6f026d64
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c3ebdafbf2992b50f68053cf01ef42e285cdb4cdfa68df37001118898abdbf92
              started: "2026-10-20T03:54:54.909Z"
              spawned: false
              bytes: 452494
              windows:
                - start: "2026-10-20T03:54:54.909Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15773-30ba-7fd1-a14d-117b4947d2b0
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f48d52c254c920ccc35103a443c2bc82a899caf33c9b5aac72b685559225e762
              started: "2026-10-20T03:55:29.867Z"
              spawned: false
              bytes: 413603
              windows:
                - start: "2026-10-20T03:55:29.867Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15773-e28b-7e71-a28e-1412dd5a3698
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fddacb7f6bd07bb2cacf33e86e19ba1f8cd39d89187255d4a77109fd4a853290
              started: "2026-10-20T03:56:15.104Z"
              spawned: true
              bytes: 463283
              windows:
                - start: "2026-10-20T03:56:15.104Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15774-1140-78d0-a327-5734dca0eab7
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: c812386785b161e27ff3f425058819952060a75457251b20030e27f2e13acb44
              started: "2026-10-20T03:56:27.037Z"
              spawned: true
              bytes: 437362
              windows:
                - start: "2026-10-20T03:56:27.037Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15775-fbd5-7101-9ac2-8fbdc812ee2c
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 8071fc76f40bba1456151fe7bbb8316fd0951386a52f7e96fa838fb3cfd6c63d
              started: "2026-10-20T03:58:32.65Z"
              spawned: true
              bytes: 599209
              windows:
                - start: "2026-10-20T03:58:32.65Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15a7c-66b4-7112-a945-614b32b4bdf5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: e8d805037a9a57cdcb82217628ffe1d0fecfde0a2e07b5c4c764250e7f64a745
              started: "2026-10-20T18:04:24.804Z"
              spawned: true
              bytes: 558075
              windows:
                - start: "2026-10-20T18:04:24.804Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15a7c-8cd8-75c1-9218-94721e444c0b
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 69a6f68e663659a26a052b19047f155a5445ee4ff4f09c76e74daa739b47d02d
              started: "2026-10-20T18:04:34.548Z"
              spawned: true
              bytes: 1123591
              windows:
                - start: "2026-10-20T18:04:34.548Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15a8a-dfb4-76d0-a7ab-68ee9dc3509e
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: bd5a3caaf31490873256ebbbec90ef53cae5a7d39948c9dd92f342c892abc475
              started: "2026-10-20T18:20:13.203Z"
              spawned: true
              bytes: 1122451
              windows:
                - start: "2026-10-20T18:20:13.203Z"
                  end: "2026-10-20T14:28:45-07:00"
            - harness: codex
              id: 01a15a8b-0713-7060-b1ea-6fcb20ec6af3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 8bbe2f18da277901b9162f79004f2fe3a6f29e968045cf5a93982e00990f7aca
              started: "2026-10-20T18:20:23.271Z"
              spawned: true
              bytes: 648291
              windows:
                - start: "2026-10-20T18:20:23.271Z"
                  end: "2026-10-20T14:28:45-07:00"
        queue_waits:
            - attempt_id: 9ec35740f7e8bf2624140cf43048b1ab
              slot: pair:6
              record_digest: 8eace18943ef8227448ce688741bf5630600c92f546f458998ab19fdd90788b9
              start: "2026-10-20T18:30:48.677885Z"
              end: "2026-10-20T19:23:09.815282Z"
              queue_seconds: 3141.137397
              raw_seconds: 3141.137397
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 86.39396666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T11:11:37-07:00"
              finalized: "2026-10-20T11:14:07-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T11:16:40-07:00"
              finalized: "2026-10-20T11:19:30-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-20T14:23:09-07:00"
              finalized: "2026-10-20T14:28:44-07:00"
              finalizing: true
      "n": 4
      timestamp: "2026-10-20T14:33:28-07:00"
      dispatched: "2026-10-20T14:30:02-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: rclone_folders.go:161 retains confirmed and uncertain capacity reservations. Both successive-year cases and the uncertain-month regression pass on head and fail when the reservation update is removed through a temporary Go overlay.
          round: 4
        - id: BR-2
          disposition: addressed
          note: README.md:2086 now documents UTC month destinations, legacy conflict refusal, per-folder batches and the shared 4 GiB budget; these match drive_months.go and drive.go.
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: edad1fb9759600c2ad25aceb49e787dce7deffb7
      blocked: false
---

# Gate ledger — pair#706 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T11:14:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `remote-capacity-reservation` (kind: defect) Year creation reuses stale root capacity and can exceed the listing limit
  cmd/internal/historyarchive/drive_months.go:199-202 passes the original root inventory to every EnsureChild call without accounting for newly created years. With 99,999 root entries and pending bundles in two absent years, both mkdir attempts are admitted; the second creates entry 100,001, whose confirmation and subsequent root listings fail DecodeFlatInventory's limit. ARCH-CONSTRAINTS and ARCH-ORDER: carry forward confirmed parent observations and reserve capacity for potentially completed creates. Add a coordinator regression covering successive year creations at capacity, including an uncertain first outcome.
- **BR-2** [Important] `user-surface-documentation` (kind: defect) README update appears missing for month-folder uploads and per-folder batching
  README.md:2009 still describes one upload batch, while M1 routes new pairs to YYYY/MM and runs a batch per active physical folder. README.md is unchanged in the reviewed range despite this new file-tree convention and the boundary's docs gate. Update the current upload description, legacy behavior, and shared byte budget now; document M2 recovery features when delivered.

## Round 2 — 2026-10-20T11:19:30-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — rclone_folders.go:161-175 reserves capacity after confirmed or uncertain creates; drive_months.go:201,218 applies it to both hierarchy levels. The year-capacity and uncertain month-capacity regressions pass at head. Bypassing the helper in a disposable copy makes all three cases fail; the copy was removed.
- BR-2 — addressed — README.md:2009 now documents YYYY/MM uploads, retained legacy pairs, serial physical-folder batches, and the shared 4 GiB budget. The pinned before/after passage agrees with drive_months.go routing and runFolder execution.

## Round 3 — 2026-10-20T14:28:44-07:00 (codex) — passed

## Round 4 — 2026-10-20T14:33:28-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — rclone_folders.go:161 retains confirmed and uncertain capacity reservations. Both successive-year cases and the uncertain-month regression pass on head and fail when the reservation update is removed through a temporary Go overlay.
- BR-2 — addressed — README.md:2086 now documents UTC month destinations, legacy conflict refusal, per-folder batches and the shared 4 GiB budget; these match drive_months.go and drive.go.

## Open findings

(none — every finding has been disposed)
