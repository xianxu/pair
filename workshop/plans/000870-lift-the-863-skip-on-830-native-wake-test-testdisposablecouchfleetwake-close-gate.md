---
gate: boundary-review
issue: 870
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9ef1b4d1-efbc-4e69-b54f-bb7303c315b9
            repository: github.com/xianxu/pair
            issue_id: "000870"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/f120c53c1adaf278
            reviewed_head: d1729d36e04386cee75eae1e42c66278d58421e7
            slot: pair:6
            requested_at: "2026-10-25T08:19:47.654465Z"
          phase: finished
          pid: 63906
          started_at: "2026-10-25T08:19:47.753207Z"
          finished_at: "2026-10-25T08:20:53.708735Z"
          session_id: 6d0838ec-cfbc-4462-9ccd-f1cfb28ff3f8
          outcome: returned
      "n": 1
      timestamp: "2026-10-25T01:20:55-07:00"
      dispatched: "2026-10-25T01:19:45-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Repeat couch --wake ("a repeat starting nothing") is only logged, never asserted
          detail: 'fleet_acceptance_test.go:403-408 checks only exit 0 and t.Logf''s stdout, so a repeat that woke the plain worker or re-woke a hub would pass. After the repeat, assert that asleep() still shows slot 2 asleep and slots 0 and 1 awake, that fleet.Agent(2) has no live PID, and/or that stdout reports nothing to wake. Family sweep in this window: the first --wake (390-394) also only logs, but the asleep map at 399 pins its effect, so the repeat is the only unasserted clause.'
          family: done-when-clause-unasserted
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: wakeFleetConverse re-implements the Accepted-body wait loop already present in sibling tests (ARCH-DRY)
          detail: 'Near-identical loops: fleet_acceptance_test.go:113-127, :439-450, startup_restart_fleet_test.go:197-211, slash_keys_acceptance_test.go:172. They could become one waitFleetAccepted(ctx, fleet, n, body) helper.'
          family: duplicated-accept-wait-loop
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#870 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-25T01:20:55-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-clause-unasserted` (kind: defect) Repeat couch --wake ("a repeat starting nothing") is only logged, never asserted
  fleet_acceptance_test.go:403-408 checks only exit 0 and t.Logf's stdout, so a repeat that woke the plain worker or re-woke a hub would pass. After the repeat, assert that asleep() still shows slot 2 asleep and slots 0 and 1 awake, that fleet.Agent(2) has no live PID, and/or that stdout reports nothing to wake. Family sweep in this window: the first --wake (390-394) also only logs, but the asleep map at 399 pins its effect, so the repeat is the only unasserted clause.
- **BR-2** [Minor] `duplicated-accept-wait-loop` (kind: defect) wakeFleetConverse re-implements the Accepted-body wait loop already present in sibling tests (ARCH-DRY)
  Near-identical loops: fleet_acceptance_test.go:113-127, :439-450, startup_restart_fleet_test.go:197-211, slash_keys_acceptance_test.go:172. They could become one waitFleetAccepted(ctx, fleet, n, body) helper.

## Open findings

- **BR-1** [Important] `done-when-clause-unasserted` (kind: defect) Repeat couch --wake ("a repeat starting nothing") is only logged, never asserted
- **BR-2** [Minor] `duplicated-accept-wait-loop` (kind: defect) wakeFleetConverse re-implements the Accepted-body wait loop already present in sibling tests (ARCH-DRY)
