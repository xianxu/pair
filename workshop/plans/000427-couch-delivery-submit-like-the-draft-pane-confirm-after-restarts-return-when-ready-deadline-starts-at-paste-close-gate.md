---
gate: boundary-review
issue: 427
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T12:51:23-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Plan says a pre-Build wrapper returns with a warning; code fails the receipt as unready (and the poll is 50ms, not 100ms)
          detail: live_restart_probe.go:57 returns RestartUnready, which becomes failed/unready; atlas matches the code. Add a Revisions entry to the plan.
          family: plan-code-drift
          round: 1
        - id: BR-2
          severity: Minor
          title: awaitReadiness goroutine runs on context.Background, so Couch shutdown does not cancel it
          detail: Bounded by readyWithin (2m), so it cannot leak forever; passing the console lifetime context would end it on shutdown.
          family: unbounded-goroutine-extent
          round: 1
        - id: BR-3
          severity: Minor
          title: No test for confirmation when a turn was already running at submit (turnAtSubmit true)
          family: test-gap-turn-already-running
          round: 1
      recipe: milestone-review
      reviewed: 6e18ca97788703205fdd8671531d3039945c1a0f
      blocked: false
    - "n": 2
      timestamp: "2026-10-10T12:53:41-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan's Revisions entry (2026-10-10) records the failed/unready outcome and the 50ms poll, matching live_restart_probe.go:157 and peer_delivery.go:103.
          round: 2
        - id: BR-2
          disposition: withdrawn
          note: 'Declined in Revisions with a sound reason: readyWithin (2m) bounds the goroutine and process exit ends it, so its lifetime is bounded and this is not a leak.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: peer_delivery_test.go adds "queued behind a running turn" and "running turn, text stays" with turnActive set before the submit write; both pass.
          round: 2
      recipe: milestone-review
      reviewed: 45eb3e60e59fc52035b4191acb6f7d061e58a846
      blocked: false
---

# Gate ledger — pair#427 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T12:51:23-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `plan-code-drift` Plan says a pre-Build wrapper returns with a warning; code fails the receipt as unready (and the poll is 50ms, not 100ms)
  live_restart_probe.go:57 returns RestartUnready, which becomes failed/unready; atlas matches the code. Add a Revisions entry to the plan.
- **BR-2** [Minor] `unbounded-goroutine-extent` awaitReadiness goroutine runs on context.Background, so Couch shutdown does not cancel it
  Bounded by readyWithin (2m), so it cannot leak forever; passing the console lifetime context would end it on shutdown.
- **BR-3** [Minor] `test-gap-turn-already-running` No test for confirmation when a turn was already running at submit (turnAtSubmit true)

## Round 2 — 2026-10-10T12:53:41-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — The plan's Revisions entry (2026-10-10) records the failed/unready outcome and the 50ms poll, matching live_restart_probe.go:157 and peer_delivery.go:103.
- BR-2 — withdrawn — Declined in Revisions with a sound reason: readyWithin (2m) bounds the goroutine and process exit ends it, so its lifetime is bounded and this is not a leak.
- BR-3 — addressed — peer_delivery_test.go adds "queued behind a running turn" and "running turn, text stays" with turnActive set before the submit write; both pass.

## Open findings

(none — every finding has been disposed)
