---
gate: boundary-review
issue: 812
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d681bbc3-7ae8-4bed-b01d-ce214d481b3c
            repository: github.com/xianxu/pair
            issue_id: "000812"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/253c3ca3b6440386
            reviewed_head: 3c335cfdb96d94ceea14b5a8aff42c2ae5801925
            slot: pair:1
            requested_at: "2026-10-23T12:34:36.109541Z"
          phase: finished
          pid: 83066
          started_at: "2026-10-23T12:34:36.201473Z"
          finished_at: "2026-10-23T12:37:49.310047Z"
          session_id: 01a168c1-893b-7130-80f3-369223c706b3
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 3c335cfdb96d94ceea14b5a8aff42c2ae5801925
        tracker_head: 51698e2f7f3866a4baad5443693d03b62a3d5c1e
        branch_point: c0dbb32decc052500172ded01ff36e3fa8e49cc9
        since: "2026-10-23T05:01:56-07:00"
        until: "2026-10-23T05:34:03-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T05:01:56-07:00"
              end: "2026-10-23T05:34:03-07:00"
              began: "2026-10-23T05:01:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 7379508128a1c54ae7c232c3b5cd16f38d5c2a36275ac3f40f4b4d36d3c84f12
              started: "2026-10-21T18:31:20.491Z"
              spawned: false
              bytes: 24849433
              windows:
                - start: "2026-10-23T05:01:56-07:00"
                  end: "2026-10-23T05:34:03-07:00"
            - harness: codex
              id: 01a168a8-b55a-7a73-8097-1ac736feaf5d
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 06c8f98592b07caa7e74f16b9ceacb0a17b13596b7a74f6364075a3d9793593b
              started: "2026-10-23T12:07:29.699Z"
              spawned: false
              bytes: 561161
              windows:
                - start: "2026-10-23T12:07:29.699Z"
                  end: "2026-10-23T05:34:03-07:00"
            - harness: codex
              id: 01a168ab-54df-7c01-8ac0-4585962b813f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: f1b522d6e7d3e8c92b0590d06eab78c08e472aca9d08b5a0fda076944c0eac45
              started: "2026-10-23T12:10:21.706Z"
              spawned: false
              bytes: 412483
              windows:
                - start: "2026-10-23T12:10:21.706Z"
                  end: "2026-10-23T05:34:03-07:00"
            - harness: codex
              id: 01a168ad-70cc-7cd1-9eef-d36c47c664ea
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 11cfa55d1f5b7c87394a55b661e1498c15abc3d97aceafe93a733e68452e472b
              started: "2026-10-23T12:12:39.913Z"
              spawned: false
              bytes: 424029
              windows:
                - start: "2026-10-23T12:12:39.913Z"
                  end: "2026-10-23T05:34:03-07:00"
            - harness: codex
              id: 01a168ae-07ac-7712-b1f5-80a3ecf73c25
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0499eb9ff178ec31a8fa1698ec22703a2fefbea34cb49e6389f997c29fef30ae
              started: "2026-10-23T12:13:18.617Z"
              spawned: false
              bytes: 365523
              windows:
                - start: "2026-10-23T12:13:18.617Z"
                  end: "2026-10-23T05:34:03-07:00"
            - harness: codex
              id: 01a168ae-eb19-7db2-9fcb-2a5b77f384f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              digest: 0d0fa6b64a0bca4e562631102e316c906268514d71f9ca23a2a363a18215285a
              started: "2026-10-23T12:14:16.434Z"
              spawned: true
              bytes: 1630894
              windows:
                - start: "2026-10-23T12:14:16.434Z"
                  end: "2026-10-23T05:34:03-07:00"
            - harness: codex
              id: 01a168af-1978-7970-9324-ac1dace7b00e
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              digest: cd059e854d7a4d7e65a5bfce105cd3c681acfc4cdd51b8e424d879a5a774a5b9
              started: "2026-10-23T12:14:28.316Z"
              spawned: true
              bytes: 1532507
              windows:
                - start: "2026-10-23T12:14:28.316Z"
                  end: "2026-10-23T05:34:03-07:00"
      "n": 1
      timestamp: "2026-10-23T05:37:50-07:00"
      dispatched: "2026-10-23T05:34:34-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Local queue lifecycle diagnostics still bypass attention
          detail: 'cmd/internal/couchtty/console_lifecycle_failure.go:48 publishes DiagnosticWarning through feed-only Notify, including unproven cleanup and diagnostic persistence failures. DispatchOperation skips its new warning callback for the attempt already carried by local queue requests, so it cannot compensate. ARCH-PURPOSE: route this terminal owner through Warn with source identity and add a local queue regression proving attention publication, as required by the plan''s explicit finishLifecycleAttempt audit anchor.'
          family: actionable-producer-coverage
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Successful Chrome fidelity reports become attention obligations
          detail: 'cmd/internal/couchtty/console_webcast.go:58-60 routes every CastConfig.Logf message through setWarning. cmd/internal/webcast/cast.go:519-524 emits successful fidelity summaries through that callback, and production webcastSettings enables the check for recorded casts. ARCH-PURPOSE: classify outcomes at their producers so routine success remains feed-only; add recorded-cast success and actionable-failure attention regressions without English-text classification.'
          family: attention-requires-actionable-source
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5c1563ac-423e-457c-afa5-4d89e9d5ab4e
            repository: github.com/xianxu/pair
            issue_id: "000812"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/06e98f190d7d2bb0
            reviewed_head: 2e1dccf9dd632fefffcbdf854c45b636b4f7525a
            slot: pair:1
            requested_at: "2026-10-23T12:48:58.477558Z"
          phase: finished
          pid: 73169
          started_at: "2026-10-23T12:48:58.57536Z"
          finished_at: "2026-10-23T12:55:17.272569Z"
          session_id: 01a168ce-b1e1-7723-8dee-3f2535a0c8f8
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2e1dccf9dd632fefffcbdf854c45b636b4f7525a
        tracker_head: 51698e2f7f3866a4baad5443693d03b62a3d5c1e
        branch_point: c0dbb32decc052500172ded01ff36e3fa8e49cc9
        since: "2026-10-23T05:01:56-07:00"
        until: "2026-10-23T05:48:09-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T05:01:56-07:00"
              end: "2026-10-23T05:48:09-07:00"
              began: "2026-10-23T05:01:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 87293a6b73ab769f6290232ae601054e426c0e07f1c44bbc72c44735639fe994
              started: "2026-10-21T18:31:20.491Z"
              spawned: false
              bytes: 25798004
              windows:
                - start: "2026-10-23T05:01:56-07:00"
                  end: "2026-10-23T05:48:09-07:00"
            - harness: codex
              id: 01a168a8-b55a-7a73-8097-1ac736feaf5d
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 06c8f98592b07caa7e74f16b9ceacb0a17b13596b7a74f6364075a3d9793593b
              started: "2026-10-23T12:07:29.699Z"
              spawned: false
              bytes: 561161
              windows:
                - start: "2026-10-23T12:07:29.699Z"
                  end: "2026-10-23T05:48:09-07:00"
            - harness: codex
              id: 01a168ab-54df-7c01-8ac0-4585962b813f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: f1b522d6e7d3e8c92b0590d06eab78c08e472aca9d08b5a0fda076944c0eac45
              started: "2026-10-23T12:10:21.706Z"
              spawned: false
              bytes: 412483
              windows:
                - start: "2026-10-23T12:10:21.706Z"
                  end: "2026-10-23T05:48:09-07:00"
            - harness: codex
              id: 01a168ad-70cc-7cd1-9eef-d36c47c664ea
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 11cfa55d1f5b7c87394a55b661e1498c15abc3d97aceafe93a733e68452e472b
              started: "2026-10-23T12:12:39.913Z"
              spawned: false
              bytes: 424029
              windows:
                - start: "2026-10-23T12:12:39.913Z"
                  end: "2026-10-23T05:48:09-07:00"
            - harness: codex
              id: 01a168ae-07ac-7712-b1f5-80a3ecf73c25
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0499eb9ff178ec31a8fa1698ec22703a2fefbea34cb49e6389f997c29fef30ae
              started: "2026-10-23T12:13:18.617Z"
              spawned: false
              bytes: 365523
              windows:
                - start: "2026-10-23T12:13:18.617Z"
                  end: "2026-10-23T05:48:09-07:00"
            - harness: codex
              id: 01a168ae-eb19-7db2-9fcb-2a5b77f384f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              digest: 0d0fa6b64a0bca4e562631102e316c906268514d71f9ca23a2a363a18215285a
              started: "2026-10-23T12:14:16.434Z"
              spawned: true
              bytes: 1630894
              windows:
                - start: "2026-10-23T12:14:16.434Z"
                  end: "2026-10-23T05:48:09-07:00"
            - harness: codex
              id: 01a168af-1978-7970-9324-ac1dace7b00e
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              digest: 277ecbc33a2db557d42b119159ea5f761d808bb247d175e9a47bc348211e32ce
              started: "2026-10-23T12:14:28.316Z"
              spawned: true
              bytes: 2101902
              windows:
                - start: "2026-10-23T12:14:28.316Z"
                  end: "2026-10-23T05:48:09-07:00"
        cuts:
            - dispatched: "2026-10-23T05:34:34-07:00"
              finalized: "2026-10-23T05:37:50-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-23T05:55:18-07:00"
      dispatched: "2026-10-23T05:48:56-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: finishLifecycleAttempt now calls Warn with the origin thread. Reversing that change made TestLifecycleLocalQueuedCompletionWarningOwnsExactOrigin fail for cleanup and diagnostic persistence; the pinned-head regression passes under race.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Typed Warnf separates actionable failures from informational Logf. Restoring generic-log warning routing made the actual recorded-cast healthy-fidelity regression fail with attention=1; healthy/error regressions pass under race on the pinned head.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Critical
          title: The background low-priority-agent warning still bypasses attention
          detail: 'cmd/internal/couchcmd/message_service.go:582 wires the renicer to feed-only Notify. renicer.go:180 emits a once-per-slot instruction to relaunch an agent whose priority cannot be restored automatically; removing tab notices leaves this actionable asynchronous warning without attention. ARCH-PURPOSE: This is the 2nd finding in family `actionable-producer-coverage`. Earlier rounds fixed instances. State the rule covering all asynchronous sources requiring operator intervention, enumerate every remaining Notify/setNotice callback against it, and sweep that enumeration in this round. Route this source through Warn with its binding identity and add a production-wiring regression plus ordinary-operation negative controls.'
          family: actionable-producer-coverage
          round: 2
        - kind: defect
          id: BR-4
          severity: Important
          title: Affected regression tests still enforce removed behavior or use an invalid success fixture
          detail: 'cmd/internal/couchtty/terminal_migration_test.go:23,89,144,170, console_test.go:267, and cmd/internal/couchcmd/attention_ephemeral_test.go:90 still require notice text in terminal chrome and fail on the pinned head. Preserve their parser-isolation, publication-order, resize, focus, and persistence invariants using supported visible surfaces. Separately, notice_attention_test.go:124 calls a fixture with a background borrowed attachment “ordinary successful startup”; four of five diagnostic runs failed with the legitimate BORROWED ATTACH warning. Supply confirmed ownership and wait for background completions before asserting absence. ARCH-ORDER: retain controlled ordering coverage while migrating these oracles.'
          family: regression-oracles-follow-contract
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a09cbb80-6972-4c9c-b4ed-07931fc6c3c5
            repository: github.com/xianxu/pair
            issue_id: "000812"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/53e4eba121cf521e
            reviewed_head: cafbf526b5ca0ecfbd4b75518de7ac3758585ebf
            slot: pair:1
            requested_at: "2026-10-23T13:20:21.224581Z"
          phase: finished
          pid: 68622
          started_at: "2026-10-23T13:20:21.333787Z"
          finished_at: "2026-10-23T13:30:39.668318Z"
          session_id: 01a168eb-6c8a-7a01-b817-90049483eede
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: cafbf526b5ca0ecfbd4b75518de7ac3758585ebf
        tracker_head: 51698e2f7f3866a4baad5443693d03b62a3d5c1e
        branch_point: c0dbb32decc052500172ded01ff36e3fa8e49cc9
        since: "2026-10-23T05:01:56-07:00"
        until: "2026-10-23T06:16:43-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T05:01:56-07:00"
              end: "2026-10-23T06:16:43-07:00"
              began: "2026-10-23T05:01:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 019f30e0c51a4ced949737487db01c416c82e2cbdf86e8cd031d13d2b204b794
              started: "2026-10-21T18:31:20.491Z"
              spawned: false
              bytes: 27753984
              windows:
                - start: "2026-10-23T05:01:56-07:00"
                  end: "2026-10-23T06:16:43-07:00"
            - harness: codex
              id: 01a168a8-b55a-7a73-8097-1ac736feaf5d
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 06c8f98592b07caa7e74f16b9ceacb0a17b13596b7a74f6364075a3d9793593b
              started: "2026-10-23T12:07:29.699Z"
              spawned: false
              bytes: 561161
              windows:
                - start: "2026-10-23T12:07:29.699Z"
                  end: "2026-10-23T06:16:43-07:00"
            - harness: codex
              id: 01a168ab-54df-7c01-8ac0-4585962b813f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: f1b522d6e7d3e8c92b0590d06eab78c08e472aca9d08b5a0fda076944c0eac45
              started: "2026-10-23T12:10:21.706Z"
              spawned: false
              bytes: 412483
              windows:
                - start: "2026-10-23T12:10:21.706Z"
                  end: "2026-10-23T06:16:43-07:00"
            - harness: codex
              id: 01a168ad-70cc-7cd1-9eef-d36c47c664ea
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 11cfa55d1f5b7c87394a55b661e1498c15abc3d97aceafe93a733e68452e472b
              started: "2026-10-23T12:12:39.913Z"
              spawned: false
              bytes: 424029
              windows:
                - start: "2026-10-23T12:12:39.913Z"
                  end: "2026-10-23T06:16:43-07:00"
            - harness: codex
              id: 01a168ae-07ac-7712-b1f5-80a3ecf73c25
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0499eb9ff178ec31a8fa1698ec22703a2fefbea34cb49e6389f997c29fef30ae
              started: "2026-10-23T12:13:18.617Z"
              spawned: false
              bytes: 365523
              windows:
                - start: "2026-10-23T12:13:18.617Z"
                  end: "2026-10-23T06:16:43-07:00"
            - harness: codex
              id: 01a168ae-eb19-7db2-9fcb-2a5b77f384f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              digest: 2d267aa4923b553fadaf141cbcb46a16ad846e6a49ff38c2e32a72e7b3e67837
              started: "2026-10-23T12:14:16.434Z"
              spawned: true
              bytes: 1992488
              windows:
                - start: "2026-10-23T12:14:16.434Z"
                  end: "2026-10-23T06:16:43-07:00"
            - harness: codex
              id: 01a168af-1978-7970-9324-ac1dace7b00e
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15fbb-6812-71c3-b0f5-b4651b817396
              digest: b0fc83d9781ccdc30eeda4ee77db2a29f121fdf300b3aeb6ffa10f7806564caa
              started: "2026-10-23T12:14:28.316Z"
              spawned: true
              bytes: 3290635
              windows:
                - start: "2026-10-23T12:14:28.316Z"
                  end: "2026-10-23T06:16:43-07:00"
        cuts:
            - dispatched: "2026-10-23T05:34:34-07:00"
              finalized: "2026-10-23T05:37:50-07:00"
              finalizing: false
            - dispatched: "2026-10-23T05:48:56-07:00"
              finalized: "2026-10-23T05:55:18-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-23T06:30:41-07:00"
      dispatched: "2026-10-23T06:20:19-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-3
          disposition: addressed
          note: Production uses newConsoleRenicer with exact binding identity. Reverting Warn to Notify in a disposable checkout fails TestRenicerConsoleWarningIsOncePerExactBinding; normal-priority controls pass. The plan now enumerates remaining producer classes.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Named terminal oracles now assert supported badges, retained feed and switcher surfaces while preserving ordering/focus invariants. Startup supplies retained ownership evidence and joins restart-drained. It passes 20 repetitions; restoring the previous fixture fails all five reversal runs.
          round: 3
      findings:
        - kind: defect
          id: BR-5
          severity: Important
          title: Async warning test opens attention before publication is guaranteed
          detail: 'cmd/internal/couchcmd/lifecycle_diagnostic_attention_test.go:19 calls openWarningAttention immediately after queuing the diagnostic. The helper at notice_attention_test.go:45 toggles first, but console_attention.go:144 leaves an empty list closed; later warning publication does not open it. The complete package run stalled here for minutes, confirmed by its captured stack, and was interrupted at 470.7 seconds. This is the 2nd finding in family regression-oracles-follow-contract. ARCH-ORDER: fix the class by requiring publication acknowledgment before opening attention across every asynchronous helper caller, and force both publication/input orderings in regression coverage.'
          family: regression-oracles-follow-contract
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#812 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T05:37:50-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `actionable-producer-coverage` (kind: defect) Local queue lifecycle diagnostics still bypass attention
  cmd/internal/couchtty/console_lifecycle_failure.go:48 publishes DiagnosticWarning through feed-only Notify, including unproven cleanup and diagnostic persistence failures. DispatchOperation skips its new warning callback for the attempt already carried by local queue requests, so it cannot compensate. ARCH-PURPOSE: route this terminal owner through Warn with source identity and add a local queue regression proving attention publication, as required by the plan's explicit finishLifecycleAttempt audit anchor.
- **BR-2** [Critical] `attention-requires-actionable-source` (kind: defect) Successful Chrome fidelity reports become attention obligations
  cmd/internal/couchtty/console_webcast.go:58-60 routes every CastConfig.Logf message through setWarning. cmd/internal/webcast/cast.go:519-524 emits successful fidelity summaries through that callback, and production webcastSettings enables the check for recorded casts. ARCH-PURPOSE: classify outcomes at their producers so routine success remains feed-only; add recorded-cast success and actionable-failure attention regressions without English-text classification.

## Round 2 — 2026-10-23T05:55:18-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — finishLifecycleAttempt now calls Warn with the origin thread. Reversing that change made TestLifecycleLocalQueuedCompletionWarningOwnsExactOrigin fail for cleanup and diagnostic persistence; the pinned-head regression passes under race.
- BR-2 — addressed — Typed Warnf separates actionable failures from informational Logf. Restoring generic-log warning routing made the actual recorded-cast healthy-fidelity regression fail with attention=1; healthy/error regressions pass under race on the pinned head.

### Raised

- **BR-3** [Critical] `actionable-producer-coverage` (kind: defect) The background low-priority-agent warning still bypasses attention
  cmd/internal/couchcmd/message_service.go:582 wires the renicer to feed-only Notify. renicer.go:180 emits a once-per-slot instruction to relaunch an agent whose priority cannot be restored automatically; removing tab notices leaves this actionable asynchronous warning without attention. ARCH-PURPOSE: This is the 2nd finding in family `actionable-producer-coverage`. Earlier rounds fixed instances. State the rule covering all asynchronous sources requiring operator intervention, enumerate every remaining Notify/setNotice callback against it, and sweep that enumeration in this round. Route this source through Warn with its binding identity and add a production-wiring regression plus ordinary-operation negative controls.
- **BR-4** [Important] `regression-oracles-follow-contract` (kind: defect) Affected regression tests still enforce removed behavior or use an invalid success fixture
  cmd/internal/couchtty/terminal_migration_test.go:23,89,144,170, console_test.go:267, and cmd/internal/couchcmd/attention_ephemeral_test.go:90 still require notice text in terminal chrome and fail on the pinned head. Preserve their parser-isolation, publication-order, resize, focus, and persistence invariants using supported visible surfaces. Separately, notice_attention_test.go:124 calls a fixture with a background borrowed attachment “ordinary successful startup”; four of five diagnostic runs failed with the legitimate BORROWED ATTACH warning. Supply confirmed ownership and wait for background completions before asserting absence. ARCH-ORDER: retain controlled ordering coverage while migrating these oracles.

## Round 3 — 2026-10-23T06:30:41-07:00 (codex) — BLOCKED

### Disposed

- BR-3 — addressed — Production uses newConsoleRenicer with exact binding identity. Reverting Warn to Notify in a disposable checkout fails TestRenicerConsoleWarningIsOncePerExactBinding; normal-priority controls pass. The plan now enumerates remaining producer classes.
- BR-4 — addressed — Named terminal oracles now assert supported badges, retained feed and switcher surfaces while preserving ordering/focus invariants. Startup supplies retained ownership evidence and joins restart-drained. It passes 20 repetitions; restoring the previous fixture fails all five reversal runs.

### Raised

- **BR-5** [Important] `regression-oracles-follow-contract` (kind: defect) Async warning test opens attention before publication is guaranteed
  cmd/internal/couchcmd/lifecycle_diagnostic_attention_test.go:19 calls openWarningAttention immediately after queuing the diagnostic. The helper at notice_attention_test.go:45 toggles first, but console_attention.go:144 leaves an empty list closed; later warning publication does not open it. The complete package run stalled here for minutes, confirmed by its captured stack, and was interrupted at 470.7 seconds. This is the 2nd finding in family regression-oracles-follow-contract. ARCH-ORDER: fix the class by requiring publication acknowledgment before opening attention across every asynchronous helper caller, and force both publication/input orderings in regression coverage.

## Open findings

- **BR-5** [Important] `regression-oracles-follow-contract` (kind: defect) Async warning test opens attention before publication is guaranteed
