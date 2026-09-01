---
gate: boundary-review
issue: 161
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-01T14:52:38-07:00"
      agent: codex
      boundary: M1
      blocked: false
    - "n": 2
      timestamp: "2026-09-01T15:05:53-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Normal composer submissions do not open a lifecycle generation
          detail: Submission publication is conditioned on emitting exactly bare CR, excluding active Codex LF and active Claude backslash-CR paths. Native completion is then silently ignored while the reducer is idle; publish from the accepted submission decision and pin both composer paths with production-flow tests.
          family: submission-boundary-reachability
          round: 2
        - id: BR-2
          severity: Important
          title: Native notification regression injects an unrelated progress opener
          detail: The prior direct native OSC test was changed to prepend OSC progress working, so it no longer catches missing submission-to-native reachability. Restore a real submission-to-completion regression and keep progress opening as a separate case.
          family: regression-test-reachability
          round: 2
        - id: BR-3
          severity: Important
          title: New lifecycle source is absent from the exhaustive artifact inventory
          detail: TestProductionArtifactReferencesAreExactlyClassified fails because cmd/internal/wrapcmd/notification_lifecycle.go is unclassified. Add the appropriate inventory entry and rerun the repository suite.
          family: production-source-inventory
          round: 2
        - id: BR-4
          severity: Important
          title: Reducer tests do not exercise the plan's keyed and source-order contract
          detail: The checked plan claims arbitrary keyed/keyless sequences and source permutations, but coverage omits ordering permutations, key mismatches, differing keyed starts, abort outcome distinction, and production submission integration.
          family: lifecycle-contract-coverage
          round: 2
      boundary: M2
      blocked: true
    - "n": 3
      timestamp: "2026-09-01T15:12:52-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Legacy and KKP Alt+Enter now publish the lifecycle opener for Codex and Claude, with production-flow tests.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Submission-to-native completion is restored as an independent regression; progress-opened native completion remains separate.
          round: 3
        - id: BR-3
          disposition: addressed
          note: notification_lifecycle.go is classified and the exhaustive artifact inventory test passes.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Tests cover source order, keyed mismatch and replacement, distinct abort fallback, arbitrary sequences, and production submission integration.
          round: 3
      boundary: M2
      blocked: false
---

# Gate ledger — 000161-couch-missed-codex-notifications#161 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T14:52:38-07:00 (codex) — passed

## Round 2 — 2026-09-01T15:05:53-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `submission-boundary-reachability` Normal composer submissions do not open a lifecycle generation
  Submission publication is conditioned on emitting exactly bare CR, excluding active Codex LF and active Claude backslash-CR paths. Native completion is then silently ignored while the reducer is idle; publish from the accepted submission decision and pin both composer paths with production-flow tests.
- **BR-2** [Important] `regression-test-reachability` Native notification regression injects an unrelated progress opener
  The prior direct native OSC test was changed to prepend OSC progress working, so it no longer catches missing submission-to-native reachability. Restore a real submission-to-completion regression and keep progress opening as a separate case.
- **BR-3** [Important] `production-source-inventory` New lifecycle source is absent from the exhaustive artifact inventory
  TestProductionArtifactReferencesAreExactlyClassified fails because cmd/internal/wrapcmd/notification_lifecycle.go is unclassified. Add the appropriate inventory entry and rerun the repository suite.
- **BR-4** [Important] `lifecycle-contract-coverage` Reducer tests do not exercise the plan's keyed and source-order contract
  The checked plan claims arbitrary keyed/keyless sequences and source permutations, but coverage omits ordering permutations, key mismatches, differing keyed starts, abort outcome distinction, and production submission integration.

## Round 3 — 2026-09-01T15:12:52-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Legacy and KKP Alt+Enter now publish the lifecycle opener for Codex and Claude, with production-flow tests.
- BR-2 — addressed — Submission-to-native completion is restored as an independent regression; progress-opened native completion remains separate.
- BR-3 — addressed — notification_lifecycle.go is classified and the exhaustive artifact inventory test passes.
- BR-4 — addressed — Tests cover source order, keyed mismatch and replacement, distinct abort fallback, arbitrary sequences, and production submission integration.

## Open findings

(none — every finding has been disposed)
