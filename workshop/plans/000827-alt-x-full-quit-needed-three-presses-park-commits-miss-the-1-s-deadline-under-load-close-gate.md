---
gate: boundary-review
issue: 827
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 36606122-ce8c-4669-b6b6-a1b1ed5ac82f
            repository: github.com/xianxu/pair
            issue_id: "000827"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/03745bda5067b9fb
            reviewed_head: 81b47b3744ce3a937a181df9ddf84f3d4a3b25f3
            slot: pair:6
            requested_at: "2026-10-23T09:53:11.997873Z"
          phase: finished
          pid: 16264
          started_at: "2026-10-23T09:53:12.099614Z"
          finished_at: "2026-10-23T09:56:12.191304Z"
          session_id: 8c6994ef-e25c-4dd1-b458-c489dfdeb385
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 81b47b3744ce3a937a181df9ddf84f3d4a3b25f3
        tracker_head: 8979d0a3453647e5c9f821e2ac471450e51d2554
        branch_point: ce5146c9cc9dfaea20e25b74888052fbbeecd862
        since: "2026-10-22T23:31:37-07:00"
        until: "2026-10-23T02:52:41-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T23:31:37-07:00"
              end: "2026-10-23T02:52:41-07:00"
              began: "2026-10-22T23:31:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: claude
              id: 5ba7d27e-c4f7-4a9b-9e2c-f7ac71ef7e9a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f1e7d7b3ffa37996122e1d29760af7bc51a3fb9f97ee7234ddc8ba58c4fdb4e2
              started: "2026-10-23T07:52:37.426Z"
              spawned: false
              bytes: 3819127
              windows:
                - start: "2026-10-23T07:52:37.426Z"
                  end: "2026-10-23T02:52:41-07:00"
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2a9e721b1f528deddf4ce3e51f99753e3bd69a25e62d0b37a9d6671cbf8732df
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 125840536
              windows:
                - start: "2026-10-22T23:31:37-07:00"
                  end: "2026-10-23T02:52:41-07:00"
            - harness: codex
              id: 01a16775-e5c7-7a32-bcae-bf52931081e1
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 7f751be703864507edd4fba1628f0b05efd97270147476f78f32a935ec64e7c9
              started: "2026-10-23T06:32:22.391Z"
              spawned: true
              bytes: 785234
              windows:
                - start: "2026-10-23T06:32:22.391Z"
                  end: "2026-10-23T02:52:41-07:00"
            - harness: codex
              id: 01a16785-954b-7c51-9f71-33f27fd21d05
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f2be8bf13f242d5ecdc0b231a2b66d5e07669543a27db0af1669c8c00711e9c8
              started: "2026-10-23T06:49:30.64Z"
              spawned: false
              bytes: 603394
              windows:
                - start: "2026-10-23T06:49:30.64Z"
                  end: "2026-10-23T02:52:41-07:00"
            - harness: codex
              id: 01a16788-4096-7683-a7a1-b14076f7acc3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 99bce1d304ea8d13487f08f3bae8151ade78ad23ad0ccdf9c30539383bb7341f
              started: "2026-10-23T06:52:25.563Z"
              spawned: false
              bytes: 439088
              windows:
                - start: "2026-10-23T06:52:25.563Z"
                  end: "2026-10-23T02:52:41-07:00"
            - harness: codex
              id: 01a16789-8e8d-7860-a839-cc98ae2a0d21
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 7eb5522f07a7d162f9b82b6d21ee00edffaeb37a9214765de464303099eec100
              started: "2026-10-23T06:53:51.106Z"
              spawned: false
              bytes: 557788
              windows:
                - start: "2026-10-23T06:53:51.106Z"
                  end: "2026-10-23T02:52:41-07:00"
            - harness: codex
              id: 01a1678a-61df-7702-8f82-38b9f2ca840e
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 5dffb35235c55fa1b69838a29caacd65509eb1765bd74feadc00f2d41d595085
              started: "2026-10-23T06:54:45.155Z"
              spawned: false
              bytes: 388548
              windows:
                - start: "2026-10-23T06:54:45.155Z"
                  end: "2026-10-23T02:52:41-07:00"
        queue_waits:
            - attempt_id: aac5a4ecca626162b6fbf7e16ad18855
              slot: pair:6
              record_digest: e52a6c5101bd34bd7ced9702e24d9742a30bbacdf068b98c18edc54ec5fe809f
              start: "2026-10-23T09:02:38.664139Z"
              end: "2026-10-23T09:02:42.129203Z"
              queue_seconds: 3.465064
              raw_seconds: 3.465064
              exact: true
            - attempt_id: 9d92b3131b1da4391cb34c3f86a40bb7
              slot: pair:6
              record_digest: 3446e6afffe701befa4a5c5221acca75b4d8337f2c657d9401f2978646856929
              start: "2026-10-23T09:14:31.00841Z"
              end: "2026-10-23T09:40:14.956884Z"
              queue_seconds: 1543.948474
              raw_seconds: 1543.948474
              exact: true
      "n": 1
      timestamp: "2026-10-23T02:56:13-07:00"
      dispatched: "2026-10-23T02:53:10-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic,openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Skipped (unknown-incarnation) threads let a full quit return success and Couch exits, contrary to the plan's state table
          detail: leaveParkOne returns leaveSkipped with nil error (leave_park.go:160), so Leave(LeavePark) returns no error and console.go:2130 reports and calls Stop. The plan's State table says unknown incarnation yields one partial summary and Couch stays open. FormatLeaveParkSummary lists skipped threads but only runs on the error path. Either treat skipped as incomplete under LeavePark (with a test) or record a plan Revision keeping the old exit behaviour.
          family: plan-state-table-contract-drift
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Unresolved summary lines repeat the recovery advice in both Reason and Next
          detail: 'leaveParkOne wraps its error with withParkRecoveryAdvice, then leaveParkUnresolved computes ParkRecoveryAdvice again for Next, producing "...; <advice>; next: <advice>".'
          family: duplicated-derived-report-text
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Recovery advice for unresolved targets runs one at a time after the 60s budget, up to 5s each
          detail: park.go:343-353 calls leaveParkUnresolved per target after wg.Wait, each with recoveryObservationTimeout (5s) on a WithoutCancel context, adding unbudgeted exit latency with many open transactions.
          family: unbudgeted-post-budget-work
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: recoveryBackend comment says it never waits on a nested busy store, now false
          detail: park_recovery_store.go:56; withUpdateLockContext now polls within storeReadLockWait.
          family: stale-invariant-comment
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 10216be2-8dcb-4834-98ae-b83440de1781
            repository: github.com/xianxu/pair
            issue_id: "000827"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/b6c8bb7f40849aa3
            reviewed_head: 289ef0c5d2dfc1da8ae9fb8751c4f52999787bcb
            slot: pair:6
            requested_at: "2026-10-23T10:53:13.376383Z"
          phase: finished
          pid: 86802
          started_at: "2026-10-23T10:53:13.48695Z"
          finished_at: "2026-10-23T10:54:45.618456Z"
          session_id: adc42a05-642a-4d53-abeb-408e7a3e0104
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 289ef0c5d2dfc1da8ae9fb8751c4f52999787bcb
        tracker_head: 454e9f882bd10ebb25936212a299f38fc076c6db
        branch_point: ce5146c9cc9dfaea20e25b74888052fbbeecd862
        since: "2026-10-22T23:31:37-07:00"
        until: "2026-10-23T02:58:33-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T23:31:37-07:00"
              end: "2026-10-23T02:58:33-07:00"
              began: "2026-10-22T23:31:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: claude
              id: 5ba7d27e-c4f7-4a9b-9e2c-f7ac71ef7e9a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 21b8a08373f164e6d89c146b3e225c19c75c3ca2aaded680537a8ccca900120e
              started: "2026-10-23T07:52:37.426Z"
              spawned: false
              bytes: 4120999
              windows:
                - start: "2026-10-23T07:52:37.426Z"
                  end: "2026-10-23T02:58:33-07:00"
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2a9e721b1f528deddf4ce3e51f99753e3bd69a25e62d0b37a9d6671cbf8732df
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 125840536
              windows:
                - start: "2026-10-22T23:31:37-07:00"
                  end: "2026-10-23T02:58:33-07:00"
            - harness: codex
              id: 01a16775-e5c7-7a32-bcae-bf52931081e1
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 7f751be703864507edd4fba1628f0b05efd97270147476f78f32a935ec64e7c9
              started: "2026-10-23T06:32:22.391Z"
              spawned: true
              bytes: 785234
              windows:
                - start: "2026-10-23T06:32:22.391Z"
                  end: "2026-10-23T02:58:33-07:00"
            - harness: codex
              id: 01a16785-954b-7c51-9f71-33f27fd21d05
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f2be8bf13f242d5ecdc0b231a2b66d5e07669543a27db0af1669c8c00711e9c8
              started: "2026-10-23T06:49:30.64Z"
              spawned: false
              bytes: 603394
              windows:
                - start: "2026-10-23T06:49:30.64Z"
                  end: "2026-10-23T02:58:33-07:00"
            - harness: codex
              id: 01a16788-4096-7683-a7a1-b14076f7acc3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 99bce1d304ea8d13487f08f3bae8151ade78ad23ad0ccdf9c30539383bb7341f
              started: "2026-10-23T06:52:25.563Z"
              spawned: false
              bytes: 439088
              windows:
                - start: "2026-10-23T06:52:25.563Z"
                  end: "2026-10-23T02:58:33-07:00"
            - harness: codex
              id: 01a16789-8e8d-7860-a839-cc98ae2a0d21
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 7eb5522f07a7d162f9b82b6d21ee00edffaeb37a9214765de464303099eec100
              started: "2026-10-23T06:53:51.106Z"
              spawned: false
              bytes: 557788
              windows:
                - start: "2026-10-23T06:53:51.106Z"
                  end: "2026-10-23T02:58:33-07:00"
            - harness: codex
              id: 01a1678a-61df-7702-8f82-38b9f2ca840e
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 5dffb35235c55fa1b69838a29caacd65509eb1765bd74feadc00f2d41d595085
              started: "2026-10-23T06:54:45.155Z"
              spawned: false
              bytes: 388548
              windows:
                - start: "2026-10-23T06:54:45.155Z"
                  end: "2026-10-23T02:58:33-07:00"
        queue_waits:
            - attempt_id: aac5a4ecca626162b6fbf7e16ad18855
              slot: pair:6
              record_digest: e52a6c5101bd34bd7ced9702e24d9742a30bbacdf068b98c18edc54ec5fe809f
              start: "2026-10-23T09:02:38.664139Z"
              end: "2026-10-23T09:02:42.129203Z"
              queue_seconds: 3.465064
              raw_seconds: 3.465064
              exact: true
            - attempt_id: 9d92b3131b1da4391cb34c3f86a40bb7
              slot: pair:6
              record_digest: 3446e6afffe701befa4a5c5221acca75b4d8337f2c657d9401f2978646856929
              start: "2026-10-23T09:14:31.00841Z"
              end: "2026-10-23T09:40:14.956884Z"
              queue_seconds: 1543.948474
              raw_seconds: 1543.948474
              exact: true
        cuts:
            - dispatched: "2026-10-23T02:53:10-07:00"
              finalized: "2026-10-23T02:56:13-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-23T03:54:46-07:00"
      dispatched: "2026-10-23T03:53:11-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic,openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: leave_park.go:198-200 turns skipped into unresolved plus an error; TestLeaveParkSkippedThreadKeepsCouchOpen went red with the error line removed and green when restored.
          round: 2
        - id: BR-2
          disposition: addressed
          note: leaveParkOne no longer wraps with recovery advice (leave_park.go:174); pinned by the Reason/Next de-dup assertion at leave_park_test.go:344.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Advice lookups now run in parallel at the LifecycleParallelism bound (leave_park.go:203-223), each still capped by recoveryObservationTimeout, so the tail is bounded rather than serial.
          round: 2
        - id: BR-4
          disposition: addressed
          note: park_recovery_store.go:56-57 now says it waits only within storeReadLockWait and then refuses typed, which matches withUpdateLockContext.
          round: 2
      findings:
        - kind: defect
          id: BR-5
          severity: Minor
          title: LeaveResult.Unresolved doc comment omits skipped threads added by the BR-1 fix
          detail: 'park.go:175-177 lists "failed, not started, or unreadable", but leaveParkUnresolved also reports skipped incarnations. This is the 2nd finding in this family. Rule: when a change adds a member to an enumerated contract, grep the field/type name and update every comment that lists its members in the same commit.'
          family: stale-invariant-comment
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 289ef0c5d2dfc1da8ae9fb8751c4f52999787bcb
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: adbb3c19-0773-40e8-893a-f4fafe117645
            repository: github.com/xianxu/pair
            issue_id: "000827"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/be5b38dc0295aabf
            reviewed_head: 54ee25977fe7c6710ccc08173c9f7f3907f23a02
            slot: pair:6
            requested_at: "2026-10-23T12:00:28.596834Z"
          phase: finished
          pid: 57701
          started_at: "2026-10-23T12:00:28.78903Z"
          finished_at: "2026-10-23T12:01:07.989724Z"
          session_id: 76431d3d-68db-4060-9315-50ace62ff924
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 54ee25977fe7c6710ccc08173c9f7f3907f23a02
        tracker_head: a9861e99509a94410353e8d0ce9e57c314c67691
        branch_point: ce5146c9cc9dfaea20e25b74888052fbbeecd862
        since: "2026-10-22T23:31:37-07:00"
        until: "2026-10-23T03:55:09-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T23:31:37-07:00"
              end: "2026-10-23T03:55:09-07:00"
              began: "2026-10-22T23:31:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: claude
              id: 5ba7d27e-c4f7-4a9b-9e2c-f7ac71ef7e9a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 3a2c9ef04cf17e04433539655dd00dd23d3d9e2592a1be1c4c9ac2382991834b
              started: "2026-10-23T07:52:37.426Z"
              spawned: false
              bytes: 4979016
              windows:
                - start: "2026-10-23T07:52:37.426Z"
                  end: "2026-10-23T03:55:09-07:00"
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2a9e721b1f528deddf4ce3e51f99753e3bd69a25e62d0b37a9d6671cbf8732df
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 125840536
              windows:
                - start: "2026-10-22T23:31:37-07:00"
                  end: "2026-10-23T03:55:09-07:00"
            - harness: codex
              id: 01a16775-e5c7-7a32-bcae-bf52931081e1
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 7f751be703864507edd4fba1628f0b05efd97270147476f78f32a935ec64e7c9
              started: "2026-10-23T06:32:22.391Z"
              spawned: true
              bytes: 785234
              windows:
                - start: "2026-10-23T06:32:22.391Z"
                  end: "2026-10-23T03:55:09-07:00"
            - harness: codex
              id: 01a16785-954b-7c51-9f71-33f27fd21d05
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f2be8bf13f242d5ecdc0b231a2b66d5e07669543a27db0af1669c8c00711e9c8
              started: "2026-10-23T06:49:30.64Z"
              spawned: false
              bytes: 603394
              windows:
                - start: "2026-10-23T06:49:30.64Z"
                  end: "2026-10-23T03:55:09-07:00"
            - harness: codex
              id: 01a16788-4096-7683-a7a1-b14076f7acc3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 99bce1d304ea8d13487f08f3bae8151ade78ad23ad0ccdf9c30539383bb7341f
              started: "2026-10-23T06:52:25.563Z"
              spawned: false
              bytes: 439088
              windows:
                - start: "2026-10-23T06:52:25.563Z"
                  end: "2026-10-23T03:55:09-07:00"
            - harness: codex
              id: 01a16789-8e8d-7860-a839-cc98ae2a0d21
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 7eb5522f07a7d162f9b82b6d21ee00edffaeb37a9214765de464303099eec100
              started: "2026-10-23T06:53:51.106Z"
              spawned: false
              bytes: 557788
              windows:
                - start: "2026-10-23T06:53:51.106Z"
                  end: "2026-10-23T03:55:09-07:00"
            - harness: codex
              id: 01a1678a-61df-7702-8f82-38b9f2ca840e
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 5dffb35235c55fa1b69838a29caacd65509eb1765bd74feadc00f2d41d595085
              started: "2026-10-23T06:54:45.155Z"
              spawned: false
              bytes: 388548
              windows:
                - start: "2026-10-23T06:54:45.155Z"
                  end: "2026-10-23T03:55:09-07:00"
        queue_waits:
            - attempt_id: aac5a4ecca626162b6fbf7e16ad18855
              slot: pair:6
              record_digest: e52a6c5101bd34bd7ced9702e24d9742a30bbacdf068b98c18edc54ec5fe809f
              start: "2026-10-23T09:02:38.664139Z"
              end: "2026-10-23T09:02:42.129203Z"
              queue_seconds: 3.465064
              raw_seconds: 3.465064
              exact: true
            - attempt_id: 9d92b3131b1da4391cb34c3f86a40bb7
              slot: pair:6
              record_digest: 3446e6afffe701befa4a5c5221acca75b4d8337f2c657d9401f2978646856929
              start: "2026-10-23T09:14:31.00841Z"
              end: "2026-10-23T09:40:14.956884Z"
              queue_seconds: 1543.948474
              raw_seconds: 1543.948474
              exact: true
            - attempt_id: b63572aa9aaf1fbcf33ecf902bc4c364
              slot: pair:6
              record_digest: 98b8c1fc8460fe6ae8747572594889379a1a6524f02b5a01964d222d37c8e789
              start: "2026-10-23T09:58:48.703352Z"
              end: "2026-10-23T10:40:38.9704Z"
              queue_seconds: 2510.267048
              raw_seconds: 2510.267048
              exact: true
        cuts:
            - dispatched: "2026-10-23T02:53:10-07:00"
              finalized: "2026-10-23T02:56:13-07:00"
              finalizing: false
            - dispatched: "2026-10-23T03:53:11-07:00"
              finalized: "2026-10-23T03:54:46-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-23T05:01:44-07:00"
      dispatched: "2026-10-23T05:00:25-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic,openai)
      dispose:
        - id: BR-5
          disposition: addressed
          note: park.go:176 now lists "failed, not started, skipped, or unreadable", matching leaveParkUnresolved (leave_park.go:191-227). A grep of couchcore found only one other comment listing the kinds (leave_park.go:179-184), and it already includes skipped. The rule is recorded at workshop/lessons.md:2731.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 54ee25977fe7c6710ccc08173c9f7f3907f23a02
      blocked: false
---

# Gate ledger — pair#827 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T02:56:13-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `plan-state-table-contract-drift` (kind: defect) Skipped (unknown-incarnation) threads let a full quit return success and Couch exits, contrary to the plan's state table
  leaveParkOne returns leaveSkipped with nil error (leave_park.go:160), so Leave(LeavePark) returns no error and console.go:2130 reports and calls Stop. The plan's State table says unknown incarnation yields one partial summary and Couch stays open. FormatLeaveParkSummary lists skipped threads but only runs on the error path. Either treat skipped as incomplete under LeavePark (with a test) or record a plan Revision keeping the old exit behaviour.
- **BR-2** [Minor] `duplicated-derived-report-text` (kind: defect) Unresolved summary lines repeat the recovery advice in both Reason and Next
  leaveParkOne wraps its error with withParkRecoveryAdvice, then leaveParkUnresolved computes ParkRecoveryAdvice again for Next, producing "...; <advice>; next: <advice>".
- **BR-3** [Minor] `unbudgeted-post-budget-work` (kind: defect) Recovery advice for unresolved targets runs one at a time after the 60s budget, up to 5s each
  park.go:343-353 calls leaveParkUnresolved per target after wg.Wait, each with recoveryObservationTimeout (5s) on a WithoutCancel context, adding unbudgeted exit latency with many open transactions.
- **BR-4** [Minor] `stale-invariant-comment` (kind: defect) recoveryBackend comment says it never waits on a nested busy store, now false
  park_recovery_store.go:56; withUpdateLockContext now polls within storeReadLockWait.

## Round 2 — 2026-10-23T03:54:46-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — leave_park.go:198-200 turns skipped into unresolved plus an error; TestLeaveParkSkippedThreadKeepsCouchOpen went red with the error line removed and green when restored.
- BR-2 — addressed — leaveParkOne no longer wraps with recovery advice (leave_park.go:174); pinned by the Reason/Next de-dup assertion at leave_park_test.go:344.
- BR-3 — addressed — Advice lookups now run in parallel at the LifecycleParallelism bound (leave_park.go:203-223), each still capped by recoveryObservationTimeout, so the tail is bounded rather than serial.
- BR-4 — addressed — park_recovery_store.go:56-57 now says it waits only within storeReadLockWait and then refuses typed, which matches withUpdateLockContext.

### Raised

- **BR-5** [Minor] `stale-invariant-comment` (kind: defect) LeaveResult.Unresolved doc comment omits skipped threads added by the BR-1 fix
  park.go:175-177 lists "failed, not started, or unreadable", but leaveParkUnresolved also reports skipped incarnations. This is the 2nd finding in this family. Rule: when a change adds a member to an enumerated contract, grep the field/type name and update every comment that lists its members in the same commit.

## Round 3 — 2026-10-23T05:01:44-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — park.go:176 now lists "failed, not started, skipped, or unreadable", matching leaveParkUnresolved (leave_park.go:191-227). A grep of couchcore found only one other comment listing the kinds (leave_park.go:179-184), and it already includes skipped. The rule is recorded at workshop/lessons.md:2731.

## Open findings

(none — every finding has been disposed)
