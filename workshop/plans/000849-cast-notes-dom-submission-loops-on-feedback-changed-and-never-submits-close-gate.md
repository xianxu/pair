---
gate: boundary-review
issue: 849
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 19c11fcc-7572-4279-8992-ced14070256c
            repository: github.com/xianxu/pair
            issue_id: "000849"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/193cdffb14b6c59c
            reviewed_head: da2535cc4f4909d97b5d18c92ae759acc9b4d6d4
            slot: pair:6
            requested_at: "2026-10-24T05:51:59.757327Z"
          phase: finished
          pid: 11504
          started_at: "2026-10-24T05:51:59.866344Z"
          finished_at: "2026-10-24T05:55:57.918377Z"
          session_id: 9357dd91-d710-4563-b6c2-8ad2f9fc0aca
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: da2535cc4f4909d97b5d18c92ae759acc9b4d6d4
        tracker_head: aa1226d0e3478480bcb0f042d6963c8efe45e69e
        branch_point: 4ffd05c54f8a7fbcf32f13bb95952b0d637cb59a
        since: "2026-10-23T17:38:50-07:00"
        until: "2026-10-23T22:51:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:38:50-07:00"
              end: "2026-10-23T22:51:19-07:00"
              began: "2026-10-23T17:38:50-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: claude
              id: 5ba7d27e-c4f7-4a9b-9e2c-f7ac71ef7e9a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: e84bf98289f2dca77d695b8903e8b30da221b93f6a958a76ed6f872dbcaca2e3
              started: "2026-10-23T07:52:37.426Z"
              spawned: false
              bytes: 20721293
              windows:
                - start: "2026-10-23T17:38:50-07:00"
                  end: "2026-10-23T22:51:19-07:00"
        queue_waits:
            - attempt_id: afbb4afad3d6f65e1ebf95a33e7161a4
              slot: pair:6
              record_digest: 0c6a6398072130c403f211f240473519e7846253797c13190b64916707509da3
              start: "2026-10-24T04:41:28.474788Z"
              end: "2026-10-24T04:41:41.984802Z"
              queue_seconds: 13.510014
              raw_seconds: 13.510014
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.07971666666666667
      "n": 1
      timestamp: "2026-10-23T22:55:59-07:00"
      dispatched: "2026-10-23T22:51:57-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Agent ReadingGuide still describes the visible-only gated contract and replay-rendered-current-view
          detail: cmd/internal/submission/model.go:133-134 (stored in every bundle, sent to the agent) says only console-visible feedback is selected, a stale visible selection refuses, and new sketch_page uses replay-rendered-current-view with sketch_page.current; after this change presses emit replay-rendered with no current (cast_submission.go:424-428) and include offscreen feedback. Rewrite to match atlas/webcast-submission.md and update model_test.go.
          family: contract-consumers-not-swept
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: couch SKILL.md and README.md still describe the old visible-selection submission contract
          detail: cmd/internal/couchcmd/skills/couch/SKILL.md:721,758,776-779 and README.md:1549 claim visible-only eligibility, stale-selection refusal and replay-rendered-current-view provenance; sweep them alongside the ReadingGuide fix.
          family: contract-consumers-not-swept
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Unsaved-note refusal is narrower than Done-when/Plan (untouched open edit does not refuse); no Revisions entry
          detail: Plan says an open edit refuses with "note N is being edited"; collab.js unsaved refuses only a changed edit or a draft. The choice is in the Log but the Done-when and Plan were not revised.
          family: plan-claims-match-code
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: Observer selection dedupe has no non-live unit test despite the Plan claiming one
          detail: Only the WEBCAST_LIVE_CHROME-gated TestSubmissionManagerLiveSubmitsOnAChangingPage counts selection records; the lastSel reset on resync is untested. A vm-based test like tests/webcast/observer-sketch-exclusion.test.mjs could cover it.
          family: plan-claims-match-code
          round: 1
        - kind: unverifiable
          id: BR-5
          severity: Minor
          title: Live Chrome tests (50 ms changing page, unsaved draft CDP eval) could not run in the review sandbox
          detail: Ran WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive; Chrome exited at launch with "GPU process isn't usable". A host run of these tests plus cmd/internal/webcast TestOperatorFeedbackLivePrepareAndStalePanel, ideally also failing on base 4ffd05c5, would answer it.
          family: live-chrome-evidence
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a76f1f17-726b-4131-b600-b37f8fa8072a
            repository: github.com/xianxu/pair
            issue_id: "000849"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/2077636fdb72f489
            reviewed_head: b858551ce706ac16f45d3e96069847db5c33e5c2
            slot: pair:6
            requested_at: "2026-10-24T05:59:37.224601Z"
          phase: finished
          pid: 94584
          started_at: "2026-10-24T05:59:37.32469Z"
          finished_at: "2026-10-24T06:03:32.936203Z"
          session_id: 45b2d198-d51f-4328-882b-9689d66e4f4c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: b858551ce706ac16f45d3e96069847db5c33e5c2
        tracker_head: aa1226d0e3478480bcb0f042d6963c8efe45e69e
        branch_point: 4ffd05c54f8a7fbcf32f13bb95952b0d637cb59a
        since: "2026-10-23T17:38:50-07:00"
        until: "2026-10-23T22:58:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:38:50-07:00"
              end: "2026-10-23T22:58:54-07:00"
              began: "2026-10-23T17:38:50-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: claude
              id: 5ba7d27e-c4f7-4a9b-9e2c-f7ac71ef7e9a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 53549853330db132165122c5205fdc1c047758e88a5861bcf69fc1f490e5f652
              started: "2026-10-23T07:52:37.426Z"
              spawned: false
              bytes: 20996543
              windows:
                - start: "2026-10-23T17:38:50-07:00"
                  end: "2026-10-23T22:58:54-07:00"
        queue_waits:
            - attempt_id: afbb4afad3d6f65e1ebf95a33e7161a4
              slot: pair:6
              record_digest: 0c6a6398072130c403f211f240473519e7846253797c13190b64916707509da3
              start: "2026-10-24T04:41:28.474788Z"
              end: "2026-10-24T04:41:41.984802Z"
              queue_seconds: 13.510014
              raw_seconds: 13.510014
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.07971666666666667
        cuts:
            - dispatched: "2026-10-23T22:51:57-07:00"
              finalized: "2026-10-23T22:55:59-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-23T23:03:34-07:00"
      dispatched: "2026-10-23T22:59:35-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: model.go:133-134 now describes the press selection (offscreen included, 8 oldest, only a note being written refuses) and replay-rendered from the newest finish, which matches cast_submission.go:380-428; model_test.go and sketch_page_test.go pin the new terms and pass.
          round: 2
        - id: BR-2
          disposition: addressed
          note: SKILL.md:721-725,757-760,775-783 and README.md:1549 now state the press-time contract; a grep finds no "actually visible", "stale visible selection" or "current recorded prefix" left outside the legacy-bundle notes; skill_privacy_test pins the new phrases and passes.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Done-when bullet 2 and the Plan now say "a new note not saved, or an edit whose text changed and is not saved", with a 2026-10-23 close-review Revisions entry, matching collab.js noteDialog.unsaved.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The Plan and Revisions now say the observer dedupe is tested live only; the existing vm harness (observer-sketch-exclusion) stubs no DOM, so the plan's claim now matches the code.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: No dispute or host evidence recorded; WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive still fails at Chrome launch with "GPU process isn't usable". A host run of these tests plus TestOperatorFeedbackLivePrepareAndStalePanel, ideally failing on base 4ffd05c5, would answer it.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: b858551ce706ac16f45d3e96069847db5c33e5c2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1b66477f-a4fc-4491-8f96-e0ba34461036
            repository: github.com/xianxu/pair
            issue_id: "000849"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/4159f53a3b2c260e
            reviewed_head: f18b57703a6a24792522ce52633687bcc97f7a04
            slot: pair:6
            requested_at: "2026-10-24T06:21:38.445141Z"
          phase: finished
          pid: 2954
          started_at: "2026-10-24T06:21:38.660887Z"
          finished_at: "2026-10-24T06:25:14.661083Z"
          session_id: 2a5927e0-06bb-44a5-91e1-e1368d7da0b1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: f18b57703a6a24792522ce52633687bcc97f7a04
        tracker_head: feca0002c2fe9cbb2366232775ca69f712cd160e
        branch_point: 4ffd05c54f8a7fbcf32f13bb95952b0d637cb59a
        since: "2026-10-23T17:38:50-07:00"
        until: "2026-10-23T23:07:02-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:38:50-07:00"
              end: "2026-10-23T23:07:02-07:00"
              began: "2026-10-23T17:38:50-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: claude
              id: 5ba7d27e-c4f7-4a9b-9e2c-f7ac71ef7e9a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: b8ed590b6382413dcb560006ed2a09f4a37c9305822ee84b71eed8ae3add136b
              started: "2026-10-23T07:52:37.426Z"
              spawned: false
              bytes: 21261311
              windows:
                - start: "2026-10-23T17:38:50-07:00"
                  end: "2026-10-23T23:07:02-07:00"
        queue_waits:
            - attempt_id: afbb4afad3d6f65e1ebf95a33e7161a4
              slot: pair:6
              record_digest: 0c6a6398072130c403f211f240473519e7846253797c13190b64916707509da3
              start: "2026-10-24T04:41:28.474788Z"
              end: "2026-10-24T04:41:41.984802Z"
              queue_seconds: 13.510014
              raw_seconds: 13.510014
              exact: true
            - attempt_id: 5462e9c7d6af6c866332ff73813bae49
              slot: pair:6
              record_digest: 546103fa54c51b19b0c78cb1ce786967a88750ed571b51bc3f1042ab09e248cd
              start: "2026-10-24T06:05:18.893808Z"
              end: "2026-10-24T06:05:47.743653Z"
              queue_seconds: 28.849845
              raw_seconds: 28.849845
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.07971666666666667
        cuts:
            - dispatched: "2026-10-23T22:51:57-07:00"
              finalized: "2026-10-23T22:55:59-07:00"
              finalizing: false
            - dispatched: "2026-10-23T22:59:35-07:00"
              finalized: "2026-10-23T23:03:34-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-23T23:25:22-07:00"
      dispatched: "2026-10-23T23:21:34-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: Still unverifiable here. Re-ran WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive at f18b57703 and Chrome exited at launch ("GPU process isn't usable", exit_code=6). No dispute with host evidence was filed. The Log claims a host run but gives no captured output. Minor, does not block.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: f18b57703a6a24792522ce52633687bcc97f7a04
      blocked: false
---

# Gate ledger — pair#849 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T22:55:59-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `contract-consumers-not-swept` (kind: defect) Agent ReadingGuide still describes the visible-only gated contract and replay-rendered-current-view
  cmd/internal/submission/model.go:133-134 (stored in every bundle, sent to the agent) says only console-visible feedback is selected, a stale visible selection refuses, and new sketch_page uses replay-rendered-current-view with sketch_page.current; after this change presses emit replay-rendered with no current (cast_submission.go:424-428) and include offscreen feedback. Rewrite to match atlas/webcast-submission.md and update model_test.go.
- **BR-2** [Important] `contract-consumers-not-swept` (kind: defect) couch SKILL.md and README.md still describe the old visible-selection submission contract
  cmd/internal/couchcmd/skills/couch/SKILL.md:721,758,776-779 and README.md:1549 claim visible-only eligibility, stale-selection refusal and replay-rendered-current-view provenance; sweep them alongside the ReadingGuide fix.
- **BR-3** [Minor] `plan-claims-match-code` (kind: defect) Unsaved-note refusal is narrower than Done-when/Plan (untouched open edit does not refuse); no Revisions entry
  Plan says an open edit refuses with "note N is being edited"; collab.js unsaved refuses only a changed edit or a draft. The choice is in the Log but the Done-when and Plan were not revised.
- **BR-4** [Minor] `plan-claims-match-code` (kind: defect) Observer selection dedupe has no non-live unit test despite the Plan claiming one
  Only the WEBCAST_LIVE_CHROME-gated TestSubmissionManagerLiveSubmitsOnAChangingPage counts selection records; the lastSel reset on resync is untested. A vm-based test like tests/webcast/observer-sketch-exclusion.test.mjs could cover it.
- **BR-5** [Minor] `live-chrome-evidence` (kind: unverifiable) Live Chrome tests (50 ms changing page, unsaved draft CDP eval) could not run in the review sandbox
  Ran WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive; Chrome exited at launch with "GPU process isn't usable". A host run of these tests plus cmd/internal/webcast TestOperatorFeedbackLivePrepareAndStalePanel, ideally also failing on base 4ffd05c5, would answer it.

## Round 2 — 2026-10-23T23:03:34-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — model.go:133-134 now describes the press selection (offscreen included, 8 oldest, only a note being written refuses) and replay-rendered from the newest finish, which matches cast_submission.go:380-428; model_test.go and sketch_page_test.go pin the new terms and pass.
- BR-2 — addressed — SKILL.md:721-725,757-760,775-783 and README.md:1549 now state the press-time contract; a grep finds no "actually visible", "stale visible selection" or "current recorded prefix" left outside the legacy-bundle notes; skill_privacy_test pins the new phrases and passes.
- BR-3 — addressed — Done-when bullet 2 and the Plan now say "a new note not saved, or an edit whose text changed and is not saved", with a 2026-10-23 close-review Revisions entry, matching collab.js noteDialog.unsaved.
- BR-4 — addressed — The Plan and Revisions now say the observer dedupe is tested live only; the existing vm harness (observer-sketch-exclusion) stubs no DOM, so the plan's claim now matches the code.
- BR-5 — not-addressed — No dispute or host evidence recorded; WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive still fails at Chrome launch with "GPU process isn't usable". A host run of these tests plus TestOperatorFeedbackLivePrepareAndStalePanel, ideally failing on base 4ffd05c5, would answer it.

## Round 3 — 2026-10-23T23:25:22-07:00 (claude) — passed

### Disposed

- BR-5 — not-addressed — Still unverifiable here. Re-ran WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive at f18b57703 and Chrome exited at launch ("GPU process isn't usable", exit_code=6). No dispute with host evidence was filed. The Log claims a host run but gives no captured output. Minor, does not block.

## Open findings

- **BR-5** [Minor] `live-chrome-evidence` (kind: unverifiable) Live Chrome tests (50 ms changing page, unsaved draft CDP eval) could not run in the review sandbox
