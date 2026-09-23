---
gate: plan-quality
issue: 305
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-23T12:20:49-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Make remote baseline capture immune to tracking-ref races
          detail: The plan fetches into the usual tracking ref and reads it afterward while external Git commands remain outside the lock, so the recorded SHA can differ from the fetch result. Specify an immutable capture mechanism and test an intervening external fetch.
          family: baseline-capture-is-atomic
          round: 1
        - id: PQ-2
          severity: Important
          title: 'Define the #306 reservation spanning selection and provisioning'
          detail: 'The plan delegates reservation to #306 but does not define its token, owner, lifetime, or atomic handoff across number selection and provisioning. Make the contract executable and declare pair#306 as a dependency.'
          family: cross-issue-capability-contract
          round: 1
        - id: PQ-3
          severity: Important
          title: Replace enumerated test cases with function-level adversarial strategies
          detail: Tasks 1–3 enumerate prose cases instead of naming risky functions and giving one strategy line per function. Compress the test plan to named functions, adversarial input classes, and mechanical guards.
          family: test-strategy-function-level
          round: 1
        - id: PQ-4
          severity: Minor
          title: Give the runtime budgets a basis and clarify subprocess concurrency
          detail: The plan states timeout and output limits as defaults without basis, and “one synchronous subprocess” is ambiguous for a flow that invokes several external commands.
          family: operating-envelope-basis
          round: 1
        - id: PQ-5
          severity: Minor
          title: Define lifecycle ownership for retained worktrees and clones
          detail: Persistent slot directories and dependency clones have no removal owner, discovery path, or stated consequence of unbounded retention.
          family: retained-artifact-lifecycle
          round: 1
        - id: PQ-6
          severity: Minor
          title: Add an explicit non-goals section
          detail: 'The plan relies on scattered exclusions rather than stating the deliberate non-goals and why they belong to #306, Ariadne, Weave, or a later workflow.'
          family: explicit-non-goals
          round: 1
      blocked: true
---

# Gate ledger — pair#305 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T12:20:49-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `baseline-capture-is-atomic` Make remote baseline capture immune to tracking-ref races
  The plan fetches into the usual tracking ref and reads it afterward while external Git commands remain outside the lock, so the recorded SHA can differ from the fetch result. Specify an immutable capture mechanism and test an intervening external fetch.
- **PQ-2** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
  The plan delegates reservation to #306 but does not define its token, owner, lifetime, or atomic handoff across number selection and provisioning. Make the contract executable and declare pair#306 as a dependency.
- **PQ-3** [Important] `test-strategy-function-level` Replace enumerated test cases with function-level adversarial strategies
  Tasks 1–3 enumerate prose cases instead of naming risky functions and giving one strategy line per function. Compress the test plan to named functions, adversarial input classes, and mechanical guards.
- **PQ-4** [Minor] `operating-envelope-basis` Give the runtime budgets a basis and clarify subprocess concurrency
  The plan states timeout and output limits as defaults without basis, and “one synchronous subprocess” is ambiguous for a flow that invokes several external commands.
- **PQ-5** [Minor] `retained-artifact-lifecycle` Define lifecycle ownership for retained worktrees and clones
  Persistent slot directories and dependency clones have no removal owner, discovery path, or stated consequence of unbounded retention.
- **PQ-6** [Minor] `explicit-non-goals` Add an explicit non-goals section
  The plan relies on scattered exclusions rather than stating the deliberate non-goals and why they belong to #306, Ariadne, Weave, or a later workflow.

## Open findings

- **PQ-1** [Important] `baseline-capture-is-atomic` Make remote baseline capture immune to tracking-ref races
- **PQ-2** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
- **PQ-3** [Important] `test-strategy-function-level` Replace enumerated test cases with function-level adversarial strategies
- **PQ-4** [Minor] `operating-envelope-basis` Give the runtime budgets a basis and clarify subprocess concurrency
- **PQ-5** [Minor] `retained-artifact-lifecycle` Define lifecycle ownership for retained worktrees and clones
- **PQ-6** [Minor] `explicit-non-goals` Add an explicit non-goals section
