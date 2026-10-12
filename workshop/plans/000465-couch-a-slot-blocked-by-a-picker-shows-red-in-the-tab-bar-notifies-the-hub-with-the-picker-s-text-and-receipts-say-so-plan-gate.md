---
gate: plan-quality
issue: 465
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T00:49:45-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: The picker hold must set judged/operatorHold; the picker branch returns before judged=true
          detail: peer_delivery.go:459-462 returns "waiting for picker to close" before judged=true, so the deferred operatorHoldLocked never opens a hold. The picker branch must set judged, operatorHold = true, true, or the receipt never goes held. Also read the title once per hold, not on every dispatch, and note that d.waits is keyed by the detail string.
          family: hold-open-requires-judged
          round: 1
        - id: PQ-2
          severity: Minor
          title: The Kind rule in Revisions contradicts Design §4 and the Tests line "selection tail is a question"
          detail: Revision (a) adds a third kind, plain "picker", and requires the Enter to select footer for "question". Align the ReadPicker tests with the revised rule.
          family: revision-not-folded-into-tests
          round: 1
        - id: PQ-3
          severity: Minor
          title: A picker is reported only after 3s of output quiet; continuous redraw could hide it
          detail: settleFired is re-armed by output, so a picker shown under a status-line animation may never publish Picker(true). Check this in the live smoke test.
          family: quiet-gated-fact-reporting
          round: 1
        - id: PQ-4
          severity: Minor
          title: No sequence test for Picker arriving mid-episode after the generic 10m path is eligible
          detail: Add a watcher test where the Picker fact flips from nil/false to true within one LastActivity episode, and assert there is exactly one notice.
          family: arch-order-sequence-coverage
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T00:51:15-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Plan unchanged since round 1; peer_delivery.go:455-458 still returns before judged=true (line 477). Carried to close.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Design §4 and Tests still state the two-kind rule; Revision (a) is not folded in. Carried to close.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: The live-check step still doesn't cover a picker under continuous redraw. Carried to close.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Tests still lack the mid-episode Picker flip sequence. Carried to close.
          round: 2
      blocked: false
content_hash: e32da3e1d10fa0748a341629eaec7609fb7e97d30d28413f16cfa2de1f561ef0
---

# Gate ledger — pair#465 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T00:49:45-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `hold-open-requires-judged` The picker hold must set judged/operatorHold; the picker branch returns before judged=true
  peer_delivery.go:459-462 returns "waiting for picker to close" before judged=true, so the deferred operatorHoldLocked never opens a hold. The picker branch must set judged, operatorHold = true, true, or the receipt never goes held. Also read the title once per hold, not on every dispatch, and note that d.waits is keyed by the detail string.
- **PQ-2** [Minor] `revision-not-folded-into-tests` The Kind rule in Revisions contradicts Design §4 and the Tests line "selection tail is a question"
  Revision (a) adds a third kind, plain "picker", and requires the Enter to select footer for "question". Align the ReadPicker tests with the revised rule.
- **PQ-3** [Minor] `quiet-gated-fact-reporting` A picker is reported only after 3s of output quiet; continuous redraw could hide it
  settleFired is re-armed by output, so a picker shown under a status-line animation may never publish Picker(true). Check this in the live smoke test.
- **PQ-4** [Minor] `arch-order-sequence-coverage` No sequence test for Picker arriving mid-episode after the generic 10m path is eligible
  Add a watcher test where the Picker fact flips from nil/false to true within one LastActivity episode, and assert there is exactly one notice.

## Round 2 — 2026-10-12T00:51:15-07:00 (claude) — passed

### Disposed

- PQ-1 — not-addressed — Plan unchanged since round 1; peer_delivery.go:455-458 still returns before judged=true (line 477). Carried to close.
- PQ-2 — not-addressed — Design §4 and Tests still state the two-kind rule; Revision (a) is not folded in. Carried to close.
- PQ-3 — not-addressed — The live-check step still doesn't cover a picker under continuous redraw. Carried to close.
- PQ-4 — not-addressed — Tests still lack the mid-episode Picker flip sequence. Carried to close.

## Open findings

- **PQ-1** [Minor] `hold-open-requires-judged` The picker hold must set judged/operatorHold; the picker branch returns before judged=true
- **PQ-2** [Minor] `revision-not-folded-into-tests` The Kind rule in Revisions contradicts Design §4 and the Tests line "selection tail is a question"
- **PQ-3** [Minor] `quiet-gated-fact-reporting` A picker is reported only after 3s of output quiet; continuous redraw could hide it
- **PQ-4** [Minor] `arch-order-sequence-coverage` No sequence test for Picker arriving mid-episode after the generic 10m path is eligible
