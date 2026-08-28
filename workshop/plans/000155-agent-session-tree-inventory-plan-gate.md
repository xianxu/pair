---
gate: plan-quality
issue: 155
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-28T12:35:27-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define the durable launch generation and offline-recovery boundary
          detail: The plan keeps its launch baseline in watcher memory but promises post-progress/pre-ledger crash recovery. Specify the typed ledger lifecycle, supersession and join rules, and deterministic post-launch log/native projection without timestamp authority (ARCH-PURE, ARCH-PURPOSE).
          family: ledger-launch-generation
          round: 1
        - id: PQ-2
          severity: Important
          title: Give all ledger appends one cross-process serialization owner
          detail: Launcher and watcher currently perform independent read-modify-replace appends, so concurrent writes can lose an establishment row. Name the shared locked/durable append seam and test real concurrent append and failure behavior (ARCH-DRY, ARCH-MOCK).
          family: ledger-append-serialization
          round: 1
        - id: PQ-3
          severity: Important
          title: Define the Pair-log durability prerequisite for recoverable rounds
          detail: Current send behavior submits even when markdown-log persistence fails, although offline recovery treats that log as authoritative evidence. Make durable logging a prerequisite or narrow the stated recovery guarantee (ARCH-PURPOSE).
          family: pair-log-durability-before-send
          round: 1
        - id: PQ-4
          severity: Important
          title: Replace enumerated procedural tests with named production-function strategies
          detail: The plan lists case inventories and red-green implementation procedures but does not consistently name the production functions under unit test. Compress each risky surface to the function name plus one adversarial-input class and mechanical guard.
          family: test-plan-abstraction
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-28T12:42:45-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Typed launch records use durable physical ordinals, explicit supersession and binding joins, and content-free Pair-log/native watermarks for deterministic offline suffix recovery.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: LedgerStore is the sole cross-process append owner, with exclusive locking, ordinal allocation under lock, append/fsync semantics, and real concurrent/failure tests.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: SessionLogStore makes locked atomic markdown-log persistence and directory fsync a prerequisite to submission, preserving the draft and sending nothing on failure.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: The Risky Function Test Strategies table names each production function, adversarial input class, and mechanical guard while leaving individual cases to executable fixtures and fuzz seeds.
          round: 2
      blocked: false
content_hash: dd3b7a4d25d8212ea02cfb5eb90183db0555f98f4c58687020a50b8aace6cf10
---

# Gate ledger — pair#155 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T12:35:27-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `ledger-launch-generation` Define the durable launch generation and offline-recovery boundary
  The plan keeps its launch baseline in watcher memory but promises post-progress/pre-ledger crash recovery. Specify the typed ledger lifecycle, supersession and join rules, and deterministic post-launch log/native projection without timestamp authority (ARCH-PURE, ARCH-PURPOSE).
- **PQ-2** [Important] `ledger-append-serialization` Give all ledger appends one cross-process serialization owner
  Launcher and watcher currently perform independent read-modify-replace appends, so concurrent writes can lose an establishment row. Name the shared locked/durable append seam and test real concurrent append and failure behavior (ARCH-DRY, ARCH-MOCK).
- **PQ-3** [Important] `pair-log-durability-before-send` Define the Pair-log durability prerequisite for recoverable rounds
  Current send behavior submits even when markdown-log persistence fails, although offline recovery treats that log as authoritative evidence. Make durable logging a prerequisite or narrow the stated recovery guarantee (ARCH-PURPOSE).
- **PQ-4** [Important] `test-plan-abstraction` Replace enumerated procedural tests with named production-function strategies
  The plan lists case inventories and red-green implementation procedures but does not consistently name the production functions under unit test. Compress each risky surface to the function name plus one adversarial-input class and mechanical guard.

## Round 2 — 2026-08-28T12:42:45-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Typed launch records use durable physical ordinals, explicit supersession and binding joins, and content-free Pair-log/native watermarks for deterministic offline suffix recovery.
- PQ-2 — addressed — LedgerStore is the sole cross-process append owner, with exclusive locking, ordinal allocation under lock, append/fsync semantics, and real concurrent/failure tests.
- PQ-3 — addressed — SessionLogStore makes locked atomic markdown-log persistence and directory fsync a prerequisite to submission, preserving the draft and sending nothing on failure.
- PQ-4 — addressed — The Risky Function Test Strategies table names each production function, adversarial input class, and mechanical guard while leaving individual cases to executable fixtures and fuzz seeds.

## Open findings

(none — every finding has been disposed)
