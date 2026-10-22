---
gate: boundary-review
issue: 773
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d323f9de-6dc5-40d4-b8cf-4f890b2bf591
            repository: github.com/xianxu/pair
            issue_id: "000773"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/5a1b4c03b2d10c9d
            reviewed_head: e9c154b231493652ebe6e2c285ce0e57e624c625
            slot: pair:7
            requested_at: "2026-10-22T13:00:37.331191Z"
          phase: finished
          pid: 20120
          started_at: "2026-10-22T13:00:37.483234Z"
          finished_at: "2026-10-22T13:08:43.656937Z"
          session_id: 01a163b3-0070-7372-97d3-e816903d9f5c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: e9c154b231493652ebe6e2c285ce0e57e624c625
        tracker_head: 5c6b4ccca2c31c8883b5d8ad1d1168952d600881
        branch_point: 7702a3dddbfdb5c7d8a021bb8949df98f0a6b282
        since: "2026-10-21T08:12:54-07:00"
        until: "2026-10-22T05:59:35-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T08:12:54-07:00"
              end: "2026-10-22T05:59:35-07:00"
              began: "2026-10-21T08:12:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 3bf183daeec0665888989d0688012260fe617ad22e06285a4a163491ca9c0cae
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 243261639
              windows:
                - start: "2026-10-21T08:12:54-07:00"
                  end: "2026-10-22T05:59:35-07:00"
            - harness: codex
              id: 01a15fe5-900c-74c0-bc86-cf4dad2c47c5
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: ea925fc6ceeb90a5fd1e143f43968bda6bb72db43b7e625caee96c1fc626ff3e
              started: "2026-10-21T19:17:22.726Z"
              spawned: true
              bytes: 983254
              windows:
                - start: "2026-10-21T19:17:22.726Z"
                  end: "2026-10-22T05:59:35-07:00"
            - harness: codex
              id: 01a162f6-bc9f-7b53-9c68-a4231142b9fd
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e915fbaf81c3553e87ec5ac9202faa81a532d5030539dff1b1467ac17dc4469a
              started: "2026-10-22T09:34:59.805Z"
              spawned: true
              bytes: 1100519
              windows:
                - start: "2026-10-22T09:34:59.805Z"
                  end: "2026-10-22T05:59:35-07:00"
            - harness: codex
              id: 01a162fa-4c90-7a03-9f61-61218160d2ee
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b05b590e096ee1d80e5ebfbb0763a3c723f144402c5bd512cccbb92c7111387d
              started: "2026-10-22T09:38:53.317Z"
              spawned: true
              bytes: 951096
              windows:
                - start: "2026-10-22T09:38:53.317Z"
                  end: "2026-10-22T05:59:35-07:00"
            - harness: codex
              id: 01a16344-2324-7830-bb8e-b342bb933f3e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: d4d6f1e803fbdc1ba024857bcdd308ab16511f4900a375321797f5eb589a7683
              started: "2026-10-22T10:59:32.355Z"
              spawned: true
              bytes: 1002606
              windows:
                - start: "2026-10-22T10:59:32.355Z"
                  end: "2026-10-22T05:59:35-07:00"
        queue_waits:
            - attempt_id: 6c97fd46f24e0ceee10c4d23ac849831
              slot: pair:7
              record_digest: 775647f87110f6c709c123cb1cf47b278fdae943dda791ccde40d545c9aeeea8
              start: "2026-10-22T09:46:02.013481Z"
              end: "2026-10-22T10:04:21.232325Z"
              queue_seconds: 1099.218844
              raw_seconds: 1099.218844
              exact: true
            - attempt_id: e33828c37ba00eb2c2c83a139f4d0dce
              slot: pair:7
              record_digest: 8b3c4ee68d8b39166980e17510681da2e147d0560dc90e5cbbd54d2291d96efe
              start: "2026-10-22T11:01:59.299969Z"
              end: "2026-10-22T11:49:20.966653Z"
              queue_seconds: 2841.666684
              raw_seconds: 2841.666684
              exact: true
        protected:
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T08:12:54-07:00"
              end: "2026-10-21T15:56:12.692Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:13:36.627Z"
              end: "2026-10-21T15:55:44.958Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:21:25.399Z"
              end: "2026-10-21T15:41:03.686Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:43:27.716Z"
              end: "2026-10-21T15:44:45.319Z"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T15:56:12.692Z"
              end: "2026-10-21T09:15:28-07:00"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T09:15:28-07:00"
              end: "2026-10-21T16:36:57.546Z"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T16:40:00.572Z"
              end: "2026-10-21T09:40:16-07:00"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T10:03:53-07:00"
              end: "2026-10-21T18:10:37.283Z"
      "n": 1
      timestamp: "2026-10-22T06:08:45-07:00"
      dispatched: "2026-10-22T06:00:34-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Public warm attachment does not transport the caller-owned startup deadline
          detail: 'cmd/internal/couchcore/resume.go:452 routes ordinary listed-terminal resume through attachListedTerminalAdmitted, bypassing launchTrackedThread. warm_attach.go:92 supplies no PAIR_STARTUP_DEADLINE, and line 116 retains the relative ten-second helper timeout. Consequently the helper cannot reject delayed exec against the caller''s absolute deadline, and Pair selects a fresh fallback. ARCH-PURPOSE: complete the plan''s explicit warm-attach sweep by selecting and transporting one deadline while preserving the separate interactive lifetime. Add regression coverage through this public entry, including delayed helper execution and deadline preservation.'
          family: startup-budget-consumer-coverage
          round: 1
        - id: BR-2
          severity: Important
          title: Local console progress displays operation age instead of current-stage age
          detail: 'cmd/internal/couchcore/startup_progress.go:21 forwards only stage text to the console reporter. couchtty/menu_operations.go:311 updates that text, but lines 329–332 continue measuring from the operation''s StartedAt. After twenty seconds of preparation and five seconds of readiness, the console displays twenty-five seconds beside readiness; the remote CLI correctly displays stage age. ARCH-PURPOSE: retain a stage-start timestamp in the console''s pure progress transition, preserve it on repeated reports, and test stage changes with controlled time.'
          family: stage-progress-consumer-coverage
          round: 1
        - id: BR-3
          severity: Important
          title: The shared startup timer invalidates warm-adoption failure fixtures
          detail: 'cmd/internal/couchcore/startup_budget.go:18–28 turns resumeRegistrationTimeout into a real timer before preparation. warm_failure_test.go:274–279 still supplies a 200 ms sentinel and explicitly expects its registrationDeadline factory to avoid a timer. The broader package run failed TestAFailedTrackedWarmAdoptionRetireStillLeavesTheRecordRecoverable and two TestAFailedTrackedWarmAdoptionKeepsItsSession cases; one expired during spawn, and the retire case reached zero intended session reads. ARCH-ORDER: separate budget-propagation assertions from deterministic failure injection so these tests reach and verify their named transitions without depending on preparation speed.'
          family: deterministic-failure-injection
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a0ca37da-1daf-4bb6-af4d-a55bc7629653
            repository: github.com/xianxu/pair
            issue_id: "000773"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/beb50a2909b4ab64
            reviewed_head: f3c423c6462058da7433262a1617a9fb42ab3386
            slot: pair:7
            requested_at: "2026-10-22T14:15:15.614014Z"
          phase: finished
          pid: 55129
          started_at: "2026-10-22T14:15:15.705453Z"
          finished_at: "2026-10-22T14:19:04.594664Z"
          session_id: 01a163f7-5507-71f0-a0e8-910505da279a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: f3c423c6462058da7433262a1617a9fb42ab3386
        tracker_head: 9f9f7dc7723e153bf49147dcfd35f478c0dece5b
        branch_point: 7702a3dddbfdb5c7d8a021bb8949df98f0a6b282
        since: "2026-10-21T08:12:54-07:00"
        until: "2026-10-22T07:14:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T08:12:54-07:00"
              end: "2026-10-22T07:14:10-07:00"
              began: "2026-10-21T08:12:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 3ba133953755a6e97ff7265f8c984977de323b6eda593b3b1fca1ba495e5c920
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 244111860
              windows:
                - start: "2026-10-21T08:12:54-07:00"
                  end: "2026-10-22T07:14:10-07:00"
            - harness: codex
              id: 01a15fe5-900c-74c0-bc86-cf4dad2c47c5
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: ea925fc6ceeb90a5fd1e143f43968bda6bb72db43b7e625caee96c1fc626ff3e
              started: "2026-10-21T19:17:22.726Z"
              spawned: true
              bytes: 983254
              windows:
                - start: "2026-10-21T19:17:22.726Z"
                  end: "2026-10-22T07:14:10-07:00"
            - harness: codex
              id: 01a162f6-bc9f-7b53-9c68-a4231142b9fd
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e915fbaf81c3553e87ec5ac9202faa81a532d5030539dff1b1467ac17dc4469a
              started: "2026-10-22T09:34:59.805Z"
              spawned: true
              bytes: 1100519
              windows:
                - start: "2026-10-22T09:34:59.805Z"
                  end: "2026-10-22T07:14:10-07:00"
            - harness: codex
              id: 01a162fa-4c90-7a03-9f61-61218160d2ee
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b05b590e096ee1d80e5ebfbb0763a3c723f144402c5bd512cccbb92c7111387d
              started: "2026-10-22T09:38:53.317Z"
              spawned: true
              bytes: 951096
              windows:
                - start: "2026-10-22T09:38:53.317Z"
                  end: "2026-10-22T07:14:10-07:00"
            - harness: codex
              id: 01a16344-2324-7830-bb8e-b342bb933f3e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: d4d6f1e803fbdc1ba024857bcdd308ab16511f4900a375321797f5eb589a7683
              started: "2026-10-22T10:59:32.355Z"
              spawned: true
              bytes: 1002606
              windows:
                - start: "2026-10-22T10:59:32.355Z"
                  end: "2026-10-22T07:14:10-07:00"
            - harness: codex
              id: 01a163e9-31fa-7d21-9ee5-6496d5d5ab4d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 18dace9718b3b068e3be9e00bdb3c9a3d9eda8a0fa3855a098dc20894669b649
              started: "2026-10-22T13:59:49.587Z"
              spawned: true
              bytes: 1107110
              windows:
                - start: "2026-10-22T13:59:49.587Z"
                  end: "2026-10-22T07:14:10-07:00"
            - harness: codex
              id: 01a163e9-5c57-78d0-9465-b5edb01666cb
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b0820064392365bc19b762e43e729ca0254e421681fb971d8f683481910e5a41
              started: "2026-10-22T14:00:00.436Z"
              spawned: true
              bytes: 1160072
              windows:
                - start: "2026-10-22T14:00:00.436Z"
                  end: "2026-10-22T07:14:10-07:00"
        queue_waits:
            - attempt_id: 6c97fd46f24e0ceee10c4d23ac849831
              slot: pair:7
              record_digest: 775647f87110f6c709c123cb1cf47b278fdae943dda791ccde40d545c9aeeea8
              start: "2026-10-22T09:46:02.013481Z"
              end: "2026-10-22T10:04:21.232325Z"
              queue_seconds: 1099.218844
              raw_seconds: 1099.218844
              exact: true
            - attempt_id: e33828c37ba00eb2c2c83a139f4d0dce
              slot: pair:7
              record_digest: 8b3c4ee68d8b39166980e17510681da2e147d0560dc90e5cbbd54d2291d96efe
              start: "2026-10-22T11:01:59.299969Z"
              end: "2026-10-22T11:49:20.966653Z"
              queue_seconds: 2841.666684
              raw_seconds: 2841.666684
              exact: true
        protected:
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T08:12:54-07:00"
              end: "2026-10-21T15:56:12.692Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:13:36.627Z"
              end: "2026-10-21T15:55:44.958Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:21:25.399Z"
              end: "2026-10-21T15:41:03.686Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:43:27.716Z"
              end: "2026-10-21T15:44:45.319Z"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T15:56:12.692Z"
              end: "2026-10-21T09:15:28-07:00"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T09:15:28-07:00"
              end: "2026-10-21T16:36:57.546Z"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T16:40:00.572Z"
              end: "2026-10-21T09:40:16-07:00"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T10:03:53-07:00"
              end: "2026-10-21T18:10:37.283Z"
        cuts:
            - dispatched: "2026-10-22T06:00:34-07:00"
              finalized: "2026-10-22T06:08:45-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-22T07:19:05-07:00"
      dispatched: "2026-10-22T07:15:13-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: warm_attach.go selects and transports one absolute deadline while preserving interactive lifetime. TestPublicWarmAttachmentTransportsCallerStartupBudget passes and fails against the previous implementation, detecting missing transport, the ten-second helper timeout, and registration renewal. Helper tests separately cover delayed acknowledgement and pre-exec expiry.
          round: 2
        - id: BR-2
          disposition: addressed
          note: menu_operations.go retains stage-start time, preserves it on repeated reports, and uses it for row and notice rendering. TestMenuOperationStageAgeProjectsRowAndNotice passes; removing the timestamp correction reproduces the incorrect 25s readiness display.
          round: 2
        - id: BR-3
          disposition: addressed
          note: General warm-failure fixtures no longer install the preparation-sensitive 200ms timer. Dedicated propagation controls use virtual time. TestTrackedWarmFailureInjectionSurvivesPreparationDelay passes; restoring the sentinel fails before the intended registration transition.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: f3c423c6462058da7433262a1617a9fb42ab3386
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9fc72d40-07bb-4a3e-b753-98f38b5408f7
            repository: github.com/xianxu/pair
            issue_id: "000773"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/cf5322dbb450c37a
            reviewed_head: be30575e0c8d797c4babc08a478f00cdaebbf089
            slot: pair:7
            requested_at: "2026-10-22T20:30:01.216043Z"
          phase: finished
          pid: 58929
          started_at: "2026-10-22T20:30:01.524382Z"
          finished_at: "2026-10-22T20:31:14.602127Z"
          session_id: 01a1654e-71d3-75e3-8dc9-1065d4b82cdc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: be30575e0c8d797c4babc08a478f00cdaebbf089
        tracker_head: de9cce90ffd0fcd9d0d87b3c29f33e5c79c8513c
        branch_point: 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc
        since: "2026-10-21T08:12:54-07:00"
        until: "2026-10-22T12:59:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T08:12:54-07:00"
              end: "2026-10-22T12:59:24-07:00"
              began: "2026-10-21T08:12:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 34035ac73818267eb5d92e8f1879d7a747ecd5f4664bd12d116b4aa1c0b05d16
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 246995838
              windows:
                - start: "2026-10-21T08:12:54-07:00"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a15fe5-900c-74c0-bc86-cf4dad2c47c5
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: ea925fc6ceeb90a5fd1e143f43968bda6bb72db43b7e625caee96c1fc626ff3e
              started: "2026-10-21T19:17:22.726Z"
              spawned: true
              bytes: 983254
              windows:
                - start: "2026-10-21T19:17:22.726Z"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a162f6-bc9f-7b53-9c68-a4231142b9fd
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e915fbaf81c3553e87ec5ac9202faa81a532d5030539dff1b1467ac17dc4469a
              started: "2026-10-22T09:34:59.805Z"
              spawned: true
              bytes: 1100519
              windows:
                - start: "2026-10-22T09:34:59.805Z"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a162fa-4c90-7a03-9f61-61218160d2ee
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b05b590e096ee1d80e5ebfbb0763a3c723f144402c5bd512cccbb92c7111387d
              started: "2026-10-22T09:38:53.317Z"
              spawned: true
              bytes: 951096
              windows:
                - start: "2026-10-22T09:38:53.317Z"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a16344-2324-7830-bb8e-b342bb933f3e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: d4d6f1e803fbdc1ba024857bcdd308ab16511f4900a375321797f5eb589a7683
              started: "2026-10-22T10:59:32.355Z"
              spawned: true
              bytes: 1002606
              windows:
                - start: "2026-10-22T10:59:32.355Z"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a163e9-31fa-7d21-9ee5-6496d5d5ab4d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 18dace9718b3b068e3be9e00bdb3c9a3d9eda8a0fa3855a098dc20894669b649
              started: "2026-10-22T13:59:49.587Z"
              spawned: true
              bytes: 1107110
              windows:
                - start: "2026-10-22T13:59:49.587Z"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a163e9-5c57-78d0-9465-b5edb01666cb
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b0820064392365bc19b762e43e729ca0254e421681fb971d8f683481910e5a41
              started: "2026-10-22T14:00:00.436Z"
              spawned: true
              bytes: 1160072
              windows:
                - start: "2026-10-22T14:00:00.436Z"
                  end: "2026-10-22T12:59:24-07:00"
            - harness: codex
              id: 01a164fe-5fe7-7161-8046-98e6d9175d18
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b834909c55928df4c3c469a6d485b5b3a262145229740c241b0602e306442176
              started: "2026-10-22T19:02:34.901Z"
              spawned: true
              bytes: 1197993
              windows:
                - start: "2026-10-22T19:02:34.901Z"
                  end: "2026-10-22T12:59:24-07:00"
        queue_waits:
            - attempt_id: 6c97fd46f24e0ceee10c4d23ac849831
              slot: pair:7
              record_digest: 775647f87110f6c709c123cb1cf47b278fdae943dda791ccde40d545c9aeeea8
              start: "2026-10-22T09:46:02.013481Z"
              end: "2026-10-22T10:04:21.232325Z"
              queue_seconds: 1099.218844
              raw_seconds: 1099.218844
              exact: true
            - attempt_id: e33828c37ba00eb2c2c83a139f4d0dce
              slot: pair:7
              record_digest: 8b3c4ee68d8b39166980e17510681da2e147d0560dc90e5cbbd54d2291d96efe
              start: "2026-10-22T11:01:59.299969Z"
              end: "2026-10-22T11:49:20.966653Z"
              queue_seconds: 2841.666684
              raw_seconds: 2841.666684
              exact: true
            - attempt_id: 4c8e890e44b09f6a184c006d4f3b5c55
              slot: pair:7
              record_digest: 3d551c171f0f55ebc96d24f79b609b83161d8f82206f7d673debbf97f5fd3666
              start: "2026-10-22T14:14:30.451425Z"
              end: "2026-10-22T15:36:16.293287Z"
              queue_seconds: 4905.841862
              raw_seconds: 4905.841862
              exact: true
            - attempt_id: 1d8fae6745226f7d80414d02eccb7c87
              slot: pair:7
              record_digest: 03bcbd2dae31b9932425b35d04a6650475087efb48f723d2a0d890d9f1870f7f
              start: "2026-10-22T16:35:39.432263Z"
              end: "2026-10-22T18:49:44.294935Z"
              queue_seconds: 8044.862672
              raw_seconds: 8044.862672
              exact: true
            - attempt_id: 4b42e6ed7c8af4526a196f8a61977aa8
              slot: pair:7
              record_digest: 8479efce3d63573009781ce34caef6df23d8ec9b19b626dfb0519c0a8b9190a1
              start: "2026-10-22T19:01:20.839401Z"
              end: "2026-10-22T19:13:56.328565Z"
              queue_seconds: 755.489164
              raw_seconds: 755.489164
              exact: true
        protected:
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T08:12:54-07:00"
              end: "2026-10-21T15:56:12.692Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:13:36.627Z"
              end: "2026-10-21T15:55:44.958Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:21:25.399Z"
              end: "2026-10-21T15:41:03.686Z"
            - issue: "736"
              slot: pair:7
              start: "2026-10-21T15:43:27.716Z"
              end: "2026-10-21T15:44:45.319Z"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T15:56:12.692Z"
              end: "2026-10-21T09:15:28-07:00"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T09:15:28-07:00"
              end: "2026-10-21T16:36:57.546Z"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T16:40:00.572Z"
              end: "2026-10-21T09:40:16-07:00"
            - issue: "774"
              slot: pair:7
              start: "2026-10-21T10:03:53-07:00"
              end: "2026-10-21T18:10:37.283Z"
        cuts:
            - dispatched: "2026-10-22T06:00:34-07:00"
              finalized: "2026-10-22T06:08:45-07:00"
              finalizing: false
            - dispatched: "2026-10-22T07:15:13-07:00"
              finalized: "2026-10-22T07:19:05-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-22T13:31:20-07:00"
      dispatched: "2026-10-22T13:29:58-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Previously disposed correction is unchanged; TestPublicWarmAttachmentTransportsCallerStartupBudget passes at the pinned head.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Previously disposed correction is unchanged; TestMenuOperationStageAgeProjectsRowAndNotice passes at the pinned head.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Previously disposed correction is unchanged; TestTrackedWarmFailureInjectionSurvivesPreparationDelay passes at the pinned head.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: be30575e0c8d797c4babc08a478f00cdaebbf089
      blocked: false
---

# Gate ledger — pair#773 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T06:08:45-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `startup-budget-consumer-coverage` Public warm attachment does not transport the caller-owned startup deadline
  cmd/internal/couchcore/resume.go:452 routes ordinary listed-terminal resume through attachListedTerminalAdmitted, bypassing launchTrackedThread. warm_attach.go:92 supplies no PAIR_STARTUP_DEADLINE, and line 116 retains the relative ten-second helper timeout. Consequently the helper cannot reject delayed exec against the caller's absolute deadline, and Pair selects a fresh fallback. ARCH-PURPOSE: complete the plan's explicit warm-attach sweep by selecting and transporting one deadline while preserving the separate interactive lifetime. Add regression coverage through this public entry, including delayed helper execution and deadline preservation.
- **BR-2** [Important] `stage-progress-consumer-coverage` Local console progress displays operation age instead of current-stage age
  cmd/internal/couchcore/startup_progress.go:21 forwards only stage text to the console reporter. couchtty/menu_operations.go:311 updates that text, but lines 329–332 continue measuring from the operation's StartedAt. After twenty seconds of preparation and five seconds of readiness, the console displays twenty-five seconds beside readiness; the remote CLI correctly displays stage age. ARCH-PURPOSE: retain a stage-start timestamp in the console's pure progress transition, preserve it on repeated reports, and test stage changes with controlled time.
- **BR-3** [Important] `deterministic-failure-injection` The shared startup timer invalidates warm-adoption failure fixtures
  cmd/internal/couchcore/startup_budget.go:18–28 turns resumeRegistrationTimeout into a real timer before preparation. warm_failure_test.go:274–279 still supplies a 200 ms sentinel and explicitly expects its registrationDeadline factory to avoid a timer. The broader package run failed TestAFailedTrackedWarmAdoptionRetireStillLeavesTheRecordRecoverable and two TestAFailedTrackedWarmAdoptionKeepsItsSession cases; one expired during spawn, and the retire case reached zero intended session reads. ARCH-ORDER: separate budget-propagation assertions from deterministic failure injection so these tests reach and verify their named transitions without depending on preparation speed.

## Round 2 — 2026-10-22T07:19:05-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — warm_attach.go selects and transports one absolute deadline while preserving interactive lifetime. TestPublicWarmAttachmentTransportsCallerStartupBudget passes and fails against the previous implementation, detecting missing transport, the ten-second helper timeout, and registration renewal. Helper tests separately cover delayed acknowledgement and pre-exec expiry.
- BR-2 — addressed — menu_operations.go retains stage-start time, preserves it on repeated reports, and uses it for row and notice rendering. TestMenuOperationStageAgeProjectsRowAndNotice passes; removing the timestamp correction reproduces the incorrect 25s readiness display.
- BR-3 — addressed — General warm-failure fixtures no longer install the preparation-sensitive 200ms timer. Dedicated propagation controls use virtual time. TestTrackedWarmFailureInjectionSurvivesPreparationDelay passes; restoring the sentinel fails before the intended registration transition.

## Round 3 — 2026-10-22T13:31:20-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Previously disposed correction is unchanged; TestPublicWarmAttachmentTransportsCallerStartupBudget passes at the pinned head.
- BR-2 — addressed — Previously disposed correction is unchanged; TestMenuOperationStageAgeProjectsRowAndNotice passes at the pinned head.
- BR-3 — addressed — Previously disposed correction is unchanged; TestTrackedWarmFailureInjectionSurvivesPreparationDelay passes at the pinned head.

## Open findings

(none — every finding has been disposed)
