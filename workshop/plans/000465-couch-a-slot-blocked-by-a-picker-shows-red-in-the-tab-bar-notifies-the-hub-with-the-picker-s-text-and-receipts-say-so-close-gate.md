---
gate: boundary-review
issue: 465
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T09:57:57-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: The picker hold must set judged/operatorHold; the picker branch returns before judged=true
          detail: |-
            peer_delivery.go:459-462 returns "waiting for picker to close" before judged=true, so the deferred operatorHoldLocked never opens a hold. The picker branch must set judged, operatorHold = true, true, or the receipt never goes held. Also read the title once per hold, not on every dispatch, and note that d.waits is keyed by the detail string.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: hold-open-requires-judged
          round: 1
        - id: BR-2
          severity: Minor
          title: The Kind rule in Revisions contradicts Design §4 and the Tests line "selection tail is a question"
          detail: |-
            Revision (a) adds a third kind, plain "picker", and requires the Enter to select footer for "question". Align the ReadPicker tests with the revised rule.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: revision-not-folded-into-tests
          round: 1
        - id: BR-3
          severity: Minor
          title: A picker is reported only after 3s of output quiet; continuous redraw could hide it
          detail: |-
            settleFired is re-armed by output, so a picker shown under a status-line animation may never publish Picker(true). Check this in the live smoke test.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: quiet-gated-fact-reporting
          round: 1
        - id: BR-4
          severity: Minor
          title: No sequence test for Picker arriving mid-episode after the generic 10m path is eligible
          detail: |-
            Add a watcher test where the Picker fact flips from nil/false to true within one LastActivity episode, and assert there is exactly one notice.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: arch-order-sequence-coverage
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T09:57:57-07:00"
      dispatched: "2026-10-12T09:49:17-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Critical
          title: SessionClient.session crashes on a nil conn when the hello fallback redial fails
          detail: session_transport.go:302 defers a closure over conn; the fallback loop reassigns conn from DialContext, which returns a nil interface on error, so the deferred Close crashes and takes the wrapper down (no recover). Reproduced with a broker that drops the hello then removes its socket. Guard with a nil check (or dial into a temporary) and add the repro as a regression test.
          family: error-path-resource-ownership
          round: 2
        - id: BR-6
          severity: Minor
          title: pickerTitle stays stale across back-to-back pickers, and an unrecognized title re-reads the tail on every dispatch
          detail: peer_delivery.go:1051 clears the cache only when a dispatch sees the latch down, and caches nothing when ReadPicker finds no title, so it re-renders 60 tail lines every 1s under inputAdmission.
          family: per-episode-cache-invalidation
          round: 2
        - id: BR-7
          severity: Minor
          title: Plan section 1 still says the settle probe reports Picker on hello-v2; the code reports it at the latch flip on hello-v3
          family: plan-matches-code
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T10:04:36-07:00"
      dispatched: "2026-10-12T10:00:39-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: peer_delivery.go:438-443 sets judged, operatorHold = true, true before returning; TestPeerDeliveryHeldByAPickerNamesIt asserts Held.
          round: 3
        - id: BR-2
          disposition: addressed
          note: picker_test.go covers PickerQuestion via the Enter-to-select footer and PickerUnknown ("Pick a model"), matching Revision (a).
          round: 3
        - id: BR-3
          disposition: addressed
          note: Picker is now reported when the latch changes (setPickerLocked), not gated on quiet; TestPickerReportedToCouchAtOnce covers the repainting countdown.
          round: 3
        - id: BR-4
          disposition: addressed
          note: TestPickerArrivingMidEpisodeDoesNotNotifyTwice flips Picker mid-episode after the generic notice and asserts one notice.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Each attempt owns its conn (session_transport.go:308-328). Reverting to a shared deferred conn makes TestNegotiationFallbackRedialFailureReturnsError panic with a nil pointer; restored, it passes.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Title cached per pickerEpisode (bumped on the latch's false-to-true flip in setPickerLocked); TestPickerHoldDetailReadsOncePerEpisode pins one read per episode and a fresh read for the next picker.
          round: 3
        - id: BR-7
          disposition: addressed
          note: 'Plan Revisions entry (2026-10-12, build and close review) supersedes section 1: reported when the latch flips, on hello-v3; atlas/couch.md:266-280 agrees.'
          round: 3
      findings:
        - id: BR-8
          severity: Minor
          title: README tab/switcher colour section does not mention the red behind-a-picker state
          detail: README.md:766-775 lists the tab and switcher colour states (idle fade, what never fades); the new red state, and the fact that a blocked thread never fades, appear only in atlas/couch.md and SKILL.md.
          family: readme-surface-coverage
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: f37e7e65678c4bcde17ae39c1d729ba5ec8901ea
      blocked: false
---

# Gate ledger — pair#465 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T09:57:57-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `hold-open-requires-judged` The picker hold must set judged/operatorHold; the picker branch returns before judged=true
  peer_delivery.go:459-462 returns "waiting for picker to close" before judged=true, so the deferred operatorHoldLocked never opens a hold. The picker branch must set judged, operatorHold = true, true, or the receipt never goes held. Also read the title once per hold, not on every dispatch, and note that d.waits is keyed by the detail string.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `revision-not-folded-into-tests` The Kind rule in Revisions contradicts Design §4 and the Tests line "selection tail is a question"
  Revision (a) adds a third kind, plain "picker", and requires the Enter to select footer for "question". Align the ReadPicker tests with the revised rule.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `quiet-gated-fact-reporting` A picker is reported only after 3s of output quiet; continuous redraw could hide it
  settleFired is re-armed by output, so a picker shown under a status-line animation may never publish Picker(true). Check this in the live smoke test.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-4** [Minor] `arch-order-sequence-coverage` No sequence test for Picker arriving mid-episode after the generic 10m path is eligible
  Add a watcher test where the Picker fact flips from nil/false to true within one LastActivity episode, and assert there is exactly one notice.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-10-12T09:57:57-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Critical] `error-path-resource-ownership` SessionClient.session crashes on a nil conn when the hello fallback redial fails
  session_transport.go:302 defers a closure over conn; the fallback loop reassigns conn from DialContext, which returns a nil interface on error, so the deferred Close crashes and takes the wrapper down (no recover). Reproduced with a broker that drops the hello then removes its socket. Guard with a nil check (or dial into a temporary) and add the repro as a regression test.
- **BR-6** [Minor] `per-episode-cache-invalidation` pickerTitle stays stale across back-to-back pickers, and an unrecognized title re-reads the tail on every dispatch
  peer_delivery.go:1051 clears the cache only when a dispatch sees the latch down, and caches nothing when ReadPicker finds no title, so it re-renders 60 tail lines every 1s under inputAdmission.
- **BR-7** [Minor] `plan-matches-code` Plan section 1 still says the settle probe reports Picker on hello-v2; the code reports it at the latch flip on hello-v3

## Round 3 — 2026-10-12T10:04:36-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — peer_delivery.go:438-443 sets judged, operatorHold = true, true before returning; TestPeerDeliveryHeldByAPickerNamesIt asserts Held.
- BR-2 — addressed — picker_test.go covers PickerQuestion via the Enter-to-select footer and PickerUnknown ("Pick a model"), matching Revision (a).
- BR-3 — addressed — Picker is now reported when the latch changes (setPickerLocked), not gated on quiet; TestPickerReportedToCouchAtOnce covers the repainting countdown.
- BR-4 — addressed — TestPickerArrivingMidEpisodeDoesNotNotifyTwice flips Picker mid-episode after the generic notice and asserts one notice.
- BR-5 — addressed — Each attempt owns its conn (session_transport.go:308-328). Reverting to a shared deferred conn makes TestNegotiationFallbackRedialFailureReturnsError panic with a nil pointer; restored, it passes.
- BR-6 — addressed — Title cached per pickerEpisode (bumped on the latch's false-to-true flip in setPickerLocked); TestPickerHoldDetailReadsOncePerEpisode pins one read per episode and a fresh read for the next picker.
- BR-7 — addressed — Plan Revisions entry (2026-10-12, build and close review) supersedes section 1: reported when the latch flips, on hello-v3; atlas/couch.md:266-280 agrees.

### Raised

- **BR-8** [Minor] `readme-surface-coverage` README tab/switcher colour section does not mention the red behind-a-picker state
  README.md:766-775 lists the tab and switcher colour states (idle fade, what never fades); the new red state, and the fact that a blocked thread never fades, appear only in atlas/couch.md and SKILL.md.

## Open findings

- **BR-8** [Minor] `readme-surface-coverage` README tab/switcher colour section does not mention the red behind-a-picker state
