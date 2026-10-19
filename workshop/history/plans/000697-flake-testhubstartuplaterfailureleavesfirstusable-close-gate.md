---
gate: boundary-review
issue: 697
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6a5fbe0d-2ea5-4bcf-9d2e-bba29b183ffb
            repository: github.com/xianxu/pair
            issue_id: "000697"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/759bfdce717ff848
            reviewed_head: f6c15b64cd32332e81f197a16300fdabde99fbd3
            slot: pair:2
            requested_at: "2026-10-19T14:42:10.053409Z"
          phase: finished
          pid: 68726
          started_at: "2026-10-19T14:42:10.659876Z"
          finished_at: "2026-10-19T14:44:04.327197Z"
          session_id: 81164074-ff57-422e-a534-ea29187ee4ae
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: f6c15b64cd32332e81f197a16300fdabde99fbd3
        tracker_head: b86fe898da4c4cd1bd6e6c89a7262d53976656e0
        branch_point: 8c2b04a6c81bb4aae43df070110866ebdd8a38e5
        since: "2026-10-19T06:54:40-07:00"
        until: "2026-10-19T07:06:30-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T06:54:40-07:00"
              end: "2026-10-19T07:06:30-07:00"
              began: "2026-10-19T06:54:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 78b72da07c9ff97b04d89e878d672ce88016a620e3c10e97f09f9c5eebc07ec0
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 45261495
              windows:
                - start: "2026-10-19T06:54:40-07:00"
                  end: "2026-10-19T07:06:30-07:00"
      "n": 1
      timestamp: "2026-10-19T07:44:06-07:00"
      dispatched: "2026-10-19T07:42:04-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: Warm-first dispatcher wrapper sends each call to r.calls twice
          detail: The wrapper at hubs_startup_test.go:211-223 wraps r.console.Ops(), which is already the rig wrapper at lines 125-133 that sends to r.calls, so it should just return actual(call), as line 247 does. This is the only instance in the window.
          family: test-dispatcher-wrap-reuse
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: testwait.Until (t.Fatalf) is called from the console operation goroutine
          detail: At hubs_startup_test.go:214 the barrier runs inside the dispatcher, off the test goroutine, so a timeout calls FailNow from the wrong goroutine. Use non-fatal testwait.Wait or a channel barrier there. This is the only instance in the window.
          family: fatal-off-test-goroutine
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: f6c15b64cd32332e81f197a16300fdabde99fbd3
      blocked: false
---

# Gate ledger — pair#697 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T07:44:06-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `test-dispatcher-wrap-reuse` (kind: defect) Warm-first dispatcher wrapper sends each call to r.calls twice
  The wrapper at hubs_startup_test.go:211-223 wraps r.console.Ops(), which is already the rig wrapper at lines 125-133 that sends to r.calls, so it should just return actual(call), as line 247 does. This is the only instance in the window.
- **BR-2** [Minor] `fatal-off-test-goroutine` (kind: defect) testwait.Until (t.Fatalf) is called from the console operation goroutine
  At hubs_startup_test.go:214 the barrier runs inside the dispatcher, off the test goroutine, so a timeout calls FailNow from the wrong goroutine. Use non-fatal testwait.Wait or a channel barrier there. This is the only instance in the window.

## Open findings

- **BR-1** [Minor] `test-dispatcher-wrap-reuse` (kind: defect) Warm-first dispatcher wrapper sends each call to r.calls twice
- **BR-2** [Minor] `fatal-off-test-goroutine` (kind: defect) testwait.Until (t.Fatalf) is called from the console operation goroutine
