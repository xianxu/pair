---
gate: plan-quality
issue: 751
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: da12f930-e264-4f89-a9ac-565c8bebe897
            repository: github.com/xianxu/pair
            issue_id: "000751"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot9/pair
            reviewed_head: 0abdd53a12fd9e7a38f434f2810de6a6bc665cef
            slot: pair:9
            requested_at: "2026-10-21T04:43:23.689696Z"
          phase: finished
          pid: 21224
          started_at: "2026-10-21T04:43:23.943889Z"
          finished_at: "2026-10-21T04:44:40.853957Z"
          session_id: 01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
          outcome: returned
      "n": 1
      timestamp: "2026-10-20T21:44:41-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Bound the additional control history retained for sketches
          detail: 'ARCH-CONSTRAINTS / ARCH-FUNERAL: retaining every ctl whenever sketch history exists introduces indefinitely growing Hub catch-up history outside the 32 MiB sketch-event budget. Specify bounded causal retention or an enforced capacity with overflow behavior, and verify replay equivalence and retention bounds across repeated control changes and segment compactions.'
          family: retained-history-must-be-bounded
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named-function strategies
          detail: Tasks 1–5 enumerate test cases instead of providing the required concise strategy per risky function. Name the codec functions, NextSketch, browser transition function, ProjectPoint, PressureWidth, CatchUp, OrderedTime and SketchBudget.Next; state each adversarial input class and mechanical guard, retaining independent invariants and counted-work assertions.
          family: function-level-adversarial-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#751 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T21:44:41-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `retained-history-must-be-bounded` Bound the additional control history retained for sketches
  ARCH-CONSTRAINTS / ARCH-FUNERAL: retaining every ctl whenever sketch history exists introduces indefinitely growing Hub catch-up history outside the 32 MiB sketch-event budget. Specify bounded causal retention or an enforced capacity with overflow behavior, and verify replay equivalence and retention bounds across repeated control changes and segment compactions.
- **PQ-2** [Important] `function-level-adversarial-test-strategy` Replace prose test-case inventories with named-function strategies
  Tasks 1–5 enumerate test cases instead of providing the required concise strategy per risky function. Name the codec functions, NextSketch, browser transition function, ProjectPoint, PressureWidth, CatchUp, OrderedTime and SketchBudget.Next; state each adversarial input class and mechanical guard, retaining independent invariants and counted-work assertions.

## Open findings

- **PQ-1** [Important] `retained-history-must-be-bounded` Bound the additional control history retained for sketches
- **PQ-2** [Important] `function-level-adversarial-test-strategy` Replace prose test-case inventories with named-function strategies
