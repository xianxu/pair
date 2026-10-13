---
gate: plan-quality
issue: 517
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T20:17:05-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Bind handoff callbacks to the exact switch request, not only its incarnation
          detail: 'ARCH-ORDER/ARCH-SECURE: the documented self callback identifies the source incarnation but not the pending request it answers. Specify how the request identifier reaches the outgoing agent and returns in its callback, how standalone self admission differs from a callback, and how expired or cancelled request identifiers are rejected even when a newer request targets the same incarnation.'
          family: asynchronous-request-correlation
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named functions and adversarial strategies
          detail: M1–M3 enumerate test cases but omit production function names for the new transition model, descriptor validation, content validation, previous selection, and bounded replay. Name those functions and give each risky function one strategy line describing the adversarial input class and mechanical guard; compress the case inventories as required by this gate.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#517 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T20:17:05-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `asynchronous-request-correlation` Bind handoff callbacks to the exact switch request, not only its incarnation
  ARCH-ORDER/ARCH-SECURE: the documented self callback identifies the source incarnation but not the pending request it answers. Specify how the request identifier reaches the outgoing agent and returns in its callback, how standalone self admission differs from a callback, and how expired or cancelled request identifiers are rejected even when a newer request targets the same incarnation.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies
  M1–M3 enumerate test cases but omit production function names for the new transition model, descriptor validation, content validation, previous selection, and bounded replay. Name those functions and give each risky function one strategy line describing the adversarial input class and mechanical guard; compress the case inventories as required by this gate.

## Open findings

- **PQ-1** [Important] `asynchronous-request-correlation` Bind handoff callbacks to the exact switch request, not only its incarnation
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies
