---
gate: boundary-review
issue: 755
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8a325c2d-ac07-4f9e-8efb-c570bad9fb2d
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/da1e6dd27e86a5d5
            reviewed_head: 5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d
            slot: pair:4
            requested_at: "2026-10-21T06:18:18.297399Z"
          phase: finished
          pid: 75745
          started_at: "2026-10-21T06:18:18.43672Z"
          finished_at: "2026-10-21T06:23:05.697153Z"
          session_id: 01a15d1c-4f3d-76d3-a0ec-7b5763f68316
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d
        tracker_head: 1f186b33f779b6063716a9ab3401de9226e23af9
        branch_point: 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92
        since: "2026-10-20T22:40:54-07:00"
        until: "2026-10-20T23:17:51-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:54-07:00"
              end: "2026-10-20T23:17:51-07:00"
              began: "2026-10-20T22:40:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: c9fcc62e2f3fe43991463769530d64402867cc5510e7e98a97ba50f5d77a4d76
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 15600169
              windows:
                - start: "2026-10-20T22:40:54-07:00"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15cfa-72c5-7572-8b02-f2056ee548f6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5401f0f3bd86c869cb32d5df44c87f0fd8258346cda55aaceb20581356217caf
              started: "2026-10-21T05:41:19.814Z"
              spawned: true
              bytes: 652919
              windows:
                - start: "2026-10-21T05:41:19.814Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6e1e1b3acdc4e7f6c4b3ee9921c792a7dbc8ab590b6fc7513e8a02c4152ccab9
              started: "2026-10-21T05:55:18.212Z"
              spawned: false
              bytes: 545233
              windows:
                - start: "2026-10-21T05:55:18.212Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d09-e16e-7973-869b-447feb0de7d4
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 7666bc2cfa2b5266548b5bf7684a18adcdc8aa89a2cb40d27161f6eb7b38336f
              started: "2026-10-21T05:58:11.397Z"
              spawned: false
              bytes: 505167
              windows:
                - start: "2026-10-21T05:58:11.397Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0b-c808-77a0-8a74-b058604cf1e8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 775a5ebd6ce47b7647d64baca6dc6d6075af73394146897950b4df76ba0ae6fb
              started: "2026-10-21T06:00:16.2Z"
              spawned: false
              bytes: 518046
              windows:
                - start: "2026-10-21T06:00:16.2Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0c-4f3e-7c23-9eb6-c0e3bfaee8ee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 68125bf8491fa1a9237a8a23b25136ec9a96c11f2527fd3bb588f3a4b2fad9d3
              started: "2026-10-21T06:00:50.659Z"
              spawned: false
              bytes: 447230
              windows:
                - start: "2026-10-21T06:00:50.659Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0d-9ec0-70d1-a196-c8d913b2f3c9
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: a3213c5c1610c6162b59f80e20c949d2754f33f228bb0a53adea5038abd561b8
              started: "2026-10-21T06:02:16.226Z"
              spawned: true
              bytes: 399198
              windows:
                - start: "2026-10-21T06:02:16.226Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0e-032a-7401-bb22-542ec2559bee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 3d980590c5f4ba3ff9eceab50c9eb0f1ce8e3ce4a2ab2bc2ac0e58587b65bfcc
              started: "2026-10-21T06:02:41.947Z"
              spawned: true
              bytes: 396746
              windows:
                - start: "2026-10-21T06:02:41.947Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d12-f90c-7480-af77-512cb85c5c24
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: b3af1b23f47d568ec119d8edf4e2a83b475d100a1b6e3d6a8cf9ac9c1b41ac8c
              started: "2026-10-21T06:08:07.079Z"
              spawned: true
              bytes: 545002
              windows:
                - start: "2026-10-21T06:08:07.079Z"
                  end: "2026-10-20T23:17:51-07:00"
      "n": 1
      timestamp: "2026-10-20T23:23:07-07:00"
      dispatched: "2026-10-20T23:18:15-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Ancestor symlinks bypass the private-root HTTP boundary
          detail: 'cmd/internal/mockserve/project.go:90-97 checks only the final path component with Lstat; os.Root.Open follows ancestor symlinks whose targets remain inside the project root. After starting a valid project, creating mock-1/private -> .. and requesting /mock-1/private/design.md returned HTTP 200 with the private root document before the next scan rejected the symlink. ARCH-SECURE: enforce race-safe, no-follow path resolution at the read boundary rather than relying on an earlier scan. Sweep document, asset and scanner reads, and add a deterministic post-initial-scan ancestor-replacement regression that fails without the fix.'
          family: filesystem-route-isolation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#755 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T23:23:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `filesystem-route-isolation` (kind: defect) Ancestor symlinks bypass the private-root HTTP boundary
  cmd/internal/mockserve/project.go:90-97 checks only the final path component with Lstat; os.Root.Open follows ancestor symlinks whose targets remain inside the project root. After starting a valid project, creating mock-1/private -> .. and requesting /mock-1/private/design.md returned HTTP 200 with the private root document before the next scan rejected the symlink. ARCH-SECURE: enforce race-safe, no-follow path resolution at the read boundary rather than relying on an earlier scan. Sweep document, asset and scanner reads, and add a deterministic post-initial-scan ancestor-replacement regression that fails without the fix.

## Open findings

- **BR-1** [Critical] `filesystem-route-isolation` (kind: defect) Ancestor symlinks bypass the private-root HTTP boundary
