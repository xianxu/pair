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
    - "n": 2
      timestamp: "2026-10-14T01:31:29-07:00"
      dispatched: "2026-10-14T01:25:09-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: hub_move.go:159-176 confirms unchanged claim snapshots. Removing this correction makes both TestHubMoveRetryConfirmsAlreadyVisibleClaims cases fail; pinned-head tests pass.
          round: 2
        - id: BR-2
          disposition: addressed
          note: supervision.go:187 checks retained admission ownership alongside current ownership. Removing the guard makes TestHubSupervisionDelayedReplyPreservesAdmissionOwnership/foreign-to-own fail; pinned-head tests pass.
          round: 2
        - id: BR-3
          disposition: addressed
          note: atlas/couch.md:3593-3595 now identifies concurrent queue workers, dispatchHubOperation's effect gate, and the adoption chain; hubs.go implements both ordering mechanisms.
          round: 2
      findings:
        - id: BR-4
          severity: Important
          title: Secondary-hub deliveries disappear from supervision facts
          detail: 'cmd/internal/couchcmd/message_service.go:1093-1102 still uses hubBinding(), which resolves only authority.hub (main), to admit hub_delivery facts. A disposable production-adapter test confirmed that main receives one fact while a confirmed secondary-hub delivery receives zero. ARCH-PURPOSE: classify every active hub through the shared ownership projection, retaining recipient evidence across delivery, and test main, secondary, and non-hub recipients.'
          family: ownership-consumers-share-projection
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-14T01:43:01-07:00"
      dispatched: "2026-10-14T01:35:29-07:00"
      agent: codex
      dispose:
        - id: BR-4
          disposition: addressed
          note: message_service.go:1095 uses shared isHubBinding classification before delivery. TestSupervisionFactsAllHubRecipientsPreserveDeliveryClassification covers main, secondary and worker recipients, including disconnects. Restoring the old hubBinding comparison makes both secondary cases fail with zero facts instead of one; pinned HEAD passes.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 9effccd231e0121e6c54cded1017eb3979f3db26
      blocked: false
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

## Round 2 — 2026-10-14T01:31:29-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — hub_move.go:159-176 confirms unchanged claim snapshots. Removing this correction makes both TestHubMoveRetryConfirmsAlreadyVisibleClaims cases fail; pinned-head tests pass.
- BR-2 — addressed — supervision.go:187 checks retained admission ownership alongside current ownership. Removing the guard makes TestHubSupervisionDelayedReplyPreservesAdmissionOwnership/foreign-to-own fail; pinned-head tests pass.
- BR-3 — addressed — atlas/couch.md:3593-3595 now identifies concurrent queue workers, dispatchHubOperation's effect gate, and the adoption chain; hubs.go implements both ordering mechanisms.

### Raised

- **BR-4** [Important] `ownership-consumers-share-projection` Secondary-hub deliveries disappear from supervision facts
  cmd/internal/couchcmd/message_service.go:1093-1102 still uses hubBinding(), which resolves only authority.hub (main), to admit hub_delivery facts. A disposable production-adapter test confirmed that main receives one fact while a confirmed secondary-hub delivery receives zero. ARCH-PURPOSE: classify every active hub through the shared ownership projection, retaining recipient evidence across delivery, and test main, secondary, and non-hub recipients.

## Round 3 — 2026-10-14T01:43:01-07:00 (codex) — passed

### Disposed

- BR-4 — addressed — message_service.go:1095 uses shared isHubBinding classification before delivery. TestSupervisionFactsAllHubRecipientsPreserveDeliveryClassification covers main, secondary and worker recipients, including disconnects. Restoring the old hubBinding comparison makes both secondary cases fail with zero facts instead of one; pinned HEAD passes.

## Open findings

(none — every finding has been disposed)
