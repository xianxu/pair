---
gate: plan-quality
issue: 559
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-14T16:45:08-07:00"
      agent: codex
      blocked: false
    - "n": 2
      timestamp: "2026-10-14T16:47:24-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Name the cache functions that will be unit-tested
          detail: The risky-function table names conceptual seams rather than concrete functions for session-name loading, activity reuse and cache publication. Name the proposed functions/methods, including the pure reuse or transition functions (ARCH-PURE, ARCH-ORDER), and retain one concise adversarial-input and mechanical-guard strategy per function; the named production integration tests do not replace this unit-test contract.
          family: named-unit-test-contract
          round: 2
      blocked: true
content_hash: 4bf6d78ab841450499da7be1daa295ec0cf9ec17b32ef43a129ad40e2b148b4d
---

# Gate ledger — pair#559 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T16:45:08-07:00 (codex) — passed

## Round 2 — 2026-10-14T16:47:24-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `named-unit-test-contract` Name the cache functions that will be unit-tested
  The risky-function table names conceptual seams rather than concrete functions for session-name loading, activity reuse and cache publication. Name the proposed functions/methods, including the pure reuse or transition functions (ARCH-PURE, ARCH-ORDER), and retain one concise adversarial-input and mechanical-guard strategy per function; the named production integration tests do not replace this unit-test contract.

## Open findings

- **PQ-1** [Important] `named-unit-test-contract` Name the cache functions that will be unit-tested
