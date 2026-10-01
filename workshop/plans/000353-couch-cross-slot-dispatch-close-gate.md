---
gate: boundary-review
issue: 353
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-30T21:27:35-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Orientation and peer delivery can paste into the same composer and submit combined text
          detail: cmd/internal/wrapcmd/peer_delivery.go:199 and orientation.go:111 serialize callbacks without retaining ownership across paste/render/submit. A deterministic scratch regression produced orientation paste, peer paste before repaint, then orientation submission of both. Add shared transaction ownership and controlled tests for both arrival orders. ARCH-ORDER, ARCH-DRY.
          family: automatic-input-transaction-ownership
          round: 1
        - id: BR-2
          severity: Critical
          title: The frame limit rejects valid message bodies and supported actor inventories
          detail: cmd/internal/couchmessage/transport.go:199-204 uses JSON encoding with a 32768-byte cap. Scratch regressions measured 49285 bytes for a valid 8192-byte less-than-character body and 39016 bytes for 128 ordinary actor records. Response write errors at line 180 become EOF. Align serialized bounds across sends, endpoint commits, receipts and discovery; test maximum payloads and inventories. ARCH-CONSTRAINTS.
          family: wire-capacity-matches-domain-bounds
          round: 1
        - id: BR-3
          severity: Important
          title: Incarnation-specific wrapper sockets have no cleanup after process crashes
          detail: cmd/internal/wrapcmd/peer_runtime.go:128-138 creates a new socket per incarnation; couchmessage/transport.go:152 removes it only during graceful teardown. Supervisor startup cleans only the broker socket, and reconciliation merely disconnects actors. Add proven-dead-owner cleanup and crash regressions preserving live and replacement handles. ARCH-FUNERAL.
          family: runtime-artifact-crash-cleanup
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-30T21:36:44-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Shared transaction ownership and both automatic-input regression tests cover competing arrival orders, cancellation, submission, and fresh-empty-repaint release.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Derived frame bounds and transport_bounds_test.go exercise maximum messages, commits, receipts, and actor inventories through sockets; overflow now reports uncertainty explicitly.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Startup collects proven-dead PID-owned sockets; real process-crash and controlled replacement tests verify cleanup while preserving live, uncertain, and replacement owners.
          round: 2
      findings:
        - id: BR-4
          severity: Critical
          title: Core concepts names a transport Client that does not exist
          detail: workshop/plans/000353-couch-cross-slot-dispatch-plan.md:123 lists Client, but transport.go:212 implements Call instead. Append a revision superseding the mapping with Server/Call. This documentation-only discrepancy is Critical under the explicitly requested Core concepts consistency rule.
          family: core-concept-mappings-match-implementation
          round: 2
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#353 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T21:27:35-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `automatic-input-transaction-ownership` Orientation and peer delivery can paste into the same composer and submit combined text
  cmd/internal/wrapcmd/peer_delivery.go:199 and orientation.go:111 serialize callbacks without retaining ownership across paste/render/submit. A deterministic scratch regression produced orientation paste, peer paste before repaint, then orientation submission of both. Add shared transaction ownership and controlled tests for both arrival orders. ARCH-ORDER, ARCH-DRY.
- **BR-2** [Critical] `wire-capacity-matches-domain-bounds` The frame limit rejects valid message bodies and supported actor inventories
  cmd/internal/couchmessage/transport.go:199-204 uses JSON encoding with a 32768-byte cap. Scratch regressions measured 49285 bytes for a valid 8192-byte less-than-character body and 39016 bytes for 128 ordinary actor records. Response write errors at line 180 become EOF. Align serialized bounds across sends, endpoint commits, receipts and discovery; test maximum payloads and inventories. ARCH-CONSTRAINTS.
- **BR-3** [Important] `runtime-artifact-crash-cleanup` Incarnation-specific wrapper sockets have no cleanup after process crashes
  cmd/internal/wrapcmd/peer_runtime.go:128-138 creates a new socket per incarnation; couchmessage/transport.go:152 removes it only during graceful teardown. Supervisor startup cleans only the broker socket, and reconciliation merely disconnects actors. Add proven-dead-owner cleanup and crash regressions preserving live and replacement handles. ARCH-FUNERAL.

## Round 2 — 2026-09-30T21:36:44-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Shared transaction ownership and both automatic-input regression tests cover competing arrival orders, cancellation, submission, and fresh-empty-repaint release.
- BR-2 — addressed — Derived frame bounds and transport_bounds_test.go exercise maximum messages, commits, receipts, and actor inventories through sockets; overflow now reports uncertainty explicitly.
- BR-3 — addressed — Startup collects proven-dead PID-owned sockets; real process-crash and controlled replacement tests verify cleanup while preserving live, uncertain, and replacement owners.

### Raised

- **BR-4** [Critical] `core-concept-mappings-match-implementation` Core concepts names a transport Client that does not exist
  workshop/plans/000353-couch-cross-slot-dispatch-plan.md:123 lists Client, but transport.go:212 implements Call instead. Append a revision superseding the mapping with Server/Call. This documentation-only discrepancy is Critical under the explicitly requested Core concepts consistency rule.

## Open findings

- **BR-4** [Critical] `core-concept-mappings-match-implementation` Core concepts names a transport Client that does not exist
