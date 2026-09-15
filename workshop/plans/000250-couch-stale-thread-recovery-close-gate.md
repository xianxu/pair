---
gate: boundary-review
issue: 250
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T18:16:09-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Archive rejects empty threads that never published a session binding
          detail: detach.go:251 unconditionally invokes recovery observation, whose missing-binding error prevents archiving a readable, unoccupied thread after pre-session launch failure. Preserve the existing never-bound, non-signalling archive contract and add a Couch.ArchiveThread regression without a session-index entry. ARCH-PURPOSE.
          family: archive-preserves-unbound-escape
          round: 1
        - id: BR-2
          severity: Important
          title: Recovery retry and generation lookup discard caller deadlines
          detail: continuation_recovery.go:269 and switchcontext.go:325 reach non-context session observation using context.Background, allowing a short-deadline operation to wait for Zellij's independent five-second timeout. Thread the caller context through both paths and test cancellation at these boundaries. ARCH-CONSTRAINTS.
          family: observations-preserve-caller-deadlines
          round: 1
      blocked: true
---

# Gate ledger — pair#250 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T18:16:09-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `archive-preserves-unbound-escape` Archive rejects empty threads that never published a session binding
  detach.go:251 unconditionally invokes recovery observation, whose missing-binding error prevents archiving a readable, unoccupied thread after pre-session launch failure. Preserve the existing never-bound, non-signalling archive contract and add a Couch.ArchiveThread regression without a session-index entry. ARCH-PURPOSE.
- **BR-2** [Important] `observations-preserve-caller-deadlines` Recovery retry and generation lookup discard caller deadlines
  continuation_recovery.go:269 and switchcontext.go:325 reach non-context session observation using context.Background, allowing a short-deadline operation to wait for Zellij's independent five-second timeout. Thread the caller context through both paths and test cancellation at these boundaries. ARCH-CONSTRAINTS.

## Open findings

- **BR-1** [Critical] `archive-preserves-unbound-escape` Archive rejects empty threads that never published a session binding
- **BR-2** [Important] `observations-preserve-caller-deadlines` Recovery retry and generation lookup discard caller deadlines
