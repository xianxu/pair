---
gate: plan-quality
issue: 552
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-14T13:42:39-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace prose test inventories with named functions and adversarial strategies
          detail: The plan enumerates test cases at lines 97, 108 and 112 but does not name the pure lease-transition or report-validation functions to be unit-tested. Name those functions and the existing parser surface, then give one adversarial-input class and mechanical guard per risky function; compress the inventories while preserving root reproductions as acceptance evidence (ARCH-PURE).
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Important
          title: Define occurrence ordering before promising bounded duplicate suppression
          detail: Lines 87–89 promise duplicate suppression with at most one retained occurrence per connected binding, but define no ordering or replay window for episode IDs. Specify a monotonic per-binding occurrence sequence with a high-water rejection rule, or another bounded protocol that prevents an older duplicate from repeating effects after a newer occurrence; verify that rule through controllable report ordering (ARCH-ORDER, ARCH-FUNERAL).
          family: bounded-deduplication-order
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T13:44:28-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The superseding revision names risky functions and adversarial strategies with mechanical guards while preserving root reproductions as acceptance evidence.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Monotonic sequences and atomic high-water rejection cover reordered duplicates; reconnect retention, incarnation replacement and saturation refusal bound suppression state.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-14T13:45:35-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The superseding revision names risky functions and pairs each with an adversarial input class and mechanical guard.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Monotonic sequences, atomic high-water admission, reconnect retention and explicit saturation refusal define bounded duplicate suppression.
          round: 3
      blocked: false
content_hash: fcbf0a2a168b792ca507504861a35add096417293ac2a5298505b15611c05560
---

# Gate ledger — pair#552 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T13:42:39-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace prose test inventories with named functions and adversarial strategies
  The plan enumerates test cases at lines 97, 108 and 112 but does not name the pure lease-transition or report-validation functions to be unit-tested. Name those functions and the existing parser surface, then give one adversarial-input class and mechanical guard per risky function; compress the inventories while preserving root reproductions as acceptance evidence (ARCH-PURE).
- **PQ-2** [Important] `bounded-deduplication-order` Define occurrence ordering before promising bounded duplicate suppression
  Lines 87–89 promise duplicate suppression with at most one retained occurrence per connected binding, but define no ordering or replay window for episode IDs. Specify a monotonic per-binding occurrence sequence with a high-water rejection rule, or another bounded protocol that prevents an older duplicate from repeating effects after a newer occurrence; verify that rule through controllable report ordering (ARCH-ORDER, ARCH-FUNERAL).

## Round 2 — 2026-10-14T13:44:28-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The superseding revision names risky functions and adversarial strategies with mechanical guards while preserving root reproductions as acceptance evidence.
- PQ-2 — addressed — Monotonic sequences and atomic high-water rejection cover reordered duplicates; reconnect retention, incarnation replacement and saturation refusal bound suppression state.

## Round 3 — 2026-10-14T13:45:35-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The superseding revision names risky functions and pairs each with an adversarial input class and mechanical guard.
- PQ-2 — addressed — Monotonic sequences, atomic high-water admission, reconnect retention and explicit saturation refusal define bounded duplicate suppression.

## Open findings

(none — every finding has been disposed)
