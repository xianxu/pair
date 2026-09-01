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
    - "n": 4
      timestamp: "2026-09-01T15:28:31-07:00"
      agent: codex
      findings:
        - id: BR-5
          severity: Critical
          title: Journal IO blocks the PTY master loop instead of using the required bounded delivery seam
          detail: 'This is the 2nd finding in family `lifecycle-contract-coverage`. The master-loop ticker calls Advance synchronously; Advance performs stat, open, seek, an effectively suffix-sized ReadAll, JSON decoding, and processing of every returned record before PTY events resume. This violates the declared separate-goroutine/channel contract and permits journal growth or slow filesystem IO to stall terminal forwarding. Do not patch only the ReadAll site: enforce the class rule that all lifecycle journal IO remains outside the PTY owner loop, with bounded incremental reads and durable-cursor backpressure, while only reducer mutation executes on the master loop.'
          family: lifecycle-contract-coverage
          round: 4
        - id: BR-6
          severity: Critical
          title: The Core concepts table does not describe entities that actually exist at M3
          detail: The table marks CodexWorkingRecognizer as new although its M4 file does not exist, and names AuthorizedTranscriptFollower and LifecycleJournal entities that have no matching declarations at their stated paths. Under the Core concepts cross-check this is a code/plan contradiction. Revise statuses to distinguish planned from delivered entities and use the actual declaration names, or implement the declared surfaces.
          family: plan-entity-inventory
          round: 4
        - id: BR-7
          severity: Important
          title: The required live Codex lifecycle conformance remains unchanged
          detail: 'This is the 2nd finding in family `regression-test-reachability`. The plan requires TestLiveNativeSessionShapeConformance to assert task_started/task_complete identity and monotonic offsets, but the pinned range does not modify that test; it still only runs the generic inventory report. State the rule for all production envelopes consumed by notification authority: each must be asserted through the recurring live conformance seam when parser support is introduced.'
          family: regression-test-reachability
          round: 4
      boundary: M3
      blocked: true
    - "n": 5
      timestamp: "2026-09-01T15:33:23-07:00"
      agent: codex
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: Candidate worker isolation and a regression exist at HEAD, but the pinned range is empty, so the fix and its fail-without-fix test cannot be reviewed.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: The HEAD plan appears corrected, but no plan change or mechanical reachability check exists in the pinned range.
          round: 5
        - id: BR-7
          disposition: not-addressed
          note: HEAD invokes lifecycle conformance validation, but the empty range contains no live-conformance modification attributable to this boundary.
          round: 5
      findings:
        - id: BR-8
          severity: Critical
          title: The pinned M3 review range contains no changes
          detail: Base and head are the same commit and tree, so stat, name-status, and full patch are empty. This is the 2nd finding in family `submission-boundary-reachability`; enforce a class rule that milestone review rejects identical pre/post boundaries, then rerun with the actual pre-fix base and post-fix head.
          family: submission-boundary-reachability
          round: 5
      boundary: M3
      blocked: true
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

## Round 4 — 2026-09-01T15:28:31-07:00 (codex) — BLOCKED

### Raised

- **BR-5** [Critical] `lifecycle-contract-coverage` Journal IO blocks the PTY master loop instead of using the required bounded delivery seam
  This is the 2nd finding in family `lifecycle-contract-coverage`. The master-loop ticker calls Advance synchronously; Advance performs stat, open, seek, an effectively suffix-sized ReadAll, JSON decoding, and processing of every returned record before PTY events resume. This violates the declared separate-goroutine/channel contract and permits journal growth or slow filesystem IO to stall terminal forwarding. Do not patch only the ReadAll site: enforce the class rule that all lifecycle journal IO remains outside the PTY owner loop, with bounded incremental reads and durable-cursor backpressure, while only reducer mutation executes on the master loop.
- **BR-6** [Critical] `plan-entity-inventory` The Core concepts table does not describe entities that actually exist at M3
  The table marks CodexWorkingRecognizer as new although its M4 file does not exist, and names AuthorizedTranscriptFollower and LifecycleJournal entities that have no matching declarations at their stated paths. Under the Core concepts cross-check this is a code/plan contradiction. Revise statuses to distinguish planned from delivered entities and use the actual declaration names, or implement the declared surfaces.
- **BR-7** [Important] `regression-test-reachability` The required live Codex lifecycle conformance remains unchanged
  This is the 2nd finding in family `regression-test-reachability`. The plan requires TestLiveNativeSessionShapeConformance to assert task_started/task_complete identity and monotonic offsets, but the pinned range does not modify that test; it still only runs the generic inventory report. State the rule for all production envelopes consumed by notification authority: each must be asserted through the recurring live conformance seam when parser support is introduced.

## Round 5 — 2026-09-01T15:33:23-07:00 (codex) — BLOCKED

### Disposed

- BR-5 — not-addressed — Candidate worker isolation and a regression exist at HEAD, but the pinned range is empty, so the fix and its fail-without-fix test cannot be reviewed.
- BR-6 — not-addressed — The HEAD plan appears corrected, but no plan change or mechanical reachability check exists in the pinned range.
- BR-7 — not-addressed — HEAD invokes lifecycle conformance validation, but the empty range contains no live-conformance modification attributable to this boundary.

### Raised

- **BR-8** [Critical] `submission-boundary-reachability` The pinned M3 review range contains no changes
  Base and head are the same commit and tree, so stat, name-status, and full patch are empty. This is the 2nd finding in family `submission-boundary-reachability`; enforce a class rule that milestone review rejects identical pre/post boundaries, then rerun with the actual pre-fix base and post-fix head.

## Open findings

- **BR-5** [Critical] `lifecycle-contract-coverage` Journal IO blocks the PTY master loop instead of using the required bounded delivery seam
- **BR-6** [Critical] `plan-entity-inventory` The Core concepts table does not describe entities that actually exist at M3
- **BR-7** [Important] `regression-test-reachability` The required live Codex lifecycle conformance remains unchanged
- **BR-8** [Critical] `submission-boundary-reachability` The pinned M3 review range contains no changes
