---
gate: boundary-review
issue: 758
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7a1f5416-a281-4580-9698-8074dacfef93
            repository: github.com/xianxu/pair
            issue_id: "000758"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/653fd1bc9ae6485d
            reviewed_head: 1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf
            slot: pair:6
            requested_at: "2026-10-22T12:58:08.115608Z"
          phase: finished
          pid: 5890
          started_at: "2026-10-22T12:58:08.24957Z"
          finished_at: "2026-10-22T13:01:46.159896Z"
          session_id: 01a163b0-b992-73d2-ba88-164aae0a719c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf
        tracker_head: 5c6b4ccca2c31c8883b5d8ad1d1168952d600881
        branch_point: 1e70379fba4a3653258f5542ac1d1aecf0ca1b43
        since: "2026-10-22T04:59:40-07:00"
        until: "2026-10-22T05:57:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T04:59:40-07:00"
              end: "2026-10-22T05:57:24-07:00"
              began: "2026-10-22T04:59:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c8d8c5d9f6970d84722ef9b8d933844e0ca624c9f075efa8b6925db73fb7011c
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 106608768
              windows:
                - start: "2026-10-22T04:59:40-07:00"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a16385-a739-7d02-bb3c-91b494692f62
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 0d5890be200455c6b30bf6f3fecf916917cbe0a51b25858342abd7ac97ddef05
              started: "2026-10-22T12:11:06.635Z"
              spawned: false
              bytes: 606483
              windows:
                - start: "2026-10-22T12:11:06.635Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a16388-246e-7f63-8a56-8030faac80db
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 42404cb3df8e525a88c7e7e06ca1e705434b6ad04081d93389964744e7becef9
              started: "2026-10-22T12:13:49.312Z"
              spawned: false
              bytes: 468029
              windows:
                - start: "2026-10-22T12:13:49.312Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a16389-e7e3-7b83-b2e5-be7e7000c3af
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 53270051d542b69ffa68ac63946416d67fc9809925a5b6d0da5578031da113f9
              started: "2026-10-22T12:15:46.091Z"
              spawned: false
              bytes: 483664
              windows:
                - start: "2026-10-22T12:15:46.091Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638a-810e-70a2-ae02-305e486d3f94
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 94d4f24c41960c868664ddbc1d3ace5d790b646d75a369de42ab7d1c6c8696d4
              started: "2026-10-22T12:16:24.11Z"
              spawned: false
              bytes: 484388
              windows:
                - start: "2026-10-22T12:16:24.11Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638c-09b6-7bd3-bbef-7872745d3d2a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: dd1fc8a332153f760232991b4d5c0853dc0ed1edeaeb173845d18802257df7f1
              started: "2026-10-22T12:18:04.437Z"
              spawned: true
              bytes: 1220334
              windows:
                - start: "2026-10-22T12:18:04.437Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638c-40b3-7432-9bb5-491da977da77
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 2ba61ee1b44c7ae2867e104a6b22c58adbfb589876ba0c93365b75f98a16c38c
              started: "2026-10-22T12:18:18.521Z"
              spawned: true
              bytes: 1222539
              windows:
                - start: "2026-10-22T12:18:18.521Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638c-c122-73f0-84e1-124985c62b54
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 1758fbb99c3a3bf81c0c7bd0372e8b9a18183a5b96c058e94f1af888c6fa6c0f
              started: "2026-10-22T12:18:51.407Z"
              spawned: true
              bytes: 1163023
              windows:
                - start: "2026-10-22T12:18:51.407Z"
                  end: "2026-10-22T05:57:24-07:00"
      "n": 1
      timestamp: "2026-10-22T06:01:46-07:00"
      dispatched: "2026-10-22T05:58:05-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: A late negative lookup reports confirmed absence despite a current successor
          detail: cmd/internal/editorendpoint/owner.go:414-424 fences retirement against the captured binding but unconditionally returns Code=ok with EditorEndpoint=null afterward. If B replaces A while A's probe is pending, a subsequent nil response for A cannot establish absence of B. Recheck current authority under the same lock as retirement and return stale or validate the successor. Extend owner_test.go:234 to assert the delayed lookup response; line 260 currently discards it. ARCH-ORDER, ARCH-SECURE.
          family: observation-generation-fencing
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
          detail: cmd/internal/editorendpoint/owner.go:129-131 treats every Store.Publish error as uncertain publication. However, store.go:112-114 can reject encoding before invoking any writer, including codec.go:139-140 rejecting a candidate above 8 MiB. A valid near-capacity registry therefore becomes unavailable for lookup, unregister, and cleanup until restart after one oversized admission. Preflight candidate encoding and classify size exhaustion as capacity without poisoning or changing authority; reserve publication uncertainty for attempted writes. Add an owner-level regression proving no writer call and continued operation after refusal. ARCH-CONSTRAINTS, ARCH-ORDER.
          family: prepublication-refusal-isolation
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
          detail: cmd/internal/editorendpoint/owner_test.go:69-73 only appends notices and always succeeds; couchcmd/editor_endpoint_service_test.go:124-125 is a no-op. The native callback at nvim_live_test.go:65-68 only writes a token. None can hold or fail Superseded, although plan line 168 marks held-notification qualification complete. Add controllable notification completion/failure through the production seam and verify publication precedes delivery, failure preserves the successor, and shutdown cancels and joins an in-flight notification. Model pending/current attachment identity in the editor double. ARCH-MOCK, ARCH-ORDER, ARCH-PURPOSE.
          family: external-lifecycle-qualification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 20ddc55f-d1cb-45cd-84ce-e3e7f9a942dd
            repository: github.com/xianxu/pair
            issue_id: "000758"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/b8110d64d1141f22
            reviewed_head: 3a98841ea83eb10e2af5189bb66c8754fd0db5a7
            slot: pair:6
            requested_at: "2026-10-22T13:13:31.726687Z"
          phase: finished
          pid: 64593
          started_at: "2026-10-22T13:13:31.903171Z"
          finished_at: "2026-10-22T13:17:02.880271Z"
          session_id: 01a163be-d1f5-7f70-b15f-dd18c9c24c50
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 3a98841ea83eb10e2af5189bb66c8754fd0db5a7
        tracker_head: 7b177d9bfce5ec4380df7b2da60e8531a4dfc6cd
        branch_point: 1e70379fba4a3653258f5542ac1d1aecf0ca1b43
        since: "2026-10-22T04:59:40-07:00"
        until: "2026-10-22T06:12:32-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T04:59:40-07:00"
              end: "2026-10-22T06:12:32-07:00"
              began: "2026-10-22T04:59:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: a1f54b93d859910a596d86ee230886d65033c764f29026aca8ffc19987a987c6
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 107469346
              windows:
                - start: "2026-10-22T04:59:40-07:00"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a16385-a739-7d02-bb3c-91b494692f62
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 0d5890be200455c6b30bf6f3fecf916917cbe0a51b25858342abd7ac97ddef05
              started: "2026-10-22T12:11:06.635Z"
              spawned: false
              bytes: 606483
              windows:
                - start: "2026-10-22T12:11:06.635Z"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a16388-246e-7f63-8a56-8030faac80db
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 42404cb3df8e525a88c7e7e06ca1e705434b6ad04081d93389964744e7becef9
              started: "2026-10-22T12:13:49.312Z"
              spawned: false
              bytes: 468029
              windows:
                - start: "2026-10-22T12:13:49.312Z"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a16389-e7e3-7b83-b2e5-be7e7000c3af
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 53270051d542b69ffa68ac63946416d67fc9809925a5b6d0da5578031da113f9
              started: "2026-10-22T12:15:46.091Z"
              spawned: false
              bytes: 483664
              windows:
                - start: "2026-10-22T12:15:46.091Z"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a1638a-810e-70a2-ae02-305e486d3f94
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 94d4f24c41960c868664ddbc1d3ace5d790b646d75a369de42ab7d1c6c8696d4
              started: "2026-10-22T12:16:24.11Z"
              spawned: false
              bytes: 484388
              windows:
                - start: "2026-10-22T12:16:24.11Z"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a1638c-09b6-7bd3-bbef-7872745d3d2a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: dd1fc8a332153f760232991b4d5c0853dc0ed1edeaeb173845d18802257df7f1
              started: "2026-10-22T12:18:04.437Z"
              spawned: true
              bytes: 1220334
              windows:
                - start: "2026-10-22T12:18:04.437Z"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a1638c-40b3-7432-9bb5-491da977da77
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 4d65b38e7da5231be3dd03d0a6e62f60dd03f8cb223c2b8311ea26d8a76ade9c
              started: "2026-10-22T12:18:18.521Z"
              spawned: true
              bytes: 1616100
              windows:
                - start: "2026-10-22T12:18:18.521Z"
                  end: "2026-10-22T06:12:32-07:00"
            - harness: codex
              id: 01a1638c-c122-73f0-84e1-124985c62b54
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 1758fbb99c3a3bf81c0c7bd0372e8b9a18183a5b96c058e94f1af888c6fa6c0f
              started: "2026-10-22T12:18:51.407Z"
              spawned: true
              bytes: 1163023
              windows:
                - start: "2026-10-22T12:18:51.407Z"
                  end: "2026-10-22T06:12:32-07:00"
        cuts:
            - dispatched: "2026-10-22T05:58:05-07:00"
              finalized: "2026-10-22T06:01:46-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-22T06:17:03-07:00"
      dispatched: "2026-10-22T06:13:28-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: owner.go:433-456 fences negative retirement and response against current authority. TestOwnerLateNegativeLookupCannotRetireSuccessor covers nil, mismatch, death and document absence; all four fail with the pre-fix owner supplied through a temporary Go overlay and pass at head.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Store.publish distinguishes pre-writer refusal from attempted IO. TestOwnerEncodedCapacityRefusalKeepsAuthorityUsable fails with the pre-fix owner and passes at head, verifying no writer call and continued lookup, unregister and cleanup. Additional tests preserve uncertainty for writer-returned capacity errors.
          round: 2
        - id: BR-3
          disposition: addressed
          note: owner_test.go:740-877 exercises held, failed and canceled Superseded delivery through the production RPC seam, including durable publication before delivery, successor preservation and joined shutdown. The fixture tracks pending/current attachment identity. This correction adds qualification without changing production behavior.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Important
          title: Checked publication-byte qualification has no byte accounting
          detail: 'ARCH-CONSTRAINTS: workshop/plans/000758-editor-endpoints-plan.md:119,157,167 explicitly promises and marks N8/16 publication-byte qualification complete. However, owner_test.go:184-195 counts only writes, and TestOwnerBatchOnePublicationAndPlateau at line 431 checks publication calls and retained-record counts without recording encoded sizes or cumulative bytes. Instrument the injected writer and extend the N8/16 scenarios to verify snapshot-byte growth, retry/no-op zero-write behavior and bounded repeated-reopen size.'
          family: operating-envelope-qualification
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: cb392adb-75e3-43d1-8b56-8e39a47c6e1b
            repository: github.com/xianxu/pair
            issue_id: "000758"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e88d998153dfca81
            reviewed_head: be457f2be2f39f88610d131c03955aef84989843
            slot: pair:6
            requested_at: "2026-10-22T13:25:47.497224Z"
          phase: finished
          pid: 46658
          started_at: "2026-10-22T13:25:47.63079Z"
          finished_at: "2026-10-22T13:28:01.35204Z"
          session_id: 01a163ca-0b6f-72b1-8956-fe0bd7f66262
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: be457f2be2f39f88610d131c03955aef84989843
        tracker_head: 7b177d9bfce5ec4380df7b2da60e8531a4dfc6cd
        branch_point: 1e70379fba4a3653258f5542ac1d1aecf0ca1b43
        since: "2026-10-22T04:59:40-07:00"
        until: "2026-10-22T06:24:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T04:59:40-07:00"
              end: "2026-10-22T06:24:54-07:00"
              began: "2026-10-22T04:59:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 481d596893d975fafc85fc080f66acea738367b72b3380b3026fc16d4c6c4613
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 107925352
              windows:
                - start: "2026-10-22T04:59:40-07:00"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a16385-a739-7d02-bb3c-91b494692f62
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 0d5890be200455c6b30bf6f3fecf916917cbe0a51b25858342abd7ac97ddef05
              started: "2026-10-22T12:11:06.635Z"
              spawned: false
              bytes: 606483
              windows:
                - start: "2026-10-22T12:11:06.635Z"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a16388-246e-7f63-8a56-8030faac80db
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 42404cb3df8e525a88c7e7e06ca1e705434b6ad04081d93389964744e7becef9
              started: "2026-10-22T12:13:49.312Z"
              spawned: false
              bytes: 468029
              windows:
                - start: "2026-10-22T12:13:49.312Z"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a16389-e7e3-7b83-b2e5-be7e7000c3af
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 53270051d542b69ffa68ac63946416d67fc9809925a5b6d0da5578031da113f9
              started: "2026-10-22T12:15:46.091Z"
              spawned: false
              bytes: 483664
              windows:
                - start: "2026-10-22T12:15:46.091Z"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a1638a-810e-70a2-ae02-305e486d3f94
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 94d4f24c41960c868664ddbc1d3ace5d790b646d75a369de42ab7d1c6c8696d4
              started: "2026-10-22T12:16:24.11Z"
              spawned: false
              bytes: 484388
              windows:
                - start: "2026-10-22T12:16:24.11Z"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a1638c-09b6-7bd3-bbef-7872745d3d2a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: dd1fc8a332153f760232991b4d5c0853dc0ed1edeaeb173845d18802257df7f1
              started: "2026-10-22T12:18:04.437Z"
              spawned: true
              bytes: 1220334
              windows:
                - start: "2026-10-22T12:18:04.437Z"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a1638c-40b3-7432-9bb5-491da977da77
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 2f5f6a297b30241ce483bf31a46d353691835eb8f270a7687cae260a49568996
              started: "2026-10-22T12:18:18.521Z"
              spawned: true
              bytes: 1831069
              windows:
                - start: "2026-10-22T12:18:18.521Z"
                  end: "2026-10-22T06:24:54-07:00"
            - harness: codex
              id: 01a1638c-c122-73f0-84e1-124985c62b54
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 1758fbb99c3a3bf81c0c7bd0372e8b9a18183a5b96c058e94f1af888c6fa6c0f
              started: "2026-10-22T12:18:51.407Z"
              spawned: true
              bytes: 1163023
              windows:
                - start: "2026-10-22T12:18:51.407Z"
                  end: "2026-10-22T06:24:54-07:00"
        cuts:
            - dispatched: "2026-10-22T05:58:05-07:00"
              finalized: "2026-10-22T06:01:46-07:00"
              finalizing: false
            - dispatched: "2026-10-22T06:13:28-07:00"
              finalized: "2026-10-22T06:17:03-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-22T06:28:02-07:00"
      dispatched: "2026-10-22T06:25:44-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-4
          disposition: addressed
          note: owner_test.go:155-179,210,459-623 records actual encoded writer payload lengths and verifies N8/16 snapshot/cumulative bounds, zero retry/no-op/reload writes, single batch publication, and repeated-reopen size plateau. The race run passed; negative controls reject duplicate publications and retained attempt growth. This correction adds qualification without changing production behavior.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: be457f2be2f39f88610d131c03955aef84989843
      blocked: false
---

# Gate ledger — pair#758 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T06:01:46-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `observation-generation-fencing` (kind: defect) A late negative lookup reports confirmed absence despite a current successor
  cmd/internal/editorendpoint/owner.go:414-424 fences retirement against the captured binding but unconditionally returns Code=ok with EditorEndpoint=null afterward. If B replaces A while A's probe is pending, a subsequent nil response for A cannot establish absence of B. Recheck current authority under the same lock as retirement and return stale or validate the successor. Extend owner_test.go:234 to assert the delayed lookup response; line 260 currently discards it. ARCH-ORDER, ARCH-SECURE.
- **BR-2** [Critical] `prepublication-refusal-isolation` (kind: defect) Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
  cmd/internal/editorendpoint/owner.go:129-131 treats every Store.Publish error as uncertain publication. However, store.go:112-114 can reject encoding before invoking any writer, including codec.go:139-140 rejecting a candidate above 8 MiB. A valid near-capacity registry therefore becomes unavailable for lookup, unregister, and cleanup until restart after one oversized admission. Preflight candidate encoding and classify size exhaustion as capacity without poisoning or changing authority; reserve publication uncertainty for attempted writes. Add an owner-level regression proving no writer call and continued operation after refusal. ARCH-CONSTRAINTS, ARCH-ORDER.
- **BR-3** [Important] `external-lifecycle-qualification` (kind: defect) Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
  cmd/internal/editorendpoint/owner_test.go:69-73 only appends notices and always succeeds; couchcmd/editor_endpoint_service_test.go:124-125 is a no-op. The native callback at nvim_live_test.go:65-68 only writes a token. None can hold or fail Superseded, although plan line 168 marks held-notification qualification complete. Add controllable notification completion/failure through the production seam and verify publication precedes delivery, failure preserves the successor, and shutdown cancels and joins an in-flight notification. Model pending/current attachment identity in the editor double. ARCH-MOCK, ARCH-ORDER, ARCH-PURPOSE.

## Round 2 — 2026-10-22T06:17:03-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — owner.go:433-456 fences negative retirement and response against current authority. TestOwnerLateNegativeLookupCannotRetireSuccessor covers nil, mismatch, death and document absence; all four fail with the pre-fix owner supplied through a temporary Go overlay and pass at head.
- BR-2 — addressed — Store.publish distinguishes pre-writer refusal from attempted IO. TestOwnerEncodedCapacityRefusalKeepsAuthorityUsable fails with the pre-fix owner and passes at head, verifying no writer call and continued lookup, unregister and cleanup. Additional tests preserve uncertainty for writer-returned capacity errors.
- BR-3 — addressed — owner_test.go:740-877 exercises held, failed and canceled Superseded delivery through the production RPC seam, including durable publication before delivery, successor preservation and joined shutdown. The fixture tracks pending/current attachment identity. This correction adds qualification without changing production behavior.

### Raised

- **BR-4** [Important] `operating-envelope-qualification` (kind: defect) Checked publication-byte qualification has no byte accounting
  ARCH-CONSTRAINTS: workshop/plans/000758-editor-endpoints-plan.md:119,157,167 explicitly promises and marks N8/16 publication-byte qualification complete. However, owner_test.go:184-195 counts only writes, and TestOwnerBatchOnePublicationAndPlateau at line 431 checks publication calls and retained-record counts without recording encoded sizes or cumulative bytes. Instrument the injected writer and extend the N8/16 scenarios to verify snapshot-byte growth, retry/no-op zero-write behavior and bounded repeated-reopen size.

## Round 3 — 2026-10-22T06:28:02-07:00 (codex) — passed

### Disposed

- BR-4 — addressed — owner_test.go:155-179,210,459-623 records actual encoded writer payload lengths and verifies N8/16 snapshot/cumulative bounds, zero retry/no-op/reload writes, single batch publication, and repeated-reopen size plateau. The race run passed; negative controls reject duplicate publications and retained attempt growth. This correction adds qualification without changing production behavior.

## Open findings

(none — every finding has been disposed)
