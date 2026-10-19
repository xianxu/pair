---
gate: boundary-review
issue: 688
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1835dbc8-4b3d-499a-844f-d3953869481e
            repository: github.com/xianxu/pair
            issue_id: "000688"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9f3934c902e40db2
            reviewed_head: 73ef6ffca269065079768347a0cc32efc7d2c3d6
            slot: pair:7
            requested_at: "2026-10-19T10:07:19.088536Z"
          phase: finished
          pid: 21972
          started_at: "2026-10-19T10:07:19.289723Z"
          finished_at: "2026-10-19T10:09:56.54869Z"
          session_id: 01a153a1-432f-77e2-b00c-17997d94ad49
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 73ef6ffca269065079768347a0cc32efc7d2c3d6
        tracker_head: b4442c77958e5a3c256a7f235322fb4751e1bf95
        branch_point: a6b6973dcf7dab9d192605c4f647fefc9d5d4981
        since: "2026-10-19T01:53:02-07:00"
        until: "2026-10-19T03:06:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T01:53:02-07:00"
              end: "2026-10-19T03:06:04-07:00"
              began: "2026-10-19T01:53:02-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 32ac94b089e5d765db644a3bd7cd98acfd56cf59268f393e77c05feaa957dc18
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 153287442
              windows:
                - start: "2026-10-19T01:53:02-07:00"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a1535d-c093-7563-916a-4aae8e4ff82e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 0b2db4505bda0736c61bf9ba6336c72d1f73cb376382d611413307fe4b26b71f
              started: "2026-10-19T08:53:35.602Z"
              spawned: true
              bytes: 1021999
              windows:
                - start: "2026-10-19T08:53:35.602Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a1536f-406a-7bb2-ba63-ed4b95d46c1d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c7ab9abb6789f7a0448d47f08bba42bd2de6f1958d17afb774a55eaf5cdfaeb
              started: "2026-10-19T09:12:44.28Z"
              spawned: false
              bytes: 671753
              windows:
                - start: "2026-10-19T09:12:44.28Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a15374-ce65-7651-b72b-80b6ffeb3301
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 029b76addae97b0db7e2cf4364a7e0c5cb3c9e61c7ae3f832a1051a892f32069
              started: "2026-10-19T09:18:47.34Z"
              spawned: false
              bytes: 441852
              windows:
                - start: "2026-10-19T09:18:47.34Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a15376-9926-7cf1-b91c-bb26e288a54d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a9af8e9d0a82ee1cdd44acd3dd3b7b7b54989466585806e39acf3ecc450284cb
              started: "2026-10-19T09:20:44.682Z"
              spawned: false
              bytes: 457304
              windows:
                - start: "2026-10-19T09:20:44.682Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a15376-f5eb-7b13-9a62-d56820a6e716
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43d61d9388c25365355eec02df18d2fb5c620ca8ba0906fcdbe279e249500323
              started: "2026-10-19T09:21:08.919Z"
              spawned: false
              bytes: 434844
              windows:
                - start: "2026-10-19T09:21:08.919Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a15378-17ad-7542-a155-44719bc8c46d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 78074bfd53b6381e9627cc3ecfab36e43ec6196ed6bc6ecf9eca5ac7061eead0
              started: "2026-10-19T09:22:21.861Z"
              spawned: true
              bytes: 800204
              windows:
                - start: "2026-10-19T09:22:21.861Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a15379-6eff-7841-a471-f6abd97c5d42
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 0845b68d392656fe95038be06a4d7371ceee8f21154aa086bb01f5d9ca16d86d
              started: "2026-10-19T09:23:49.734Z"
              spawned: true
              bytes: 437481
              windows:
                - start: "2026-10-19T09:23:49.734Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a1538a-9010-73c3-9d49-10f6c9c49119
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: bde7bc5fa4a69a8e4a4311408fc44cc486d22e80a48c719e3f608ffbda87e30e
              started: "2026-10-19T09:42:32.991Z"
              spawned: false
              bytes: 482721
              windows:
                - start: "2026-10-19T09:42:32.991Z"
                  end: "2026-10-19T03:06:04-07:00"
            - harness: codex
              id: 01a1538b-00d7-7913-8489-0eb452f6a06f
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6c59bfdf82b4d771d7a056f2165386cebb08dc816ab8c8e1b1c88898a5d6dec4
              started: "2026-10-19T09:43:01.993Z"
              spawned: false
              bytes: 407988
              windows:
                - start: "2026-10-19T09:43:01.993Z"
                  end: "2026-10-19T03:06:04-07:00"
      "n": 1
      timestamp: "2026-10-19T03:09:58-07:00"
      dispatched: "2026-10-19T03:07:15-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Publication failure strands later validated databases
          detail: cmd/internal/historyarchive/continuous.go:240-244 returns from archiveRoot after any ordinary publication error. With multiple fresh databases, a file-specific encoding or bundle-conflict failure therefore prevents later databases from publishing; their summaries remain failed with “snapshot validated; publication pending” after the pass has ended. This contradicts the plan’s partial-failure isolation contract (line 90; ARCH-PURPOSE). Accumulate per-database publication errors and continue remaining snapshot requests unless the context is cancelled. Add a production-pass regression where the first publication fails and a later database archives successfully, with accurate final summaries and an overall error.
          family: per-database-failure-isolation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7ea0d15c-2165-42a1-af19-aa66ba5b0ca8
            repository: github.com/xianxu/pair
            issue_id: "000688"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/6d197c0735f85358
            reviewed_head: ba8519c1beebeb3961cc0891f99515886550cd05
            slot: pair:7
            requested_at: "2026-10-19T10:17:41.607821Z"
          phase: finished
          pid: 99585
          started_at: "2026-10-19T10:17:41.840006Z"
          finished_at: "2026-10-19T10:20:46.381348Z"
          session_id: 01a153aa-c2a5-7c91-95eb-63c224057db6
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: ba8519c1beebeb3961cc0891f99515886550cd05
        tracker_head: 475e97a7241987d9e2710d69b944651d12202f7a
        branch_point: a6b6973dcf7dab9d192605c4f647fefc9d5d4981
        since: "2026-10-19T01:53:02-07:00"
        until: "2026-10-19T03:15:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T01:53:02-07:00"
              end: "2026-10-19T03:15:54-07:00"
              began: "2026-10-19T01:53:02-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: efaf0486f8813b6a6ee50b4715e1b52a417b759c6b824bb7cca22325785fb017
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 154776426
              windows:
                - start: "2026-10-19T01:53:02-07:00"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a1535d-c093-7563-916a-4aae8e4ff82e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 25746de8c949ef1c192a4d49d4f546a1f13262ac56c52cf343425bc6dd7d64cb
              started: "2026-10-19T08:53:35.602Z"
              spawned: true
              bytes: 1280740
              windows:
                - start: "2026-10-19T08:53:35.602Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a1536f-406a-7bb2-ba63-ed4b95d46c1d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c7ab9abb6789f7a0448d47f08bba42bd2de6f1958d17afb774a55eaf5cdfaeb
              started: "2026-10-19T09:12:44.28Z"
              spawned: false
              bytes: 671753
              windows:
                - start: "2026-10-19T09:12:44.28Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a15374-ce65-7651-b72b-80b6ffeb3301
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 029b76addae97b0db7e2cf4364a7e0c5cb3c9e61c7ae3f832a1051a892f32069
              started: "2026-10-19T09:18:47.34Z"
              spawned: false
              bytes: 441852
              windows:
                - start: "2026-10-19T09:18:47.34Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a15376-9926-7cf1-b91c-bb26e288a54d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a9af8e9d0a82ee1cdd44acd3dd3b7b7b54989466585806e39acf3ecc450284cb
              started: "2026-10-19T09:20:44.682Z"
              spawned: false
              bytes: 457304
              windows:
                - start: "2026-10-19T09:20:44.682Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a15376-f5eb-7b13-9a62-d56820a6e716
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43d61d9388c25365355eec02df18d2fb5c620ca8ba0906fcdbe279e249500323
              started: "2026-10-19T09:21:08.919Z"
              spawned: false
              bytes: 434844
              windows:
                - start: "2026-10-19T09:21:08.919Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a15378-17ad-7542-a155-44719bc8c46d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 78074bfd53b6381e9627cc3ecfab36e43ec6196ed6bc6ecf9eca5ac7061eead0
              started: "2026-10-19T09:22:21.861Z"
              spawned: true
              bytes: 800204
              windows:
                - start: "2026-10-19T09:22:21.861Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a15379-6eff-7841-a471-f6abd97c5d42
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 0845b68d392656fe95038be06a4d7371ceee8f21154aa086bb01f5d9ca16d86d
              started: "2026-10-19T09:23:49.734Z"
              spawned: true
              bytes: 437481
              windows:
                - start: "2026-10-19T09:23:49.734Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a1538a-9010-73c3-9d49-10f6c9c49119
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: bde7bc5fa4a69a8e4a4311408fc44cc486d22e80a48c719e3f608ffbda87e30e
              started: "2026-10-19T09:42:32.991Z"
              spawned: false
              bytes: 482721
              windows:
                - start: "2026-10-19T09:42:32.991Z"
                  end: "2026-10-19T03:15:54-07:00"
            - harness: codex
              id: 01a1538b-00d7-7913-8489-0eb452f6a06f
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6c59bfdf82b4d771d7a056f2165386cebb08dc816ab8c8e1b1c88898a5d6dec4
              started: "2026-10-19T09:43:01.993Z"
              spawned: false
              bytes: 407988
              windows:
                - start: "2026-10-19T09:43:01.993Z"
                  end: "2026-10-19T03:15:54-07:00"
        cuts:
            - dispatched: "2026-10-19T03:07:15-07:00"
              finalized: "2026-10-19T03:09:58-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-19T03:20:49-07:00"
      dispatched: "2026-10-19T03:17:38-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: continuous_snapshot_integration_test.go:70 exercises successive publication failures, later success, aggregate errors, terminal summaries and cancellation. Restoring the early return in a disposable checkout makes this regression fail; pinned-head tests pass.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: Cross-kind staging overlap can overwrite another native source
          detail: cmd/internal/historyarchive/continuous.go:71-77 checks each native root only against staging for that same kind. With snapshots/codex-database symlinked to home/.claude, both roots pass validation, then sqlite_snapshot.go:182 replaces home/.claude/state.sqlite. A disposable production-pass regression observed its original bytes replaced by "version 0". ARCH-SECURE and ARCH-PURPOSE require every snapshot output to be disjoint from every native source, including filtered kinds. Validate the complete source/output cross-product before mutation and add cross-kind alias and direct-overlap regressions.
          family: native-source-output-isolation
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 750bbb9d-4948-4f44-9ad1-ffede3dfb3b8
            repository: github.com/xianxu/pair
            issue_id: "000688"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/1f87029a67631ed8
            reviewed_head: f4b39f45d45569edd2f716ef9cd5e2efc2010d91
            slot: pair:7
            requested_at: "2026-10-19T10:24:25.320054Z"
          phase: finished
          pid: 86319
          started_at: "2026-10-19T10:24:25.496258Z"
          finished_at: "2026-10-19T10:27:12.378372Z"
          session_id: 01a153b0-eb54-7d31-8683-35ca6ead8065
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: f4b39f45d45569edd2f716ef9cd5e2efc2010d91
        tracker_head: 475e97a7241987d9e2710d69b944651d12202f7a
        branch_point: a6b6973dcf7dab9d192605c4f647fefc9d5d4981
        since: "2026-10-19T01:53:02-07:00"
        until: "2026-10-19T03:23:32-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T01:53:02-07:00"
              end: "2026-10-19T03:23:32-07:00"
              began: "2026-10-19T01:53:02-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ee084358525b98d00dda3d94a7d1521cad377d55c1c93e3aff4eb9b523b8be84
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 155113685
              windows:
                - start: "2026-10-19T01:53:02-07:00"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a1535d-c093-7563-916a-4aae8e4ff82e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 25746de8c949ef1c192a4d49d4f546a1f13262ac56c52cf343425bc6dd7d64cb
              started: "2026-10-19T08:53:35.602Z"
              spawned: true
              bytes: 1280740
              windows:
                - start: "2026-10-19T08:53:35.602Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a1536f-406a-7bb2-ba63-ed4b95d46c1d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c7ab9abb6789f7a0448d47f08bba42bd2de6f1958d17afb774a55eaf5cdfaeb
              started: "2026-10-19T09:12:44.28Z"
              spawned: false
              bytes: 671753
              windows:
                - start: "2026-10-19T09:12:44.28Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a15374-ce65-7651-b72b-80b6ffeb3301
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 029b76addae97b0db7e2cf4364a7e0c5cb3c9e61c7ae3f832a1051a892f32069
              started: "2026-10-19T09:18:47.34Z"
              spawned: false
              bytes: 441852
              windows:
                - start: "2026-10-19T09:18:47.34Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a15376-9926-7cf1-b91c-bb26e288a54d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a9af8e9d0a82ee1cdd44acd3dd3b7b7b54989466585806e39acf3ecc450284cb
              started: "2026-10-19T09:20:44.682Z"
              spawned: false
              bytes: 457304
              windows:
                - start: "2026-10-19T09:20:44.682Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a15376-f5eb-7b13-9a62-d56820a6e716
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43d61d9388c25365355eec02df18d2fb5c620ca8ba0906fcdbe279e249500323
              started: "2026-10-19T09:21:08.919Z"
              spawned: false
              bytes: 434844
              windows:
                - start: "2026-10-19T09:21:08.919Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a15378-17ad-7542-a155-44719bc8c46d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 78074bfd53b6381e9627cc3ecfab36e43ec6196ed6bc6ecf9eca5ac7061eead0
              started: "2026-10-19T09:22:21.861Z"
              spawned: true
              bytes: 800204
              windows:
                - start: "2026-10-19T09:22:21.861Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a15379-6eff-7841-a471-f6abd97c5d42
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 0845b68d392656fe95038be06a4d7371ceee8f21154aa086bb01f5d9ca16d86d
              started: "2026-10-19T09:23:49.734Z"
              spawned: true
              bytes: 437481
              windows:
                - start: "2026-10-19T09:23:49.734Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a1538a-9010-73c3-9d49-10f6c9c49119
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: bde7bc5fa4a69a8e4a4311408fc44cc486d22e80a48c719e3f608ffbda87e30e
              started: "2026-10-19T09:42:32.991Z"
              spawned: false
              bytes: 482721
              windows:
                - start: "2026-10-19T09:42:32.991Z"
                  end: "2026-10-19T03:23:32-07:00"
            - harness: codex
              id: 01a1538b-00d7-7913-8489-0eb452f6a06f
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6c59bfdf82b4d771d7a056f2165386cebb08dc816ab8c8e1b1c88898a5d6dec4
              started: "2026-10-19T09:43:01.993Z"
              spawned: false
              bytes: 407988
              windows:
                - start: "2026-10-19T09:43:01.993Z"
                  end: "2026-10-19T03:23:32-07:00"
        cuts:
            - dispatched: "2026-10-19T03:07:15-07:00"
              finalized: "2026-10-19T03:09:58-07:00"
              finalizing: false
            - dispatched: "2026-10-19T03:17:38-07:00"
              finalized: "2026-10-19T03:20:49-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-19T03:27:13-07:00"
      dispatched: "2026-10-19T03:24:22-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-2
          disposition: addressed
          note: continuous.go:69-88 checks every discovered native root against scratch and every snapshot staging destination, including filtered kinds. Direct-overlap and symlink regressions pass at head; restoring the prior guard in a disposable checkout makes both fail and replaces foreign source bytes with "version 0". The disposable checkout was removed.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: f4b39f45d45569edd2f716ef9cd5e2efc2010d91
      blocked: false
---

# Gate ledger — pair#688 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T03:09:58-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `per-database-failure-isolation` (kind: defect) Publication failure strands later validated databases
  cmd/internal/historyarchive/continuous.go:240-244 returns from archiveRoot after any ordinary publication error. With multiple fresh databases, a file-specific encoding or bundle-conflict failure therefore prevents later databases from publishing; their summaries remain failed with “snapshot validated; publication pending” after the pass has ended. This contradicts the plan’s partial-failure isolation contract (line 90; ARCH-PURPOSE). Accumulate per-database publication errors and continue remaining snapshot requests unless the context is cancelled. Add a production-pass regression where the first publication fails and a later database archives successfully, with accurate final summaries and an overall error.

## Round 2 — 2026-10-19T03:20:49-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — continuous_snapshot_integration_test.go:70 exercises successive publication failures, later success, aggregate errors, terminal summaries and cancellation. Restoring the early return in a disposable checkout makes this regression fail; pinned-head tests pass.

### Raised

- **BR-2** [Critical] `native-source-output-isolation` (kind: defect) Cross-kind staging overlap can overwrite another native source
  cmd/internal/historyarchive/continuous.go:71-77 checks each native root only against staging for that same kind. With snapshots/codex-database symlinked to home/.claude, both roots pass validation, then sqlite_snapshot.go:182 replaces home/.claude/state.sqlite. A disposable production-pass regression observed its original bytes replaced by "version 0". ARCH-SECURE and ARCH-PURPOSE require every snapshot output to be disjoint from every native source, including filtered kinds. Validate the complete source/output cross-product before mutation and add cross-kind alias and direct-overlap regressions.

## Round 3 — 2026-10-19T03:27:13-07:00 (codex) — passed

### Disposed

- BR-2 — addressed — continuous.go:69-88 checks every discovered native root against scratch and every snapshot staging destination, including filtered kinds. Direct-overlap and symlink regressions pass at head; restoring the prior guard in a disposable checkout makes both fail and replaces foreign source bytes with "version 0". The disposable checkout was removed.

## Open findings

(none — every finding has been disposed)
