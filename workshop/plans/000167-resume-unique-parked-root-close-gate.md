---
gate: boundary-review
issue: 167
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-01T18:30:07-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Restart acceptance bypasses interactive dispatch and console attachment
          detail: The test calls dispatchInteractiveStart directly, so reverting the production runTypedOperation routing would not make it fail. Exercise the public launch path over reconstructed persisted state and assert the resumed identity reaches initial home attachment.
          family: production-path-acceptance
          round: 1
        - id: BR-2
          severity: Important
          title: README still documents the pre-change Leave Couch restart behavior
          detail: README.md says bare couch reopens the switcher for manual Enter, but the implementation now automatically resumes one exact eligible parked root. Document automatic unique resume and the zero-or-ambiguous new-root fallback.
          family: user-surface-documentation
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-01T18:39:12-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The replacement test traverses production interactive routing and initial console attachment; reverting the routing would attach a new root and fail its exact parked-address assertion.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: README now describes automatic unique resume, but its immediately following “Every TUI start allocates” statement still contradicts resumed startup.
          round: 2
      blocked: true
---

# Gate ledger — pair#167 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T18:30:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `production-path-acceptance` Restart acceptance bypasses interactive dispatch and console attachment
  The test calls dispatchInteractiveStart directly, so reverting the production runTypedOperation routing would not make it fail. Exercise the public launch path over reconstructed persisted state and assert the resumed identity reaches initial home attachment.
- **BR-2** [Important] `user-surface-documentation` README still documents the pre-change Leave Couch restart behavior
  README.md says bare couch reopens the switcher for manual Enter, but the implementation now automatically resumes one exact eligible parked root. Document automatic unique resume and the zero-or-ambiguous new-root fallback.

## Round 2 — 2026-09-01T18:39:12-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The replacement test traverses production interactive routing and initial console attachment; reverting the routing would attach a new root and fail its exact parked-address assertion.
- BR-2 — not-addressed — README now describes automatic unique resume, but its immediately following “Every TUI start allocates” statement still contradicts resumed startup.

## Open findings

- **BR-2** [Important] `user-surface-documentation` README still documents the pre-change Leave Couch restart behavior
