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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a750da2d-99ae-4c0b-a4c5-a2e26088d435
            repository: github.com/xianxu/pair
            issue_id: "000752"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a980fba124a9a86a
            reviewed_head: 636d1e1a19a65fb3f5897cee9f3b15df2f51bd7a
            slot: pair:9
            requested_at: "2026-10-21T10:59:31.671278Z"
          phase: finished
          pid: 52529
          started_at: "2026-10-21T10:59:31.765548Z"
          finished_at: "2026-10-21T11:03:17.495705Z"
          session_id: 01a15e1d-c635-7962-b2e9-6c2b7e72831a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 636d1e1a19a65fb3f5897cee9f3b15df2f51bd7a
        tracker_head: f4da4d3ef2cddf52c77db0bde616741c314b4fa0
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:30:29-07:00"
        until: "2026-10-21T03:58:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:30:29-07:00"
              end: "2026-10-21T03:58:55-07:00"
              began: "2026-10-21T00:30:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 45124c48421ccd34a6e84b9eb08cb0d41906c925eed0e814acb94f1df5998b02
              started: "2026-10-21T04:42:37.82Z"
              spawned: false
              bytes: 17977875
              windows:
                - start: "2026-10-21T00:30:29-07:00"
                  end: "2026-10-21T03:58:55-07:00"
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
                  end: "2026-10-21T03:58:55-07:00"
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
                  end: "2026-10-21T03:58:55-07:00"
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
                  end: "2026-10-21T03:58:55-07:00"
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
            - dispatched: "2026-10-21T03:49:07-07:00"
              finalized: "2026-10-21T03:53:20-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-21T04:03:18-07:00"
      dispatched: "2026-10-21T03:59:30-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan retains one function-verification strategy table; task checklists reference it instead of repeating case inventories.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Retain the accepted disposition. Authoritative metadata reaches shared policy; lifecycle and pending-label Node regressions pass.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Queue-saturation regressions pass for movement, pointer-up, unknown confirmation and explicit refusal recovery, retaining one truncated terminal.
          round: 4
        - id: BR-4
          disposition: addressed
          note: The corrected Core concepts locations and modification statuses match the inspected pinned range.
          round: 4
        - id: BR-5
          disposition: withdrawn
          note: Retain the prior evidence-based withdrawal. This environment again failed Chrome startup with GPU-process errors/CDP EOF; that repeats the disposed environment limitation.
          round: 4
        - id: BR-6
          disposition: addressed
          note: Both producers use base width 3. In a disposable pinned-head copy, restoring width 4 independently failed TestSketchServiceFrozenAnchorAndExactRetry and the optimistic-contact pressure regression. Both pass at head.
          round: 4
        - id: BR-7
          disposition: addressed
          note: Plan lines 29–44 restrict PURE to named deterministic entities; lines 59 and 65–68 explicitly classify random journal ownership, policy callbacks, canvas/scheduling and delivery callbacks as INTEGRATION. Supporting implementations include journal init(), paintSketchPaths() and createSketchDelivery(); Revisions records the complete classification sweep.
          round: 4
      findings:
        - kind: defect
          id: BR-8
          severity: Critical
          title: Leaving before the asynchronous anchor resolves discards the stroke
          detail: 'cmd/internal/webcast/web/sketch.js:461–464 calls finish(), immediately dispatches leave, then awaits pending work. If scene.hitTest is unresolved, finish only buffers the terminal; the later anchor callback at lines 363–365 emits stroke-begin/points/end after sketch-policy.js:173 has entered label-required, so the guard at lines 190–201 rejects every command. Using the existing mount harness, all six combinations of Done/Escape/tool change and open/released contact completed with zero strokes and only sketch-begin plus sketch-label sent. ARCH-ORDER/ARCH-PURPOSE: introduce an explicit finishing transition that blocks new input and label submission while admitting and settling the existing gesture. This is the 2nd finding in family bounded-input-retains-terminal-work: state the preservation rule and sweep every voluntary exit across pending-anchor, queued and uncertain-work states on both adapters, with controlled-order regression tests.'
          family: bounded-input-retains-terminal-work
          round: 4
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 44f88c08-8708-45c0-b5ca-4c68de1d92c5
            repository: github.com/xianxu/pair
            issue_id: "000752"
            recipe: milestone-review
            round: 5
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/d48da7d342cc7fc3
            reviewed_head: 208edef0488a0171064d6892db65d9a878880288
            slot: pair:9
            requested_at: "2026-10-21T11:18:13.540205Z"
          phase: finished
          pid: 78378
          started_at: "2026-10-21T11:18:13.708684Z"
          finished_at: "2026-10-21T11:21:38.74408Z"
          session_id: 01a15e2e-e583-7853-82da-5bdbb6b61475
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 208edef0488a0171064d6892db65d9a878880288
        tracker_head: 627c11120a0bef231a162263412c952c7c74601e
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:30:29-07:00"
        until: "2026-10-21T04:17:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:30:29-07:00"
              end: "2026-10-21T04:17:19-07:00"
              began: "2026-10-21T00:30:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: c07d81451b30e3023f50c795e9776badbc782a36191a43aedcb72ab1b94e5697
              started: "2026-10-21T04:42:37.82Z"
              spawned: false
              bytes: 19951345
              windows:
                - start: "2026-10-21T00:30:29-07:00"
                  end: "2026-10-21T04:17:19-07:00"
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
                  end: "2026-10-21T04:17:19-07:00"
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
                  end: "2026-10-21T04:17:19-07:00"
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
                  end: "2026-10-21T04:17:19-07:00"
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
            - dispatched: "2026-10-21T03:49:07-07:00"
              finalized: "2026-10-21T03:53:20-07:00"
              finalizing: false
            - dispatched: "2026-10-21T03:59:30-07:00"
              finalized: "2026-10-21T04:03:18-07:00"
              finalizing: false
      "n": 5
      timestamp: "2026-10-21T04:21:39-07:00"
      dispatched: "2026-10-21T04:18:10-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Retain prior disposition: implementation tasks reference the shared function-verification table instead of repeating case inventories.'
          round: 5
        - id: BR-2
          disposition: addressed
          note: 'Retain prior disposition: console authoritative lifecycle and late-label regressions pass in the pinned-head Node suite.'
          round: 5
        - id: BR-3
          disposition: addressed
          note: 'Retain prior disposition: queue-saturation terminal and pointer-up regressions pass; the finding below concerns a separate pending-anchor ordering.'
          round: 5
        - id: BR-4
          disposition: addressed
          note: 'Retain prior disposition: Core concepts paths and new/modified/reused classifications match the inspected implementation range.'
          round: 5
        - id: BR-5
          disposition: withdrawn
          note: Retain prior withdrawal. Local live checks again encounter GPU-process startup failure/CDP EOF; this does not reopen the previously disposed environment finding.
          round: 5
        - id: BR-6
          disposition: addressed
          note: 'Retain prior disposition: producer and optimistic painter use width 3; durable-width and pressure regressions pass.'
          round: 5
        - id: BR-7
          disposition: addressed
          note: 'Retain prior disposition: PURE rows identify deterministic symbols, while callbacks, canvas, journal ownership and live adapters are classified INTEGRATION.'
          round: 5
        - id: BR-8
          disposition: addressed
          note: All 16 viewer/console delayed-anchor exit regressions pass at head and fail with the finishing fix removed in a temporary copy; the temporary copy was removed.
          round: 5
      findings:
        - kind: defect
          id: BR-9
          severity: Critical
          title: A second contact before the previous anchor resolves crashes drawing on both surfaces
          detail: 'At cmd/internal/webcast/web/sketch.js:352–357, pointer-up clears contact while the logical gesture remains pending its anchor. The down guard at lines 367–376 admits another contact and mutates contact before createSketchGesture.start rejects it at sketch-policy.js:437. Controlled viewer and console harnesses reproduce “gesture already open”, followed by a TypeError reading contact.last at sketch.js:346 on the next pointer-up; the second stroke is absent. ARCH-ORDER/ARCH-PURPOSE: this is the 3rd finding in family bounded-input-retains-terminal-work. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that physical contact release does not retire logical gesture ownership. Enumerate pointer-up, pointercancel, lost capture and automatic buffer/point-limit endings, and prevent or explicitly queue successor gestures until their predecessor relinquishes ownership. Add controlled-order regressions on both adapters.'
          family: bounded-input-retains-terminal-work
          round: 5
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 256d00bf-cf86-4b57-9c5b-5e91a5efea40
            repository: github.com/xianxu/pair
            issue_id: "000752"
            recipe: milestone-review
            round: 6
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/5f546d39033bfe5d
            reviewed_head: 320d01886adaa2859c3bfd1c51306826dc38e32b
            slot: pair:9
            requested_at: "2026-10-21T11:37:50.369516Z"
          phase: finished
          pid: 54911
          started_at: "2026-10-21T11:37:50.470553Z"
          finished_at: "2026-10-21T11:42:23.283692Z"
          session_id: 01a15e40-d98d-7b72-9af9-318b4623b594
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 320d01886adaa2859c3bfd1c51306826dc38e32b
        tracker_head: e901f36afdc2db444d2638e4e3959136820600fc
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:30:29-07:00"
        until: "2026-10-21T04:37:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:30:29-07:00"
              end: "2026-10-21T04:37:19-07:00"
              began: "2026-10-21T00:30:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 665c78cb105767006665970e4c786a93042a974b1ef154d56d868398ce8d4294
              started: "2026-10-21T04:42:37.82Z"
              spawned: false
              bytes: 21229921
              windows:
                - start: "2026-10-21T00:30:29-07:00"
                  end: "2026-10-21T04:37:19-07:00"
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
                  end: "2026-10-21T04:37:19-07:00"
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
                  end: "2026-10-21T04:37:19-07:00"
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
                  end: "2026-10-21T04:37:19-07:00"
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
            - dispatched: "2026-10-21T03:49:07-07:00"
              finalized: "2026-10-21T03:53:20-07:00"
              finalizing: false
            - dispatched: "2026-10-21T03:59:30-07:00"
              finalized: "2026-10-21T04:03:18-07:00"
              finalizing: false
            - dispatched: "2026-10-21T04:18:10-07:00"
              finalized: "2026-10-21T04:21:39-07:00"
              finalizing: false
      "n": 6
      timestamp: "2026-10-21T04:42:24-07:00"
      dispatched: "2026-10-21T04:37:48-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Retain disposition; the plan uses the function verification strategy table.
          round: 6
        - id: BR-2
          disposition: addressed
          note: Console handoff and old-author label regressions pass.
          round: 6
        - id: BR-3
          disposition: addressed
          note: Queue saturation reserves a truncated terminal; move and pointer-up regressions pass.
          round: 6
        - id: BR-4
          disposition: addressed
          note: Core concepts now identify existing implementation paths and distinguish new, modified and reused surfaces.
          round: 6
        - id: BR-5
          disposition: withdrawn
          note: Retain withdrawal. This environment again failed during Chrome GPU/sandbox startup before assertions; no new conformance claim is made.
          round: 6
        - id: BR-6
          disposition: addressed
          note: Producer and optimistic painter both use base width 3; relevant Go and Node tests pass.
          round: 6
        - id: BR-7
          disposition: addressed
          note: Core concepts distinguish deterministic symbols from effectful wrappers and their integration tests.
          round: 6
        - id: BR-8
          disposition: addressed
          note: Delayed-anchor exit regressions pass for both adapters, including open and released contacts.
          round: 6
        - id: BR-9
          disposition: addressed
          note: Eight pointer-up/cancel/lost-capture/buffer-limit regressions pass at head and all fail with “gesture already open” when sketch.js is restored to its pre-fix version in a disposable checkout.
          round: 6
      findings:
        - kind: defect
          id: BR-10
          severity: Important
          title: Enter does not submit the required sketch label on either surface
          detail: 'ARCH-PURPOSE: the approved interaction at workshop/history/plans/000750-cast-sketch-plan.md:36 says Enter saves a nonblank label and returns to the requested tool. The shared handler at cmd/internal/webcast/web/sketch.js:507 handles only Escape. Controlled viewer and console reproductions at pinned head leave zero sketch-label commands after Enter. Route Enter through saveLabel, preserving its settlement guards, and add both-adapter regressions proving the requested tool opens only after acceptance.'
          family: approved-controls-match-behavior
          round: 6
      recipe: milestone-review
      isolation: fenced
      reviewed: 320d01886adaa2859c3bfd1c51306826dc38e32b
      blocked: false
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

## Round 4 — 2026-10-21T04:03:18-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The plan retains one function-verification strategy table; task checklists reference it instead of repeating case inventories.
- BR-2 — addressed — Retain the accepted disposition. Authoritative metadata reaches shared policy; lifecycle and pending-label Node regressions pass.
- BR-3 — addressed — Queue-saturation regressions pass for movement, pointer-up, unknown confirmation and explicit refusal recovery, retaining one truncated terminal.
- BR-4 — addressed — The corrected Core concepts locations and modification statuses match the inspected pinned range.
- BR-5 — withdrawn — Retain the prior evidence-based withdrawal. This environment again failed Chrome startup with GPU-process errors/CDP EOF; that repeats the disposed environment limitation.
- BR-6 — addressed — Both producers use base width 3. In a disposable pinned-head copy, restoring width 4 independently failed TestSketchServiceFrozenAnchorAndExactRetry and the optimistic-contact pressure regression. Both pass at head.
- BR-7 — addressed — Plan lines 29–44 restrict PURE to named deterministic entities; lines 59 and 65–68 explicitly classify random journal ownership, policy callbacks, canvas/scheduling and delivery callbacks as INTEGRATION. Supporting implementations include journal init(), paintSketchPaths() and createSketchDelivery(); Revisions records the complete classification sweep.

### Raised

- **BR-8** [Critical] `bounded-input-retains-terminal-work` (kind: defect) Leaving before the asynchronous anchor resolves discards the stroke
  cmd/internal/webcast/web/sketch.js:461–464 calls finish(), immediately dispatches leave, then awaits pending work. If scene.hitTest is unresolved, finish only buffers the terminal; the later anchor callback at lines 363–365 emits stroke-begin/points/end after sketch-policy.js:173 has entered label-required, so the guard at lines 190–201 rejects every command. Using the existing mount harness, all six combinations of Done/Escape/tool change and open/released contact completed with zero strokes and only sketch-begin plus sketch-label sent. ARCH-ORDER/ARCH-PURPOSE: introduce an explicit finishing transition that blocks new input and label submission while admitting and settling the existing gesture. This is the 2nd finding in family bounded-input-retains-terminal-work: state the preservation rule and sweep every voluntary exit across pending-anchor, queued and uncertain-work states on both adapters, with controlled-order regression tests.

## Round 5 — 2026-10-21T04:21:39-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Retain prior disposition: implementation tasks reference the shared function-verification table instead of repeating case inventories.
- BR-2 — addressed — Retain prior disposition: console authoritative lifecycle and late-label regressions pass in the pinned-head Node suite.
- BR-3 — addressed — Retain prior disposition: queue-saturation terminal and pointer-up regressions pass; the finding below concerns a separate pending-anchor ordering.
- BR-4 — addressed — Retain prior disposition: Core concepts paths and new/modified/reused classifications match the inspected implementation range.
- BR-5 — withdrawn — Retain prior withdrawal. Local live checks again encounter GPU-process startup failure/CDP EOF; this does not reopen the previously disposed environment finding.
- BR-6 — addressed — Retain prior disposition: producer and optimistic painter use width 3; durable-width and pressure regressions pass.
- BR-7 — addressed — Retain prior disposition: PURE rows identify deterministic symbols, while callbacks, canvas, journal ownership and live adapters are classified INTEGRATION.
- BR-8 — addressed — All 16 viewer/console delayed-anchor exit regressions pass at head and fail with the finishing fix removed in a temporary copy; the temporary copy was removed.

### Raised

- **BR-9** [Critical] `bounded-input-retains-terminal-work` (kind: defect) A second contact before the previous anchor resolves crashes drawing on both surfaces
  At cmd/internal/webcast/web/sketch.js:352–357, pointer-up clears contact while the logical gesture remains pending its anchor. The down guard at lines 367–376 admits another contact and mutates contact before createSketchGesture.start rejects it at sketch-policy.js:437. Controlled viewer and console harnesses reproduce “gesture already open”, followed by a TypeError reading contact.last at sketch.js:346 on the next pointer-up; the second stroke is absent. ARCH-ORDER/ARCH-PURPOSE: this is the 3rd finding in family bounded-input-retains-terminal-work. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that physical contact release does not retire logical gesture ownership. Enumerate pointer-up, pointercancel, lost capture and automatic buffer/point-limit endings, and prevent or explicitly queue successor gestures until their predecessor relinquishes ownership. Add controlled-order regressions on both adapters.

## Round 6 — 2026-10-21T04:42:24-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retain disposition; the plan uses the function verification strategy table.
- BR-2 — addressed — Console handoff and old-author label regressions pass.
- BR-3 — addressed — Queue saturation reserves a truncated terminal; move and pointer-up regressions pass.
- BR-4 — addressed — Core concepts now identify existing implementation paths and distinguish new, modified and reused surfaces.
- BR-5 — withdrawn — Retain withdrawal. This environment again failed during Chrome GPU/sandbox startup before assertions; no new conformance claim is made.
- BR-6 — addressed — Producer and optimistic painter both use base width 3; relevant Go and Node tests pass.
- BR-7 — addressed — Core concepts distinguish deterministic symbols from effectful wrappers and their integration tests.
- BR-8 — addressed — Delayed-anchor exit regressions pass for both adapters, including open and released contacts.
- BR-9 — addressed — Eight pointer-up/cancel/lost-capture/buffer-limit regressions pass at head and all fail with “gesture already open” when sketch.js is restored to its pre-fix version in a disposable checkout.

### Raised

- **BR-10** [Important] `approved-controls-match-behavior` (kind: defect) Enter does not submit the required sketch label on either surface
  ARCH-PURPOSE: the approved interaction at workshop/history/plans/000750-cast-sketch-plan.md:36 says Enter saves a nonblank label and returns to the requested tool. The shared handler at cmd/internal/webcast/web/sketch.js:507 handles only Escape. Controlled viewer and console reproductions at pinned head leave zero sketch-label commands after Enter. Route Enter through saveLabel, preserving its settlement guards, and add both-adapter regressions proving the requested tool opens only after acceptance.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-5** — after round 2 (2026-10-21T10:12:38Z, Xian Xu on Xian’s MacBook Pro at pair:9 (/Users/xianxu/workspace/worktree/pair-slot9/pair))
  rationale: The review checkout could not start Chrome before its assertions ran; the same required tests execute successfully on this host. Corrected implementation is committed and host conformance is measured; TL operator console smoke remains a separate pre-landing hold.
  evidence: On committed8be34c6fab5b1cc1c2431a7cadb1da915b2e793d: WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run SketchLive\|OperatorSketchLive -count=1 -v exited0, package9.541s. ScenePrivacy2.16s; actualConsoleDrawing2.51s; OpenContactHandoffLabel1.80s; TwoViewers2.91s. Pressure beforeup, Undo/Redo/replay, named-target/blob denial, moved-original/replacement and old-label/new-holder source continuity assertions all executed and passed.

## Open findings

- **BR-10** [Important] `approved-controls-match-behavior` (kind: defect) Enter does not submit the required sketch label on either surface
