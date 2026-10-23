---
gate: plan-quality
issue: 812
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: e164b234-10ec-4a4a-b27f-75efef22cc95
            repository: github.com/xianxu/pair
            issue_id: "000812"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot1/pair
            reviewed_head: 1797bd4c85248e15fc15f3b5a570e5be0d233aa0
            slot: pair:1
            requested_at: "2026-10-23T12:07:28.89234Z"
          phase: finished
          pid: 13840
          started_at: "2026-10-23T12:07:29.019136Z"
          finished_at: "2026-10-23T12:08:50.443779Z"
          session_id: 01a168a8-b55a-7a73-8097-1ac736feaf5d
          outcome: returned
      "n": 1
      timestamp: "2026-10-23T05:09:07-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Replace test-case inventories with named risky functions and adversarial strategies.
          detail: Tasks 1–2 enumerate rendering and event cases while leaving the new projection and publisher unnamed and without individual test strategies. Name the functions to be unit-tested and give one adversarial-input/mechanical-guard line per risky function, including a controllable ordering seam for concurrent publication and selection (ARCH-PURE, ARCH-ORDER); move case enumeration into executable tests.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#812 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T05:09:07-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named risky functions and adversarial strategies.
  Tasks 1–2 enumerate rendering and event cases while leaving the new projection and publisher unnamed and without individual test strategies. Name the functions to be unit-tested and give one adversarial-input/mechanical-guard line per risky function, including a controllable ordering seam for concurrent publication and selection (ARCH-PURE, ARCH-ORDER); move case enumeration into executable tests.

## Open findings

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named risky functions and adversarial strategies.
