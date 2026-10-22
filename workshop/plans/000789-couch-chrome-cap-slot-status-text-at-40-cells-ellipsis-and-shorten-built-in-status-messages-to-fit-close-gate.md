---
gate: boundary-review
issue: 789
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1a716322-555f-449c-8e3f-47d698858ce5
            repository: github.com/xianxu/pair
            issue_id: "000789"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/7d1307c1a5fff308
            reviewed_head: b532c3b064bdcb1b10f02a9232591650b88970d7
            slot: pair:4
            requested_at: "2026-10-22T14:17:24.991459Z"
          phase: finished
          pid: 77451
          started_at: "2026-10-22T14:17:25.082902Z"
          finished_at: "2026-10-22T14:19:30.025612Z"
          session_id: 01a163f9-4e64-7760-8e47-c188daa0a272
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b532c3b064bdcb1b10f02a9232591650b88970d7
        tracker_head: 9f9f7dc7723e153bf49147dcfd35f478c0dece5b
        branch_point: 5df9efb7d0df1a627f8d929895181fcf7b40ef73
        since: "2026-10-22T06:59:35-07:00"
        until: "2026-10-22T07:17:03-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T06:59:35-07:00"
              end: "2026-10-22T07:17:03-07:00"
              began: "2026-10-22T06:59:35-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: cb54473dc828c51c936d0cd90be2596aa9a2c922c2f3e57a7dd4bc4ceebd78db
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 47473938
              windows:
                - start: "2026-10-22T06:59:35-07:00"
                  end: "2026-10-22T07:17:03-07:00"
            - harness: codex
              id: 01a163ea-73ce-7920-a8e0-09745fc11c15
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: eb25e90bed6928aa65584b1fec73cc92437adce03dfb4e3d0348ea8f215cfab6
              started: "2026-10-22T14:01:11.985Z"
              spawned: true
              bytes: 1227377
              windows:
                - start: "2026-10-22T14:01:11.985Z"
                  end: "2026-10-22T07:17:03-07:00"
      "n": 1
      timestamp: "2026-10-22T07:19:30-07:00"
      dispatched: "2026-10-22T07:17:23-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Catalog passes while built-in status messages still exceed 40 cells
          detail: 'cmd/internal/couchtty/status_catalog_test.go:98-133 drops selector values and formatted substitutions, including values fixed by the program, and misses callback returns. ARCH-PURPOSE: the producer-cleanup contract remains incomplete. The affected family comprises capabilityNotice retry messages in cast_control.go:41 (41/45 cells before diagnostic suffixes); console_webcast.go:329 copy messages for view-only/pointer/drive (45/43/41), :335 not-ready messages for view-only/pointer (44/42), and :299''s callback message (72); and console_castcodes.go:397/:485/:487 rotation messages containing built-in cast/link names. Sweep these producer sites and their finite variants, shorten the fixed wording, and add regression coverage that measures built-in substitutions and callback-produced notices rather than erasing them.'
          family: built-in-status-enforcement
          round: 1
        - id: BR-2
          severity: Important
          title: Lifecycle wording change leaves its regression test failing
          detail: 'cmd/internal/couchcmd/lifecycle_notices_test.go:684 still requires ": 1 observations", although lifecycle_notices.go:179 now emits "Lifecycle: 1 slot metadata unverified". Running go test ./cmd/internal/couchcmd -run ''Test(LifecycleDrainReportsOnlyUntargetedFacts|Supervision|Renicer)'' -count=1 -timeout=60s fails at line 685 with that message. This is the remaining stale wording assertion identified in the inspected window; update it while retaining the assertion that exactly one untargeted fact is counted.'
          family: producer-assertion-consistency
          round: 1
        - id: BR-3
          severity: Important
          title: README omits the changed Ctrl+Y selection behavior
          detail: cmd/internal/couchtty/console_menu.go:477 adds capability diagnostics ahead of the last feed notice. README.md:456 and :1730-1733 still promise the last notice when no selected failure applies, including repeated copies until another notice arrives. These are both affected README passages in this family. Update them to explain capability precedence and full-text recovery; README.md is unchanged in this range. The atlas update correctly describes the new behavior.
          family: user-surface-documentation
          round: 1
        - id: BR-4
          severity: Important
          title: Done-when criteria were not reconciled with the revised contract
          detail: workshop/issues/000789-couch-chrome-cap-slot-status-text-at-40-cells-ellipsis-and-shorten-built-in-status-messages-to-fit.md:44 requires every built-in status message to fit, whereas :49/:60 explicitly permit dynamic labels and errors to exceed the limit and enforce only fixed template contributions. The revised scope also adds capability-copy behavior without an explicit acceptance clause. Reconcile the second Done-when clause with the intended exemption and add capability-copy precedence and full-text recovery criteria. These are the acceptance gaps introduced by the scope revision.
          family: acceptance-contract-consistency
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#789 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T07:19:30-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `built-in-status-enforcement` Catalog passes while built-in status messages still exceed 40 cells
  cmd/internal/couchtty/status_catalog_test.go:98-133 drops selector values and formatted substitutions, including values fixed by the program, and misses callback returns. ARCH-PURPOSE: the producer-cleanup contract remains incomplete. The affected family comprises capabilityNotice retry messages in cast_control.go:41 (41/45 cells before diagnostic suffixes); console_webcast.go:329 copy messages for view-only/pointer/drive (45/43/41), :335 not-ready messages for view-only/pointer (44/42), and :299's callback message (72); and console_castcodes.go:397/:485/:487 rotation messages containing built-in cast/link names. Sweep these producer sites and their finite variants, shorten the fixed wording, and add regression coverage that measures built-in substitutions and callback-produced notices rather than erasing them.
- **BR-2** [Important] `producer-assertion-consistency` Lifecycle wording change leaves its regression test failing
  cmd/internal/couchcmd/lifecycle_notices_test.go:684 still requires ": 1 observations", although lifecycle_notices.go:179 now emits "Lifecycle: 1 slot metadata unverified". Running go test ./cmd/internal/couchcmd -run 'Test(LifecycleDrainReportsOnlyUntargetedFacts|Supervision|Renicer)' -count=1 -timeout=60s fails at line 685 with that message. This is the remaining stale wording assertion identified in the inspected window; update it while retaining the assertion that exactly one untargeted fact is counted.
- **BR-3** [Important] `user-surface-documentation` README omits the changed Ctrl+Y selection behavior
  cmd/internal/couchtty/console_menu.go:477 adds capability diagnostics ahead of the last feed notice. README.md:456 and :1730-1733 still promise the last notice when no selected failure applies, including repeated copies until another notice arrives. These are both affected README passages in this family. Update them to explain capability precedence and full-text recovery; README.md is unchanged in this range. The atlas update correctly describes the new behavior.
- **BR-4** [Important] `acceptance-contract-consistency` Done-when criteria were not reconciled with the revised contract
  workshop/issues/000789-couch-chrome-cap-slot-status-text-at-40-cells-ellipsis-and-shorten-built-in-status-messages-to-fit.md:44 requires every built-in status message to fit, whereas :49/:60 explicitly permit dynamic labels and errors to exceed the limit and enforce only fixed template contributions. The revised scope also adds capability-copy behavior without an explicit acceptance clause. Reconcile the second Done-when clause with the intended exemption and add capability-copy precedence and full-text recovery criteria. These are the acceptance gaps introduced by the scope revision.

## Open findings

- **BR-1** [Critical] `built-in-status-enforcement` Catalog passes while built-in status messages still exceed 40 cells
- **BR-2** [Important] `producer-assertion-consistency` Lifecycle wording change leaves its regression test failing
- **BR-3** [Important] `user-surface-documentation` README omits the changed Ctrl+Y selection behavior
- **BR-4** [Important] `acceptance-contract-consistency` Done-when criteria were not reconciled with the revised contract
