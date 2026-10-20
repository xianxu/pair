---
gate: boundary-review
issue: 727
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 501abd18-d14e-459a-bb1b-6bb3c09076dd
            repository: github.com/xianxu/pair
            issue_id: "000727"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/bcb8e3e2dacdfffa
            reviewed_head: 4617a1608cf2dee3fdd037436587f2806474d031
            slot: pair:7
            requested_at: "2026-10-20T16:20:23.152605Z"
          phase: finished
          pid: 32221
          started_at: "2026-10-20T16:20:23.247057Z"
          finished_at: "2026-10-20T16:23:15.227509Z"
          session_id: 01a15a1d-2b41-70e2-b7e6-f3168ac30bab
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4617a1608cf2dee3fdd037436587f2806474d031
        tracker_head: df13eb8fedf40744fb0c47f289bc8c88a2959702
        branch_point: 8620374573f7a42310ffe6348d7946d5326dd15f
        since: "2026-10-20T08:47:04-07:00"
        until: "2026-10-20T09:16:59-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T08:47:04-07:00"
              end: "2026-10-20T09:16:59-07:00"
              began: "2026-10-20T08:47:04-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ad60089776fe19eb9a51c3c05174d53a87ea2cda35f8cb55182f14ca4adf2f51
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 197723094
              windows:
                - start: "2026-10-20T08:47:04-07:00"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a159ff-1dfb-7780-bc4e-14af28341ab2
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 4569c28cb6f6f1630f2fce258ea8ee127432faad6e7c34af4049728303aa34d6
              started: "2026-10-20T15:47:34.137Z"
              spawned: true
              bytes: 995168
              windows:
                - start: "2026-10-20T15:47:34.137Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a05-cb5b-7600-bcda-f2d221fa6946
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e4bb165011c00ee3950c2ba7f08e9263e84380ec9dd2d265e04658108d679660
              started: "2026-10-20T15:54:52.502Z"
              spawned: false
              bytes: 607435
              windows:
                - start: "2026-10-20T15:54:52.502Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a08-085e-7d70-a4ec-bf90ed500d8b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ef6be92d140da503ed98c59715b1bd6e7b21437566f9433965c79a8cc49c8f27
              started: "2026-10-20T15:57:18.778Z"
              spawned: false
              bytes: 513210
              windows:
                - start: "2026-10-20T15:57:18.778Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a09-c99f-7271-8bbd-cb205f1cd2b7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8c1ae75f8f6627c6a13fa48a83cfc75eb525c928f200fc09a7f3dbecab549120
              started: "2026-10-20T15:59:13.75Z"
              spawned: false
              bytes: 431060
              windows:
                - start: "2026-10-20T15:59:13.75Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0c-31ed-7002-88e3-80248b6d55cf
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d744eec5c6a408404d96ffb7146daad69bae90e695e3853fcb5b5272a7e5a89c
              started: "2026-10-20T16:01:51.488Z"
              spawned: false
              bytes: 443293
              windows:
                - start: "2026-10-20T16:01:51.488Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0c-a319-72c2-99f3-dd301032c08d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e2407cccf92659f8d39d5762efc99529bf4c808c8d7e96e1db5ce8789f37f1d1
              started: "2026-10-20T16:02:20.495Z"
              spawned: false
              bytes: 403720
              windows:
                - start: "2026-10-20T16:02:20.495Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0d-fd4d-7fb1-afbb-a944fe169e13
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f478bca0df2df1407a37014fcd217a26e7c2de92298ab18e2f4159b4fb76bc5a
              started: "2026-10-20T16:03:48.782Z"
              spawned: true
              bytes: 953484
              windows:
                - start: "2026-10-20T16:03:48.782Z"
                  end: "2026-10-20T09:16:59-07:00"
            - harness: codex
              id: 01a15a0e-2f46-75a0-bebe-7553805d3d33
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b02aed9824c3ae1846adc7f8e9590bc70d9733da19422891dae5ba2fa3a97ded
              started: "2026-10-20T16:04:01.622Z"
              spawned: true
              bytes: 948255
              windows:
                - start: "2026-10-20T16:04:01.622Z"
                  end: "2026-10-20T09:16:59-07:00"
      "n": 1
      timestamp: "2026-10-20T09:23:16-07:00"
      dispatched: "2026-10-20T09:20:21-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Input failures are classified as normal completion and bypass shell recovery
          detail: cmd/internal/termcmd/run.go:430-432 discards the input pump's outcome and calls stopLocked(nil); line 457 then classifies mux.done as RecoveryChildExit. A decoder failure, such as invalid UTF-8, only displays a notice before returning, while read errors are discarded outright. With successful cleanup, recovery receives nil and exits without a fallback shell. Return a typed pump outcome, preserve unexpected failures, and test decoder failure and non-EOF read failure through runShellRecovery. This violates the expanded recovery contract and ARCH-PURPOSE/ARCH-ORDER.
          family: recovery-outcome-classification
          round: 1
        - id: BR-2
          severity: Critical
          title: Recreating the missing terminal moves focus away from the invoking pane
          detail: cmd/internal/layoutcmd/layoutcmd.go:252 creates the pane without --no-focus, contrary to README.md:174's explicit focus-preserving contract. Installed Zellij 0.45.1 documents --no-focus as the option that prevents client focus changes; the new fake likewise marks this pane focused. Add --no-focus and assert that the invoking pane/client retains focus after recovery, so subsequent typing stays in the draft or agent. ARCH-PURPOSE.
          family: shortcut-focus-preservation
          round: 1
        - id: BR-3
          severity: Important
          title: The new native pane-creation behavior has no live conformance coverage
          detail: cmd/internal/layoutcmd/terminal_recovery_test.go:46 exercises only the Python fake and exact argv equality. The range adds no native conformance test for the new --tab-id/--near-current-pane creation path, and existing live creation coverage uses a different downward-split command. Add a disposable native test through the production routing seam that verifies visible creation in the invoking tab and preserved client focus, and include it in scheduled conformance. ARCH-MOCK.
          family: external-fake-conformance
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#727 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T09:23:16-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `recovery-outcome-classification` Input failures are classified as normal completion and bypass shell recovery
  cmd/internal/termcmd/run.go:430-432 discards the input pump's outcome and calls stopLocked(nil); line 457 then classifies mux.done as RecoveryChildExit. A decoder failure, such as invalid UTF-8, only displays a notice before returning, while read errors are discarded outright. With successful cleanup, recovery receives nil and exits without a fallback shell. Return a typed pump outcome, preserve unexpected failures, and test decoder failure and non-EOF read failure through runShellRecovery. This violates the expanded recovery contract and ARCH-PURPOSE/ARCH-ORDER.
- **BR-2** [Critical] `shortcut-focus-preservation` Recreating the missing terminal moves focus away from the invoking pane
  cmd/internal/layoutcmd/layoutcmd.go:252 creates the pane without --no-focus, contrary to README.md:174's explicit focus-preserving contract. Installed Zellij 0.45.1 documents --no-focus as the option that prevents client focus changes; the new fake likewise marks this pane focused. Add --no-focus and assert that the invoking pane/client retains focus after recovery, so subsequent typing stays in the draft or agent. ARCH-PURPOSE.
- **BR-3** [Important] `external-fake-conformance` The new native pane-creation behavior has no live conformance coverage
  cmd/internal/layoutcmd/terminal_recovery_test.go:46 exercises only the Python fake and exact argv equality. The range adds no native conformance test for the new --tab-id/--near-current-pane creation path, and existing live creation coverage uses a different downward-split command. Add a disposable native test through the production routing seam that verifies visible creation in the invoking tab and preserved client focus, and include it in scheduled conformance. ARCH-MOCK.

## Open findings

- **BR-1** [Critical] `recovery-outcome-classification` Input failures are classified as normal completion and bypass shell recovery
- **BR-2** [Critical] `shortcut-focus-preservation` Recreating the missing terminal moves focus away from the invoking pane
- **BR-3** [Important] `external-fake-conformance` The new native pane-creation behavior has no live conformance coverage
