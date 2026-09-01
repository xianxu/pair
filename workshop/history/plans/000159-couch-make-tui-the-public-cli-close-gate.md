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
    - "n": 2
      timestamp: "2026-09-01T11:20:57-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The Core concepts table, explanatory prose, and delivered symbol now consistently name package-private cliInvocation.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Command tests now use typed-operation or explicit launch helpers, and the obsolete-argv audit scans all Go tests with line-local rejection-fixture allowances.
          round: 2
      findings:
        - id: BR-3
          severity: Critical
          title: --show accepts another public flag as its reference
          detail: ParseCLI accepts any non-empty second token after --show, so forms such as --show --list and --show --help succeed despite the Spec requiring every public flag form to reject combination with another flag. Reject flag-shaped references and pin the full public-flag class with parser tests. This violates ARCH-PURPOSE.
          family: closed-public-argv-grammar
          round: 2
        - id: BR-4
          severity: Important
          title: Installed smoke does not enforce the promised exact Pair invocation
          detail: The smoke recognizes only the pair resume prefix and finally checks only for pair followed by a space, so it stays green if the generated tag or required --layout2 argument disappears. Assert the exact recorded pair resume <tag> --layout2 call at the process seam.
          family: integration-smoke-observable-contract
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-01T11:28:57-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: ParseCLI rejects flag-shaped show references, and parser regressions enumerate the public reserved-flag class.
          round: 3
        - id: BR-4
          disposition: addressed
          note: The installed smoke requires exactly one pair resume <16-hex Couch tag> --layout2 invocation.
          round: 3
      blocked: false
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

## Round 2 — 2026-09-01T11:20:57-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The Core concepts table, explanatory prose, and delivered symbol now consistently name package-private cliInvocation.
- BR-2 — addressed — Command tests now use typed-operation or explicit launch helpers, and the obsolete-argv audit scans all Go tests with line-local rejection-fixture allowances.

### Raised

- **BR-3** [Critical] `closed-public-argv-grammar` --show accepts another public flag as its reference
  ParseCLI accepts any non-empty second token after --show, so forms such as --show --list and --show --help succeed despite the Spec requiring every public flag form to reject combination with another flag. Reject flag-shaped references and pin the full public-flag class with parser tests. This violates ARCH-PURPOSE.
- **BR-4** [Important] `integration-smoke-observable-contract` Installed smoke does not enforce the promised exact Pair invocation
  The smoke recognizes only the pair resume prefix and finally checks only for pair followed by a space, so it stays green if the generated tag or required --layout2 argument disappears. Assert the exact recorded pair resume <tag> --layout2 call at the process seam.

## Round 3 — 2026-09-01T11:28:57-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — ParseCLI rejects flag-shaped show references, and parser regressions enumerate the public reserved-flag class.
- BR-4 — addressed — The installed smoke requires exactly one pair resume <16-hex Couch tag> --layout2 invocation.

## Open findings

(none — every finding has been disposed)
