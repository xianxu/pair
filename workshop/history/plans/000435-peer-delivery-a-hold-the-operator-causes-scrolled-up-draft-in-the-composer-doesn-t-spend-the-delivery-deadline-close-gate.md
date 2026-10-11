---
gate: boundary-review
issue: 435
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T23:47:25-07:00"
      dispatched: "2026-10-10T23:45:24-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Operator keystroke waits (input settle / held / forwarded input) close the hold, so typing a draft spends the paste-by
          detail: peer_delivery.go dispatch returns early with operatorHold=false on "waiting for input to settle" and the other input-caused waits, closing the hold and flipping Held back to Queued; continuous typing for 90s expires the message and makes the sender's status flicker. Keep an open hold open in those branches and add a keystroke-during-hold test.
          family: operator-hold-classification
          round: 1
        - id: BR-2
          severity: Minor
          title: Expiry at HoldLimit via accumulated credit (hold already closed) omits the "held past" naming
          family: hold-expiry-naming
          round: 1
        - id: BR-3
          severity: Minor
          title: Horizon grew by 30m, so a hung recipient keeps a sender pending far longer; atlas does not mention this cost
          family: horizon-liveness-cost
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T23:57:17-07:00"
      dispatched: "2026-10-10T23:53:07-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: judged is set only after the settle and held-input waits; every earlier wait returns before it. TestPeerTypingKeepsTheHoldOpen fails if the code always judges.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The "held past" naming now keys on reaching HoldLimit, not on an open hold; TestPeerHoldLimitAfterTheHoldClosed pins it.
          round: 2
        - id: BR-3
          disposition: addressed
          note: atlas/couch.md now states that a hung recipient keeps the sender pending for about 32m instead of 2m.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: A new wrapper's held receipt reads as an invalid status to an old couch daemon
          detail: Until couch restarts, the old RemoteEndpoint.Deliver rejects held, so the sender gets indeterminate while the wrapper may still paste later; the old daemon's horizon is also 2m. Add an atlas note to restart couch with the upgrade.
          family: wrapper-couch-version-skew
          round: 2
      recipe: milestone-review
      reviewed: a577977b8e3f5f46a004d1c78ac136afc2897c7d
      blocked: false
---

# Gate ledger — pair#435 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T23:47:25-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `operator-hold-classification` Operator keystroke waits (input settle / held / forwarded input) close the hold, so typing a draft spends the paste-by
  peer_delivery.go dispatch returns early with operatorHold=false on "waiting for input to settle" and the other input-caused waits, closing the hold and flipping Held back to Queued; continuous typing for 90s expires the message and makes the sender's status flicker. Keep an open hold open in those branches and add a keystroke-during-hold test.
- **BR-2** [Minor] `hold-expiry-naming` Expiry at HoldLimit via accumulated credit (hold already closed) omits the "held past" naming
- **BR-3** [Minor] `horizon-liveness-cost` Horizon grew by 30m, so a hung recipient keeps a sender pending far longer; atlas does not mention this cost

## Round 2 — 2026-10-10T23:57:17-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — judged is set only after the settle and held-input waits; every earlier wait returns before it. TestPeerTypingKeepsTheHoldOpen fails if the code always judges.
- BR-2 — addressed — The "held past" naming now keys on reaching HoldLimit, not on an open hold; TestPeerHoldLimitAfterTheHoldClosed pins it.
- BR-3 — addressed — atlas/couch.md now states that a hung recipient keeps the sender pending for about 32m instead of 2m.

### Raised

- **BR-4** [Minor] `wrapper-couch-version-skew` A new wrapper's held receipt reads as an invalid status to an old couch daemon
  Until couch restarts, the old RemoteEndpoint.Deliver rejects held, so the sender gets indeterminate while the wrapper may still paste later; the old daemon's horizon is also 2m. Add an atlas note to restart couch with the upgrade.

## Open findings

- **BR-4** [Minor] `wrapper-couch-version-skew` A new wrapper's held receipt reads as an invalid status to an old couch daemon
