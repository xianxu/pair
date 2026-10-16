---
gate: boundary-review
issue: 595
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 0312b5fe-d8a9-46d1-868f-608248ac6572
            repository: github.com/xianxu/pair
            issue_id: "000595"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/db47110353867fed
            reviewed_head: 71b211cc49811f3df0676c657e181578765cd7f6
            slot: pair:2
            requested_at: "2026-10-16T10:51:45.547552Z"
          phase: finished
          pid: 22006
          started_at: "2026-10-16T10:51:46.012238Z"
          finished_at: "2026-10-16T11:07:20.651513Z"
          session_id: 9963600e-ec74-4ba4-9605-b9280c0825ca
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 71b211cc49811f3df0676c657e181578765cd7f6
        tracker_head: 975471e76eb89b8e8347dd0cba66b2566c62e1f0
        branch_point: 1e93c3d9e0309d35f9993e3b52f5339480074081
        since: "2026-10-15T19:45:21-07:00"
        until: "2026-10-16T03:50:47-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T19:45:21-07:00"
              end: "2026-10-15T20:45:21-07:00"
              began: "2026-10-15T19:45:21-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-16T02:59:59-07:00"
              end: "2026-10-16T03:50:47-07:00"
              began: "2026-10-16T02:59:59-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4d9aaac1c2280cafce056e7f08e02684a5297091192d5e52ff75916caf8d0efe
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 7128928
              windows:
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T19:45:21-07:00"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a1429e-ebc8-7610-b5df-8a4d6e94c9a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 303856442c5abae077043b1dddf05234698a6272e89f17d4add55b736976a625
              started: "2026-10-16T02:51:13.807Z"
              spawned: true
              bytes: 637944
              windows:
                - start: "2026-10-16T02:51:13.807Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ab-0b8a-7e51-8b96-2bd081084538
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 3d4af4fd9200a90586f21a8d46f3f4562e2bf3d84e7809767ce0b9749ac9e07e
              started: "2026-10-16T03:04:28.545Z"
              spawned: false
              bytes: 605962
              windows:
                - start: "2026-10-16T03:04:28.545Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142ab-8f17-75d0-9d16-9f5cb036d5e3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 38d33ef511e7fc987d122ca62ae9610f13cdfb430ac9ffa7bab67f9d8de0d811
              started: "2026-10-16T03:05:02.084Z"
              spawned: true
              bytes: 850604
              windows:
                - start: "2026-10-16T03:05:02.084Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ad-01a9-7c12-af5c-200906e8cdca
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ad55ba4d74274be3fa6970cbb4955997b27eb5f808a7484d3ad8d07d5c434dcb
              started: "2026-10-16T03:06:37.159Z"
              spawned: false
              bytes: 456151
              windows:
                - start: "2026-10-16T03:06:37.159Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142ae-6330-7783-9f09-31d7ff6d2a7a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b71b546d1e6d10cbbda97b487230c878ffb28afe56dfcb9486d550d6615af80d
              started: "2026-10-16T03:08:07.966Z"
              spawned: false
              bytes: 446084
              windows:
                - start: "2026-10-16T03:08:07.966Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142ae-d92a-7ee2-824b-b23fbbac5e37
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8497a635a85846893095cc0bb14dec6d92791e01a5b3b66f0333f3e18adc7b34
              started: "2026-10-16T03:08:37.905Z"
              spawned: false
              bytes: 397998
              windows:
                - start: "2026-10-16T03:08:37.905Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142af-ad08-7e40-80ec-d227c3009b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 715da457ab79624da6910aa60f79e0029d3a7696674e47abf1a4f7de1c5c6c2d
              started: "2026-10-16T03:09:31.922Z"
              spawned: true
              bytes: 642617
              windows:
                - start: "2026-10-16T03:09:31.922Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142af-eee0-7690-a5d6-7665399ff88b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: c5861c3be06e7a31b005f2628f7729989e532d2bf5bbbcd1032cfaa094eb33aa
              started: "2026-10-16T03:09:48.791Z"
              spawned: true
              bytes: 704235
              windows:
                - start: "2026-10-16T03:09:48.791Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b3-1520-7c92-a180-31e3993bd485
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b94fc705dc5704b435b1465cd05b14ef71067d82623f540f15bb6c5f3b63d24c
              started: "2026-10-16T03:13:15.222Z"
              spawned: true
              bytes: 1266036
              windows:
                - start: "2026-10-16T03:13:15.222Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b8-e0ea-7662-a597-05926edba15b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: f007c9977d91413c421707c79fc5e86901c9e9c9b9109091e54c3174964de81c
              started: "2026-10-16T03:19:35.005Z"
              spawned: true
              bytes: 623158
              windows:
                - start: "2026-10-16T03:19:35.005Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142c6-124f-7491-a52b-beb2cb26f82f
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 9ca992bb23dafdecad96a56dbbb59e90918b18dc1b663443da393b4efe43467b
              started: "2026-10-16T03:33:59.704Z"
              spawned: true
              bytes: 909366
              windows:
                - start: "2026-10-16T03:33:59.704Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142c8-0a54-7ae0-b8e6-c5d900157a3d
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: db2455471dce5c23ba032d6016d44fcec459b83f4dbaaa2dffff17405c4b7a73
              started: "2026-10-16T03:36:08.698Z"
              spawned: true
              bytes: 859979
              windows:
                - start: "2026-10-16T03:36:08.698Z"
                  end: "2026-10-15T20:45:21-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.0299
      "n": 1
      timestamp: "2026-10-16T04:07:39-07:00"
      dispatched: "2026-10-16T03:51:41-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Spec, Done-when and plan Core concepts still require the dropped capability cutover gate
          detail: Plan rows name launcher/native_lock_capability.go and readiness/record.go (absent/unchanged); issue Spec line 186 and Done-when bullet 4 still promise "relaunch older slots first" refusal and legacy-capability tests; the re-plan has no plan Revisions entry and the issue Log says "awaiting OK" while atlas cites TL e93b109d. Update plan/issue to the approved re-plan.
          family: plan-claims-undelivered-surface
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Late binding notice after a restart has started (SIGUSR2) turns the agent restart into a wrapper failure
          detail: masterPump calls Discover after locks.Stop(); Advance rejects "discovery after stop", the error is stored in nativeLockErr and returned before the restartFresh check, so the wrapper exits 1 instead of re-execing. An ID valid for validBindingNotice but rejected by NewKey likewise kills the agent group. Ignore/log rejected notices while stopping and add a wrapper-level test.
          family: stopping-rejection-treated-as-lock-failure
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: ownedGroup always waits the full 1s grace, adding a second to every wrapper exit and restart
          detail: StartStop only skips the timer on ESRCH; with only the unreaped leader left, Darwin returns EPERM (measured on this host) and Linux succeeds against zombies, so group.Wait always blocks 1s before SIGKILL and reap. The test uses a 1ms grace and cannot see it. Finish early once the group is empty.
          family: fixed-grace-instead-of-bound
          round: 1
        - kind: defect
          id: BR-4
          severity: Important
          title: nativeLockHolder synchronously execs sdlc workspace --json (up to 2s) before every agent spawn
          detail: It runs unconditionally, even for non-native wraps and unknown-ID Fresh launches, only to produce a diagnostic label, with no seam or fake (ARCH-CONSTRAINTS/ARCH-MOCK). Resolve lazily when a key is first acquired, from launch environment, behind an injectable seam.
          family: external-probe-on-startup-path
          round: 1
        - kind: defect
          id: BR-5
          severity: Important
          title: No wrapper-level tests for late collision, notice-channel handling or controlled signal/restart exits
          detail: Done-when bullets 2-3 are covered only by the primitives; nothing drives masterPump notice to Discover to StartStop to nativeLockErr, or the SIGTERM/HUP/SIGUSR2 paths in run (ARCH-ORDER), which is how the restart bug above shipped.
          family: wrapper-ordering-untested
          round: 1
        - kind: defect
          id: BR-6
          severity: Minor
          title: atlas/couch.md Owners list still names the deleted launcher/native_lock_capability.go
          family: plan-claims-undelivered-surface
          round: 1
        - kind: defect
          id: BR-7
          severity: Minor
          title: Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
          family: truthful-lock-status-notice
          round: 1
        - kind: defect
          id: BR-8
          severity: Minor
          title: Acquire fails outright on an invalid holder label instead of falling back to the generic label
          family: diagnostic-label-blocks-lock
          round: 1
        - kind: defect
          id: BR-9
          severity: Minor
          title: Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule
          family: shared-identifier-validation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 623c738e-ae9f-42c0-8a4f-2af648ce93b5
            repository: github.com/xianxu/pair
            issue_id: "000595"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/6826bb3b7d0f3a67
            reviewed_head: 7a7b60867800e9c409f91bbd5c126082255655e6
            slot: pair:2
            requested_at: "2026-10-16T11:29:59.843353Z"
          phase: finished
          pid: 44528
          started_at: "2026-10-16T11:29:59.967681Z"
          finished_at: "2026-10-16T11:34:12.076894Z"
          session_id: f8ab1b46-694a-4e05-bc60-8af702feb6c4
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 7a7b60867800e9c409f91bbd5c126082255655e6
        tracker_head: a0bc3e821302c9f54e97aeac5f28ee1dbff6d4b6
        branch_point: 1e93c3d9e0309d35f9993e3b52f5339480074081
        since: "2026-10-15T19:45:21-07:00"
        until: "2026-10-16T04:15:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T19:45:21-07:00"
              end: "2026-10-15T20:45:21-07:00"
              began: "2026-10-15T19:45:21-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-16T02:59:59-07:00"
              end: "2026-10-16T04:15:28-07:00"
              began: "2026-10-16T02:59:59-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 07c8abbf9fa0d33a0b2f48a6ef4d48d5bf91ab42e96c5b58eb537ccf2cf5dc57
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 7535807
              windows:
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:15:28-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T19:45:21-07:00"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:15:28-07:00"
            - harness: codex
              id: 01a1429e-ebc8-7610-b5df-8a4d6e94c9a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 303856442c5abae077043b1dddf05234698a6272e89f17d4add55b736976a625
              started: "2026-10-16T02:51:13.807Z"
              spawned: true
              bytes: 637944
              windows:
                - start: "2026-10-16T02:51:13.807Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ab-0b8a-7e51-8b96-2bd081084538
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 3d4af4fd9200a90586f21a8d46f3f4562e2bf3d84e7809767ce0b9749ac9e07e
              started: "2026-10-16T03:04:28.545Z"
              spawned: false
              bytes: 605962
              windows:
                - start: "2026-10-16T03:04:28.545Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:15:28-07:00"
            - harness: codex
              id: 01a142ab-8f17-75d0-9d16-9f5cb036d5e3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 38d33ef511e7fc987d122ca62ae9610f13cdfb430ac9ffa7bab67f9d8de0d811
              started: "2026-10-16T03:05:02.084Z"
              spawned: true
              bytes: 850604
              windows:
                - start: "2026-10-16T03:05:02.084Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ad-01a9-7c12-af5c-200906e8cdca
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ad55ba4d74274be3fa6970cbb4955997b27eb5f808a7484d3ad8d07d5c434dcb
              started: "2026-10-16T03:06:37.159Z"
              spawned: false
              bytes: 456151
              windows:
                - start: "2026-10-16T03:06:37.159Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:15:28-07:00"
            - harness: codex
              id: 01a142ae-6330-7783-9f09-31d7ff6d2a7a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b71b546d1e6d10cbbda97b487230c878ffb28afe56dfcb9486d550d6615af80d
              started: "2026-10-16T03:08:07.966Z"
              spawned: false
              bytes: 446084
              windows:
                - start: "2026-10-16T03:08:07.966Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:15:28-07:00"
            - harness: codex
              id: 01a142ae-d92a-7ee2-824b-b23fbbac5e37
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8497a635a85846893095cc0bb14dec6d92791e01a5b3b66f0333f3e18adc7b34
              started: "2026-10-16T03:08:37.905Z"
              spawned: false
              bytes: 397998
              windows:
                - start: "2026-10-16T03:08:37.905Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:15:28-07:00"
            - harness: codex
              id: 01a142af-ad08-7e40-80ec-d227c3009b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 715da457ab79624da6910aa60f79e0029d3a7696674e47abf1a4f7de1c5c6c2d
              started: "2026-10-16T03:09:31.922Z"
              spawned: true
              bytes: 642617
              windows:
                - start: "2026-10-16T03:09:31.922Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142af-eee0-7690-a5d6-7665399ff88b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: c5861c3be06e7a31b005f2628f7729989e532d2bf5bbbcd1032cfaa094eb33aa
              started: "2026-10-16T03:09:48.791Z"
              spawned: true
              bytes: 704235
              windows:
                - start: "2026-10-16T03:09:48.791Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b3-1520-7c92-a180-31e3993bd485
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b94fc705dc5704b435b1465cd05b14ef71067d82623f540f15bb6c5f3b63d24c
              started: "2026-10-16T03:13:15.222Z"
              spawned: true
              bytes: 1266036
              windows:
                - start: "2026-10-16T03:13:15.222Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b8-e0ea-7662-a597-05926edba15b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: f007c9977d91413c421707c79fc5e86901c9e9c9b9109091e54c3174964de81c
              started: "2026-10-16T03:19:35.005Z"
              spawned: true
              bytes: 623158
              windows:
                - start: "2026-10-16T03:19:35.005Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142c6-124f-7491-a52b-beb2cb26f82f
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 9ca992bb23dafdecad96a56dbbb59e90918b18dc1b663443da393b4efe43467b
              started: "2026-10-16T03:33:59.704Z"
              spawned: true
              bytes: 909366
              windows:
                - start: "2026-10-16T03:33:59.704Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142c8-0a54-7ae0-b8e6-c5d900157a3d
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: db2455471dce5c23ba032d6016d44fcec459b83f4dbaaa2dffff17405c4b7a73
              started: "2026-10-16T03:36:08.698Z"
              spawned: true
              bytes: 859979
              windows:
                - start: "2026-10-16T03:36:08.698Z"
                  end: "2026-10-15T20:45:21-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.0299
        cuts:
            - dispatched: "2026-10-16T03:51:41-07:00"
              finalized: "2026-10-16T04:07:39-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T04:34:35-07:00"
      dispatched: "2026-10-16T04:29:57-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Issue Spec line 40 and Done-when rewritten; plan rows struck and Revisions entry 147 added; leftover active plan prose raised as a new Minor.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Reverted the stopping guard and the errNoticeIgnored wrap; TestRestartThenLateBindingNoticeStillRestarts and TestUnusableBindingNoticeLeavesAgentRunning both go red, green at head.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: Darwin only. On Linux kill(-pgid,0) succeeds while the unreaped zombie leader is in the group, so the poll still spends the full grace and SIGKILLs; the comment at owned_group.go:44 admits Darwin only, and atlas claims Darwin/Linux.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Label resolved lazily once behind an injected func (TestHolderLabelResolvesOnlyOnFirstLock); probe measured ~0.2s; remaining placement on the pump raised as a Minor.
          round: 2
        - id: BR-5
          disposition: addressed
          note: handleBindingNotice/stopForSignal are the production call sites (wrap.go:3007,3221) and the new wrapper tests drive them against a real owned group.
          round: 2
        - id: BR-6
          disposition: addressed
          note: atlas/couch.md:4108 Owners no longer lists native_lock_capability.go.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: Deferred by the implementer; a missing watcher still gives no notice.
          round: 2
        - id: BR-8
          disposition: addressed
          note: Acquire falls back to "unlabelled"; lock_test asserts the refusal names it.
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: Deferred by the implementer; three validators remain.
          round: 2
      findings:
        - kind: defect
          id: BR-10
          severity: Minor
          title: First late lock runs the sdlc workspace label lookup on masterPump while holding the lock mutex
          detail: '2nd finding in this family. Rule: an external probe that only produces a diagnostic label never runs on the PTY I/O loop or before spawn. Start it once in the background when the wrapper starts and use its result if ready, else the fallback label. Measured about 0.2s per lookup, capped at 2s.'
          family: external-probe-on-startup-path
          round: 2
        - kind: defect
          id: BR-11
          severity: Minor
          title: Plan's active Architecture line, capability sections, test row and unchecked items 120/123 still describe the dropped cutover gate
          detail: 'Rule: when a revision drops scope, edit or strike every active statement where it stands (summary, unchecked checklist, test tables); only dated log entries stay as history. Item 123 would have the operator check for "legacy blockers" that this build does not produce.'
          family: plan-claims-undelivered-surface
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4ab35a34-d877-454a-b82f-916d25983253
            repository: github.com/xianxu/pair
            issue_id: "000595"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/884f93f09763acdf
            reviewed_head: c1a5e8b9c93b76608cd6507ae8618e38760d5458
            slot: pair:2
            requested_at: "2026-10-16T12:18:59.372059Z"
          phase: finished
          pid: 44568
          started_at: "2026-10-16T12:18:59.536766Z"
          finished_at: "2026-10-16T12:28:33.800339Z"
          session_id: ea43f953-be3b-4c42-83fc-1654a928e401
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c1a5e8b9c93b76608cd6507ae8618e38760d5458
        tracker_head: 3f22d550b52509a4ae4035a0d42189ac7e1cae35
        branch_point: 1e93c3d9e0309d35f9993e3b52f5339480074081
        since: "2026-10-15T19:45:21-07:00"
        until: "2026-10-16T04:56:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T19:45:21-07:00"
              end: "2026-10-15T20:45:21-07:00"
              began: "2026-10-15T19:45:21-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-16T02:59:59-07:00"
              end: "2026-10-16T04:56:26-07:00"
              began: "2026-10-16T02:59:59-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 52124fd49ac71ee9c300d44783448280cb4f0a57d3ffda74f4e57a8254edabb5
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 7852251
              windows:
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:56:26-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T19:45:21-07:00"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:56:26-07:00"
            - harness: codex
              id: 01a1429e-ebc8-7610-b5df-8a4d6e94c9a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 303856442c5abae077043b1dddf05234698a6272e89f17d4add55b736976a625
              started: "2026-10-16T02:51:13.807Z"
              spawned: true
              bytes: 637944
              windows:
                - start: "2026-10-16T02:51:13.807Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ab-0b8a-7e51-8b96-2bd081084538
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 3d4af4fd9200a90586f21a8d46f3f4562e2bf3d84e7809767ce0b9749ac9e07e
              started: "2026-10-16T03:04:28.545Z"
              spawned: false
              bytes: 605962
              windows:
                - start: "2026-10-16T03:04:28.545Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:56:26-07:00"
            - harness: codex
              id: 01a142ab-8f17-75d0-9d16-9f5cb036d5e3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 38d33ef511e7fc987d122ca62ae9610f13cdfb430ac9ffa7bab67f9d8de0d811
              started: "2026-10-16T03:05:02.084Z"
              spawned: true
              bytes: 850604
              windows:
                - start: "2026-10-16T03:05:02.084Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ad-01a9-7c12-af5c-200906e8cdca
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ad55ba4d74274be3fa6970cbb4955997b27eb5f808a7484d3ad8d07d5c434dcb
              started: "2026-10-16T03:06:37.159Z"
              spawned: false
              bytes: 456151
              windows:
                - start: "2026-10-16T03:06:37.159Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:56:26-07:00"
            - harness: codex
              id: 01a142ae-6330-7783-9f09-31d7ff6d2a7a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b71b546d1e6d10cbbda97b487230c878ffb28afe56dfcb9486d550d6615af80d
              started: "2026-10-16T03:08:07.966Z"
              spawned: false
              bytes: 446084
              windows:
                - start: "2026-10-16T03:08:07.966Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:56:26-07:00"
            - harness: codex
              id: 01a142ae-d92a-7ee2-824b-b23fbbac5e37
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8497a635a85846893095cc0bb14dec6d92791e01a5b3b66f0333f3e18adc7b34
              started: "2026-10-16T03:08:37.905Z"
              spawned: false
              bytes: 397998
              windows:
                - start: "2026-10-16T03:08:37.905Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T04:56:26-07:00"
            - harness: codex
              id: 01a142af-ad08-7e40-80ec-d227c3009b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 715da457ab79624da6910aa60f79e0029d3a7696674e47abf1a4f7de1c5c6c2d
              started: "2026-10-16T03:09:31.922Z"
              spawned: true
              bytes: 642617
              windows:
                - start: "2026-10-16T03:09:31.922Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142af-eee0-7690-a5d6-7665399ff88b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: c5861c3be06e7a31b005f2628f7729989e532d2bf5bbbcd1032cfaa094eb33aa
              started: "2026-10-16T03:09:48.791Z"
              spawned: true
              bytes: 704235
              windows:
                - start: "2026-10-16T03:09:48.791Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b3-1520-7c92-a180-31e3993bd485
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b94fc705dc5704b435b1465cd05b14ef71067d82623f540f15bb6c5f3b63d24c
              started: "2026-10-16T03:13:15.222Z"
              spawned: true
              bytes: 1266036
              windows:
                - start: "2026-10-16T03:13:15.222Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b8-e0ea-7662-a597-05926edba15b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: f007c9977d91413c421707c79fc5e86901c9e9c9b9109091e54c3174964de81c
              started: "2026-10-16T03:19:35.005Z"
              spawned: true
              bytes: 623158
              windows:
                - start: "2026-10-16T03:19:35.005Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142c6-124f-7491-a52b-beb2cb26f82f
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 9ca992bb23dafdecad96a56dbbb59e90918b18dc1b663443da393b4efe43467b
              started: "2026-10-16T03:33:59.704Z"
              spawned: true
              bytes: 909366
              windows:
                - start: "2026-10-16T03:33:59.704Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142c8-0a54-7ae0-b8e6-c5d900157a3d
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: db2455471dce5c23ba032d6016d44fcec459b83f4dbaaa2dffff17405c4b7a73
              started: "2026-10-16T03:36:08.698Z"
              spawned: true
              bytes: 859979
              windows:
                - start: "2026-10-16T03:36:08.698Z"
                  end: "2026-10-15T20:45:21-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.0299
        cuts:
            - dispatched: "2026-10-16T03:51:41-07:00"
              finalized: "2026-10-16T04:07:39-07:00"
              finalizing: false
            - dispatched: "2026-10-16T04:29:57-07:00"
              finalized: "2026-10-16T04:34:35-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-16T05:28:35-07:00"
      dispatched: "2026-10-16T05:18:57-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: Per-platform groupHasLiveMember ends cleanup early; reverting darwin probe makes TestOwnedGroupFinishesEarlyWhenTheGroupExits fail (5.009s), verified in checkout.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: wrap.go:2984 still notifies unconditionally on start failure (no Unbound check); nil launchWatcherArgv and os.Executable failure at wrap.go:2971 still give no notice.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: NativeLockFilename, validHolder and validBindingNotice remain separate rules; notice accepts space/Cf IDs that NewKey later rejects silently.
          round: 3
        - id: BR-10
          disposition: addressed
          note: slotLockHolder now runs once in a background goroutine at wrap.go:2896; SetHolder only relabels under the mutex; TestSetHolderRelabelsHeldLocks covers relabel.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: Architecture line, core concepts and checklist fixed, but plan line 72 (capability check), line 90 test row (maintenance refusal / migration exemption), line 108 estimate and item 118 (maintenance window) still describe the dropped gate.
          round: 3
      findings:
        - kind: defect
          id: BR-12
          severity: Minor
          title: Relabel writes then truncates in place, so a contender can read a torn holder label
          detail: 'This is the 2nd finding in family truthful-lock-status-notice. Rule: any holder text a refusal shows must be a complete label some owner actually wrote. nativelock/lock.go Relabel does WriteAt then Truncate; a loser reading between the two sees the new prefix plus the old suffix (e.g. pair:2alone). Diagnostic only; truncate first or write a fixed-size padded record.'
          family: truthful-lock-status-notice
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: c1a5e8b9c93b76608cd6507ae8618e38760d5458
      blocked: false
---

# Gate ledger — pair#595 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T04:07:39-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `plan-claims-undelivered-surface` (kind: defect) Spec, Done-when and plan Core concepts still require the dropped capability cutover gate
  Plan rows name launcher/native_lock_capability.go and readiness/record.go (absent/unchanged); issue Spec line 186 and Done-when bullet 4 still promise "relaunch older slots first" refusal and legacy-capability tests; the re-plan has no plan Revisions entry and the issue Log says "awaiting OK" while atlas cites TL e93b109d. Update plan/issue to the approved re-plan.
- **BR-2** [Important] `stopping-rejection-treated-as-lock-failure` (kind: defect) Late binding notice after a restart has started (SIGUSR2) turns the agent restart into a wrapper failure
  masterPump calls Discover after locks.Stop(); Advance rejects "discovery after stop", the error is stored in nativeLockErr and returned before the restartFresh check, so the wrapper exits 1 instead of re-execing. An ID valid for validBindingNotice but rejected by NewKey likewise kills the agent group. Ignore/log rejected notices while stopping and add a wrapper-level test.
- **BR-3** [Important] `fixed-grace-instead-of-bound` (kind: defect) ownedGroup always waits the full 1s grace, adding a second to every wrapper exit and restart
  StartStop only skips the timer on ESRCH; with only the unreaped leader left, Darwin returns EPERM (measured on this host) and Linux succeeds against zombies, so group.Wait always blocks 1s before SIGKILL and reap. The test uses a 1ms grace and cannot see it. Finish early once the group is empty.
- **BR-4** [Important] `external-probe-on-startup-path` (kind: defect) nativeLockHolder synchronously execs sdlc workspace --json (up to 2s) before every agent spawn
  It runs unconditionally, even for non-native wraps and unknown-ID Fresh launches, only to produce a diagnostic label, with no seam or fake (ARCH-CONSTRAINTS/ARCH-MOCK). Resolve lazily when a key is first acquired, from launch environment, behind an injectable seam.
- **BR-5** [Important] `wrapper-ordering-untested` (kind: defect) No wrapper-level tests for late collision, notice-channel handling or controlled signal/restart exits
  Done-when bullets 2-3 are covered only by the primitives; nothing drives masterPump notice to Discover to StartStop to nativeLockErr, or the SIGTERM/HUP/SIGUSR2 paths in run (ARCH-ORDER), which is how the restart bug above shipped.
- **BR-6** [Minor] `plan-claims-undelivered-surface` (kind: defect) atlas/couch.md Owners list still names the deleted launcher/native_lock_capability.go
- **BR-7** [Minor] `truthful-lock-status-notice` (kind: defect) Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
- **BR-8** [Minor] `diagnostic-label-blocks-lock` (kind: defect) Acquire fails outright on an invalid holder label instead of falling back to the generic label
- **BR-9** [Minor] `shared-identifier-validation` (kind: defect) Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule

## Round 2 — 2026-10-16T04:34:35-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Issue Spec line 40 and Done-when rewritten; plan rows struck and Revisions entry 147 added; leftover active plan prose raised as a new Minor.
- BR-2 — addressed — Reverted the stopping guard and the errNoticeIgnored wrap; TestRestartThenLateBindingNoticeStillRestarts and TestUnusableBindingNoticeLeavesAgentRunning both go red, green at head.
- BR-3 — not-addressed — Darwin only. On Linux kill(-pgid,0) succeeds while the unreaped zombie leader is in the group, so the poll still spends the full grace and SIGKILLs; the comment at owned_group.go:44 admits Darwin only, and atlas claims Darwin/Linux.
- BR-4 — addressed — Label resolved lazily once behind an injected func (TestHolderLabelResolvesOnlyOnFirstLock); probe measured ~0.2s; remaining placement on the pump raised as a Minor.
- BR-5 — addressed — handleBindingNotice/stopForSignal are the production call sites (wrap.go:3007,3221) and the new wrapper tests drive them against a real owned group.
- BR-6 — addressed — atlas/couch.md:4108 Owners no longer lists native_lock_capability.go.
- BR-7 — not-addressed — Deferred by the implementer; a missing watcher still gives no notice.
- BR-8 — addressed — Acquire falls back to "unlabelled"; lock_test asserts the refusal names it.
- BR-9 — not-addressed — Deferred by the implementer; three validators remain.

### Raised

- **BR-10** [Minor] `external-probe-on-startup-path` (kind: defect) First late lock runs the sdlc workspace label lookup on masterPump while holding the lock mutex
  2nd finding in this family. Rule: an external probe that only produces a diagnostic label never runs on the PTY I/O loop or before spawn. Start it once in the background when the wrapper starts and use its result if ready, else the fallback label. Measured about 0.2s per lookup, capped at 2s.
- **BR-11** [Minor] `plan-claims-undelivered-surface` (kind: defect) Plan's active Architecture line, capability sections, test row and unchecked items 120/123 still describe the dropped cutover gate
  Rule: when a revision drops scope, edit or strike every active statement where it stands (summary, unchecked checklist, test tables); only dated log entries stay as history. Item 123 would have the operator check for "legacy blockers" that this build does not produce.

## Round 3 — 2026-10-16T05:28:35-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — Per-platform groupHasLiveMember ends cleanup early; reverting darwin probe makes TestOwnedGroupFinishesEarlyWhenTheGroupExits fail (5.009s), verified in checkout.
- BR-7 — not-addressed — wrap.go:2984 still notifies unconditionally on start failure (no Unbound check); nil launchWatcherArgv and os.Executable failure at wrap.go:2971 still give no notice.
- BR-9 — not-addressed — NativeLockFilename, validHolder and validBindingNotice remain separate rules; notice accepts space/Cf IDs that NewKey later rejects silently.
- BR-10 — addressed — slotLockHolder now runs once in a background goroutine at wrap.go:2896; SetHolder only relabels under the mutex; TestSetHolderRelabelsHeldLocks covers relabel.
- BR-11 — not-addressed — Architecture line, core concepts and checklist fixed, but plan line 72 (capability check), line 90 test row (maintenance refusal / migration exemption), line 108 estimate and item 118 (maintenance window) still describe the dropped gate.

### Raised

- **BR-12** [Minor] `truthful-lock-status-notice` (kind: defect) Relabel writes then truncates in place, so a contender can read a torn holder label
  This is the 2nd finding in family truthful-lock-status-notice. Rule: any holder text a refusal shows must be a complete label some owner actually wrote. nativelock/lock.go Relabel does WriteAt then Truncate; a loser reading between the two sees the new prefix plus the old suffix (e.g. pair:2alone). Diagnostic only; truncate first or write a fixed-size padded record.

## Open findings

- **BR-7** [Minor] `truthful-lock-status-notice` (kind: defect) Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
- **BR-9** [Minor] `shared-identifier-validation` (kind: defect) Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule
- **BR-11** [Minor] `plan-claims-undelivered-surface` (kind: defect) Plan's active Architecture line, capability sections, test row and unchecked items 120/123 still describe the dropped cutover gate
- **BR-12** [Minor] `truthful-lock-status-notice` (kind: defect) Relabel writes then truncates in place, so a contender can read a torn holder label
