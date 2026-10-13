---
gate: boundary-review
issue: 517
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T20:49:39-07:00"
      dispatched: "2026-10-12T20:47:28-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Switching an already-parked source retains an older incarnation's evidence
          detail: cmd/internal/couchcore/switchagent.go:361 preserves any existing descriptor for a parked source. After A→B, a separate park of B, and B→C, previous-session access still selects A. ARCH-ORDER and ARCH-PURPOSE require comparing the descriptor's source with the current verified park, retaining it only for the same source. Add a regression sequence covering separate parking while preserving failed-launch retry behavior.
          family: evidence-reuse-requires-source-identity
          round: 1
        - id: BR-2
          severity: Critical
          title: M1 core-concepts locations and modification claims contradict the diff
          detail: workshop/plans/000517-agent-switch-plan.md:114 and :139–143 name incorrect M1 locations or unchanged files. PeekSelection lives in peek_selection.go; persistence and capture reading introduce previous_session.go and bounded.go, with runtime wiring in run.go. Correct the table and add a Revisions entry, as required by the core-concepts cross-check.
          family: core-concepts-match-implementation
          round: 1
        - id: BR-3
          severity: Important
          title: Exact capture reader lacks the promised lease and collection ordering tests
          detail: cmd/internal/scrollbackcmd/bounded_test.go:98 tests selection only. ARCH-MOCK and ARCH-ORDER require controlled coverage through RenderOwnedCaptureLines proving the exact capture is protected during replay and leases release after both success and failure. Existing retention tests exercise a different entry point.
          family: retention-ordering-regression-coverage
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T20:58:18-07:00"
      dispatched: "2026-10-12T20:54:42-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: switchagent.go:361 compares the descriptor source with the verified park. Reverting that condition makes TestPreviousSessionSuccessiveSourceParksChooseImmediatePrevious/separate_park_before_switch fail; failed-launch retry coverage also passes.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The pinned correction updates plan lines 114 and 139–143 to peek_selection.go, previous_session.go, bounded.go, and run.go, matching their actual definitions and wiring. The Revisions entry at line 326 records the correction and identifies unchanged dependencies.
          round: 2
        - id: BR-3
          disposition: addressed
          note: TestOwnedCaptureLeaseProtectsReplayAndReleases interleaves real collection with the production reader and replay. Removing either lease acquisition or release makes both success and failure cases fail.
          round: 2
      findings:
        - id: BR-4
          severity: Critical
          title: Full replay silently evicts history and reports the remaining tail as complete
          detail: 'cmd/internal/scrollbackcmd/scrollbackcmd.go:382–409 changes only the emulator history row limit, leaving its 65,536-cell and 4 MiB history limits active. A 2,000-line, 80-column recording returns only 464 lines, starting at line 1536, with no error; PeekThreadSelection then sets Complete=true. ARCH-PURPOSE and ARCH-CONSTRAINTS: configure explicit replay limits and detect every capacity-driven eviction, returning a visible error instead of declaring truncated output complete. Add regression coverage exceeding the cell and byte limits independently.'
          family: full-replay-requires-loss-detection
          round: 2
        - id: BR-5
          severity: Critical
          title: Accepted recording dimensions can panic during emulator construction
          detail: 'cmd/internal/scrollbackcmd/bounded.go:38–43 accepts 1000×1000, but replayData calls vt.NewEmulator, whose default maximum is 262,144 cells. An initial 1000×1000 resize reproducibly panics with "vt: invalid geometry"; later oversized-area resizes are silently rejected by Emulator.Resize. ARCH-SECURE and ARCH-CONSTRAINTS: align admission with explicit emulator geometry limits, use error-returning construction and resize operations, and test both initial and subsequent resize failures.'
          family: replay-admission-matches-emulator-limits
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T21:10:48-07:00"
      dispatched: "2026-10-12T21:05:54-07:00"
      agent: codex
      dispose:
        - id: BR-4
          disposition: addressed
          note: Explicit history limits and monotonic capacity-loss accounting prevent silent truncation. Removing accounting fails independent cell/byte, rejected-row, erase-after-loss, and resize-loss regressions.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Screen-area admission and checked emulator construction/resizing reject invalid geometry. Removing enforcement reproduces the initial panic and silently ignored later resize; regression tests fail.
          round: 3
        - id: BR-1
          disposition: addressed
          note: Exact verified-park identity controls descriptor reuse; successive-source and failed-launch retry tests pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: The plan's M1 file map and Revisions identify previous_session.go, peek_selection.go, bounded.go, and run.go as implemented owners and distinguish unchanged dependencies.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Controlled collection during exact-capture replay verifies protection and subsequent lease release on success and failure.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 0f3a7543cb999a96c70c2d80b64a9c71ecd73691
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T23:01:40-07:00"
      dispatched: "2026-10-12T22:54:40-07:00"
      agent: codex
      findings:
        - id: BR-6
          severity: Critical
          title: Resume can discard a surviving replacement's pending start identity
          detail: 'ARCH-ORDER / ARCH-PURPOSE: cmd/internal/couchcore/resume.go:459 checks incarnation.LaunchProfile but misses incarnation.Start.LaunchProfile before registration or unknown promotion. An isolated regression with dead supervisor/helper and a surviving target terminal remained pending after startup reconciliation, then resume erased the target claim (revision 7 to 10, incarnations empty) and returned resume-session-gone. Cover every pending/completed replacement representation before lifecycle cleanup and add interruption-sequence tests asserting retained identity and permitted effects.'
          family: replacement-recovery-preserves-unconfirmed-effects
          round: 4
        - id: BR-7
          severity: Critical
          title: The active core-concepts table claims a resolver change absent from the implementation
          detail: 'ARCH-PURE traceability: workshop/plans/000517-agent-switch-plan.md:872 claims LaunchProfileResolution gained an explicit configured/missing result, but cmd/internal/couchcore/launchprofile.go:52 is unchanged and the check lives in switchagent.go:241. This is the 2nd finding in family core-concepts-match-implementation. Earlier rounds fixed instances; do not fix only this row. Apply a rule requiring every active row''s owner, classification and change status to match the implementation against a named baseline, sweep the table, and document the correction under Revisions. One concrete entity contradiction was verified.'
          family: core-concepts-match-implementation
          round: 4
        - id: BR-8
          severity: Minor
          title: Atlas still names a deleted orientation-pruning helper
          detail: atlas/couch.md:1330 refers to dropOrientationLocked, removed from cmd/internal/couchtty/console_continuation.go in this range. Update the passage to describe reconcileContinuationOrientationLocked.
          family: documentation-referents-match-implementation
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T23:34:55-07:00"
      dispatched: "2026-10-12T23:28:27-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: resume.go checks pending profile/binding authority and rechecks before cleanup. Three isolated mutations each fail TestInterruptedSwitchResumePreservesTargetAuthority; restored tests pass.
          round: 5
        - id: BR-7
          disposition: addressed
          note: Plan lines 1020–1088 replace the active inventory with a baseline-specific ownership/classification rule and complete table. Checked owners and unchanged dependencies against M1 37707f73b; launchprofile.go remains correctly classified unchanged.
          round: 5
        - id: BR-8
          disposition: addressed
          note: atlas/couch.md:1330 now names reconcileContinuationOrientationLocked, matching the implementation at console_continuation.go:152 and removal of dropOrientationLocked.
          round: 5
      findings:
        - id: BR-9
          severity: Important
          title: Pure previous-tty projection and selection lack direct tests without IO
          detail: 'ARCH-PURE: previous_session_test.go:62 constructs filesystem-backed Couch state through newTestEnv/createParkedThreadInCouch; peek_previous_test.go exercises SelectPeekSource only through integration paths. Add value-only tests for previousSessionEvidence and SelectPeekSource covering missing park/capture, exact source selection, malformed evidence, and defensive copying. The functions themselves are pure; retain that classification.'
          family: pure-core-tested-without-io
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 6
      timestamp: "2026-10-12T23:44:59-07:00"
      dispatched: "2026-10-12T23:38:54-07:00"
      agent: codex
      dispose:
        - id: BR-9
          disposition: addressed
          note: previous_session_pure_test.go:13 and peek_selection_test.go:29,52,80 use value-only fixtures covering missing evidence, exact selection, malformed evidence and defensive copying. Removing capture copying, binding copying or validation independently fails these tests; restored code passes.
          round: 6
        - id: BR-6
          disposition: addressed
          note: Prior disposition retained. resume.go checks pending replacement authority and rechecks absence before cleanup; interrupted-switch recovery sequence tests pass.
          round: 6
        - id: BR-7
          disposition: addressed
          note: Prior disposition retained. The active plan inventory identifies ResolveLaunchProfile as unchanged and known-configuration admission as integration code in switchagent.go; comparison against the named M1 baseline confirms the unchanged dependencies.
          round: 6
        - id: BR-8
          disposition: addressed
          note: Prior disposition retained. atlas/couch.md now names reconcileContinuationOrientationLocked, which exists in console_continuation.go and owns continuation prompt pruning.
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 902ecdf391aa4d28136c7f0cb63273a4027ffed9
      blocked: false
    - "n": 7
      timestamp: "2026-10-13T07:41:22-07:00"
      dispatched: "2026-10-13T07:35:56-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Successive-source tests pass; removing source-identity comparison fails both in-switch and separately parked cases.
          round: 7
        - id: BR-2
          disposition: addressed
          note: The authoritative BR-7 inventory supersedes historical tables and identifies implemented owners and classifications.
          round: 7
        - id: BR-3
          disposition: addressed
          note: Lease tests exercise collection during replay and release on success/error; removing lease closure makes both cases fail.
          round: 7
        - id: BR-4
          disposition: addressed
          note: Capacity-loss tests cover rows, cells, bytes and subsequent history clearing; disabling loss checks makes them fail.
          round: 7
        - id: BR-5
          disposition: addressed
          note: Geometry tests pass; replacing checked emulator construction makes the geometry regression test fail.
          round: 7
        - id: BR-6
          disposition: addressed
          note: Interrupted-target sequence tests pass; bypassing pending Start authority causes record-preservation assertions to fail.
          round: 7
        - id: BR-7
          disposition: addressed
          note: The active inventory correctly calls ResolveLaunchProfile unchanged and places known-configuration admission in switchagent.go; comparison against the named M1 baseline confirms this.
          round: 7
        - id: BR-8
          disposition: addressed
          note: Atlas now names reconcileContinuationOrientationLocked, matching the existing pruning implementation and call sites in console_continuation.go.
          round: 7
        - id: BR-9
          disposition: addressed
          note: Direct projection/selection tests use values without IO; removing the selected binding copy fails the aliasing assertion.
          round: 7
      recipe: milestone-review
      isolation: fenced
      reviewed: 489ec0bfc6c07ef2864d9399415e629f9aa10d98
      blocked: false
---

# Gate ledger — pair#517 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T20:49:39-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `evidence-reuse-requires-source-identity` Switching an already-parked source retains an older incarnation's evidence
  cmd/internal/couchcore/switchagent.go:361 preserves any existing descriptor for a parked source. After A→B, a separate park of B, and B→C, previous-session access still selects A. ARCH-ORDER and ARCH-PURPOSE require comparing the descriptor's source with the current verified park, retaining it only for the same source. Add a regression sequence covering separate parking while preserving failed-launch retry behavior.
- **BR-2** [Critical] `core-concepts-match-implementation` M1 core-concepts locations and modification claims contradict the diff
  workshop/plans/000517-agent-switch-plan.md:114 and :139–143 name incorrect M1 locations or unchanged files. PeekSelection lives in peek_selection.go; persistence and capture reading introduce previous_session.go and bounded.go, with runtime wiring in run.go. Correct the table and add a Revisions entry, as required by the core-concepts cross-check.
- **BR-3** [Important] `retention-ordering-regression-coverage` Exact capture reader lacks the promised lease and collection ordering tests
  cmd/internal/scrollbackcmd/bounded_test.go:98 tests selection only. ARCH-MOCK and ARCH-ORDER require controlled coverage through RenderOwnedCaptureLines proving the exact capture is protected during replay and leases release after both success and failure. Existing retention tests exercise a different entry point.

## Round 2 — 2026-10-12T20:58:18-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — switchagent.go:361 compares the descriptor source with the verified park. Reverting that condition makes TestPreviousSessionSuccessiveSourceParksChooseImmediatePrevious/separate_park_before_switch fail; failed-launch retry coverage also passes.
- BR-2 — addressed — The pinned correction updates plan lines 114 and 139–143 to peek_selection.go, previous_session.go, bounded.go, and run.go, matching their actual definitions and wiring. The Revisions entry at line 326 records the correction and identifies unchanged dependencies.
- BR-3 — addressed — TestOwnedCaptureLeaseProtectsReplayAndReleases interleaves real collection with the production reader and replay. Removing either lease acquisition or release makes both success and failure cases fail.

### Raised

- **BR-4** [Critical] `full-replay-requires-loss-detection` Full replay silently evicts history and reports the remaining tail as complete
  cmd/internal/scrollbackcmd/scrollbackcmd.go:382–409 changes only the emulator history row limit, leaving its 65,536-cell and 4 MiB history limits active. A 2,000-line, 80-column recording returns only 464 lines, starting at line 1536, with no error; PeekThreadSelection then sets Complete=true. ARCH-PURPOSE and ARCH-CONSTRAINTS: configure explicit replay limits and detect every capacity-driven eviction, returning a visible error instead of declaring truncated output complete. Add regression coverage exceeding the cell and byte limits independently.
- **BR-5** [Critical] `replay-admission-matches-emulator-limits` Accepted recording dimensions can panic during emulator construction
  cmd/internal/scrollbackcmd/bounded.go:38–43 accepts 1000×1000, but replayData calls vt.NewEmulator, whose default maximum is 262,144 cells. An initial 1000×1000 resize reproducibly panics with "vt: invalid geometry"; later oversized-area resizes are silently rejected by Emulator.Resize. ARCH-SECURE and ARCH-CONSTRAINTS: align admission with explicit emulator geometry limits, use error-returning construction and resize operations, and test both initial and subsequent resize failures.

## Round 3 — 2026-10-12T21:10:48-07:00 (codex) — passed

### Disposed

- BR-4 — addressed — Explicit history limits and monotonic capacity-loss accounting prevent silent truncation. Removing accounting fails independent cell/byte, rejected-row, erase-after-loss, and resize-loss regressions.
- BR-5 — addressed — Screen-area admission and checked emulator construction/resizing reject invalid geometry. Removing enforcement reproduces the initial panic and silently ignored later resize; regression tests fail.
- BR-1 — addressed — Exact verified-park identity controls descriptor reuse; successive-source and failed-launch retry tests pass.
- BR-2 — addressed — The plan's M1 file map and Revisions identify previous_session.go, peek_selection.go, bounded.go, and run.go as implemented owners and distinguish unchanged dependencies.
- BR-3 — addressed — Controlled collection during exact-capture replay verifies protection and subsequent lease release on success and failure.

## Round 4 — 2026-10-12T23:01:40-07:00 (codex) — BLOCKED

### Raised

- **BR-6** [Critical] `replacement-recovery-preserves-unconfirmed-effects` Resume can discard a surviving replacement's pending start identity
  ARCH-ORDER / ARCH-PURPOSE: cmd/internal/couchcore/resume.go:459 checks incarnation.LaunchProfile but misses incarnation.Start.LaunchProfile before registration or unknown promotion. An isolated regression with dead supervisor/helper and a surviving target terminal remained pending after startup reconciliation, then resume erased the target claim (revision 7 to 10, incarnations empty) and returned resume-session-gone. Cover every pending/completed replacement representation before lifecycle cleanup and add interruption-sequence tests asserting retained identity and permitted effects.
- **BR-7** [Critical] `core-concepts-match-implementation` The active core-concepts table claims a resolver change absent from the implementation
  ARCH-PURE traceability: workshop/plans/000517-agent-switch-plan.md:872 claims LaunchProfileResolution gained an explicit configured/missing result, but cmd/internal/couchcore/launchprofile.go:52 is unchanged and the check lives in switchagent.go:241. This is the 2nd finding in family core-concepts-match-implementation. Earlier rounds fixed instances; do not fix only this row. Apply a rule requiring every active row's owner, classification and change status to match the implementation against a named baseline, sweep the table, and document the correction under Revisions. One concrete entity contradiction was verified.
- **BR-8** [Minor] `documentation-referents-match-implementation` Atlas still names a deleted orientation-pruning helper
  atlas/couch.md:1330 refers to dropOrientationLocked, removed from cmd/internal/couchtty/console_continuation.go in this range. Update the passage to describe reconcileContinuationOrientationLocked.

## Round 5 — 2026-10-12T23:34:55-07:00 (codex) — BLOCKED

### Disposed

- BR-6 — addressed — resume.go checks pending profile/binding authority and rechecks before cleanup. Three isolated mutations each fail TestInterruptedSwitchResumePreservesTargetAuthority; restored tests pass.
- BR-7 — addressed — Plan lines 1020–1088 replace the active inventory with a baseline-specific ownership/classification rule and complete table. Checked owners and unchanged dependencies against M1 37707f73b; launchprofile.go remains correctly classified unchanged.
- BR-8 — addressed — atlas/couch.md:1330 now names reconcileContinuationOrientationLocked, matching the implementation at console_continuation.go:152 and removal of dropOrientationLocked.

### Raised

- **BR-9** [Important] `pure-core-tested-without-io` Pure previous-tty projection and selection lack direct tests without IO
  ARCH-PURE: previous_session_test.go:62 constructs filesystem-backed Couch state through newTestEnv/createParkedThreadInCouch; peek_previous_test.go exercises SelectPeekSource only through integration paths. Add value-only tests for previousSessionEvidence and SelectPeekSource covering missing park/capture, exact source selection, malformed evidence, and defensive copying. The functions themselves are pure; retain that classification.

## Round 6 — 2026-10-12T23:44:59-07:00 (codex) — passed

### Disposed

- BR-9 — addressed — previous_session_pure_test.go:13 and peek_selection_test.go:29,52,80 use value-only fixtures covering missing evidence, exact selection, malformed evidence and defensive copying. Removing capture copying, binding copying or validation independently fails these tests; restored code passes.
- BR-6 — addressed — Prior disposition retained. resume.go checks pending replacement authority and rechecks absence before cleanup; interrupted-switch recovery sequence tests pass.
- BR-7 — addressed — Prior disposition retained. The active plan inventory identifies ResolveLaunchProfile as unchanged and known-configuration admission as integration code in switchagent.go; comparison against the named M1 baseline confirms the unchanged dependencies.
- BR-8 — addressed — Prior disposition retained. atlas/couch.md now names reconcileContinuationOrientationLocked, which exists in console_continuation.go and owns continuation prompt pruning.

## Round 7 — 2026-10-13T07:41:22-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Successive-source tests pass; removing source-identity comparison fails both in-switch and separately parked cases.
- BR-2 — addressed — The authoritative BR-7 inventory supersedes historical tables and identifies implemented owners and classifications.
- BR-3 — addressed — Lease tests exercise collection during replay and release on success/error; removing lease closure makes both cases fail.
- BR-4 — addressed — Capacity-loss tests cover rows, cells, bytes and subsequent history clearing; disabling loss checks makes them fail.
- BR-5 — addressed — Geometry tests pass; replacing checked emulator construction makes the geometry regression test fail.
- BR-6 — addressed — Interrupted-target sequence tests pass; bypassing pending Start authority causes record-preservation assertions to fail.
- BR-7 — addressed — The active inventory correctly calls ResolveLaunchProfile unchanged and places known-configuration admission in switchagent.go; comparison against the named M1 baseline confirms this.
- BR-8 — addressed — Atlas now names reconcileContinuationOrientationLocked, matching the existing pruning implementation and call sites in console_continuation.go.
- BR-9 — addressed — Direct projection/selection tests use values without IO; removing the selected binding copy fails the aliasing assertion.

## Open findings

(none — every finding has been disposed)
