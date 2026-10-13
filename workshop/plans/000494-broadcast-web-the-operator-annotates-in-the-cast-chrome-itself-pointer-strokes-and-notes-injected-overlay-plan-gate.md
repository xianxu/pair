---
gate: plan-quality
issue: 494
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T20:19:59-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define authoritative settlement for queued operator writes.
          detail: 'ARCH-ORDER: Existing Note/Emit success means enqueueing, and the recorder can later drop the record without returning an outcome. Name the completion seam and pending/accepted/refused state owner, including reservation cleanup and bounded reconciliation, before promising idempotent accepted results.'
          family: asynchronous-write-outcome-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Do not restore authority controls while remote input has an unconfirmed outcome.
          detail: 'ARCH-ORDER, ARCH-SECURE: inputCall can return while its CDP call continues, and a CDP timeout does not cancel browser execution. Specify the completion evidence and bounded reconciliation or escalation that keep controls protected from late input; a token deadline longer than the call timeout is insufficient.'
          family: suppression-follows-effect-lifetime
          round: 1
        - id: PQ-3
          severity: Important
          title: Compress test-case inventories into named-function strategies.
          detail: Tasks 1–5 enumerate test cases while leaving several tested functions unnamed. Replace those inventories with function names and one strategy line per risky function identifying the adversarial input or ordering class and mechanical guard.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#494 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T20:19:59-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `asynchronous-write-outcome-contract` Define authoritative settlement for queued operator writes.
  ARCH-ORDER: Existing Note/Emit success means enqueueing, and the recorder can later drop the record without returning an outcome. Name the completion seam and pending/accepted/refused state owner, including reservation cleanup and bounded reconciliation, before promising idempotent accepted results.
- **PQ-2** [Important] `suppression-follows-effect-lifetime` Do not restore authority controls while remote input has an unconfirmed outcome.
  ARCH-ORDER, ARCH-SECURE: inputCall can return while its CDP call continues, and a CDP timeout does not cancel browser execution. Specify the completion evidence and bounded reconciliation or escalation that keep controls protected from late input; a token deadline longer than the call timeout is insufficient.
- **PQ-3** [Important] `function-level-test-strategy` Compress test-case inventories into named-function strategies.
  Tasks 1–5 enumerate test cases while leaving several tested functions unnamed. Replace those inventories with function names and one strategy line per risky function identifying the adversarial input or ordering class and mechanical guard.

## Open findings

- **PQ-1** [Important] `asynchronous-write-outcome-contract` Define authoritative settlement for queued operator writes.
- **PQ-2** [Important] `suppression-follows-effect-lifetime` Do not restore authority controls while remote input has an unconfirmed outcome.
- **PQ-3** [Important] `function-level-test-strategy` Compress test-case inventories into named-function strategies.
