---
gate: boundary-review
issue: 630
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 0c1056f8-af6c-42a9-9643-95d3219af226
            repository: github.com/xianxu/pair
            issue_id: "000630"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7d6f54f42e7a59c1
            reviewed_head: 4b330574b832255236b1d146d82c4b278e48e735
            slot: pair:6
            requested_at: "2026-10-17T13:59:03.900214Z"
          phase: finished
          pid: 81446
          started_at: "2026-10-17T13:59:04.170244Z"
          finished_at: "2026-10-17T14:01:18.498979Z"
          session_id: 01a14a28-b70e-7df3-b7f9-816ce57d9a31
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4b330574b832255236b1d146d82c4b278e48e735
        tracker_head: ee8a52d3393701d436f809740afecb55ad4e9a0e
        branch_point: f823355f9b9988dfb9888eed1bac54de44edc10f
        since: "2026-10-17T05:43:25-07:00"
        until: "2026-10-17T06:58:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T05:43:25-07:00"
              end: "2026-10-17T06:58:24-07:00"
              began: "2026-10-17T05:43:25-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14367-664d-7850-a55c-bb707864cce5
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 53b3a74fd6b1e09aa4e627d22d561ec49362789a844569f7a866addd7b177c3d
              started: "2026-10-16T06:30:12.567Z"
              spawned: false
              bytes: 44029352
              windows:
                - start: "2026-10-17T05:43:25-07:00"
                  end: "2026-10-17T06:58:24-07:00"
            - harness: codex
              id: 01a149e8-2b6d-7b12-a8b3-8f9b1ae5ddaa
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f608cc4e4108a44cdf832a8cd5dd30462cc03e05a16f6d1f28bd8e3ddda01f0f
              started: "2026-10-17T12:48:35.256Z"
              spawned: false
              bytes: 594776
              windows:
                - start: "2026-10-17T12:48:35.256Z"
                  end: "2026-10-17T06:58:24-07:00"
            - harness: codex
              id: 01a149e8-4a21-7280-a2b2-02c22b9c8a87
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: cd49fadc37e79e9fb23fe9571d1f4f4f2c44dd29ebebe387a4856040ae9ea53a
              started: "2026-10-17T12:48:42.815Z"
              spawned: true
              bytes: 1042348
              windows:
                - start: "2026-10-17T12:48:42.815Z"
                  end: "2026-10-17T06:58:24-07:00"
            - harness: codex
              id: 01a149ea-778d-7812-9060-f8cdeb0da324
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2256a261de5e09cd208595faf2bf2a3e33c6b14e0c0e3677a82bf95f6beeecc6
              started: "2026-10-17T12:51:05.845Z"
              spawned: false
              bytes: 421559
              windows:
                - start: "2026-10-17T12:51:05.845Z"
                  end: "2026-10-17T06:58:24-07:00"
            - harness: codex
              id: 01a149ec-6536-7a30-8eff-798075ed8eaf
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 09a48ab807517cd0580829bc39ba58bd6d807d32792553908a620f2cfeb2e785
              started: "2026-10-17T12:53:12.346Z"
              spawned: false
              bytes: 418577
              windows:
                - start: "2026-10-17T12:53:12.346Z"
                  end: "2026-10-17T06:58:24-07:00"
            - harness: codex
              id: 01a149ec-ca2b-73c1-b700-b9bfac84db4b
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 7de1e50de20812765efd70e1eba58fd0edb81571809a4a0ed4a2f2e5360ea6e7
              started: "2026-10-17T12:53:37.94Z"
              spawned: false
              bytes: 380216
              windows:
                - start: "2026-10-17T12:53:37.94Z"
                  end: "2026-10-17T06:58:24-07:00"
            - harness: codex
              id: 01a149f2-d3a9-7a81-97f9-eecd8fd9eb88
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 175cf0f61cb7214f5ce566bc82c8671a7611c9dce18f0effd51b8e04d91c1593
              started: "2026-10-17T13:00:13.197Z"
              spawned: true
              bytes: 652201
              windows:
                - start: "2026-10-17T13:00:13.197Z"
                  end: "2026-10-17T06:58:24-07:00"
      "n": 1
      timestamp: "2026-10-17T07:01:19-07:00"
      dispatched: "2026-10-17T06:59:01-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Reject missing registrations before entering legacy document IO.
          detail: 'cmd/internal/sessionsync/registry.go:36 calls withDocument, which falls back to locked(path, fn) when the registration is absent (lines 581–583). The rejection at line 37 therefore occurs only after legacy file loading and an unbounded LOCK_EX acquisition (sync.go:145–150). If a file is forgotten between worker enumeration and inspection and its legacy flock is held, the reminder worker blocks while holding the registry lock; neither its 30-second context nor service cancellation can interrupt it. ARCH-CONSTRAINTS/ARCH-ORDER: add a registered-only route that rejects absence under the registry lock before legacy IO, and regress inspection of an unregistered file with a held legacy flock.'
          family: registered-inspection-fails-closed
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 401810d4-9cc9-43f0-873a-7682683d1b65
            repository: github.com/xianxu/pair
            issue_id: "000630"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/c800cc726b9c5207
            reviewed_head: e87b903a0ab15c4b7c3e3e3da6e151e88c9e7659
            slot: pair:6
            requested_at: "2026-10-17T14:47:40.678427Z"
          phase: finished
          pid: 81304
          started_at: "2026-10-17T14:47:40.907113Z"
          finished_at: "2026-10-17T14:49:49.85577Z"
          session_id: 01a14a55-3851-7b02-b7ea-4f3bfe3935b4
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e87b903a0ab15c4b7c3e3e3da6e151e88c9e7659
        tracker_head: 1c3a4deb1c3396be7b56a2b6238989c1b7b99d4e
        branch_point: f823355f9b9988dfb9888eed1bac54de44edc10f
        since: "2026-10-17T05:43:25-07:00"
        until: "2026-10-17T07:47:07-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T05:43:25-07:00"
              end: "2026-10-17T07:47:07-07:00"
              began: "2026-10-17T05:43:25-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14367-664d-7850-a55c-bb707864cce5
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: ce7910de3b821e112d29182d8a7ff7769ca13160f8e5bb7c91b2e73ed78f652a
              started: "2026-10-16T06:30:12.567Z"
              spawned: false
              bytes: 44990550
              windows:
                - start: "2026-10-17T05:43:25-07:00"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a149e8-2b6d-7b12-a8b3-8f9b1ae5ddaa
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f608cc4e4108a44cdf832a8cd5dd30462cc03e05a16f6d1f28bd8e3ddda01f0f
              started: "2026-10-17T12:48:35.256Z"
              spawned: false
              bytes: 594776
              windows:
                - start: "2026-10-17T12:48:35.256Z"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a149e8-4a21-7280-a2b2-02c22b9c8a87
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: cd49fadc37e79e9fb23fe9571d1f4f4f2c44dd29ebebe387a4856040ae9ea53a
              started: "2026-10-17T12:48:42.815Z"
              spawned: true
              bytes: 1042348
              windows:
                - start: "2026-10-17T12:48:42.815Z"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a149ea-778d-7812-9060-f8cdeb0da324
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2256a261de5e09cd208595faf2bf2a3e33c6b14e0c0e3677a82bf95f6beeecc6
              started: "2026-10-17T12:51:05.845Z"
              spawned: false
              bytes: 421559
              windows:
                - start: "2026-10-17T12:51:05.845Z"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a149ec-6536-7a30-8eff-798075ed8eaf
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 09a48ab807517cd0580829bc39ba58bd6d807d32792553908a620f2cfeb2e785
              started: "2026-10-17T12:53:12.346Z"
              spawned: false
              bytes: 418577
              windows:
                - start: "2026-10-17T12:53:12.346Z"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a149ec-ca2b-73c1-b700-b9bfac84db4b
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 7de1e50de20812765efd70e1eba58fd0edb81571809a4a0ed4a2f2e5360ea6e7
              started: "2026-10-17T12:53:37.94Z"
              spawned: false
              bytes: 380216
              windows:
                - start: "2026-10-17T12:53:37.94Z"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a149f2-d3a9-7a81-97f9-eecd8fd9eb88
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 175cf0f61cb7214f5ce566bc82c8671a7611c9dce18f0effd51b8e04d91c1593
              started: "2026-10-17T13:00:13.197Z"
              spawned: true
              bytes: 652201
              windows:
                - start: "2026-10-17T13:00:13.197Z"
                  end: "2026-10-17T07:47:07-07:00"
            - harness: codex
              id: 01a14a2c-5982-7830-b55b-aac61b145a79
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: b086970d34a6bcf826268a03fc683104c8fa621e3f7ffba4db7d24e056a8d00e
              started: "2026-10-17T14:03:03Z"
              spawned: true
              bytes: 498160
              windows:
                - start: "2026-10-17T14:03:03Z"
                  end: "2026-10-17T07:47:07-07:00"
        cuts:
            - dispatched: "2026-10-17T06:59:01-07:00"
              finalized: "2026-10-17T07:01:19-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-17T07:49:51-07:00"
      dispatched: "2026-10-17T07:47:37-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: registry.go:593 rejects absent registrations under the registry lock before legacy IO. Focused and race tests pass. In disposable copies, restoring legacy fallback made both held-flock cases and the forgotten-during-check shutdown regression fail. Temporary copies were removed.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: e87b903a0ab15c4b7c3e3e3da6e151e88c9e7659
      blocked: false
---

# Gate ledger — pair#630 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T07:01:19-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `registered-inspection-fails-closed` (kind: defect) Reject missing registrations before entering legacy document IO.
  cmd/internal/sessionsync/registry.go:36 calls withDocument, which falls back to locked(path, fn) when the registration is absent (lines 581–583). The rejection at line 37 therefore occurs only after legacy file loading and an unbounded LOCK_EX acquisition (sync.go:145–150). If a file is forgotten between worker enumeration and inspection and its legacy flock is held, the reminder worker blocks while holding the registry lock; neither its 30-second context nor service cancellation can interrupt it. ARCH-CONSTRAINTS/ARCH-ORDER: add a registered-only route that rejects absence under the registry lock before legacy IO, and regress inspection of an unregistered file with a held legacy flock.

## Round 2 — 2026-10-17T07:49:51-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — registry.go:593 rejects absent registrations under the registry lock before legacy IO. Focused and race tests pass. In disposable copies, restoring legacy fallback made both held-flock cases and the forgotten-during-check shutdown regression fail. Temporary copies were removed.

## Open findings

(none — every finding has been disposed)
