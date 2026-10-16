---
gate: boundary-review
issue: 593
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 688632a3-ab2a-4ebd-a0ea-88c5c4e5129e
            repository: github.com/xianxu/pair
            issue_id: "000593"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/5f31f29c012d12c2
            reviewed_head: efaedce1e1978905b2c27e03e1cd3d119f23cc46
            slot: pair:1
            requested_at: "2026-10-16T03:51:27.814323Z"
          phase: finished
          pid: 1131
          started_at: "2026-10-16T03:51:28.069743Z"
          finished_at: "2026-10-16T03:54:27.377121Z"
          session_id: 01a142d6-1427-78a2-bb8d-a18a38be5107
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: efaedce1e1978905b2c27e03e1cd3d119f23cc46
        tracker_head: 75cccc13127c3afc7630bada10da2958f3444917
        branch_point: a1beae74bf3c0cc2a71a2884861dadcdfc13c653
        since: "2026-10-15T20:15:22-07:00"
        until: "2026-10-15T20:48:16-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:15:22-07:00"
              end: "2026-10-15T20:48:16-07:00"
              began: "2026-10-15T20:15:22-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a13337-c560-79d0-b42a-1aa85da46014
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 29304344041f41d71d4276624c83aecbd3cd845f4d555e8b323a67be3f5c2b18
              started: "2026-10-13T03:05:03.782Z"
              spawned: false
              bytes: 39254711
              windows:
                - start: "2026-10-15T20:15:22-07:00"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142b5-4b20-7c72-a925-2e8814f03b16
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 605b9fb68e98ca3e80adfde29f23c5490f566dde37feb306e6b100e82a862367
              started: "2026-10-16T03:15:40.066Z"
              spawned: true
              bytes: 486038
              windows:
                - start: "2026-10-16T03:15:40.066Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142ba-7ff7-73f0-a700-e6a6692d5459
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 18e483a7e74ad6e6d5db64d627a36325f9d2e92fd98248211e4668dd64942830
              started: "2026-10-16T03:21:21.793Z"
              spawned: false
              bytes: 501547
              windows:
                - start: "2026-10-16T03:21:21.793Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142bc-409e-7c60-933e-21eb3d9716c4
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dff5bcc801e89e2ecd5b4dacb3f7c2aee8666ec5c5c3e181c36b41181b14e5d0
              started: "2026-10-16T03:23:16.239Z"
              spawned: false
              bytes: 440993
              windows:
                - start: "2026-10-16T03:23:16.239Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142bd-f045-7da0-8ca0-d29c83da51f8
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 30f2b370f8a9226e6f17de12cde3e4e0503ea7a7ac0128b9e6afd1ea6f8a4870
              started: "2026-10-16T03:25:07.107Z"
              spawned: false
              bytes: 398455
              windows:
                - start: "2026-10-16T03:25:07.107Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142be-5790-76d0-bbfd-3fff65e8a11d
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: dadb51424694c6051418d3f2e2b17eb241a3f9cba4504bf6785e56144d28ce1f
              started: "2026-10-16T03:25:33.773Z"
              spawned: false
              bytes: 410216
              windows:
                - start: "2026-10-16T03:25:33.773Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142c0-2214-7340-b8e4-8731cda1dae7
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: e365659e2b783bfe201bd79e2f4f891e541aa3ecac3d96d369a2cded1ed1a8e5
              started: "2026-10-16T03:27:30.632Z"
              spawned: true
              bytes: 746110
              windows:
                - start: "2026-10-16T03:27:30.632Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142c0-6d1c-72c0-9646-f924bbe4ad9b
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 3db5434e9d8bf456d7fc035f101a967e15f479e58fe80728cf30330356e5f869
              started: "2026-10-16T03:27:49.659Z"
              spawned: true
              bytes: 974483
              windows:
                - start: "2026-10-16T03:27:49.659Z"
                  end: "2026-10-15T20:48:16-07:00"
            - harness: codex
              id: 01a142c0-cce9-7652-998d-aafb5fad5e47
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 6f592de2794d52e2e2fe98c80719a2c4f1943fee53ca7b3a2d92c3a0e0cd2a8d
              started: "2026-10-16T03:28:14.202Z"
              spawned: true
              bytes: 484425
              windows:
                - start: "2026-10-16T03:28:14.202Z"
                  end: "2026-10-15T20:48:16-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.0299
      "n": 1
      timestamp: "2026-10-15T20:54:28-07:00"
      dispatched: "2026-10-15T20:51:25-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Sink staging cleanup permanently blocks writes after 5,000 retained bundles
          detail: 'cmd/internal/historyarchive/sink.go:348 counts every destination-directory entry against MaxFiles before filtering for .partial- files. A kind/month directory containing 5,000 complete bundle/label pairs plus .archive-put.lock has 10,001 entries, so every subsequent Put fails during cleanup, even with no abandoned staging files. ARCH-CONSTRAINTS and ARCH-FUNERAL: isolate staging enumeration from retained objects or provide an enforced publication/rollover policy that avoids this cliff; add a regression with more than 10,000 retained final entries and verify publication and stale-stage cleanup still work.'
          family: retained-storage-must-not-exhaust-maintenance
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: The catalog source declaration excludes numbered catalog shards
          detail: 'cmd/internal/historyarchive/sources.json:200 declares only archive/*.jsonl, while catalog.go:280 produces files such as history.jsonl.000001. Those files contain later records in the same logical catalog but do not match its source declaration. ARCH-PURPOSE: declare every catalog shard while excluding locks/staging, and add a regression checking the inventory against names produced by catalog rollover.'
          family: inventory-must-cover-produced-artifacts
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#593 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T20:54:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `retained-storage-must-not-exhaust-maintenance` (kind: defect) Sink staging cleanup permanently blocks writes after 5,000 retained bundles
  cmd/internal/historyarchive/sink.go:348 counts every destination-directory entry against MaxFiles before filtering for .partial- files. A kind/month directory containing 5,000 complete bundle/label pairs plus .archive-put.lock has 10,001 entries, so every subsequent Put fails during cleanup, even with no abandoned staging files. ARCH-CONSTRAINTS and ARCH-FUNERAL: isolate staging enumeration from retained objects or provide an enforced publication/rollover policy that avoids this cliff; add a regression with more than 10,000 retained final entries and verify publication and stale-stage cleanup still work.
- **BR-2** [Important] `inventory-must-cover-produced-artifacts` (kind: defect) The catalog source declaration excludes numbered catalog shards
  cmd/internal/historyarchive/sources.json:200 declares only archive/*.jsonl, while catalog.go:280 produces files such as history.jsonl.000001. Those files contain later records in the same logical catalog but do not match its source declaration. ARCH-PURPOSE: declare every catalog shard while excluding locks/staging, and add a regression checking the inventory against names produced by catalog rollover.

## Open findings

- **BR-1** [Critical] `retained-storage-must-not-exhaust-maintenance` (kind: defect) Sink staging cleanup permanently blocks writes after 5,000 retained bundles
- **BR-2** [Important] `inventory-must-cover-produced-artifacts` (kind: defect) The catalog source declaration excludes numbered catalog shards
