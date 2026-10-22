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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1c7433db-59e7-47b0-b653-b39692a8d3b4
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 5
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/cbf61589e9f832b6
            reviewed_head: d6c24469c2599d1a6d4ac45257afd8be8d9fa09f
            slot: pair:9
            requested_at: "2026-10-22T17:16:31.57146Z"
          phase: finished
          pid: 20988
          started_at: "2026-10-22T17:16:31.801659Z"
          finished_at: "2026-10-22T17:20:38.566348Z"
          session_id: 01a1649d-4ab7-7d42-945d-aad964ee1af9
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: d6c24469c2599d1a6d4ac45257afd8be8d9fa09f
        tracker_head: 977a3247b842938ff57a72030af7c1c1333cc33e
        branch_point: 39be359a3c55bcce5fd485548cd01e9d318d8b5e
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-22T10:15:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-22T10:15:45-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 872f0fb227e9085660e889b1e38c45033b12e022108a89de362e679f0d4b2f04
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 32304754
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-22T10:15:45-07:00"
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
                  end: "2026-10-22T10:15:45-07:00"
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
                  end: "2026-10-22T10:15:45-07:00"
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
                  end: "2026-10-22T10:15:45-07:00"
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
                  end: "2026-10-22T10:15:45-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: ec3bc02690a7897f5ceb85f08bc74670fd1ea57fdc78b6540f83096f4df9e0d0
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 4689656
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-22T10:15:45-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 077c4abbb372e2270874afa218d0cfc777ea6aaf0894735af8e40f0e09baae2e
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 5184954
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-22T10:15:45-07:00"
            - harness: codex
              id: 01a16479-593c-72c1-8c56-0ab2fb2d8d38
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 4045c6b50e88c6f7903347fc83b422f960be8e06cfcef1254a93dac4447e12aa
              started: "2026-10-22T16:37:17.742Z"
              spawned: false
              bytes: 608560
              windows:
                - start: "2026-10-22T16:37:17.742Z"
                  end: "2026-10-22T10:15:45-07:00"
            - harness: codex
              id: 01a16479-d353-7592-b10b-f52ec506bc33
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 1594826f2a5e4f55f9edee4de6fffad1e33456a6ee0864d2ca6796094763d33f
              started: "2026-10-22T16:37:48.561Z"
              spawned: false
              bytes: 550504
              windows:
                - start: "2026-10-22T16:37:48.561Z"
                  end: "2026-10-22T10:15:45-07:00"
            - harness: codex
              id: 01a1647b-645c-7e42-9827-fa4c62413563
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: a059bd659f8629ff77cd0440ba095368c02c7ec2d449174fec9fab9b3f856df9
              started: "2026-10-22T16:39:31.055Z"
              spawned: false
              bytes: 642807
              windows:
                - start: "2026-10-22T16:39:31.055Z"
                  end: "2026-10-22T10:15:45-07:00"
            - harness: codex
              id: 01a1647b-e38b-7623-aee0-a413c4337961
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e197fe43cac1eacc4727e9e560556d6831226adfb76ee4ec273dfddf071171fe
              started: "2026-10-22T16:40:03.557Z"
              spawned: false
              bytes: 526243
              windows:
                - start: "2026-10-22T16:40:03.557Z"
                  end: "2026-10-22T10:15:45-07:00"
        queue_waits:
            - attempt_id: 73583e3bf33e33f4d10db045e1492e42
              slot: pair:9
              record_digest: 9a8afc5345b4c1c93c00ef74801ebbd9fb61cd8c83ea0075fa3b0142059b9dac
              start: "2026-10-22T03:47:16.877561Z"
              end: "2026-10-22T04:10:56.123514Z"
              queue_seconds: 1419.245953
              raw_seconds: 1419.245953
              exact: true
            - attempt_id: 6e5ec36c332e27136f65a278d4410345
              slot: pair:9
              record_digest: f06313e24df9faed3bddfc98a1f8dccf413034a0ce96d7f725ddc047edb88cd2
              start: "2026-10-22T04:23:02.411368Z"
              end: "2026-10-22T04:55:40.697752Z"
              queue_seconds: 1958.286384
              raw_seconds: 1958.286384
              exact: true
            - attempt_id: fb1135928540e24dd58d6f6b8718bb52
              slot: pair:9
              record_digest: c8c012cd6262697df265a02d703e95378357ff8b89bc2f283853d651d27540ac
              start: "2026-10-22T10:00:53.728898Z"
              end: "2026-10-22T10:12:55.551601Z"
              queue_seconds: 721.822703
              raw_seconds: 721.822703
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
            - dispatched: "2026-10-21T21:23:35-07:00"
              finalized: "2026-10-21T21:24:49-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-22T10:20:40-07:00"
      dispatched: "2026-10-22T10:16:28-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Cancellation settles the exact publication through PreviewPublishFailed; controlled unchanged-refresh, terminal cancellation and expired-publication regressions passed.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Selected facts reserve mandatory anchor capacity before ancestors/context; unselected-declaration and mandatory-handle budget regressions passed.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Zero is excluded from declared anchor identities; region-only Build/ContextAt regression passed, including under race detection.
          round: 5
        - id: BR-4
          disposition: addressed
          note: Codec preflight rejects null numeric evidence before typed decoding; null-pressure and known-scalar regressions passed.
          round: 5
        - id: BR-5
          disposition: addressed
          note: The corrected Core concepts inventory separates pure selection from Store IO and identifies the actual preview, export and renderer locations.
          round: 5
        - id: BR-6
          disposition: withdrawn
          note: Retained native log SHA256 matches its evidence record; its source revision differs from pinned head only in evidence/tracker files. The log records passing shared-page, legacy rendering, cancellation and fidelity checks, with the stress-fixture qualification preserved.
          round: 5
        - id: BR-7
          disposition: addressed
          note: Explicit required-member schemas are reachable through Store decoding; deletion, valid-zero, optional-omission and Store tampering regressions passed.
          round: 5
      findings:
        - kind: defect
          id: BR-8
          severity: Critical
          title: Shared historical images silently move ink onto replacement elements
          detail: cmd/internal/webcast/web/sketch-page-image.js:52 uses Renderer.placementOf without requiring the original recorded node. When that node is removed, annotation-placement.js:123-134 searches for a matching replacement. A read-only probe through the production Renderer.placementOf and preparePageInk removed node 7, inserted a different node with matching text, and successfully projected the original points onto the replacement. This contradicts plan line 295's explicit replaced-anchor refusal (ARCH-SECURE, ARCH-PURPOSE). Require connected original recorded-node identity for shared historical rendering and add a regression through the actual placement adapter; preserve the existing live annotation reanchoring policy.
          family: historical-anchor-identity
          round: 5
        - kind: defect
          id: BR-9
          severity: Minor
          title: Agent-facing instructions retain obsolete per-sketch evidence references
          detail: nvim/dom_submission_state.lua:43-44 describes sketch-N.png historical source images, while couchcmd/skills/couch/SKILL.md:667-670 directs readers to image.plan.paths. New shared items instead use sketch-page.png and sketch.paths. The canonical appended ReadingGuide and opening skill paragraph are correct, limiting the impact, but these conflicting instructions should distinguish shared and legacy variants (ARCH-PURPOSE). Update the emitted-prompt regression alongside the text.
          family: handoff-contract-alignment
          round: 5
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9e6cc63f-f024-4e09-a7f6-1a87fdd9bcee
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 6
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/49de3740666da5f2
            reviewed_head: 892b1fc78543154c558fdcf561e5a3b6eff8b040
            slot: pair:9
            requested_at: "2026-10-22T17:31:05.989662Z"
          phase: finished
          pid: 87455
          started_at: "2026-10-22T17:31:06.139592Z"
          finished_at: "2026-10-22T17:33:26.633318Z"
          session_id: 01a164aa-a1d5-7940-b5ab-9c43b707df85
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 892b1fc78543154c558fdcf561e5a3b6eff8b040
        tracker_head: b5e6ebb3d79e16e9a4374b6ee9263a2d293160c9
        branch_point: 39be359a3c55bcce5fd485548cd01e9d318d8b5e
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-22T10:30:17-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-22T10:30:17-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: bfcf8eb9d94c6e3801e6b9eb6c3f95b0ef4acc1c1dd84f488a0ad6609cc0aaef
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 33765719
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-22T10:30:17-07:00"
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
                  end: "2026-10-22T10:30:17-07:00"
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
                  end: "2026-10-22T10:30:17-07:00"
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
                  end: "2026-10-22T10:30:17-07:00"
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
                  end: "2026-10-22T10:30:17-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: ec3bc02690a7897f5ceb85f08bc74670fd1ea57fdc78b6540f83096f4df9e0d0
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 4689656
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-22T10:30:17-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: d977fe553f97dd23c52bd43dd9b250bdaa6c30f401c80d33dda6c00677bcec11
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 5463017
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-22T10:30:17-07:00"
            - harness: codex
              id: 01a16479-593c-72c1-8c56-0ab2fb2d8d38
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 4045c6b50e88c6f7903347fc83b422f960be8e06cfcef1254a93dac4447e12aa
              started: "2026-10-22T16:37:17.742Z"
              spawned: false
              bytes: 608560
              windows:
                - start: "2026-10-22T16:37:17.742Z"
                  end: "2026-10-22T10:30:17-07:00"
            - harness: codex
              id: 01a16479-d353-7592-b10b-f52ec506bc33
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 1594826f2a5e4f55f9edee4de6fffad1e33456a6ee0864d2ca6796094763d33f
              started: "2026-10-22T16:37:48.561Z"
              spawned: false
              bytes: 550504
              windows:
                - start: "2026-10-22T16:37:48.561Z"
                  end: "2026-10-22T10:30:17-07:00"
            - harness: codex
              id: 01a1647b-645c-7e42-9827-fa4c62413563
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: a059bd659f8629ff77cd0440ba095368c02c7ec2d449174fec9fab9b3f856df9
              started: "2026-10-22T16:39:31.055Z"
              spawned: false
              bytes: 642807
              windows:
                - start: "2026-10-22T16:39:31.055Z"
                  end: "2026-10-22T10:30:17-07:00"
            - harness: codex
              id: 01a1647b-e38b-7623-aee0-a413c4337961
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e197fe43cac1eacc4727e9e560556d6831226adfb76ee4ec273dfddf071171fe
              started: "2026-10-22T16:40:03.557Z"
              spawned: false
              bytes: 526243
              windows:
                - start: "2026-10-22T16:40:03.557Z"
                  end: "2026-10-22T10:30:17-07:00"
        queue_waits:
            - attempt_id: 73583e3bf33e33f4d10db045e1492e42
              slot: pair:9
              record_digest: 9a8afc5345b4c1c93c00ef74801ebbd9fb61cd8c83ea0075fa3b0142059b9dac
              start: "2026-10-22T03:47:16.877561Z"
              end: "2026-10-22T04:10:56.123514Z"
              queue_seconds: 1419.245953
              raw_seconds: 1419.245953
              exact: true
            - attempt_id: 6e5ec36c332e27136f65a278d4410345
              slot: pair:9
              record_digest: f06313e24df9faed3bddfc98a1f8dccf413034a0ce96d7f725ddc047edb88cd2
              start: "2026-10-22T04:23:02.411368Z"
              end: "2026-10-22T04:55:40.697752Z"
              queue_seconds: 1958.286384
              raw_seconds: 1958.286384
              exact: true
            - attempt_id: fb1135928540e24dd58d6f6b8718bb52
              slot: pair:9
              record_digest: c8c012cd6262697df265a02d703e95378357ff8b89bc2f283853d651d27540ac
              start: "2026-10-22T10:00:53.728898Z"
              end: "2026-10-22T10:12:55.551601Z"
              queue_seconds: 721.822703
              raw_seconds: 721.822703
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
            - dispatched: "2026-10-21T21:23:35-07:00"
              finalized: "2026-10-21T21:24:49-07:00"
              finalizing: true
            - dispatched: "2026-10-22T10:16:28-07:00"
              finalized: "2026-10-22T10:20:40-07:00"
              finalizing: false
      "n": 6
      timestamp: "2026-10-22T10:33:27-07:00"
      dispatched: "2026-10-22T10:31:02-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-8
          disposition: addressed
          note: sketch-page-image.js:46-57 routes production rendering through originalOnly placement. The actual Renderer adapter regression passes and fails on disconnected replacement when that option is removed in a disposable copy; live replacement placement remains covered.
          round: 6
        - id: BR-9
          disposition: addressed
          note: dom_submission_state.lua:43-47 and couch SKILL.md:667-678 distinguish sketch-page.png/shared paths from legacy sketch-N.png/image.plan.paths. The emitted-prompt regression passes and fails with the prior prompt restored.
          round: 6
      recipe: milestone-review
      isolation: fenced
      reviewed: 892b1fc78543154c558fdcf561e5a3b6eff8b040
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f6f33495-d562-4ed6-97cd-cb1ca7ab6811
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 7
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/b7209862b3272fbd
            reviewed_head: 4fc20f722c297c8e1e0c1731f0ca6d443ffdf568
            slot: pair:9
            requested_at: "2026-10-22T18:35:33.2005Z"
          phase: finished
          pid: 63408
          started_at: "2026-10-22T18:35:33.406808Z"
          finished_at: "2026-10-22T18:38:38.055582Z"
          session_id: 01a164e5-a5b3-7843-9582-7a0c4ea59218
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 4fc20f722c297c8e1e0c1731f0ca6d443ffdf568
        tracker_head: 6aa97480153131c7afa95f032b4b839bcd729829
        branch_point: fea66e36b0d25b3e37f2dc70a084226a8ed1697a
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-22T11:34:12-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-22T11:34:12-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 915becc6dce851bc80393ebcf8e948e52c46accf3f59ca9710fd5fdd62acb5d8
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 36586537
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-22T11:34:12-07:00"
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
                  end: "2026-10-22T11:34:12-07:00"
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
                  end: "2026-10-22T11:34:12-07:00"
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
                  end: "2026-10-22T11:34:12-07:00"
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
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: d9173c3b0c381aba56460f2c23e03417c46ab007cb76016f83ebba1fdc2baa09
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 5197180
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: c295be1c5d705b02751375c7e53aa173b1d0a84e95775cd981dc7298caa90d41
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 6435766
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a16479-593c-72c1-8c56-0ab2fb2d8d38
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 4045c6b50e88c6f7903347fc83b422f960be8e06cfcef1254a93dac4447e12aa
              started: "2026-10-22T16:37:17.742Z"
              spawned: false
              bytes: 608560
              windows:
                - start: "2026-10-22T16:37:17.742Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a16479-d353-7592-b10b-f52ec506bc33
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 1594826f2a5e4f55f9edee4de6fffad1e33456a6ee0864d2ca6796094763d33f
              started: "2026-10-22T16:37:48.561Z"
              spawned: false
              bytes: 550504
              windows:
                - start: "2026-10-22T16:37:48.561Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a1647b-645c-7e42-9827-fa4c62413563
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: a059bd659f8629ff77cd0440ba095368c02c7ec2d449174fec9fab9b3f856df9
              started: "2026-10-22T16:39:31.055Z"
              spawned: false
              bytes: 642807
              windows:
                - start: "2026-10-22T16:39:31.055Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a1647b-e38b-7623-aee0-a413c4337961
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e197fe43cac1eacc4727e9e560556d6831226adfb76ee4ec273dfddf071171fe
              started: "2026-10-22T16:40:03.557Z"
              spawned: false
              bytes: 526243
              windows:
                - start: "2026-10-22T16:40:03.557Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a164d0-bf92-7a21-807c-ac181f0f9609
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 63ab7e4df67810dd9ad500299deb0f1a042321a05e33bdd1ffd28b4605640203
              started: "2026-10-22T18:12:45.163Z"
              spawned: false
              bytes: 676830
              windows:
                - start: "2026-10-22T18:12:45.163Z"
                  end: "2026-10-22T11:34:12-07:00"
            - harness: codex
              id: 01a164d1-75c0-76b2-9860-a7072090d766
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 0f5049bd370990e9974ad2314fed0f01eef0dcc2d0ce0bdcfbe3ab9636697711
              started: "2026-10-22T18:13:31.681Z"
              spawned: false
              bytes: 525537
              windows:
                - start: "2026-10-22T18:13:31.681Z"
                  end: "2026-10-22T11:34:12-07:00"
        queue_waits:
            - attempt_id: 73583e3bf33e33f4d10db045e1492e42
              slot: pair:9
              record_digest: 9a8afc5345b4c1c93c00ef74801ebbd9fb61cd8c83ea0075fa3b0142059b9dac
              start: "2026-10-22T03:47:16.877561Z"
              end: "2026-10-22T04:10:56.123514Z"
              queue_seconds: 1419.245953
              raw_seconds: 1419.245953
              exact: true
            - attempt_id: 6e5ec36c332e27136f65a278d4410345
              slot: pair:9
              record_digest: f06313e24df9faed3bddfc98a1f8dccf413034a0ce96d7f725ddc047edb88cd2
              start: "2026-10-22T04:23:02.411368Z"
              end: "2026-10-22T04:55:40.697752Z"
              queue_seconds: 1958.286384
              raw_seconds: 1958.286384
              exact: true
            - attempt_id: fb1135928540e24dd58d6f6b8718bb52
              slot: pair:9
              record_digest: c8c012cd6262697df265a02d703e95378357ff8b89bc2f283853d651d27540ac
              start: "2026-10-22T10:00:53.728898Z"
              end: "2026-10-22T10:12:55.551601Z"
              queue_seconds: 721.822703
              raw_seconds: 721.822703
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
            - dispatched: "2026-10-21T21:23:35-07:00"
              finalized: "2026-10-21T21:24:49-07:00"
              finalizing: true
            - dispatched: "2026-10-22T10:16:28-07:00"
              finalized: "2026-10-22T10:20:40-07:00"
              finalizing: false
            - dispatched: "2026-10-22T10:31:02-07:00"
              finalized: "2026-10-22T10:33:27-07:00"
              finalizing: true
      "n": 7
      timestamp: "2026-10-22T11:38:40-07:00"
      dispatched: "2026-10-22T11:35:29-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Canceled publications settle through PreviewPublishFailed before refresh; controlled cancellation, late-success and terminal-join regressions pass in couchtty.
          round: 7
        - id: BR-2
          disposition: addressed
          note: Selected-output budgeting reserves mandatory anchor facts; unselected-declaration and all-mandatory-handle regressions pass in query.
          round: 7
        - id: BR-3
          disposition: addressed
          note: Nonpositive absent anchors are excluded; region-only Build and capture-context regressions pass in webcast/submission.
          round: 7
        - id: BR-4
          disposition: addressed
          note: Schema preflight rejects null numeric tuple members; null-pressure, known-scalar and Store read-path regressions pass.
          round: 7
        - id: BR-5
          disposition: addressed
          note: The revised Core concepts inventory names preview_selection.go, sketch_export.go and the filesystem-backed DeliveredSketchVersions integration; those entities and boundaries exist.
          round: 7
        - id: BR-6
          disposition: withdrawn
          note: Retain the prior withdrawal; this review does not reopen the disposed native-environment finding or claim a fresh native run.
          round: 7
        - id: BR-7
          disposition: addressed
          note: Explicit required-member metadata is consumed by filterManifest; required-member deletion and Store missing-source regressions pass.
          round: 7
        - id: BR-8
          disposition: addressed
          note: prepareHistoricalPageInk requests originalOnly placement; actual Renderer adapter tests reject disconnected/replaced originals and pass connected-original controls.
          round: 7
        - id: BR-9
          disposition: addressed
          note: Lua handoff now describes sketch-page.png, sketch.index and paths alongside legacy image.plan.paths; the real Neovim transaction script passes.
          round: 7
      findings:
        - kind: defect
          id: BR-10
          severity: Critical
          title: Transient viewer disconnects repaint already-submitted feedback
          detail: cmd/internal/webcast/web/viewer.js:138 calls feedbackMask.beginStream on every EventSource error, including automatic reconnect with readyState 0. beginStream clears the confirmed mask and invokes redraw callbacks, while this path retains the old model and page. A direct production-connect probe changed both exact noteVisible and sketchVisible from false to true and increased redraws from 1 to 2 immediately after the error. Submitted ink/pins therefore reappear throughout a connection outage, contradicting the plan's same-stream reconnect-retains-mask contract (ARCH-ORDER, ARCH-PURPOSE). Preserve the confirmed mask across transient reconnects, retaining revision fencing; replace it atomically when the reconnect's authoritative scope/full snapshot arrives. Add a production-connect regression covering the outage interval, repeated errors, stale revisions and recovery.
          family: confirmed-display-continuity
          round: 7
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3f745b41-b1d1-4f5e-b972-1be1b8ec6c5d
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 8
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/71b3d85778b94835
            reviewed_head: 9bf8a80b2835e3dbdd2a2324556892db86d12279
            slot: pair:9
            requested_at: "2026-10-22T19:00:19.521129Z"
          phase: finished
          pid: 32580
          started_at: "2026-10-22T19:00:19.916285Z"
          finished_at: "2026-10-22T19:04:11.463791Z"
          session_id: 01a164fc-5400-7f11-9897-00e68325cffa
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 9bf8a80b2835e3dbdd2a2324556892db86d12279
        tracker_head: 7138cf2b250529a894437d0bc81949e46d7e118a
        branch_point: fea66e36b0d25b3e37f2dc70a084226a8ed1697a
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-22T11:59:15-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-22T11:59:15-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 50e6e4a27ae1e05cb9d4f2f89e216e0f108630601300b5ccae40906e0036e129
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 38243490
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-22T11:59:15-07:00"
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
                  end: "2026-10-22T11:59:15-07:00"
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
                  end: "2026-10-22T11:59:15-07:00"
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
                  end: "2026-10-22T11:59:15-07:00"
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
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: d8f00146b824a17319bc108677d96042d15315be7b5d629ec6e7a6473bc85629
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 5397090
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 53256bdb95bdda0864ee6ef5827f9386cca11d425755478cbc086400dc2328ea
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 6756832
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a16479-593c-72c1-8c56-0ab2fb2d8d38
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 4045c6b50e88c6f7903347fc83b422f960be8e06cfcef1254a93dac4447e12aa
              started: "2026-10-22T16:37:17.742Z"
              spawned: false
              bytes: 608560
              windows:
                - start: "2026-10-22T16:37:17.742Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a16479-d353-7592-b10b-f52ec506bc33
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 1594826f2a5e4f55f9edee4de6fffad1e33456a6ee0864d2ca6796094763d33f
              started: "2026-10-22T16:37:48.561Z"
              spawned: false
              bytes: 550504
              windows:
                - start: "2026-10-22T16:37:48.561Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a1647b-645c-7e42-9827-fa4c62413563
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: a059bd659f8629ff77cd0440ba095368c02c7ec2d449174fec9fab9b3f856df9
              started: "2026-10-22T16:39:31.055Z"
              spawned: false
              bytes: 642807
              windows:
                - start: "2026-10-22T16:39:31.055Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a1647b-e38b-7623-aee0-a413c4337961
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e197fe43cac1eacc4727e9e560556d6831226adfb76ee4ec273dfddf071171fe
              started: "2026-10-22T16:40:03.557Z"
              spawned: false
              bytes: 526243
              windows:
                - start: "2026-10-22T16:40:03.557Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a164d0-bf92-7a21-807c-ac181f0f9609
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 63ab7e4df67810dd9ad500299deb0f1a042321a05e33bdd1ffd28b4605640203
              started: "2026-10-22T18:12:45.163Z"
              spawned: false
              bytes: 676830
              windows:
                - start: "2026-10-22T18:12:45.163Z"
                  end: "2026-10-22T11:59:15-07:00"
            - harness: codex
              id: 01a164d1-75c0-76b2-9860-a7072090d766
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 0f5049bd370990e9974ad2314fed0f01eef0dcc2d0ce0bdcfbe3ab9636697711
              started: "2026-10-22T18:13:31.681Z"
              spawned: false
              bytes: 525537
              windows:
                - start: "2026-10-22T18:13:31.681Z"
                  end: "2026-10-22T11:59:15-07:00"
        queue_waits:
            - attempt_id: 73583e3bf33e33f4d10db045e1492e42
              slot: pair:9
              record_digest: 9a8afc5345b4c1c93c00ef74801ebbd9fb61cd8c83ea0075fa3b0142059b9dac
              start: "2026-10-22T03:47:16.877561Z"
              end: "2026-10-22T04:10:56.123514Z"
              queue_seconds: 1419.245953
              raw_seconds: 1419.245953
              exact: true
            - attempt_id: 6e5ec36c332e27136f65a278d4410345
              slot: pair:9
              record_digest: f06313e24df9faed3bddfc98a1f8dccf413034a0ce96d7f725ddc047edb88cd2
              start: "2026-10-22T04:23:02.411368Z"
              end: "2026-10-22T04:55:40.697752Z"
              queue_seconds: 1958.286384
              raw_seconds: 1958.286384
              exact: true
            - attempt_id: fb1135928540e24dd58d6f6b8718bb52
              slot: pair:9
              record_digest: c8c012cd6262697df265a02d703e95378357ff8b89bc2f283853d651d27540ac
              start: "2026-10-22T10:00:53.728898Z"
              end: "2026-10-22T10:12:55.551601Z"
              queue_seconds: 721.822703
              raw_seconds: 721.822703
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
            - dispatched: "2026-10-21T21:23:35-07:00"
              finalized: "2026-10-21T21:24:49-07:00"
              finalizing: true
            - dispatched: "2026-10-22T10:16:28-07:00"
              finalized: "2026-10-22T10:20:40-07:00"
              finalizing: false
            - dispatched: "2026-10-22T10:31:02-07:00"
              finalized: "2026-10-22T10:33:27-07:00"
              finalizing: true
            - dispatched: "2026-10-22T11:35:29-07:00"
              finalized: "2026-10-22T11:38:40-07:00"
              finalizing: false
      "n": 8
      timestamp: "2026-10-22T12:04:13-07:00"
      dispatched: "2026-10-22T12:00:14-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Cancellation settles the exact publication; canceled-refresh and expired-publication regressions cover the production manager.
          round: 8
        - id: BR-2
          disposition: addressed
          note: Mandatory-anchor reservation precedes optional context; anchor-budget and mandatory-handle tests pass.
          round: 8
        - id: BR-3
          disposition: addressed
          note: Region-only notes retain region facts without inventing node-zero anchors; builder regression passes.
          round: 8
        - id: BR-4
          disposition: addressed
          note: Strict schema preflight rejects null numeric evidence; codec and Store regressions pass.
          round: 8
        - id: BR-5
          disposition: addressed
          note: Current concept inventories distinguish pure selection/validation from Store, Board, renderer, and publication integration boundaries.
          round: 8
        - id: BR-6
          disposition: withdrawn
          note: Withdrawal retained. Native log SHA-256 matches its evidence record; the evidence revision and reviewed head differ only in evidence files and the issue log.
          round: 8
        - id: BR-7
          disposition: addressed
          note: Required sketch members are explicit in both schema variants; exhaustive omission and Store-read regressions pass.
          round: 8
        - id: BR-8
          disposition: addressed
          note: Historical page rendering requests originalOnly placement; replacement-anchor refusal has browser and native regressions.
          round: 8
        - id: BR-9
          disposition: addressed
          note: Shipped guide lines 665–705 and ReadingGuide describe root sketch_page/index/paths evidence and qualify per-sketch images as legacy, matching the implemented variants.
          round: 8
        - id: BR-10
          disposition: addressed
          note: Production-connect regression passes repeated errors, stale revisions, and recovery. Restoring beginStream's old clear() call in a disposable pinned checkout makes it fail at viewer.test.mjs:258; checkout removed afterward.
          round: 8
      findings:
        - kind: defect
          id: BR-11
          severity: Important
          title: Shipped-skill test still requires wording removed by the shared-image correction
          detail: cmd/internal/couchcmd/skill_privacy_test.go:18 requires “Its image has `replay-rendered` provenance”; the current guide correctly explains shared sketch_page and legacy images instead. TestSkillBroadcastPrivacyContract/historical_sketch_provenance consequently fails at line 44. This is the 2nd finding in family handoff-contract-alignment (ARCH-PURPOSE). State the rule that every handoff contract assertion follows the current shared/legacy representations, then sweep ReadingGuide and shipped-skill assertions together; the inspected assertions contain this stale sentence. Align the test with that contract and rerun couchcmd plus required final checks.
          family: handoff-contract-alignment
          round: 8
      recipe: milestone-review
      isolation: fenced
      reviewed: 9bf8a80b2835e3dbdd2a2324556892db86d12279
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a57d7d80-4288-417a-bfd1-7dea490152c8
            repository: github.com/xianxu/pair
            issue_id: "000753"
            recipe: milestone-review
            round: 9
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/24e520732e58b6df
            reviewed_head: 81f8ac7a6182720518a5fda17f4a877dbc529a2a
            slot: pair:9
            requested_at: "2026-10-22T21:52:29.143253Z"
          phase: finished
          pid: 33700
          started_at: "2026-10-22T21:52:29.32588Z"
          finished_at: "2026-10-22T21:57:08.718563Z"
          session_id: 01a16599-f02f-7d42-a98a-56cc89f9e902
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 81f8ac7a6182720518a5fda17f4a877dbc529a2a
        tracker_head: 94005784862c3cec1877ca08b645d25d441d9269
        branch_point: dc8e119eb1b75a91da2c4a75592988ca3c10986a
        since: "2026-10-21T18:40:37-07:00"
        until: "2026-10-22T14:51:08-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T18:40:37-07:00"
              end: "2026-10-22T14:51:08-07:00"
              began: "2026-10-21T18:40:37-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15e49-d730-7d33-b4b3-809814e36de7
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 00fbc3603e47725958a674dddbcc23dbe43c426014f33da8d6155569a35da0f3
              started: "2026-10-21T11:47:40.397Z"
              spawned: false
              bytes: 43833713
              windows:
                - start: "2026-10-21T18:40:37-07:00"
                  end: "2026-10-22T14:51:08-07:00"
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
                  end: "2026-10-22T14:51:08-07:00"
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
                  end: "2026-10-22T14:51:08-07:00"
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
                  end: "2026-10-22T14:51:08-07:00"
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
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a16163-584d-7811-84e6-1ee39612fbb2
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: 17d3e2b4d7f2e9abe208ac3e050db8a9c202f526d919ceec056ff2745b0634e8
              started: "2026-10-22T02:14:23.145Z"
              spawned: true
              bytes: 7709691
              windows:
                - start: "2026-10-22T02:14:23.145Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a16171-a029-7232-b04f-7fdea9a0e516
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15e49-d730-7d33-b4b3-809814e36de7
              digest: c3636e8f2c62b0d4ad42ec6f6dd5586a204f10fbc92cfa5b427f1160b10cdd8d
              started: "2026-10-22T02:29:59.009Z"
              spawned: true
              bytes: 9003891
              windows:
                - start: "2026-10-22T02:29:59.009Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a16479-593c-72c1-8c56-0ab2fb2d8d38
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 4045c6b50e88c6f7903347fc83b422f960be8e06cfcef1254a93dac4447e12aa
              started: "2026-10-22T16:37:17.742Z"
              spawned: false
              bytes: 608560
              windows:
                - start: "2026-10-22T16:37:17.742Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a16479-d353-7592-b10b-f52ec506bc33
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 1594826f2a5e4f55f9edee4de6fffad1e33456a6ee0864d2ca6796094763d33f
              started: "2026-10-22T16:37:48.561Z"
              spawned: false
              bytes: 550504
              windows:
                - start: "2026-10-22T16:37:48.561Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a1647b-645c-7e42-9827-fa4c62413563
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: a059bd659f8629ff77cd0440ba095368c02c7ec2d449174fec9fab9b3f856df9
              started: "2026-10-22T16:39:31.055Z"
              spawned: false
              bytes: 642807
              windows:
                - start: "2026-10-22T16:39:31.055Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a1647b-e38b-7623-aee0-a413c4337961
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e197fe43cac1eacc4727e9e560556d6831226adfb76ee4ec273dfddf071171fe
              started: "2026-10-22T16:40:03.557Z"
              spawned: false
              bytes: 526243
              windows:
                - start: "2026-10-22T16:40:03.557Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a164d0-bf92-7a21-807c-ac181f0f9609
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 63ab7e4df67810dd9ad500299deb0f1a042321a05e33bdd1ffd28b4605640203
              started: "2026-10-22T18:12:45.163Z"
              spawned: false
              bytes: 676830
              windows:
                - start: "2026-10-22T18:12:45.163Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a164d1-75c0-76b2-9860-a7072090d766
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 0f5049bd370990e9974ad2314fed0f01eef0dcc2d0ce0bdcfbe3ab9636697711
              started: "2026-10-22T18:13:31.681Z"
              spawned: false
              bytes: 525537
              windows:
                - start: "2026-10-22T18:13:31.681Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a1657a-6420-7633-808f-be531aa7cac6
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: b4133353af86379dc1ed5e1d17085a4f4c57525784c7c982e61da4f4fad1cf95
              started: "2026-10-22T21:18:02.957Z"
              spawned: false
              bytes: 639594
              windows:
                - start: "2026-10-22T21:18:02.957Z"
                  end: "2026-10-22T14:51:08-07:00"
            - harness: codex
              id: 01a1657a-e09e-7d83-84b2-14be06e54f9b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 2323dd2ffb1e0397e68903b34a8e322dfecb5583f4cb7c1ee163f2bc94d1cd6e
              started: "2026-10-22T21:18:34.586Z"
              spawned: false
              bytes: 518638
              windows:
                - start: "2026-10-22T21:18:34.586Z"
                  end: "2026-10-22T14:51:08-07:00"
        queue_waits:
            - attempt_id: 73583e3bf33e33f4d10db045e1492e42
              slot: pair:9
              record_digest: 9a8afc5345b4c1c93c00ef74801ebbd9fb61cd8c83ea0075fa3b0142059b9dac
              start: "2026-10-22T03:47:16.877561Z"
              end: "2026-10-22T04:10:56.123514Z"
              queue_seconds: 1419.245953
              raw_seconds: 1419.245953
              exact: true
            - attempt_id: 6e5ec36c332e27136f65a278d4410345
              slot: pair:9
              record_digest: f06313e24df9faed3bddfc98a1f8dccf413034a0ce96d7f725ddc047edb88cd2
              start: "2026-10-22T04:23:02.411368Z"
              end: "2026-10-22T04:55:40.697752Z"
              queue_seconds: 1958.286384
              raw_seconds: 1958.286384
              exact: true
            - attempt_id: fb1135928540e24dd58d6f6b8718bb52
              slot: pair:9
              record_digest: c8c012cd6262697df265a02d703e95378357ff8b89bc2f283853d651d27540ac
              start: "2026-10-22T10:00:53.728898Z"
              end: "2026-10-22T10:12:55.551601Z"
              queue_seconds: 721.822703
              raw_seconds: 721.822703
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 49.55413333333333
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
            - dispatched: "2026-10-21T21:23:35-07:00"
              finalized: "2026-10-21T21:24:49-07:00"
              finalizing: true
            - dispatched: "2026-10-22T10:16:28-07:00"
              finalized: "2026-10-22T10:20:40-07:00"
              finalizing: false
            - dispatched: "2026-10-22T10:31:02-07:00"
              finalized: "2026-10-22T10:33:27-07:00"
              finalizing: true
            - dispatched: "2026-10-22T11:35:29-07:00"
              finalized: "2026-10-22T11:38:40-07:00"
              finalizing: false
            - dispatched: "2026-10-22T12:00:14-07:00"
              finalized: "2026-10-22T12:04:13-07:00"
              finalizing: true
      "n": 9
      timestamp: "2026-10-22T14:57:28-07:00"
      dispatched: "2026-10-22T14:52:25-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Retained: canceled publications settle through PreviewPublishFailed; controlled cancellation and late-ACK regressions pass.'
          round: 9
        - id: BR-2
          disposition: addressed
          note: 'Retained: element-fact allocation reserves mandatory anchor evidence before optional declarations; budget-priority regressions pass.'
          round: 9
        - id: BR-3
          disposition: addressed
          note: 'Retained: absent anchors no longer create node-zero facts; region-only Build and capture-context regression passes.'
          round: 9
        - id: BR-4
          disposition: addressed
          note: 'Retained: strict schema preflight rejects null pressure and invalid known-field types; codec and Store regressions pass.'
          round: 9
        - id: BR-5
          disposition: addressed
          note: 'Retained: corrected Core concepts tables and symbol reconciliation distinguish pure selectors from filesystem, recorder and browser integration owners.'
          round: 9
        - id: BR-6
          disposition: withdrawn
          note: 'Retained withdrawal: native logs match their recorded SHA-256 hashes and report successful fidelity, historical rendering and integrated current-view qualification; real-operator acceptance remains explicitly separate.'
          round: 9
        - id: BR-7
          disposition: addressed
          note: 'Retained: explicit required-member schemas reject omitted source evidence; required-field deletion and Store-read regressions pass.'
          round: 9
        - id: BR-8
          disposition: addressed
          note: 'Retained: historical rendering requires original identity through originalOnly placement; replacement-anchor refusal regressions pass.'
          round: 9
        - id: BR-9
          disposition: addressed
          note: 'Retained: ReadingGuide and shipped skill describe root sketch_page evidence, original paths, current projections and legacy per-sketch representations consistently with the codec.'
          round: 9
        - id: BR-10
          disposition: addressed
          note: 'Retained: beginStream preserves confirmed display facts while rotating stream ownership; reconnect and outage browser regressions pass.'
          round: 9
        - id: BR-11
          disposition: addressed
          note: skill_privacy_test.go:18 now asserts current-view sketch_page provenance alongside shared indexes, original paths and legacy images. Read-only evaluation finds the old historical-provenance assertion fails while all 33 current assertions match; couchcmd passes.
          round: 9
      recipe: milestone-review
      isolation: fenced
      reviewed: 81f8ac7a6182720518a5fda17f4a877dbc529a2a
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

## Round 5 — 2026-10-22T10:20:40-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Cancellation settles the exact publication through PreviewPublishFailed; controlled unchanged-refresh, terminal cancellation and expired-publication regressions passed.
- BR-2 — addressed — Selected facts reserve mandatory anchor capacity before ancestors/context; unselected-declaration and mandatory-handle budget regressions passed.
- BR-3 — addressed — Zero is excluded from declared anchor identities; region-only Build/ContextAt regression passed, including under race detection.
- BR-4 — addressed — Codec preflight rejects null numeric evidence before typed decoding; null-pressure and known-scalar regressions passed.
- BR-5 — addressed — The corrected Core concepts inventory separates pure selection from Store IO and identifies the actual preview, export and renderer locations.
- BR-6 — withdrawn — Retained native log SHA256 matches its evidence record; its source revision differs from pinned head only in evidence/tracker files. The log records passing shared-page, legacy rendering, cancellation and fidelity checks, with the stress-fixture qualification preserved.
- BR-7 — addressed — Explicit required-member schemas are reachable through Store decoding; deletion, valid-zero, optional-omission and Store tampering regressions passed.

### Raised

- **BR-8** [Critical] `historical-anchor-identity` (kind: defect) Shared historical images silently move ink onto replacement elements
  cmd/internal/webcast/web/sketch-page-image.js:52 uses Renderer.placementOf without requiring the original recorded node. When that node is removed, annotation-placement.js:123-134 searches for a matching replacement. A read-only probe through the production Renderer.placementOf and preparePageInk removed node 7, inserted a different node with matching text, and successfully projected the original points onto the replacement. This contradicts plan line 295's explicit replaced-anchor refusal (ARCH-SECURE, ARCH-PURPOSE). Require connected original recorded-node identity for shared historical rendering and add a regression through the actual placement adapter; preserve the existing live annotation reanchoring policy.
- **BR-9** [Minor] `handoff-contract-alignment` (kind: defect) Agent-facing instructions retain obsolete per-sketch evidence references
  nvim/dom_submission_state.lua:43-44 describes sketch-N.png historical source images, while couchcmd/skills/couch/SKILL.md:667-670 directs readers to image.plan.paths. New shared items instead use sketch-page.png and sketch.paths. The canonical appended ReadingGuide and opening skill paragraph are correct, limiting the impact, but these conflicting instructions should distinguish shared and legacy variants (ARCH-PURPOSE). Update the emitted-prompt regression alongside the text.

## Round 6 — 2026-10-22T10:33:27-07:00 (codex) — passed

### Disposed

- BR-8 — addressed — sketch-page-image.js:46-57 routes production rendering through originalOnly placement. The actual Renderer adapter regression passes and fails on disconnected replacement when that option is removed in a disposable copy; live replacement placement remains covered.
- BR-9 — addressed — dom_submission_state.lua:43-47 and couch SKILL.md:667-678 distinguish sketch-page.png/shared paths from legacy sketch-N.png/image.plan.paths. The emitted-prompt regression passes and fails with the prior prompt restored.

## Round 7 — 2026-10-22T11:38:40-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Canceled publications settle through PreviewPublishFailed before refresh; controlled cancellation, late-success and terminal-join regressions pass in couchtty.
- BR-2 — addressed — Selected-output budgeting reserves mandatory anchor facts; unselected-declaration and all-mandatory-handle regressions pass in query.
- BR-3 — addressed — Nonpositive absent anchors are excluded; region-only Build and capture-context regressions pass in webcast/submission.
- BR-4 — addressed — Schema preflight rejects null numeric tuple members; null-pressure, known-scalar and Store read-path regressions pass.
- BR-5 — addressed — The revised Core concepts inventory names preview_selection.go, sketch_export.go and the filesystem-backed DeliveredSketchVersions integration; those entities and boundaries exist.
- BR-6 — withdrawn — Retain the prior withdrawal; this review does not reopen the disposed native-environment finding or claim a fresh native run.
- BR-7 — addressed — Explicit required-member metadata is consumed by filterManifest; required-member deletion and Store missing-source regressions pass.
- BR-8 — addressed — prepareHistoricalPageInk requests originalOnly placement; actual Renderer adapter tests reject disconnected/replaced originals and pass connected-original controls.
- BR-9 — addressed — Lua handoff now describes sketch-page.png, sketch.index and paths alongside legacy image.plan.paths; the real Neovim transaction script passes.

### Raised

- **BR-10** [Critical] `confirmed-display-continuity` (kind: defect) Transient viewer disconnects repaint already-submitted feedback
  cmd/internal/webcast/web/viewer.js:138 calls feedbackMask.beginStream on every EventSource error, including automatic reconnect with readyState 0. beginStream clears the confirmed mask and invokes redraw callbacks, while this path retains the old model and page. A direct production-connect probe changed both exact noteVisible and sketchVisible from false to true and increased redraws from 1 to 2 immediately after the error. Submitted ink/pins therefore reappear throughout a connection outage, contradicting the plan's same-stream reconnect-retains-mask contract (ARCH-ORDER, ARCH-PURPOSE). Preserve the confirmed mask across transient reconnects, retaining revision fencing; replace it atomically when the reconnect's authoritative scope/full snapshot arrives. Add a production-connect regression covering the outage interval, repeated errors, stale revisions and recovery.

## Round 8 — 2026-10-22T12:04:13-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Cancellation settles the exact publication; canceled-refresh and expired-publication regressions cover the production manager.
- BR-2 — addressed — Mandatory-anchor reservation precedes optional context; anchor-budget and mandatory-handle tests pass.
- BR-3 — addressed — Region-only notes retain region facts without inventing node-zero anchors; builder regression passes.
- BR-4 — addressed — Strict schema preflight rejects null numeric evidence; codec and Store regressions pass.
- BR-5 — addressed — Current concept inventories distinguish pure selection/validation from Store, Board, renderer, and publication integration boundaries.
- BR-6 — withdrawn — Withdrawal retained. Native log SHA-256 matches its evidence record; the evidence revision and reviewed head differ only in evidence files and the issue log.
- BR-7 — addressed — Required sketch members are explicit in both schema variants; exhaustive omission and Store-read regressions pass.
- BR-8 — addressed — Historical page rendering requests originalOnly placement; replacement-anchor refusal has browser and native regressions.
- BR-9 — addressed — Shipped guide lines 665–705 and ReadingGuide describe root sketch_page/index/paths evidence and qualify per-sketch images as legacy, matching the implemented variants.
- BR-10 — addressed — Production-connect regression passes repeated errors, stale revisions, and recovery. Restoring beginStream's old clear() call in a disposable pinned checkout makes it fail at viewer.test.mjs:258; checkout removed afterward.

### Raised

- **BR-11** [Important] `handoff-contract-alignment` (kind: defect) Shipped-skill test still requires wording removed by the shared-image correction
  cmd/internal/couchcmd/skill_privacy_test.go:18 requires “Its image has `replay-rendered` provenance”; the current guide correctly explains shared sketch_page and legacy images instead. TestSkillBroadcastPrivacyContract/historical_sketch_provenance consequently fails at line 44. This is the 2nd finding in family handoff-contract-alignment (ARCH-PURPOSE). State the rule that every handoff contract assertion follows the current shared/legacy representations, then sweep ReadingGuide and shipped-skill assertions together; the inspected assertions contain this stale sentence. Align the test with that contract and rerun couchcmd plus required final checks.

## Round 9 — 2026-10-22T14:57:28-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retained: canceled publications settle through PreviewPublishFailed; controlled cancellation and late-ACK regressions pass.
- BR-2 — addressed — Retained: element-fact allocation reserves mandatory anchor evidence before optional declarations; budget-priority regressions pass.
- BR-3 — addressed — Retained: absent anchors no longer create node-zero facts; region-only Build and capture-context regression passes.
- BR-4 — addressed — Retained: strict schema preflight rejects null pressure and invalid known-field types; codec and Store regressions pass.
- BR-5 — addressed — Retained: corrected Core concepts tables and symbol reconciliation distinguish pure selectors from filesystem, recorder and browser integration owners.
- BR-6 — withdrawn — Retained withdrawal: native logs match their recorded SHA-256 hashes and report successful fidelity, historical rendering and integrated current-view qualification; real-operator acceptance remains explicitly separate.
- BR-7 — addressed — Retained: explicit required-member schemas reject omitted source evidence; required-field deletion and Store-read regressions pass.
- BR-8 — addressed — Retained: historical rendering requires original identity through originalOnly placement; replacement-anchor refusal regressions pass.
- BR-9 — addressed — Retained: ReadingGuide and shipped skill describe root sketch_page evidence, original paths, current projections and legacy per-sketch representations consistently with the codec.
- BR-10 — addressed — Retained: beginStream preserves confirmed display facts while rotating stream ownership; reconnect and outage browser regressions pass.
- BR-11 — addressed — skill_privacy_test.go:18 now asserts current-view sketch_page provenance alongside shared indexes, original paths and legacy images. Read-only evaluation finds the old historical-provenance assertion fails while all 33 current assertions match; couchcmd passes.

## Open findings

(none — every finding has been disposed)
