---
gate: boundary-review
issue: 585
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c6f1c062-2b0f-44ea-951c-6476d1f67510
            repository: github.com/xianxu/pair
            issue_id: "000585"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/b2a4a29dcb3059b2
            reviewed_head: c25736eab2950b3cae5906268db0550da7b02d94
            slot: pair:3
            requested_at: "2026-10-16T01:52:06.423255Z"
          phase: finished
          pid: 80736
          started_at: "2026-10-16T01:52:06.599342Z"
          finished_at: "2026-10-16T01:54:08.887581Z"
          session_id: 01a14268-cd49-7e42-bc6b-2be787f15725
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c25736eab2950b3cae5906268db0550da7b02d94
        tracker_head: c91a739979ba6239958bf2a9411c02c45b54a138
        branch_point: 4c6fcc8d437ce52c708b0a6bdef09ce42b86917b
        since: "2026-10-15T18:08:06-07:00"
        until: "2026-10-15T18:51:03-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T18:08:06-07:00"
              end: "2026-10-15T18:51:03-07:00"
              began: "2026-10-15T18:08:06-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a13b1c-d331-7843-9659-349166408f6a
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 95ce8eba4af0390d8fcd735b57adb8a10785e65480e4dec85943c7be09acb026
              started: "2026-10-14T15:51:55.914Z"
              spawned: false
              bytes: 26777797
              windows:
                - start: "2026-10-15T18:08:06-07:00"
                  end: "2026-10-15T18:51:03-07:00"
            - harness: codex
              id: 01a14241-4a69-70f3-8406-0949614e85d4
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: 7f54e64b155d35bd21f4b45b3606bf84e29164b13b8bc4dfc2bd15f2be9fb668
              started: "2026-10-16T01:08:57.751Z"
              spawned: true
              bytes: 499158
              windows:
                - start: "2026-10-16T01:08:57.751Z"
                  end: "2026-10-15T18:51:03-07:00"
            - harness: codex
              id: 01a14241-646c-77d0-baf4-3e13b3de0d06
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: dbfba7feb716b1eb1b703f700be323ab3c75cf382e2273e56560e2a4da85b776
              started: "2026-10-16T01:09:04.644Z"
              spawned: true
              bytes: 494756
              windows:
                - start: "2026-10-16T01:09:04.644Z"
                  end: "2026-10-15T18:51:03-07:00"
      "n": 1
      timestamp: "2026-10-15T18:54:10-07:00"
      dispatched: "2026-10-15T18:52:03-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Reaped command leaders can leave live descendants behind after Close
          detail: 'cmd/internal/broadcast/cloudflared.go:320-325 and :330-334 skip group termination once processExited closes. An unguarded fake that starts `sleep 300 &`, prints its failure, and exits leaves that descendant holding stderr; after the opening timeout and forced reader close, Close returns without killing the descendant because h.child is zero. The previous unconditional group SIGTERM reached it. Every instance in this window is the two production guards, the matching cleanup assumption in cloudflared_stderr_test.go:81-85, and the claim in atlas/broadcast.md:310-311. Track group lifetime independently of leader reaping, preserve safe group cleanup, and add a real descendant regression asserting that Close both releases stderr and terminates the descendant. ARCH-PURPOSE: reader release alone does not fulfill the promised process cleanup.'
          family: process-group-lifetime
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#585 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T18:54:10-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `process-group-lifetime` (kind: defect) Reaped command leaders can leave live descendants behind after Close
  cmd/internal/broadcast/cloudflared.go:320-325 and :330-334 skip group termination once processExited closes. An unguarded fake that starts `sleep 300 &`, prints its failure, and exits leaves that descendant holding stderr; after the opening timeout and forced reader close, Close returns without killing the descendant because h.child is zero. The previous unconditional group SIGTERM reached it. Every instance in this window is the two production guards, the matching cleanup assumption in cloudflared_stderr_test.go:81-85, and the claim in atlas/broadcast.md:310-311. Track group lifetime independently of leader reaping, preserve safe group cleanup, and add a real descendant regression asserting that Close both releases stderr and terminates the descendant. ARCH-PURPOSE: reader release alone does not fulfill the promised process cleanup.

## Open findings

- **BR-1** [Critical] `process-group-lifetime` (kind: defect) Reaped command leaders can leave live descendants behind after Close
