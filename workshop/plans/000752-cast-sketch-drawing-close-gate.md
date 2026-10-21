---
gate: boundary-review
issue: 752
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-21T02:55:07-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Compress repeated prose test-case inventories
          detail: |-
            The function verification table already names risky functions, adversarial input classes and mechanical guards, but implementation tasks repeat enumerated test cases, notably at plan lines 177 and 192–193. Replace those repeated inventories with references to the strategy table, retaining verification commands and independent invariants.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: strategy-over-case-inventory
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 49c4f4f9-2fc4-41fe-9ee7-c8d91eed8d50
            repository: github.com/xianxu/pair
            issue_id: "000752"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/c1af59ffccdebf8f
            reviewed_head: 60a400cb13be0e54e0469e36ce5d3805461cc07c
            slot: pair:9
            requested_at: "2026-10-21T09:50:14.707355Z"
          phase: finished
          pid: 59176
          started_at: "2026-10-21T09:50:14.802301Z"
          finished_at: "2026-10-21T09:55:06.46759Z"
          session_id: 01a15dde-581e-7a91-ad27-a35d3046dd27
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 60a400cb13be0e54e0469e36ce5d3805461cc07c
        tracker_head: 5c7dc2861b6dcfd22503c354e9088050ae58cb5e
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:30:29-07:00"
        until: "2026-10-21T02:49:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:30:29-07:00"
              end: "2026-10-21T02:49:55-07:00"
              began: "2026-10-21T00:30:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 352225fb000986ae2aa42312c372b9cf2fdff8b7d891031628abd4454f9b6517
              started: "2026-10-21T04:42:37.82Z"
              spawned: false
              bytes: 15495428
              windows:
                - start: "2026-10-21T00:30:29-07:00"
                  end: "2026-10-21T02:49:55-07:00"
            - harness: codex
              id: 01a15d72-f362-71f2-ac46-571622add1dd
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 21e6662300c1bed4a15a4792083ab14d80a05c66377f7b4994ab4ba521841f36
              started: "2026-10-21T07:52:57.643Z"
              spawned: false
              bytes: 801571
              windows:
                - start: "2026-10-21T07:52:57.643Z"
                  end: "2026-10-21T02:49:55-07:00"
            - harness: codex
              id: 01a15d78-07d8-7e30-b172-25de09195a37
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c6881ead8ac3f533d352dd6c63982205c1fd552b98e8bb53bd78a6c051b1549f
              started: "2026-10-21T07:58:30.132Z"
              spawned: false
              bytes: 507194
              windows:
                - start: "2026-10-21T07:58:30.132Z"
                  end: "2026-10-21T02:49:55-07:00"
            - harness: codex
              id: 01a15d78-9f81-7ee3-bed1-6016d32c5c39
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: fd4a263d9b10335f2bb096cfaa2510a3be42c97276d0db2b453e1d80fd5ec242
              started: "2026-10-21T07:59:08.967Z"
              spawned: false
              bytes: 485363
              windows:
                - start: "2026-10-21T07:59:08.967Z"
                  end: "2026-10-21T02:49:55-07:00"
        queue_waits:
            - attempt_id: d33c24095b0e99efb97794a1065963d6
              slot: pair:9
              record_digest: cd51443b21b75b0eb664f444fd77c71f6f5f0f027620d25753056067f5b68556
              start: "2026-10-21T09:32:04.598544Z"
              end: "2026-10-21T09:38:11.880148Z"
              queue_seconds: 367.281604
              raw_seconds: 367.281604
              exact: true
      "n": 2
      timestamp: "2026-10-21T02:55:07-07:00"
      dispatched: "2026-10-21T02:50:12-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-2
          severity: Critical
          title: Console handoff leaves partial-stroke label saving permanently blocked
          detail: 'cmd/internal/webcast/web/sketch.js:470 refuses label saving while cached status.open_stroke is populated, but its lifecycle refresh at :663 requires model.sketchMetadata and actual ctl/stop/sketch-interrupt records. The console model in operator.js:173 has no such query, publishes synthetic view records, and its sketch journal receiver only refreshes painting. Reproduced an accepted open stroke followed by handoff and authoritative sealing: labelRequests remained zero with “Wait for the stroke confirmation before saving its label.” ARCH-ORDER/ARCH-PURPOSE: feed authoritative sketch metadata and lifecycle transitions into the console tool, including handoff, pause, expiry, clear and end; add an interrupted-contact console regression.'
          family: authoritative-state-reaches-ui
          round: 2
        - id: BR-3
          severity: Critical
          title: Queue saturation cancels the gesture without sealing it and can crash pointer-up
          detail: 'cmd/internal/webcast/web/sketch-policy.js:207 emits only stop-gesture when sixteen point chunks are queued; sketch.js handles that by canceling the gesture and clearing contact without retaining terminal work or entering visible recovery. Holding the first points reply and filling the queue reproduced a drained queue with the server stroke still open, zero stroke-end requests, and no status message; when saturation occurs inside finish() at sketch.js:322, line 324 throws “Cannot read properties of null (reading ''pointer'')”. ARCH-CONSTRAINTS/ARCH-ORDER: preserve a truncated terminal operation behind accepted queued work, expose reconciliation, and make finish safe against synchronous cancellation.'
          family: bounded-input-retains-terminal-work
          round: 2
        - id: BR-4
          severity: Critical
          title: The Core concepts table names nonexistent and unchanged implementation locations
          detail: workshop/plans/000752-cast-sketch-drawing-plan.md:50 names board/sketch_authority.go, which does not exist; SketchAuthority is in board/sketch.go:14. Rows 52–58 also claim modifications to recording/hub.go, browser.go, server_drive.go, web/annotation-placement.js, recorder/operator.go, board/operator.go, fake.go and fake_drive.go that are absent from the pinned diff. Per the required Core concepts cross-check, reconcile the complete table with actual files and statuses, including sketch_source.go, cast_sketch.go and writer_sketch.go, and record the correction under Revisions.
          family: core-concept-map-matches-code
          round: 2
        - kind: unverifiable
          id: BR-5
          severity: Important
          title: Required Chrome conformance assertions could not execute
          detail: Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v. Both operator tests failed before opening their page with GPU-process/sandbox initialization failures, and TestSketchLiveTwoViewers failed with CDP EOF. Supply successful host results for these tests on the reviewed implementation, or provide an environment where Chrome starts, to verify private-scene isolation and real two-viewer/console behavior.
          family: live-conformance-evidence
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f33df438-785c-4978-be54-05bb60d101f3
            repository: github.com/xianxu/pair
            issue_id: "000752"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/540ec201ce77f0f3
            reviewed_head: e9ea2bbf3ed082d870a2f67af932ce78dff28d93
            slot: pair:9
            requested_at: "2026-10-21T10:49:10.097006Z"
          phase: finished
          pid: 23898
          started_at: "2026-10-21T10:49:10.264146Z"
          finished_at: "2026-10-21T10:53:19.593029Z"
          session_id: 01a15e14-4afd-7230-8ed8-0e57f11ca24e
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: e9ea2bbf3ed082d870a2f67af932ce78dff28d93
        tracker_head: 9deb05ed1862cd3d57a905e3fe5bb5083a0ac74c
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:30:29-07:00"
        until: "2026-10-21T03:48:27-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:30:29-07:00"
              end: "2026-10-21T03:48:27-07:00"
              began: "2026-10-21T00:30:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 1db81f6aefeaab2514233dc7232593df8933e30bd1644330cedb7d1293e91f3b
              started: "2026-10-21T04:42:37.82Z"
              spawned: false
              bytes: 17466371
              windows:
                - start: "2026-10-21T00:30:29-07:00"
                  end: "2026-10-21T03:48:27-07:00"
            - harness: codex
              id: 01a15d72-f362-71f2-ac46-571622add1dd
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 21e6662300c1bed4a15a4792083ab14d80a05c66377f7b4994ab4ba521841f36
              started: "2026-10-21T07:52:57.643Z"
              spawned: false
              bytes: 801571
              windows:
                - start: "2026-10-21T07:52:57.643Z"
                  end: "2026-10-21T03:48:27-07:00"
            - harness: codex
              id: 01a15d78-07d8-7e30-b172-25de09195a37
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c6881ead8ac3f533d352dd6c63982205c1fd552b98e8bb53bd78a6c051b1549f
              started: "2026-10-21T07:58:30.132Z"
              spawned: false
              bytes: 507194
              windows:
                - start: "2026-10-21T07:58:30.132Z"
                  end: "2026-10-21T03:48:27-07:00"
            - harness: codex
              id: 01a15d78-9f81-7ee3-bed1-6016d32c5c39
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: fd4a263d9b10335f2bb096cfaa2510a3be42c97276d0db2b453e1d80fd5ec242
              started: "2026-10-21T07:59:08.967Z"
              spawned: false
              bytes: 485363
              windows:
                - start: "2026-10-21T07:59:08.967Z"
                  end: "2026-10-21T03:48:27-07:00"
        queue_waits:
            - attempt_id: d33c24095b0e99efb97794a1065963d6
              slot: pair:9
              record_digest: cd51443b21b75b0eb664f444fd77c71f6f5f0f027620d25753056067f5b68556
              start: "2026-10-21T09:32:04.598544Z"
              end: "2026-10-21T09:38:11.880148Z"
              queue_seconds: 367.281604
              raw_seconds: 367.281604
              exact: true
            - attempt_id: 53424823b0e2ed3099f5dab5c5a415b2
              slot: pair:9
              record_digest: a978dd3d291fdae1b9bc1ebcc7c7efd0d01c6dcbd09aa84438f6030c86facb5d
              start: "2026-10-21T10:14:41.500881Z"
              end: "2026-10-21T10:35:20.504649Z"
              queue_seconds: 1239.003768
              raw_seconds: 1239.003768
              exact: true
        cuts:
            - dispatched: "2026-10-21T02:50:12-07:00"
              finalized: "2026-10-21T02:55:07-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-21T03:53:20-07:00"
      dispatched: "2026-10-21T03:49:07-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Inspected the before/after plan diff: Tasks 1–5 now reference the Function verification strategy table while retaining commands and independent invariants. The correction is recorded under Revisions.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: Console delivery now supplies authoritative metadata to shared reconciliation. Lifecycle, stale-status, draft-retention and held-status tests pass. Reverting the browser fixes in a disposable copy makes the interrupted-contact label regression fail.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Saturation reserves one truncated terminal and enters visible reconciliation; finish detaches contact before synchronous effects. Move, pointer-up, unknown and refused-operation regressions pass. Reverting the fixes reproduces both missing recovery and the null.pointer crash.
          round: 3
        - id: BR-4
          disposition: addressed
          note: The revised integration table names board/sketch.go, recording/sketch_source.go, cast_sketch.go and board/writer_sketch.go correctly and identifies unchanged interfaces/doubles as reused. These path/status corrections match the pinned diff and are recorded under Revisions. A distinct PURE classification contradiction is reported below.
          round: 3
        - id: BR-5
          disposition: withdrawn
          note: The supplied host run on 8be34c6fab5b1cc1c2431a7cadb1da915b2e793d executed all four required tests successfully. The diff from that commit to reviewed head changes only tracker/plan records, so the measurement covers this executable implementation. My rerun still fails before assertions with Chrome GPU/sandbox startup errors and CDP EOF; that does not contradict the supplied host results.
          round: 3
      findings:
        - id: BR-6
          severity: Critical
          title: Both live and persisted strokes violate the approved 3 px width contract
          detail: 'board/sketch.go:305 persists Width: 4 and web/sketch.js:354 uses optimistic width 4. The approved design at workshop/history/plans/000750-cast-sketch-plan.md:200 and execution plan at workshop/plans/000752-cast-sketch-drawing-plan.md:85 require base width 3. This makes every pressure-derived width one-third thicker, including replay. A disposable Go test overlay changing the existing service assertion to the specified width fails with “durable width 4 differs from approved fixed sketch tool width 3.” ARCH-PURPOSE: restore 3 in both paths and pin the specified pressure widths in regression tests; the existing assertion at board/sketch_test.go:49 currently enshrines the incorrect value.'
          family: approved-rendering-contract
          round: 3
        - id: BR-7
          severity: Critical
          title: The Core concepts table still labels effectful components as PURE
          detail: 'workshop/plans/000752-cast-sketch-drawing-plan.md:38 classifies the painter as PURE, but web/sketch-render.js:343 mutates an injected canvas context, and tests/webcast/sketch-render.test.mjs:92 supplies a canvas double. Row 40 also classifies the journal as PURE although board/sketch_delivery.go:136 obtains a random epoch through randomID(). ARCH-PURE: distinguish the pure geometry/projection entities from the INTEGRATION canvas and random-initialization adapters, and record the correction under Revisions. This is the 2nd finding in family core-concept-map-matches-code. State and apply the rule across every table row: classification must follow actual effects and test dependencies, not merely file placement; sweep the complete table rather than correcting only these examples.'
          family: core-concept-map-matches-code
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
disputes:
    - id: BR-5
      rationale: The review checkout could not start Chrome before its assertions ran; the same required tests execute successfully on this host. Corrected implementation is committed and host conformance is measured; TL operator console smoke remains a separate pre-landing hold.
      evidence: 'On committed8be34c6fab5b1cc1c2431a7cadb1da915b2e793d: WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run SketchLive\|OperatorSketchLive -count=1 -v exited0, package9.541s. ScenePrivacy2.16s; actualConsoleDrawing2.51s; OpenContactHandoffLabel1.80s; TwoViewers2.91s. Pressure beforeup, Undo/Redo/replay, named-target/blob denial, moved-original/replacement and old-label/new-holder source continuity assertions all executed and passed.'
      after: 2
      timestamp: "2026-10-21T10:12:38Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:9 (/Users/xianxu/workspace/worktree/pair-slot9/pair)
---

# Gate ledger — pair#752 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T02:55:07-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `strategy-over-case-inventory` Compress repeated prose test-case inventories
  The function verification table already names risky functions, adversarial input classes and mechanical guards, but implementation tasks repeat enumerated test cases, notably at plan lines 177 and 192–193. Replace those repeated inventories with references to the strategy table, retaining verification commands and independent invariants.
  (carried from plan-quality PQ-1, deferred to the boundary review)

## Round 2 — 2026-10-21T02:55:07-07:00 (codex) — BLOCKED

### Raised

- **BR-2** [Critical] `authoritative-state-reaches-ui` Console handoff leaves partial-stroke label saving permanently blocked
  cmd/internal/webcast/web/sketch.js:470 refuses label saving while cached status.open_stroke is populated, but its lifecycle refresh at :663 requires model.sketchMetadata and actual ctl/stop/sketch-interrupt records. The console model in operator.js:173 has no such query, publishes synthetic view records, and its sketch journal receiver only refreshes painting. Reproduced an accepted open stroke followed by handoff and authoritative sealing: labelRequests remained zero with “Wait for the stroke confirmation before saving its label.” ARCH-ORDER/ARCH-PURPOSE: feed authoritative sketch metadata and lifecycle transitions into the console tool, including handoff, pause, expiry, clear and end; add an interrupted-contact console regression.
- **BR-3** [Critical] `bounded-input-retains-terminal-work` Queue saturation cancels the gesture without sealing it and can crash pointer-up
  cmd/internal/webcast/web/sketch-policy.js:207 emits only stop-gesture when sixteen point chunks are queued; sketch.js handles that by canceling the gesture and clearing contact without retaining terminal work or entering visible recovery. Holding the first points reply and filling the queue reproduced a drained queue with the server stroke still open, zero stroke-end requests, and no status message; when saturation occurs inside finish() at sketch.js:322, line 324 throws “Cannot read properties of null (reading 'pointer')”. ARCH-CONSTRAINTS/ARCH-ORDER: preserve a truncated terminal operation behind accepted queued work, expose reconciliation, and make finish safe against synchronous cancellation.
- **BR-4** [Critical] `core-concept-map-matches-code` The Core concepts table names nonexistent and unchanged implementation locations
  workshop/plans/000752-cast-sketch-drawing-plan.md:50 names board/sketch_authority.go, which does not exist; SketchAuthority is in board/sketch.go:14. Rows 52–58 also claim modifications to recording/hub.go, browser.go, server_drive.go, web/annotation-placement.js, recorder/operator.go, board/operator.go, fake.go and fake_drive.go that are absent from the pinned diff. Per the required Core concepts cross-check, reconcile the complete table with actual files and statuses, including sketch_source.go, cast_sketch.go and writer_sketch.go, and record the correction under Revisions.
- **BR-5** [Important] `live-conformance-evidence` (kind: unverifiable) Required Chrome conformance assertions could not execute
  Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v. Both operator tests failed before opening their page with GPU-process/sandbox initialization failures, and TestSketchLiveTwoViewers failed with CDP EOF. Supply successful host results for these tests on the reviewed implementation, or provide an environment where Chrome starts, to verify private-scene isolation and real two-viewer/console behavior.

## Round 3 — 2026-10-21T03:53:20-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Inspected the before/after plan diff: Tasks 1–5 now reference the Function verification strategy table while retaining commands and independent invariants. The correction is recorded under Revisions.
- BR-2 — addressed — Console delivery now supplies authoritative metadata to shared reconciliation. Lifecycle, stale-status, draft-retention and held-status tests pass. Reverting the browser fixes in a disposable copy makes the interrupted-contact label regression fail.
- BR-3 — addressed — Saturation reserves one truncated terminal and enters visible reconciliation; finish detaches contact before synchronous effects. Move, pointer-up, unknown and refused-operation regressions pass. Reverting the fixes reproduces both missing recovery and the null.pointer crash.
- BR-4 — addressed — The revised integration table names board/sketch.go, recording/sketch_source.go, cast_sketch.go and board/writer_sketch.go correctly and identifies unchanged interfaces/doubles as reused. These path/status corrections match the pinned diff and are recorded under Revisions. A distinct PURE classification contradiction is reported below.
- BR-5 — withdrawn — The supplied host run on 8be34c6fab5b1cc1c2431a7cadb1da915b2e793d executed all four required tests successfully. The diff from that commit to reviewed head changes only tracker/plan records, so the measurement covers this executable implementation. My rerun still fails before assertions with Chrome GPU/sandbox startup errors and CDP EOF; that does not contradict the supplied host results.

### Raised

- **BR-6** [Critical] `approved-rendering-contract` Both live and persisted strokes violate the approved 3 px width contract
  board/sketch.go:305 persists Width: 4 and web/sketch.js:354 uses optimistic width 4. The approved design at workshop/history/plans/000750-cast-sketch-plan.md:200 and execution plan at workshop/plans/000752-cast-sketch-drawing-plan.md:85 require base width 3. This makes every pressure-derived width one-third thicker, including replay. A disposable Go test overlay changing the existing service assertion to the specified width fails with “durable width 4 differs from approved fixed sketch tool width 3.” ARCH-PURPOSE: restore 3 in both paths and pin the specified pressure widths in regression tests; the existing assertion at board/sketch_test.go:49 currently enshrines the incorrect value.
- **BR-7** [Critical] `core-concept-map-matches-code` The Core concepts table still labels effectful components as PURE
  workshop/plans/000752-cast-sketch-drawing-plan.md:38 classifies the painter as PURE, but web/sketch-render.js:343 mutates an injected canvas context, and tests/webcast/sketch-render.test.mjs:92 supplies a canvas double. Row 40 also classifies the journal as PURE although board/sketch_delivery.go:136 obtains a random epoch through randomID(). ARCH-PURE: distinguish the pure geometry/projection entities from the INTEGRATION canvas and random-initialization adapters, and record the correction under Revisions. This is the 2nd finding in family core-concept-map-matches-code. State and apply the rule across every table row: classification must follow actual effects and test dependencies, not merely file placement; sweep the complete table rather than correcting only these examples.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-5** — after round 2 (2026-10-21T10:12:38Z, Xian Xu on Xian’s MacBook Pro at pair:9 (/Users/xianxu/workspace/worktree/pair-slot9/pair))
  rationale: The review checkout could not start Chrome before its assertions ran; the same required tests execute successfully on this host. Corrected implementation is committed and host conformance is measured; TL operator console smoke remains a separate pre-landing hold.
  evidence: On committed8be34c6fab5b1cc1c2431a7cadb1da915b2e793d: WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run SketchLive\|OperatorSketchLive -count=1 -v exited0, package9.541s. ScenePrivacy2.16s; actualConsoleDrawing2.51s; OpenContactHandoffLabel1.80s; TwoViewers2.91s. Pressure beforeup, Undo/Redo/replay, named-target/blob denial, moved-original/replacement and old-label/new-holder source continuity assertions all executed and passed.

## Open findings

- **BR-6** [Critical] `approved-rendering-contract` Both live and persisted strokes violate the approved 3 px width contract
- **BR-7** [Critical] `core-concept-map-matches-code` The Core concepts table still labels effectful components as PURE
