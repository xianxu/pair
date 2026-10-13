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

## Open findings

- **BR-4** [Critical] `full-replay-requires-loss-detection` Full replay silently evicts history and reports the remaining tail as complete
- **BR-5** [Critical] `replay-admission-matches-emulator-limits` Accepted recording dimensions can panic during emulator construction
