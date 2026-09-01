---
gate: plan-quality
issue: 161
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-01T14:40:55-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace enumerated test cases and diff procedures with named-function risk strategies
          detail: The plan repeatedly enumerates cases and implementation operations at lines 138-149, 163-184, 194-202, 221-232, 246-273, and 289-297. Compress these into one adversarial-input/mechanical-guard strategy line for each risky unit-tested function, naming at least Reduce, NotificationRewriter.Feed, the lifecycle journal append/tail functions, the authorized transcript-following function, and the rendered-working recognizer. Keep production-path acceptance and verification commands, but leave individual cases and procedural diff instructions to the tests and implementation.
          family: executable-test-strategy
          round: 1
        - id: PQ-2
          severity: Important
          title: State the design's non-goals and why they are excluded
          detail: The plan has no explicit non-goals section. It should delimit at least Codex notify-hook mutation, newest-file or cwd-only transcript selection, Worked-for text as authority, transcript normalization, new Couch projection policy, and support for uncaptured turn_complete envelopes, with a brief reason for each exclusion.
          family: explicit-non-goals
          round: 1
        - id: PQ-3
          severity: Important
          title: Define recurring live conformance for the Codex transcript dependency
          detail: ARCH-MOCK requires a live conformance cadence for external behavior the system depends on. The plan names the stateful sessioninventory Runtime fake and one live latency measurement, but not when or how Codex task_started, task_complete, abort, identity, and offset envelopes are checked against an installed Codex. Reuse the existing opt-in TestLiveNativeSessionShapeConformance seam at cmd/internal/sessioninventory/conformance_live_test.go:9, extend its asserted lifecycle surface, and state an operator or scheduled cadence plus drift response.
          family: external-contract-conformance
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-01T14:41:41-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Enumerated test cases and procedural diff instructions remain; the required named-function risk strategies were not substituted.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: The plan still has no explicit non-goals section with reasons for the required exclusions.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: The existing live conformance seam, recurring cadence, asserted lifecycle envelope surface, and drift response remain unspecified.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-01T14:43:10-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: The named-function strategy section is improved, but enumerated cases and procedural diff instructions remain in the lifecycle contract and implementation chunks; apply the executable-test-strategy rule across the entire plan.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: The Non-goals section explicitly excludes all six requested behaviors and gives a reason for each.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: The plan extends TestLiveNativeSessionShapeConformance with lifecycle, identity, outcome, and offset assertions, a pre-release and scheduled macOS cadence, and a fail-closed drift response.
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-01T14:44:34-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The plan now states one named adversarial-input and mechanical-guard strategy per risky function, while task steps retain only references, production acceptance, and verification commands.
          round: 4
      blocked: false
content_hash: 32fc04d9da8fe0fc29e141ddef8b3f6a4b263e457af8701f6216bc90385ff269
---

# Gate ledger — pair#161 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T14:40:55-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `executable-test-strategy` Replace enumerated test cases and diff procedures with named-function risk strategies
  The plan repeatedly enumerates cases and implementation operations at lines 138-149, 163-184, 194-202, 221-232, 246-273, and 289-297. Compress these into one adversarial-input/mechanical-guard strategy line for each risky unit-tested function, naming at least Reduce, NotificationRewriter.Feed, the lifecycle journal append/tail functions, the authorized transcript-following function, and the rendered-working recognizer. Keep production-path acceptance and verification commands, but leave individual cases and procedural diff instructions to the tests and implementation.
- **PQ-2** [Important] `explicit-non-goals` State the design's non-goals and why they are excluded
  The plan has no explicit non-goals section. It should delimit at least Codex notify-hook mutation, newest-file or cwd-only transcript selection, Worked-for text as authority, transcript normalization, new Couch projection policy, and support for uncaptured turn_complete envelopes, with a brief reason for each exclusion.
- **PQ-3** [Important] `external-contract-conformance` Define recurring live conformance for the Codex transcript dependency
  ARCH-MOCK requires a live conformance cadence for external behavior the system depends on. The plan names the stateful sessioninventory Runtime fake and one live latency measurement, but not when or how Codex task_started, task_complete, abort, identity, and offset envelopes are checked against an installed Codex. Reuse the existing opt-in TestLiveNativeSessionShapeConformance seam at cmd/internal/sessioninventory/conformance_live_test.go:9, extend its asserted lifecycle surface, and state an operator or scheduled cadence plus drift response.

## Round 2 — 2026-09-01T14:41:41-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Enumerated test cases and procedural diff instructions remain; the required named-function risk strategies were not substituted.
- PQ-2 — not-addressed — The plan still has no explicit non-goals section with reasons for the required exclusions.
- PQ-3 — not-addressed — The existing live conformance seam, recurring cadence, asserted lifecycle envelope surface, and drift response remain unspecified.

## Round 3 — 2026-09-01T14:43:10-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — not-addressed — The named-function strategy section is improved, but enumerated cases and procedural diff instructions remain in the lifecycle contract and implementation chunks; apply the executable-test-strategy rule across the entire plan.
- PQ-2 — addressed — The Non-goals section explicitly excludes all six requested behaviors and gives a reason for each.
- PQ-3 — addressed — The plan extends TestLiveNativeSessionShapeConformance with lifecycle, identity, outcome, and offset assertions, a pre-release and scheduled macOS cadence, and a fail-closed drift response.

## Round 4 — 2026-09-01T14:44:34-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The plan now states one named adversarial-input and mechanical-guard strategy per risky function, while task steps retain only references, production acceptance, and verification commands.

## Open findings

(none — every finding has been disposed)
