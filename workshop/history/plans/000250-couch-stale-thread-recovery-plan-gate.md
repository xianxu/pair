---
gate: plan-quality
issue: 250
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-14T13:28:44-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace prose test inventories with named risky-function strategies.
          detail: Tasks 3–5 enumerate test cases rather than supplying the required function-level strategy. Name the recovery decider and generation-admission functions, and state an adversarial input or event class plus mechanical oracle for each risky function, including checkpoint.Request.Validate and checkpoint.Advance; compress duplicated case inventories while retaining the behavioral transition table.
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Minor
          title: Bound recovery observations on the inventory path and define retry exhaustion.
          detail: 'ARCH-CONSTRAINTS: The plan adds session, process, ledger and checkpoint observations to inventory without stating their scheduling or workload budget, and “bounded revision retries” names no bound. Specify the refresh/operation path, observation reuse or fan-out limit, deadline and retry limit with their basis, and the visible result on exhaustion.'
          family: explicit-operating-envelope
          round: 1
        - id: PQ-3
          severity: Minor
          title: Name the ongoing conformance check for the reused external seams.
          detail: 'ARCH-MOCK: Stateful fixtures and one-time disposable acceptance are specified, but no live conformance cadence is named. Identify the existing recurring check or state when the process/session fake’s modeled behavior will be compared with the real dependency.'
          family: external-fake-conformance
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-14T13:31:54-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The authoritative revision supplies named risky-function strategies and mechanical oracles, superseding earlier test inventories while retaining the transition table.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Recovery adds no inventory IO; explicit operations specify observation reuse, five-second phase deadlines, eight total attempts, cancellation, and visible exhaustion.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Names the verified weekly/manual/PR/main-push conformance workflow and requires recovery coverage in its live test selection and path filters.
          round: 2
      blocked: false
content_hash: 6a6dd177bb1f6259b98f2a855bf0ffa1740100bec100957084ffb1a591cc979c
---

# Gate ledger — pair#250 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T13:28:44-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace prose test inventories with named risky-function strategies.
  Tasks 3–5 enumerate test cases rather than supplying the required function-level strategy. Name the recovery decider and generation-admission functions, and state an adversarial input or event class plus mechanical oracle for each risky function, including checkpoint.Request.Validate and checkpoint.Advance; compress duplicated case inventories while retaining the behavioral transition table.
- **PQ-2** [Minor] `explicit-operating-envelope` Bound recovery observations on the inventory path and define retry exhaustion.
  ARCH-CONSTRAINTS: The plan adds session, process, ledger and checkpoint observations to inventory without stating their scheduling or workload budget, and “bounded revision retries” names no bound. Specify the refresh/operation path, observation reuse or fan-out limit, deadline and retry limit with their basis, and the visible result on exhaustion.
- **PQ-3** [Minor] `external-fake-conformance` Name the ongoing conformance check for the reused external seams.
  ARCH-MOCK: Stateful fixtures and one-time disposable acceptance are specified, but no live conformance cadence is named. Identify the existing recurring check or state when the process/session fake’s modeled behavior will be compared with the real dependency.

## Round 2 — 2026-09-14T13:31:54-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The authoritative revision supplies named risky-function strategies and mechanical oracles, superseding earlier test inventories while retaining the transition table.
- PQ-2 — addressed — Recovery adds no inventory IO; explicit operations specify observation reuse, five-second phase deadlines, eight total attempts, cancellation, and visible exhaustion.
- PQ-3 — addressed — Names the verified weekly/manual/PR/main-push conformance workflow and requires recovery coverage in its live test selection and path filters.

## Open findings

(none — every finding has been disposed)
