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
    - "n": 2
      timestamp: "2026-09-23T12:22:48-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Fetch output is parsed as the immutable captured baseline; later tracking-ref changes cannot alter it (plan:164-170).
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: 'The plan describes #306''s reservation concept, but the issue metadata still lacks pair#306 as a dependency and does not define an executable token/owner/lifetime handoff spanning selection through provisioning (issue:3-4; plan:42-50).'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Tests now name functions and pair adversarial input classes with mechanical guards rather than enumerating prose cases (plan:260-310).
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Runtime limits are identified as initial policy ceilings with injected test values, and subprocess concurrency is clarified as sequential commands with bounded process ownership (plan:231-243).
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Retention names the operator as owner, gives discovery/removal paths, and states that worktrees and clones persist until explicit removal (plan:355-363).
          round: 2
        - id: PQ-6
          disposition: addressed
          note: The plan has an explicit non-goals section assigning thread admission, grouping, transfer, dependency repair, retry mode, and removal elsewhere (plan:365-368).
          round: 2
      findings:
        - id: PQ-7
          severity: Important
          title: Model provisioning's interrupting and concurrent event ordering explicitly
          detail: 'ARCH-ORDER is not satisfied by the host decision table alone: the plan must name the state/event transitions and policies for process death or cancellation during Git/Weave, a second request observing the same host, late compile completion after cancellation, and marker publication races, including who remains running, who cancels or queues, and how each interleaving is reproduced. The current text only says “reconcile” and “another compile is acceptable” (plan:216-229, 231-241), leaving the legal temporal behavior implicit.'
          family: external-transition-ordering
          round: 2
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

## Round 2 — 2026-09-23T12:22:48-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — addressed — Fetch output is parsed as the immutable captured baseline; later tracking-ref changes cannot alter it (plan:164-170).
- PQ-2 — not-addressed — The plan describes #306's reservation concept, but the issue metadata still lacks pair#306 as a dependency and does not define an executable token/owner/lifetime handoff spanning selection through provisioning (issue:3-4; plan:42-50).
- PQ-3 — addressed — Tests now name functions and pair adversarial input classes with mechanical guards rather than enumerating prose cases (plan:260-310).
- PQ-4 — addressed — Runtime limits are identified as initial policy ceilings with injected test values, and subprocess concurrency is clarified as sequential commands with bounded process ownership (plan:231-243).
- PQ-5 — addressed — Retention names the operator as owner, gives discovery/removal paths, and states that worktrees and clones persist until explicit removal (plan:355-363).
- PQ-6 — addressed — The plan has an explicit non-goals section assigning thread admission, grouping, transfer, dependency repair, retry mode, and removal elsewhere (plan:365-368).

### Raised

- **PQ-7** [Important] `external-transition-ordering` Model provisioning's interrupting and concurrent event ordering explicitly
  ARCH-ORDER is not satisfied by the host decision table alone: the plan must name the state/event transitions and policies for process death or cancellation during Git/Weave, a second request observing the same host, late compile completion after cancellation, and marker publication races, including who remains running, who cancels or queues, and how each interleaving is reproduced. The current text only says “reconcile” and “another compile is acceptable” (plan:216-229, 231-241), leaving the legal temporal behavior implicit.

## Open findings

- **PQ-2** [Important] `cross-issue-capability-contract` Define the #306 reservation spanning selection and provisioning
- **PQ-7** [Important] `external-transition-ordering` Model provisioning's interrupting and concurrent event ordering explicitly
