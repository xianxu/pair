---
gate: plan-quality
issue: 546
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-14T00:05:05-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define move idempotency separately for runtime and persisted ownership
          detail: 'ARCH-ORDER: The unqualified duplicate/no-op rule can skip persistence after an ephemeral move. Define the no-op predicate for each mode: --persist must reconcile durable claims even when runtime ownership already matches, and retries must finish incomplete persistence before reporting satisfaction.'
          family: persistence-aware-idempotency
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test inventories with named function strategies
          detail: Name the unit-tested parser, resolver, claim-edit planner, move transition and presentation functions, then give each risky function one adversarial input or event class and mechanical guard. Compress the enumerated prose cases into those strategies; retain the integration seams and verification commands.
          family: function-test-strategy-contract
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T00:08:13-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Persistent satisfaction requires freshly read durable claims without runtime overrides; retries reconcile incomplete persistence even when runtime ownership matches.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The replacement table names risky functions and supplies adversarial strategies and mechanical guards, retaining integration seams and verification commands.
          round: 2
      blocked: false
content_hash: 2ba332ad1b9599ec7cf856ffbacef31a6ea61061e25abc8cd2633958be81a59b
---

# Gate ledger — pair#546 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T00:05:05-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `persistence-aware-idempotency` Define move idempotency separately for runtime and persisted ownership
  ARCH-ORDER: The unqualified duplicate/no-op rule can skip persistence after an ephemeral move. Define the no-op predicate for each mode: --persist must reconcile durable claims even when runtime ownership already matches, and retries must finish incomplete persistence before reporting satisfaction.
- **PQ-2** [Important] `function-test-strategy-contract` Replace prose test inventories with named function strategies
  Name the unit-tested parser, resolver, claim-edit planner, move transition and presentation functions, then give each risky function one adversarial input or event class and mechanical guard. Compress the enumerated prose cases into those strategies; retain the integration seams and verification commands.

## Round 2 — 2026-10-14T00:08:13-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Persistent satisfaction requires freshly read durable claims without runtime overrides; retries reconcile incomplete persistence even when runtime ownership matches.
- PQ-2 — addressed — The replacement table names risky functions and supplies adversarial strategies and mechanical guards, retaining integration seams and verification commands.

## Open findings

(none — every finding has been disposed)
