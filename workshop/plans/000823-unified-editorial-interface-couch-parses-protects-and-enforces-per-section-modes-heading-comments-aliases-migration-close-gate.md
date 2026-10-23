---
gate: boundary-review
issue: 823
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e1063d65-ffa9-4199-96e1-d0f18f6eb7c7
            repository: github.com/xianxu/pair
            issue_id: "000823"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/52487aa9f600139f
            reviewed_head: a859c72eb4600b7c979093b3714e4a7a75334867
            slot: pair:9
            requested_at: "2026-10-23T11:38:35.700277Z"
          phase: finished
          pid: 53375
          started_at: "2026-10-23T11:38:35.887383Z"
          finished_at: "2026-10-23T11:42:52.003239Z"
          session_id: 01a1688e-43ca-7641-8932-7764fd263afe
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: a859c72eb4600b7c979093b3714e4a7a75334867
        tracker_head: a570c4cc3ad262716f83fc909f33a0fb50729491
        branch_point: 4623c797574cd6aa12d765f6465454f74f02cc98
        since: "2026-10-23T03:16:56-07:00"
        until: "2026-10-23T04:37:39-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T03:16:56-07:00"
              end: "2026-10-23T04:37:39-07:00"
              began: "2026-10-23T03:16:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 375a9492baeeb9e36653162de76fb10126244446586025351e107c216940341f
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 48919610
              windows:
                - start: "2026-10-23T03:16:56-07:00"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a16855-b204-7ea1-aa9c-807df3683ad5
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 3e811c0021fe87951208cc26317c88a66bb839923bfc56761d1ac159b542cf09
              started: "2026-10-23T10:36:49.595Z"
              spawned: false
              bytes: 836383
              windows:
                - start: "2026-10-23T10:36:49.595Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a16859-6f9d-7f41-b16c-a59b0f09c784
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: d187e918d6a5bf9b086ec1e7ba56dbcbfff13e1723fb2ba4023e6b5031bd8284
              started: "2026-10-23T10:40:54.568Z"
              spawned: false
              bytes: 432559
              windows:
                - start: "2026-10-23T10:40:54.568Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a1685b-959d-72f1-b83a-9ac59140f8ae
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 8d7338b0583b26ccd3da5ad9aebc674ef47cd0f8dbb59b1ad2b18de43f2b1d02
              started: "2026-10-23T10:43:15.332Z"
              spawned: false
              bytes: 461896
              windows:
                - start: "2026-10-23T10:43:15.332Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a1685c-1fd0-7683-938e-3900db4ac004
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 594c64029dab5c95fe95fe386105ff56d18f63ad8e497711dcb462764475cd56
              started: "2026-10-23T10:43:50.676Z"
              spawned: false
              bytes: 391270
              windows:
                - start: "2026-10-23T10:43:50.676Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a1685d-2b84-7221-9e94-b0070d719a70
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 64dc0df8421382252a1d49d4a42de4424f67e47539400a742c7bd9287e255c6d
              started: "2026-10-23T10:44:58.977Z"
              spawned: true
              bytes: 454476
              windows:
                - start: "2026-10-23T10:44:58.977Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a1685f-7603-7460-ba07-8af9662424f1
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 7ee007c88b8bf95e7f5899e487505f855b7fdfd9d7bc0cd49f3bf1389b1cb25e
              started: "2026-10-23T10:47:29.147Z"
              spawned: true
              bytes: 273450
              windows:
                - start: "2026-10-23T10:47:29.147Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a16860-1703-7fd0-a1db-82fd9b20b674
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: dcf8e20d7d1e5c5a1710f118e48644c39f717eca3b81bfdf6b04923c198bbaea
              started: "2026-10-23T10:48:10.359Z"
              spawned: true
              bytes: 169503
              windows:
                - start: "2026-10-23T10:48:10.359Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 048d2b8cbff694b6b98e39e0ea10e5041aa828888c8d1390e3fe7c2edef3b8d9
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 3921442
              windows:
                - start: "2026-10-23T10:51:20.121Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a16863-c14a-7090-88fd-e38f711b177a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: 0f62e162017019262adcc8d28a8884993e1a715a8756128aa79f31366d887495
              started: "2026-10-23T10:52:10.533Z"
              spawned: true
              bytes: 1459557
              windows:
                - start: "2026-10-23T10:52:10.533Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a16867-fdac-7010-975a-fc85db57b909
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: 0a269e61297e41268bb1f0eaddec7297db37d37afca34e041833e840d877765c
              started: "2026-10-23T10:56:48.091Z"
              spawned: true
              bytes: 759985
              windows:
                - start: "2026-10-23T10:56:48.091Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a1686a-1cc1-7642-81bf-0bce7b8fb3bb
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: af3fa58a417fa8827577f568267cc3637fa0fdf33f17b9b09b958f492e3cb1d1
              started: "2026-10-23T10:59:07.116Z"
              spawned: true
              bytes: 1475114
              windows:
                - start: "2026-10-23T10:59:07.116Z"
                  end: "2026-10-23T04:37:39-07:00"
            - harness: codex
              id: 01a1686d-6e91-79c0-bbda-816809b34261
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: d24a5d66586c47d3d53566c792b5c8ac7902f75a4bc2997c5fcd706584357574
              started: "2026-10-23T11:02:44.675Z"
              spawned: true
              bytes: 1481924
              windows:
                - start: "2026-10-23T11:02:44.675Z"
                  end: "2026-10-23T04:37:39-07:00"
      "n": 1
      timestamp: "2026-10-23T04:42:53-07:00"
      dispatched: "2026-10-23T04:38:32-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Chat item protection excludes trailing fenced-code syntax
          detail: cmd/internal/editorial/document.go:222-229 derives item extents from Goldmark block Lines(), which excludes fenced-code delimiters; edits.go:450-454 then trusts that truncated extent for protection and answer placement. For a chat item containing "- Why?\n\n  ```\n  example\n  ```\n", both ValidateEdits and production BuildEdit accept replacing the closing fence with "~~~", violating the operator-item preservation contract (ARCH-PURPOSE, ARCH-SECURE). Derive complete item source extents, including trailing syntax, and add regressions covering nonempty/empty fences and other block children without complete Lines() ranges.
          family: protected-source-extent
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: README update is missing for editorial settings and migration commands
          detail: cmd/internal/couchcmd/session_sync.go:65-66 introduces migrate-editorial and --confirm, while run.go:1516-1521 adds resolved editorial reads and policy-bound section hashes. README.md is unchanged in the pinned range; its session-sync usage at lines 510-527 omits migration and retains the older settings description. Update that usage with preview/confirmation, canonical settings and heading directives, protected ownership, and opaque section-hash behavior to satisfy the Docs update gate.
          family: user-surface-documentation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#823 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T04:42:53-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `protected-source-extent` (kind: defect) Chat item protection excludes trailing fenced-code syntax
  cmd/internal/editorial/document.go:222-229 derives item extents from Goldmark block Lines(), which excludes fenced-code delimiters; edits.go:450-454 then trusts that truncated extent for protection and answer placement. For a chat item containing "- Why?\n\n  ```\n  example\n  ```\n", both ValidateEdits and production BuildEdit accept replacing the closing fence with "~~~", violating the operator-item preservation contract (ARCH-PURPOSE, ARCH-SECURE). Derive complete item source extents, including trailing syntax, and add regressions covering nonempty/empty fences and other block children without complete Lines() ranges.
- **BR-2** [Important] `user-surface-documentation` (kind: defect) README update is missing for editorial settings and migration commands
  cmd/internal/couchcmd/session_sync.go:65-66 introduces migrate-editorial and --confirm, while run.go:1516-1521 adds resolved editorial reads and policy-bound section hashes. README.md is unchanged in the pinned range; its session-sync usage at lines 510-527 omits migration and retains the older settings description. Update that usage with preview/confirmation, canonical settings and heading directives, protected ownership, and opaque section-hash behavior to satisfy the Docs update gate.

## Open findings

- **BR-1** [Critical] `protected-source-extent` (kind: defect) Chat item protection excludes trailing fenced-code syntax
- **BR-2** [Important] `user-surface-documentation` (kind: defect) README update is missing for editorial settings and migration commands
