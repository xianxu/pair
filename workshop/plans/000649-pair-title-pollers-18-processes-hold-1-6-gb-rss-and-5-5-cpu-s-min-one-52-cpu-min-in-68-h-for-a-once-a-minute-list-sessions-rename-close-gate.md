---
gate: boundary-review
issue: 649
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 37005df4-831d-4c83-808b-90c0b85a747b
            repository: github.com/xianxu/pair
            issue_id: "000649"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/153fcd4112efa65a
            reviewed_head: 52da7b8eb248aeecafb3035f62958a5549c76529
            slot: pair:6
            requested_at: "2026-10-18T10:32:08.751237Z"
          phase: finished
          pid: 89695
          started_at: "2026-10-18T10:32:08.987902Z"
          finished_at: "2026-10-18T10:34:56.303812Z"
          session_id: 01a14e91-a1ed-77d0-ad7a-f6b19888a995
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 52da7b8eb248aeecafb3035f62958a5549c76529
        tracker_head: 1dccf4e85cd186f9a037c58d299d781c64d2c782
        branch_point: 3851b7338ff107d9b676d1db4eda7c7f21a2aa62
        since: "2026-10-17T11:43:27-07:00"
        until: "2026-10-18T03:31:25-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T11:43:27-07:00"
              end: "2026-10-18T03:31:25-07:00"
              began: "2026-10-17T11:43:27-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2845b94b7babc84a149e21ab38f029203271b1e4e2023d93920fd09f32e38d00
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 6388802
              windows:
                - start: "2026-10-17T11:43:27-07:00"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b2d-98d1-7683-949a-1c6b03c5d8f3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 094913c1a3068aad86510ba3bfe917b9e401068141645278681d427c419da72d
              started: "2026-10-17T18:44:01.949Z"
              spawned: true
              bytes: 942386
              windows:
                - start: "2026-10-17T18:44:01.949Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b32-28e0-7892-82e2-cd1402a4400d
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: abaca4e08d6cd230a1b185a38b9eb509013e20955d02808b02ec276b2a1bb938
              started: "2026-10-17T18:49:01.682Z"
              spawned: false
              bytes: 652398
              windows:
                - start: "2026-10-17T18:49:01.682Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b35-0ced-7610-8f7f-a26658a38c33
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c1db662c46deb90a7b8c88bf602aa4c38d0babac5bffa5a139866314ce14cc69
              started: "2026-10-17T18:52:10.767Z"
              spawned: false
              bytes: 440105
              windows:
                - start: "2026-10-17T18:52:10.767Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b36-00a5-7883-ac24-203c32bbbc44
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: db54197bb2355ea2008f875d996998ce31d77d10a6aa3449feafb69830793318
              started: "2026-10-17T18:53:13.317Z"
              spawned: false
              bytes: 469184
              windows:
                - start: "2026-10-17T18:53:13.317Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b36-6196-7f23-9466-6916cd22f958
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 5958f9bee14eeebefd0b6a98a2b2c32aa5b188ead4432f18d85f2cba279b6a8b
              started: "2026-10-17T18:53:38.446Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-17T18:53:38.446Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b37-280c-7371-8b76-a1e062c8c9a6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 7a31ab53c5bbff5a08e7bde70a06a934b0505fd2769db839dd05579a7ff2ad37
              started: "2026-10-17T18:54:28.588Z"
              spawned: true
              bytes: 495054
              windows:
                - start: "2026-10-17T18:54:28.588Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b37-4791-7cb0-a3fa-3ecffe85808a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 73df336d4d328baeaa1fd74fcff0583b19a0cbc5745752411c828d566fe1810a
              started: "2026-10-17T18:54:36.634Z"
              spawned: true
              bytes: 641027
              windows:
                - start: "2026-10-17T18:54:36.634Z"
                  end: "2026-10-18T03:31:25-07:00"
            - harness: codex
              id: 01a14b37-8631-7082-a9b4-1f3e6c0de9da
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fb68d456a155d8cb366c5d78a60face0260c8238388e88d248d53277baa2b8bd
              started: "2026-10-17T18:54:52.672Z"
              spawned: true
              bytes: 1202656
              windows:
                - start: "2026-10-17T18:54:52.672Z"
                  end: "2026-10-18T03:31:25-07:00"
        queue_waits:
            - attempt_id: c0796f4dfe80368cf357507b001784e5
              slot: pair:6
              record_digest: e178e012249f1f6794e067cb4fc8beefd2a68bdbb7d7c7a56608a0dbe8ac6f8d
              start: "2026-10-18T09:27:08.258262Z"
              end: "2026-10-18T09:45:20.724502Z"
              queue_seconds: 1092.46624
              raw_seconds: 1092.46624
              exact: true
            - attempt_id: 0619b0f0bcec5545314c1d0545bc00d8
              slot: pair:6
              record_digest: 62c2fa4b0821c4a9e343fe7db9bd598976418c31edb22e43d3d0713d569d7a28
              start: "2026-10-18T09:55:55.306272Z"
              end: "2026-10-18T10:20:09.097075Z"
              queue_seconds: 1453.790803
              raw_seconds: 1453.790803
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 187.45245
      "n": 1
      timestamp: "2026-10-18T03:34:57-07:00"
      dispatched: "2026-10-18T03:32:05-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Title decisions omit wrapper activity merged into idle fading
          detail: cmd/internal/couchcmd/titles.go:168-172 computes frame eligibility and cmux heat from facts.Activity before cmd/internal/couchtty/console_activity.go:177-185 merges wrapper work. During silent active/background work, fading stays current while frame updates stop after two minutes and cmux heat can age incorrectly. The plan explicitly requires the final activity timestamp for both consumers. Inject admitted wrapper activity before title decisions and add a regression asserting both emitted titles and fading with stale persisted activity (ARCH-PURPOSE).
          family: shared-activity-consumer-consistency
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: The 1,000-tick heap regression never constructs a compact projection
          detail: cmd/internal/couchcmd/titles_test.go:211-213 replaces production IO with fakeCouchTitles, whose source returns constant facts without constructing TitleProjection or reading native history. The heap assertion therefore cannot detect retained projection/cache growth, and this workload has no history-work counters. Exercise real compact projections over disposable file-backed fixtures for 1,000 ticks, including growth and generation turnover, while asserting retained heap, cache retirement, warm history work, and command counts (ARCH-CONSTRAINTS, ARCH-PURPOSE).
          family: production-path-resource-verification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e29ff72a-a518-4374-9980-6b1d79716dd4
            repository: github.com/xianxu/pair
            issue_id: "000649"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a62b8366e3469799
            reviewed_head: 0160140172a8a79674f9d14d3c595906ccbfca7e
            slot: pair:6
            requested_at: "2026-10-18T11:15:33.153288Z"
          phase: finished
          pid: 63599
          started_at: "2026-10-18T11:15:33.344258Z"
          finished_at: "2026-10-18T11:17:36.347902Z"
          session_id: 01a14eb9-5f0d-7f50-9876-3c00305bfd65
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 0160140172a8a79674f9d14d3c595906ccbfca7e
        tracker_head: 302d930942f5db1022c1891280b88eed4704e3d7
        branch_point: 3851b7338ff107d9b676d1db4eda7c7f21a2aa62
        since: "2026-10-17T11:43:27-07:00"
        until: "2026-10-18T04:14:52-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T11:43:27-07:00"
              end: "2026-10-18T04:14:52-07:00"
              began: "2026-10-17T11:43:27-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 4388c4c576ff8c4a4820dde4ae47b846766a9f62029253445c35ac1e6eab0d8d
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 7460380
              windows:
                - start: "2026-10-17T11:43:27-07:00"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b2d-98d1-7683-949a-1c6b03c5d8f3
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 094913c1a3068aad86510ba3bfe917b9e401068141645278681d427c419da72d
              started: "2026-10-17T18:44:01.949Z"
              spawned: true
              bytes: 942386
              windows:
                - start: "2026-10-17T18:44:01.949Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b32-28e0-7892-82e2-cd1402a4400d
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: abaca4e08d6cd230a1b185a38b9eb509013e20955d02808b02ec276b2a1bb938
              started: "2026-10-17T18:49:01.682Z"
              spawned: false
              bytes: 652398
              windows:
                - start: "2026-10-17T18:49:01.682Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b35-0ced-7610-8f7f-a26658a38c33
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c1db662c46deb90a7b8c88bf602aa4c38d0babac5bffa5a139866314ce14cc69
              started: "2026-10-17T18:52:10.767Z"
              spawned: false
              bytes: 440105
              windows:
                - start: "2026-10-17T18:52:10.767Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b36-00a5-7883-ac24-203c32bbbc44
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: db54197bb2355ea2008f875d996998ce31d77d10a6aa3449feafb69830793318
              started: "2026-10-17T18:53:13.317Z"
              spawned: false
              bytes: 469184
              windows:
                - start: "2026-10-17T18:53:13.317Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b36-6196-7f23-9466-6916cd22f958
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 5958f9bee14eeebefd0b6a98a2b2c32aa5b188ead4432f18d85f2cba279b6a8b
              started: "2026-10-17T18:53:38.446Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-17T18:53:38.446Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b37-280c-7371-8b76-a1e062c8c9a6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: de87e5fd246d42d6ac3b0041244e3901571594c8496dcae5e8c8bdff79621f09
              started: "2026-10-17T18:54:28.588Z"
              spawned: true
              bytes: 827417
              windows:
                - start: "2026-10-17T18:54:28.588Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b37-4791-7cb0-a3fa-3ecffe85808a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 73df336d4d328baeaa1fd74fcff0583b19a0cbc5745752411c828d566fe1810a
              started: "2026-10-17T18:54:36.634Z"
              spawned: true
              bytes: 641027
              windows:
                - start: "2026-10-17T18:54:36.634Z"
                  end: "2026-10-18T04:14:52-07:00"
            - harness: codex
              id: 01a14b37-8631-7082-a9b4-1f3e6c0de9da
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fb68d456a155d8cb366c5d78a60face0260c8238388e88d248d53277baa2b8bd
              started: "2026-10-17T18:54:52.672Z"
              spawned: true
              bytes: 1202656
              windows:
                - start: "2026-10-17T18:54:52.672Z"
                  end: "2026-10-18T04:14:52-07:00"
        queue_waits:
            - attempt_id: c0796f4dfe80368cf357507b001784e5
              slot: pair:6
              record_digest: e178e012249f1f6794e067cb4fc8beefd2a68bdbb7d7c7a56608a0dbe8ac6f8d
              start: "2026-10-18T09:27:08.258262Z"
              end: "2026-10-18T09:45:20.724502Z"
              queue_seconds: 1092.46624
              raw_seconds: 1092.46624
              exact: true
            - attempt_id: 0619b0f0bcec5545314c1d0545bc00d8
              slot: pair:6
              record_digest: 62c2fa4b0821c4a9e343fe7db9bd598976418c31edb22e43d3d0713d569d7a28
              start: "2026-10-18T09:55:55.306272Z"
              end: "2026-10-18T10:20:09.097075Z"
              queue_seconds: 1453.790803
              raw_seconds: 1453.790803
              exact: true
            - attempt_id: e870e668582e734772bf8b6d58f866e5
              slot: pair:6
              record_digest: 224d6db5a0788b209a705294f8d21558de77f13e19d48b2cb8553143ca0bb0ef
              start: "2026-10-18T10:40:36.895941Z"
              end: "2026-10-18T11:00:06.922588Z"
              queue_seconds: 1170.026647
              raw_seconds: 1170.026647
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 187.45245
        cuts:
            - dispatched: "2026-10-18T03:32:05-07:00"
              finalized: "2026-10-18T03:34:57-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T04:17:37-07:00"
      dispatched: "2026-10-18T04:15:30-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Console snapshots wrapper activity before the combined pass; titles.go:167-181 merges it before frame, heat, and fading outputs. The regression passes at head and fails with that merge removed in a disposable checkout.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The 1,000-tick test now exercises two file-backed compact projections, transcript growth, and ten launch turnovers. It verifies heap, projection retirement, warm history work, and command counts. Injecting projection recreation makes it fail on tick 1.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 0160140172a8a79674f9d14d3c595906ccbfca7e
      blocked: false
---

# Gate ledger — pair#649 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T03:34:57-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `shared-activity-consumer-consistency` (kind: defect) Title decisions omit wrapper activity merged into idle fading
  cmd/internal/couchcmd/titles.go:168-172 computes frame eligibility and cmux heat from facts.Activity before cmd/internal/couchtty/console_activity.go:177-185 merges wrapper work. During silent active/background work, fading stays current while frame updates stop after two minutes and cmux heat can age incorrectly. The plan explicitly requires the final activity timestamp for both consumers. Inject admitted wrapper activity before title decisions and add a regression asserting both emitted titles and fading with stale persisted activity (ARCH-PURPOSE).
- **BR-2** [Important] `production-path-resource-verification` (kind: defect) The 1,000-tick heap regression never constructs a compact projection
  cmd/internal/couchcmd/titles_test.go:211-213 replaces production IO with fakeCouchTitles, whose source returns constant facts without constructing TitleProjection or reading native history. The heap assertion therefore cannot detect retained projection/cache growth, and this workload has no history-work counters. Exercise real compact projections over disposable file-backed fixtures for 1,000 ticks, including growth and generation turnover, while asserting retained heap, cache retirement, warm history work, and command counts (ARCH-CONSTRAINTS, ARCH-PURPOSE).

## Round 2 — 2026-10-18T04:17:37-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Console snapshots wrapper activity before the combined pass; titles.go:167-181 merges it before frame, heat, and fading outputs. The regression passes at head and fails with that merge removed in a disposable checkout.
- BR-2 — addressed — The 1,000-tick test now exercises two file-backed compact projections, transcript growth, and ten launch turnovers. It verifies heap, projection retirement, warm history work, and command counts. Injecting projection recreation makes it fail on tick 1.

## Open findings

(none — every finding has been disposed)
