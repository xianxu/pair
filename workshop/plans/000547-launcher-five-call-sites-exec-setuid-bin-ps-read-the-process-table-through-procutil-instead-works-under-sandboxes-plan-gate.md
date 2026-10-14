---
gate: plan-quality
issue: 547
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-14T00:09:58-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Name the risky functions and give each a mechanical test strategy.
          detail: The plan lists test topics and files but does not name the new native decoders, enriched snapshot, shared server projection, or ancestry projection functions. For ARCH-PURE and ARCH-ORDER, replace the prose case inventory with one strategy line per risky function, naming the adversarial input or event class and the guard or invariant the tests enforce.
          family: named-function-test-strategy
          round: 1
        - id: PQ-2
          severity: Minor
          title: State the behavioral and lifecycle non-goals.
          detail: 'The plan has no explicit non-goals section. Clarify the intended boundary of behavior preservation, including argv whitespace matching and ownership of lifecycle authorization changes coordinated with #528, so the native-reader migration does not silently expand into behavior redesign.'
          family: explicit-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T00:11:25-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Named risky functions now have mechanical strategies covering adversarial inputs and evidence, identity and cancellation invariants.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: 'Explicit non-goals preserve whitespace matching and legacy string semantics while assigning lifecycle authorization redesign to #528.'
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-14T00:13:33-07:00"
      agent: codex
      blocked: false
content_hash: 4045e35351f12fb2b67088d64fc610f9063a45cc5abee60bace666c851283bea
---

# Gate ledger — pair#547 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T00:09:58-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `named-function-test-strategy` Name the risky functions and give each a mechanical test strategy.
  The plan lists test topics and files but does not name the new native decoders, enriched snapshot, shared server projection, or ancestry projection functions. For ARCH-PURE and ARCH-ORDER, replace the prose case inventory with one strategy line per risky function, naming the adversarial input or event class and the guard or invariant the tests enforce.
- **PQ-2** [Minor] `explicit-non-goals` State the behavioral and lifecycle non-goals.
  The plan has no explicit non-goals section. Clarify the intended boundary of behavior preservation, including argv whitespace matching and ownership of lifecycle authorization changes coordinated with #528, so the native-reader migration does not silently expand into behavior redesign.

## Round 2 — 2026-10-14T00:11:25-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Named risky functions now have mechanical strategies covering adversarial inputs and evidence, identity and cancellation invariants.
- PQ-2 — addressed — Explicit non-goals preserve whitespace matching and legacy string semantics while assigning lifecycle authorization redesign to #528.

## Round 3 — 2026-10-14T00:13:33-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
