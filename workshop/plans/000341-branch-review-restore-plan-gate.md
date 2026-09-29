---
gate: plan-quality
issue: 341
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-28T20:58:16-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify recovery that survives normal pane exit
          detail: 'ARCH-FUNERAL and ARCH-PURPOSE: existing swap/undo behavior does not preserve unsaved text through normal exit when document saving is suppressed. Choose an explicit persistence and recovery mechanism, including ownership, removal, and write-failure behavior; verify recovery after process exit without modifying the newly checked-out document.'
          family: recovery-durability-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace enumerated test cases with function-specific strategies
          detail: Tasks 1–3 contain prose test-case inventories prohibited by this gate. Compress them into named unit-test surfaces, including classifyIdentity, transition, and validate_context, with one adversarial-class and mechanical-guard strategy per risky function; retain end-to-end acceptance and mutation checks.
          family: function-level-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-28T20:59:49-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Explicit atomic recovery storage survives orderly exit, blocks quit on persistence failure, bounds residue, and defines recovery and removal without writing the newly checked-out document.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Named risky functions now have adversarial-class and mechanical-guard strategies; tasks reference these strategies while retaining end-to-end acceptance and mutation checks.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-28T21:01:38-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Atomic recovery snapshots, synchronous quit protection, bounded storage and explicit recovery/removal ownership replace reliance on swap or undo.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: The verification strategy names risky functions and gives adversarial input classes with mechanical guards.
          round: 3
      blocked: false
content_hash: 9614ee18b83ad0b810fa9903ea5e4d0d27c2241db0f5cdc6668932093a48abeb
---

# Gate ledger — pair#341 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T20:58:16-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `recovery-durability-contract` Specify recovery that survives normal pane exit
  ARCH-FUNERAL and ARCH-PURPOSE: existing swap/undo behavior does not preserve unsaved text through normal exit when document saving is suppressed. Choose an explicit persistence and recovery mechanism, including ownership, removal, and write-failure behavior; verify recovery after process exit without modifying the newly checked-out document.
- **PQ-2** [Important] `function-level-test-strategy` Replace enumerated test cases with function-specific strategies
  Tasks 1–3 contain prose test-case inventories prohibited by this gate. Compress them into named unit-test surfaces, including classifyIdentity, transition, and validate_context, with one adversarial-class and mechanical-guard strategy per risky function; retain end-to-end acceptance and mutation checks.

## Round 2 — 2026-09-28T20:59:49-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Explicit atomic recovery storage survives orderly exit, blocks quit on persistence failure, bounds residue, and defines recovery and removal without writing the newly checked-out document.
- PQ-2 — addressed — Named risky functions now have adversarial-class and mechanical-guard strategies; tasks reference these strategies while retaining end-to-end acceptance and mutation checks.

## Round 3 — 2026-09-28T21:01:38-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Atomic recovery snapshots, synchronous quit protection, bounded storage and explicit recovery/removal ownership replace reliance on swap or undo.
- PQ-2 — addressed — The verification strategy names risky functions and gives adversarial input classes with mechanical guards.

## Open findings

(none — every finding has been disposed)
