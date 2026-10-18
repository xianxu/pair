---
gate: boundary-review
issue: 574
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c1a27c7f-2f04-4a32-a784-dfab7b64b45e
            repository: github.com/xianxu/pair
            issue_id: "000574"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/b05fce0cf4d231fb
            reviewed_head: f7dcd06d22ebd05a56ce8b618ae0167df7bc5285
            slot: pair:3
            requested_at: "2026-10-18T02:53:28.302702Z"
          phase: finished
          pid: 30373
          started_at: "2026-10-18T02:53:28.559104Z"
          finished_at: "2026-10-18T02:58:32.920539Z"
          session_id: 01a14ced-b496-7550-9bec-ce977a6d9751
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: f7dcd06d22ebd05a56ce8b618ae0167df7bc5285
        tracker_head: fa5dcae2b3e85f33fea20b6fc3706045497f4c22
        branch_point: 1ee00c35969770c9e6e6a01282393613617b0305
        since: "2026-10-17T18:43:29-07:00"
        until: "2026-10-17T19:52:46-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T18:43:29-07:00"
              end: "2026-10-17T19:52:46-07:00"
              began: "2026-10-17T18:43:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 7e4a086dc0446a516e040ce7e6e12cf1bb1077b5de8209b7bc8375ca646b210f
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 47005728
              windows:
                - start: "2026-10-17T18:43:29-07:00"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cad-be2e-7273-91f4-2c10da85764f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 1af66ee662a4086ed77b3d425315291eef93ad38bd2ea3f886c9b793824dae7c
              started: "2026-10-18T01:43:37.328Z"
              spawned: true
              bytes: 902917
              windows:
                - start: "2026-10-18T01:43:37.328Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14caf-c568-7220-8fe1-2067e193e741
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: bfbaed435befe78a0c9be422ce7a95008b6043144aaad9781feb2760d4ef25e5
              started: "2026-10-18T01:45:50.331Z"
              spawned: true
              bytes: 625717
              windows:
                - start: "2026-10-18T01:45:50.331Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cba-05a4-7e70-b948-7e18755a0929
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 63e75471785dafcf92316cd5c1a951cdf95b291ebf6bce66b2e68c0338fd8612
              started: "2026-10-18T01:57:02.796Z"
              spawned: false
              bytes: 674143
              windows:
                - start: "2026-10-18T01:57:02.796Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cbb-88ae-7231-8f72-9c8c1ab1014e
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 6e4be5ad2cee4d6b350a95785f439993d10d620bbaf442f501a8e6322d746a14
              started: "2026-10-18T01:58:41.421Z"
              spawned: false
              bytes: 517552
              windows:
                - start: "2026-10-18T01:58:41.421Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cbc-9dd7-7f30-b557-2e8e66e2b6fc
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d580dcc1419a661bbc63e68033f8d3cd69f6106f01301ed3df10eb55f24fefc8
              started: "2026-10-18T01:59:52.349Z"
              spawned: false
              bytes: 504135
              windows:
                - start: "2026-10-18T01:59:52.349Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cbd-038e-7052-8469-e69950cddf5a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: fc98168ccc6abf5c055daab7b1146ce905c7f2dc35538a5ad09bb144fae8b7a3
              started: "2026-10-18T02:00:18.33Z"
              spawned: false
              bytes: 446775
              windows:
                - start: "2026-10-18T02:00:18.33Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cbd-dd8f-7ef2-833b-e01083dc15dc
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: e2e3f684b7a35e967a25bbf2cf49c207b9988597dfbee5c86ab6c8c4a29ec6a2
              started: "2026-10-18T02:01:14.07Z"
              spawned: true
              bytes: 977737
              windows:
                - start: "2026-10-18T02:01:14.07Z"
                  end: "2026-10-17T19:52:46-07:00"
            - harness: codex
              id: 01a14cbe-1a69-7ea0-ae07-a7f1bd9f41e7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: b5d25b4c9fe3392d4223a226cb5e56bc84f5731e20f407996d14b68d48f3b1a6
              started: "2026-10-18T02:01:29.613Z"
              spawned: true
              bytes: 1131600
              windows:
                - start: "2026-10-18T02:01:29.613Z"
                  end: "2026-10-17T19:52:46-07:00"
        queue_waits:
            - attempt_id: c4b8c5baa145861a3cfb06c7513acb31
              slot: pair:3
              record_digest: 1b37e9b0b7786f07ea20ad3984224641b5f24df6be8c9cc5fa2a130a45d06c53
              start: "2026-10-18T02:30:57.643161Z"
              end: "2026-10-18T02:39:23.465143Z"
              queue_seconds: 505.821982
              raw_seconds: 505.821982
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 29.706033333333334
      "n": 1
      timestamp: "2026-10-17T19:58:34-07:00"
      dispatched: "2026-10-17T19:53:25-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: RecoveryResumeResult is incorrectly treated as proof of Resume mode
          detail: 'cmd/internal/couchtty/console_lifecycle.go:98 unconditionally labels every RecoveryResumeResult as recovered (Resume). That result also represents Continue (couchcore/recovery_prepared.go:91) and Fresh (recovery_prepared.go:435), including recovery-confirmation completions. The hub can therefore be told a conversation resumed when it actually started fresh. ARCH-PURPOSE: carry the actual typed recovery mode through the result and test successful Continue, Resume, Fresh, and confirmation paths through finishOperation.'
          family: preserve-confirmed-recovery-mode
          round: 1
        - id: BR-2
          severity: Critical
          title: Started notices use the preceding menu agent instead of the adopted child's agent
          detail: 'cmd/internal/couchtty/console_lifecycle.go:79 obtains Agent from the existing menu row and never overrides it from child.Record.Args.Stack. A successful agent switch can consequently announce the previous agent; a never-started row can omit the agent entirely. The confirmed launch already supplies the actual agent (couchcore/couch.go:488 and switchagent.go:415), while menu refresh occurs asynchronously after attachment. ARCH-PURPOSE: prefer the completed child''s agent and add regressions with empty and conflicting pre-operation menu metadata.'
          family: prefer-confirmed-completion-metadata
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1e15afc6-c4c8-45e2-a7fe-f6425a7a4e96
            repository: github.com/xianxu/pair
            issue_id: "000574"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/d5acc848102d83ab
            reviewed_head: 50f8d6831f6cf62c1ea481de5da9190604022082
            slot: pair:3
            requested_at: "2026-10-18T03:23:37.819756Z"
          phase: finished
          pid: 21713
          started_at: "2026-10-18T03:23:38.019848Z"
          finished_at: "2026-10-18T03:26:17.245684Z"
          session_id: 01a14d09-5056-75b2-ade7-6f1df363b7ab
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 50f8d6831f6cf62c1ea481de5da9190604022082
        tracker_head: fa5dcae2b3e85f33fea20b6fc3706045497f4c22
        branch_point: 1ee00c35969770c9e6e6a01282393613617b0305
        since: "2026-10-17T18:43:29-07:00"
        until: "2026-10-17T20:23:01-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T18:43:29-07:00"
              end: "2026-10-17T20:23:01-07:00"
              began: "2026-10-17T18:43:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 5f6896ea1e30b1be4c0f95fa749d6a164508fa397249a87d2cc109cb4fe7bf0c
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 47970476
              windows:
                - start: "2026-10-17T18:43:29-07:00"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cad-be2e-7273-91f4-2c10da85764f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 1af66ee662a4086ed77b3d425315291eef93ad38bd2ea3f886c9b793824dae7c
              started: "2026-10-18T01:43:37.328Z"
              spawned: true
              bytes: 902917
              windows:
                - start: "2026-10-18T01:43:37.328Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14caf-c568-7220-8fe1-2067e193e741
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: bfbaed435befe78a0c9be422ce7a95008b6043144aaad9781feb2760d4ef25e5
              started: "2026-10-18T01:45:50.331Z"
              spawned: true
              bytes: 625717
              windows:
                - start: "2026-10-18T01:45:50.331Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cba-05a4-7e70-b948-7e18755a0929
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 63e75471785dafcf92316cd5c1a951cdf95b291ebf6bce66b2e68c0338fd8612
              started: "2026-10-18T01:57:02.796Z"
              spawned: false
              bytes: 674143
              windows:
                - start: "2026-10-18T01:57:02.796Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cbb-88ae-7231-8f72-9c8c1ab1014e
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 6e4be5ad2cee4d6b350a95785f439993d10d620bbaf442f501a8e6322d746a14
              started: "2026-10-18T01:58:41.421Z"
              spawned: false
              bytes: 517552
              windows:
                - start: "2026-10-18T01:58:41.421Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cbc-9dd7-7f30-b557-2e8e66e2b6fc
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d580dcc1419a661bbc63e68033f8d3cd69f6106f01301ed3df10eb55f24fefc8
              started: "2026-10-18T01:59:52.349Z"
              spawned: false
              bytes: 504135
              windows:
                - start: "2026-10-18T01:59:52.349Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cbd-038e-7052-8469-e69950cddf5a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: fc98168ccc6abf5c055daab7b1146ce905c7f2dc35538a5ad09bb144fae8b7a3
              started: "2026-10-18T02:00:18.33Z"
              spawned: false
              bytes: 446775
              windows:
                - start: "2026-10-18T02:00:18.33Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cbd-dd8f-7ef2-833b-e01083dc15dc
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: e2e3f684b7a35e967a25bbf2cf49c207b9988597dfbee5c86ab6c8c4a29ec6a2
              started: "2026-10-18T02:01:14.07Z"
              spawned: true
              bytes: 977737
              windows:
                - start: "2026-10-18T02:01:14.07Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cbe-1a69-7ea0-ae07-a7f1bd9f41e7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: b5d25b4c9fe3392d4223a226cb5e56bc84f5731e20f407996d14b68d48f3b1a6
              started: "2026-10-18T02:01:29.613Z"
              spawned: true
              bytes: 1131600
              windows:
                - start: "2026-10-18T02:01:29.613Z"
                  end: "2026-10-17T20:23:01-07:00"
            - harness: codex
              id: 01a14cf2-a2c0-7d31-9ab8-f2633b81275d
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: edb340fdfc07f713102cb68389ce93888b09f1d9eac816a124af8a729881b2a0
              started: "2026-10-18T02:58:52.31Z"
              spawned: true
              bytes: 665994
              windows:
                - start: "2026-10-18T02:58:52.31Z"
                  end: "2026-10-17T20:23:01-07:00"
        queue_waits:
            - attempt_id: c4b8c5baa145861a3cfb06c7513acb31
              slot: pair:3
              record_digest: 1b37e9b0b7786f07ea20ad3984224641b5f24df6be8c9cc5fa2a130a45d06c53
              start: "2026-10-18T02:30:57.643161Z"
              end: "2026-10-18T02:39:23.465143Z"
              queue_seconds: 505.821982
              raw_seconds: 505.821982
              exact: true
            - attempt_id: 84513702e3ecb4af4b7e6060f55c1d45
              slot: pair:3
              record_digest: 52ca6709cfb8ca0bdd87257e325ff49f120be0f3632be6915a4b2d250299361d
              start: "2026-10-18T03:01:34.179579Z"
              end: "2026-10-18T03:11:56.227011Z"
              queue_seconds: 622.047432
              raw_seconds: 622.047432
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 45.42231666666667
        cuts:
            - dispatched: "2026-10-17T19:53:25-07:00"
              finalized: "2026-10-17T19:58:34-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-17T20:26:18-07:00"
      dispatched: "2026-10-17T20:23:34-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Recovery constructors preserve Continue, ResumeDisk and Fresh; finishOperation projects the typed mode, including confirmation results. Recovery regressions pass; restoring unconditional Resume makes the Console regression fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Completion projection prefers the adopted child's Args.Stack. Regressions cover empty/conflicting menu agents across operation types; removing that preference makes the regression fail.
          round: 2
      findings:
        - id: BR-3
          severity: Critical
          title: Pending lifecycle notices remain assigned to the previous owning hub
          detail: 'cmd/internal/couchcmd/lifecycle_notices.go:167 resolves ownership only when observing a fact, and :210-224 refreshes connections without rerouting existing facts. A disposable regression collected a fact under hub A, moved its slot to hub B, made B ready, and drained the cohort: no notice reached B because the fact still belonged to A. If A remains available, it can receive the stale projection instead. ARCH-PURPOSE, ARCH-ORDER: resolve current ownership before freezing/admitting unsent projections, preserve accepted-input uncertainty, and test ownership changes before drain and admission.'
          family: revalidate-current-recipient-ownership
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3d84c556-44af-44da-a8fc-600c358873f0
            repository: github.com/xianxu/pair
            issue_id: "000574"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/1e71b7e610ec96de
            reviewed_head: f962a74b9031836f2783f9d472cc395ff3fd5b57
            slot: pair:3
            requested_at: "2026-10-18T03:52:18.95488Z"
          phase: finished
          pid: 62494
          started_at: "2026-10-18T03:52:19.197935Z"
          finished_at: "2026-10-18T03:57:37.790147Z"
          session_id: 01a14d23-941f-7b51-b04f-626d0d5bca5c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: f962a74b9031836f2783f9d472cc395ff3fd5b57
        tracker_head: fa5dcae2b3e85f33fea20b6fc3706045497f4c22
        branch_point: 1ee00c35969770c9e6e6a01282393613617b0305
        since: "2026-10-17T18:43:29-07:00"
        until: "2026-10-17T20:51:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T18:43:29-07:00"
              end: "2026-10-17T20:51:45-07:00"
              began: "2026-10-17T18:43:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 2fd1ba41b217a7a7010ddb657a00a4827b29ba0b531e0f85f167f4a3082121f8
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 49184038
              windows:
                - start: "2026-10-17T18:43:29-07:00"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cad-be2e-7273-91f4-2c10da85764f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 1af66ee662a4086ed77b3d425315291eef93ad38bd2ea3f886c9b793824dae7c
              started: "2026-10-18T01:43:37.328Z"
              spawned: true
              bytes: 902917
              windows:
                - start: "2026-10-18T01:43:37.328Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14caf-c568-7220-8fe1-2067e193e741
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: bfbaed435befe78a0c9be422ce7a95008b6043144aaad9781feb2760d4ef25e5
              started: "2026-10-18T01:45:50.331Z"
              spawned: true
              bytes: 625717
              windows:
                - start: "2026-10-18T01:45:50.331Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cba-05a4-7e70-b948-7e18755a0929
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 63e75471785dafcf92316cd5c1a951cdf95b291ebf6bce66b2e68c0338fd8612
              started: "2026-10-18T01:57:02.796Z"
              spawned: false
              bytes: 674143
              windows:
                - start: "2026-10-18T01:57:02.796Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cbb-88ae-7231-8f72-9c8c1ab1014e
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 6e4be5ad2cee4d6b350a95785f439993d10d620bbaf442f501a8e6322d746a14
              started: "2026-10-18T01:58:41.421Z"
              spawned: false
              bytes: 517552
              windows:
                - start: "2026-10-18T01:58:41.421Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cbc-9dd7-7f30-b557-2e8e66e2b6fc
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d580dcc1419a661bbc63e68033f8d3cd69f6106f01301ed3df10eb55f24fefc8
              started: "2026-10-18T01:59:52.349Z"
              spawned: false
              bytes: 504135
              windows:
                - start: "2026-10-18T01:59:52.349Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cbd-038e-7052-8469-e69950cddf5a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: fc98168ccc6abf5c055daab7b1146ce905c7f2dc35538a5ad09bb144fae8b7a3
              started: "2026-10-18T02:00:18.33Z"
              spawned: false
              bytes: 446775
              windows:
                - start: "2026-10-18T02:00:18.33Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cbd-dd8f-7ef2-833b-e01083dc15dc
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: e2e3f684b7a35e967a25bbf2cf49c207b9988597dfbee5c86ab6c8c4a29ec6a2
              started: "2026-10-18T02:01:14.07Z"
              spawned: true
              bytes: 977737
              windows:
                - start: "2026-10-18T02:01:14.07Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cbe-1a69-7ea0-ae07-a7f1bd9f41e7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: b5d25b4c9fe3392d4223a226cb5e56bc84f5731e20f407996d14b68d48f3b1a6
              started: "2026-10-18T02:01:29.613Z"
              spawned: true
              bytes: 1131600
              windows:
                - start: "2026-10-18T02:01:29.613Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14cf2-a2c0-7d31-9ab8-f2633b81275d
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: edb340fdfc07f713102cb68389ce93888b09f1d9eac816a124af8a729881b2a0
              started: "2026-10-18T02:58:52.31Z"
              spawned: true
              bytes: 665994
              windows:
                - start: "2026-10-18T02:58:52.31Z"
                  end: "2026-10-17T20:51:45-07:00"
            - harness: codex
              id: 01a14d0d-ff8c-7d40-b4cf-f897593b39d9
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 07d859dbe22a3fce9880ec2b5c4d3bbc24c75cebc313862660ff6d248fbffd4b
              started: "2026-10-18T03:28:45.637Z"
              spawned: true
              bytes: 526818
              windows:
                - start: "2026-10-18T03:28:45.637Z"
                  end: "2026-10-17T20:51:45-07:00"
        queue_waits:
            - attempt_id: c4b8c5baa145861a3cfb06c7513acb31
              slot: pair:3
              record_digest: 1b37e9b0b7786f07ea20ad3984224641b5f24df6be8c9cc5fa2a130a45d06c53
              start: "2026-10-18T02:30:57.643161Z"
              end: "2026-10-18T02:39:23.465143Z"
              queue_seconds: 505.821982
              raw_seconds: 505.821982
              exact: true
            - attempt_id: 84513702e3ecb4af4b7e6060f55c1d45
              slot: pair:3
              record_digest: 52ca6709cfb8ca0bdd87257e325ff49f120be0f3632be6915a4b2d250299361d
              start: "2026-10-18T03:01:34.179579Z"
              end: "2026-10-18T03:11:56.227011Z"
              queue_seconds: 622.047432
              raw_seconds: 622.047432
              exact: true
            - attempt_id: a00e90b8bd7736b8f95144e448531520
              slot: pair:3
              record_digest: 796e33601a219cb2c52d895be9c8c562bd2aa4669dd5c9fbc9a1683e62d9fbe0
              start: "2026-10-18T03:37:12.545773Z"
              end: "2026-10-18T03:39:59.164246Z"
              queue_seconds: 166.618473
              raw_seconds: 166.618473
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 61.42311666666666
        cuts:
            - dispatched: "2026-10-17T19:53:25-07:00"
              finalized: "2026-10-17T19:58:34-07:00"
              finalizing: false
            - dispatched: "2026-10-17T20:23:34-07:00"
              finalized: "2026-10-17T20:26:18-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-17T20:57:39-07:00"
      dispatched: "2026-10-17T20:52:15-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: Ownership changes discard unadmitted projections, notify the new owner, and preserve accepted-input uncertainty. TestLifecyclePublisherDropsMovedOwnership passes all four phases; removing the ownershipChanged call makes the regression fail. The disposable checkout was removed.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: f962a74b9031836f2783f9d472cc395ff3fd5b57
      blocked: false
---

# Gate ledger — pair#574 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T19:58:34-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `preserve-confirmed-recovery-mode` RecoveryResumeResult is incorrectly treated as proof of Resume mode
  cmd/internal/couchtty/console_lifecycle.go:98 unconditionally labels every RecoveryResumeResult as recovered (Resume). That result also represents Continue (couchcore/recovery_prepared.go:91) and Fresh (recovery_prepared.go:435), including recovery-confirmation completions. The hub can therefore be told a conversation resumed when it actually started fresh. ARCH-PURPOSE: carry the actual typed recovery mode through the result and test successful Continue, Resume, Fresh, and confirmation paths through finishOperation.
- **BR-2** [Critical] `prefer-confirmed-completion-metadata` Started notices use the preceding menu agent instead of the adopted child's agent
  cmd/internal/couchtty/console_lifecycle.go:79 obtains Agent from the existing menu row and never overrides it from child.Record.Args.Stack. A successful agent switch can consequently announce the previous agent; a never-started row can omit the agent entirely. The confirmed launch already supplies the actual agent (couchcore/couch.go:488 and switchagent.go:415), while menu refresh occurs asynchronously after attachment. ARCH-PURPOSE: prefer the completed child's agent and add regressions with empty and conflicting pre-operation menu metadata.

## Round 2 — 2026-10-17T20:26:18-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Recovery constructors preserve Continue, ResumeDisk and Fresh; finishOperation projects the typed mode, including confirmation results. Recovery regressions pass; restoring unconditional Resume makes the Console regression fail.
- BR-2 — addressed — Completion projection prefers the adopted child's Args.Stack. Regressions cover empty/conflicting menu agents across operation types; removing that preference makes the regression fail.

### Raised

- **BR-3** [Critical] `revalidate-current-recipient-ownership` Pending lifecycle notices remain assigned to the previous owning hub
  cmd/internal/couchcmd/lifecycle_notices.go:167 resolves ownership only when observing a fact, and :210-224 refreshes connections without rerouting existing facts. A disposable regression collected a fact under hub A, moved its slot to hub B, made B ready, and drained the cohort: no notice reached B because the fact still belonged to A. If A remains available, it can receive the stale projection instead. ARCH-PURPOSE, ARCH-ORDER: resolve current ownership before freezing/admitting unsent projections, preserve accepted-input uncertainty, and test ownership changes before drain and admission.

## Round 3 — 2026-10-17T20:57:39-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — Ownership changes discard unadmitted projections, notify the new owner, and preserve accepted-input uncertainty. TestLifecyclePublisherDropsMovedOwnership passes all four phases; removing the ownershipChanged call makes the regression fail. The disposable checkout was removed.

## Open findings

(none — every finding has been disposed)
