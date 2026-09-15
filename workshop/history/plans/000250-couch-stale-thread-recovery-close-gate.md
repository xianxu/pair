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
    - "n": 2
      timestamp: "2026-09-14T18:24:45-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: detach.go now permits revision-fenced, non-signalling archive only for an empty record with explicitly absent binding. TestCouchArchiveNeverBoundThreadDoesNotSignal exercises the previously rejected case; companion tests reject unreadable indexes, retained requests and concurrent changes.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Retry and final archive observation now preserve caller context; production readiness composition installs SessionContext. Cancellation regressions exercise retry, generation, registration and status reads, and passed.
          round: 2
      findings:
        - id: BR-3
          severity: Important
          title: Settled unknown-target recovery bypasses the incarnation transition model
          detail: cmd/internal/couchcore/continuation_recovery.go:87 directly assigns IncarnationLive inside UpdateExistingThread, committing an intermediate state before reconcileRecoveryHelper retires it. ARCH-ORDER requires this transition to belong to the pure model. Express exact-receipt reconciliation as a named transition with a revision-checked store operation; test interruption between reconciliation and attachment.
          family: lifecycle-transitions-through-owned-model
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-14T18:35:46-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: ReconcileRegisteredTarget in starttransaction.go:249 replaces the synthetic Live intermediate state through ThreadStore.ReconcileRegisteredTarget. continuation_recovery_test.go:329 exercises interruption after retirement, same-attempt reattachment, lost receipt, revived helper, and revision conflict. Both new regression tests passed independently. The interruption assertion rejects the previous implementation's persisted Live intermediate state.
          round: 3
        - id: BR-1
          disposition: addressed
          note: The narrow missing-binding archive escape remains guarded against unreadable indexes, retained requests, and concurrent replacement; the affected archive tests passed.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Production session and generation observations retain caller context; cancellation regressions and affected package tests passed.
          round: 3
      blocked: false
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

## Round 2 — 2026-09-14T18:24:45-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — detach.go now permits revision-fenced, non-signalling archive only for an empty record with explicitly absent binding. TestCouchArchiveNeverBoundThreadDoesNotSignal exercises the previously rejected case; companion tests reject unreadable indexes, retained requests and concurrent changes.
- BR-2 — addressed — Retry and final archive observation now preserve caller context; production readiness composition installs SessionContext. Cancellation regressions exercise retry, generation, registration and status reads, and passed.

### Raised

- **BR-3** [Important] `lifecycle-transitions-through-owned-model` Settled unknown-target recovery bypasses the incarnation transition model
  cmd/internal/couchcore/continuation_recovery.go:87 directly assigns IncarnationLive inside UpdateExistingThread, committing an intermediate state before reconcileRecoveryHelper retires it. ARCH-ORDER requires this transition to belong to the pure model. Express exact-receipt reconciliation as a named transition with a revision-checked store operation; test interruption between reconciliation and attachment.

## Round 3 — 2026-09-14T18:35:46-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — ReconcileRegisteredTarget in starttransaction.go:249 replaces the synthetic Live intermediate state through ThreadStore.ReconcileRegisteredTarget. continuation_recovery_test.go:329 exercises interruption after retirement, same-attempt reattachment, lost receipt, revived helper, and revision conflict. Both new regression tests passed independently. The interruption assertion rejects the previous implementation's persisted Live intermediate state.
- BR-1 — addressed — The narrow missing-binding archive escape remains guarded against unreadable indexes, retained requests, and concurrent replacement; the affected archive tests passed.
- BR-2 — addressed — Production session and generation observations retain caller context; cancellation regressions and affected package tests passed.

## Open findings

(none — every finding has been disposed)
