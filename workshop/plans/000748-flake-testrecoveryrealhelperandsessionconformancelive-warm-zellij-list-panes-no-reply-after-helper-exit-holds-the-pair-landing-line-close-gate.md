---
gate: boundary-review
issue: 748
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7705cc97-8675-4aae-bd13-3d1a95616701
            repository: github.com/xianxu/pair
            issue_id: "000748"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/495753bc7a03398e
            reviewed_head: 004fe5d455080cc16330700a1e61a02a3324768a
            slot: pair:3
            requested_at: "2026-10-21T02:20:04.314444Z"
          phase: finished
          pid: 41384
          started_at: "2026-10-21T02:20:04.488621Z"
          finished_at: "2026-10-21T02:23:20.484678Z"
          session_id: 01a15c42-3362-77b3-a2c9-0e9bdf1f8cde
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 004fe5d455080cc16330700a1e61a02a3324768a
        tracker_head: e0eaecd328ebe03030b34ca5f19702ed2cf9414d
        branch_point: 127001eae6379ce571b6097f4573deb57ca90416
        since: "2026-10-20T17:33:54-07:00"
        until: "2026-10-20T19:19:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:33:54-07:00"
              end: "2026-10-20T19:19:04-07:00"
              began: "2026-10-20T17:33:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: f3bb71baa83c762c28c4239fd5af689188dd34107d7994c9d152b99189bd1c53
              started: "2026-10-20T21:55:01.064Z"
              spawned: false
              bytes: 5900091
              windows:
                - start: "2026-10-20T17:33:54-07:00"
                  end: "2026-10-20T19:19:04-07:00"
            - harness: codex
              id: 01a15be1-7f3d-7fe1-b320-918261b6457c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              digest: 76a1ff22ff8cb95fdd5df9b723673acfc0c715abfc6886bf7a39a144b5be06f6
              started: "2026-10-21T00:34:27.441Z"
              spawned: true
              bytes: 1551049
              windows:
                - start: "2026-10-21T00:34:27.441Z"
                  end: "2026-10-20T19:19:04-07:00"
      "n": 1
      timestamp: "2026-10-20T19:23:21-07:00"
      dispatched: "2026-10-20T19:20:01-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: The non-retryable-result regression passes when forbidden retries are enabled
          detail: 'cmd/internal/launcher/zellij_read_recovery_test.go:54 supplies exactly the expected number of frames, but zellij_empty_reply_test.go:96 rejects exhausted frames before updating the invocation counter at line 110. Extra children therefore remain invisible. Verified with a temporary Go overlay: replacing the retry gate with “if err == nil” makes mutations, unrecognized argv and nonzero failures retry, yet TestZellijKnownBugReadBoundAndNonRetryableResults still passes. Enumerated affected cases in that table: nonzero, mutation, unknown-flags, and crossed exhaustion; malformed/invalid-pane replies return nil boundary errors and do not exercise error retry. Count every child before frame validation, or inject an independently counted collector; assert the expected final error and make this unsafe mutant fail.'
          family: subprocess-policy-tests-count-every-attempt
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: The native qualification counts query errors rather than bad final answers
          detail: workshop/evidence/000748/native-production-query-diagnostic.go.txt:13 discards stdout and increments badFinal only when runContext returns an error. That boundary deliberately returns malformed or missing-decision-field same-kind payloads without a parsing error, leaving validation to callers. Consequently this control cannot establish its “0 bad final answers” claim. ARCH-PURPOSE requires completing the planned final-answer qualification. Validate each returned pane reply with the production requirements, retain validation failures, and rerun the control. This family's claims occur in native-production-query-result.txt, workshop/evidence/000748/README.md:7, and the issue's implementation-and-qualification passage at line 101; refresh them together.
          family: qualification-validates-returned-evidence
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Query comments still describe the former one-attempt contract
          detail: In cmd/internal/launcher/zellij.go, line 28 says the boundary never retries, line 149 promises no automatic retries, lines 51–52, 60–61 and 72–74 describe subprocess costs as one run, and lines 168–169 call the observer error caller-visible even when a later attempt succeeds. These are all instances of the changed logical-read versus subprocess-attempt distinction. Describe bounded attempts and attempt-local observer errors consistently.
          family: comments-track-query-attempt-semantics
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#748 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T19:23:21-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `subprocess-policy-tests-count-every-attempt` (kind: defect) The non-retryable-result regression passes when forbidden retries are enabled
  cmd/internal/launcher/zellij_read_recovery_test.go:54 supplies exactly the expected number of frames, but zellij_empty_reply_test.go:96 rejects exhausted frames before updating the invocation counter at line 110. Extra children therefore remain invisible. Verified with a temporary Go overlay: replacing the retry gate with “if err == nil” makes mutations, unrecognized argv and nonzero failures retry, yet TestZellijKnownBugReadBoundAndNonRetryableResults still passes. Enumerated affected cases in that table: nonzero, mutation, unknown-flags, and crossed exhaustion; malformed/invalid-pane replies return nil boundary errors and do not exercise error retry. Count every child before frame validation, or inject an independently counted collector; assert the expected final error and make this unsafe mutant fail.
- **BR-2** [Important] `qualification-validates-returned-evidence` (kind: defect) The native qualification counts query errors rather than bad final answers
  workshop/evidence/000748/native-production-query-diagnostic.go.txt:13 discards stdout and increments badFinal only when runContext returns an error. That boundary deliberately returns malformed or missing-decision-field same-kind payloads without a parsing error, leaving validation to callers. Consequently this control cannot establish its “0 bad final answers” claim. ARCH-PURPOSE requires completing the planned final-answer qualification. Validate each returned pane reply with the production requirements, retain validation failures, and rerun the control. This family's claims occur in native-production-query-result.txt, workshop/evidence/000748/README.md:7, and the issue's implementation-and-qualification passage at line 101; refresh them together.
- **BR-3** [Minor] `comments-track-query-attempt-semantics` (kind: defect) Query comments still describe the former one-attempt contract
  In cmd/internal/launcher/zellij.go, line 28 says the boundary never retries, line 149 promises no automatic retries, lines 51–52, 60–61 and 72–74 describe subprocess costs as one run, and lines 168–169 call the observer error caller-visible even when a later attempt succeeds. These are all instances of the changed logical-read versus subprocess-attempt distinction. Describe bounded attempts and attempt-local observer errors consistently.

## Open findings

- **BR-1** [Important] `subprocess-policy-tests-count-every-attempt` (kind: defect) The non-retryable-result regression passes when forbidden retries are enabled
- **BR-2** [Important] `qualification-validates-returned-evidence` (kind: defect) The native qualification counts query errors rather than bad final answers
- **BR-3** [Minor] `comments-track-query-attempt-semantics` (kind: defect) Query comments still describe the former one-attempt contract
