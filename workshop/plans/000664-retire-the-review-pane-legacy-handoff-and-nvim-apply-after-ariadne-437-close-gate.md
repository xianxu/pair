---
gate: boundary-review
issue: 664
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e07e6e32-56ce-4e27-be76-896f75794c3d
            repository: github.com/xianxu/pair
            issue_id: "000664"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/325f4353ad4609d5
            reviewed_head: e9b99eb5c103e48584b4435ec350295e0fc60cf1
            slot: pair:2
            requested_at: "2026-10-19T02:10:08.231902Z"
          phase: finished
          pid: 87469
          started_at: "2026-10-19T02:10:08.345389Z"
          finished_at: "2026-10-19T02:15:53.902742Z"
          session_id: 7511cd26-1235-4480-9418-5d2946696e9e
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e9b99eb5c103e48584b4435ec350295e0fc60cf1
        tracker_head: 51f04bc5c359c8bc430d63572c7414d68bf99871
        branch_point: 455737dcd073ae5a8610f9b9e7483203c7708f07
        since: "2026-10-18T18:18:27-07:00"
        until: "2026-10-18T18:40:31-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T18:18:27-07:00"
              end: "2026-10-18T18:40:31-07:00"
              began: "2026-10-18T18:18:27-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 0d5d827d1689deb4958a9d7b73dd4c0107a5ac8a14a8ed254545fb0c8e75131d
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 38783703
              windows:
                - start: "2026-10-18T18:18:27-07:00"
                  end: "2026-10-18T18:40:31-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
      "n": 1
      timestamp: "2026-10-18T19:16:05-07:00"
      dispatched: "2026-10-18T19:10:06-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: agent_round_rule_test on_lines detector detaches at :edit!, so the declined-round check passes no matter what
          detail: 'A probe showed nvim_buf_attach is detached by `:edit!` (on_detach fires, later set_lines give on_lines delta 0). So in nvim/review/agent_round_rule_test.lua:63-65, `line_changes == typed` passes without checking anything, even the operator''s own edit is not counted, and nothing positively controls on_lines or the writefile/io.open/os.rename spies. The plan''s claimed marker-resolve positive control is also missing. Every instance: the declined-round assertion at line 65, the missing on_lines control, the missing file-spy control, the missing marker accept/reject control. Fix: track changedtick deltas, or re-attach in on_reload, and assert that each detector fires on an operator action.'
          family: detector-without-positive-control
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Done-when clause 3 (live round with an ariadne#437 agent) is unchecked and has no Log evidence
          detail: The last Plan item ("a live round with an ariadne#437 agent (operator smoke)") is unchecked, and the Log has no smoke entry showing a live round applied through `pair review apply` with the pane only reloading and decorating. Record the smoke result before close, or hand the clause to a named owner.
          family: done-when-clause-unevidenced
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: spinner.lua header comment still says the spinner clears when "the handoff lands"
          detail: nvim/review/spinner.lua:3. It is now cleared by after_applied_round (and the definition paths). This is the only stale handoff comment left under nvim/.
          family: stale-legacy-vocabulary
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: restore.validate_context `legacy` parameter is now dead
          detail: nvim/review/restore.lua:7. The only legacy=true source was the retired restore_controller.legacy; review.lua passes false or nil.
          family: dead-seam-after-retirement
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: review-window-test rebuilds the round hash as sha256():sub(1,16) instead of round_watch.hash
          detail: This is an ARCH-DRY issue in tests/review-window-test.sh's applied-round block. agent_round_rule_test.lua already uses round_watch.hash, and the window test should use it too.
          family: duplicated-round-hash
          round: 1
        - kind: defect
          id: BR-6
          severity: Minor
          title: A zero-edit agent reply may leave the awaiting spinner and the restore block set
          detail: The legacy empty handoff cleared awaiting; now only an applied-round artifact does. If an agent review produces no edits and no artifact, "agent request pending" blocks Alt+c restore. Check what the engine and ariadne#437 do for an empty round.
          family: spinner-clear-paths
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
disputes:
    - id: BR-2
      rationale: 'The live-round clause cannot be evidenced before landing: it needs this build running in the operator''s live Couch with an ariadne#437 agent, which happens only after merge and a Couch restart. The TL holds live checks for the next restart (same disposition as #654 BR-1). Everything the live round exercises is covered headless: the same pair review apply path, reload and decorate.'
      evidence: 'bash tests/review-applied-round-test.sh: ok (real pair review apply, reload plus decoration, commit Body). bash tests/review-branch-restore-test.sh: ok (real pair review apply round in a live pane, CRLF/BOM preserved). nvim -l nvim/review/agent_round_rule_test.lua: ok; with on_applied_round mutated to set lines it fails 4 assertions. The live Couch runs a pre-#664 binary (vcs.revision predates this branch), so a live round today would exercise the old code.'
      after: 1
      timestamp: "2026-10-19T02:17:59Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair)
---

# Gate ledger — pair#664 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T19:16:05-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `detector-without-positive-control` (kind: defect) agent_round_rule_test on_lines detector detaches at :edit!, so the declined-round check passes no matter what
  A probe showed nvim_buf_attach is detached by `:edit!` (on_detach fires, later set_lines give on_lines delta 0). So in nvim/review/agent_round_rule_test.lua:63-65, `line_changes == typed` passes without checking anything, even the operator's own edit is not counted, and nothing positively controls on_lines or the writefile/io.open/os.rename spies. The plan's claimed marker-resolve positive control is also missing. Every instance: the declined-round assertion at line 65, the missing on_lines control, the missing file-spy control, the missing marker accept/reject control. Fix: track changedtick deltas, or re-attach in on_reload, and assert that each detector fires on an operator action.
- **BR-2** [Important] `done-when-clause-unevidenced` (kind: defect) Done-when clause 3 (live round with an ariadne#437 agent) is unchecked and has no Log evidence
  The last Plan item ("a live round with an ariadne#437 agent (operator smoke)") is unchecked, and the Log has no smoke entry showing a live round applied through `pair review apply` with the pane only reloading and decorating. Record the smoke result before close, or hand the clause to a named owner.
- **BR-3** [Minor] `stale-legacy-vocabulary` (kind: defect) spinner.lua header comment still says the spinner clears when "the handoff lands"
  nvim/review/spinner.lua:3. It is now cleared by after_applied_round (and the definition paths). This is the only stale handoff comment left under nvim/.
- **BR-4** [Minor] `dead-seam-after-retirement` (kind: defect) restore.validate_context `legacy` parameter is now dead
  nvim/review/restore.lua:7. The only legacy=true source was the retired restore_controller.legacy; review.lua passes false or nil.
- **BR-5** [Minor] `duplicated-round-hash` (kind: defect) review-window-test rebuilds the round hash as sha256():sub(1,16) instead of round_watch.hash
  This is an ARCH-DRY issue in tests/review-window-test.sh's applied-round block. agent_round_rule_test.lua already uses round_watch.hash, and the window test should use it too.
- **BR-6** [Minor] `spinner-clear-paths` (kind: defect) A zero-edit agent reply may leave the awaiting spinner and the restore block set
  The legacy empty handoff cleared awaiting; now only an applied-round artifact does. If an agent review produces no edits and no artifact, "agent request pending" blocks Alt+c restore. Check what the engine and ariadne#437 do for an empty round.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-2** — after round 1 (2026-10-19T02:17:59Z, Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair))
  rationale: The live-round clause cannot be evidenced before landing: it needs this build running in the operator's live Couch with an ariadne#437 agent, which happens only after merge and a Couch restart. The TL holds live checks for the next restart (same disposition as #654 BR-1). Everything the live round exercises is covered headless: the same pair review apply path, reload and decorate.
  evidence: bash tests/review-applied-round-test.sh: ok (real pair review apply, reload plus decoration, commit Body). bash tests/review-branch-restore-test.sh: ok (real pair review apply round in a live pane, CRLF/BOM preserved). nvim -l nvim/review/agent_round_rule_test.lua: ok; with on_applied_round mutated to set lines it fails 4 assertions. The live Couch runs a pre-#664 binary (vcs.revision predates this branch), so a live round today would exercise the old code.

## Open findings

- **BR-1** [Important] `detector-without-positive-control` (kind: defect) agent_round_rule_test on_lines detector detaches at :edit!, so the declined-round check passes no matter what
- **BR-3** [Minor] `stale-legacy-vocabulary` (kind: defect) spinner.lua header comment still says the spinner clears when "the handoff lands"
- **BR-4** [Minor] `dead-seam-after-retirement` (kind: defect) restore.validate_context `legacy` parameter is now dead
- **BR-5** [Minor] `duplicated-round-hash` (kind: defect) review-window-test rebuilds the round hash as sha256():sub(1,16) instead of round_watch.hash
- **BR-6** [Minor] `spinner-clear-paths` (kind: defect) A zero-edit agent reply may leave the awaiting spinner and the restore block set
