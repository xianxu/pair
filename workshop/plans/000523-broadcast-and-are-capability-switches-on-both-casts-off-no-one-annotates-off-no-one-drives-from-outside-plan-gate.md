---
gate: plan-quality
issue: 523
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T23:08:04-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define release recovery before reusing the existing Releaser.
          detail: 'ARCH-ORDER: controller_mem.go:162 discards asynchronous release outcomes, while tab.go:89 clears held-input evidence before dispatch; retry can therefore falsely succeed after a timeout. Specify retained partial/uncertain progress, completion evidence, attempt fencing and worker ownership, and extend the pure capability transition model to cover the revision''s failure/retry outcomes.'
          family: uncertain-effect-reconciliation
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace enumerated test cases with named function-level strategies.
          detail: Tasks 1–5 and the recovery revision enumerate test cases rather than providing the required compact strategy per risky function. Name each tested production function and give one line identifying its adversarial input or event class and mechanical guard; preserve acceptance criteria without duplicating executable test cases in prose.
          family: function-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#523 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T23:08:04-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `uncertain-effect-reconciliation` Define release recovery before reusing the existing Releaser.
  ARCH-ORDER: controller_mem.go:162 discards asynchronous release outcomes, while tab.go:89 clears held-input evidence before dispatch; retry can therefore falsely succeed after a timeout. Specify retained partial/uncertain progress, completion evidence, attempt fencing and worker ownership, and extend the pure capability transition model to cover the revision's failure/retry outcomes.
- **PQ-2** [Important] `function-test-strategy` Replace enumerated test cases with named function-level strategies.
  Tasks 1–5 and the recovery revision enumerate test cases rather than providing the required compact strategy per risky function. Name each tested production function and give one line identifying its adversarial input or event class and mechanical guard; preserve acceptance criteria without duplicating executable test cases in prose.

## Open findings

- **PQ-1** [Important] `uncertain-effect-reconciliation` Define release recovery before reusing the existing Releaser.
- **PQ-2** [Important] `function-test-strategy` Replace enumerated test cases with named function-level strategies.
