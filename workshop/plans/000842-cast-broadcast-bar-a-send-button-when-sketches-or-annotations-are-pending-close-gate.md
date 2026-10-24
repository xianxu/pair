---
gate: boundary-review
issue: 842
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8c21a1b4-831f-412c-82e0-35a6f73b024d
            repository: github.com/xianxu/pair
            issue_id: "000842"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/0543ab481621ccf4
            reviewed_head: f1b91fa4cecfab3fac3731105d05fb93331e7faf
            slot: pair:7
            requested_at: "2026-10-24T07:22:03.004821Z"
          phase: finished
          pid: 76236
          started_at: "2026-10-24T07:22:03.30821Z"
          finished_at: "2026-10-24T07:25:21.029921Z"
          session_id: 01a16cc9-c0f8-7e91-b7f8-567c085db44d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: f1b91fa4cecfab3fac3731105d05fb93331e7faf
        tracker_head: a38649452f77c2be486bd662573ca196676939db
        branch_point: aafc7c8c08f9d49b950b38265beecc9ed569b1b2
        since: "2026-10-23T22:15:11-07:00"
        until: "2026-10-24T00:09:39-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T22:15:11-07:00"
              end: "2026-10-24T00:09:39-07:00"
              began: "2026-10-23T22:15:11-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 59ae6c8904467e70ac23baa3cc0b62b4f606e4b88c9e743967434a1b00bdcf37
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 265724832
              windows:
                - start: "2026-10-23T22:15:11-07:00"
                  end: "2026-10-24T00:09:39-07:00"
            - harness: codex
              id: 01a16c56-8c0e-78e0-b2f5-5dba1ef231c1
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: cad751272af46a9e5934ffb2815d3ff120c27a623335d2bf4efb539514c514f2
              started: "2026-10-24T05:16:13.771Z"
              spawned: true
              bytes: 1688072
              windows:
                - start: "2026-10-24T05:16:13.771Z"
                  end: "2026-10-24T00:09:39-07:00"
            - harness: codex
              id: 01a16c56-9e08-73d2-9b1d-456be665d9fa
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 109b92b5b63fe5aa2e5fdc73ad0f24d8d65bb23e9ef76863e7deda595ecb8ff2
              started: "2026-10-24T05:16:18.375Z"
              spawned: true
              bytes: 1308144
              windows:
                - start: "2026-10-24T05:16:18.375Z"
                  end: "2026-10-24T00:09:39-07:00"
            - harness: codex
              id: 01a16cb3-0bcf-7da2-b96a-08b03c7b74c6
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 2c0042f643948f8f0d825ec45979ba2d76b8a6722d5ce59318d2c0db0426b671
              started: "2026-10-24T06:57:15.86Z"
              spawned: true
              bytes: 1168656
              windows:
                - start: "2026-10-24T06:57:15.86Z"
                  end: "2026-10-24T00:09:39-07:00"
            - harness: codex
              id: 01a16cb4-ef7a-7111-b5a4-6efe1f28098d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 7edcb57c86f6fb12163ae4325320ac27e28223eff8d42b2ef2e066bb2674ac19
              started: "2026-10-24T06:59:19.636Z"
              spawned: true
              bytes: 865095
              windows:
                - start: "2026-10-24T06:59:19.636Z"
                  end: "2026-10-24T00:09:39-07:00"
      "n": 1
      timestamp: "2026-10-24T00:25:44-07:00"
      dispatched: "2026-10-24T00:20:10-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: README update appears missing for the Send button and wider shortcut availability
          detail: README.md:1819 still describes Shift+Ctrl+Alt+Return as a draft action, while cmd/internal/workbenchshortcut/shortcut.go:183 makes it available across panes, including fullscreen right-terminal applications. README is unchanged in this range and does not describe the new Chrome status ⏎ control. Update the submission instructions with button visibility, shortcut availability, and the terminal-feedback limitation.
          family: user-surface-documentation
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Pane-replacement test does not observe the successor endpoint
          detail: cmd/internal/couchtty/console_send_test.go:74 creates a successor child, but lines 99–103 flush and count writes only on r.child, the original child. Accidental routing to the successor would therefore satisfy the zero-write assertion. Retain and clean up the successor, flush both endpoints, and assert that neither receives the chord after replacement (ARCH-ORDER).
          family: endpoint-isolation-test-oracle
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a6349f19-b63e-4fc1-a6e1-83da4da91abe
            repository: github.com/xianxu/pair
            issue_id: "000842"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/4a19e5a9d227be68
            reviewed_head: 27f192b8930fe672722112a425c97ca21a5d774e
            slot: pair:7
            requested_at: "2026-10-24T07:33:26.244811Z"
          phase: finished
          pid: 26864
          started_at: "2026-10-24T07:33:26.381459Z"
          finished_at: "2026-10-24T07:35:51.171603Z"
          session_id: 01a16cd4-2c6f-71e2-a0b3-b2c59bc7c9df
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 27f192b8930fe672722112a425c97ca21a5d774e
        tracker_head: e3d2006a430b74c6d5539843f637115b0408cb2f
        branch_point: aafc7c8c08f9d49b950b38265beecc9ed569b1b2
        since: "2026-10-23T22:15:11-07:00"
        until: "2026-10-24T00:32:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T22:15:11-07:00"
              end: "2026-10-24T00:32:10-07:00"
              began: "2026-10-23T22:15:11-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 0c95af842e5083019ff1190d1cfc2d8d9b01c9fcff0fe5d32406d086334e65e1
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 266113178
              windows:
                - start: "2026-10-23T22:15:11-07:00"
                  end: "2026-10-24T00:32:10-07:00"
            - harness: codex
              id: 01a16c56-8c0e-78e0-b2f5-5dba1ef231c1
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: cad751272af46a9e5934ffb2815d3ff120c27a623335d2bf4efb539514c514f2
              started: "2026-10-24T05:16:13.771Z"
              spawned: true
              bytes: 1688072
              windows:
                - start: "2026-10-24T05:16:13.771Z"
                  end: "2026-10-24T00:32:10-07:00"
            - harness: codex
              id: 01a16c56-9e08-73d2-9b1d-456be665d9fa
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 109b92b5b63fe5aa2e5fdc73ad0f24d8d65bb23e9ef76863e7deda595ecb8ff2
              started: "2026-10-24T05:16:18.375Z"
              spawned: true
              bytes: 1308144
              windows:
                - start: "2026-10-24T05:16:18.375Z"
                  end: "2026-10-24T00:32:10-07:00"
            - harness: codex
              id: 01a16cb3-0bcf-7da2-b96a-08b03c7b74c6
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 2c0042f643948f8f0d825ec45979ba2d76b8a6722d5ce59318d2c0db0426b671
              started: "2026-10-24T06:57:15.86Z"
              spawned: true
              bytes: 1168656
              windows:
                - start: "2026-10-24T06:57:15.86Z"
                  end: "2026-10-24T00:32:10-07:00"
            - harness: codex
              id: 01a16cb4-ef7a-7111-b5a4-6efe1f28098d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 7edcb57c86f6fb12163ae4325320ac27e28223eff8d42b2ef2e066bb2674ac19
              started: "2026-10-24T06:59:19.636Z"
              spawned: true
              bytes: 865095
              windows:
                - start: "2026-10-24T06:59:19.636Z"
                  end: "2026-10-24T00:32:10-07:00"
        cuts:
            - dispatched: "2026-10-24T00:20:10-07:00"
              finalized: "2026-10-24T00:25:44-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-24T00:36:03-07:00"
      dispatched: "2026-10-24T00:33:22-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:1819–1830 now documents availability from any Pair pane, fullscreen right-terminal reservation, Chrome Send visibility, and the terminal-feedback limitation. These match workbenchshortcut/shortcut.go:183 and couchtty/console_presentation.go:17.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'console_send_test.go:61–118 retains and cleans up the successor, flushes both endpoints, and checks both sinks. In a disposable copy, deliberately routing to the current successor made the replacement test fail at line 114: “successor received 1 send chord(s).”'
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 27f192b8930fe672722112a425c97ca21a5d774e
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 271a74cf-2714-43b6-a954-0e0b16a55683
            repository: github.com/xianxu/pair
            issue_id: "000842"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/d1daaf92c4282285
            reviewed_head: 7b0b8e601bc099c82ec15011de13a310fb69c1bc
            slot: pair:7
            requested_at: "2026-10-24T10:23:01.97201Z"
          phase: finished
          pid: 86301
          started_at: "2026-10-24T10:23:02.124506Z"
          finished_at: "2026-10-24T10:24:47.327701Z"
          session_id: 01a16d6f-718a-7b91-8bc8-678c364bce5b
          outcome: returned
      "n": 3
      timestamp: "2026-10-24T03:24:50-07:00"
      dispatched: "2026-10-24T03:22:59-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:1819–1830 retains the button visibility, wider shortcut availability, and terminal limitation documentation; this interdiff leaves those instructions unchanged.
          round: 3
        - id: BR-2
          disposition: addressed
          note: console_send_test.go:61–118 retains, flushes, and checks the successor endpoint alongside the original endpoint. The focused endpoint-isolation test passes; this interdiff preserves the corrected oracle.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 7b0b8e601bc099c82ec15011de13a310fb69c1bc
      blocked: false
---

# Gate ledger — pair#842 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T00:25:44-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-surface-documentation` (kind: defect) README update appears missing for the Send button and wider shortcut availability
  README.md:1819 still describes Shift+Ctrl+Alt+Return as a draft action, while cmd/internal/workbenchshortcut/shortcut.go:183 makes it available across panes, including fullscreen right-terminal applications. README is unchanged in this range and does not describe the new Chrome status ⏎ control. Update the submission instructions with button visibility, shortcut availability, and the terminal-feedback limitation.
- **BR-2** [Important] `endpoint-isolation-test-oracle` (kind: defect) Pane-replacement test does not observe the successor endpoint
  cmd/internal/couchtty/console_send_test.go:74 creates a successor child, but lines 99–103 flush and count writes only on r.child, the original child. Accidental routing to the successor would therefore satisfy the zero-write assertion. Retain and clean up the successor, flush both endpoints, and assert that neither receives the chord after replacement (ARCH-ORDER).

## Round 2 — 2026-10-24T00:36:03-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — README.md:1819–1830 now documents availability from any Pair pane, fullscreen right-terminal reservation, Chrome Send visibility, and the terminal-feedback limitation. These match workbenchshortcut/shortcut.go:183 and couchtty/console_presentation.go:17.
- BR-2 — addressed — console_send_test.go:61–118 retains and cleans up the successor, flushes both endpoints, and checks both sinks. In a disposable copy, deliberately routing to the current successor made the replacement test fail at line 114: “successor received 1 send chord(s).”

## Round 3 — 2026-10-24T03:24:50-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — README.md:1819–1830 retains the button visibility, wider shortcut availability, and terminal limitation documentation; this interdiff leaves those instructions unchanged.
- BR-2 — addressed — console_send_test.go:61–118 retains, flushes, and checks the successor endpoint alongside the original endpoint. The focused endpoint-isolation test passes; this interdiff preserves the corrected oracle.

## Open findings

(none — every finding has been disposed)
