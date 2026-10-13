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

## Open findings

- **BR-1** [Critical] `evidence-reuse-requires-source-identity` Switching an already-parked source retains an older incarnation's evidence
- **BR-2** [Critical] `core-concepts-match-implementation` M1 core-concepts locations and modification claims contradict the diff
- **BR-3** [Important] `retention-ordering-regression-coverage` Exact capture reader lacks the promised lease and collection ordering tests
