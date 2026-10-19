---
gate: plan-quality
issue: 700
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 59c11618-2c87-49bb-81cb-9f6c67a1cbb9
            repository: github.com/xianxu/pair
            issue_id: "000700"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot7/pair
            reviewed_head: 1cabc48751a883fcf2124ec658e5bc360644a2f0
            slot: pair:7
            requested_at: "2026-10-19T13:19:36.879302Z"
          phase: finished
          pid: 51303
          started_at: "2026-10-19T13:19:36.977185Z"
          finished_at: "2026-10-19T13:20:34.668941Z"
          session_id: 01a15451-4f02-7f03-a9b5-bd0314d11eb9
          outcome: returned
      "n": 1
      timestamp: "2026-10-19T06:20:34-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Define ownership and routing for scheduler-generated attention items
          detail: 'ARCH-PURPOSE / ARCH-ORDER: The global scheduler has no authenticated hub caller, while existing notify items obtain their Thread from that caller and selection navigates to that thread. Name the owner/address resolution rule, behavior with absent or multiple hubs and owner replacement, and how an unavailable owner affects admission and the promised reachable warning.'
          family: notification-owner-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace test-case inventories with named risky functions and strategies
          detail: Plan tasks 1–3 enumerate test cases without naming the new functions to unit-test. Name the freshness predicate, daily-admission transition, threshold parser and report encoder/decoder, then give one adversarial-input class and mechanical guard per risky function; use reproducible event-sequence testing for admission and malformed-input fuzzing for decoding, retaining a concise integration strategy through the production seams.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#700 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T06:20:34-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `notification-owner-contract` Define ownership and routing for scheduler-generated attention items
  ARCH-PURPOSE / ARCH-ORDER: The global scheduler has no authenticated hub caller, while existing notify items obtain their Thread from that caller and selection navigates to that thread. Name the owner/address resolution rule, behavior with absent or multiple hubs and owner replacement, and how an unavailable owner affects admission and the promised reachable warning.
- **PQ-2** [Important] `function-level-test-strategy` Replace test-case inventories with named risky functions and strategies
  Plan tasks 1–3 enumerate test cases without naming the new functions to unit-test. Name the freshness predicate, daily-admission transition, threshold parser and report encoder/decoder, then give one adversarial-input class and mechanical guard per risky function; use reproducible event-sequence testing for admission and malformed-input fuzzing for decoding, retaining a concise integration strategy through the production seams.

## Open findings

- **PQ-1** [Important] `notification-owner-contract` Define ownership and routing for scheduler-generated attention items
- **PQ-2** [Important] `function-level-test-strategy` Replace test-case inventories with named risky functions and strategies
