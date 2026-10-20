---
gate: plan-quality
issue: 706
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 1be42e47-6608-4a68-8c9a-6ed436ea9b12
            repository: github.com/xianxu/pair
            issue_id: "000706"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: e0f4f3b381192cc5c8887288cd7ac3e6d2a562c9
            slot: pair:6
            requested_at: "2026-10-20T03:51:46.663075Z"
          phase: finished
          pid: 71431
          started_at: "2026-10-20T03:51:46.886442Z"
          finished_at: "2026-10-20T03:52:39.548463Z"
          session_id: 01a1576f-cd4b-72f2-95dc-3dced4835dcb
          outcome: returned
      "n": 1
      timestamp: "2026-10-19T20:52:39-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Replace test-case inventories with named risky functions and mechanical guards
          detail: 'The M1/M2 test rows enumerate cases but do not name the layout parsing/validation functions or folder transition function and give each an adversarial input class plus mechanical guard. Compress these rows into function-specific strategies: malformed-input fuzzing and round-trip invariants for layout/receipt codecs, generated event sequences with independently stated invariants for the production folder transition function (ARCH-PURE, ARCH-ORDER), and named restore/index functions with their bounds and atomicity guards.'
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#706 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T20:52:39-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named risky functions and mechanical guards
  The M1/M2 test rows enumerate cases but do not name the layout parsing/validation functions or folder transition function and give each an adversarial input class plus mechanical guard. Compress these rows into function-specific strategies: malformed-input fuzzing and round-trip invariants for layout/receipt codecs, generated event sequences with independently stated invariants for the production folder transition function (ARCH-PURE, ARCH-ORDER), and named restore/index functions with their bounds and atomicity guards.

## Open findings

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named risky functions and mechanical guards
