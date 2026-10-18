---
gate: boundary-review
issue: 654
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ec1863c6-9de3-4574-816f-e7b65161aba9
            repository: github.com/xianxu/pair
            issue_id: "000654"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/b00e82452cc65eec
            reviewed_head: 07df865f7fdefe2c7243cb888a9c8e8d51662c2b
            slot: pair:2
            requested_at: "2026-10-18T18:50:14.546424Z"
          phase: finished
          pid: 8068
          started_at: "2026-10-18T18:50:14.654029Z"
          finished_at: "2026-10-18T18:52:21.27386Z"
          session_id: 8a9146c0-d9ee-46e1-b5ba-4389d7e97a84
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 07df865f7fdefe2c7243cb888a9c8e8d51662c2b
        tracker_head: c0850423b93ebb8a709f063b166a339dec75642c
        branch_point: db5ddb4e2bf5af6ec85c3c916e51b17523c12085
        since: "2026-10-18T00:48:48-07:00"
        until: "2026-10-18T11:21:00-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T00:48:48-07:00"
              end: "2026-10-18T11:16:20-07:00"
              began: "2026-10-18T00:48:48-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
            - start: "2026-10-18T11:16:40-07:00"
              end: "2026-10-18T11:21:00-07:00"
              began: "2026-10-18T11:16:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 35ee268f1ed7601f7f6d20a79e4247ab8884606f2a94172ad883f6fc92582a76
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 34082754
              windows:
                - start: "2026-10-18T11:16:40-07:00"
                  end: "2026-10-18T11:21:00-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: fabb6fef8e6383d37e0d356ee08164736d7b4da6beb8ccad59b0d7ae27a8f012
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 29339748
              windows:
                - start: "2026-10-18T00:48:48-07:00"
                  end: "2026-10-18T11:16:20-07:00"
      "n": 1
      timestamp: "2026-10-18T11:52:22-07:00"
      dispatched: "2026-10-18T11:50:13-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: unverifiable
          id: BR-1
          severity: Important
          title: Done-when clause "A Qoder slot's pane title shows its count. Checked live." is not done; its Plan item is still unchecked
          detail: 'The issue''s final Plan item (live check after the next Couch restart) is still [ ], and the window holds no evidence of a live observation. A review checkout cannot run Couch or Qoder. Evidence that would answer this: a Log entry recording a real Qoder pane title showing a percentage (e.g. "qoder (40%)") after the restart, with the Plan item ticked, or an explicit decision to move this clause out of this close. This is the only instance of the family in the window.'
          family: done-when-clause-undelivered-at-close
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Design text still says ContextRatio is *float64 with range [0, 1]; the code uses float64 with range (0, 1]
          detail: 'Both changes are justified in the 2026-10-18 Log, but the Design section has no Revisions entry, so the plan still states the old contract. Instances: the field type (usage.go:17) and the accepted range (usage.go:105).'
          family: plan-design-drift-unrevised
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3d5649d9-e495-4d89-b542-6da6696776c1
            repository: github.com/xianxu/pair
            issue_id: "000654"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/32d165db8e4e1457
            reviewed_head: d934e9d23fae9ce3b087ad9975fd4fecda292f92
            slot: pair:2
            requested_at: "2026-10-18T18:53:15.125099Z"
          phase: finished
          pid: 27578
          started_at: "2026-10-18T18:53:15.226665Z"
          finished_at: "2026-10-18T18:59:50.397702Z"
          session_id: 37738440-0702-4631-a513-5a0b44259580
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: d934e9d23fae9ce3b087ad9975fd4fecda292f92
        tracker_head: c0850423b93ebb8a709f063b166a339dec75642c
        branch_point: db5ddb4e2bf5af6ec85c3c916e51b17523c12085
        since: "2026-10-18T00:48:48-07:00"
        until: "2026-10-18T11:52:46-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T00:48:48-07:00"
              end: "2026-10-18T11:16:20-07:00"
              began: "2026-10-18T00:48:48-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
            - start: "2026-10-18T11:16:40-07:00"
              end: "2026-10-18T11:52:46-07:00"
              began: "2026-10-18T11:16:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bb8b2ee2d5421d1e45a6b4042f7f232dd65e19348b42fac53d85b1d3af230baa
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 34184889
              windows:
                - start: "2026-10-18T11:16:40-07:00"
                  end: "2026-10-18T11:52:46-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: fabb6fef8e6383d37e0d356ee08164736d7b4da6beb8ccad59b0d7ae27a8f012
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 29339748
              windows:
                - start: "2026-10-18T00:48:48-07:00"
                  end: "2026-10-18T11:16:20-07:00"
        queue_waits:
            - attempt_id: d94e4d31d47933278b9ff5c36f9dc1c5
              slot: pair:2
              record_digest: 98c73f7457b392d3f6b7ebdc65f415cd703ed50f5c76e4f4da5123c8b06d007c
              start: "2026-10-18T18:21:10.366951Z"
              end: "2026-10-18T18:35:33.693325Z"
              queue_seconds: 863.326374
              raw_seconds: 863.326374
              exact: true
        cuts:
            - dispatched: "2026-10-18T11:50:13-07:00"
              finalized: "2026-10-18T11:52:22-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T11:59:51-07:00"
      dispatched: "2026-10-18T11:53:13-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-2
          disposition: addressed
          note: Revisions entry (2026-10-18, BR-2) supersedes the Design's *float64 and [0, 1] claims, matching usage.go:17 and :105. The Plan's first item still says [0, 1]; widening the Revisions line to cover it would finish the sweep.
          round: 2
        - id: BR-1
          disposition: withdrawn
          note: Evidence holds. TestTitleProjectionProviderAppendParity/qoder, rerun here and passing, asserts HasUsage with Figure 40% through the same Usage.Figure() call that titlepoller/projection.go:53 and couchcmd/titles.go:342 make. A pane title can only show the figure after landing and a Couch restart, so the live half is a post-landing TL check (the unchecked Plan item records it), not a defect in this range.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: d934e9d23fae9ce3b087ad9975fd4fecda292f92
      blocked: false
disputes:
    - id: BR-1
      rationale: 'The live clause cannot be met before landing: a pane title shows a figure only from the title poller of a Couch running a build that contains this fix, and pair:0 runs main, rebuilt after merge; Qoder is out of credits, so no new Qoder session can be started on a branch build either. The TL holds the post-restart live check, as for pair#653 (its BR-4 disputed the same way). The code half is evidenced by the closest pre-landing measurement: the production parser over the complete live Qoder transcripts.'
      evidence: 'Re-ran TokenUsageFromJSONL(AgentQoder) over the full on-disk transcripts ~/.qoder/projects/-Users-xianxu-workspace-worktree-pair-slot4-pair/948602c0...jsonl and slot8-pair/7febf963...jsonl: true 42% and true 40%, equal to their last context_usage_ratio 0.42245 and 0.39998; TestTitleProjectionProviderAppendParity/qoder asserts the title projection yields HasUsage with Figure 40% through the same path the poller formats.'
      after: 1
      timestamp: "2026-10-18T18:52:44Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair)
---

# Gate ledger — pair#654 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T11:52:22-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-clause-undelivered-at-close` (kind: unverifiable) Done-when clause "A Qoder slot's pane title shows its count. Checked live." is not done; its Plan item is still unchecked
  The issue's final Plan item (live check after the next Couch restart) is still [ ], and the window holds no evidence of a live observation. A review checkout cannot run Couch or Qoder. Evidence that would answer this: a Log entry recording a real Qoder pane title showing a percentage (e.g. "qoder (40%)") after the restart, with the Plan item ticked, or an explicit decision to move this clause out of this close. This is the only instance of the family in the window.
- **BR-2** [Minor] `plan-design-drift-unrevised` (kind: defect) Design text still says ContextRatio is *float64 with range [0, 1]; the code uses float64 with range (0, 1]
  Both changes are justified in the 2026-10-18 Log, but the Design section has no Revisions entry, so the plan still states the old contract. Instances: the field type (usage.go:17) and the accepted range (usage.go:105).

## Round 2 — 2026-10-18T11:59:51-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — Revisions entry (2026-10-18, BR-2) supersedes the Design's *float64 and [0, 1] claims, matching usage.go:17 and :105. The Plan's first item still says [0, 1]; widening the Revisions line to cover it would finish the sweep.
- BR-1 — withdrawn — Evidence holds. TestTitleProjectionProviderAppendParity/qoder, rerun here and passing, asserts HasUsage with Figure 40% through the same Usage.Figure() call that titlepoller/projection.go:53 and couchcmd/titles.go:342 make. A pane title can only show the figure after landing and a Couch restart, so the live half is a post-landing TL check (the unchecked Plan item records it), not a defect in this range.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-1** — after round 1 (2026-10-18T18:52:44Z, Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair))
  rationale: The live clause cannot be met before landing: a pane title shows a figure only from the title poller of a Couch running a build that contains this fix, and pair:0 runs main, rebuilt after merge; Qoder is out of credits, so no new Qoder session can be started on a branch build either. The TL holds the post-restart live check, as for pair#653 (its BR-4 disputed the same way). The code half is evidenced by the closest pre-landing measurement: the production parser over the complete live Qoder transcripts.
  evidence: Re-ran TokenUsageFromJSONL(AgentQoder) over the full on-disk transcripts ~/.qoder/projects/-Users-xianxu-workspace-worktree-pair-slot4-pair/948602c0...jsonl and slot8-pair/7febf963...jsonl: true 42% and true 40%, equal to their last context_usage_ratio 0.42245 and 0.39998; TestTitleProjectionProviderAppendParity/qoder asserts the title projection yields HasUsage with Figure 40% through the same path the poller formats.

## Open findings

(none — every finding has been disposed)
