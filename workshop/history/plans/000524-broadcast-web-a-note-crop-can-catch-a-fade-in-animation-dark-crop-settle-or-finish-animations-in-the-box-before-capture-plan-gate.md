---
gate: plan-quality
issue: 524
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-13T11:01:06-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define acknowledgment recovery independently of the active draft.
          detail: 'ARCH-ORDER, ARCH-PURPOSE: an old acknowledgment arriving during successor preparation can persist dispatched without scheduling resolution, while status polling cannot resume it. Name the bounded recovery owner and its behavior across busy workers, successor epochs, failed resolution and restart, so genuine acknowledgments resolve captured versions without consuming or blocking newer drafts.'
          family: acknowledgment-recovery-ownership
          round: 1
        - id: PQ-2
          severity: Important
          title: Specify the supersession transaction and legal subsequent events.
          detail: 'ARCH-ORDER, ARCH-PURE: define the pure transition authority for delivery plus supersession, including whether begin is rejected after supersession while genuine ack remains legal. State when supersession becomes durable relative to successor admission/publication, how partial writes or crashes recover, and which lock serializes this with acknowledgment; individual atomic manifest replacement does not establish that contract.'
          family: durable-transition-contract
          round: 1
        - id: PQ-3
          severity: Important
          title: Replace prose case inventories with named function-level test strategies.
          detail: Compress the revised verification checklist into named test surfaces such as dom_submission_state.step, submit_operator_text, submissionManager.request/resolve, and the proposed supersession transition/store operation. Give each risky function one adversarial-input or event-sequence strategy and mechanical guard, using controlled callback/worker ordering and injected persistence failures rather than another enumerated case list.
          family: executable-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T11:04:37-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Independent bounded acknowledgment receipts and shared ack/status scheduling define recovery across successor epochs, busy workers, failures and restart.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Pure transition rules and Store.mu serialization define durable supersession before publication, partial-write recovery and successor-safe retries.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The authoritative function-level strategy table replaces the case inventory with controlled event ordering and injected persistence failures.
          round: 2
      blocked: false
content_hash: 80c3231c235af9ac4c1b01bbed7adb9ceadb3466707fd619eaedb25fcad3d562
---

# Gate ledger — pair#524 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T11:01:06-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `acknowledgment-recovery-ownership` Define acknowledgment recovery independently of the active draft.
  ARCH-ORDER, ARCH-PURPOSE: an old acknowledgment arriving during successor preparation can persist dispatched without scheduling resolution, while status polling cannot resume it. Name the bounded recovery owner and its behavior across busy workers, successor epochs, failed resolution and restart, so genuine acknowledgments resolve captured versions without consuming or blocking newer drafts.
- **PQ-2** [Important] `durable-transition-contract` Specify the supersession transaction and legal subsequent events.
  ARCH-ORDER, ARCH-PURE: define the pure transition authority for delivery plus supersession, including whether begin is rejected after supersession while genuine ack remains legal. State when supersession becomes durable relative to successor admission/publication, how partial writes or crashes recover, and which lock serializes this with acknowledgment; individual atomic manifest replacement does not establish that contract.
- **PQ-3** [Important] `executable-test-strategy` Replace prose case inventories with named function-level test strategies.
  Compress the revised verification checklist into named test surfaces such as dom_submission_state.step, submit_operator_text, submissionManager.request/resolve, and the proposed supersession transition/store operation. Give each risky function one adversarial-input or event-sequence strategy and mechanical guard, using controlled callback/worker ordering and injected persistence failures rather than another enumerated case list.

## Round 2 — 2026-10-13T11:04:37-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Independent bounded acknowledgment receipts and shared ack/status scheduling define recovery across successor epochs, busy workers, failures and restart.
- PQ-2 — addressed — Pure transition rules and Store.mu serialization define durable supersession before publication, partial-write recovery and successor-safe retries.
- PQ-3 — addressed — The authoritative function-level strategy table replaces the case inventory with controlled event ordering and injected persistence failures.

## Open findings

(none — every finding has been disposed)
