---
gate: plan-quality
issue: 779
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 5d7f1c6b-d78b-4c98-b2d4-90270b85a3fe
            repository: github.com/xianxu/pair
            issue_id: "000779"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot4/pair
            reviewed_head: 48123129cd2ef42b5a59c50847c54447850069d2
            slot: pair:4
            requested_at: "2026-10-21T23:52:06.362849Z"
          phase: finished
          pid: 71106
          started_at: "2026-10-21T23:52:06.551465Z"
          finished_at: "2026-10-21T23:53:05.784322Z"
          session_id: 01a160e1-185d-79c3-9789-cf2a92808425
          outcome: returned
      "n": 1
      timestamp: "2026-10-21T16:53:06-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Distinguish admitted layout from confirmed painting during output stalls
          detail: 'ARCH-ORDER: Presenter.paintPublication can return nil without painting or notifying taps (presenter.go:471–477, 508–525), after ResizeLayout has resized the child. Define the pending/confirmed layout authority, permitted input while painting is unconfirmed, and reconciliation on repaint; name the enforcing component and a controllable-writer sequence test. “Publication succeeds” and the existing failure path do not cover this outcome.'
          family: publication-outcome-authority
          round: 1
        - id: PQ-2
          severity: Important
          title: Resolve active-chip priority against fixed controls in the one-row fallback
          detail: 'ARCH-PURPOSE / ARCH-SECURE: At host height two, fixed row-zero controls can exhaust the only chrome row before any active chip is drawn. Specify precedence between those controls and the guaranteed positive-width active span, including fail-closed broadcast behavior when LIVE is omitted or clipped; pin the combined constrained-width/height behavior in planStatusRows and the row drawer.'
          family: constrained-priority-contract
          round: 1
        - id: PQ-3
          severity: Minor
          title: Compress test inventories and name the remaining risky functions
          detail: The verification table enumerates individual scenarios instead of consistently stating adversarial classes and mechanical guards. Replace those inventories with one strategy line per named risky function, including the Console publication entry points and broadcast admission/overlay guards; retain the useful complexity counters and stateful ordering seams.
          family: risk-strategy-test-contract
          round: 1
      blocked: true
---

# Gate ledger — pair#779 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T16:53:06-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `publication-outcome-authority` Distinguish admitted layout from confirmed painting during output stalls
  ARCH-ORDER: Presenter.paintPublication can return nil without painting or notifying taps (presenter.go:471–477, 508–525), after ResizeLayout has resized the child. Define the pending/confirmed layout authority, permitted input while painting is unconfirmed, and reconciliation on repaint; name the enforcing component and a controllable-writer sequence test. “Publication succeeds” and the existing failure path do not cover this outcome.
- **PQ-2** [Important] `constrained-priority-contract` Resolve active-chip priority against fixed controls in the one-row fallback
  ARCH-PURPOSE / ARCH-SECURE: At host height two, fixed row-zero controls can exhaust the only chrome row before any active chip is drawn. Specify precedence between those controls and the guaranteed positive-width active span, including fail-closed broadcast behavior when LIVE is omitted or clipped; pin the combined constrained-width/height behavior in planStatusRows and the row drawer.
- **PQ-3** [Minor] `risk-strategy-test-contract` Compress test inventories and name the remaining risky functions
  The verification table enumerates individual scenarios instead of consistently stating adversarial classes and mechanical guards. Replace those inventories with one strategy line per named risky function, including the Console publication entry points and broadcast admission/overlay guards; retain the useful complexity counters and stateful ordering seams.

## Open findings

- **PQ-1** [Important] `publication-outcome-authority` Distinguish admitted layout from confirmed painting during output stalls
- **PQ-2** [Important] `constrained-priority-contract` Resolve active-chip priority against fixed controls in the one-row fallback
- **PQ-3** [Minor] `risk-strategy-test-contract` Compress test inventories and name the remaining risky functions
