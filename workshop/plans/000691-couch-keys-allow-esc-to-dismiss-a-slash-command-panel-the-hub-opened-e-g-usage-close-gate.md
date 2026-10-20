---
gate: boundary-review
issue: 691
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 70517577-9cda-4b46-8adc-dffadf5efacf
            repository: github.com/xianxu/pair
            issue_id: "000691"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/eb3d87f963a1b076
            reviewed_head: 28bd9ba860eb03dc5f5c0aeb7f6cf97a66b1d0f3
            slot: pair:2
            requested_at: "2026-10-20T01:09:53.516252Z"
          phase: finished
          pid: 50597
          started_at: "2026-10-20T01:09:53.702198Z"
          finished_at: "2026-10-20T01:13:16.436478Z"
          session_id: 4dfa281b-a278-4e52-ad6b-990aa1f7ed4c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 28bd9ba860eb03dc5f5c0aeb7f6cf97a66b1d0f3
        tracker_head: 91adacc6663ccab0b0da2171771c60d0f47b44ad
        branch_point: 3dd13c0ea80aa05d1aff1ec3eb25437a5f082ddb
        since: "2026-10-19T10:56:26-07:00"
        until: "2026-10-19T17:51:43-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T10:56:26-07:00"
              end: "2026-10-19T17:51:43-07:00"
              began: "2026-10-19T10:56:26-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4358490350ec9bc40b79ecb350cca995f3dc71048bebcd0be2d9b2759046be6c
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 48898128
              windows:
                - start: "2026-10-19T10:56:26-07:00"
                  end: "2026-10-19T17:51:43-07:00"
        queue_waits:
            - attempt_id: cd4ee0b94dcaed3f18e219054115a78f
              slot: pair:2
              record_digest: da028b1b028b59f89850af753ac08c92d8920e29802c3d6337d7f993d29b95fb
              start: "2026-10-20T00:04:54.726784Z"
              end: "2026-10-20T00:42:58.023022Z"
              queue_seconds: 2283.296238
              raw_seconds: 2283.296238
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.7426333333333334
      "n": 1
      timestamp: "2026-10-19T18:13:17-07:00"
      dispatched: "2026-10-19T18:09:50-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Done-when live check (clear-line text:/usage submit, then esc, on a live Claude slot) is not delivered at whole-issue close
          detail: Done-when bullet 1 requires a recognizer fixture plus a live check; the Plan item "Live check after landing" is still unchecked and the Log records only a replay of the old build. Record the hub's live run on this build in the Log, or revise Done-when so the live check is explicitly a post-merge step.
          family: done-when-evidence-missing
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 41e1cd41-d451-4353-a89a-891156aff7b8
            repository: github.com/xianxu/pair
            issue_id: "000691"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/54fc0b5a3a246cb2
            reviewed_head: 4675381be1ac22635fecf3523b303c7a15f6f433
            slot: pair:2
            requested_at: "2026-10-20T01:14:42.068531Z"
          phase: finished
          pid: 72874
          started_at: "2026-10-20T01:14:42.236741Z"
          finished_at: "2026-10-20T01:19:51.569289Z"
          session_id: 20a435e9-cd96-4b07-98d6-8f9b912dc76d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 4675381be1ac22635fecf3523b303c7a15f6f433
        tracker_head: 91adacc6663ccab0b0da2171771c60d0f47b44ad
        branch_point: 3dd13c0ea80aa05d1aff1ec3eb25437a5f082ddb
        since: "2026-10-19T10:56:26-07:00"
        until: "2026-10-19T18:13:52-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T10:56:26-07:00"
              end: "2026-10-19T18:13:52-07:00"
              began: "2026-10-19T10:56:26-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a56d473d7f94ef5cb3a409129c370968ac911c6dc63ef49ff760f0cbcb365673
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 48984687
              windows:
                - start: "2026-10-19T10:56:26-07:00"
                  end: "2026-10-19T18:13:52-07:00"
        queue_waits:
            - attempt_id: cd4ee0b94dcaed3f18e219054115a78f
              slot: pair:2
              record_digest: da028b1b028b59f89850af753ac08c92d8920e29802c3d6337d7f993d29b95fb
              start: "2026-10-20T00:04:54.726784Z"
              end: "2026-10-20T00:42:58.023022Z"
              queue_seconds: 2283.296238
              raw_seconds: 2283.296238
              exact: true
            - attempt_id: ce1c1ce18b53f9e7939df11f1e5c5308
              slot: pair:2
              record_digest: 9632d6fb691d4dc5b2aa94316ce97ed3758e94c0397155b1d156f761d3d798c8
              start: "2026-10-20T00:51:50.252789Z"
              end: "2026-10-20T00:59:53.324764Z"
              queue_seconds: 483.071975
              raw_seconds: 483.071975
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.593
        cuts:
            - dispatched: "2026-10-19T18:09:50-07:00"
              finalized: "2026-10-19T18:13:17-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-19T18:19:53-07:00"
      dispatched: "2026-10-19T18:14:38-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Revisions 2026-10-19 (BR-1 entry) makes the live check an explicit post-merge hub step (a failure reopens the issue); the close is gated on the 2.1.304 fixture tests, which I re-ran green with the panel term mutation going red.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Minor
          title: Panel refusals still say "composer must be empty" rather than naming esc as the only admitted key
          detail: agent_drive.go:641 and model.go:341 give the same message for a non-esc key on a recognized panel; a panel-specific hint would point the hub at `--keys SLOT esc`.
          family: refusal-guidance-wording
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 4675381be1ac22635fecf3523b303c7a15f6f433
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 965a6ab7-785a-49c1-90e3-271c5790cb9b
            repository: github.com/xianxu/pair
            issue_id: "000691"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/16a4005f15d7a965
            reviewed_head: 0127de1adbef4af2f798ff5a796d05ab578d8f62
            slot: pair:2
            requested_at: "2026-10-20T06:41:47.822109Z"
          phase: finished
          pid: 22170
          started_at: "2026-10-20T06:41:48.100993Z"
          finished_at: "2026-10-20T06:45:35.779041Z"
          session_id: e75a8ce4-304d-441e-aed7-d2d5b73c10ab
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 0127de1adbef4af2f798ff5a796d05ab578d8f62
        tracker_head: 35d731a35e0c151e392d510e0f2e6eff7f012887
        branch_point: b98802e09dfa33a77fd723d34948d247a5dd2e34
        since: "2026-10-19T10:56:26-07:00"
        until: "2026-10-19T23:18:49-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T10:56:26-07:00"
              end: "2026-10-19T23:18:49-07:00"
              began: "2026-10-19T10:56:26-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 7d21d57c-2837-40a9-8da0-3ab6255112f6
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 477aef09988094febffbe0b4369664920b80265d3427398be0025a04b5d530c5
              started: "2026-10-20T06:17:53.348Z"
              spawned: false
              bytes: 529123
              windows:
                - start: "2026-10-20T06:17:53.348Z"
                  end: "2026-10-19T23:18:49-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 3c997ff236f29c4775e761026e89370abf696c707e405eea546c638d334a1d0f
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 51527554
              windows:
                - start: "2026-10-19T10:56:26-07:00"
                  end: "2026-10-19T23:18:49-07:00"
        queue_waits:
            - attempt_id: cd4ee0b94dcaed3f18e219054115a78f
              slot: pair:2
              record_digest: da028b1b028b59f89850af753ac08c92d8920e29802c3d6337d7f993d29b95fb
              start: "2026-10-20T00:04:54.726784Z"
              end: "2026-10-20T00:42:58.023022Z"
              queue_seconds: 2283.296238
              raw_seconds: 2283.296238
              exact: true
            - attempt_id: ce1c1ce18b53f9e7939df11f1e5c5308
              slot: pair:2
              record_digest: 9632d6fb691d4dc5b2aa94316ce97ed3758e94c0397155b1d156f761d3d798c8
              start: "2026-10-20T00:51:50.252789Z"
              end: "2026-10-20T00:59:53.324764Z"
              queue_seconds: 483.071975
              raw_seconds: 483.071975
              exact: true
            - attempt_id: 2924851d5401f4a3e7419c21b7a261c7
              slot: pair:2
              record_digest: 1d16d8da4a882acacc9ef789a89832f94c3fa8ab6f4185561ef33db96231b4d3
              start: "2026-10-20T01:33:24.999426Z"
              end: "2026-10-20T02:00:01.699183Z"
              queue_seconds: 1596.699757
              raw_seconds: 1596.699757
              exact: true
            - attempt_id: 0bd80ef28c69da0bddfae4d53277a8fb
              slot: pair:2
              record_digest: 7449e5fa7be0d5ad88e3a93e9c3fb18b55f6e7b624dda0da50e9ee6af7a04710
              start: "2026-10-20T02:23:16.713835Z"
              end: "2026-10-20T02:52:05.489712Z"
              queue_seconds: 1728.775877
              raw_seconds: 1728.775877
              exact: true
            - attempt_id: 59087d99b62331e30d21a91c60bf6211
              slot: pair:2
              record_digest: bfaa9c8b9e977da38866f00e828ab87a63623d1d972cf3284d90252f0700fbc0
              start: "2026-10-20T03:11:11.77043Z"
              end: "2026-10-20T03:17:45.145594Z"
              queue_seconds: 393.375164
              raw_seconds: 393.375164
              exact: true
            - attempt_id: a61f0200798345d1cb12419146c00045
              slot: pair:2
              record_digest: a9954a2b36f8eeaf3a0c6f64bd9a2528b21f799d1c6eef1290577f19679cd10d
              start: "2026-10-20T03:28:55.010894Z"
              end: "2026-10-20T04:08:23.491694Z"
              queue_seconds: 2368.4808
              raw_seconds: 2368.4808
              exact: true
            - attempt_id: b2728941baf6ce9b02474a500271d4d8
              slot: pair:2
              record_digest: dacb0f5882c7ff0b8b734e6fa7cb1c118046d30e755769aa1623a65165127e3c
              start: "2026-10-20T04:20:01.74038Z"
              end: "2026-10-20T04:48:43.275022Z"
              queue_seconds: 1721.534642
              raw_seconds: 1721.534642
              exact: true
            - attempt_id: d9aeec08cd03228d44e0aaf84f705bd7
              slot: pair:2
              record_digest: d5853f65baae00b8bd15bd39b238f0d27c923f08e378789bccbfb12b495b039f
              start: "2026-10-20T05:17:02.488914Z"
              end: "2026-10-20T05:51:10.074897Z"
              queue_seconds: 2047.585983
              raw_seconds: 2047.585983
              exact: true
        protected:
            - issue: "716"
              slot: pair:2
              start: "2026-10-20T02:15:05.843Z"
              end: "2026-10-19T19:23:08-07:00"
            - issue: "716"
              slot: pair:2
              start: "2026-10-19T20:09:42-07:00"
              end: "2026-10-20T03:11:00.435Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 135.59473333333332
        cuts:
            - dispatched: "2026-10-19T18:09:50-07:00"
              finalized: "2026-10-19T18:13:17-07:00"
              finalizing: false
            - dispatched: "2026-10-19T18:14:38-07:00"
              finalized: "2026-10-19T18:19:53-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-19T23:45:36-07:00"
      dispatched: "2026-10-19T23:41:44-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-2
          disposition: addressed
          note: 'Request.Refusal returns PanelRefusal naming `couch --keys SLOT esc` (recipe.go:91-99, #716); asserted in agentdrive panel_test.go:219, couchcmd agent_drive_service_test.go:407, wrapcmd agent_drive_panel_test.go:64.'
          round: 3
      findings:
        - kind: defect
          id: BR-3
          severity: Minor
          title: agentdrive "footer words above the composer" variant actually inserts inside the composer box
          detail: panel_test.go:47 inserts at closing-rule-index minus 1 (after the opening rule), so the variant passes without testing the case its name describes; walk back to the opening rule as wrapcmd agent_drive_panel_test.go:118-126 does.
          family: test-variant-fidelity
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 0127de1adbef4af2f798ff5a796d05ab578d8f62
      blocked: false
---

# Gate ledger — pair#691 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T18:13:17-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-evidence-missing` (kind: defect) Done-when live check (clear-line text:/usage submit, then esc, on a live Claude slot) is not delivered at whole-issue close
  Done-when bullet 1 requires a recognizer fixture plus a live check; the Plan item "Live check after landing" is still unchecked and the Log records only a replay of the old build. Record the hub's live run on this build in the Log, or revise Done-when so the live check is explicitly a post-merge step.

## Round 2 — 2026-10-19T18:19:53-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Revisions 2026-10-19 (BR-1 entry) makes the live check an explicit post-merge hub step (a failure reopens the issue); the close is gated on the 2.1.304 fixture tests, which I re-ran green with the panel term mutation going red.

### Raised

- **BR-2** [Minor] `refusal-guidance-wording` (kind: defect) Panel refusals still say "composer must be empty" rather than naming esc as the only admitted key
  agent_drive.go:641 and model.go:341 give the same message for a non-esc key on a recognized panel; a panel-specific hint would point the hub at `--keys SLOT esc`.

## Round 3 — 2026-10-19T23:45:36-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — Request.Refusal returns PanelRefusal naming `couch --keys SLOT esc` (recipe.go:91-99, #716); asserted in agentdrive panel_test.go:219, couchcmd agent_drive_service_test.go:407, wrapcmd agent_drive_panel_test.go:64.

### Raised

- **BR-3** [Minor] `test-variant-fidelity` (kind: defect) agentdrive "footer words above the composer" variant actually inserts inside the composer box
  panel_test.go:47 inserts at closing-rule-index minus 1 (after the opening rule), so the variant passes without testing the case its name describes; walk back to the opening rule as wrapcmd agent_drive_panel_test.go:118-126 does.

## Open findings

- **BR-3** [Minor] `test-variant-fidelity` (kind: defect) agentdrive "footer words above the composer" variant actually inserts inside the composer box
