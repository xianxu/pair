---
gate: boundary-review
issue: 683
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e70bee73-e1f3-4b8d-9f61-4ca79ec49278
            repository: github.com/xianxu/pair
            issue_id: "000683"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/fd7443b0a18d13be
            reviewed_head: 3b9370348abba854cd5be0fa7d5c676be7ef3db7
            slot: pair:5
            requested_at: "2026-10-19T10:16:09.387885Z"
          phase: finished
          pid: 27646
          started_at: "2026-10-19T10:16:09.501685Z"
          finished_at: "2026-10-19T10:24:36.922835Z"
          session_id: 01a153a9-5951-7b93-9aa0-36212748b2cf
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 3b9370348abba854cd5be0fa7d5c676be7ef3db7
        tracker_head: 475e97a7241987d9e2710d69b944651d12202f7a
        branch_point: 5afc269d3add99de21b5f894d8436ac6da15c796
        since: "2026-10-19T02:08:51-07:00"
        until: "2026-10-19T03:14:50-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T02:08:51-07:00"
              end: "2026-10-19T03:14:50-07:00"
              began: "2026-10-19T02:08:51-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b4335941471c4ce8f13909e69dfe771f41af5c6090562e8029957c182f3b4115
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 71356782
              windows:
                - start: "2026-10-19T02:08:51-07:00"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a1536c-8666-7f61-9073-bfc2748bd76c
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: f8f512d3258733e3a00eb342ade6e91f08cf74f98af914a09d3910147ebaef5b
              started: "2026-10-19T09:09:43.699Z"
              spawned: true
              bytes: 682784
              windows:
                - start: "2026-10-19T09:09:43.699Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a1536f-bf0b-7423-a0de-b3da4eb30b17
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: e3517902d68b463af127a1b2c51c0bab1afc2c4b180caa1a32de19f9086f1c3a
              started: "2026-10-19T09:13:14.856Z"
              spawned: true
              bytes: 847298
              windows:
                - start: "2026-10-19T09:13:14.856Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a15376-0707-71a2-8e30-2442c7675920
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 3f6041598f933f72df5eb5e64ab12e9c97fbf81b0c4ca2d82e7fd81fb8eefa64
              started: "2026-10-19T09:20:07.027Z"
              spawned: false
              bytes: 780139
              windows:
                - start: "2026-10-19T09:20:07.027Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a15378-f8ff-76b0-b047-1672c4644144
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 47b7ed65dcbdd7faa3aa3c6badf9769224d3164aa57075485ecc13c11e5df3f1
              started: "2026-10-19T09:23:20.256Z"
              spawned: false
              bytes: 497628
              windows:
                - start: "2026-10-19T09:23:20.256Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a1537a-8f1a-7922-b8fd-1e791b716a50
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 8eb21d32c5068e21b40c2538e6adba6b4ccd2c7fc825936211962a62201ff7fc
              started: "2026-10-19T09:25:05.764Z"
              spawned: false
              bytes: 538640
              windows:
                - start: "2026-10-19T09:25:05.764Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a1537b-1ce7-7f20-b48a-3779ee1b70c5
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 1c46b36aa0404a571b47df74a2da496240d9714d62135567a43bfcce528ac884
              started: "2026-10-19T09:25:40.154Z"
              spawned: false
              bytes: 437676
              windows:
                - start: "2026-10-19T09:25:40.154Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a1537c-517e-7d83-95be-9f3c38120ef5
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 573a85adccb98297dd999831692f5f33243d76f70ba5da3dab556c33f2fec0a8
              started: "2026-10-19T09:26:58.731Z"
              spawned: true
              bytes: 1604682
              windows:
                - start: "2026-10-19T09:26:58.731Z"
                  end: "2026-10-19T03:14:50-07:00"
            - harness: codex
              id: 01a15381-d64b-75a3-b529-38a0e5aa956c
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 3a8eeee9919f646db38aae3f87c3f35537bb9fee9315a77fba15a84263c43bc5
              started: "2026-10-19T09:33:00.479Z"
              spawned: true
              bytes: 978194
              windows:
                - start: "2026-10-19T09:33:00.479Z"
                  end: "2026-10-19T03:14:50-07:00"
      "n": 1
      timestamp: "2026-10-19T03:24:38-07:00"
      dispatched: "2026-10-19T03:16:07-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Diagnostic target binding prevents recovery of unreadable records
          detail: cmd/internal/couchcore/lifecyclefailure_record.go:44 requires GetThread to decode successfully before reboot can execute. Reboot intentionally supports archiving unreadable records; TestAnUnreadableRecordCanBeRebootedThroughTheRuntime and TestARefusalsNamedCommandsActuallyWork both reproducibly fail with "exact thread binding unavailable ... invalid JSON value". Preserve that recovery path using verified exact-address evidence without requiring a decoded slot path, while retaining guards against replacement targets. ARCH-PURPOSE.
          family: diagnostics-preserve-recovery-contracts
          round: 1
        - id: BR-2
          severity: Critical
          title: Admitted queue operations can silently run without terminal recording
          detail: 'cmd/internal/couchcmd/lifecyclefailure_queue.go:35-43 leaves attempt nil when cached target lookup fails, but still enqueues the operation. Preparation failures then remain transient; successful dispatch can finish its own attempt before adoption/readiness, whose later failures are unrecorded. An enumerable sibling is hub_move.go:95-105: its "hub-move" key is rejected by the parser, and its payload-only dispatcher call supplies no fallback target. Pass verified diagnostic targets independently of scheduling keys and establish terminal ownership across every admitted producer, with missing-cache and post-dispatch failure regressions. ARCH-PURPOSE, ARCH-ORDER.'
          family: terminal-owner-coverage
          round: 1
        - id: BR-3
          severity: Important
          title: Standalone switch failures allocate and finish multiple attempts
          detail: cmd/internal/couchcore/launchfailure_capture.go:155-164 calls captureLaunchFailure when no launchFailureError exists, which already creates and finishes a standalone attempt at lines 139-145, then calls finishLaunchFailureCapture again. Existing standalone launch failures likewise publish before final switch enrichment and publish again afterward. Keep one enclosing attempt through capture and enrichment, then finish once; strengthen TestLaunchFailureCaptureSwitchFinalError to assert exactly one publication. ARCH-ORDER.
          family: single-terminal-publication
          round: 1
        - id: BR-4
          severity: Important
          title: The artifact source guard fails on three newly introduced symbols
          detail: go test ./cmd/internal/artifactpath -count=1 fails TestNoProductionSymbolIsReferencedOnlyByTests for NewLifecycleFailureStore and NewFakeLifecycleFailureStore at cmd/internal/couchcore/lifecyclefailure.go:163-166, and ActiveSlots at operationattempt_record.go:103. Remove unnecessary exports, use the intended production surface, or document legitimate test-observation exceptions in the existing allowlist, then rerun the guard.
          family: required-source-guard-compliance
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#683 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T03:24:38-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `diagnostics-preserve-recovery-contracts` Diagnostic target binding prevents recovery of unreadable records
  cmd/internal/couchcore/lifecyclefailure_record.go:44 requires GetThread to decode successfully before reboot can execute. Reboot intentionally supports archiving unreadable records; TestAnUnreadableRecordCanBeRebootedThroughTheRuntime and TestARefusalsNamedCommandsActuallyWork both reproducibly fail with "exact thread binding unavailable ... invalid JSON value". Preserve that recovery path using verified exact-address evidence without requiring a decoded slot path, while retaining guards against replacement targets. ARCH-PURPOSE.
- **BR-2** [Critical] `terminal-owner-coverage` Admitted queue operations can silently run without terminal recording
  cmd/internal/couchcmd/lifecyclefailure_queue.go:35-43 leaves attempt nil when cached target lookup fails, but still enqueues the operation. Preparation failures then remain transient; successful dispatch can finish its own attempt before adoption/readiness, whose later failures are unrecorded. An enumerable sibling is hub_move.go:95-105: its "hub-move" key is rejected by the parser, and its payload-only dispatcher call supplies no fallback target. Pass verified diagnostic targets independently of scheduling keys and establish terminal ownership across every admitted producer, with missing-cache and post-dispatch failure regressions. ARCH-PURPOSE, ARCH-ORDER.
- **BR-3** [Important] `single-terminal-publication` Standalone switch failures allocate and finish multiple attempts
  cmd/internal/couchcore/launchfailure_capture.go:155-164 calls captureLaunchFailure when no launchFailureError exists, which already creates and finishes a standalone attempt at lines 139-145, then calls finishLaunchFailureCapture again. Existing standalone launch failures likewise publish before final switch enrichment and publish again afterward. Keep one enclosing attempt through capture and enrichment, then finish once; strengthen TestLaunchFailureCaptureSwitchFinalError to assert exactly one publication. ARCH-ORDER.
- **BR-4** [Important] `required-source-guard-compliance` The artifact source guard fails on three newly introduced symbols
  go test ./cmd/internal/artifactpath -count=1 fails TestNoProductionSymbolIsReferencedOnlyByTests for NewLifecycleFailureStore and NewFakeLifecycleFailureStore at cmd/internal/couchcore/lifecyclefailure.go:163-166, and ActiveSlots at operationattempt_record.go:103. Remove unnecessary exports, use the intended production surface, or document legitimate test-observation exceptions in the existing allowlist, then rerun the guard.

## Open findings

- **BR-1** [Critical] `diagnostics-preserve-recovery-contracts` Diagnostic target binding prevents recovery of unreadable records
- **BR-2** [Critical] `terminal-owner-coverage` Admitted queue operations can silently run without terminal recording
- **BR-3** [Important] `single-terminal-publication` Standalone switch failures allocate and finish multiple attempts
- **BR-4** [Important] `required-source-guard-compliance` The artifact source guard fails on three newly introduced symbols
