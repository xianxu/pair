---
gate: boundary-review
issue: 546
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-14T01:18:56-07:00"
      dispatched: "2026-10-14T01:12:54-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Persistent retries can report success without confirming durability
          detail: 'cmd/internal/couchcmd/hub_move.go:152-159 replans from visible bytes, dropping already-satisfied edits. If publication renamed the file but directory sync failed, retry supplies no edit to hubClaimFiles.Apply, bypassing its reconciliation sync and reporting persisted success. Fault injection reproduced success while SyncDirectory was still failing. ARCH-ORDER: reconcile durability even when no byte changes remain.'
          family: durable-success-requires-confirmation
          round: 1
        - id: BR-2
          severity: Critical
          title: Reassignment can turn a cross-hub message into reply evidence
          detail: 'cmd/internal/couchcmd/supervision.go:157-164 checks ownership at delivery completion without retaining admission ownership. A foreign hub sends an explicit message, the worker moves to that hub, and delayed delivery clears the worker''s unanswered report, suppressing nudges and escalation. A controlled sequence reproduced this. ARCH-ORDER: retain recipient ownership evidence at admission and prevent cross-hub messages from becoming own-hub replies.'
          family: reply-evidence-preserves-admission-ownership
          round: 1
        - id: BR-3
          severity: Minor
          title: Atlas still describes the operation queue as serial
          detail: atlas/couch.md:3592 and 3600 attribute serialization to the queue, although it has multiple workers. Document dispatchHubOperation's effect gate and the explicit startup adoption chain as the ordering mechanisms. ARCH-ORDER.
          family: architecture-docs-match-execution
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#546 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T01:18:56-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `durable-success-requires-confirmation` Persistent retries can report success without confirming durability
  cmd/internal/couchcmd/hub_move.go:152-159 replans from visible bytes, dropping already-satisfied edits. If publication renamed the file but directory sync failed, retry supplies no edit to hubClaimFiles.Apply, bypassing its reconciliation sync and reporting persisted success. Fault injection reproduced success while SyncDirectory was still failing. ARCH-ORDER: reconcile durability even when no byte changes remain.
- **BR-2** [Critical] `reply-evidence-preserves-admission-ownership` Reassignment can turn a cross-hub message into reply evidence
  cmd/internal/couchcmd/supervision.go:157-164 checks ownership at delivery completion without retaining admission ownership. A foreign hub sends an explicit message, the worker moves to that hub, and delayed delivery clears the worker's unanswered report, suppressing nudges and escalation. A controlled sequence reproduced this. ARCH-ORDER: retain recipient ownership evidence at admission and prevent cross-hub messages from becoming own-hub replies.
- **BR-3** [Minor] `architecture-docs-match-execution` Atlas still describes the operation queue as serial
  atlas/couch.md:3592 and 3600 attribute serialization to the queue, although it has multiple workers. Document dispatchHubOperation's effect gate and the explicit startup adoption chain as the ordering mechanisms. ARCH-ORDER.

## Open findings

- **BR-1** [Critical] `durable-success-requires-confirmation` Persistent retries can report success without confirming durability
- **BR-2** [Critical] `reply-evidence-preserves-admission-ownership` Reassignment can turn a cross-hub message into reply evidence
- **BR-3** [Minor] `architecture-docs-match-execution` Atlas still describes the operation queue as serial
