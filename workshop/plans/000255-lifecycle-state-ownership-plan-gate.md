---
gate: plan-quality
issue: 255
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-15T09:50:24-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace prose case inventories with named function-level test strategies
          detail: workshop/plans/000255-terminal-abstraction-plan.md:151-179 enumerates individual test cases but does not name the report-validation, comparison, and Candidate lifecycle functions to be tested. Name those functions and give each risky function one adversarial-input/mechanical-guard strategy; move individual cases into the executable matrix (ARCH-PURE).
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Minor
          title: Correct the shared FakeHost partial-write capability claim
          detail: The plan at line 40 attributes controlled partial/error writes to hostty.Fake, but cmd/internal/hostty/fake.go:41 only writes to a buffer. Those controls live in cmd/internal/couchtty/mousetrace_test.go:69; identify that helper and the planned shared extension rather than describing the capability as already available (ARCH-MOCK, ARCH-DRY).
          family: existing-behavior-evidence
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-15T09:52:00-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Named function-level adversarial strategies and mechanical guards replace repeated case inventories; exact cases move into executable fixtures.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The plan correctly attributes partial/error writes to mouseTraceHost and identifies extraction or extension for shared presentation tests.
          round: 2
      blocked: false
content_hash: 0a00e902a45fa9e5872309d83e350ad0bb6bbaf1dd5b96506b6f1fea49dc9bf8
---

# Gate ledger — pair#255 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-15T09:50:24-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace prose case inventories with named function-level test strategies
  workshop/plans/000255-terminal-abstraction-plan.md:151-179 enumerates individual test cases but does not name the report-validation, comparison, and Candidate lifecycle functions to be tested. Name those functions and give each risky function one adversarial-input/mechanical-guard strategy; move individual cases into the executable matrix (ARCH-PURE).
- **PQ-2** [Minor] `existing-behavior-evidence` Correct the shared FakeHost partial-write capability claim
  The plan at line 40 attributes controlled partial/error writes to hostty.Fake, but cmd/internal/hostty/fake.go:41 only writes to a buffer. Those controls live in cmd/internal/couchtty/mousetrace_test.go:69; identify that helper and the planned shared extension rather than describing the capability as already available (ARCH-MOCK, ARCH-DRY).

## Round 2 — 2026-09-15T09:52:00-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Named function-level adversarial strategies and mechanical guards replace repeated case inventories; exact cases move into executable fixtures.
- PQ-2 — addressed — The plan correctly attributes partial/error writes to mouseTraceHost and identifies extraction or extension for shared presentation tests.

## Open findings

(none — every finding has been disposed)
