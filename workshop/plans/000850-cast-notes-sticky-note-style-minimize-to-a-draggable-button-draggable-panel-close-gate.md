---
gate: boundary-review
issue: 850
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2bce9d48-e357-4601-a878-7130b97597a1
            repository: github.com/xianxu/pair
            issue_id: "000850"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/c486f21bfc290e0a
            reviewed_head: 1c2a16c82e6670aea3739eb5da1c18a2fa8ae73a
            slot: pair:4
            requested_at: "2026-10-24T02:56:09.343153Z"
          phase: finished
          pid: 88649
          started_at: "2026-10-24T02:56:09.494632Z"
          finished_at: "2026-10-24T02:59:07.298822Z"
          session_id: 01a16bd6-509f-7691-b2c5-4784dbb562bc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1c2a16c82e6670aea3739eb5da1c18a2fa8ae73a
        tracker_head: f248cd76ef957ef2a0a647f233cb776a05ebd65c
        branch_point: 221a833b2dc8504a9498d2f5b4b944b99ea7907d
        since: "2026-10-23T19:27:12-07:00"
        until: "2026-10-23T19:54:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T19:27:12-07:00"
              end: "2026-10-23T19:54:10-07:00"
              began: "2026-10-23T19:27:12-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: d20569c7ac8ba802765e954fb6fdb3550582cacb367cf1060d8f590a0ada60f9
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 72123546
              windows:
                - start: "2026-10-23T19:27:12-07:00"
                  end: "2026-10-23T19:54:10-07:00"
            - harness: codex
              id: 01a16bc4-3047-7c21-b815-b4015e6a22a6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 553061ea07be0a00fe4488b83ef72e95142fdac5cd359c4bdcdf8bfa1632c06f
              started: "2026-10-24T02:36:22.651Z"
              spawned: false
              bytes: 468087
              windows:
                - start: "2026-10-24T02:36:22.651Z"
                  end: "2026-10-23T19:54:10-07:00"
            - harness: codex
              id: 01a16bc6-10ec-7ff1-a5fc-f87626ddbce3
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4519288b8feb9467db258f866edc2cb893f14503a055c8526b7bfa7f4a38470b
              started: "2026-10-24T02:38:25.484Z"
              spawned: false
              bytes: 407441
              windows:
                - start: "2026-10-24T02:38:25.484Z"
                  end: "2026-10-23T19:54:10-07:00"
            - harness: codex
              id: 01a16bc7-ca93-7801-bb78-7f02c258a5da
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: b31e24f5efca06e75145aa1911e9d52cd5573fef1c9579ad46c9dc183b73208b
              started: "2026-10-24T02:40:18.938Z"
              spawned: false
              bytes: 434431
              windows:
                - start: "2026-10-24T02:40:18.938Z"
                  end: "2026-10-23T19:54:10-07:00"
            - harness: codex
              id: 01a16bc8-65c5-7803-81dd-735857bada64
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 3642c7f20db4787b7323631c927f95ed1d1ba47c4dbc756ceabcb458cfbc8160
              started: "2026-10-24T02:40:58.955Z"
              spawned: false
              bytes: 386042
              windows:
                - start: "2026-10-24T02:40:58.955Z"
                  end: "2026-10-23T19:54:10-07:00"
            - harness: codex
              id: 01a16bc9-8056-7df2-bd92-70de1ca26cc5
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 4bbbf44c4c76f504747bdddfe094155890d3dfaded4f2b6d35b2ffd5f20af663
              started: "2026-10-24T02:42:10.338Z"
              spawned: true
              bytes: 856390
              windows:
                - start: "2026-10-24T02:42:10.338Z"
                  end: "2026-10-23T19:54:10-07:00"
            - harness: codex
              id: 01a16bcd-4c16-76c0-b49c-d1938d780928
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: aa625e0892d0d3d3393fd07754a74fde709e2758178d50a757af79be39293f4c
              started: "2026-10-24T02:46:18.996Z"
              spawned: true
              bytes: 899778
              windows:
                - start: "2026-10-24T02:46:18.996Z"
                  end: "2026-10-23T19:54:10-07:00"
      "n": 1
      timestamp: "2026-10-23T19:59:08-07:00"
      dispatched: "2026-10-23T19:56:06-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Visible sketches can lose their note control after scrolling.
          detail: cmd/internal/webcast/web/sketch-notes.js:12–19 anchors each sketch control exclusively to its first point; lines 52–54 position it without viewport clamping, while sketch.css:83 clips overflow. A production-function reproduction with a stroke spanning page y=100–700 and placement shifted upward 500px leaves visible ink at viewport y=0–200 but positions View note at top:-392px. This breaks the promised visible control for on-view sketches (ARCH-PURPOSE, ARCH-CONSTRAINTS). Derive the control from visible painted bounds, keep its complete hit area reachable, and cover scrolling, multiple paths, and viewport edges.
          family: viewport-reachability
          round: 1
        - id: BR-2
          severity: Important
          title: Updating the minimized count does not reclamp the growing button.
          detail: cmd/internal/webcast/web/floating-panel.js:150–153 changes the Notes (N) text without measuring its new size or sending a geometry transition. After moving the minimized button to the right boundary, increasing the count can extend it past the viewport. A variable-width DOM reproduction changed its right edge from 788px to 812px in an 800px viewport. Remeasure and reclamp after size-changing updates, preserving mode and keeping drag movement constant-cost; add a count-growth regression (ARCH-CONSTRAINTS).
          family: viewport-reachability
          round: 1
        - id: BR-3
          severity: Important
          title: Mode changes mutate authoritative state outside the transition function.
          detail: cmd/internal/webcast/web/floating-panel.js:104 directly assigns state.mode before calling floatingPanelTransition. This bypasses the plan's model-owned state boundary and ARCH-ORDER's structural requirement. Route the mode change through the reducer, render that state, then submit measured geometry as a separate event; test the production adapter's transition sequence.
          family: authoritative-transition-boundary
          round: 1
        - kind: unverifiable
          id: BR-4
          severity: Important
          title: Required native browser and operator acceptance remain unverified.
          detail: 'Running WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run ''TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorScript|TestServer'' -count=1 -timeout 3m -v failed both native tests before opening a page: Chrome reported GPU process exit_code=6 and “GPU process isn''t usable.” The issue and plan also leave the staged operator look outstanding. Provide non-skipped passing browser results for the corrected head on a working host and recorded completion of the specified operator look.'
          family: boundary-verification-evidence
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#850 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T19:59:08-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `viewport-reachability` Visible sketches can lose their note control after scrolling.
  cmd/internal/webcast/web/sketch-notes.js:12–19 anchors each sketch control exclusively to its first point; lines 52–54 position it without viewport clamping, while sketch.css:83 clips overflow. A production-function reproduction with a stroke spanning page y=100–700 and placement shifted upward 500px leaves visible ink at viewport y=0–200 but positions View note at top:-392px. This breaks the promised visible control for on-view sketches (ARCH-PURPOSE, ARCH-CONSTRAINTS). Derive the control from visible painted bounds, keep its complete hit area reachable, and cover scrolling, multiple paths, and viewport edges.
- **BR-2** [Important] `viewport-reachability` Updating the minimized count does not reclamp the growing button.
  cmd/internal/webcast/web/floating-panel.js:150–153 changes the Notes (N) text without measuring its new size or sending a geometry transition. After moving the minimized button to the right boundary, increasing the count can extend it past the viewport. A variable-width DOM reproduction changed its right edge from 788px to 812px in an 800px viewport. Remeasure and reclamp after size-changing updates, preserving mode and keeping drag movement constant-cost; add a count-growth regression (ARCH-CONSTRAINTS).
- **BR-3** [Important] `authoritative-transition-boundary` Mode changes mutate authoritative state outside the transition function.
  cmd/internal/webcast/web/floating-panel.js:104 directly assigns state.mode before calling floatingPanelTransition. This bypasses the plan's model-owned state boundary and ARCH-ORDER's structural requirement. Route the mode change through the reducer, render that state, then submit measured geometry as a separate event; test the production adapter's transition sequence.
- **BR-4** [Important] `boundary-verification-evidence` (kind: unverifiable) Required native browser and operator acceptance remain unverified.
  Running WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorScript|TestServer' -count=1 -timeout 3m -v failed both native tests before opening a page: Chrome reported GPU process exit_code=6 and “GPU process isn't usable.” The issue and plan also leave the staged operator look outstanding. Provide non-skipped passing browser results for the corrected head on a working host and recorded completion of the specified operator look.

## Open findings

- **BR-1** [Critical] `viewport-reachability` Visible sketches can lose their note control after scrolling.
- **BR-2** [Important] `viewport-reachability` Updating the minimized count does not reclamp the growing button.
- **BR-3** [Important] `authoritative-transition-boundary` Mode changes mutate authoritative state outside the transition function.
- **BR-4** [Important] `boundary-verification-evidence` (kind: unverifiable) Required native browser and operator acceptance remain unverified.
