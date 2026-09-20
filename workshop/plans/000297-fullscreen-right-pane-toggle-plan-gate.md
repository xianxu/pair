---
gate: plan-quality
issue: 297
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-20T11:39:44-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Make effect-outcome transitions part of the production pure model
          detail: 'ARCH-ORDER / ARCH-PURE: FullscreenPlan currently selects only the initial operation; save, toggle, focus and clear outcomes remain prose rules for the IO executor. Name the state/event types and transition function that owns subsequent effects, including unconfirmed outcomes, and require the executor and sequence tests to use that function.'
          family: pure-transition-ownership
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose case inventories with named functions and adversarial strategies
          detail: Task 1 enumerates test cases beginning “draft, agent, same terminal, split half” without naming the new parser/decision functions or specifying a strategy per risky function. Compress these inventories into function names plus adversarial input classes and mechanical guards, including malformed observations and reproducible effect-order sequences.
          family: function-level-test-strategy
          round: 1
        - id: PQ-3
          severity: Minor
          title: Specify when the stateful fake is checked against Zellij again
          detail: 'ARCH-MOCK: Recorded live findings establish initial behavior but provide no recurring conformance cadence. Name the repeatable disposable-session check and its trigger, such as supported Zellij upgrades, so fullscreen/focus behavior cannot silently drift from the fake.'
          family: external-conformance-cadence
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-20T11:41:24-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: FullscreenState, FullscreenEvent and FullscreenTransition own production effect sequencing, including unconfirmed outcomes; the executor and sequence tests must use that function.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Task 1 names parser, planner, transition and executor functions with malformed-input fuzzing, generated inventories, exhaustive short sequences and deterministic fault/concurrency injection.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 3 names TestFullscreenZellijConformance, its disposable-session command, and required runs before close and on supported Zellij upgrades or modeled-behavior changes.
          round: 2
      blocked: false
content_hash: 34b2913bf74dbf31b91595d8727b840df6a15927d882e3df300dcd71b46b5966
---

# Gate ledger — pair#297 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-20T11:39:44-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `pure-transition-ownership` Make effect-outcome transitions part of the production pure model
  ARCH-ORDER / ARCH-PURE: FullscreenPlan currently selects only the initial operation; save, toggle, focus and clear outcomes remain prose rules for the IO executor. Name the state/event types and transition function that owns subsequent effects, including unconfirmed outcomes, and require the executor and sequence tests to use that function.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose case inventories with named functions and adversarial strategies
  Task 1 enumerates test cases beginning “draft, agent, same terminal, split half” without naming the new parser/decision functions or specifying a strategy per risky function. Compress these inventories into function names plus adversarial input classes and mechanical guards, including malformed observations and reproducible effect-order sequences.
- **PQ-3** [Minor] `external-conformance-cadence` Specify when the stateful fake is checked against Zellij again
  ARCH-MOCK: Recorded live findings establish initial behavior but provide no recurring conformance cadence. Name the repeatable disposable-session check and its trigger, such as supported Zellij upgrades, so fullscreen/focus behavior cannot silently drift from the fake.

## Round 2 — 2026-09-20T11:41:24-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — FullscreenState, FullscreenEvent and FullscreenTransition own production effect sequencing, including unconfirmed outcomes; the executor and sequence tests must use that function.
- PQ-2 — addressed — Task 1 names parser, planner, transition and executor functions with malformed-input fuzzing, generated inventories, exhaustive short sequences and deterministic fault/concurrency injection.
- PQ-3 — addressed — Task 3 names TestFullscreenZellijConformance, its disposable-session command, and required runs before close and on supported Zellij upgrades or modeled-behavior changes.

## Open findings

(none — every finding has been disposed)
