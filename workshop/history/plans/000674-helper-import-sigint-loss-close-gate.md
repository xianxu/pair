---
gate: boundary-review
issue: 674
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: dacf75bc-0f71-4c10-ae54-64900e1aa5a9
            repository: github.com/xianxu/pair
            issue_id: "000674"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/f77e15398001ad86
            reviewed_head: 902e3b20a057a77c6375d270a655cdb285d971bb
            slot: pair:2
            requested_at: "2026-10-18T20:41:48.34433Z"
          phase: finished
          pid: 43638
          started_at: "2026-10-18T20:41:48.57428Z"
          finished_at: "2026-10-18T20:44:31.536232Z"
          session_id: f0f5bd0f-ca12-4ffe-9fa7-c3aad7c60c68
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 902e3b20a057a77c6375d270a655cdb285d971bb
        tracker_head: f8219055a09176ddafc3512b8e99a6b8ee3abdfb
        branch_point: 8c302ba8d673a9fcffa0ffeadd433091e5ae5a26
        since: "2026-10-18T12:30:25-07:00"
        until: "2026-10-18T12:34:18-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T12:30:25-07:00"
              end: "2026-10-18T12:34:18-07:00"
              began: "2026-10-18T12:30:25-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: d42e776f2c4e56d363f8ac016da1853750e0a788020192d05f67d4032195e187
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 36474721
              windows:
                - start: "2026-10-18T12:30:25-07:00"
                  end: "2026-10-18T12:34:18-07:00"
      "n": 1
      timestamp: "2026-10-18T13:44:33-07:00"
      dispatched: "2026-10-18T13:41:45-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: test_cancellation_during_startup keeps a stale "exit 1" comment and only asserts nonzero; it can now require 128+signal
          detail: With launch-time blocking, Popen returns post-exec with the signals pending, so cancel always runs; tighten tests/without-ancestors-test.py:142-145 to assertEqual(code, 128 + cancellation) and drop the stale comment.
          family: stale-weak-assertion-after-fix
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: caller_blocked diagnostic is sampled after the mask is restored, so it never shows the launch-time child mask
          detail: tests/without-ancestors-test.py:78 always reports the restored caller mask; only the gate bootstrap reports the child's mask, so the Plan's diagnostics claim is narrower than stated.
          family: diagnostic-reports-wrong-moment
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Helper unconditionally unblocks SIGINT/SIGTERM, and non-fixture launchers (main_test.go, pair-embedded-runtime-test.sh) keep the startup window
          detail: tests/without-ancestors.py:38 overrides any caller's deliberate block; other consumers do not block before launch, so the comment should say only the fixture launcher closes the window.
          family: helper-contract-undocumented
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: 902e3b20a057a77c6375d270a655cdb285d971bb
      blocked: false
---

# Gate ledger — pair#674 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T13:44:33-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `stale-weak-assertion-after-fix` (kind: defect) test_cancellation_during_startup keeps a stale "exit 1" comment and only asserts nonzero; it can now require 128+signal
  With launch-time blocking, Popen returns post-exec with the signals pending, so cancel always runs; tighten tests/without-ancestors-test.py:142-145 to assertEqual(code, 128 + cancellation) and drop the stale comment.
- **BR-2** [Minor] `diagnostic-reports-wrong-moment` (kind: defect) caller_blocked diagnostic is sampled after the mask is restored, so it never shows the launch-time child mask
  tests/without-ancestors-test.py:78 always reports the restored caller mask; only the gate bootstrap reports the child's mask, so the Plan's diagnostics claim is narrower than stated.
- **BR-3** [Minor] `helper-contract-undocumented` (kind: defect) Helper unconditionally unblocks SIGINT/SIGTERM, and non-fixture launchers (main_test.go, pair-embedded-runtime-test.sh) keep the startup window
  tests/without-ancestors.py:38 overrides any caller's deliberate block; other consumers do not block before launch, so the comment should say only the fixture launcher closes the window.

## Open findings

- **BR-1** [Minor] `stale-weak-assertion-after-fix` (kind: defect) test_cancellation_during_startup keeps a stale "exit 1" comment and only asserts nonzero; it can now require 128+signal
- **BR-2** [Minor] `diagnostic-reports-wrong-moment` (kind: defect) caller_blocked diagnostic is sampled after the mask is restored, so it never shows the launch-time child mask
- **BR-3** [Minor] `helper-contract-undocumented` (kind: defect) Helper unconditionally unblocks SIGINT/SIGTERM, and non-fixture launchers (main_test.go, pair-embedded-runtime-test.sh) keep the startup window
