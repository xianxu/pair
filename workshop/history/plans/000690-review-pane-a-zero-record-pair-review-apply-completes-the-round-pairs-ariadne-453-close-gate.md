---
gate: boundary-review
issue: 690
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4c3723bd-5c25-46b0-a8db-728da7771b2f
            repository: github.com/xianxu/pair
            issue_id: "000690"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/fcf8adfc812f2645
            reviewed_head: b84d11e53eb79faf17115f68a9a9f5a7bc5b0c39
            slot: pair:2
            requested_at: "2026-10-19T10:06:45.316577Z"
          phase: finished
          pid: 94442
          started_at: "2026-10-19T10:06:45.655176Z"
          finished_at: "2026-10-19T10:08:33.517992Z"
          session_id: ce33cf02-83ed-4374-b4d5-be1919fb5fed
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b84d11e53eb79faf17115f68a9a9f5a7bc5b0c39
        tracker_head: b4442c77958e5a3c256a7f235322fb4751e1bf95
        branch_point: 0830075fb39de9e5adf9b1120cd353d6191e3fda
        since: "2026-10-19T02:30:24-07:00"
        until: "2026-10-19T02:42:17-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T02:30:24-07:00"
              end: "2026-10-19T02:42:17-07:00"
              began: "2026-10-19T02:30:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 41f06be97e93aaafa02c7bd0fb83d0b87ba6ef8ff69b01822b4eec96e024e53c
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 42527602
              windows:
                - start: "2026-10-19T02:30:24-07:00"
                  end: "2026-10-19T02:42:17-07:00"
      "n": 1
      timestamp: "2026-10-19T03:08:37-07:00"
      dispatched: "2026-10-19T03:06:42-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: The live no-edits pane check in Done-when is still unchecked at the whole-issue close
          detail: 'The Plan item "Live no-edits pane check: the TL holds it for the next Couch restart" is [ ]. It is the only instance in the window. Either record the live evidence in the Log, or add a Revisions entry moving the clause out of this issue before close.'
          family: done-when-clause-undelivered-at-close
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Docs still say every applied round reloads, but an empty round no longer does
          detail: 'Instances: atlas/review-workbench.md:103 ("on each applied round on_applied_round reloads the file") and workshop/targets/review-protocol.md:263 ("on_applied_round only reloads and decorates"). Add "(an empty round only ends the turn)" to both.'
          family: doc-claim-stale-after-behavior-change
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: is_empty errors on spans = JSON null when the hashes match
          detail: 'nvim/review/round_watch.lua:27 applies # to round.spans, which raises an error for vim.NIL. The Go engine always writes [], so only a foreign or older artifact would trigger it. Guard it with type(round.spans) ~= ''table''.'
          family: defensive-artifact-decode
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: The empty-round Warning branch in ApplyFile has no test
          detail: 'cmd/internal/sessionsync/applyfile.go:112-114: the "empty round could not be recorded; commit nothing" text is untested. TestApplyFileRoundWriteFailureWarns covers only a round that changed the file.'
          family: warning-path-untested
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2fd1d9ab-c6f2-4b9d-8071-3dc0024bb7bf
            repository: github.com/xianxu/pair
            issue_id: "000690"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/79964afad28fd987
            reviewed_head: 8e19651f97a513921e4e6ab3cff48759787e275b
            slot: pair:2
            requested_at: "2026-10-19T10:13:11.858047Z"
          phase: finished
          pid: 3877
          started_at: "2026-10-19T10:13:12.047363Z"
          finished_at: "2026-10-19T10:14:27.187015Z"
          session_id: 360adbf1-f091-4c30-b33c-19766069e130
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 8e19651f97a513921e4e6ab3cff48759787e275b
        tracker_head: b4442c77958e5a3c256a7f235322fb4751e1bf95
        branch_point: 0830075fb39de9e5adf9b1120cd353d6191e3fda
        since: "2026-10-19T02:30:24-07:00"
        until: "2026-10-19T03:12:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T02:30:24-07:00"
              end: "2026-10-19T03:12:26-07:00"
              began: "2026-10-19T02:30:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: c0d724f66d007abaa366df3bf133d64b9d356d9985a438917b0053b71d0a80ab
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 42627910
              windows:
                - start: "2026-10-19T02:30:24-07:00"
                  end: "2026-10-19T03:12:26-07:00"
        queue_waits:
            - attempt_id: a06930d02685f5879ecb658da31ac030
              slot: pair:2
              record_digest: a699a3d4b60d9bb111b7b3e0fc58d8be3df2ae9607abcde1215d36a36d895839
              start: "2026-10-19T09:42:24.714884Z"
              end: "2026-10-19T09:53:50.67866Z"
              queue_seconds: 685.963776
              raw_seconds: 685.963776
              exact: true
        cuts:
            - dispatched: "2026-10-19T03:06:42-07:00"
              finalized: "2026-10-19T03:08:37-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-19T03:14:31-07:00"
      dispatched: "2026-10-19T03:13:09-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Done-when no longer claims the live check; the Plan lists it as a post-landing note held by the TL, with a Revisions entry giving why (it needs this branch landed and a Couch restart).
          round: 2
        - id: BR-2
          disposition: addressed
          note: atlas/review-workbench.md:103 and review-protocol.md:263 now say "an empty round only ends the turn"; on_applied_round matches (init.lua is_empty early return).
          round: 2
        - id: BR-3
          disposition: addressed
          note: round_watch.lua:27 guards non-table spans; round_watch_test.lua:43 tests vim.NIL and errors when the guard is removed (verified).
          round: 2
        - id: BR-4
          disposition: addressed
          note: TestApplyFileEmptyRoundWriteFailureWarns asserts "commit nothing" and an empty Body; it fails when the empty-round Warning branch is removed (verified).
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 8e19651f97a513921e4e6ab3cff48759787e275b
      blocked: false
---

# Gate ledger — pair#690 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T03:08:37-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-clause-undelivered-at-close` (kind: defect) The live no-edits pane check in Done-when is still unchecked at the whole-issue close
  The Plan item "Live no-edits pane check: the TL holds it for the next Couch restart" is [ ]. It is the only instance in the window. Either record the live evidence in the Log, or add a Revisions entry moving the clause out of this issue before close.
- **BR-2** [Minor] `doc-claim-stale-after-behavior-change` (kind: defect) Docs still say every applied round reloads, but an empty round no longer does
  Instances: atlas/review-workbench.md:103 ("on each applied round on_applied_round reloads the file") and workshop/targets/review-protocol.md:263 ("on_applied_round only reloads and decorates"). Add "(an empty round only ends the turn)" to both.
- **BR-3** [Minor] `defensive-artifact-decode` (kind: defect) is_empty errors on spans = JSON null when the hashes match
  nvim/review/round_watch.lua:27 applies # to round.spans, which raises an error for vim.NIL. The Go engine always writes [], so only a foreign or older artifact would trigger it. Guard it with type(round.spans) ~= 'table'.
- **BR-4** [Minor] `warning-path-untested` (kind: defect) The empty-round Warning branch in ApplyFile has no test
  cmd/internal/sessionsync/applyfile.go:112-114: the "empty round could not be recorded; commit nothing" text is untested. TestApplyFileRoundWriteFailureWarns covers only a round that changed the file.

## Round 2 — 2026-10-19T03:14:31-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Done-when no longer claims the live check; the Plan lists it as a post-landing note held by the TL, with a Revisions entry giving why (it needs this branch landed and a Couch restart).
- BR-2 — addressed — atlas/review-workbench.md:103 and review-protocol.md:263 now say "an empty round only ends the turn"; on_applied_round matches (init.lua is_empty early return).
- BR-3 — addressed — round_watch.lua:27 guards non-table spans; round_watch_test.lua:43 tests vim.NIL and errors when the guard is removed (verified).
- BR-4 — addressed — TestApplyFileEmptyRoundWriteFailureWarns asserts "commit nothing" and an empty Body; it fails when the empty-round Warning branch is removed (verified).

## Open findings

(none — every finding has been disposed)
