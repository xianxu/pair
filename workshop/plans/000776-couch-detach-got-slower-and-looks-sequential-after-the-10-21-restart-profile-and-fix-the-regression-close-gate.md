---
gate: boundary-review
issue: 776
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 95b66f25-f3eb-4d86-b6b4-0cacf6679991
            repository: github.com/xianxu/pair
            issue_id: "000776"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9764e9b79d971783
            reviewed_head: f41bac24797392cca7187a74d645e00072c8020e
            slot: pair:6
            requested_at: "2026-10-21T20:05:16.896795Z"
          phase: finished
          pid: 32838
          started_at: "2026-10-21T20:05:17.005238Z"
          finished_at: "2026-10-21T20:07:16.720138Z"
          session_id: 01a16011-6d5b-77e2-9120-835c348f7095
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: f41bac24797392cca7187a74d645e00072c8020e
        tracker_head: 4d0b9ff0507eebd70316fee7b01a23973c47db46
        branch_point: a9522724d4d84586afe222d201029d0713a89fc8
        since: "2026-10-21T12:04:14-07:00"
        until: "2026-10-21T13:03:59-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T12:04:14-07:00"
              end: "2026-10-21T13:03:59-07:00"
              began: "2026-10-21T12:04:14-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 784062db4e4f505e8f52e7c7c9934429d6a1f75c6b17aa3da42e152e2b07384c
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 90056293
              windows:
                - start: "2026-10-21T12:04:14-07:00"
                  end: "2026-10-21T13:03:59-07:00"
      "n": 1
      timestamp: "2026-10-21T13:07:26-07:00"
      dispatched: "2026-10-21T13:05:15-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Align Done when with the approved cancellation regression contract
          detail: 'workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md:47-48 still requires a regression that bounds detach time or asserts parallel retirement. The revised implementation instead supplies deterministic group-completion tests (provision_cancel_test.go:49,83) and a descendant-lease control (provision_subprocess_test.go:145). The existing parallel-Leave test also passes without this fix, so it cannot establish that regression requirement. Family enumeration: this second Done-when bullet is the sole stale acceptance clause in this window. Revise it explicitly to accept completion before grace on confirmed group absence, while preserving descendant cleanup and error handling.'
          family: acceptance-contract-traceability
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 145d4c4f-c047-43d0-a2bf-b38ea31df120
            repository: github.com/xianxu/pair
            issue_id: "000776"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/4f8f627c22e4a810
            reviewed_head: 1af622b6443e765282dc49d2e1d7209d338c49cd
            slot: pair:6
            requested_at: "2026-10-21T20:08:40.713486Z"
          phase: finished
          pid: 74128
          started_at: "2026-10-21T20:08:40.818485Z"
          finished_at: "2026-10-21T20:09:44.411139Z"
          session_id: 01a16014-8999-7e71-b2a8-1bf4f7e029c8
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1af622b6443e765282dc49d2e1d7209d338c49cd
        tracker_head: 4d0b9ff0507eebd70316fee7b01a23973c47db46
        branch_point: a9522724d4d84586afe222d201029d0713a89fc8
        since: "2026-10-21T12:04:14-07:00"
        until: "2026-10-21T13:08:17-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T12:04:14-07:00"
              end: "2026-10-21T13:08:17-07:00"
              began: "2026-10-21T12:04:14-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 9596607a70fc972633707e12a05e350a47d5c2472f307549453839068c6a6a22
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 90258979
              windows:
                - start: "2026-10-21T12:04:14-07:00"
                  end: "2026-10-21T13:08:17-07:00"
        cuts:
            - dispatched: "2026-10-21T13:05:15-07:00"
              finalized: "2026-10-21T13:07:26-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T13:09:45-07:00"
      dispatched: "2026-10-21T13:08:38-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The pinned before/after diff replaces the stale timing-or-parallelism requirement at issue lines 46–47 with completion before grace on confirmed group absence, conservative unavailable-probe handling, and native descendant lease cleanup. These requirements match provision_cancel_test.go:49,83 and provision_subprocess_test.go:145. The Revisions section preserves the approval and original clause.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 1af622b6443e765282dc49d2e1d7209d338c49cd
      blocked: false
---

# Gate ledger — pair#776 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T13:07:26-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `acceptance-contract-traceability` (kind: defect) Align Done when with the approved cancellation regression contract
  workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md:47-48 still requires a regression that bounds detach time or asserts parallel retirement. The revised implementation instead supplies deterministic group-completion tests (provision_cancel_test.go:49,83) and a descendant-lease control (provision_subprocess_test.go:145). The existing parallel-Leave test also passes without this fix, so it cannot establish that regression requirement. Family enumeration: this second Done-when bullet is the sole stale acceptance clause in this window. Revise it explicitly to accept completion before grace on confirmed group absence, while preserving descendant cleanup and error handling.

## Round 2 — 2026-10-21T13:09:45-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The pinned before/after diff replaces the stale timing-or-parallelism requirement at issue lines 46–47 with completion before grace on confirmed group absence, conservative unavailable-probe handling, and native descendant lease cleanup. These requirements match provision_cancel_test.go:49,83 and provision_subprocess_test.go:145. The Revisions section preserves the approval and original clause.

## Open findings

(none — every finding has been disposed)
