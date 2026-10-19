---
gate: boundary-review
issue: 673
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1cc615ce-ec28-47c9-8fe7-ec51e72e9234
            repository: github.com/xianxu/pair
            issue_id: "000673"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/4ff83248dbce0d6a
            reviewed_head: 83d74e96af019f3b425b9fd2b22baeb15b0f7284
            slot: pair:2
            requested_at: "2026-10-19T04:12:18.388118Z"
          phase: finished
          pid: 5805
          started_at: "2026-10-19T04:12:18.76743Z"
          finished_at: "2026-10-19T04:15:52.783018Z"
          session_id: 1c26cb79-1054-47af-931c-34185c2ba36f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 83d74e96af019f3b425b9fd2b22baeb15b0f7284
        tracker_head: 90289f85f39d5829e356bc0fb3ebefc1f32085f5
        branch_point: 74fa58d2c4319cec875ecf227159c95ac779b365
        since: "2026-10-18T12:05:17-07:00"
        until: "2026-10-18T20:28:29-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T12:05:17-07:00"
              end: "2026-10-18T20:28:29-07:00"
              began: "2026-10-18T12:05:17-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: e18a92eb10a8e1372c764f01b00a05902a9e8a2329b778025369be58e8d04546
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 39704965
              windows:
                - start: "2026-10-18T12:05:17-07:00"
                  end: "2026-10-18T20:28:29-07:00"
        queue_waits:
            - attempt_id: d1b55d9960a4d972f112135140da794e
              slot: pair:2
              record_digest: 18f003017b032f930a47a0d9a29ede42084f0603edcd1280710f5539b697bd58
              start: "2026-10-18T19:34:38.24251Z"
              end: "2026-10-18T20:27:06.633442Z"
              queue_seconds: 3148.390932
              raw_seconds: 3148.390932
              exact: true
            - attempt_id: 5bf0374293571563365a48c27b7f733c
              slot: pair:2
              record_digest: 7f9d61e5017020693342c67f7409e1ae822c273a9520ddd2d32e1a26293283db
              start: "2026-10-18T22:37:56.901724Z"
              end: "2026-10-18T22:55:05.772147Z"
              queue_seconds: 1028.870423
              raw_seconds: 1028.870423
              exact: true
            - attempt_id: d67579650589c0818fa1fec023f084a7
              slot: pair:2
              record_digest: c05446992d6d80df8ae5a57e2e7b8dd393c3514011163f6d6e6d99c9825d42ed
              start: "2026-10-19T00:51:54.781877Z"
              end: "2026-10-19T01:02:07.781488Z"
              queue_seconds: 612.999611
              raw_seconds: 612.999611
              exact: true
            - attempt_id: 17e2d970d54b2f4458a8bf0ddffe48ec
              slot: pair:2
              record_digest: 64c92014da13bb36471a2afdffcd1925225d586e2d4bfd318f94df35a8697de9
              start: "2026-10-19T01:40:52.312221Z"
              end: "2026-10-19T01:52:59.129222Z"
              queue_seconds: 726.817001
              raw_seconds: 726.817001
              exact: true
            - attempt_id: af5ab027615fcdeae44cf1f1e6df5307
              slot: pair:2
              record_digest: b03accd4d09f78ac23652720f9039730e13dc8eb3ea6658806c92c75c7094727
              start: "2026-10-19T02:24:51.05315Z"
              end: "2026-10-19T03:03:11.064246Z"
              queue_seconds: 2300.011096
              raw_seconds: 2300.011096
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
      "n": 1
      timestamp: "2026-10-18T21:15:54-07:00"
      dispatched: "2026-10-18T21:12:14-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Core concepts table contradicts the shipped code (paths, API names, fields, restore behavior)
          detail: 'AttentionRows is listed at couchtty/attention_panel.go, which does not exist; the code is attentionRows in console_attention.go. topRightBox is really renderMenuAt(top). Raise/ConditionSet/Reconcile are really Notify/Conditions/RestoreAttention. The item field Slot is really Thread plus Label. The plan says loaded items show immediately; the code hides restored condition items for a 2-minute grace. Delete vs Backspace. Fix: add a plan Revisions entry.'
          family: plan-table-matches-code
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Attention store's 256 KiB load/save cap has no matching bound on the writer
          detail: 'There are 50 notify items and each body can be 4096 bytes. json.Marshal escapes each < > & to 6 bytes, so about 11 maximum-size bodies exceed AttentionStoreLimit. After that every Save is refused (a notice on every change) and nothing persists until the operator dismisses items. Fix: evict the oldest notifications until the JSON fits, or truncate fields when items are created, and turn off HTML escaping; add a test at the cap.'
          family: writer-bound-matches-reader-cap
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: No tests for the notify and prompt sources (handleNotify, publishBlocked to observeAttention)
          detail: 'The Console tests inject facts directly, so deleting the observeAttention call in notify_service.go or message_service.go keeps every test green. Add service-rig tests: an accepted hub notify emits one fact with the right thread and label, a refused or non-hub request emits none, and a change in the picker set emits a prompt fact.'
          family: source-hook-regression-test
          round: 1
        - kind: defect
          id: BR-4
          severity: Important
          title: Planned pure tests for row formatting and top-right box geometry are missing
          detail: The plan's test surface lists AttentionRows formatting and selection, and topRightBox geometry at small and odd sizes. Neither exists; attentionRows and renderMenuAt(top=true) are only exercised through the Console fixture.
          family: planned-pure-tests-delivered
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: Clicking the status-row badge while the panel is open reopens it instead of closing it
          detail: routeAttentionPanel closes the panel on a click outside the box and returns false, then the status-row handler calls toggleAttention, which opens it again.
          family: click-outside-then-toggle
          round: 1
        - kind: defect
          id: BR-6
          severity: Minor
          title: ObserveAttention does a synchronous fsync'd save on message-service goroutines despite the must-not-block contract
          detail: publishBlocked calls it while holding s.blocked, and handleNotify calls it before the OSC write.
          family: observer-must-not-block
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: fc262efe-3923-44be-bc8b-438ad57553d5
            repository: github.com/xianxu/pair
            issue_id: "000673"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/74d1f79af281067a
            reviewed_head: 9e0df2abd3ad43b083e7236710ebffb4b64bfcc4
            slot: pair:2
            requested_at: "2026-10-19T04:35:44.402246Z"
          phase: finished
          pid: 67016
          started_at: "2026-10-19T04:35:44.588784Z"
          finished_at: "2026-10-19T04:59:53.612764Z"
          session_id: 433b5e2e-b61c-4617-808b-f9639cb3e99a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 9e0df2abd3ad43b083e7236710ebffb4b64bfcc4
        tracker_head: a46cbc5b5569913aac51dd924df2bb54df032b63
        branch_point: 74fa58d2c4319cec875ecf227159c95ac779b365
        since: "2026-10-18T12:05:17-07:00"
        until: "2026-10-18T21:35:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T12:05:17-07:00"
              end: "2026-10-18T21:35:04-07:00"
              began: "2026-10-18T12:05:17-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cbc83b23433f1e10a450423fa5c4279a9bec74cb8e3e1289e848e7f1c2706093
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 40514128
              windows:
                - start: "2026-10-18T12:05:17-07:00"
                  end: "2026-10-18T21:35:04-07:00"
        queue_waits:
            - attempt_id: d1b55d9960a4d972f112135140da794e
              slot: pair:2
              record_digest: 18f003017b032f930a47a0d9a29ede42084f0603edcd1280710f5539b697bd58
              start: "2026-10-18T19:34:38.24251Z"
              end: "2026-10-18T20:27:06.633442Z"
              queue_seconds: 3148.390932
              raw_seconds: 3148.390932
              exact: true
            - attempt_id: 5bf0374293571563365a48c27b7f733c
              slot: pair:2
              record_digest: 7f9d61e5017020693342c67f7409e1ae822c273a9520ddd2d32e1a26293283db
              start: "2026-10-18T22:37:56.901724Z"
              end: "2026-10-18T22:55:05.772147Z"
              queue_seconds: 1028.870423
              raw_seconds: 1028.870423
              exact: true
            - attempt_id: d67579650589c0818fa1fec023f084a7
              slot: pair:2
              record_digest: c05446992d6d80df8ae5a57e2e7b8dd393c3514011163f6d6e6d99c9825d42ed
              start: "2026-10-19T00:51:54.781877Z"
              end: "2026-10-19T01:02:07.781488Z"
              queue_seconds: 612.999611
              raw_seconds: 612.999611
              exact: true
            - attempt_id: 17e2d970d54b2f4458a8bf0ddffe48ec
              slot: pair:2
              record_digest: 64c92014da13bb36471a2afdffcd1925225d586e2d4bfd318f94df35a8697de9
              start: "2026-10-19T01:40:52.312221Z"
              end: "2026-10-19T01:52:59.129222Z"
              queue_seconds: 726.817001
              raw_seconds: 726.817001
              exact: true
            - attempt_id: af5ab027615fcdeae44cf1f1e6df5307
              slot: pair:2
              record_digest: b03accd4d09f78ac23652720f9039730e13dc8eb3ea6658806c92c75c7094727
              start: "2026-10-19T02:24:51.05315Z"
              end: "2026-10-19T03:03:11.064246Z"
              queue_seconds: 2300.011096
              raw_seconds: 2300.011096
              exact: true
            - attempt_id: 587d480cc1f9ff8ed149856894dbc1e9
              slot: pair:2
              record_digest: 1d5a016716e12deec9365df4aa7bbfd1ab6fe3accad6bb69c561bde49354b7cb
              start: "2026-10-19T03:28:42.630386Z"
              end: "2026-10-19T03:53:35.009233Z"
              queue_seconds: 1492.378847
              raw_seconds: 1492.378847
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
        cuts:
            - dispatched: "2026-10-18T21:12:14-07:00"
              finalized: "2026-10-18T21:15:54-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T21:59:54-07:00"
      dispatched: "2026-10-18T21:35:41-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan Revisions entry dated 2026-10-18 maps every name, path, identity, restore-grace and Backspace difference to the shipped code; spot-checked against console_attention.go and attention_list.go.
          round: 2
        - id: BR-2
          disposition: addressed
          note: capped() now evicts the oldest notify items until encodeAttention fits; with the fits() term reverted, TestAttentionListStaysWithinTheStoreLimit fails (275220 > 262144).
          round: 2
        - id: BR-3
          disposition: addressed
          note: With the observeAttention call removed, TestNotifyRaisesOneAttentionItem fails (handleNotify) and TestBlockedPromptsReachTheAttentionList fails (publishBlocked); the refused worker case is covered.
          round: 2
        - id: BR-4
          disposition: addressed
          note: TestAttentionRowsFormat and TestAttentionBoxTopRightGeometry exercise attentionRows and renderMenuAt(top=true) directly, without IO, at several sizes.
          round: 2
        - id: BR-5
          disposition: addressed
          note: With the onBadge return reverted, TestAttentionMouseAndDismiss times out waiting for the badge click to close the panel; with the fix it passes in 0.4s.
          round: 2
        - id: BR-6
          disposition: addressed
          note: Saves run in one coalescing background saver; TestAttentionObserveNeverWaitsOnTheStore uses a gated store and would block if the save were synchronous.
          round: 2
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: Attention saver goroutine outlives the Console; teardown never waits for it, so the last change before quit can be lost
          detail: 'console_attention.go:94 starts runAttentionSaves with go, and Run/teardown (console.go:784) never waits for it (ARCH-ORDER extent). Dismissing or picking an item, or a notification arriving, right before exit can leave list.json one change behind; the synchronous save it replaced did not. Fix: track the saver (WaitGroup or done channel) and have teardown wait for it with a bounded timeout; add a test that dismisses, stops the console, and asserts the store is saved without waitFor.'
          family: spawned-work-bounded-extent
          round: 2
        - kind: defect
          id: BR-8
          severity: Minor
          title: Garbled comment line in TestAttentionMouseAndDismiss
          detail: console_attention_test.go:157 reads "// <tab>// The decoder hands..."; a doubled comment marker left over from an edit.
          family: test-comment-hygiene
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 788e3991-7db3-446e-82e1-210905069398
            repository: github.com/xianxu/pair
            issue_id: "000673"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ea8886387138637a
            reviewed_head: 6d00d20b44696671955e7d550d02c143c8de1e27
            slot: pair:2
            requested_at: "2026-10-19T05:05:06.400211Z"
          phase: finished
          pid: 27829
          started_at: "2026-10-19T05:05:06.504833Z"
          finished_at: "2026-10-19T05:24:26.359964Z"
          session_id: e8a26a33-af3b-4ecd-a58f-17c40c12d5d1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 6d00d20b44696671955e7d550d02c143c8de1e27
        tracker_head: 2d9664714805d512c034a15bfd2368aa92ab8029
        branch_point: 74fa58d2c4319cec875ecf227159c95ac779b365
        since: "2026-10-18T12:05:17-07:00"
        until: "2026-10-18T22:04:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T12:05:17-07:00"
              end: "2026-10-18T22:04:45-07:00"
              began: "2026-10-18T12:05:17-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 21827cb37cbfe84c719c3f392c1b878cdeefa8b02ad3ec087ada76571ee20331
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 40700726
              windows:
                - start: "2026-10-18T12:05:17-07:00"
                  end: "2026-10-18T22:04:45-07:00"
        queue_waits:
            - attempt_id: d1b55d9960a4d972f112135140da794e
              slot: pair:2
              record_digest: 18f003017b032f930a47a0d9a29ede42084f0603edcd1280710f5539b697bd58
              start: "2026-10-18T19:34:38.24251Z"
              end: "2026-10-18T20:27:06.633442Z"
              queue_seconds: 3148.390932
              raw_seconds: 3148.390932
              exact: true
            - attempt_id: 5bf0374293571563365a48c27b7f733c
              slot: pair:2
              record_digest: 7f9d61e5017020693342c67f7409e1ae822c273a9520ddd2d32e1a26293283db
              start: "2026-10-18T22:37:56.901724Z"
              end: "2026-10-18T22:55:05.772147Z"
              queue_seconds: 1028.870423
              raw_seconds: 1028.870423
              exact: true
            - attempt_id: d67579650589c0818fa1fec023f084a7
              slot: pair:2
              record_digest: c05446992d6d80df8ae5a57e2e7b8dd393c3514011163f6d6e6d99c9825d42ed
              start: "2026-10-19T00:51:54.781877Z"
              end: "2026-10-19T01:02:07.781488Z"
              queue_seconds: 612.999611
              raw_seconds: 612.999611
              exact: true
            - attempt_id: 17e2d970d54b2f4458a8bf0ddffe48ec
              slot: pair:2
              record_digest: 64c92014da13bb36471a2afdffcd1925225d586e2d4bfd318f94df35a8697de9
              start: "2026-10-19T01:40:52.312221Z"
              end: "2026-10-19T01:52:59.129222Z"
              queue_seconds: 726.817001
              raw_seconds: 726.817001
              exact: true
            - attempt_id: af5ab027615fcdeae44cf1f1e6df5307
              slot: pair:2
              record_digest: b03accd4d09f78ac23652720f9039730e13dc8eb3ea6658806c92c75c7094727
              start: "2026-10-19T02:24:51.05315Z"
              end: "2026-10-19T03:03:11.064246Z"
              queue_seconds: 2300.011096
              raw_seconds: 2300.011096
              exact: true
            - attempt_id: 587d480cc1f9ff8ed149856894dbc1e9
              slot: pair:2
              record_digest: 1d5a016716e12deec9365df4aa7bbfd1ab6fe3accad6bb69c561bde49354b7cb
              start: "2026-10-19T03:28:42.630386Z"
              end: "2026-10-19T03:53:35.009233Z"
              queue_seconds: 1492.378847
              raw_seconds: 1492.378847
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
        cuts:
            - dispatched: "2026-10-18T21:12:14-07:00"
              finalized: "2026-10-18T21:15:54-07:00"
              finalizing: false
            - dispatched: "2026-10-18T21:35:41-07:00"
              finalized: "2026-10-18T21:59:54-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-18T22:24:27-07:00"
      dispatched: "2026-10-18T22:05:04-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-7
          disposition: addressed
          note: saveAttention starts the saver via c.GoTracked (console_attention.go:95-97), which teardown joins at console.go:1162; TestAttentionShutdownWaitsForTheLastSave passes at head and fails when the prior go c.runAttentionSaves() is restored in this checkout.
          round: 3
        - id: BR-8
          disposition: addressed
          note: console_attention_test.go:157-158 now reads as one clean comment; the doubled marker is removed in 6d00d20b4.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 6d00d20b44696671955e7d550d02c143c8de1e27
      blocked: false
---

# Gate ledger — pair#673 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T21:15:54-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `plan-table-matches-code` (kind: defect) Core concepts table contradicts the shipped code (paths, API names, fields, restore behavior)
  AttentionRows is listed at couchtty/attention_panel.go, which does not exist; the code is attentionRows in console_attention.go. topRightBox is really renderMenuAt(top). Raise/ConditionSet/Reconcile are really Notify/Conditions/RestoreAttention. The item field Slot is really Thread plus Label. The plan says loaded items show immediately; the code hides restored condition items for a 2-minute grace. Delete vs Backspace. Fix: add a plan Revisions entry.
- **BR-2** [Important] `writer-bound-matches-reader-cap` (kind: defect) Attention store's 256 KiB load/save cap has no matching bound on the writer
  There are 50 notify items and each body can be 4096 bytes. json.Marshal escapes each < > & to 6 bytes, so about 11 maximum-size bodies exceed AttentionStoreLimit. After that every Save is refused (a notice on every change) and nothing persists until the operator dismisses items. Fix: evict the oldest notifications until the JSON fits, or truncate fields when items are created, and turn off HTML escaping; add a test at the cap.
- **BR-3** [Important] `source-hook-regression-test` (kind: defect) No tests for the notify and prompt sources (handleNotify, publishBlocked to observeAttention)
  The Console tests inject facts directly, so deleting the observeAttention call in notify_service.go or message_service.go keeps every test green. Add service-rig tests: an accepted hub notify emits one fact with the right thread and label, a refused or non-hub request emits none, and a change in the picker set emits a prompt fact.
- **BR-4** [Important] `planned-pure-tests-delivered` (kind: defect) Planned pure tests for row formatting and top-right box geometry are missing
  The plan's test surface lists AttentionRows formatting and selection, and topRightBox geometry at small and odd sizes. Neither exists; attentionRows and renderMenuAt(top=true) are only exercised through the Console fixture.
- **BR-5** [Minor] `click-outside-then-toggle` (kind: defect) Clicking the status-row badge while the panel is open reopens it instead of closing it
  routeAttentionPanel closes the panel on a click outside the box and returns false, then the status-row handler calls toggleAttention, which opens it again.
- **BR-6** [Minor] `observer-must-not-block` (kind: defect) ObserveAttention does a synchronous fsync'd save on message-service goroutines despite the must-not-block contract
  publishBlocked calls it while holding s.blocked, and handleNotify calls it before the OSC write.

## Round 2 — 2026-10-18T21:59:54-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Plan Revisions entry dated 2026-10-18 maps every name, path, identity, restore-grace and Backspace difference to the shipped code; spot-checked against console_attention.go and attention_list.go.
- BR-2 — addressed — capped() now evicts the oldest notify items until encodeAttention fits; with the fits() term reverted, TestAttentionListStaysWithinTheStoreLimit fails (275220 > 262144).
- BR-3 — addressed — With the observeAttention call removed, TestNotifyRaisesOneAttentionItem fails (handleNotify) and TestBlockedPromptsReachTheAttentionList fails (publishBlocked); the refused worker case is covered.
- BR-4 — addressed — TestAttentionRowsFormat and TestAttentionBoxTopRightGeometry exercise attentionRows and renderMenuAt(top=true) directly, without IO, at several sizes.
- BR-5 — addressed — With the onBadge return reverted, TestAttentionMouseAndDismiss times out waiting for the badge click to close the panel; with the fix it passes in 0.4s.
- BR-6 — addressed — Saves run in one coalescing background saver; TestAttentionObserveNeverWaitsOnTheStore uses a gated store and would block if the save were synchronous.

### Raised

- **BR-7** [Important] `spawned-work-bounded-extent` (kind: defect) Attention saver goroutine outlives the Console; teardown never waits for it, so the last change before quit can be lost
  console_attention.go:94 starts runAttentionSaves with go, and Run/teardown (console.go:784) never waits for it (ARCH-ORDER extent). Dismissing or picking an item, or a notification arriving, right before exit can leave list.json one change behind; the synchronous save it replaced did not. Fix: track the saver (WaitGroup or done channel) and have teardown wait for it with a bounded timeout; add a test that dismisses, stops the console, and asserts the store is saved without waitFor.
- **BR-8** [Minor] `test-comment-hygiene` (kind: defect) Garbled comment line in TestAttentionMouseAndDismiss
  console_attention_test.go:157 reads "// <tab>// The decoder hands..."; a doubled comment marker left over from an edit.

## Round 3 — 2026-10-18T22:24:27-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — saveAttention starts the saver via c.GoTracked (console_attention.go:95-97), which teardown joins at console.go:1162; TestAttentionShutdownWaitsForTheLastSave passes at head and fails when the prior go c.runAttentionSaves() is restored in this checkout.
- BR-8 — addressed — console_attention_test.go:157-158 now reads as one clean comment; the doubled marker is removed in 6d00d20b4.

## Open findings

(none — every finding has been disposed)
