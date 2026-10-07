---
gate: boundary-review
issue: 395
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T15:36:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Hub.Activate arms grace even when the last accepted frame shows LIVE, so an idle screen ends the broadcast after 1s
          detail: 'hub.go Activate sets missing=true whenever !missing, but before Activate the missing flag is never set from frame content. Reproduced with an overlay test: offer(live), Activate(), fire grace leads to ErrIndicatorHidden. Track a shown bool from every accept (or collapse active/missing/graceC into an explicit inactive|shown|hidden state, ARCH-ORDER) and arm only when not shown; add a regression test and randomize the Activate position in TestHubRandomInterleavings.'
          family: hub-state-flag-constellation
          round: 1
        - id: BR-2
          severity: Minor
          title: Plan promises a final End message to subscribers; hub only closes channels
          detail: M2 server must read Hub.Err() and distinguish a hub end from Subscription.Close. Implement the End message or add a Revisions entry.
          family: plan-code-drift
          round: 1
        - id: BR-3
          severity: Minor
          title: TestTapNotCalledOnFailedPaint omits the refused PresentView transition case the plan lists
          family: plan-test-coverage-gap
          round: 1
        - id: BR-4
          severity: Minor
          title: Hub resync ticker runs for the hub lifetime even with no resyncing subscriber
          family: idle-background-work
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T15:52:10-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Tagged off/shown/hidden watch; overlay mutant (Activate ignores shown) turns the new subtest and TestHubRandomInterleavings red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Plan Revisions 2026-10-07 documents close-queues + Hub.Err() and the M2 server mapping; matches hub.go end().
          round: 2
        - id: BR-3
          disposition: addressed
          note: Plan Revisions entry records the refused-PresentView branch as defence in depth with no public trigger; consistent with presenter.go:380.
          round: 2
        - id: BR-4
          disposition: addressed
          note: run() selects tickC only while resyncing > 0; TestHubRealTickerResyncsQuietScreen exercises the gate and checks resyncing returns to 0.
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#395 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T15:36:37-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `hub-state-flag-constellation` Hub.Activate arms grace even when the last accepted frame shows LIVE, so an idle screen ends the broadcast after 1s
  hub.go Activate sets missing=true whenever !missing, but before Activate the missing flag is never set from frame content. Reproduced with an overlay test: offer(live), Activate(), fire grace leads to ErrIndicatorHidden. Track a shown bool from every accept (or collapse active/missing/graceC into an explicit inactive|shown|hidden state, ARCH-ORDER) and arm only when not shown; add a regression test and randomize the Activate position in TestHubRandomInterleavings.
- **BR-2** [Minor] `plan-code-drift` Plan promises a final End message to subscribers; hub only closes channels
  M2 server must read Hub.Err() and distinguish a hub end from Subscription.Close. Implement the End message or add a Revisions entry.
- **BR-3** [Minor] `plan-test-coverage-gap` TestTapNotCalledOnFailedPaint omits the refused PresentView transition case the plan lists
- **BR-4** [Minor] `idle-background-work` Hub resync ticker runs for the hub lifetime even with no resyncing subscriber

## Round 2 — 2026-10-07T15:52:10-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Tagged off/shown/hidden watch; overlay mutant (Activate ignores shown) turns the new subtest and TestHubRandomInterleavings red.
- BR-2 — addressed — Plan Revisions 2026-10-07 documents close-queues + Hub.Err() and the M2 server mapping; matches hub.go end().
- BR-3 — addressed — Plan Revisions entry records the refused-PresentView branch as defence in depth with no public trigger; consistent with presenter.go:380.
- BR-4 — addressed — run() selects tickC only while resyncing > 0; TestHubRealTickerResyncsQuietScreen exercises the gate and checks resyncing returns to 0.

## Open findings

(none — every finding has been disposed)
