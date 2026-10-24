---
gate: boundary-review
issue: 830
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1d45c4b8-ed93-48c6-9aa9-a12aa3a09dfd
            repository: github.com/xianxu/pair
            issue_id: "000830"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/63f6e911ab27370f
            reviewed_head: a520f3a5af36b66ca0a612aee8deaa95f1a0ec34
            slot: pair:6
            requested_at: "2026-10-24T11:01:18.152032Z"
          phase: finished
          pid: 90665
          started_at: "2026-10-24T11:01:18.435683Z"
          finished_at: "2026-10-24T11:23:12.90944Z"
          session_id: 5f2a0e49-bf34-4b2b-925c-ea4a8f9e4e26
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T04:23:14-07:00"
      dispatched: "2026-10-24T04:00:37-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Plan file still describes the replaced design (switcher key, couch --asleep --json, WakeHubs); no Revisions, all tasks unchecked
          detail: 'workshop/plans/000830-...-plan.md Decisions 6-7 and T1/T5/T6 contradict the code: the code has no --asleep verb or wake key, and the cause is validated as non-empty, not an enum. The revisions live only in the issue file. Add a plan-level Revisions entry and tick the tasks so ariadne#564 does not code against a contract that does not exist.'
          family: plan-doc-tracks-revisions
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: 'Done when is stale after the Spec revisions: claims a pair test with ariadne#564''s resume and an untested restart/clear-on-wake'
          detail: The wake is now the TL's couch --wake from the main hub. Pending-first worker waking is ariadne#564's, and pair only delivers asleep_at in the recover-plan rows. The end-to-end restart/clear test (TestDisposableCouchFleetWake) is skipped until pair#863. Reword Done when to match what this repo delivers and proves.
          family: done-when-tracks-spec
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Wake's effect gate re-checks caller and hub-ness but not that the target is still asleep
          detail: A hub plain-parked between admission and the resume running would still be resumed by wake (wake.go:331-344).
          family: effect-gate-reproves-admission-facts
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: handleWake spends one 1.5s slotAdmissionBudget on the snapshot read plus every per-hub admission
          detail: On a large store, later hubs come back not admitted with a deadline error. The failure is visible and a rerun is safe, but it is unmeasured.
          family: admission-budget-covers-fanout
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: No portable test checks the production chain from Couch.AsleepSlots to the hub-root match in handleWake
          detail: The wake rig fakes authority.asleep. The real snapshot-to-Target.Slot.WorktreeRoot-to-IsHub path is only covered by the skipped native T6.
          family: integration-composition-untested
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9dc68a6a-87c5-4238-9eb6-b11b5f6c628e
            repository: github.com/xianxu/pair
            issue_id: "000830"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/bacc870bae2c8212
            reviewed_head: 6fbb5568d24833bd4ae5c0d965845e5a38adfaab
            slot: pair:6
            requested_at: "2026-10-24T11:32:00.361707Z"
          phase: finished
          pid: 58990
          started_at: "2026-10-24T11:32:00.520487Z"
          finished_at: "2026-10-24T11:49:46.298162Z"
          session_id: 97a74949-9a5d-4362-97ff-61564a6b79b1
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T04:50:20-07:00"
      dispatched: "2026-10-24T04:31:57-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan now has a Revisions section covering the move from the key and --asleep to couch --wake, the string cause, and the skipped T6; tasks ticked (6fbb5568 plan diff).
          round: 2
        - id: BR-2
          disposition: addressed
          note: Done when now names couch --wake from the main hub, gives worker ordering to ariadne#564, and states the T6 skip until pair#863; matches wake_test, asleep_test and recoverplan_test.
          round: 2
        - id: BR-3
          disposition: addressed
          note: wake.go:84-93 re-reads the asleep set in the guard; TestWakeEffectGateRefusesAHubNoLongerAsleep goes red when the check is reverted (verified in the review checkout) and green when restored.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: handleWake still wraps the snapshot read and every per-hub admission in a single slotAdmissionBudget (wake.go:23); still unmeasured. Minor.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: The wake rig still fakes authority.asleep; no portable test runs Couch.AsleepSlots through handleWake's hub-root match. Minor.
          round: 2
      findings:
        - kind: defect
          id: BR-6
          severity: Minor
          title: Archive then RestoreThread brings the asleep mark back, contradicting the plan's "unarchive → parked"
          detail: RestoreThread (couchcore/retention.go:230) restores the archived bytes exactly, and nothing clears Asleep or VerifiedPark, so a restored record reads asleep and a restored hub is resumed by the next couch --wake. Either clear the mark when archiving, or change the plan's state-table row to unarchive → asleep.
          family: asleep-mark-retired-by-every-park-exit
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 6fbb5568d24833bd4ae5c0d965845e5a38adfaab
      blocked: false
---

# Gate ledger — pair#830 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T04:23:14-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `plan-doc-tracks-revisions` (kind: defect) Plan file still describes the replaced design (switcher key, couch --asleep --json, WakeHubs); no Revisions, all tasks unchecked
  workshop/plans/000830-...-plan.md Decisions 6-7 and T1/T5/T6 contradict the code: the code has no --asleep verb or wake key, and the cause is validated as non-empty, not an enum. The revisions live only in the issue file. Add a plan-level Revisions entry and tick the tasks so ariadne#564 does not code against a contract that does not exist.
- **BR-2** [Important] `done-when-tracks-spec` (kind: defect) Done when is stale after the Spec revisions: claims a pair test with ariadne#564's resume and an untested restart/clear-on-wake
  The wake is now the TL's couch --wake from the main hub. Pending-first worker waking is ariadne#564's, and pair only delivers asleep_at in the recover-plan rows. The end-to-end restart/clear test (TestDisposableCouchFleetWake) is skipped until pair#863. Reword Done when to match what this repo delivers and proves.
- **BR-3** [Minor] `effect-gate-reproves-admission-facts` (kind: defect) Wake's effect gate re-checks caller and hub-ness but not that the target is still asleep
  A hub plain-parked between admission and the resume running would still be resumed by wake (wake.go:331-344).
- **BR-4** [Minor] `admission-budget-covers-fanout` (kind: defect) handleWake spends one 1.5s slotAdmissionBudget on the snapshot read plus every per-hub admission
  On a large store, later hubs come back not admitted with a deadline error. The failure is visible and a rerun is safe, but it is unmeasured.
- **BR-5** [Minor] `integration-composition-untested` (kind: defect) No portable test checks the production chain from Couch.AsleepSlots to the hub-root match in handleWake
  The wake rig fakes authority.asleep. The real snapshot-to-Target.Slot.WorktreeRoot-to-IsHub path is only covered by the skipped native T6.

## Round 2 — 2026-10-24T04:50:20-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Plan now has a Revisions section covering the move from the key and --asleep to couch --wake, the string cause, and the skipped T6; tasks ticked (6fbb5568 plan diff).
- BR-2 — addressed — Done when now names couch --wake from the main hub, gives worker ordering to ariadne#564, and states the T6 skip until pair#863; matches wake_test, asleep_test and recoverplan_test.
- BR-3 — addressed — wake.go:84-93 re-reads the asleep set in the guard; TestWakeEffectGateRefusesAHubNoLongerAsleep goes red when the check is reverted (verified in the review checkout) and green when restored.
- BR-4 — not-addressed — handleWake still wraps the snapshot read and every per-hub admission in a single slotAdmissionBudget (wake.go:23); still unmeasured. Minor.
- BR-5 — not-addressed — The wake rig still fakes authority.asleep; no portable test runs Couch.AsleepSlots through handleWake's hub-root match. Minor.

### Raised

- **BR-6** [Minor] `asleep-mark-retired-by-every-park-exit` (kind: defect) Archive then RestoreThread brings the asleep mark back, contradicting the plan's "unarchive → parked"
  RestoreThread (couchcore/retention.go:230) restores the archived bytes exactly, and nothing clears Asleep or VerifiedPark, so a restored record reads asleep and a restored hub is resumed by the next couch --wake. Either clear the mark when archiving, or change the plan's state-table row to unarchive → asleep.

## Open findings

- **BR-4** [Minor] `admission-budget-covers-fanout` (kind: defect) handleWake spends one 1.5s slotAdmissionBudget on the snapshot read plus every per-hub admission
- **BR-5** [Minor] `integration-composition-untested` (kind: defect) No portable test checks the production chain from Couch.AsleepSlots to the hub-root match in handleWake
- **BR-6** [Minor] `asleep-mark-retired-by-every-park-exit` (kind: defect) Archive then RestoreThread brings the asleep mark back, contradicting the plan's "unarchive → parked"
