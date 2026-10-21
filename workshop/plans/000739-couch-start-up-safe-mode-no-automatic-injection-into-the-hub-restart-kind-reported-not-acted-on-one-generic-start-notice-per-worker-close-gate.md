---
gate: boundary-review
issue: 739
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: eeff0018-1f9c-4afc-8e8d-26785ef0f9be
            repository: github.com/xianxu/pair
            issue_id: "000739"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/6849008991c1da7f
            reviewed_head: 588e2ce61585d7106f3a1d168982b9b84bbfdcfc
            slot: pair:7
            requested_at: "2026-10-21T08:28:24.707489Z"
          phase: finished
          pid: 6743
          started_at: "2026-10-21T08:28:24.801199Z"
          finished_at: "2026-10-21T08:34:01.54105Z"
          session_id: 01a15d93-6c5b-7432-8021-b8d0aea3967d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 588e2ce61585d7106f3a1d168982b9b84bbfdcfc
        tracker_head: 0488795041f347ec4e234200fd5e593513279de8
        branch_point: 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92
        since: "2026-10-20T22:40:29-07:00"
        until: "2026-10-21T01:09:07-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:29-07:00"
              end: "2026-10-21T01:09:07-07:00"
              began: "2026-10-20T22:40:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: b180a6eb165fa47320f180ffc1f4dfaeeef71bbff494b47c1f8b1005f445e770
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 221251034
              windows:
                - start: "2026-10-20T22:40:29-07:00"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15cfa-2b10-70a2-8cef-bdef57736088
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: aad4d416bb9801a059bde76736e6caa39537129731fe77038d1df7d4fb166156
              started: "2026-10-21T05:41:01.468Z"
              spawned: true
              bytes: 729423
              windows:
                - start: "2026-10-21T05:41:01.468Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d34-a166-7851-b4c1-4c61abe2dcac
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 29b4e0e3e437cde1a330a6b6695724cad182e668f8211f95ea03b0e81fd7b362
              started: "2026-10-21T06:44:52.963Z"
              spawned: false
              bytes: 637799
              windows:
                - start: "2026-10-21T06:44:52.963Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d3c-0f44-7f93-b3e3-4b2b6a1d992e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6ff2b2176b79eff7f4553e31b010933c449f545fa9ee3e0c8b2a631e5a6f82c8
              started: "2026-10-21T06:52:59.861Z"
              spawned: false
              bytes: 419585
              windows:
                - start: "2026-10-21T06:52:59.861Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d3c-4459-7cd3-a8c4-9282f9dc3bfc
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 44858118e83cd8f3eddec03d7b921ad09feea4b4e5dfbf82cfe66aacbbc826fb
              started: "2026-10-21T06:53:13.275Z"
              spawned: true
              bytes: 2635708
              windows:
                - start: "2026-10-21T06:53:13.275Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d3d-f23b-79b2-853b-10996f109622
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 3a33af03cea0c64703314d15011420122fc14c789b5ddcb6ec755508672c78d3
              started: "2026-10-21T06:55:03.541Z"
              spawned: false
              bytes: 442984
              windows:
                - start: "2026-10-21T06:55:03.541Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d3e-836f-7f71-99e6-40c892831e25
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d74160388ad775821602929ee3e41c4455d6b9b9f948803200e53960e12534e7
              started: "2026-10-21T06:55:40.72Z"
              spawned: false
              bytes: 427601
              windows:
                - start: "2026-10-21T06:55:40.72Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d3f-e9b6-7682-8430-674cd5964b27
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e163c778643b9a28bd084a0b6e0fa29e5db25964097a061c55765da25c954af9
              started: "2026-10-21T06:57:12.214Z"
              spawned: true
              bytes: 2347004
              windows:
                - start: "2026-10-21T06:57:12.214Z"
                  end: "2026-10-21T01:09:07-07:00"
            - harness: codex
              id: 01a15d42-b9ca-78c3-b0ab-b4d02683f0e3
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 8e70a57d48a768f3a588b0f912510c761321843eb67a959ce2f8649c2b7ae686
              started: "2026-10-21T07:00:16.666Z"
              spawned: true
              bytes: 2365051
              windows:
                - start: "2026-10-21T07:00:16.666Z"
                  end: "2026-10-21T01:09:07-07:00"
      "n": 1
      timestamp: "2026-10-21T01:34:02-07:00"
      dispatched: "2026-10-21T01:28:23-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Core concepts table contradicts the delivered implementation
          detail: workshop/plans/000739-startup-safe-mode-plan.md:55 lists startup lifecycle provenance as a modified PURE entity in couchtty/console_lifecycle.go, but that file has no changes in the pinned range; the new suppression implementation is in couchcmd/lifecycle_notices.go. Line 67 also places ObserveStartupContext in startup.go although it is defined in startup_policy.go. Correct the locations, classifications and statuses, and add a Revisions entry. This is the Core concepts contradiction mandated as Critical by this review contract (ARCH-PURE).
          family: core-concept-traceability
          round: 1
        - id: BR-2
          severity: Important
          title: Existing startup integration test still requires automatic agent launch
          detail: cmd/internal/couchcmd/startup_observation_test.go:28 requires a non-nil startup handle and runner operations for an empty fixture. The new StartInteractive behavior correctly returns neither, so TestInteractiveHubStartupOpensConsoleWithSelectedChild fails both in the package run and in isolation. Update the test to assert a usable empty switcher without launching, and retain a separately seeded surviving-session attachment case (ARCH-PURPOSE).
          family: behavioral-test-contract-alignment
          round: 1
        - id: BR-3
          severity: Important
          title: Ordinary couchcore and couchtty suites exceed enforced write budgets
          detail: The five-package go test invocation failed couchcore with 154050560 bytes against its 134217728-byte cap and couchtty with 87949312 bytes against its 67108864-byte cap. The guards are applied at cmd/internal/couchcore/launchhelper_test.go:50 and cmd/internal/couchtty/reattach_testmain_test.go:21. Reduce fixture writes and verify the ordinary suites within their existing envelopes; direct in-memory fixtures for pure startup policy tests are one concrete opportunity (ARCH-CONSTRAINTS, ARCH-PURE).
          family: test-operating-envelope
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#739 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T01:34:02-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `core-concept-traceability` Core concepts table contradicts the delivered implementation
  workshop/plans/000739-startup-safe-mode-plan.md:55 lists startup lifecycle provenance as a modified PURE entity in couchtty/console_lifecycle.go, but that file has no changes in the pinned range; the new suppression implementation is in couchcmd/lifecycle_notices.go. Line 67 also places ObserveStartupContext in startup.go although it is defined in startup_policy.go. Correct the locations, classifications and statuses, and add a Revisions entry. This is the Core concepts contradiction mandated as Critical by this review contract (ARCH-PURE).
- **BR-2** [Important] `behavioral-test-contract-alignment` Existing startup integration test still requires automatic agent launch
  cmd/internal/couchcmd/startup_observation_test.go:28 requires a non-nil startup handle and runner operations for an empty fixture. The new StartInteractive behavior correctly returns neither, so TestInteractiveHubStartupOpensConsoleWithSelectedChild fails both in the package run and in isolation. Update the test to assert a usable empty switcher without launching, and retain a separately seeded surviving-session attachment case (ARCH-PURPOSE).
- **BR-3** [Important] `test-operating-envelope` Ordinary couchcore and couchtty suites exceed enforced write budgets
  The five-package go test invocation failed couchcore with 154050560 bytes against its 134217728-byte cap and couchtty with 87949312 bytes against its 67108864-byte cap. The guards are applied at cmd/internal/couchcore/launchhelper_test.go:50 and cmd/internal/couchtty/reattach_testmain_test.go:21. Reduce fixture writes and verify the ordinary suites within their existing envelopes; direct in-memory fixtures for pure startup policy tests are one concrete opportunity (ARCH-CONSTRAINTS, ARCH-PURE).

## Open findings

- **BR-1** [Critical] `core-concept-traceability` Core concepts table contradicts the delivered implementation
- **BR-2** [Important] `behavioral-test-contract-alignment` Existing startup integration test still requires automatic agent launch
- **BR-3** [Important] `test-operating-envelope` Ordinary couchcore and couchtty suites exceed enforced write budgets
