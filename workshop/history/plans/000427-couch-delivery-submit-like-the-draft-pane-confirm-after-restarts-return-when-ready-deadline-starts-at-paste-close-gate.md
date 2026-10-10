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
    - "n": 3
      timestamp: "2026-10-10T13:08:20-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan Revisions record that a pre-Build wrapper ends the receipt failed/unready and that the poll is 50ms; code matches (live_restart_probe.go AwaitReady, peer_delivery.go peerPastedPoll).
          round: 3
        - id: BR-2
          disposition: withdrawn
          note: Still withdrawn; readyWithin bounds the goroutine, and the plan Revisions say so.
          round: 3
        - id: BR-3
          disposition: addressed
          note: peer_delivery_test.go:106-115 covers busy-at-submit ("a turn was already running"); passes in a clean environment.
          round: 3
      findings:
        - id: BR-4
          severity: Minor
          title: Plan body still states the 100ms poll and pre-Build warning; only Revisions correct it
          detail: 'This is the 2nd finding in family plan-code-drift. Rule: when a Revisions entry contradicts a body line, annotate that line with a pointer to the revision. Informational only; the append-only convention is being followed.'
          family: plan-code-drift
          round: 3
      recipe: milestone-review
      reviewed: 912e77368735c004b4e79f406bdd7127f6891cdb
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

## Round 3 — 2026-10-10T13:08:20-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Plan Revisions record that a pre-Build wrapper ends the receipt failed/unready and that the poll is 50ms; code matches (live_restart_probe.go AwaitReady, peer_delivery.go peerPastedPoll).
- BR-2 — withdrawn — Still withdrawn; readyWithin bounds the goroutine, and the plan Revisions say so.
- BR-3 — addressed — peer_delivery_test.go:106-115 covers busy-at-submit ("a turn was already running"); passes in a clean environment.

### Raised

- **BR-4** [Minor] `plan-code-drift` Plan body still states the 100ms poll and pre-Build warning; only Revisions correct it
  This is the 2nd finding in family plan-code-drift. Rule: when a Revisions entry contradicts a body line, annotate that line with a pointer to the revision. Informational only; the append-only convention is being followed.

## Open findings

- **BR-4** [Minor] `plan-code-drift` Plan body still states the 100ms poll and pre-Build warning; only Revisions correct it
