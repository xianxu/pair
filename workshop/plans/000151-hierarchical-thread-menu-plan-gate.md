---
gate: plan-quality
issue: 151
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-30T20:07:24-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Compress the enumerated test and procedural-diff plan into function-level strategies
          detail: The plan repeatedly enumerates individual cases and restates the intended diff, including the projection cases, grant lifecycle cases, renderer matrix, operation matrix, and thirteen-row restoration matrix. Retain the risky function names, but replace these inventories with one adversarial-input class and mechanical guard per function; the Spec and Done-when already own the detailed behavioral contract.
          family: test-strategy-over-specification
          round: 1
        - id: PQ-2
          severity: Important
          title: ARCH-MOCK requires live conformance for the new runner cancellation contract
          detail: The plan changes Runner.StartBlocked and FakeRunner to model context cancellation across real helper creation, acknowledgement, registration, and cleanup, but only names the existing conformance suite. That suite does not compare this new behavior against the real runner. Name the real cancellation conformance test, the corresponding fake state transition, and the workflow cadence or change-trigger that will keep them compared.
          family: external-double-live-conformance
          round: 1
      blocked: true
---

# Gate ledger — pair#151 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T20:07:24-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `test-strategy-over-specification` Compress the enumerated test and procedural-diff plan into function-level strategies
  The plan repeatedly enumerates individual cases and restates the intended diff, including the projection cases, grant lifecycle cases, renderer matrix, operation matrix, and thirteen-row restoration matrix. Retain the risky function names, but replace these inventories with one adversarial-input class and mechanical guard per function; the Spec and Done-when already own the detailed behavioral contract.
- **PQ-2** [Important] `external-double-live-conformance` ARCH-MOCK requires live conformance for the new runner cancellation contract
  The plan changes Runner.StartBlocked and FakeRunner to model context cancellation across real helper creation, acknowledgement, registration, and cleanup, but only names the existing conformance suite. That suite does not compare this new behavior against the real runner. Name the real cancellation conformance test, the corresponding fake state transition, and the workflow cadence or change-trigger that will keep them compared.

## Open findings

- **PQ-1** [Important] `test-strategy-over-specification` Compress the enumerated test and procedural-diff plan into function-level strategies
- **PQ-2** [Important] `external-double-live-conformance` ARCH-MOCK requires live conformance for the new runner cancellation contract
