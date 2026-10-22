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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6d5d19e3-351d-43e6-bea9-9853f5a49374
            repository: github.com/xianxu/pair
            issue_id: "000789"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/513c6232e3ed386c
            reviewed_head: b3b63b6136cb7c7a686106d74ff100f888e5a511
            slot: pair:4
            requested_at: "2026-10-22T14:28:57.510982Z"
          phase: finished
          pid: 12857
          started_at: "2026-10-22T14:28:57.701927Z"
          finished_at: "2026-10-22T14:31:59.803653Z"
          session_id: 01a16403-e095-77c0-9a1f-8b8388a729a9
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b3b63b6136cb7c7a686106d74ff100f888e5a511
        tracker_head: 46f3b8de749f248b238ce5d3e155c6ca15354b04
        branch_point: 5df9efb7d0df1a627f8d929895181fcf7b40ef73
        since: "2026-10-22T06:59:35-07:00"
        until: "2026-10-22T07:28:22-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T06:59:35-07:00"
              end: "2026-10-22T07:28:22-07:00"
              began: "2026-10-22T06:59:35-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 04cd1f0c7267ca0be6a8a6fe0d44dfb63627ba237c0fe3deaf18931c07c2b979
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 48025800
              windows:
                - start: "2026-10-22T06:59:35-07:00"
                  end: "2026-10-22T07:28:22-07:00"
            - harness: codex
              id: 01a163ea-73ce-7920-a8e0-09745fc11c15
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 06480a5c32ae909ec2ff010fc91bd098d3b0c18426b8b5338a27ed10f3f2ce86
              started: "2026-10-22T14:01:11.985Z"
              spawned: true
              bytes: 1535106
              windows:
                - start: "2026-10-22T14:01:11.985Z"
                  end: "2026-10-22T07:28:22-07:00"
        cuts:
            - dispatched: "2026-10-22T07:17:23-07:00"
              finalized: "2026-10-22T07:19:30-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-22T07:32:01-07:00"
      dispatched: "2026-10-22T07:28:54-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'ARCH-PURPOSE: the named sites are corrected and regression tests detect restored overlong wording, but the built-in-status-enforcement family remains incomplete. status_catalog_test.go:220-228 scans only couchtty and couchcmd, missing upstream producers. console_broadcast.go:273 combines broadcast.EndReason with fixed wording, producing 51/77/46-cell messages for three finite outcomes. console_webcast.go:221 forwards webcast/chrome.go:663''s fixed 91-cell recovery message. console_continuation.go:215/:273 forwards checkpoint/miss.go:53-60 templates containing 73/103/103 fixed cells even with slot, reason and path empty. An expanded runtime matrix against the actual broadcast and continuation producers fails on these outputs while the committed catalog passes. This is a repeat in family built-in-status-enforcement: state and enforce the rule across all notice origins, including upstream helpers and fields, rather than fixing only these instances; enumerate finite outputs and shorten their chrome wording while preserving dynamic diagnostics.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: lifecycle_notices_test.go:683-685 now checks the new wording while retaining the exactly-one count invariant. The focused lifecycle/supervision/renicer selection passes; restoring the old count assertion through a temporary Go overlay makes the exact regression fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The pinned README diff updates both passages at :456 and :1732 to describe capability precedence and full-text recovery. These match console_menu.go:468-487; the clipboard regression also verifies selected-error and held-payload precedence.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The pinned issue diff reconciles Done-when with finite built-in substitutions, dynamic diagnostic exemptions, capability-copy precedence and full-text recovery. These criteria agree with the revised Spec and the intended renderer/copy behavior.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 24dd4f00-e767-4157-aa96-68559a721a82
            repository: github.com/xianxu/pair
            issue_id: "000789"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/eee6a9a3f04d1810
            reviewed_head: c4b930ad27f9dbbc5810d87b49380cca5ecae691
            slot: pair:4
            requested_at: "2026-10-22T14:44:16.74361Z"
          phase: finished
          pid: 97890
          started_at: "2026-10-22T14:44:16.900674Z"
          finished_at: "2026-10-22T14:48:02.279831Z"
          session_id: 01a16411-e70f-7933-82cc-e18bb3723cee
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c4b930ad27f9dbbc5810d87b49380cca5ecae691
        tracker_head: bc336d38562c4b5da328d29e87a2e19a008c5fe4
        branch_point: 5df9efb7d0df1a627f8d929895181fcf7b40ef73
        since: "2026-10-22T06:59:35-07:00"
        until: "2026-10-22T07:43:40-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T06:59:35-07:00"
              end: "2026-10-22T07:43:40-07:00"
              began: "2026-10-22T06:59:35-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4a01933a70475a3b54b8d754b7c918d42b8532fb436920065ef9462475e8cadd
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 48674287
              windows:
                - start: "2026-10-22T06:59:35-07:00"
                  end: "2026-10-22T07:43:40-07:00"
            - harness: codex
              id: 01a163ea-73ce-7920-a8e0-09745fc11c15
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 7d14928e0e02788709629ec2489da24e2ba1ea2fa996bbb9f9f2fef3b9f9ff0c
              started: "2026-10-22T14:01:11.985Z"
              spawned: true
              bytes: 1750702
              windows:
                - start: "2026-10-22T14:01:11.985Z"
                  end: "2026-10-22T07:43:40-07:00"
            - harness: codex
              id: 01a1640a-520b-79b0-b5b5-dfd2a46aee92
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: dc7cd1e7dcdf5a21095b98bcc9c1b65fabec3247703b30bc8591cd3122edbc75
              started: "2026-10-22T14:36:00.474Z"
              spawned: true
              bytes: 826505
              windows:
                - start: "2026-10-22T14:36:00.474Z"
                  end: "2026-10-22T07:43:40-07:00"
        cuts:
            - dispatched: "2026-10-22T07:17:23-07:00"
              finalized: "2026-10-22T07:19:30-07:00"
              finalizing: false
            - dispatched: "2026-10-22T07:28:54-07:00"
              finalized: "2026-10-22T07:32:01-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-22T07:48:03-07:00"
      dispatched: "2026-10-22T07:44:14-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: withdrawn
          note: The pinned Spec and Done-when explicitly exclude producer shortening and exhaustive catalog enforcement. The final diff removes that implementation and retains full upstream diagnostics. The prior finding is overtaken by this documented scope change, not resolved by a passing catalog.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Lifecycle production wording and its assertion are both restored to baseline; neither appears in the final code diff. TestLifecycleDrainReportsOnlyUntargetedFacts passes.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README.md:456 and :1732 now describe capability precedence and full-text recovery, matching console_menu.go:468-485. The clipboard regression verifies capability, selected-error and held-payload precedence.
          round: 3
        - id: BR-4
          disposition: addressed
          note: The pinned issue's Done-when now requires capped rendering and full-copy precedence while explicitly excluding producer cleanup, consistently with its revised Spec and final code.
          round: 3
      findings:
        - kind: defect
          id: BR-5
          severity: Important
          title: Two screen assertions still require text beyond the new display cap
          detail: 'cmd/internal/couchtty/console_castcodes_test.go:98 waits for the 41-cell "Too many wrong codes on the LIVE view link"; console_menu_operation_test.go:145 expects "BORROWED ATTACH: lifecycle claim is open", but chrome now ends at "ope…". The full couchtty run fails both, and the resume test also fails independently. A temporary overlay disabling only the cap makes both pass. ARCH-PURPOSE: this is the 2nd finding in family producer-assertion-consistency, with two confirmed remaining sites. State the rule and sweep screen-level notice assertions: assert the bounded display independently, and check complete diagnostics through retained data or copying. Preserve the rotation security and attachment invariants; do not repair only these two instances.'
          family: producer-assertion-consistency
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7f572df3-d6af-4fda-ad58-3bfd78c51acd
            repository: github.com/xianxu/pair
            issue_id: "000789"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a83280c29f7923e1
            reviewed_head: 63749b7beef096449a9701ee0f95121984ae9d06
            slot: pair:4
            requested_at: "2026-10-22T14:56:26.256274Z"
          phase: finished
          pid: 87980
          started_at: "2026-10-22T14:56:26.457927Z"
          finished_at: "2026-10-22T14:58:28.199751Z"
          session_id: 01a1641d-0917-7df2-8377-ded7f6b00bce
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 63749b7beef096449a9701ee0f95121984ae9d06
        tracker_head: bc336d38562c4b5da328d29e87a2e19a008c5fe4
        branch_point: 5df9efb7d0df1a627f8d929895181fcf7b40ef73
        since: "2026-10-22T06:59:35-07:00"
        until: "2026-10-22T07:55:48-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T06:59:35-07:00"
              end: "2026-10-22T07:55:48-07:00"
              began: "2026-10-22T06:59:35-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 36b786f051a03b0ab7a50aa624170977b9732735f17f4e13b2b11c5e6bc9a836
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 49078891
              windows:
                - start: "2026-10-22T06:59:35-07:00"
                  end: "2026-10-22T07:55:48-07:00"
            - harness: codex
              id: 01a163ea-73ce-7920-a8e0-09745fc11c15
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 455588ab419559a2ca7ce7b67e4b43b3cd2ca1cb8d40134dc93418db54f92e13
              started: "2026-10-22T14:01:11.985Z"
              spawned: true
              bytes: 1922788
              windows:
                - start: "2026-10-22T14:01:11.985Z"
                  end: "2026-10-22T07:55:48-07:00"
            - harness: codex
              id: 01a1640a-520b-79b0-b5b5-dfd2a46aee92
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: dc7cd1e7dcdf5a21095b98bcc9c1b65fabec3247703b30bc8591cd3122edbc75
              started: "2026-10-22T14:36:00.474Z"
              spawned: true
              bytes: 826505
              windows:
                - start: "2026-10-22T14:36:00.474Z"
                  end: "2026-10-22T07:55:48-07:00"
        cuts:
            - dispatched: "2026-10-22T07:17:23-07:00"
              finalized: "2026-10-22T07:19:30-07:00"
              finalizing: false
            - dispatched: "2026-10-22T07:28:54-07:00"
              finalized: "2026-10-22T07:32:01-07:00"
              finalizing: false
            - dispatched: "2026-10-22T07:44:14-07:00"
              finalized: "2026-10-22T07:48:03-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-22T07:58:29-07:00"
      dispatched: "2026-10-22T07:56:23-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-5
          disposition: addressed
          note: console_castcodes_test.go:94 and console_menu_operation_test.go:111 now assert explicit bounded display strings and complete retained/copied diagnostics, while preserving rotation-generation, code-secrecy, exact-handle and projection invariants. Restoring each baseline test through temporary overlays reproduces its original failure. The complete couchtty package passes in 47.879s.
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: 63749b7beef096449a9701ee0f95121984ae9d06
      blocked: false
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

## Round 2 — 2026-10-22T07:32:01-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — ARCH-PURPOSE: the named sites are corrected and regression tests detect restored overlong wording, but the built-in-status-enforcement family remains incomplete. status_catalog_test.go:220-228 scans only couchtty and couchcmd, missing upstream producers. console_broadcast.go:273 combines broadcast.EndReason with fixed wording, producing 51/77/46-cell messages for three finite outcomes. console_webcast.go:221 forwards webcast/chrome.go:663's fixed 91-cell recovery message. console_continuation.go:215/:273 forwards checkpoint/miss.go:53-60 templates containing 73/103/103 fixed cells even with slot, reason and path empty. An expanded runtime matrix against the actual broadcast and continuation producers fails on these outputs while the committed catalog passes. This is a repeat in family built-in-status-enforcement: state and enforce the rule across all notice origins, including upstream helpers and fields, rather than fixing only these instances; enumerate finite outputs and shorten their chrome wording while preserving dynamic diagnostics.
- BR-2 — addressed — lifecycle_notices_test.go:683-685 now checks the new wording while retaining the exactly-one count invariant. The focused lifecycle/supervision/renicer selection passes; restoring the old count assertion through a temporary Go overlay makes the exact regression fail.
- BR-3 — addressed — The pinned README diff updates both passages at :456 and :1732 to describe capability precedence and full-text recovery. These match console_menu.go:468-487; the clipboard regression also verifies selected-error and held-payload precedence.
- BR-4 — addressed — The pinned issue diff reconciles Done-when with finite built-in substitutions, dynamic diagnostic exemptions, capability-copy precedence and full-text recovery. These criteria agree with the revised Spec and the intended renderer/copy behavior.

## Round 3 — 2026-10-22T07:48:03-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — withdrawn — The pinned Spec and Done-when explicitly exclude producer shortening and exhaustive catalog enforcement. The final diff removes that implementation and retains full upstream diagnostics. The prior finding is overtaken by this documented scope change, not resolved by a passing catalog.
- BR-2 — addressed — Lifecycle production wording and its assertion are both restored to baseline; neither appears in the final code diff. TestLifecycleDrainReportsOnlyUntargetedFacts passes.
- BR-3 — addressed — README.md:456 and :1732 now describe capability precedence and full-text recovery, matching console_menu.go:468-485. The clipboard regression verifies capability, selected-error and held-payload precedence.
- BR-4 — addressed — The pinned issue's Done-when now requires capped rendering and full-copy precedence while explicitly excluding producer cleanup, consistently with its revised Spec and final code.

### Raised

- **BR-5** [Important] `producer-assertion-consistency` (kind: defect) Two screen assertions still require text beyond the new display cap
  cmd/internal/couchtty/console_castcodes_test.go:98 waits for the 41-cell "Too many wrong codes on the LIVE view link"; console_menu_operation_test.go:145 expects "BORROWED ATTACH: lifecycle claim is open", but chrome now ends at "ope…". The full couchtty run fails both, and the resume test also fails independently. A temporary overlay disabling only the cap makes both pass. ARCH-PURPOSE: this is the 2nd finding in family producer-assertion-consistency, with two confirmed remaining sites. State the rule and sweep screen-level notice assertions: assert the bounded display independently, and check complete diagnostics through retained data or copying. Preserve the rotation security and attachment invariants; do not repair only these two instances.

## Round 4 — 2026-10-22T07:58:29-07:00 (codex) — passed

### Disposed

- BR-5 — addressed — console_castcodes_test.go:94 and console_menu_operation_test.go:111 now assert explicit bounded display strings and complete retained/copied diagnostics, while preserving rotation-generation, code-secrecy, exact-handle and projection invariants. Restoring each baseline test through temporary overlays reproduces its original failure. The complete couchtty package passes in 47.879s.

## Open findings

(none — every finding has been disposed)
