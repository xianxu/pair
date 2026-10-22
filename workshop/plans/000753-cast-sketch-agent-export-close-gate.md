---
gate: boundary-review
issue: 753
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 20c83b16-0476-44c9-92fa-909102ca7784
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/bfa179fdcc579e45
            reviewed_head: 813350c3bbf43e64f2e37d1381a99035840ce371
            slot: pair:9
            requested_at: "2026-10-22T03:07:49.277355Z"
          phase: finished
          pid: 29862
          started_at: "2026-10-22T03:07:49.44463Z"
          finished_at: "2026-10-22T03:13:16.906191Z"
          session_id: 01a16194-4716-7602-8b3a-edf738cbaead
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 813350c3bbf43e64f2e37d1381a99035840ce371
        tracker_head: 7f976d8659b5e617bfb7e76f70ec06509746e8c8
        branch_point: d2540fd6ae6eaa06e8a90217b285c22116247558
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-21T20:07:08-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-21T20:07:08-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 7f1980ad00239a513b6a0236ec3b72808d583ba0a543940d3f027ac18a9c0f17
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 22798371
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-21T20:07:08-07:00"
            - harness: codex
              id: 01a1615b-b3b2-71d3-878e-78e13474733a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 875cc0385845359821419e9c1c22b64d19d3a1a1c51b8bd89a7644ade60bfd46
              started: "2026-10-22T02:06:02.68Z"
              spawned: false
              bytes: 781349
              windows:
                - start: "2026-10-22T02:06:02.68Z"
                  end: "2026-10-21T20:07:08-07:00"
            - harness: codex
              id: 01a1615f-3510-79f2-9cae-31cf36613788
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c47b782ebe76c153ab719771b1df19f3a47df32607a4dfc8f027c6246cc3538d
              started: "2026-10-22T02:09:52.302Z"
              spawned: false
              bytes: 462083
              windows:
                - start: "2026-10-22T02:09:52.302Z"
                  end: "2026-10-21T20:07:08-07:00"
            - harness: codex
              id: 01a16161-37d2-7071-bfc4-bdb6afa7db78
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 08a97e6e51d0b85438c720071e9692010efbf73613a778bae018f6757d5213af
              started: "2026-10-22T02:12:03.967Z"
              spawned: false
              bytes: 465728
              windows:
                - start: "2026-10-22T02:12:03.967Z"
                  end: "2026-10-21T20:07:08-07:00"
            - harness: codex
              id: 01a16161-e237-7f33-bb66-192582ab999d
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e8b1541e3537a5f034fd0ec8963fa2ff873e31d2608ab45319e221eb5a788811
              started: "2026-10-22T02:12:47.67Z"
              spawned: false
              bytes: 447786
              windows:
                - start: "2026-10-22T02:12:47.67Z"
                  end: "2026-10-21T20:07:08-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: feb8652d41443099c1045fbeb2990d6f718609930bacdc55f71dd5dc14c94b58
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 2793241
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-21T20:07:08-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 238925860df3cc2393d3fc161d674783f9b2a63a44b621452357dea7a71c4211
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 1876933
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-21T20:07:08-07:00"
      "n": 1
      timestamp: "2026-10-21T20:13:18-07:00"
      dispatched: "2026-10-21T20:07:45-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Canceled preview publications can remain permanently Publishing
          detail: 'cmd/internal/couchtty/console_submission_preview.go:154 cancels and joins a publication after a refresh notification, then continues without settling its state. If host and candidates are unchanged, NextPreview ignores both repeated inputs and PreviewRefresh cannot leave Publishing. Production SubmissionPreviewUpdates preloads precisely such a notification at chrome_submission_preview.go:125; receiver-progress notifications can also reproduce it. A held publisher plus a buffered unchanged-host notification deterministically failed the review probe: no replacement publication occurred and preparation remained refused. ARCH-ORDER: settle the canceled publication through the transition model before refreshing, and cover initial, duplicate, and catchup notifications.'
          family: canceled-operation-state-settlement
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Unrelated DOM attributes consume the budget before mandatory anchor facts
          detail: 'cmd/internal/webcast/query/element_facts.go:98 spends the shared 64 KiB attribute budget during whole-DOM traversal, before facts() prioritizes anchors. A probe with 70 preceding unrelated elements carrying bounded data attributes and a later anchor with a valid 1024-byte ID lost that ID, although the selected anchor facts easily fit the output budget. ARCH-PURPOSE: reserve or allocate declared-fact bytes in anchor-first, ancestor-next, nearest-context order; test starvation by unselected nodes.'
          family: mandatory-evidence-budget-priority
          round: 1
        - kind: defect
          id: BR-3
          severity: Critical
          title: Valid region-only notes now fail submission preparation
          detail: cmd/internal/webcast/query/querier.go:225 always inserts n.At.Node into the anchor list, including zero. The resulting synthetic node-zero fact is rejected by cmd/internal/submission/sketch.go:138. recording/record.go:548 explicitly permits a note with a region and no node; a review probe successfully encoded those notes but Build failed with "invalid declared element fact". Preserve the existing note contract by omitting absent anchor handles while retaining region intersection facts, and add a region-only submission regression.
          family: optional-anchor-compatibility
          round: 1
        - kind: defect
          id: BR-4
          severity: Critical
          title: Null pressure passes decoding and validation as fabricated zero pressure
          detail: 'cmd/internal/submission/codec.go:103 checks tuple length but not numeric member types, then json.Unmarshal at line 89 converts null into a float''s zero value. Replacing [20,20,0.2] with [20,20,null] in a valid sketch manifest passed both DecodePreserving and Validate in the review probe, yielding pressure 0. ARCH-SECURE/ARCH-EVOLVE: reject null and other invalid types for required numeric evidence, sweep the known scalar and tuple schemas, and add strict-type regressions.'
          family: strict-known-field-decoding
          round: 1
        - kind: defect
          id: BR-5
          severity: Critical
          title: The Core concepts inventory contradicts implemented entities and boundaries
          detail: 'workshop/plans/000753-cast-sketch-agent-export-plan.md:70-83 lists PreviewSelection in sketch.go, a PURE DeliveredSketchSet in sketch.go, query/render_sketch.go, and webcast/submission/sketch.go. Those definitions/files do not exist at those locations: PreviewSelection lives in preview.go, delivery derivation is the filesystem-backed Store.DeliveredSketchVersions in store.go, and export is implemented through sketch_export.go and sketch_image_export.go. ARCH-PURE: revise the inventory to name actual entities and paths, separate pure selection from integration delivery derivation, and record the correction under Revisions.'
          family: core-concept-traceability
          round: 1
        - kind: unverifiable
          id: BR-6
          severity: Important
          title: Native historical rendering and fidelity preservation remain unverified
          detail: 'Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run ''TestLiveSketchImage|TestOperatorSketchLiveDrawing|TestLiveFidelity'' -count=1 -v. Fidelity fixtures, broken-render detection, operator drawing, and historical pressure/batch rendering failed with Chrome/CDP EOF; operator Chrome reported "GPU process isn''t usable", and historical rendering also reported "operation not permitted". Cancellation cleanup passed. ARCH-MOCK: provide passing results for these checks on the final source-equivalent head in a supported native environment, including pressure pixels, eight-image timing, fidelity preservation, and joined cleanup.'
          family: native-conformance-evidence
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4a17e301-afe7-4298-9075-c5cf61a13aed
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/f8d411e70d5c53d7
            reviewed_head: 4c2bd430be755eae08053496e0babe913f6382a9
            slot: pair:9
            requested_at: "2026-10-22T03:35:27.204559Z"
          phase: finished
          pid: 5571
          started_at: "2026-10-22T03:35:27.439719Z"
          finished_at: "2026-10-22T03:39:27.259077Z"
          session_id: 01a161ad-93b8-7e93-a1dc-3a79e861cac9
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4c2bd430be755eae08053496e0babe913f6382a9
        tracker_head: 73a5750606dd781a91e683d951b202c90f24be26
        branch_point: d2540fd6ae6eaa06e8a90217b285c22116247558
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-21T20:34:31-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-21T20:34:31-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 9e1f30655d13e3f9fd9ddb6f4cb199ea48db666ace23bbfea5f1f21b60895a74
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 24612216
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-21T20:34:31-07:00"
            - harness: codex
              id: 01a1615b-b3b2-71d3-878e-78e13474733a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 875cc0385845359821419e9c1c22b64d19d3a1a1c51b8bd89a7644ade60bfd46
              started: "2026-10-22T02:06:02.68Z"
              spawned: false
              bytes: 781349
              windows:
                - start: "2026-10-22T02:06:02.68Z"
                  end: "2026-10-21T20:34:31-07:00"
            - harness: codex
              id: 01a1615f-3510-79f2-9cae-31cf36613788
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c47b782ebe76c153ab719771b1df19f3a47df32607a4dfc8f027c6246cc3538d
              started: "2026-10-22T02:09:52.302Z"
              spawned: false
              bytes: 462083
              windows:
                - start: "2026-10-22T02:09:52.302Z"
                  end: "2026-10-21T20:34:31-07:00"
            - harness: codex
              id: 01a16161-37d2-7071-bfc4-bdb6afa7db78
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 08a97e6e51d0b85438c720071e9692010efbf73613a778bae018f6757d5213af
              started: "2026-10-22T02:12:03.967Z"
              spawned: false
              bytes: 465728
              windows:
                - start: "2026-10-22T02:12:03.967Z"
                  end: "2026-10-21T20:34:31-07:00"
            - harness: codex
              id: 01a16161-e237-7f33-bb66-192582ab999d
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e8b1541e3537a5f034fd0ec8963fa2ff873e31d2608ab45319e221eb5a788811
              started: "2026-10-22T02:12:47.67Z"
              spawned: false
              bytes: 447786
              windows:
                - start: "2026-10-22T02:12:47.67Z"
                  end: "2026-10-21T20:34:31-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 6e28d3e8df6b95fe77b608f09974c68589a9ba68d14282999a3f6cab214a0f18
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 3120074
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-21T20:34:31-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 386c931ce2f4e0a5f51a02c837ed09bb4a7381f30df53db0cad9987656567116
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 2260836
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-21T20:34:31-07:00"
        cuts:
            - dispatched: "2026-10-21T20:07:45-07:00"
              finalized: "2026-10-21T20:13:18-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T20:39:29-07:00"
      dispatched: "2026-10-21T20:35:23-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Cancellation now settles the exact publication through PreviewPublishFailed. The checked-in regression passes on head and fails with pre-fix production code for all five unchanged-input notification cases.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Selected facts now consume the output budget in anchor-first order with mandatory-handle reservations. The starvation regression passes on head and fails with the previous implementation; unused measurement fields were retained solely to compile the counterfactual.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Absent anchor handles are omitted by both note projection and the shared extractor. The region-only Build/ContextAt regression passes on head and fails with the pre-fix implementations.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Numeric tuple members now reject null before Go decoding. The null-pressure regression passes on head and fails with the previous codec, which produces pressure zero and passes Validate. Missing-field handling is a separate remaining instance reported below.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Plan lines 62–91 now locate PreviewSelection in preview.go, distinguish pure SelectSketchPreview from INTEGRATION Store.DeliveredSketchVersions, and name the actual export files. These match the definitions in preview.go, preview_selection.go, store.go, sketch_export.go and sketch_image_export.go; Revisions records the correction.
          round: 2
        - id: BR-6
          disposition: withdrawn
          note: Inspected native-85406e681.log and verified SHA256 d7942734429e891879c962c39102fc7781eaf7f3c4d9daa39e82440efae7799e. The requested native checks passed in 77.227s, including pressure/batch rendering, fidelity fixtures, broken-render detection and cancellation cleanup. The eight-image measurement was 1.796754583s. git diff confirms the reviewed head changes only issue prose after that source commit.
          round: 2
      findings:
        - kind: defect
          id: BR-7
          severity: Critical
          title: Missing required sketch source fields become fabricated zero-valued evidence
          detail: 'cmd/internal/submission/codec.go:189 validates only fields present in the JSON object; it never checks required-member presence. Three independent probes deleting sketches[0].source.t, record_index, or scroll from the valid checked-in fixture all passed DecodePreserving and Validate: source time 10 became 0, record index 2 became 0, and absent scroll became [0,0]. This violates the strict required-field and original-source evidence contracts (ARCH-SECURE, ARCH-EVOLVE). This is the 2nd finding in family strict-known-field-decoding: do not patch only these three fields. Define required versus optional members across the new evidence schema, enforce presence before decoding, and sweep omission regressions across that inventory while preserving intentional legacy defaults; measured prevalence here is 3/3 probed source members.'
          family: strict-known-field-decoding
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c51462ad-6097-4714-be11-f1f600ea7f11
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/4a867103b057ef59
            reviewed_head: 052330f68d4f81b9baff10d90e3559c6a5abaf55
            slot: pair:9
            requested_at: "2026-10-22T03:47:51.997749Z"
          phase: finished
          pid: 26082
          started_at: "2026-10-22T03:47:52.170972Z"
          finished_at: "2026-10-22T03:50:40.974503Z"
          session_id: 01a161b8-f06b-7eb2-9d9e-069e4223328a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 052330f68d4f81b9baff10d90e3559c6a5abaf55
        tracker_head: 73a5750606dd781a91e683d951b202c90f24be26
        branch_point: d2540fd6ae6eaa06e8a90217b285c22116247558
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-21T20:46:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-21T20:46:55-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: faffeca1a3cdfdedcbd040e819b89ef6e96764329010894d60570a4ad7908b87
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 25463723
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-21T20:46:55-07:00"
            - harness: codex
              id: 01a1615b-b3b2-71d3-878e-78e13474733a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 875cc0385845359821419e9c1c22b64d19d3a1a1c51b8bd89a7644ade60bfd46
              started: "2026-10-22T02:06:02.68Z"
              spawned: false
              bytes: 781349
              windows:
                - start: "2026-10-22T02:06:02.68Z"
                  end: "2026-10-21T20:46:55-07:00"
            - harness: codex
              id: 01a1615f-3510-79f2-9cae-31cf36613788
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c47b782ebe76c153ab719771b1df19f3a47df32607a4dfc8f027c6246cc3538d
              started: "2026-10-22T02:09:52.302Z"
              spawned: false
              bytes: 462083
              windows:
                - start: "2026-10-22T02:09:52.302Z"
                  end: "2026-10-21T20:46:55-07:00"
            - harness: codex
              id: 01a16161-37d2-7071-bfc4-bdb6afa7db78
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 08a97e6e51d0b85438c720071e9692010efbf73613a778bae018f6757d5213af
              started: "2026-10-22T02:12:03.967Z"
              spawned: false
              bytes: 465728
              windows:
                - start: "2026-10-22T02:12:03.967Z"
                  end: "2026-10-21T20:46:55-07:00"
            - harness: codex
              id: 01a16161-e237-7f33-bb66-192582ab999d
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e8b1541e3537a5f034fd0ec8963fa2ff873e31d2608ab45319e221eb5a788811
              started: "2026-10-22T02:12:47.67Z"
              spawned: false
              bytes: 447786
              windows:
                - start: "2026-10-22T02:12:47.67Z"
                  end: "2026-10-21T20:46:55-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 6e28d3e8df6b95fe77b608f09974c68589a9ba68d14282999a3f6cab214a0f18
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 3120074
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-21T20:46:55-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 98e418a4aaf1b786df3aa42b3921cb715526d69c370c65cd6923b98aaa890d45
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 2626073
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-21T20:46:55-07:00"
        cuts:
            - dispatched: "2026-10-21T20:07:45-07:00"
              finalized: "2026-10-21T20:13:18-07:00"
              finalizing: false
            - dispatched: "2026-10-21T20:35:23-07:00"
              finalized: "2026-10-21T20:39:29-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-21T20:50:42-07:00"
      dispatched: "2026-10-21T20:47:48-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-7
          disposition: addressed
          note: codec.go:85-101 declares required members across sketch evidence; filterManifest checks presence before typed decoding at lines 203-207. Tests exercise 120 required omissions, optional/legacy compatibility, and six Store read paths. Running the new regressions with the pre-fix codec through a temporary Go overlay fails, including all three original source-member probes; pinned-head tests pass.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 052330f68d4f81b9baff10d90e3559c6a5abaf55
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 39f83b2c-5183-46f4-99a0-dd10f0ad369c
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: fix-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/0bb12c97bd990fee
            reviewed_head: 37ef740f65a78dc037696aa62189f71c58c8a22e
            slot: pair:9
            requested_at: "2026-10-22T04:23:40.295333Z"
          phase: finished
          pid: 27949
          started_at: "2026-10-22T04:23:40.538714Z"
          finished_at: "2026-10-22T04:24:47.542095Z"
          session_id: 01a161d9-b952-7601-b9ce-a74a716b4d37
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 37ef740f65a78dc037696aa62189f71c58c8a22e
        tracker_head: 502eb94e8ea724751c477940e88096bacda7ee92
        branch_point: d2540fd6ae6eaa06e8a90217b285c22116247558
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-21T21:22:42-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-21T21:22:42-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 6987c5c274ffb67266534d2dbce65a25904a2369c02a112be91c54c7346a0dfd
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 26425463
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-21T21:22:42-07:00"
            - harness: codex
              id: 01a1615b-b3b2-71d3-878e-78e13474733a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 875cc0385845359821419e9c1c22b64d19d3a1a1c51b8bd89a7644ade60bfd46
              started: "2026-10-22T02:06:02.68Z"
              spawned: false
              bytes: 781349
              windows:
                - start: "2026-10-22T02:06:02.68Z"
                  end: "2026-10-21T21:22:42-07:00"
            - harness: codex
              id: 01a1615f-3510-79f2-9cae-31cf36613788
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c47b782ebe76c153ab719771b1df19f3a47df32607a4dfc8f027c6246cc3538d
              started: "2026-10-22T02:09:52.302Z"
              spawned: false
              bytes: 462083
              windows:
                - start: "2026-10-22T02:09:52.302Z"
                  end: "2026-10-21T21:22:42-07:00"
            - harness: codex
              id: 01a16161-37d2-7071-bfc4-bdb6afa7db78
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 08a97e6e51d0b85438c720071e9692010efbf73613a778bae018f6757d5213af
              started: "2026-10-22T02:12:03.967Z"
              spawned: false
              bytes: 465728
              windows:
                - start: "2026-10-22T02:12:03.967Z"
                  end: "2026-10-21T21:22:42-07:00"
            - harness: codex
              id: 01a16161-e237-7f33-bb66-192582ab999d
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e8b1541e3537a5f034fd0ec8963fa2ff873e31d2608ab45319e221eb5a788811
              started: "2026-10-22T02:12:47.67Z"
              spawned: false
              bytes: 447786
              windows:
                - start: "2026-10-22T02:12:47.67Z"
                  end: "2026-10-21T21:22:42-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 6e28d3e8df6b95fe77b608f09974c68589a9ba68d14282999a3f6cab214a0f18
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 3120074
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-21T21:22:42-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 98e418a4aaf1b786df3aa42b3921cb715526d69c370c65cd6923b98aaa890d45
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 2626073
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-21T21:22:42-07:00"
        queue_waits:
            - attempt_id: 73583e3bf33e33f4d10db045e1492e42
              slot: pair:9
              record_digest: 9a8afc5345b4c1c93c00ef74801ebbd9fb61cd8c83ea0075fa3b0142059b9dac
              start: "2026-10-22T03:47:16.877561Z"
              end: "2026-10-22T04:10:56.123514Z"
              queue_seconds: 1419.245953
              raw_seconds: 1419.245953
              exact: true
        cuts:
            - dispatched: "2026-10-21T20:07:45-07:00"
              finalized: "2026-10-21T20:13:18-07:00"
              finalizing: false
            - dispatched: "2026-10-21T20:35:23-07:00"
              finalized: "2026-10-21T20:39:29-07:00"
              finalizing: false
            - dispatched: "2026-10-21T20:47:48-07:00"
              finalized: "2026-10-21T20:50:42-07:00"
              finalizing: true
      "n": 4
      timestamp: "2026-10-21T21:24:49-07:00"
      dispatched: "2026-10-21T21:23:35-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      recipe: fix-review
      isolation: fenced
      reviewed: 37ef740f65a78dc037696aa62189f71c58c8a22e
      blocked: false
---

# Gate ledger — pair#753 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T20:13:18-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `canceled-operation-state-settlement` (kind: defect) Canceled preview publications can remain permanently Publishing
  cmd/internal/couchtty/console_submission_preview.go:154 cancels and joins a publication after a refresh notification, then continues without settling its state. If host and candidates are unchanged, NextPreview ignores both repeated inputs and PreviewRefresh cannot leave Publishing. Production SubmissionPreviewUpdates preloads precisely such a notification at chrome_submission_preview.go:125; receiver-progress notifications can also reproduce it. A held publisher plus a buffered unchanged-host notification deterministically failed the review probe: no replacement publication occurred and preparation remained refused. ARCH-ORDER: settle the canceled publication through the transition model before refreshing, and cover initial, duplicate, and catchup notifications.
- **BR-2** [Critical] `mandatory-evidence-budget-priority` (kind: defect) Unrelated DOM attributes consume the budget before mandatory anchor facts
  cmd/internal/webcast/query/element_facts.go:98 spends the shared 64 KiB attribute budget during whole-DOM traversal, before facts() prioritizes anchors. A probe with 70 preceding unrelated elements carrying bounded data attributes and a later anchor with a valid 1024-byte ID lost that ID, although the selected anchor facts easily fit the output budget. ARCH-PURPOSE: reserve or allocate declared-fact bytes in anchor-first, ancestor-next, nearest-context order; test starvation by unselected nodes.
- **BR-3** [Critical] `optional-anchor-compatibility` (kind: defect) Valid region-only notes now fail submission preparation
  cmd/internal/webcast/query/querier.go:225 always inserts n.At.Node into the anchor list, including zero. The resulting synthetic node-zero fact is rejected by cmd/internal/submission/sketch.go:138. recording/record.go:548 explicitly permits a note with a region and no node; a review probe successfully encoded those notes but Build failed with "invalid declared element fact". Preserve the existing note contract by omitting absent anchor handles while retaining region intersection facts, and add a region-only submission regression.
- **BR-4** [Critical] `strict-known-field-decoding` (kind: defect) Null pressure passes decoding and validation as fabricated zero pressure
  cmd/internal/submission/codec.go:103 checks tuple length but not numeric member types, then json.Unmarshal at line 89 converts null into a float's zero value. Replacing [20,20,0.2] with [20,20,null] in a valid sketch manifest passed both DecodePreserving and Validate in the review probe, yielding pressure 0. ARCH-SECURE/ARCH-EVOLVE: reject null and other invalid types for required numeric evidence, sweep the known scalar and tuple schemas, and add strict-type regressions.
- **BR-5** [Critical] `core-concept-traceability` (kind: defect) The Core concepts inventory contradicts implemented entities and boundaries
  workshop/plans/000753-cast-sketch-agent-export-plan.md:70-83 lists PreviewSelection in sketch.go, a PURE DeliveredSketchSet in sketch.go, query/render_sketch.go, and webcast/submission/sketch.go. Those definitions/files do not exist at those locations: PreviewSelection lives in preview.go, delivery derivation is the filesystem-backed Store.DeliveredSketchVersions in store.go, and export is implemented through sketch_export.go and sketch_image_export.go. ARCH-PURE: revise the inventory to name actual entities and paths, separate pure selection from integration delivery derivation, and record the correction under Revisions.
- **BR-6** [Important] `native-conformance-evidence` (kind: unverifiable) Native historical rendering and fidelity preservation remain unverified
  Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestLiveSketchImage|TestOperatorSketchLiveDrawing|TestLiveFidelity' -count=1 -v. Fidelity fixtures, broken-render detection, operator drawing, and historical pressure/batch rendering failed with Chrome/CDP EOF; operator Chrome reported "GPU process isn't usable", and historical rendering also reported "operation not permitted". Cancellation cleanup passed. ARCH-MOCK: provide passing results for these checks on the final source-equivalent head in a supported native environment, including pressure pixels, eight-image timing, fidelity preservation, and joined cleanup.

## Round 2 — 2026-10-21T20:39:29-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Cancellation now settles the exact publication through PreviewPublishFailed. The checked-in regression passes on head and fails with pre-fix production code for all five unchanged-input notification cases.
- BR-2 — addressed — Selected facts now consume the output budget in anchor-first order with mandatory-handle reservations. The starvation regression passes on head and fails with the previous implementation; unused measurement fields were retained solely to compile the counterfactual.
- BR-3 — addressed — Absent anchor handles are omitted by both note projection and the shared extractor. The region-only Build/ContextAt regression passes on head and fails with the pre-fix implementations.
- BR-4 — addressed — Numeric tuple members now reject null before Go decoding. The null-pressure regression passes on head and fails with the previous codec, which produces pressure zero and passes Validate. Missing-field handling is a separate remaining instance reported below.
- BR-5 — addressed — Plan lines 62–91 now locate PreviewSelection in preview.go, distinguish pure SelectSketchPreview from INTEGRATION Store.DeliveredSketchVersions, and name the actual export files. These match the definitions in preview.go, preview_selection.go, store.go, sketch_export.go and sketch_image_export.go; Revisions records the correction.
- BR-6 — withdrawn — Inspected native-85406e681.log and verified SHA256 d7942734429e891879c962c39102fc7781eaf7f3c4d9daa39e82440efae7799e. The requested native checks passed in 77.227s, including pressure/batch rendering, fidelity fixtures, broken-render detection and cancellation cleanup. The eight-image measurement was 1.796754583s. git diff confirms the reviewed head changes only issue prose after that source commit.

### Raised

- **BR-7** [Critical] `strict-known-field-decoding` (kind: defect) Missing required sketch source fields become fabricated zero-valued evidence
  cmd/internal/submission/codec.go:189 validates only fields present in the JSON object; it never checks required-member presence. Three independent probes deleting sketches[0].source.t, record_index, or scroll from the valid checked-in fixture all passed DecodePreserving and Validate: source time 10 became 0, record index 2 became 0, and absent scroll became [0,0]. This violates the strict required-field and original-source evidence contracts (ARCH-SECURE, ARCH-EVOLVE). This is the 2nd finding in family strict-known-field-decoding: do not patch only these three fields. Define required versus optional members across the new evidence schema, enforce presence before decoding, and sweep omission regressions across that inventory while preserving intentional legacy defaults; measured prevalence here is 3/3 probed source members.

## Round 3 — 2026-10-21T20:50:42-07:00 (codex) — passed

### Disposed

- BR-7 — addressed — codec.go:85-101 declares required members across sketch evidence; filterManifest checks presence before typed decoding at lines 203-207. Tests exercise 120 required omissions, optional/legacy compatibility, and six Store read paths. Running the new regressions with the pre-fix codec through a temporary Go overlay fails, including all three original source-member probes; pinned-head tests pass.

## Round 4 — 2026-10-21T21:24:49-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
