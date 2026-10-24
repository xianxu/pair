---
gate: boundary-review
issue: 856
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3ead4b48-d9b9-4c34-8887-1a056169ab49
            repository: github.com/xianxu/pair
            issue_id: "000856"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/992c80e3ee587d09
            reviewed_head: 65c775663adf450c6f832785fc6763417fa3636f
            slot: pair:4
            requested_at: "2026-10-24T07:52:31.010654Z"
          phase: finished
          pid: 49790
          started_at: "2026-10-24T07:52:31.143759Z"
          finished_at: "2026-10-24T07:56:17.217766Z"
          session_id: 01a16ce5-a3fa-7431-8739-21dc8ba4ba6b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 65c775663adf450c6f832785fc6763417fa3636f
        tracker_head: 9522e4b2d46cfb4b81530d9a824d8f7a887cc1a3
        branch_point: 99846a5015f753f05edaf4778618849f377a2f71
        since: "2026-10-23T21:35:40-07:00"
        until: "2026-10-24T00:51:41-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T21:35:40-07:00"
              end: "2026-10-24T00:51:41-07:00"
              began: "2026-10-23T21:35:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 00dc915705cfbd1103fb80b9f07f910bf8e9233f5ca954bee2c15944721c3496
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 80512467
              windows:
                - start: "2026-10-23T21:35:40-07:00"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c32-dc67-7af0-9969-fd4afa64378a
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: f1762aee3fc513f59135fb541b58a5dee6046d15998e9c0c83b52228cafc32c1
              started: "2026-10-24T04:37:15.093Z"
              spawned: true
              bytes: 957805
              windows:
                - start: "2026-10-24T04:37:15.093Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c5f-6e79-76d0-9510-40c453020dd0
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 0128526c21ac92b46cd071cf21df055b68e4109a763cf46ac4555d5849bd3d90
              started: "2026-10-24T05:25:56.563Z"
              spawned: false
              bytes: 749881
              windows:
                - start: "2026-10-24T05:25:56.563Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c5f-6fc9-7da3-ba3e-9676f1802243
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 6158a06b6e0c0d9cf09bce4c53261f4ed27cc91be086340caab36dfddf9a89f0
              started: "2026-10-24T05:25:56.442Z"
              spawned: true
              bytes: 402835
              windows:
                - start: "2026-10-24T05:25:56.442Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c5f-98b4-7c83-935e-610f5d49ac8a
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 59ebd92a01e765ded153b151e02a9e94d0ce51e1f308f3e565c5af4989c38639
              started: "2026-10-24T05:26:06.952Z"
              spawned: true
              bytes: 635348
              windows:
                - start: "2026-10-24T05:26:06.952Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c61-dd5a-7830-947d-5796ee2c16dc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 0633061cccdcaca64f996303461da8b84b3ef0c341e6e65c883909d7a84b082c
              started: "2026-10-24T05:28:35.959Z"
              spawned: false
              bytes: 548952
              windows:
                - start: "2026-10-24T05:28:35.959Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c62-999c-7d60-949d-a32b05d14cb6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: b0b7c014d1a3cec051a6fb3238c29a571f20b0c50bcf81aa60a393a3d3cf49b3
              started: "2026-10-24T05:29:24.309Z"
              spawned: false
              bytes: 437678
              windows:
                - start: "2026-10-24T05:29:24.309Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c64-2b75-7372-8d16-5abb181fcd47
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 511777de1841966e0ee3f4d50c2abd68de546cbcc2f67fa898cbcda4ead3edc7
              started: "2026-10-24T05:31:06.584Z"
              spawned: true
              bytes: 260223
              windows:
                - start: "2026-10-24T05:31:06.584Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16c65-02ca-7611-bfdb-a24748eaea02
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 705c50a973fa1ee2b3e93ce702293694649a3275c45b9fa1d122e851693d1a29
              started: "2026-10-24T05:32:01.812Z"
              spawned: true
              bytes: 225090
              windows:
                - start: "2026-10-24T05:32:01.812Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16ccc-d58c-77e2-8ff3-3052bfa76ada
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 0697b9fa1239f0798916d1bd90452cb2c170c056d00b2bc1d754bf705592d93e
              started: "2026-10-24T07:25:25.957Z"
              spawned: true
              bytes: 966117
              windows:
                - start: "2026-10-24T07:25:25.957Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16ccd-21f1-7fe1-9f91-8ad882965aa8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 1df020ac626f407e7ec3694c1c50e5f2ae548fc066f232c7053bb07005680911
              started: "2026-10-24T07:25:45.58Z"
              spawned: true
              bytes: 804048
              windows:
                - start: "2026-10-24T07:25:45.58Z"
                  end: "2026-10-24T00:51:41-07:00"
            - harness: codex
              id: 01a16ccd-fb6c-74e3-bd30-950ae1c123e7
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 98a7355c90df269d4d5e574ece0fccab2948ed5308812001115b2e80708b261c
              started: "2026-10-24T07:26:41.167Z"
              spawned: true
              bytes: 1013592
              windows:
                - start: "2026-10-24T07:26:41.167Z"
                  end: "2026-10-24T00:51:41-07:00"
      "n": 1
      timestamp: "2026-10-24T00:56:18-07:00"
      dispatched: "2026-10-24T00:52:28-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: The unified Notes panel loses sketches when their ink leaves the viewport
          detail: cmd/internal/webcast/web/sketch.js:836 feeds the sole Notes surface through sketchNoteEntries, whose web/sketch-notes.js:6-8 rejects missing or offscreen painted bounds. Reproduction with the same completed sketch returns one row for visible bounds and zero rows for offscreen or null bounds. With the separate preview cards removed, its label and Show on page action become inaccessible, contrary to the issue's one-entry-per-sketch contract (ARCH-PURPOSE). Derive row membership from accepted sketch identity/metadata independently of placement; use geometry only for availability, glow and reveal, and add scroll-away/missing-anchor regressions.
          family: feedback-membership-independent-of-placement
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Annotation hover glow scales its origin but leaves its dimensions unscaled
          detail: cmd/internal/webcast/web/annotate.js:657-661 transforms the box and clip origins through v.fromPage but passes their page-space widths/heights directly to canvas. A production mount with viewer scale 0.5 paints box [10,20,80,40] as [5,10,80,40], rather than [5,10,40,20]; its clip is similarly twice the intended size. This violates the confirmed-geometry hover contract (ARCH-PURPOSE). Transform both corners of both rectangles, as the existing place helper does, and test non-unit scale plus clipping through the real hover handler.
          family: page-to-client-rectangle-projection
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: The minimized Notes control gives no indication of pending status
          detail: cmd/internal/webcast/web/feedback-panel.js:128-133 writes status only inside the header; web/floating-panel.js:111-112 hides that header when minimized and always labels the remaining control Notes (N). Reproduction after publishing a stale-preview warning leaves the visible control unchanged. The plan explicitly requires the minimized Notes control to convey status (workshop/plans/000856-cast-feedback-workspace-plan.md:62; ARCH-PURPOSE). Add a visible/accessibly named status indication to the minimized control and cover status arrival, clearing and restoration.
          family: minimized-status-observability
          round: 1
        - kind: unverifiable
          id: BR-4
          severity: Important
          title: Native browser interaction and layout qualification could not run
          detail: 'Attempted WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run ''^(TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorScript.*)$'' -count=1 -timeout 120s -v. Both native tests failed before opening a page: Chrome reported GPU process exit_code=6 and FATAL GPU process isn''t usable. Thus native hit testing, drawing/editor integration and toolbar separation remain unverified here (ARCH-MOCK, ARCH-CONSTRAINTS). Provide non-skipped passing output tied to the reviewed source from a working Chrome host, including viewport geometry assertions, or restore browser startup in this environment and rerun.'
          family: native-ui-qualification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c9ca4987-1d23-4678-be3a-8c7188e27ffc
            repository: github.com/xianxu/pair
            issue_id: "000856"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/127a30ea17f8f3f4
            reviewed_head: 79bac7f1c24df9f034e42b08b01315bff48b3b25
            slot: pair:4
            requested_at: "2026-10-24T08:06:09.907947Z"
          phase: finished
          pid: 22078
          started_at: "2026-10-24T08:06:10.089741Z"
          finished_at: "2026-10-24T08:08:21.198784Z"
          session_id: 01a16cf2-2369-7b92-8d59-cc92924b8bd2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 79bac7f1c24df9f034e42b08b01315bff48b3b25
        tracker_head: 9522e4b2d46cfb4b81530d9a824d8f7a887cc1a3
        branch_point: 99846a5015f753f05edaf4778618849f377a2f71
        since: "2026-10-23T21:35:40-07:00"
        until: "2026-10-24T01:01:14-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T21:35:40-07:00"
              end: "2026-10-24T01:01:14-07:00"
              began: "2026-10-23T21:35:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: cc65e2b7624c51248dc718ac6b80520d0cb5b6e4fb9c00b03fd3df2796d01c4e
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 81362520
              windows:
                - start: "2026-10-23T21:35:40-07:00"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c32-dc67-7af0-9969-fd4afa64378a
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: f1762aee3fc513f59135fb541b58a5dee6046d15998e9c0c83b52228cafc32c1
              started: "2026-10-24T04:37:15.093Z"
              spawned: true
              bytes: 957805
              windows:
                - start: "2026-10-24T04:37:15.093Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c5f-6e79-76d0-9510-40c453020dd0
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 0128526c21ac92b46cd071cf21df055b68e4109a763cf46ac4555d5849bd3d90
              started: "2026-10-24T05:25:56.563Z"
              spawned: false
              bytes: 749881
              windows:
                - start: "2026-10-24T05:25:56.563Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c5f-6fc9-7da3-ba3e-9676f1802243
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 6158a06b6e0c0d9cf09bce4c53261f4ed27cc91be086340caab36dfddf9a89f0
              started: "2026-10-24T05:25:56.442Z"
              spawned: true
              bytes: 402835
              windows:
                - start: "2026-10-24T05:25:56.442Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c5f-98b4-7c83-935e-610f5d49ac8a
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 59ebd92a01e765ded153b151e02a9e94d0ce51e1f308f3e565c5af4989c38639
              started: "2026-10-24T05:26:06.952Z"
              spawned: true
              bytes: 635348
              windows:
                - start: "2026-10-24T05:26:06.952Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c61-dd5a-7830-947d-5796ee2c16dc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 0633061cccdcaca64f996303461da8b84b3ef0c341e6e65c883909d7a84b082c
              started: "2026-10-24T05:28:35.959Z"
              spawned: false
              bytes: 548952
              windows:
                - start: "2026-10-24T05:28:35.959Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c62-999c-7d60-949d-a32b05d14cb6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: b0b7c014d1a3cec051a6fb3238c29a571f20b0c50bcf81aa60a393a3d3cf49b3
              started: "2026-10-24T05:29:24.309Z"
              spawned: false
              bytes: 437678
              windows:
                - start: "2026-10-24T05:29:24.309Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c64-2b75-7372-8d16-5abb181fcd47
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 511777de1841966e0ee3f4d50c2abd68de546cbcc2f67fa898cbcda4ead3edc7
              started: "2026-10-24T05:31:06.584Z"
              spawned: true
              bytes: 260223
              windows:
                - start: "2026-10-24T05:31:06.584Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16c65-02ca-7611-bfdb-a24748eaea02
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 705c50a973fa1ee2b3e93ce702293694649a3275c45b9fa1d122e851693d1a29
              started: "2026-10-24T05:32:01.812Z"
              spawned: true
              bytes: 225090
              windows:
                - start: "2026-10-24T05:32:01.812Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16ccc-d58c-77e2-8ff3-3052bfa76ada
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 29dac59e97aed32f2992ce31a0103056a923db61f8381958b1daa69c160c9d0c
              started: "2026-10-24T07:25:25.957Z"
              spawned: true
              bytes: 1182635
              windows:
                - start: "2026-10-24T07:25:25.957Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16ccd-21f1-7fe1-9f91-8ad882965aa8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 1df020ac626f407e7ec3694c1c50e5f2ae548fc066f232c7053bb07005680911
              started: "2026-10-24T07:25:45.58Z"
              spawned: true
              bytes: 804048
              windows:
                - start: "2026-10-24T07:25:45.58Z"
                  end: "2026-10-24T01:01:14-07:00"
            - harness: codex
              id: 01a16ccd-fb6c-74e3-bd30-950ae1c123e7
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5779d0d7a70e0e49f6e09a2d8fca15bc38bae91939d581e59c7098d0015d4aa3
              started: "2026-10-24T07:26:41.167Z"
              spawned: true
              bytes: 1504724
              windows:
                - start: "2026-10-24T07:26:41.167Z"
                  end: "2026-10-24T01:01:14-07:00"
        cuts:
            - dispatched: "2026-10-24T00:52:28-07:00"
              finalized: "2026-10-24T00:56:18-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-24T01:08:46-07:00"
      dispatched: "2026-10-24T01:03:16-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: sketch-notes.js now derives membership independently of bounds. Pure and mounted-panel regressions cover offscreen, missing and restored placement; restoring the previous helper caused six failures.
          round: 2
        - id: BR-2
          disposition: addressed
          note: annotate.js projects both corners of box and clip through clientBox. Mounted hover tests cover non-unit, nonuniform scales; restoring unscaled dimensions caused both geometry regressions to fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Minimized Notes visibly indicates Status and exposes the full message through its accessible name and tooltip. Tests cover arrival, clearing, restoration and clamping; removing the visible suffix failed the regression.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Retained native-review-fixes.log supplies eight non-skipped passes on 791e8742c, whose cmd/ and tests/ trees match this head. However, cast_notes_panel_live_test.go:32-42 mounts only the standalone panel, without cast-tools CSS, toolbar or protected workspace layout; its 1280/320 bounds assertions cannot establish toolbar separation. operator_sketch_live_test.go:162 checks button reachability, without the required viewport separation matrix. The repeated WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^(TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorLiveAnnotation|TestOperatorFeedbackLivePrepareAndStalePanel|TestOperatorSketchLiveOpenContactHandoffLabel|TestOperatorModuleAssembly|TestOperatorSharedModulesExecute|TestOperatorSketchSharedBundleExecutes)$' -count=1 -v -timeout 180s still failed native startup with GPU exit_code=6. Supply source-bound, non-skipped host results measuring the actual toolbar against expanded/minimized Notes, editors and status at common desktop/narrow widths (ARCH-MOCK, ARCH-CONSTRAINTS).
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#856 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T00:56:18-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `feedback-membership-independent-of-placement` (kind: defect) The unified Notes panel loses sketches when their ink leaves the viewport
  cmd/internal/webcast/web/sketch.js:836 feeds the sole Notes surface through sketchNoteEntries, whose web/sketch-notes.js:6-8 rejects missing or offscreen painted bounds. Reproduction with the same completed sketch returns one row for visible bounds and zero rows for offscreen or null bounds. With the separate preview cards removed, its label and Show on page action become inaccessible, contrary to the issue's one-entry-per-sketch contract (ARCH-PURPOSE). Derive row membership from accepted sketch identity/metadata independently of placement; use geometry only for availability, glow and reveal, and add scroll-away/missing-anchor regressions.
- **BR-2** [Critical] `page-to-client-rectangle-projection` (kind: defect) Annotation hover glow scales its origin but leaves its dimensions unscaled
  cmd/internal/webcast/web/annotate.js:657-661 transforms the box and clip origins through v.fromPage but passes their page-space widths/heights directly to canvas. A production mount with viewer scale 0.5 paints box [10,20,80,40] as [5,10,80,40], rather than [5,10,40,20]; its clip is similarly twice the intended size. This violates the confirmed-geometry hover contract (ARCH-PURPOSE). Transform both corners of both rectangles, as the existing place helper does, and test non-unit scale plus clipping through the real hover handler.
- **BR-3** [Important] `minimized-status-observability` (kind: defect) The minimized Notes control gives no indication of pending status
  cmd/internal/webcast/web/feedback-panel.js:128-133 writes status only inside the header; web/floating-panel.js:111-112 hides that header when minimized and always labels the remaining control Notes (N). Reproduction after publishing a stale-preview warning leaves the visible control unchanged. The plan explicitly requires the minimized Notes control to convey status (workshop/plans/000856-cast-feedback-workspace-plan.md:62; ARCH-PURPOSE). Add a visible/accessibly named status indication to the minimized control and cover status arrival, clearing and restoration.
- **BR-4** [Important] `native-ui-qualification` (kind: unverifiable) Native browser interaction and layout qualification could not run
  Attempted WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^(TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorScript.*)$' -count=1 -timeout 120s -v. Both native tests failed before opening a page: Chrome reported GPU process exit_code=6 and FATAL GPU process isn't usable. Thus native hit testing, drawing/editor integration and toolbar separation remain unverified here (ARCH-MOCK, ARCH-CONSTRAINTS). Provide non-skipped passing output tied to the reviewed source from a working Chrome host, including viewport geometry assertions, or restore browser startup in this environment and rerun.

## Round 2 — 2026-10-24T01:08:46-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — sketch-notes.js now derives membership independently of bounds. Pure and mounted-panel regressions cover offscreen, missing and restored placement; restoring the previous helper caused six failures.
- BR-2 — addressed — annotate.js projects both corners of box and clip through clientBox. Mounted hover tests cover non-unit, nonuniform scales; restoring unscaled dimensions caused both geometry regressions to fail.
- BR-3 — addressed — Minimized Notes visibly indicates Status and exposes the full message through its accessible name and tooltip. Tests cover arrival, clearing, restoration and clamping; removing the visible suffix failed the regression.
- BR-4 — not-addressed — Retained native-review-fixes.log supplies eight non-skipped passes on 791e8742c, whose cmd/ and tests/ trees match this head. However, cast_notes_panel_live_test.go:32-42 mounts only the standalone panel, without cast-tools CSS, toolbar or protected workspace layout; its 1280/320 bounds assertions cannot establish toolbar separation. operator_sketch_live_test.go:162 checks button reachability, without the required viewport separation matrix. The repeated WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^(TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorLiveAnnotation|TestOperatorFeedbackLivePrepareAndStalePanel|TestOperatorSketchLiveOpenContactHandoffLabel|TestOperatorModuleAssembly|TestOperatorSharedModulesExecute|TestOperatorSketchSharedBundleExecutes)$' -count=1 -v -timeout 180s still failed native startup with GPU exit_code=6. Supply source-bound, non-skipped host results measuring the actual toolbar against expanded/minimized Notes, editors and status at common desktop/narrow widths (ARCH-MOCK, ARCH-CONSTRAINTS).

## Open findings

- **BR-4** [Important] `native-ui-qualification` (kind: unverifiable) Native browser interaction and layout qualification could not run
