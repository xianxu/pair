---
gate: plan-quality
issue: 569
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 29d175a5-8eda-4f5f-a2b9-922fe14bbcc8
            repository: github.com/xianxu/pair
            issue_id: "000569"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair
            reviewed_head: ea4f7a1ed323faa3f73c63901605eb15ebdc6a4e
            slot: pair:5
            requested_at: "2026-10-15T23:20:43.450262Z"
          phase: finished
          pid: 19647
          started_at: "2026-10-15T23:20:43.556631Z"
          finished_at: "2026-10-15T23:21:31.369769Z"
          session_id: 01a141de-3408-77b1-b13f-6e7b5245bb71
          outcome: returned
      "n": 1
      timestamp: "2026-10-15T16:21:32-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify how terminal residue reaches bounded escalation after polling stops
          detail: 'ARCH-ORDER / ARCH-CONSTRAINTS: the plan promises bounded reporting, prompt later refusal and no idle background polling, but dispatchPeer returns for terminal receipts before automaticInputAvailable runs (peer_delivery.go:410), and peerDeliveryPoll stops for terminal receipts (peer_delivery.go:147-153). Name the event that evaluates residue and emits the report, when its bound starts, and how positive-empty recovery is observed without another delivery; choose terminal-event escalation or an explicitly bounded recovery observer and state its lifetime.'
          family: terminal-recovery-event-source
          round: 1
        - id: PQ-2
          severity: Important
          title: Compress test inventories into named functions and adversarial strategies
          detail: Plan lines 69 and 74 enumerate test cases rather than specifying one strategy per risky function. Name the new Qoder evidence function alongside AdvancePeerDelivery, advanceAutomaticInput and ParseAutomaticInputStall, then give each an adversarial input class and mechanical guard, including reproducible event ordering for reducers and malformed terminal evidence for the projection. Keep wire-count and recovery invariants; move individual cases into executable tests.
          family: function-level-adversarial-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#569 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T16:21:32-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `terminal-recovery-event-source` Specify how terminal residue reaches bounded escalation after polling stops
  ARCH-ORDER / ARCH-CONSTRAINTS: the plan promises bounded reporting, prompt later refusal and no idle background polling, but dispatchPeer returns for terminal receipts before automaticInputAvailable runs (peer_delivery.go:410), and peerDeliveryPoll stops for terminal receipts (peer_delivery.go:147-153). Name the event that evaluates residue and emits the report, when its bound starts, and how positive-empty recovery is observed without another delivery; choose terminal-event escalation or an explicitly bounded recovery observer and state its lifetime.
- **PQ-2** [Important] `function-level-adversarial-test-strategy` Compress test inventories into named functions and adversarial strategies
  Plan lines 69 and 74 enumerate test cases rather than specifying one strategy per risky function. Name the new Qoder evidence function alongside AdvancePeerDelivery, advanceAutomaticInput and ParseAutomaticInputStall, then give each an adversarial input class and mechanical guard, including reproducible event ordering for reducers and malformed terminal evidence for the projection. Keep wire-count and recovery invariants; move individual cases into executable tests.

## Open findings

- **PQ-1** [Important] `terminal-recovery-event-source` Specify how terminal residue reaches bounded escalation after polling stops
- **PQ-2** [Important] `function-level-adversarial-test-strategy` Compress test inventories into named functions and adversarial strategies
