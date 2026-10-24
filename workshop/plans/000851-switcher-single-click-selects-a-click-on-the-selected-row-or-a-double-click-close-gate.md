---
gate: boundary-review
issue: 851
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5b3689f0-f2b3-42aa-88f4-31a6c95f0f1e
            repository: github.com/xianxu/pair
            issue_id: "000851"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a384579be7e3defa
            reviewed_head: 4c251d1ab612e26805c9c387979838b8a8ee7f58
            slot: pair:2
            requested_at: "2026-10-24T17:41:42.430356Z"
          phase: finished
          pid: 60062
          started_at: "2026-10-24T17:41:42.563935Z"
          finished_at: "2026-10-24T17:43:45.773136Z"
          session_id: 01a16f01-11fd-7d02-b49e-5fb85bf5d0d6
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T10:44:05-07:00"
      dispatched: "2026-10-24T10:41:40-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: README update is missing for the changed switcher click behavior.
          detail: README.md:1925-1928 still says any switcher-row click switches immediately and every click is a manual switch; README.md:1226 also promises an unqualified one-click warm-attachment retry. These are all affected README claims in this window. Update them to distinguish selection from activation, chips inside versus outside the switcher, and retrying an already-selected row. atlas/couch.md:3292 already documents the new policy, but README.md has no change in the pinned range.
          family: user-facing-behavior-documentation
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Done-when has not been reconciled with the revised click contract.
          detail: workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md:67-71 retains the double-click timing-window requirement, while Spec:38-44 explicitly rejects a timer and adds scoped chip behavior. The stale acceptance family comprises the timing-window clause and missing explicit acceptance for inside/outside chips, hidden-target reveal with selective filter/scope preservation, and nested-frame selection before activation. Revise Done-when to name those observable outcomes and rapid/delayed press tests.
          family: acceptance-contract-synchronization
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#851 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T10:44:05-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-facing-behavior-documentation` (kind: defect) README update is missing for the changed switcher click behavior.
  README.md:1925-1928 still says any switcher-row click switches immediately and every click is a manual switch; README.md:1226 also promises an unqualified one-click warm-attachment retry. These are all affected README claims in this window. Update them to distinguish selection from activation, chips inside versus outside the switcher, and retrying an already-selected row. atlas/couch.md:3292 already documents the new policy, but README.md has no change in the pinned range.
- **BR-2** [Important] `acceptance-contract-synchronization` (kind: defect) Done-when has not been reconciled with the revised click contract.
  workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md:67-71 retains the double-click timing-window requirement, while Spec:38-44 explicitly rejects a timer and adds scoped chip behavior. The stale acceptance family comprises the timing-window clause and missing explicit acceptance for inside/outside chips, hidden-target reveal with selective filter/scope preservation, and nested-frame selection before activation. Revise Done-when to name those observable outcomes and rapid/delayed press tests.

## Open findings

- **BR-1** [Important] `user-facing-behavior-documentation` (kind: defect) README update is missing for the changed switcher click behavior.
- **BR-2** [Important] `acceptance-contract-synchronization` (kind: defect) Done-when has not been reconciled with the revised click contract.
