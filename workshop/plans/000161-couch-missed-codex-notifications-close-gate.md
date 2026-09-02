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
    - "n": 6
      timestamp: "2026-09-01T15:36:56-07:00"
      agent: codex
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: The worker is wired in production, but its regression invokes the worker and reducer independently; reverting the master-loop integration would not make this test fail.
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: The table still marks terminalModel modified and Couch projection verified although their M4 work remains unchecked, and the prose names CodexWorkingRecognizer while the planned declaration is RecognizeCodexWorking.
          round: 6
        - id: BR-7
          disposition: addressed
          note: The opt-in live conformance test now scans Codex events and requires a keyed task_started followed by same-root task_complete with timestamps and increasing positions.
          round: 6
        - id: BR-8
          disposition: addressed
          note: The pinned objects exist and the range contains 26 changed files with substantive production and test changes.
          round: 6
      findings:
        - id: BR-9
          severity: Critical
          title: Lifecycle journal validation accepts malformed or unauthorized record semantics
          detail: 'This is the 3rd finding in family lifecycle-contract-coverage. Advance decodes one JSON value without requiring EOF, and validates only that agent, source, and outcome are non-empty; a line containing a valid record followed by garbage, or a current-launch record for another agent/source with outcome completed, can reach the notification reducer despite the fail-closed contract. Do not patch only trailing JSON: state and enforce the complete accepted-record rule, including exact framing, agent, source, outcome, identity, and timestamp constraints, with production-path regressions that fail when each guard is removed.'
          family: lifecycle-contract-coverage
          round: 6
      boundary: M3
      blocked: true
    - "n": 7
      timestamp: "2026-09-01T15:45:01-07:00"
      agent: codex
      dispose:
        - id: BR-5
          disposition: addressed
          note: Journal IO now runs in a dedicated bounded worker, and a production master-pump regression proves PTY forwarding continues while injected journal IO blocks.
          round: 7
        - id: BR-6
          disposition: addressed
          note: The Core concepts table now distinguishes planned M4 entities from delivered entities and uses declarations present at their stated paths.
          round: 7
        - id: BR-9
          disposition: addressed
          note: Strict EOF framing and a shared closed record validator now reject invalid agent, source, outcome, identity, and timestamp fields, with reader-path regressions.
          round: 7
        - id: BR-7
          disposition: addressed
          note: The opt-in live conformance test invokes the new keyed Codex lifecycle-envelope validator.
          round: 7
        - id: BR-8
          disposition: addressed
          note: The pinned review range now contains the substantive M3 implementation and regression changes.
          round: 7
      findings:
        - id: BR-10
          severity: Critical
          title: Journal reconciliation treats unauthorized identity collisions as committed lifecycle records
          detail: lifecycleIdentityPresent permissively decodes rows and compares only launch, artifact generation, and offset, so a malformed or semantically different row can suppress the intended append and silently lose notification authority. This is the 4th finding in family lifecycle-contract-coverage; enforce strict complete-record equivalence across every reconciliation candidate and pin the enumerable malformed and semantic-mismatch classes with producer-path regressions.
          family: lifecycle-contract-coverage
          round: 7
      boundary: M3
      blocked: true
    - "n": 8
      timestamp: "2026-09-01T15:49:42-07:00"
      agent: codex
      dispose:
        - id: BR-10
          disposition: addressed
          note: Strict decoding plus complete-record equality now governs reconciliation, and the producer-path regression fails when the former permissive three-field comparison is restored.
          round: 8
      boundary: M3
      blocked: false
    - "n": 9
      timestamp: "2026-09-01T16:15:08-07:00"
      agent: codex
      findings:
        - id: BR-11
          severity: Important
          title: The claimed wrapper-to-Couch production acceptance path is split across disconnected tests
          detail: This is the 3rd finding in family `submission-boundary-reachability`. `codex_working_test.go` verifies wrapper output against `notifyosc.Encode`, while `notification_test.go` independently injects bytes from that same encoder rather than bytes emitted by the wrapper. State and enforce the class rule that cross-package acceptance tests transport upstream production output into the downstream production consumer; replay the Codex fixture through `handleChunk`, feed those exact emitted bytes into Couch, and assert both status and switcher projections.
          family: submission-boundary-reachability
          round: 9
      boundary: M4
      blocked: true
    - "n": 10
      timestamp: "2026-09-01T16:21:56-07:00"
      agent: codex
      dispose:
        - id: BR-11
          disposition: addressed
          note: The acceptance test transports bytes emitted by handleChunk through Couch's running Console and asserts both the pending status chip and switcher message.
          round: 10
      boundary: M4
      blocked: false
    - "n": 11
      timestamp: "2026-09-01T16:25:01-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Legacy and KKP Alt+Enter production paths publish submission boundaries, with plain Enter remaining non-authoritative.
          round: 11
        - id: BR-2
          disposition: addressed
          note: Native completion regressions now traverse real send boundaries without an unrelated progress opener.
          round: 11
        - id: BR-3
          disposition: addressed
          note: The lifecycle sources are present in the exhaustive artifact manifest.
          round: 11
        - id: BR-4
          disposition: addressed
          note: Reducer tests cover source ordering, keyed mismatch, new keyed turns, and abort outcomes.
          round: 11
        - id: BR-5
          disposition: addressed
          note: Journal advancement runs in a worker, and a masterPump test proves PTY forwarding while injected journal IO blocks.
          round: 11
        - id: BR-6
          disposition: addressed
          note: The Core concepts table names greppable entities with accurate delivered milestone statuses.
          round: 11
        - id: BR-7
          disposition: addressed
          note: The opt-in live conformance invokes lifecycle validation for keyed same-root opener and terminal ordering.
          round: 11
        - id: BR-8
          disposition: addressed
          note: The pinned range contains the M3 implementation and subsequent fixes.
          round: 11
        - id: BR-9
          disposition: addressed
          note: Writer and tailer share strict decoding and closed validation of lifecycle record semantics.
          round: 11
        - id: BR-10
          disposition: addressed
          note: Reconciliation requires complete semantic equality, with a producer collision matrix covering each field.
          round: 11
        - id: BR-11
          disposition: addressed
          note: The joined acceptance test feeds wrapper-emitted bytes into the running Couch console and verifies both named surfaces.
          round: 11
      findings:
        - id: BR-12
          severity: Critical
          title: Claude progress OSC grants completion authority to every wrapped agent
          detail: 'NotificationRewriter emits Working and Stopped observations independently of agent identity, and handleChunk reduces them unconditionally. This is the 5th finding in family lifecycle-contract-coverage: define and enforce the complete per-agent source-authority matrix, with negative production-path coverage for every unauthorized supported agent.'
          family: lifecycle-contract-coverage
          round: 11
      blocked: true
    - "n": 12
      timestamp: "2026-09-01T16:31:28-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Legacy and KKP Alt+Enter publish submission boundaries; plain Enter remains non-authoritative.
          round: 12
        - id: BR-2
          disposition: addressed
          note: Native completion tests traverse real submission boundaries without an unrelated progress opener.
          round: 12
        - id: BR-3
          disposition: addressed
          note: Lifecycle sources are included in the exhaustive artifact inventory.
          round: 12
        - id: BR-4
          disposition: addressed
          note: Reducer coverage pins source ordering, keyed mismatch, new keyed turns, and abort outcomes.
          round: 12
        - id: BR-5
          disposition: addressed
          note: Journal IO runs outside the PTY loop, with blocked-IO forwarding coverage.
          round: 12
        - id: BR-6
          disposition: addressed
          note: The Core concepts table names existing entities at accurate paths and statuses.
          round: 12
        - id: BR-7
          disposition: addressed
          note: Live conformance checks keyed Codex opener/terminal ordering and identity.
          round: 12
        - id: BR-8
          disposition: addressed
          note: The pinned range contains the claimed M3 implementation and fixes.
          round: 12
        - id: BR-9
          disposition: addressed
          note: Lifecycle records use strict framing and closed semantic validation.
          round: 12
        - id: BR-10
          disposition: addressed
          note: Reconciliation requires exact semantic equality and is covered by the collision matrix.
          round: 12
        - id: BR-11
          disposition: addressed
          note: Joined acceptance coverage feeds wrapper output into Couch and verifies status and switcher state.
          round: 12
        - id: BR-12
          disposition: addressed
          note: Claude-only progress authority is gated before reduction; the production matrix covers every supported unauthorized agent and fails when the gate is removed.
          round: 12
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

## Round 6 — 2026-09-01T15:36:56-07:00 (codex) — BLOCKED

### Disposed

- BR-5 — not-addressed — The worker is wired in production, but its regression invokes the worker and reducer independently; reverting the master-loop integration would not make this test fail.
- BR-6 — not-addressed — The table still marks terminalModel modified and Couch projection verified although their M4 work remains unchecked, and the prose names CodexWorkingRecognizer while the planned declaration is RecognizeCodexWorking.
- BR-7 — addressed — The opt-in live conformance test now scans Codex events and requires a keyed task_started followed by same-root task_complete with timestamps and increasing positions.
- BR-8 — addressed — The pinned objects exist and the range contains 26 changed files with substantive production and test changes.

### Raised

- **BR-9** [Critical] `lifecycle-contract-coverage` Lifecycle journal validation accepts malformed or unauthorized record semantics
  This is the 3rd finding in family lifecycle-contract-coverage. Advance decodes one JSON value without requiring EOF, and validates only that agent, source, and outcome are non-empty; a line containing a valid record followed by garbage, or a current-launch record for another agent/source with outcome completed, can reach the notification reducer despite the fail-closed contract. Do not patch only trailing JSON: state and enforce the complete accepted-record rule, including exact framing, agent, source, outcome, identity, and timestamp constraints, with production-path regressions that fail when each guard is removed.

## Round 7 — 2026-09-01T15:45:01-07:00 (codex) — BLOCKED

### Disposed

- BR-5 — addressed — Journal IO now runs in a dedicated bounded worker, and a production master-pump regression proves PTY forwarding continues while injected journal IO blocks.
- BR-6 — addressed — The Core concepts table now distinguishes planned M4 entities from delivered entities and uses declarations present at their stated paths.
- BR-9 — addressed — Strict EOF framing and a shared closed record validator now reject invalid agent, source, outcome, identity, and timestamp fields, with reader-path regressions.
- BR-7 — addressed — The opt-in live conformance test invokes the new keyed Codex lifecycle-envelope validator.
- BR-8 — addressed — The pinned review range now contains the substantive M3 implementation and regression changes.

### Raised

- **BR-10** [Critical] `lifecycle-contract-coverage` Journal reconciliation treats unauthorized identity collisions as committed lifecycle records
  lifecycleIdentityPresent permissively decodes rows and compares only launch, artifact generation, and offset, so a malformed or semantically different row can suppress the intended append and silently lose notification authority. This is the 4th finding in family lifecycle-contract-coverage; enforce strict complete-record equivalence across every reconciliation candidate and pin the enumerable malformed and semantic-mismatch classes with producer-path regressions.

## Round 8 — 2026-09-01T15:49:42-07:00 (codex) — passed

### Disposed

- BR-10 — addressed — Strict decoding plus complete-record equality now governs reconciliation, and the producer-path regression fails when the former permissive three-field comparison is restored.

## Round 9 — 2026-09-01T16:15:08-07:00 (codex) — BLOCKED

### Raised

- **BR-11** [Important] `submission-boundary-reachability` The claimed wrapper-to-Couch production acceptance path is split across disconnected tests
  This is the 3rd finding in family `submission-boundary-reachability`. `codex_working_test.go` verifies wrapper output against `notifyosc.Encode`, while `notification_test.go` independently injects bytes from that same encoder rather than bytes emitted by the wrapper. State and enforce the class rule that cross-package acceptance tests transport upstream production output into the downstream production consumer; replay the Codex fixture through `handleChunk`, feed those exact emitted bytes into Couch, and assert both status and switcher projections.

## Round 10 — 2026-09-01T16:21:56-07:00 (codex) — passed

### Disposed

- BR-11 — addressed — The acceptance test transports bytes emitted by handleChunk through Couch's running Console and asserts both the pending status chip and switcher message.

## Round 11 — 2026-09-01T16:25:01-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Legacy and KKP Alt+Enter production paths publish submission boundaries, with plain Enter remaining non-authoritative.
- BR-2 — addressed — Native completion regressions now traverse real send boundaries without an unrelated progress opener.
- BR-3 — addressed — The lifecycle sources are present in the exhaustive artifact manifest.
- BR-4 — addressed — Reducer tests cover source ordering, keyed mismatch, new keyed turns, and abort outcomes.
- BR-5 — addressed — Journal advancement runs in a worker, and a masterPump test proves PTY forwarding while injected journal IO blocks.
- BR-6 — addressed — The Core concepts table names greppable entities with accurate delivered milestone statuses.
- BR-7 — addressed — The opt-in live conformance invokes lifecycle validation for keyed same-root opener and terminal ordering.
- BR-8 — addressed — The pinned range contains the M3 implementation and subsequent fixes.
- BR-9 — addressed — Writer and tailer share strict decoding and closed validation of lifecycle record semantics.
- BR-10 — addressed — Reconciliation requires complete semantic equality, with a producer collision matrix covering each field.
- BR-11 — addressed — The joined acceptance test feeds wrapper-emitted bytes into the running Couch console and verifies both named surfaces.

### Raised

- **BR-12** [Critical] `lifecycle-contract-coverage` Claude progress OSC grants completion authority to every wrapped agent
  NotificationRewriter emits Working and Stopped observations independently of agent identity, and handleChunk reduces them unconditionally. This is the 5th finding in family lifecycle-contract-coverage: define and enforce the complete per-agent source-authority matrix, with negative production-path coverage for every unauthorized supported agent.

## Round 12 — 2026-09-01T16:31:28-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Legacy and KKP Alt+Enter publish submission boundaries; plain Enter remains non-authoritative.
- BR-2 — addressed — Native completion tests traverse real submission boundaries without an unrelated progress opener.
- BR-3 — addressed — Lifecycle sources are included in the exhaustive artifact inventory.
- BR-4 — addressed — Reducer coverage pins source ordering, keyed mismatch, new keyed turns, and abort outcomes.
- BR-5 — addressed — Journal IO runs outside the PTY loop, with blocked-IO forwarding coverage.
- BR-6 — addressed — The Core concepts table names existing entities at accurate paths and statuses.
- BR-7 — addressed — Live conformance checks keyed Codex opener/terminal ordering and identity.
- BR-8 — addressed — The pinned range contains the claimed M3 implementation and fixes.
- BR-9 — addressed — Lifecycle records use strict framing and closed semantic validation.
- BR-10 — addressed — Reconciliation requires exact semantic equality and is covered by the collision matrix.
- BR-11 — addressed — Joined acceptance coverage feeds wrapper output into Couch and verifies status and switcher state.
- BR-12 — addressed — Claude-only progress authority is gated before reduction; the production matrix covers every supported unauthorized agent and fails when the gate is removed.

## Open findings

(none — every finding has been disposed)
