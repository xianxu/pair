---
gate: boundary-review
issue: 607
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 05ec9152-fb8d-4051-832c-60d1e4e85195
            repository: github.com/xianxu/pair
            issue_id: "000607"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/4b4849597082ea12
            reviewed_head: 2a756c8d00e99a177b23c1d29c3676cff445c53e
            slot: pair:2
            requested_at: "2026-10-16T09:09:10.023688Z"
          phase: finished
          pid: 5239
          started_at: "2026-10-16T09:09:10.186362Z"
          finished_at: "2026-10-16T09:13:20.694377Z"
          session_id: 0cff751c-94d1-4486-a668-9485dc1a39fd
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2a756c8d00e99a177b23c1d29c3676cff445c53e
        tracker_head: 0573ae64064a698ecacb6b2a20668b23dc23a754
        branch_point: a6dcbbcb83e1b054b2abf46adc5d8d65e914cf50
        since: "2026-10-15T23:54:34-07:00"
        until: "2026-10-16T02:08:39-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T23:54:34-07:00"
              end: "2026-10-16T02:08:39-07:00"
              began: "2026-10-15T23:54:34-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 182c407503ab1e0182efe00b7dbeaf809003c32f9085246c69b51f66298bcfe9
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 5512003
              windows:
                - start: "2026-10-15T23:54:34-07:00"
                  end: "2026-10-16T02:08:39-07:00"
      "n": 1
      timestamp: "2026-10-16T02:13:23-07:00"
      dispatched: "2026-10-16T02:09:07-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: withSlotTree treats a registered wrapper with Unknown liveness as no wrapper and reboots leaving the tree untouched
          detail: 'recovery_slot_tree.go:28 returns the retirement unchanged when observeExactProcessOrUnknown is not Live, so an unobservable (possibly live) registered wrapper reproduces the #607 orphan silently (ARCH-ORDER). Refuse or at least note on Unknown, and add a test.'
          family: uncertainty-collapsed-into-absence
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Shared-session reboot path is only tested at the helper level
          detail: TestSlotSessionSharedWithAnotherLiveSlotIsLeft calls otherLiveSessionHolder directly; no reboot test asserts that a shared session is left, the left-running note is emitted, and the wrapper tree is still stopped, although Done-when names this case.
          family: done-when-case-untested-end-to-end
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Tab bar hides a newly attached reboot pane while the inventory still lists the old address for that worktree
          detail: console_presentation.go:45 skips any pane whose address is not in inventory but whose tree is a slot. Right after a reboot this can drop the active pane's chip until the next refresh; the base code deliberately kept such lagging panes (#307), and no test covers this ordering.
          family: snapshot-lag-presentation
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: slotWrapperTree silently skips descendants with an empty identity
          detail: Such children survive the reboot and no note says so; consider adding a note to the result.
          family: silent-skip-in-stop-set
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: Issue Plan section is an empty checkbox
          detail: The delivered items are recorded only in the Log; fill in the Plan so close review can trace it.
          family: plan-traceability
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3212c9c3-a425-4039-8c58-9a090eab3f76
            repository: github.com/xianxu/pair
            issue_id: "000607"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0e5c0d0116166dc2
            reviewed_head: c8e2c29a75356cb65202fd8b0f1afcc52f4b262f
            slot: pair:2
            requested_at: "2026-10-16T09:19:34.684939Z"
          phase: finished
          pid: 57249
          started_at: "2026-10-16T09:19:34.924742Z"
          finished_at: "2026-10-16T09:22:52.846698Z"
          session_id: c98c23ee-d3fd-491d-923d-32246f7e31f6
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c8e2c29a75356cb65202fd8b0f1afcc52f4b262f
        tracker_head: ca3cc50b9aaa77af9308e57a483d7f9611265484
        branch_point: a6dcbbcb83e1b054b2abf46adc5d8d65e914cf50
        since: "2026-10-15T23:54:34-07:00"
        until: "2026-10-16T02:18:27-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T23:54:34-07:00"
              end: "2026-10-16T02:18:27-07:00"
              began: "2026-10-15T23:54:34-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8941a3c47c791c8eb2b9ddb8784435067712a3c2e7766e1458b3e81ae52fe753
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 5710879
              windows:
                - start: "2026-10-15T23:54:34-07:00"
                  end: "2026-10-16T02:18:27-07:00"
        cuts:
            - dispatched: "2026-10-16T02:09:07-07:00"
              finalized: "2026-10-16T02:13:23-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T02:23:09-07:00"
      dispatched: "2026-10-16T02:19:32-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Unknown now refuses (recovery_slot_tree.go:266-269); reverting it turns TestRebootRefusesWhenRegisteredWrapperIsUnobservable red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestRebootLeavesSessionSharedWithAnotherLiveSlot runs the full reboot; disabling the holder branch fails it.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: TL-directed design (chip follows the switcher's current row), recorded in atlas/couch.md "One chip per slot (#607)"; non-slot panes keep the lag fallback.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Note added (recovery_slot_tree.go:290-292) but no test asserts it; an unidentified child's subtree is also neither walked nor counted.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Plan now lists the three delivered items with file locations.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: c8e2c29a75356cb65202fd8b0f1afcc52f4b262f
      blocked: false
---

# Gate ledger — pair#607 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T02:13:23-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `uncertainty-collapsed-into-absence` (kind: defect) withSlotTree treats a registered wrapper with Unknown liveness as no wrapper and reboots leaving the tree untouched
  recovery_slot_tree.go:28 returns the retirement unchanged when observeExactProcessOrUnknown is not Live, so an unobservable (possibly live) registered wrapper reproduces the #607 orphan silently (ARCH-ORDER). Refuse or at least note on Unknown, and add a test.
- **BR-2** [Important] `done-when-case-untested-end-to-end` (kind: defect) Shared-session reboot path is only tested at the helper level
  TestSlotSessionSharedWithAnotherLiveSlotIsLeft calls otherLiveSessionHolder directly; no reboot test asserts that a shared session is left, the left-running note is emitted, and the wrapper tree is still stopped, although Done-when names this case.
- **BR-3** [Minor] `snapshot-lag-presentation` (kind: defect) Tab bar hides a newly attached reboot pane while the inventory still lists the old address for that worktree
  console_presentation.go:45 skips any pane whose address is not in inventory but whose tree is a slot. Right after a reboot this can drop the active pane's chip until the next refresh; the base code deliberately kept such lagging panes (#307), and no test covers this ordering.
- **BR-4** [Minor] `silent-skip-in-stop-set` (kind: defect) slotWrapperTree silently skips descendants with an empty identity
  Such children survive the reboot and no note says so; consider adding a note to the result.
- **BR-5** [Minor] `plan-traceability` (kind: defect) Issue Plan section is an empty checkbox
  The delivered items are recorded only in the Log; fill in the Plan so close review can trace it.

## Round 2 — 2026-10-16T02:23:09-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Unknown now refuses (recovery_slot_tree.go:266-269); reverting it turns TestRebootRefusesWhenRegisteredWrapperIsUnobservable red.
- BR-2 — addressed — TestRebootLeavesSessionSharedWithAnotherLiveSlot runs the full reboot; disabling the holder branch fails it.
- BR-3 — withdrawn — TL-directed design (chip follows the switcher's current row), recorded in atlas/couch.md "One chip per slot (#607)"; non-slot panes keep the lag fallback.
- BR-4 — not-addressed — Note added (recovery_slot_tree.go:290-292) but no test asserts it; an unidentified child's subtree is also neither walked nor counted.
- BR-5 — addressed — Plan now lists the three delivered items with file locations.

## Open findings

- **BR-4** [Minor] `silent-skip-in-stop-set` (kind: defect) slotWrapperTree silently skips descendants with an empty identity
