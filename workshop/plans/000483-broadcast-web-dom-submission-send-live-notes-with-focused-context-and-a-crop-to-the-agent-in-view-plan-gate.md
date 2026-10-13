---
gate: plan-quality
issue: 483
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T20:23:57-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify how the production delivery state crosses the Go/Lua boundary
          detail: 'ARCH-ORDER / ARCH-PURE: The plan says Go Delivery owns transitions, Lua owns draft submission state, and delivery metadata persists in the manifest, but prepare/status/ack does not explain how Lua''s written, indeterminate and dispatched outcomes reach that authority. Name the authoritative transition function for each side, its production invocation path, and the persistence/response ordering that preserves acknowledgment-only recovery; specify conservative recovery when a delivery outcome cannot be recorded.'
          family: authoritative-transition-owner
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named functions and adversarial strategies
          detail: Implementation tasks 1–4 enumerate test cases, contrary to this gate's test-plan contract, while most risky functions remain unnamed. Compress those inventories into one strategy line per risky function, identifying the adversarial input or event-sequence class and mechanical guard; include the actual production transition functions, manifest/PNG validators, capture barrier and guarded acknowledgment.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#483 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T20:23:57-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `authoritative-transition-owner` Specify how the production delivery state crosses the Go/Lua boundary
  ARCH-ORDER / ARCH-PURE: The plan says Go Delivery owns transitions, Lua owns draft submission state, and delivery metadata persists in the manifest, but prepare/status/ack does not explain how Lua's written, indeterminate and dispatched outcomes reach that authority. Name the authoritative transition function for each side, its production invocation path, and the persistence/response ordering that preserves acknowledgment-only recovery; specify conservative recovery when a delivery outcome cannot be recorded.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies
  Implementation tasks 1–4 enumerate test cases, contrary to this gate's test-plan contract, while most risky functions remain unnamed. Compress those inventories into one strategy line per risky function, identifying the adversarial input or event-sequence class and mechanical guard; include the actual production transition functions, manifest/PNG validators, capture barrier and guarded acknowledgment.

## Open findings

- **PQ-1** [Important] `authoritative-transition-owner` Specify how the production delivery state crosses the Go/Lua boundary
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies
