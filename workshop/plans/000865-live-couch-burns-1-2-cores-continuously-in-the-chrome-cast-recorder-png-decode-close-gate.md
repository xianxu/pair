---
gate: boundary-review
issue: 865
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 50d58553-4899-4b00-b781-66771bbe80f2
            repository: github.com/xianxu/pair
            issue_id: "000865"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/98dea3c352b55762
            reviewed_head: 5905c8c7dcb4c326932e740b598f8573894c3507
            slot: pair:9
            requested_at: "2026-10-25T03:30:50.884647Z"
          phase: finished
          pid: 66530
          started_at: "2026-10-25T03:30:51.048024Z"
          finished_at: "2026-10-25T03:32:29.078403Z"
          session_id: 01a1711c-7022-76c0-a096-def9c42fb319
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/pair
        code_head: 5905c8c7dcb4c326932e740b598f8573894c3507
        tracker_head: 9a2561a3437ea1ed8fa7dadf7b56002c7ae9ee01
        branch_point: 64e0a8452388843b1fd1791059368682eec1236e
        since: "2026-10-24T02:51:38-07:00"
        until: "2026-10-24T20:24:49-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-24T02:51:38-07:00"
              end: "2026-10-24T20:24:49-07:00"
              began: "2026-10-24T02:51:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 6f745bb9aaf3071ec1c316d4cf4b9708af6b7b0b94a3c850e2227f5a51acaa0f
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 40641837
              windows:
                - start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e3f-96da-77e2-9e7e-a2e539093730
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c07763c40ab517bc3651bcb180c7dd9a039c092290c09adeea491bf23500a3b9
              started: "2026-10-24T14:10:24.592Z"
              spawned: false
              bytes: 561992
              windows:
                - start: "2026-10-24T14:10:24.592Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:10:24.592Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e42-5d03-7223-90f5-ee5ac13cda7a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 805edfe7dde0af72ba337fc02826e61bdc0980010871d12ad3b7c43fdda83255
              started: "2026-10-24T14:13:26.372Z"
              spawned: false
              bytes: 434815
              windows:
                - start: "2026-10-24T14:13:26.372Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:13:26.372Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e44-5fe8-7922-8aeb-5f3633e5f47b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c82b051b8a95913d729cfa909b82ecdb38596ab45c54e133c11c72c9a771ca08
              started: "2026-10-24T14:15:37.989Z"
              spawned: false
              bytes: 443388
              windows:
                - start: "2026-10-24T14:15:37.989Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:15:37.989Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e45-1372-7320-905e-20e435093633
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 42dd9c56c2342c9f36d193f33a7ec59c2d9880660b69a01eddd5b66fd8294aa9
              started: "2026-10-24T14:16:23.841Z"
              spawned: false
              bytes: 431017
              windows:
                - start: "2026-10-24T14:16:23.841Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:16:23.841Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e52-2625-7130-b42a-d22b20a72c06
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: c88492b80a18085c139ecd18613b69cb5d183ee97b7f23f36e071f631e9af948
              started: "2026-10-24T14:30:40.155Z"
              spawned: true
              bytes: 1644655
              windows:
                - start: "2026-10-24T14:30:40.155Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:30:40.155Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e6a-98a8-73b1-a9c6-28115afadcd0
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: 89c6f233b6d408fe7d05696e8e1040068b53cfcfc02ccca7b78071430e141ee8
              started: "2026-10-24T14:57:22.309Z"
              spawned: true
              bytes: 740388
              windows:
                - start: "2026-10-24T14:57:22.309Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:57:22.309Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e6a-df16-7fe2-8413-53c48d501cdc
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: 145161949509960068f01d472629932f4ca145358072831112d21e68a748b5ec
              started: "2026-10-24T14:57:40.292Z"
              spawned: true
              bytes: 548650
              windows:
                - start: "2026-10-24T14:57:40.292Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:57:40.292Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e5b2b6c8cab7871440b2cbeb0e3f678201edd6130131a3584ad115140a3d0ed2
              started: "2026-10-24T15:13:32.628Z"
              spawned: false
              bytes: 15379837
              windows:
                - start: "2026-10-24T15:13:32.628Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T15:13:32.628Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e83-20e9-7961-a1b1-09fb17691f16
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              digest: 4f7df942364fcab001e18259ad6b29bac6e7e8e583854d416bef7874f27aaf98
              started: "2026-10-24T15:24:09.951Z"
              spawned: true
              bytes: 1122881
              windows:
                - start: "2026-10-24T15:24:09.951Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T15:24:09.951Z"
                  end: "2026-10-24T20:24:49-07:00"
            - harness: codex
              id: 01a16e83-3c41-7fd2-a753-f95e055b8cf2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              digest: 2a6fac3375cef1ca4aee018faf890aa9404ff9a46ef9f30b2dcbf0144c3e7a09
              started: "2026-10-24T15:24:17Z"
              spawned: true
              bytes: 631805
              windows:
                - start: "2026-10-24T15:24:17Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T15:24:17Z"
                  end: "2026-10-24T20:24:49-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 30.265533333333334
        branches:
            - root: /Users/xianxu/workspace/worktree/pair-slot9/pair
              path: /Users/xianxu/workspace/pair/.git/worktrees/pair8/logs/HEAD
              head: 7ce33d19b7aff6c93d4638a9d841b2373fd2fe8b
              branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
              digest: b85db6b3a9df399a52e53dad4cc0089ef64cbb98546377ddfe2e95735f50acce
              bytes: 71331
              intervals:
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T10:00:20.578Z"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T07:08:47-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T07:08:47-07:00"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 1
                  start: "2026-10-24T14:10:24.592Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 1
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 1
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 1
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 1
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 2
                  start: "2026-10-24T14:13:26.372Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 2
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 2
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 2
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 2
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 2
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 2
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 3
                  start: "2026-10-24T14:15:37.989Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 3
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 3
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 3
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 3
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 3
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 3
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 4
                  start: "2026-10-24T14:16:23.841Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 4
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 4
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 4
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 4
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 4
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 4
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 5
                  start: "2026-10-24T14:30:40.155Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 5
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 5
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 5
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 5
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 6
                  start: "2026-10-24T14:57:22.309Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 6
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 6
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 6
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 6
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 6
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 6
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 7
                  start: "2026-10-24T14:57:40.292Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 7
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 7
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 7
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 7
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 7
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 7
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 8
                  start: "2026-10-24T15:13:32.628Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 8
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 8
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 8
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 8
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 8
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 8
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 9
                  start: "2026-10-24T15:24:09.951Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 9
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 9
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 9
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 9
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 9
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 9
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
                - source_index: 10
                  start: "2026-10-24T15:24:17Z"
                  end: "2026-10-24T15:28:03Z"
                  branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
                  kind: branch
                - source_index: 10
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 10
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 10
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 10
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 10
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 10
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T20:24:49-07:00"
                  kind: foreign-branch
      "n": 1
      timestamp: "2026-10-24T20:32:31-07:00"
      dispatched: "2026-10-24T20:30:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 5905c8c7dcb4c326932e740b598f8573894c3507
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9bae9569-6c28-44c5-9ab5-9fdf28f3aa05
            repository: github.com/xianxu/pair
            issue_id: "000865"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/56203746be704796
            reviewed_head: 9676dff56fc63b95885f3ec084127d766ca1976e
            slot: pair:9
            requested_at: "2026-10-25T06:05:21.369366Z"
          phase: finished
          pid: 22282
          started_at: "2026-10-25T06:05:21.468378Z"
          finished_at: "2026-10-25T06:06:49.557682Z"
          session_id: 01a171a9-e3e0-7851-8260-a3d1e3d0bf93
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/pair
        code_head: 9676dff56fc63b95885f3ec084127d766ca1976e
        tracker_head: ee5a98ee1c8de1cfc296fd2415a3ce34789f4cf7
        branch_point: 4e3764c65acfff86a9073f1546e38251798005b7
        since: "2026-10-24T02:51:38-07:00"
        until: "2026-10-24T22:59:36-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-24T02:51:38-07:00"
              end: "2026-10-24T22:59:36-07:00"
              began: "2026-10-24T02:51:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 6f745bb9aaf3071ec1c316d4cf4b9708af6b7b0b94a3c850e2227f5a51acaa0f
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 40641837
              windows:
                - start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e3f-96da-77e2-9e7e-a2e539093730
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c07763c40ab517bc3651bcb180c7dd9a039c092290c09adeea491bf23500a3b9
              started: "2026-10-24T14:10:24.592Z"
              spawned: false
              bytes: 561992
              windows:
                - start: "2026-10-24T14:10:24.592Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:10:24.592Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e42-5d03-7223-90f5-ee5ac13cda7a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 805edfe7dde0af72ba337fc02826e61bdc0980010871d12ad3b7c43fdda83255
              started: "2026-10-24T14:13:26.372Z"
              spawned: false
              bytes: 434815
              windows:
                - start: "2026-10-24T14:13:26.372Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:13:26.372Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e44-5fe8-7922-8aeb-5f3633e5f47b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c82b051b8a95913d729cfa909b82ecdb38596ab45c54e133c11c72c9a771ca08
              started: "2026-10-24T14:15:37.989Z"
              spawned: false
              bytes: 443388
              windows:
                - start: "2026-10-24T14:15:37.989Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:15:37.989Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e45-1372-7320-905e-20e435093633
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 42dd9c56c2342c9f36d193f33a7ec59c2d9880660b69a01eddd5b66fd8294aa9
              started: "2026-10-24T14:16:23.841Z"
              spawned: false
              bytes: 431017
              windows:
                - start: "2026-10-24T14:16:23.841Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:16:23.841Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e52-2625-7130-b42a-d22b20a72c06
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: c88492b80a18085c139ecd18613b69cb5d183ee97b7f23f36e071f631e9af948
              started: "2026-10-24T14:30:40.155Z"
              spawned: true
              bytes: 1644655
              windows:
                - start: "2026-10-24T14:30:40.155Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:30:40.155Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e6a-98a8-73b1-a9c6-28115afadcd0
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: 89c6f233b6d408fe7d05696e8e1040068b53cfcfc02ccca7b78071430e141ee8
              started: "2026-10-24T14:57:22.309Z"
              spawned: true
              bytes: 740388
              windows:
                - start: "2026-10-24T14:57:22.309Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:57:22.309Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e6a-df16-7fe2-8413-53c48d501cdc
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: 145161949509960068f01d472629932f4ca145358072831112d21e68a748b5ec
              started: "2026-10-24T14:57:40.292Z"
              spawned: true
              bytes: 548650
              windows:
                - start: "2026-10-24T14:57:40.292Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T14:57:40.292Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 7715df1428a7c37a781f8674b10d4cc432bd9e85004de98a84a205a353a12497
              started: "2026-10-24T15:13:32.628Z"
              spawned: false
              bytes: 19501586
              windows:
                - start: "2026-10-24T15:13:32.628Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T15:13:32.628Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e83-20e9-7961-a1b1-09fb17691f16
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              digest: 4f7df942364fcab001e18259ad6b29bac6e7e8e583854d416bef7874f27aaf98
              started: "2026-10-24T15:24:09.951Z"
              spawned: true
              bytes: 1122881
              windows:
                - start: "2026-10-24T15:24:09.951Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T15:24:09.951Z"
                  end: "2026-10-24T22:59:36-07:00"
            - harness: codex
              id: 01a16e83-3c41-7fd2-a753-f95e055b8cf2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              digest: 2a6fac3375cef1ca4aee018faf890aa9404ff9a46ef9f30b2dcbf0144c3e7a09
              started: "2026-10-24T15:24:17Z"
              spawned: true
              bytes: 631805
              windows:
                - start: "2026-10-24T15:24:17Z"
                  end: "2026-10-24T08:28:06-07:00"
                - start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                - start: "2026-10-25T05:58:09Z"
                  end: "2026-10-24T22:59:36-07:00"
              dispatch_windows:
                - start: "2026-10-24T15:24:17Z"
                  end: "2026-10-24T22:59:36-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 30.265533333333334
        cuts:
            - boundary: M1
              dispatched: "2026-10-24T20:30:47-07:00"
              finalized: "2026-10-24T20:32:31-07:00"
              finalizing: true
        branches:
            - root: /Users/xianxu/workspace/worktree/pair-slot9/pair
              path: /Users/xianxu/workspace/pair/.git/worktrees/pair8/logs/HEAD
              head: 9676dff56fc63b95885f3ec084127d766ca1976e
              branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
              digest: 6e456dc0d295c170006b0bb69870ffe266c55c1fc65836991369e7ae2e3401a3
              bytes: 72251
              intervals:
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T02:51:38-07:00"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 0
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T10:00:20.578Z"
                  end: "2026-10-24T11:41:30Z"
                  branch: 000860-multi-hub-fixture-authority-race
                  kind: foreign-branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T11:41:30Z"
                  end: "2026-10-24T14:08:47Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T14:08:47Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source: codex/01a16d5a-a8d5-7a51-8f76-9ad4f9728fcc
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 1
                  start: "2026-10-24T14:10:24.592Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 1
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 1
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 1
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 1
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 1
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 2
                  start: "2026-10-24T14:13:26.372Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 2
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 2
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 2
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 2
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 2
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 2
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 2
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 2
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 3
                  start: "2026-10-24T14:15:37.989Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 3
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 3
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 3
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 3
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 3
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 3
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 3
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 3
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 4
                  start: "2026-10-24T14:16:23.841Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 4
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 4
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 4
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 4
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 4
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 4
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 4
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 4
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 5
                  start: "2026-10-24T14:30:40.155Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 5
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 5
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 5
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 5
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 5
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 5
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 6
                  start: "2026-10-24T14:57:22.309Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 6
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 6
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 6
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 6
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 6
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 6
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 6
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 6
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 7
                  start: "2026-10-24T14:57:40.292Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 7
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 7
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 7
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 7
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 7
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 7
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 7
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 7
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 8
                  start: "2026-10-24T15:13:32.628Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 8
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 8
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 8
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 8
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 8
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 8
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 8
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 8
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 9
                  start: "2026-10-24T15:24:09.951Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 9
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 9
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 9
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 9
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 9
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 9
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 9
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 9
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
                - source_index: 10
                  start: "2026-10-24T15:24:17Z"
                  end: "2026-10-24T15:28:03Z"
                  kind: branch
                - source_index: 10
                  start: "2026-10-24T15:28:03Z"
                  end: "2026-10-24T08:28:06-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 10
                  start: "2026-10-24T08:28:06-07:00"
                  end: "2026-10-24T15:28:31Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 10
                  start: "2026-10-24T15:28:31Z"
                  end: "2026-10-24T23:38:15Z"
                  branch: 000869-cli-fixture-pid-readiness
                  kind: foreign-branch
                - source_index: 10
                  start: "2026-10-24T23:38:15Z"
                  end: "2026-10-24T17:15:49-07:00"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 10
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 10
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-25T05:58:09Z"
                  branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
                  kind: foreign-branch
                - source_index: 10
                  start: "2026-10-25T05:58:09Z"
                  end: "2026-10-25T05:59:01Z"
                  branch: main-slot9
                  kind: ownership-only
                - source_index: 10
                  start: "2026-10-25T05:59:01Z"
                  end: "2026-10-24T22:59:36-07:00"
                  kind: branch
      "n": 2
      timestamp: "2026-10-24T23:06:53-07:00"
      dispatched: "2026-10-24T23:05:18-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 9676dff56fc63b95885f3ec084127d766ca1976e
      blocked: false
---

# Gate ledger — pair#865 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T20:32:31-07:00 (codex) — passed

## Round 2 — 2026-10-24T23:06:53-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
