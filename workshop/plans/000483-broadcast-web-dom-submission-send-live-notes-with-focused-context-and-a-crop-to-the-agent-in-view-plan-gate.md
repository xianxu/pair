---
gate: plan-quality
issue: 483
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T20:23:57-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify how the production delivery state crosses the Go/Lua boundary
          detail: 'ARCH-ORDER / ARCH-PURE: The plan says Go Delivery owns transitions, Lua owns draft submission state, and delivery metadata persists in the manifest, but prepare/status/ack does not explain how Lua''s written, indeterminate and dispatched outcomes reach that authority. Name the authoritative transition function for each side, its production invocation path, and the persistence/response ordering that preserves acknowledgment-only recovery; specify conservative recovery when a delivery outcome cannot be recorded.'
          family: authoritative-transition-owner
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named functions and adversarial strategies
          detail: Implementation tasks 1–4 enumerate test cases, contrary to this gate's test-plan contract, while most risky functions remain unnamed. Compress those inventories into one strategy line per risky function, identifying the adversarial input or event-sequence class and mechanical guard; include the actual production transition functions, manifest/PNG validators, capture barrier and guarded acknowledgment.
          family: function-level-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T20:25:33-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: ApplyDelivery and Store.Transition own durable progress; DOMSubmission.send invokes begin before paste and ack after confirmed dispatch, preserving acknowledgment-only recovery and conservative restart reconciliation.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Tasks now name production functions and adversarial strategies covering transitions, manifest and PNG validation, capture ordering, guarded resolution and draft retry invariants.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T20:27:48-07:00"
      agent: codex
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T21:19:38-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Go durable transitions and Lua live-attempt ownership are explicit, including begin-before-paste and acknowledgment-only recovery.
          round: 4
        - id: PQ-2
          disposition: addressed
          note: The plan names test functions and adversarial strategies covering validation, durability failures and event ordering.
          round: 4
      blocked: false
content_hash: a26a33623cce1c4a7b0b2703580453494173f9c8f256b04edf40d48e97be02bb
---

# Gate ledger — pair#483 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T20:23:57-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `authoritative-transition-owner` Specify how the production delivery state crosses the Go/Lua boundary
  ARCH-ORDER / ARCH-PURE: The plan says Go Delivery owns transitions, Lua owns draft submission state, and delivery metadata persists in the manifest, but prepare/status/ack does not explain how Lua's written, indeterminate and dispatched outcomes reach that authority. Name the authoritative transition function for each side, its production invocation path, and the persistence/response ordering that preserves acknowledgment-only recovery; specify conservative recovery when a delivery outcome cannot be recorded.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies
  Implementation tasks 1–4 enumerate test cases, contrary to this gate's test-plan contract, while most risky functions remain unnamed. Compress those inventories into one strategy line per risky function, identifying the adversarial input or event-sequence class and mechanical guard; include the actual production transition functions, manifest/PNG validators, capture barrier and guarded acknowledgment.

## Round 2 — 2026-10-12T20:25:33-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — ApplyDelivery and Store.Transition own durable progress; DOMSubmission.send invokes begin before paste and ack after confirmed dispatch, preserving acknowledgment-only recovery and conservative restart reconciliation.
- PQ-2 — addressed — Tasks now name production functions and adversarial strategies covering transitions, manifest and PNG validation, capture ordering, guarded resolution and draft retry invariants.

## Round 3 — 2026-10-12T20:27:48-07:00 (codex) — passed

## Round 4 — 2026-10-12T21:19:38-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Go durable transitions and Lua live-attempt ownership are explicit, including begin-before-paste and acknowledgment-only recovery.
- PQ-2 — addressed — The plan names test functions and adversarial strategies covering validation, durability failures and event ordering.

## Open findings

(none — every finding has been disposed)
