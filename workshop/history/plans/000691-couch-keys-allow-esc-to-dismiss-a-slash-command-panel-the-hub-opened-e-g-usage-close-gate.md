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

## Open findings

- **BR-2** [Minor] `refusal-guidance-wording` (kind: defect) Panel refusals still say "composer must be empty" rather than naming esc as the only admitted key
