---
gate: boundary-review
issue: 159
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-01T11:08:20-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Core concepts names CLIInvocation, but only cliInvocation exists
          detail: The greppable entity declared at plan line 19 is absent. Append a plan revision and name the delivered package-private entity, or export the implementation if that was intended.
          family: plan-code-entity-traceability
          round: 1
        - id: BR-2
          severity: Important
          title: The obsolete-argv sweep excludes tests that retain a legacy command interpreter
          detail: TestNoCurrentSourcesAdvertiseObsoleteCouchArgv skips all _test.go files, while runRT reconstructs the removed start argv schema and current tests still express obsolete command contracts. Migrate these tests to a typed private-operation helper and sweep test sources with narrow rejection-fixture allowlists.
          family: current-source-shadow-sweep
          round: 1
      blocked: true
---

# Gate ledger — pair#159 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T11:08:20-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `plan-code-entity-traceability` Core concepts names CLIInvocation, but only cliInvocation exists
  The greppable entity declared at plan line 19 is absent. Append a plan revision and name the delivered package-private entity, or export the implementation if that was intended.
- **BR-2** [Important] `current-source-shadow-sweep` The obsolete-argv sweep excludes tests that retain a legacy command interpreter
  TestNoCurrentSourcesAdvertiseObsoleteCouchArgv skips all _test.go files, while runRT reconstructs the removed start argv schema and current tests still express obsolete command contracts. Migrate these tests to a typed private-operation helper and sweep test sources with narrow rejection-fixture allowlists.

## Open findings

- **BR-1** [Critical] `plan-code-entity-traceability` Core concepts names CLIInvocation, but only cliInvocation exists
- **BR-2** [Important] `current-source-shadow-sweep` The obsolete-argv sweep excludes tests that retain a legacy command interpreter
