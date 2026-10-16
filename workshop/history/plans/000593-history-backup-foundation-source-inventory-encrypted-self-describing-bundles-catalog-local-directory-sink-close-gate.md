---
gate: boundary-review
issue: 593
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 688632a3-ab2a-4ebd-a0ea-88c5c4e5129e
            repository: github.com/xianxu/pair
            issue_id: "000593"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/5f31f29c012d12c2
            reviewed_head: efaedce1e1978905b2c27e03e1cd3d119f23cc46
            slot: pair:1
            requested_at: "2026-10-16T03:51:27.814323Z"
          phase: finished
          pid: 1131
          started_at: "2026-10-16T03:51:28.069743Z"
          finished_at: "2026-10-16T03:54:27.377121Z"
          session_id: 01a142d6-1427-78a2-bb8d-a18a38be5107
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: efaedce1e1978905b2c27e03e1cd3d119f23cc46
        tracker_head: 75cccc13127c3afc7630bada10da2958f3444917
        branch_point: a1beae74bf3c0cc2a71a2884861dadcdfc13c653
        since: "2026-10-15T20:15:22-07:00"
        until: "2026-10-15T20:48:16-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:15:22-07:00"
              end: "2026-10-15T20:48:16-07:00"
              began: "2026-10-15T20:15:22-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a13337-c560-79d0-b42a-1aa85da46014
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 29304344041f41d71d4276624c83aecbd3cd845f4d555e8b323a67be3f5c2b18
              started: "2026-10-13T03:05:03.782Z"
              spawned: false
              bytes: 39254711
              windows:
                - start: "2026-10-15T20:15:22-07:00"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142b5-4b20-7c72-a925-2e8814f03b16
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 605b9fb68e98ca3e80adfde29f23c5490f566dde37feb306e6b100e82a862367
              started: "2026-10-16T03:15:40.066Z"
              spawned: true
              bytes: 486038
              windows:
                - start: "2026-10-16T03:15:40.066Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142ba-7ff7-73f0-a700-e6a6692d5459
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 18e483a7e74ad6e6d5db64d627a36325f9d2e92fd98248211e4668dd64942830
              started: "2026-10-16T03:21:21.793Z"
              spawned: false
              bytes: 501547
              windows:
                - start: "2026-10-16T03:21:21.793Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142bc-409e-7c60-933e-21eb3d9716c4
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dff5bcc801e89e2ecd5b4dacb3f7c2aee8666ec5c5c3e181c36b41181b14e5d0
              started: "2026-10-16T03:23:16.239Z"
              spawned: false
              bytes: 440993
              windows:
                - start: "2026-10-16T03:23:16.239Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142bd-f045-7da0-8ca0-d29c83da51f8
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 30f2b370f8a9226e6f17de12cde3e4e0503ea7a7ac0128b9e6afd1ea6f8a4870
              started: "2026-10-16T03:25:07.107Z"
              spawned: false
              bytes: 398455
              windows:
                - start: "2026-10-16T03:25:07.107Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142be-5790-76d0-bbfd-3fff65e8a11d
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dadb51424694c6051418d3f2e2b17eb241a3f9cba4504bf6785e56144d28ce1f
              started: "2026-10-16T03:25:33.773Z"
              spawned: false
              bytes: 410216
              windows:
                - start: "2026-10-16T03:25:33.773Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142c0-2214-7340-b8e4-8731cda1dae7
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: e365659e2b783bfe201bd79e2f4f891e541aa3ecac3d96d369a2cded1ed1a8e5
              started: "2026-10-16T03:27:30.632Z"
              spawned: true
              bytes: 746110
              windows:
                - start: "2026-10-16T03:27:30.632Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142c0-6d1c-72c0-9646-f924bbe4ad9b
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 3db5434e9d8bf456d7fc035f101a967e15f479e58fe80728cf30330356e5f869
              started: "2026-10-16T03:27:49.659Z"
              spawned: true
              bytes: 974483
              windows:
                - start: "2026-10-16T03:27:49.659Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142c0-cce9-7652-998d-aafb5fad5e47
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 6f592de2794d52e2e2fe98c80719a2c4f1943fee53ca7b3a2d92c3a0e0cd2a8d
              started: "2026-10-16T03:28:14.202Z"
              spawned: true
              bytes: 484425
              windows:
                - start: "2026-10-16T03:28:14.202Z"
                  end: "2026-10-15T20:48:16-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.0299
      "n": 1
      timestamp: "2026-10-15T20:54:28-07:00"
      dispatched: "2026-10-15T20:51:25-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Sink staging cleanup permanently blocks writes after 5,000 retained bundles
          detail: 'cmd/internal/historyarchive/sink.go:348 counts every destination-directory entry against MaxFiles before filtering for .partial- files. A kind/month directory containing 5,000 complete bundle/label pairs plus .archive-put.lock has 10,001 entries, so every subsequent Put fails during cleanup, even with no abandoned staging files. ARCH-CONSTRAINTS and ARCH-FUNERAL: isolate staging enumeration from retained objects or provide an enforced publication/rollover policy that avoids this cliff; add a regression with more than 10,000 retained final entries and verify publication and stale-stage cleanup still work.'
          family: retained-storage-must-not-exhaust-maintenance
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: The catalog source declaration excludes numbered catalog shards
          detail: 'cmd/internal/historyarchive/sources.json:200 declares only archive/*.jsonl, while catalog.go:280 produces files such as history.jsonl.000001. Those files contain later records in the same logical catalog but do not match its source declaration. ARCH-PURPOSE: declare every catalog shard while excluding locks/staging, and add a regression checking the inventory against names produced by catalog rollover.'
          family: inventory-must-cover-produced-artifacts
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: dbae6e99-ab7d-4a2b-814f-4d3300a948f6
            repository: github.com/xianxu/pair
            issue_id: "000593"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/741a771560fad050
            reviewed_head: 1ba3df80dac53d57725156e1b34c632d127757ef
            slot: pair:1
            requested_at: "2026-10-16T03:58:32.723043Z"
          phase: finished
          pid: 67180
          started_at: "2026-10-16T03:58:32.926085Z"
          finished_at: "2026-10-16T04:01:30.121329Z"
          session_id: 01a142dc-8f13-7a23-b154-6c30a68e4021
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1ba3df80dac53d57725156e1b34c632d127757ef
        tracker_head: 05324c22fa1feaf87d6c0f2652211afa0a9e226e
        branch_point: a1beae74bf3c0cc2a71a2884861dadcdfc13c653
        since: "2026-10-15T20:15:22-07:00"
        until: "2026-10-15T20:57:33-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:15:22-07:00"
              end: "2026-10-15T20:57:33-07:00"
              began: "2026-10-15T20:15:22-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a13337-c560-79d0-b42a-1aa85da46014
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 41bdcf6bb1b3a826f4396c2e863326791cb0d8859d07759d06d82ee608783985
              started: "2026-10-13T03:05:03.782Z"
              spawned: false
              bytes: 39684600
              windows:
                - start: "2026-10-15T20:15:22-07:00"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142b5-4b20-7c72-a925-2e8814f03b16
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 605b9fb68e98ca3e80adfde29f23c5490f566dde37feb306e6b100e82a862367
              started: "2026-10-16T03:15:40.066Z"
              spawned: true
              bytes: 486038
              windows:
                - start: "2026-10-16T03:15:40.066Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142ba-7ff7-73f0-a700-e6a6692d5459
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 18e483a7e74ad6e6d5db64d627a36325f9d2e92fd98248211e4668dd64942830
              started: "2026-10-16T03:21:21.793Z"
              spawned: false
              bytes: 501547
              windows:
                - start: "2026-10-16T03:21:21.793Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142bc-409e-7c60-933e-21eb3d9716c4
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dff5bcc801e89e2ecd5b4dacb3f7c2aee8666ec5c5c3e181c36b41181b14e5d0
              started: "2026-10-16T03:23:16.239Z"
              spawned: false
              bytes: 440993
              windows:
                - start: "2026-10-16T03:23:16.239Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142bd-f045-7da0-8ca0-d29c83da51f8
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 30f2b370f8a9226e6f17de12cde3e4e0503ea7a7ac0128b9e6afd1ea6f8a4870
              started: "2026-10-16T03:25:07.107Z"
              spawned: false
              bytes: 398455
              windows:
                - start: "2026-10-16T03:25:07.107Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142be-5790-76d0-bbfd-3fff65e8a11d
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dadb51424694c6051418d3f2e2b17eb241a3f9cba4504bf6785e56144d28ce1f
              started: "2026-10-16T03:25:33.773Z"
              spawned: false
              bytes: 410216
              windows:
                - start: "2026-10-16T03:25:33.773Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142c0-2214-7340-b8e4-8731cda1dae7
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: e365659e2b783bfe201bd79e2f4f891e541aa3ecac3d96d369a2cded1ed1a8e5
              started: "2026-10-16T03:27:30.632Z"
              spawned: true
              bytes: 746110
              windows:
                - start: "2026-10-16T03:27:30.632Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142c0-6d1c-72c0-9646-f924bbe4ad9b
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 3db5434e9d8bf456d7fc035f101a967e15f479e58fe80728cf30330356e5f869
              started: "2026-10-16T03:27:49.659Z"
              spawned: true
              bytes: 974483
              windows:
                - start: "2026-10-16T03:27:49.659Z"
                  end: "2026-10-15T20:57:33-07:00"
            - harness: codex
              id: 01a142c0-cce9-7652-998d-aafb5fad5e47
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 1c39cac2fd6bb036b11cbd7c019335c078b7d26ba4fbaa3362a8b07de79df6e5
              started: "2026-10-16T03:28:14.202Z"
              spawned: true
              bytes: 550031
              windows:
                - start: "2026-10-16T03:28:14.202Z"
                  end: "2026-10-15T20:57:33-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.0299
        cuts:
            - dispatched: "2026-10-15T20:51:25-07:00"
              finalized: "2026-10-15T20:54:28-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T21:02:02-07:00"
      dispatched: "2026-10-15T20:58:30-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Staging cleanup now scans the reserved .archive-staging directory. TestLocalSinkPutDoesNotScanRetainedObjects passes with 10,001 retained entries and fails with the previous sink.go; stale-stage cleanup and active-writer protection tests also pass.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The inventory includes six-digit catalog shards. TestCatalogInventoryIncludesEveryRolledShard exercises actual rollover and passes; restoring the previous sources.json makes it fail on both produced numbered shards.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Important
          title: Couch source inventory omits slot-local thread storage
          detail: 'cmd/internal/historyarchive/sources.json:76-84 declares couch-threads only under pair-data/couch/threadstore. Production newSlotThreadStore uses SlotLayout.Store(), which resolves to the slot environment''s .couch directory, and writes thread.json there (threadstore_layout.go:20-21, slotlayout.go:47, threadstore.go:145-149). That supported storage location has no source declaration. ARCH-PURPOSE: this is the 2nd finding in family inventory-must-cover-produced-artifacts. Earlier rounds fixed instances. Do NOT fix only this instance: state the rule covering every supported producer/backend location, enumerate and sweep the inventory against those locations, and add regression coverage using production-generated paths, including global and slot-local Couch stores.'
          family: inventory-must-cover-produced-artifacts
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 065e7fcc-cb86-4fa1-b9a9-1ef1b1b96482
            repository: github.com/xianxu/pair
            issue_id: "000593"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/44db6a97dbbf196e
            reviewed_head: ec92c1fe36f427d3b6accd665a4b7147f79bf3ba
            slot: pair:1
            requested_at: "2026-10-16T04:09:48.078424Z"
          phase: finished
          pid: 87723
          started_at: "2026-10-16T04:09:48.212998Z"
          finished_at: "2026-10-16T04:11:34.484504Z"
          session_id: 01a142e6-dcaf-7333-8a3d-568a4933b51e
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: ec92c1fe36f427d3b6accd665a4b7147f79bf3ba
        tracker_head: 62e3f3aa7b6160d575753dd521aea597364da77b
        branch_point: a1beae74bf3c0cc2a71a2884861dadcdfc13c653
        since: "2026-10-15T20:15:22-07:00"
        until: "2026-10-15T21:09:07-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:15:22-07:00"
              end: "2026-10-15T21:09:07-07:00"
              began: "2026-10-15T20:15:22-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a13337-c560-79d0-b42a-1aa85da46014
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 8458b72de8766fb86d03637c69847137361f497c7f60821e757be753c8ac4288
              started: "2026-10-13T03:05:03.782Z"
              spawned: false
              bytes: 40194115
              windows:
                - start: "2026-10-15T20:15:22-07:00"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142b5-4b20-7c72-a925-2e8814f03b16
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 605b9fb68e98ca3e80adfde29f23c5490f566dde37feb306e6b100e82a862367
              started: "2026-10-16T03:15:40.066Z"
              spawned: true
              bytes: 486038
              windows:
                - start: "2026-10-16T03:15:40.066Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142ba-7ff7-73f0-a700-e6a6692d5459
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 18e483a7e74ad6e6d5db64d627a36325f9d2e92fd98248211e4668dd64942830
              started: "2026-10-16T03:21:21.793Z"
              spawned: false
              bytes: 501547
              windows:
                - start: "2026-10-16T03:21:21.793Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142bc-409e-7c60-933e-21eb3d9716c4
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dff5bcc801e89e2ecd5b4dacb3f7c2aee8666ec5c5c3e181c36b41181b14e5d0
              started: "2026-10-16T03:23:16.239Z"
              spawned: false
              bytes: 440993
              windows:
                - start: "2026-10-16T03:23:16.239Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142bd-f045-7da0-8ca0-d29c83da51f8
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 30f2b370f8a9226e6f17de12cde3e4e0503ea7a7ac0128b9e6afd1ea6f8a4870
              started: "2026-10-16T03:25:07.107Z"
              spawned: false
              bytes: 398455
              windows:
                - start: "2026-10-16T03:25:07.107Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142be-5790-76d0-bbfd-3fff65e8a11d
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dadb51424694c6051418d3f2e2b17eb241a3f9cba4504bf6785e56144d28ce1f
              started: "2026-10-16T03:25:33.773Z"
              spawned: false
              bytes: 410216
              windows:
                - start: "2026-10-16T03:25:33.773Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142c0-2214-7340-b8e4-8731cda1dae7
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 324654ead32bf2e527f42c8795d5e70aa8025cf8a9c78d966806a96b546c6ed2
              started: "2026-10-16T03:27:30.632Z"
              spawned: true
              bytes: 1337350
              windows:
                - start: "2026-10-16T03:27:30.632Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142c0-6d1c-72c0-9646-f924bbe4ad9b
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 3db5434e9d8bf456d7fc035f101a967e15f479e58fe80728cf30330356e5f869
              started: "2026-10-16T03:27:49.659Z"
              spawned: true
              bytes: 974483
              windows:
                - start: "2026-10-16T03:27:49.659Z"
                  end: "2026-10-15T21:09:07-07:00"
            - harness: codex
              id: 01a142c0-cce9-7652-998d-aafb5fad5e47
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 1c39cac2fd6bb036b11cbd7c019335c078b7d26ba4fbaa3362a8b07de79df6e5
              started: "2026-10-16T03:28:14.202Z"
              spawned: true
              bytes: 550031
              windows:
                - start: "2026-10-16T03:28:14.202Z"
                  end: "2026-10-15T21:09:07-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.0299
        cuts:
            - dispatched: "2026-10-15T20:51:25-07:00"
              finalized: "2026-10-15T20:54:28-07:00"
              finalizing: false
            - dispatched: "2026-10-15T20:58:30-07:00"
              finalized: "2026-10-15T21:02:02-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-15T21:11:35-07:00"
      dispatched: "2026-10-15T21:09:46-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Retained objects remain outside staging cleanup; TestLocalSinkPutDoesNotScanRetainedObjects passed in package and race runs.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Inventory includes numeric catalog shards; TestCatalogInventoryIncludesEveryRolledShard passed against actual rollover outputs.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Production-generated global and slot-local Couch paths are covered by inventory_producers_test.go:90. Restoring the previous sources.json in a disposable checkout made these tests fail on thread.json, archived records, legacy Pair paths and webcast outputs; pinned-head tests passed. The plan Revisions entry states the producer/backend coverage rule.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: ec92c1fe36f427d3b6accd665a4b7147f79bf3ba
      blocked: false
---

# Gate ledger — pair#593 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T20:54:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `retained-storage-must-not-exhaust-maintenance` (kind: defect) Sink staging cleanup permanently blocks writes after 5,000 retained bundles
  cmd/internal/historyarchive/sink.go:348 counts every destination-directory entry against MaxFiles before filtering for .partial- files. A kind/month directory containing 5,000 complete bundle/label pairs plus .archive-put.lock has 10,001 entries, so every subsequent Put fails during cleanup, even with no abandoned staging files. ARCH-CONSTRAINTS and ARCH-FUNERAL: isolate staging enumeration from retained objects or provide an enforced publication/rollover policy that avoids this cliff; add a regression with more than 10,000 retained final entries and verify publication and stale-stage cleanup still work.
- **BR-2** [Important] `inventory-must-cover-produced-artifacts` (kind: defect) The catalog source declaration excludes numbered catalog shards
  cmd/internal/historyarchive/sources.json:200 declares only archive/*.jsonl, while catalog.go:280 produces files such as history.jsonl.000001. Those files contain later records in the same logical catalog but do not match its source declaration. ARCH-PURPOSE: declare every catalog shard while excluding locks/staging, and add a regression checking the inventory against names produced by catalog rollover.

## Round 2 — 2026-10-15T21:02:02-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Staging cleanup now scans the reserved .archive-staging directory. TestLocalSinkPutDoesNotScanRetainedObjects passes with 10,001 retained entries and fails with the previous sink.go; stale-stage cleanup and active-writer protection tests also pass.
- BR-2 — addressed — The inventory includes six-digit catalog shards. TestCatalogInventoryIncludesEveryRolledShard exercises actual rollover and passes; restoring the previous sources.json makes it fail on both produced numbered shards.

### Raised

- **BR-3** [Important] `inventory-must-cover-produced-artifacts` (kind: defect) Couch source inventory omits slot-local thread storage
  cmd/internal/historyarchive/sources.json:76-84 declares couch-threads only under pair-data/couch/threadstore. Production newSlotThreadStore uses SlotLayout.Store(), which resolves to the slot environment's .couch directory, and writes thread.json there (threadstore_layout.go:20-21, slotlayout.go:47, threadstore.go:145-149). That supported storage location has no source declaration. ARCH-PURPOSE: this is the 2nd finding in family inventory-must-cover-produced-artifacts. Earlier rounds fixed instances. Do NOT fix only this instance: state the rule covering every supported producer/backend location, enumerate and sweep the inventory against those locations, and add regression coverage using production-generated paths, including global and slot-local Couch stores.

## Round 3 — 2026-10-15T21:11:35-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retained objects remain outside staging cleanup; TestLocalSinkPutDoesNotScanRetainedObjects passed in package and race runs.
- BR-2 — addressed — Inventory includes numeric catalog shards; TestCatalogInventoryIncludesEveryRolledShard passed against actual rollover outputs.
- BR-3 — addressed — Production-generated global and slot-local Couch paths are covered by inventory_producers_test.go:90. Restoring the previous sources.json in a disposable checkout made these tests fail on thread.json, archived records, legacy Pair paths and webcast outputs; pinned-head tests passed. The plan Revisions entry states the producer/backend coverage rule.

## Open findings

(none — every finding has been disposed)
