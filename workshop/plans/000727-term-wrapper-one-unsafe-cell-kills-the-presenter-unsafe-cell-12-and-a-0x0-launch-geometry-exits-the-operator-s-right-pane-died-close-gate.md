---
gate: boundary-review
issue: 727
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 501abd18-d14e-459a-bb1b-6bb3c09076dd
            repository: github.com/xianxu/pair
            issue_id: "000727"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/bcb8e3e2dacdfffa
            reviewed_head: 4617a1608cf2dee3fdd037436587f2806474d031
            slot: pair:7
            requested_at: "2026-10-20T16:20:23.152605Z"
          phase: finished
          pid: 32221
          started_at: "2026-10-20T16:20:23.247057Z"
          finished_at: "2026-10-20T16:23:15.227509Z"
          session_id: 01a15a1d-2b41-70e2-b7e6-f3168ac30bab
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4617a1608cf2dee3fdd037436587f2806474d031
        tracker_head: df13eb8fedf40744fb0c47f289bc8c88a2959702
        branch_point: 8620374573f7a42310ffe6348d7946d5326dd15f
        since: "2026-10-20T08:47:04-07:00"
        until: "2026-10-20T09:16:59-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T08:47:04-07:00"
              end: "2026-10-20T09:16:59-07:00"
              began: "2026-10-20T08:47:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ad60089776fe19eb9a51c3c05174d53a87ea2cda35f8cb55182f14ca4adf2f51
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 197723094
              windows:
                - start: "2026-10-20T08:47:04-07:00"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a159ff-1dfb-7780-bc4e-14af28341ab2
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 4569c28cb6f6f1630f2fce258ea8ee127432faad6e7c34af4049728303aa34d6
              started: "2026-10-20T15:47:34.137Z"
              spawned: true
              bytes: 995168
              windows:
                - start: "2026-10-20T15:47:34.137Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a05-cb5b-7600-bcda-f2d221fa6946
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e4bb165011c00ee3950c2ba7f08e9263e84380ec9dd2d265e04658108d679660
              started: "2026-10-20T15:54:52.502Z"
              spawned: false
              bytes: 607435
              windows:
                - start: "2026-10-20T15:54:52.502Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a08-085e-7d70-a4ec-bf90ed500d8b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ef6be92d140da503ed98c59715b1bd6e7b21437566f9433965c79a8cc49c8f27
              started: "2026-10-20T15:57:18.778Z"
              spawned: false
              bytes: 513210
              windows:
                - start: "2026-10-20T15:57:18.778Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a09-c99f-7271-8bbd-cb205f1cd2b7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c1ae75f8f6627c6a13fa48a83cfc75eb525c928f200fc09a7f3dbecab549120
              started: "2026-10-20T15:59:13.75Z"
              spawned: false
              bytes: 431060
              windows:
                - start: "2026-10-20T15:59:13.75Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0c-31ed-7002-88e3-80248b6d55cf
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d744eec5c6a408404d96ffb7146daad69bae90e695e3853fcb5b5272a7e5a89c
              started: "2026-10-20T16:01:51.488Z"
              spawned: false
              bytes: 443293
              windows:
                - start: "2026-10-20T16:01:51.488Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0c-a319-72c2-99f3-dd301032c08d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e2407cccf92659f8d39d5762efc99529bf4c808c8d7e96e1db5ce8789f37f1d1
              started: "2026-10-20T16:02:20.495Z"
              spawned: false
              bytes: 403720
              windows:
                - start: "2026-10-20T16:02:20.495Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0d-fd4d-7fb1-afbb-a944fe169e13
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f478bca0df2df1407a37014fcd217a26e7c2de92298ab18e2f4159b4fb76bc5a
              started: "2026-10-20T16:03:48.782Z"
              spawned: true
              bytes: 953484
              windows:
                - start: "2026-10-20T16:03:48.782Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0e-2f46-75a0-bebe-7553805d3d33
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b02aed9824c3ae1846adc7f8e9590bc70d9733da19422891dae5ba2fa3a97ded
              started: "2026-10-20T16:04:01.622Z"
              spawned: true
              bytes: 948255
              windows:
                - start: "2026-10-20T16:04:01.622Z"
                  end: "2026-10-20T09:16:59-07:00"
      "n": 1
      timestamp: "2026-10-20T09:23:16-07:00"
      dispatched: "2026-10-20T09:20:21-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Input failures are classified as normal completion and bypass shell recovery
          detail: cmd/internal/termcmd/run.go:430-432 discards the input pump's outcome and calls stopLocked(nil); line 457 then classifies mux.done as RecoveryChildExit. A decoder failure, such as invalid UTF-8, only displays a notice before returning, while read errors are discarded outright. With successful cleanup, recovery receives nil and exits without a fallback shell. Return a typed pump outcome, preserve unexpected failures, and test decoder failure and non-EOF read failure through runShellRecovery. This violates the expanded recovery contract and ARCH-PURPOSE/ARCH-ORDER.
          family: recovery-outcome-classification
          round: 1
        - id: BR-2
          severity: Critical
          title: Recreating the missing terminal moves focus away from the invoking pane
          detail: cmd/internal/layoutcmd/layoutcmd.go:252 creates the pane without --no-focus, contrary to README.md:174's explicit focus-preserving contract. Installed Zellij 0.45.1 documents --no-focus as the option that prevents client focus changes; the new fake likewise marks this pane focused. Add --no-focus and assert that the invoking pane/client retains focus after recovery, so subsequent typing stays in the draft or agent. ARCH-PURPOSE.
          family: shortcut-focus-preservation
          round: 1
        - id: BR-3
          severity: Important
          title: The new native pane-creation behavior has no live conformance coverage
          detail: cmd/internal/layoutcmd/terminal_recovery_test.go:46 exercises only the Python fake and exact argv equality. The range adds no native conformance test for the new --tab-id/--near-current-pane creation path, and existing live creation coverage uses a different downward-split command. Add a disposable native test through the production routing seam that verifies visible creation in the invoking tab and preserved client focus, and include it in scheduled conformance. ARCH-MOCK.
          family: external-fake-conformance
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 101cd5b0-82e0-43eb-8b2a-438e0085f4a9
            repository: github.com/xianxu/pair
            issue_id: "000727"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/d84b248216367ff5
            reviewed_head: 97c32eaad2245af63532d100ac67e53283b268eb
            slot: pair:7
            requested_at: "2026-10-20T16:29:54.970465Z"
          phase: finished
          pid: 258
          started_at: "2026-10-20T16:29:55.162863Z"
          finished_at: "2026-10-20T16:32:32.169783Z"
          session_id: 01a15a25-e5d3-70a3-9842-f8ebd5f87dbf
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 97c32eaad2245af63532d100ac67e53283b268eb
        tracker_head: 4763fd0073db81fd526421609ea00a55cd16fc20
        branch_point: 8620374573f7a42310ffe6348d7946d5326dd15f
        since: "2026-10-20T08:47:04-07:00"
        until: "2026-10-20T09:29:01-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T08:47:04-07:00"
              end: "2026-10-20T09:29:01-07:00"
              began: "2026-10-20T08:47:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6b29f650953e05f5864cc0ced3bc9cf24ba5a3660f910627a7b7e6a3f7648527
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 198332941
              windows:
                - start: "2026-10-20T08:47:04-07:00"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a159ff-1dfb-7780-bc4e-14af28341ab2
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f89118ef5ffbb6f7ba81c7786566acc8ba67a3ddbb397a737eef78a9e62f2f41
              started: "2026-10-20T15:47:34.137Z"
              spawned: true
              bytes: 1433447
              windows:
                - start: "2026-10-20T15:47:34.137Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a05-cb5b-7600-bcda-f2d221fa6946
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e4bb165011c00ee3950c2ba7f08e9263e84380ec9dd2d265e04658108d679660
              started: "2026-10-20T15:54:52.502Z"
              spawned: false
              bytes: 607435
              windows:
                - start: "2026-10-20T15:54:52.502Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a08-085e-7d70-a4ec-bf90ed500d8b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ef6be92d140da503ed98c59715b1bd6e7b21437566f9433965c79a8cc49c8f27
              started: "2026-10-20T15:57:18.778Z"
              spawned: false
              bytes: 513210
              windows:
                - start: "2026-10-20T15:57:18.778Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a09-c99f-7271-8bbd-cb205f1cd2b7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c1ae75f8f6627c6a13fa48a83cfc75eb525c928f200fc09a7f3dbecab549120
              started: "2026-10-20T15:59:13.75Z"
              spawned: false
              bytes: 431060
              windows:
                - start: "2026-10-20T15:59:13.75Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a0c-31ed-7002-88e3-80248b6d55cf
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d744eec5c6a408404d96ffb7146daad69bae90e695e3853fcb5b5272a7e5a89c
              started: "2026-10-20T16:01:51.488Z"
              spawned: false
              bytes: 443293
              windows:
                - start: "2026-10-20T16:01:51.488Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a0c-a319-72c2-99f3-dd301032c08d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e2407cccf92659f8d39d5762efc99529bf4c808c8d7e96e1db5ce8789f37f1d1
              started: "2026-10-20T16:02:20.495Z"
              spawned: false
              bytes: 403720
              windows:
                - start: "2026-10-20T16:02:20.495Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a0d-fd4d-7fb1-afbb-a944fe169e13
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f478bca0df2df1407a37014fcd217a26e7c2de92298ab18e2f4159b4fb76bc5a
              started: "2026-10-20T16:03:48.782Z"
              spawned: true
              bytes: 953484
              windows:
                - start: "2026-10-20T16:03:48.782Z"
                  end: "2026-10-20T09:29:01-07:00"
            - harness: codex
              id: 01a15a0e-2f46-75a0-bebe-7553805d3d33
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 341d723d2600a244584cd69f88d4cc4041162e6f2caa093385ae806dfb686095
              started: "2026-10-20T16:04:01.622Z"
              spawned: true
              bytes: 1126656
              windows:
                - start: "2026-10-20T16:04:01.622Z"
                  end: "2026-10-20T09:29:01-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 1.3874166666666667
        cuts:
            - dispatched: "2026-10-20T09:20:21-07:00"
              finalized: "2026-10-20T09:23:16-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-20T09:32:33-07:00"
      dispatched: "2026-10-20T09:29:52-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: run.go:430-432 propagates typed pump failures into recovery. TestRecoveryOwnsInputPumpEnding passes; disabling propagation in a disposable checkout reproduces missing errors and zero handoffs for decoder and non-EOF read failures.
          round: 2
        - id: BR-2
          disposition: addressed
          note: layoutcmd.go:252 includes --no-focus. Fake and native tests preserve invoking-pane focus; removing the flag makes both fake regression cases fail. The native negative control also demonstrates redirected keyboard input.
          round: 2
        - id: BR-3
          disposition: addressed
          note: TestTerminalRecoveryZellijConformance passed against Zellij 0.45.1 through production routing, verifying same-tab visible creation, actual client focus and foreign-tab isolation. The scheduled native runner requires its execution; all eight runner proof tests passed.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 97c32eaad2245af63532d100ac67e53283b268eb
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 570f3ede-e1f2-40bd-af09-c6bb20481737
            repository: github.com/xianxu/pair
            issue_id: "000727"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a8a85d36a450fe8d
            reviewed_head: 4db6b859d8c617b65dbe087214fde3d242c741a2
            slot: pair:7
            requested_at: "2026-10-20T16:47:05.529827Z"
          phase: finished
          pid: 65843
          started_at: "2026-10-20T16:47:05.679141Z"
          finished_at: "2026-10-20T16:48:26.617976Z"
          session_id: 01a15a35-9f49-7f82-aef8-c7bc1a807cdd
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 4db6b859d8c617b65dbe087214fde3d242c741a2
        tracker_head: 9f9f2a3aaf3e256580403b9a4f52bfa9e80fa4e1
        branch_point: 8620374573f7a42310ffe6348d7946d5326dd15f
        since: "2026-10-20T08:47:04-07:00"
        until: "2026-10-20T09:46:22-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T08:47:04-07:00"
              end: "2026-10-20T09:46:22-07:00"
              began: "2026-10-20T08:47:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 17cdc6b5e41346363a0e734574d2ff3ede626e955ad389b0ae98753693ff4d1b
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 198912925
              windows:
                - start: "2026-10-20T08:47:04-07:00"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a159ff-1dfb-7780-bc4e-14af28341ab2
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f89118ef5ffbb6f7ba81c7786566acc8ba67a3ddbb397a737eef78a9e62f2f41
              started: "2026-10-20T15:47:34.137Z"
              spawned: true
              bytes: 1433447
              windows:
                - start: "2026-10-20T15:47:34.137Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a05-cb5b-7600-bcda-f2d221fa6946
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e4bb165011c00ee3950c2ba7f08e9263e84380ec9dd2d265e04658108d679660
              started: "2026-10-20T15:54:52.502Z"
              spawned: false
              bytes: 607435
              windows:
                - start: "2026-10-20T15:54:52.502Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a08-085e-7d70-a4ec-bf90ed500d8b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ef6be92d140da503ed98c59715b1bd6e7b21437566f9433965c79a8cc49c8f27
              started: "2026-10-20T15:57:18.778Z"
              spawned: false
              bytes: 513210
              windows:
                - start: "2026-10-20T15:57:18.778Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a09-c99f-7271-8bbd-cb205f1cd2b7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c1ae75f8f6627c6a13fa48a83cfc75eb525c928f200fc09a7f3dbecab549120
              started: "2026-10-20T15:59:13.75Z"
              spawned: false
              bytes: 431060
              windows:
                - start: "2026-10-20T15:59:13.75Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a0c-31ed-7002-88e3-80248b6d55cf
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d744eec5c6a408404d96ffb7146daad69bae90e695e3853fcb5b5272a7e5a89c
              started: "2026-10-20T16:01:51.488Z"
              spawned: false
              bytes: 443293
              windows:
                - start: "2026-10-20T16:01:51.488Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a0c-a319-72c2-99f3-dd301032c08d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e2407cccf92659f8d39d5762efc99529bf4c808c8d7e96e1db5ce8789f37f1d1
              started: "2026-10-20T16:02:20.495Z"
              spawned: false
              bytes: 403720
              windows:
                - start: "2026-10-20T16:02:20.495Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a0d-fd4d-7fb1-afbb-a944fe169e13
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f478bca0df2df1407a37014fcd217a26e7c2de92298ab18e2f4159b4fb76bc5a
              started: "2026-10-20T16:03:48.782Z"
              spawned: true
              bytes: 953484
              windows:
                - start: "2026-10-20T16:03:48.782Z"
                  end: "2026-10-20T09:46:22-07:00"
            - harness: codex
              id: 01a15a0e-2f46-75a0-bebe-7553805d3d33
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e222a1a4d5a0b1c5ca0b617183cdf7459c4035601c3fbbca5c9b95efbaddacff
              started: "2026-10-20T16:04:01.622Z"
              spawned: true
              bytes: 1182479
              windows:
                - start: "2026-10-20T16:04:01.622Z"
                  end: "2026-10-20T09:46:22-07:00"
        queue_waits:
            - attempt_id: 649f032a85eeef174f700eda5f2a9264
              slot: pair:7
              record_digest: 80f0cb37c842040e1e65913017ae59a56c49be62a48d233832607cd262c1ed87
              start: "2026-10-20T16:29:22.681558Z"
              end: "2026-10-20T16:40:29.560384Z"
              queue_seconds: 666.878826
              raw_seconds: 666.878826
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 23.414283333333334
        cuts:
            - dispatched: "2026-10-20T09:20:21-07:00"
              finalized: "2026-10-20T09:23:16-07:00"
              finalizing: false
            - dispatched: "2026-10-20T09:29:52-07:00"
              finalized: "2026-10-20T09:32:33-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-20T09:48:27-07:00"
      dispatched: "2026-10-20T09:47:03-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Prior correction remains unchanged; TestRecoveryOwnsInputPumpEnding passes in this review.
          round: 3
        - id: BR-2
          disposition: addressed
          note: The --no-focus correction remains unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes both cases.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Previously accepted native conformance coverage remains unchanged and required by the scheduled runner; all eight runner proof tests pass. Native conformance was not rerun in this narrow review.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 4db6b859d8c617b65dbe087214fde3d242c741a2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b5601226-9169-4860-8db0-285e81ee4121
            repository: github.com/xianxu/pair
            issue_id: "000727"
            recipe: fix-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/2b96cac31e037de8
            reviewed_head: 7bda7379dddfada81f0dd81cd502fecec5c74201
            slot: pair:7
            requested_at: "2026-10-20T17:14:49.873072Z"
          phase: finished
          pid: 35561
          started_at: "2026-10-20T17:14:50.051299Z"
          finished_at: "2026-10-20T17:15:49.49033Z"
          session_id: 01a15a4f-04ba-7080-830e-e11d056f8ef5
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 7bda7379dddfada81f0dd81cd502fecec5c74201
        tracker_head: 41e572d03bfd402fe4025934eb7d6fe16376994f
        branch_point: 8620374573f7a42310ffe6348d7946d5326dd15f
        since: "2026-10-20T08:47:04-07:00"
        until: "2026-10-20T10:13:47-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T08:47:04-07:00"
              end: "2026-10-20T10:13:47-07:00"
              began: "2026-10-20T08:47:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: b3fe0db4fa69702bb8c9aee7f80b358ba50eeee9d4b9bae6765374e0a2e70854
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 199952740
              windows:
                - start: "2026-10-20T08:47:04-07:00"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a159ff-1dfb-7780-bc4e-14af28341ab2
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: cc785bf589504ade03cb0b784efa7b376b6a98e266c05b6aeb56abd9d78a6bf6
              started: "2026-10-20T15:47:34.137Z"
              spawned: true
              bytes: 1567093
              windows:
                - start: "2026-10-20T15:47:34.137Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a05-cb5b-7600-bcda-f2d221fa6946
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e4bb165011c00ee3950c2ba7f08e9263e84380ec9dd2d265e04658108d679660
              started: "2026-10-20T15:54:52.502Z"
              spawned: false
              bytes: 607435
              windows:
                - start: "2026-10-20T15:54:52.502Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a08-085e-7d70-a4ec-bf90ed500d8b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ef6be92d140da503ed98c59715b1bd6e7b21437566f9433965c79a8cc49c8f27
              started: "2026-10-20T15:57:18.778Z"
              spawned: false
              bytes: 513210
              windows:
                - start: "2026-10-20T15:57:18.778Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a09-c99f-7271-8bbd-cb205f1cd2b7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c1ae75f8f6627c6a13fa48a83cfc75eb525c928f200fc09a7f3dbecab549120
              started: "2026-10-20T15:59:13.75Z"
              spawned: false
              bytes: 431060
              windows:
                - start: "2026-10-20T15:59:13.75Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a0c-31ed-7002-88e3-80248b6d55cf
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d744eec5c6a408404d96ffb7146daad69bae90e695e3853fcb5b5272a7e5a89c
              started: "2026-10-20T16:01:51.488Z"
              spawned: false
              bytes: 443293
              windows:
                - start: "2026-10-20T16:01:51.488Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a0c-a319-72c2-99f3-dd301032c08d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e2407cccf92659f8d39d5762efc99529bf4c808c8d7e96e1db5ce8789f37f1d1
              started: "2026-10-20T16:02:20.495Z"
              spawned: false
              bytes: 403720
              windows:
                - start: "2026-10-20T16:02:20.495Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a0d-fd4d-7fb1-afbb-a944fe169e13
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f478bca0df2df1407a37014fcd217a26e7c2de92298ab18e2f4159b4fb76bc5a
              started: "2026-10-20T16:03:48.782Z"
              spawned: true
              bytes: 953484
              windows:
                - start: "2026-10-20T16:03:48.782Z"
                  end: "2026-10-20T10:13:47-07:00"
            - harness: codex
              id: 01a15a0e-2f46-75a0-bebe-7553805d3d33
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e222a1a4d5a0b1c5ca0b617183cdf7459c4035601c3fbbca5c9b95efbaddacff
              started: "2026-10-20T16:04:01.622Z"
              spawned: true
              bytes: 1182479
              windows:
                - start: "2026-10-20T16:04:01.622Z"
                  end: "2026-10-20T10:13:47-07:00"
        queue_waits:
            - attempt_id: 649f032a85eeef174f700eda5f2a9264
              slot: pair:7
              record_digest: 80f0cb37c842040e1e65913017ae59a56c49be62a48d233832607cd262c1ed87
              start: "2026-10-20T16:29:22.681558Z"
              end: "2026-10-20T16:40:29.560384Z"
              queue_seconds: 666.878826
              raw_seconds: 666.878826
              exact: true
            - attempt_id: 670c701f57d279ebc6057ac544689b9d
              slot: pair:7
              record_digest: 2a918fe3cde6d140922eb51af19535d4606ba1b73241ad43616ead3bd90e5a8e
              start: "2026-10-20T16:46:46.53901Z"
              end: "2026-10-20T17:01:07.277694Z"
              queue_seconds: 860.738684
              raw_seconds: 860.738684
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.503233333333334
        cuts:
            - dispatched: "2026-10-20T09:20:21-07:00"
              finalized: "2026-10-20T09:23:16-07:00"
              finalizing: false
            - dispatched: "2026-10-20T09:29:52-07:00"
              finalized: "2026-10-20T09:32:33-07:00"
              finalizing: true
            - dispatched: "2026-10-20T09:47:03-07:00"
              finalized: "2026-10-20T09:48:27-07:00"
              finalizing: true
      "n": 4
      timestamp: "2026-10-20T10:15:50-07:00"
      dispatched: "2026-10-20T10:14:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Previously accepted correction is unchanged; TestRecoveryOwnsInputPumpEnding passes.
          round: 4
        - id: BR-2
          disposition: addressed
          note: The --no-focus correction is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Previously accepted native coverage remains unchanged and required by the scheduled runner; all eight runner proof tests pass. Native conformance was not rerun in this narrow review.
          round: 4
      recipe: fix-review
      isolation: fenced
      reviewed: 7bda7379dddfada81f0dd81cd502fecec5c74201
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3f58a2b8-57c5-4b73-b0fa-872f564db048
            repository: github.com/xianxu/pair
            issue_id: "000727"
            recipe: fix-review
            round: 5
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/3b93d00131395836
            reviewed_head: b5291a95ec1267728faa50794d13bb8396565921
            slot: pair:7
            requested_at: "2026-10-20T18:46:48.06569Z"
          phase: finished
          pid: 79992
          started_at: "2026-10-20T18:46:48.193154Z"
          finished_at: "2026-10-20T18:47:23.298268Z"
          session_id: 01a15aa3-37b9-7ad1-a889-e803f9fafcec
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: b5291a95ec1267728faa50794d13bb8396565921
        tracker_head: d050860cfe1508a4fc86936e72605e787340c7ca
        branch_point: 8620374573f7a42310ffe6348d7946d5326dd15f
        since: "2026-10-20T08:47:04-07:00"
        until: "2026-10-20T11:46:13-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T08:47:04-07:00"
              end: "2026-10-20T11:46:13-07:00"
              began: "2026-10-20T08:47:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a8dd444bfc59e1c4395e9d300b9d714be3db5b9b73a9533a07c6d833e6756a74
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 202015302
              windows:
                - start: "2026-10-20T08:47:04-07:00"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a159ff-1dfb-7780-bc4e-14af28341ab2
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: cc785bf589504ade03cb0b784efa7b376b6a98e266c05b6aeb56abd9d78a6bf6
              started: "2026-10-20T15:47:34.137Z"
              spawned: true
              bytes: 1567093
              windows:
                - start: "2026-10-20T15:47:34.137Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a05-cb5b-7600-bcda-f2d221fa6946
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e4bb165011c00ee3950c2ba7f08e9263e84380ec9dd2d265e04658108d679660
              started: "2026-10-20T15:54:52.502Z"
              spawned: false
              bytes: 607435
              windows:
                - start: "2026-10-20T15:54:52.502Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a08-085e-7d70-a4ec-bf90ed500d8b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ef6be92d140da503ed98c59715b1bd6e7b21437566f9433965c79a8cc49c8f27
              started: "2026-10-20T15:57:18.778Z"
              spawned: false
              bytes: 513210
              windows:
                - start: "2026-10-20T15:57:18.778Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a09-c99f-7271-8bbd-cb205f1cd2b7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c1ae75f8f6627c6a13fa48a83cfc75eb525c928f200fc09a7f3dbecab549120
              started: "2026-10-20T15:59:13.75Z"
              spawned: false
              bytes: 431060
              windows:
                - start: "2026-10-20T15:59:13.75Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a0c-31ed-7002-88e3-80248b6d55cf
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d744eec5c6a408404d96ffb7146daad69bae90e695e3853fcb5b5272a7e5a89c
              started: "2026-10-20T16:01:51.488Z"
              spawned: false
              bytes: 443293
              windows:
                - start: "2026-10-20T16:01:51.488Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a0c-a319-72c2-99f3-dd301032c08d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e2407cccf92659f8d39d5762efc99529bf4c808c8d7e96e1db5ce8789f37f1d1
              started: "2026-10-20T16:02:20.495Z"
              spawned: false
              bytes: 403720
              windows:
                - start: "2026-10-20T16:02:20.495Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a0d-fd4d-7fb1-afbb-a944fe169e13
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f478bca0df2df1407a37014fcd217a26e7c2de92298ab18e2f4159b4fb76bc5a
              started: "2026-10-20T16:03:48.782Z"
              spawned: true
              bytes: 953484
              windows:
                - start: "2026-10-20T16:03:48.782Z"
                  end: "2026-10-20T11:46:13-07:00"
            - harness: codex
              id: 01a15a0e-2f46-75a0-bebe-7553805d3d33
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e222a1a4d5a0b1c5ca0b617183cdf7459c4035601c3fbbca5c9b95efbaddacff
              started: "2026-10-20T16:04:01.622Z"
              spawned: true
              bytes: 1182479
              windows:
                - start: "2026-10-20T16:04:01.622Z"
                  end: "2026-10-20T11:46:13-07:00"
        queue_waits:
            - attempt_id: 649f032a85eeef174f700eda5f2a9264
              slot: pair:7
              record_digest: 80f0cb37c842040e1e65913017ae59a56c49be62a48d233832607cd262c1ed87
              start: "2026-10-20T16:29:22.681558Z"
              end: "2026-10-20T16:40:29.560384Z"
              queue_seconds: 666.878826
              raw_seconds: 666.878826
              exact: true
            - attempt_id: 670c701f57d279ebc6057ac544689b9d
              slot: pair:7
              record_digest: 2a918fe3cde6d140922eb51af19535d4606ba1b73241ad43616ead3bd90e5a8e
              start: "2026-10-20T16:46:46.53901Z"
              end: "2026-10-20T17:01:07.277694Z"
              queue_seconds: 860.738684
              raw_seconds: 860.738684
              exact: true
            - attempt_id: 9ef62e46c1b80f4b9eaf5678127803b9
              slot: pair:7
              record_digest: 44c890bfeaaf64f8d3d09e6faaab08acb154f5cbeab473e9d6bf8bf791f4eec5
              start: "2026-10-20T18:04:11.421664Z"
              end: "2026-10-20T18:37:04.963578Z"
              queue_seconds: 1973.541914
              raw_seconds: 1973.541914
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.503233333333334
        cuts:
            - dispatched: "2026-10-20T09:20:21-07:00"
              finalized: "2026-10-20T09:23:16-07:00"
              finalizing: false
            - dispatched: "2026-10-20T09:29:52-07:00"
              finalized: "2026-10-20T09:32:33-07:00"
              finalizing: true
            - dispatched: "2026-10-20T09:47:03-07:00"
              finalized: "2026-10-20T09:48:27-07:00"
              finalizing: true
            - dispatched: "2026-10-20T10:14:47-07:00"
              finalized: "2026-10-20T10:15:50-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-20T11:47:24-07:00"
      dispatched: "2026-10-20T11:46:46-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Previously accepted recovery correction is unchanged; TestRecoveryOwnsInputPumpEnding passes.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Previously accepted focus correction is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Native conformance coverage remains unchanged and required by the scheduled runner; all eight runner tests pass. Native conformance was not rerun.
          round: 5
      recipe: fix-review
      isolation: fenced
      reviewed: b5291a95ec1267728faa50794d13bb8396565921
      blocked: false
---

# Gate ledger — pair#727 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T09:23:16-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `recovery-outcome-classification` Input failures are classified as normal completion and bypass shell recovery
  cmd/internal/termcmd/run.go:430-432 discards the input pump's outcome and calls stopLocked(nil); line 457 then classifies mux.done as RecoveryChildExit. A decoder failure, such as invalid UTF-8, only displays a notice before returning, while read errors are discarded outright. With successful cleanup, recovery receives nil and exits without a fallback shell. Return a typed pump outcome, preserve unexpected failures, and test decoder failure and non-EOF read failure through runShellRecovery. This violates the expanded recovery contract and ARCH-PURPOSE/ARCH-ORDER.
- **BR-2** [Critical] `shortcut-focus-preservation` Recreating the missing terminal moves focus away from the invoking pane
  cmd/internal/layoutcmd/layoutcmd.go:252 creates the pane without --no-focus, contrary to README.md:174's explicit focus-preserving contract. Installed Zellij 0.45.1 documents --no-focus as the option that prevents client focus changes; the new fake likewise marks this pane focused. Add --no-focus and assert that the invoking pane/client retains focus after recovery, so subsequent typing stays in the draft or agent. ARCH-PURPOSE.
- **BR-3** [Important] `external-fake-conformance` The new native pane-creation behavior has no live conformance coverage
  cmd/internal/layoutcmd/terminal_recovery_test.go:46 exercises only the Python fake and exact argv equality. The range adds no native conformance test for the new --tab-id/--near-current-pane creation path, and existing live creation coverage uses a different downward-split command. Add a disposable native test through the production routing seam that verifies visible creation in the invoking tab and preserved client focus, and include it in scheduled conformance. ARCH-MOCK.

## Round 2 — 2026-10-20T09:32:33-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — run.go:430-432 propagates typed pump failures into recovery. TestRecoveryOwnsInputPumpEnding passes; disabling propagation in a disposable checkout reproduces missing errors and zero handoffs for decoder and non-EOF read failures.
- BR-2 — addressed — layoutcmd.go:252 includes --no-focus. Fake and native tests preserve invoking-pane focus; removing the flag makes both fake regression cases fail. The native negative control also demonstrates redirected keyboard input.
- BR-3 — addressed — TestTerminalRecoveryZellijConformance passed against Zellij 0.45.1 through production routing, verifying same-tab visible creation, actual client focus and foreign-tab isolation. The scheduled native runner requires its execution; all eight runner proof tests passed.

## Round 3 — 2026-10-20T09:48:27-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Prior correction remains unchanged; TestRecoveryOwnsInputPumpEnding passes in this review.
- BR-2 — addressed — The --no-focus correction remains unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes both cases.
- BR-3 — addressed — Previously accepted native conformance coverage remains unchanged and required by the scheduled runner; all eight runner proof tests pass. Native conformance was not rerun in this narrow review.

## Round 4 — 2026-10-20T10:15:50-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Previously accepted correction is unchanged; TestRecoveryOwnsInputPumpEnding passes.
- BR-2 — addressed — The --no-focus correction is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
- BR-3 — addressed — Previously accepted native coverage remains unchanged and required by the scheduled runner; all eight runner proof tests pass. Native conformance was not rerun in this narrow review.

## Round 5 — 2026-10-20T11:47:24-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Previously accepted recovery correction is unchanged; TestRecoveryOwnsInputPumpEnding passes.
- BR-2 — addressed — Previously accepted focus correction is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
- BR-3 — addressed — Native conformance coverage remains unchanged and required by the scheduled runner; all eight runner tests pass. Native conformance was not rerun.

## Open findings

(none — every finding has been disposed)
