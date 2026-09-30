---
gate: plan-quality
issue: 353
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-30T16:57:13-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define eligibility observation freshness and reservation ordering.
          detail: 'ARCH-ORDER, ARCH-CONSTRAINTS: Name the producers of raw agent output, draft typing, and verified branch observations; specify their freshness bounds and how delayed, failed, or superseded observations affect reservation. Define the admission-time revalidation or version check within the two-second budget, with controlled-order tests preventing stale eligibility from admitting a family send.'
          family: admission-observation-freshness
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with function-level strategies.
          detail: Tasks 1–3 enumerate cases instead of providing the required strategy per risky function. Compress them into named strategies for Advance, ResolveRecipient, ValidateRequest, AdvancePeerDelivery, peerComposerState, and ParseCLI, identifying adversarial input classes and mechanical guards such as fuzzing, generated event sequences with independent invariants, and controlled terminal observations.
          family: function-level-test-strategy
          round: 1
        - id: PQ-3
          severity: Minor
          title: Name when supported harness behavior is requalified.
          detail: 'ARCH-MOCK: Captured fixtures and an initial live smoke establish initial qualification, but the plan names no recurring or version-triggered conformance check. State when Claude/Codex composer and paste behavior must be rechecked and how failed qualification affects delivery support.'
          family: external-conformance-cadence
          round: 1
      blocked: true
---

# Gate ledger — pair#353 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T16:57:13-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `admission-observation-freshness` Define eligibility observation freshness and reservation ordering.
  ARCH-ORDER, ARCH-CONSTRAINTS: Name the producers of raw agent output, draft typing, and verified branch observations; specify their freshness bounds and how delayed, failed, or superseded observations affect reservation. Define the admission-time revalidation or version check within the two-second budget, with controlled-order tests preventing stale eligibility from admitting a family send.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with function-level strategies.
  Tasks 1–3 enumerate cases instead of providing the required strategy per risky function. Compress them into named strategies for Advance, ResolveRecipient, ValidateRequest, AdvancePeerDelivery, peerComposerState, and ParseCLI, identifying adversarial input classes and mechanical guards such as fuzzing, generated event sequences with independent invariants, and controlled terminal observations.
- **PQ-3** [Minor] `external-conformance-cadence` Name when supported harness behavior is requalified.
  ARCH-MOCK: Captured fixtures and an initial live smoke establish initial qualification, but the plan names no recurring or version-triggered conformance check. State when Claude/Codex composer and paste behavior must be rechecked and how failed qualification affects delivery support.

## Open findings

- **PQ-1** [Important] `admission-observation-freshness` Define eligibility observation freshness and reservation ordering.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with function-level strategies.
- **PQ-3** [Minor] `external-conformance-cadence` Name when supported harness behavior is requalified.
