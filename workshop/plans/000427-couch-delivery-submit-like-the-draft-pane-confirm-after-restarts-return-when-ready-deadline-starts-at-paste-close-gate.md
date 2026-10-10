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

## Open findings

- **BR-1** [Minor] `plan-code-drift` Plan says a pre-Build wrapper returns with a warning; code fails the receipt as unready (and the poll is 50ms, not 100ms)
- **BR-2** [Minor] `unbounded-goroutine-extent` awaitReadiness goroutine runs on context.Background, so Couch shutdown does not cancel it
- **BR-3** [Minor] `test-gap-turn-already-running` No test for confirmation when a turn was already running at submit (turnAtSubmit true)
