---
gate: boundary-review
issue: 775
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: bc22fde9-cb5e-47d3-b63b-4c3be918a01c
            repository: github.com/xianxu/pair
            issue_id: "000775"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/337e65e19b2af3e2
            reviewed_head: 7ce32158d781a5f1c71a0f5107aa3ebf0b686414
            slot: pair:5
            requested_at: "2026-10-21T18:48:48.973935Z"
          phase: finished
          pid: 38225
          started_at: "2026-10-21T18:48:49.075857Z"
          finished_at: "2026-10-21T18:50:19.751472Z"
          session_id: 01a15fcb-6bbd-72d0-8211-f564aefae6cc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 7ce32158d781a5f1c71a0f5107aa3ebf0b686414
        tracker_head: 465913e8bc343a13282a29b866989f297469d439
        branch_point: eaa758a6f0cd38837655be5334400098849410ec
        since: "2026-10-21T10:56:40-07:00"
        until: "2026-10-21T11:47:40-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T10:56:40-07:00"
              end: "2026-10-21T11:47:40-07:00"
              began: "2026-10-21T10:56:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: fcf8005136061ee7a061c1fef6854dac373bfe5daae3bf506b005308cb394a84
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 63656931
              windows:
                - start: "2026-10-21T10:56:40-07:00"
                  end: "2026-10-21T11:47:40-07:00"
        queue_waits:
            - attempt_id: 1854c8857c9461ff2e2db655f22ab7f1
              slot: pair:5
              record_digest: b373e1f39283314bbda49da7e134d97c2ec3c508b1294d9658b6f6b5df1cef51
              start: "2026-10-21T17:56:40Z"
              end: "2026-10-21T18:30:45.832701Z"
              queue_seconds: 2045.832701
              raw_seconds: 2444.526379
              exact: true
      "n": 1
      timestamp: "2026-10-21T11:50:20-07:00"
      dispatched: "2026-10-21T11:48:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Both signature checks accept additional stderr instead of requiring exact replies
          detail: scripts/repro/zellij-false-miss.sh:57 and :60 use grep -Fxq, which matches any complete line rather than the complete stderr stream. These are both instances in this window. Executing the actual worker with the expected diagnostic followed by "unrelated error" produced counts "1 0 0 0" for exit0 and "0 0 1 0" for exit1. This violates the Spec's exact-signature counting contract (ARCH-PURPOSE). Compare the complete stderr bytes and add regressions covering exact positives and extra-line negatives for both signatures, including confirmed and unconfirmed exit1 liveness.
          family: exact-diagnostic-contract
          round: 1
        - id: BR-2
          severity: Important
          title: The standalone reproduction lacks top-level README and atlas updates
          detail: scripts/repro/README.md:3 introduces a runnable diagnostic with query/worker arguments, private-root conventions and named counters, but neither README.md nor atlas/ changes in this range. These are the two missing documentation consumers for this new surface. Add a top-level usage pointer and document the standalone diagnostic and its liveness limits in atlas/zellij-reply-safety.md.
          family: new-surface-documentation
          round: 1
        - id: BR-3
          severity: Minor
          title: The run manifest points to the wrong log for the initial setup failure
          detail: workshop/evidence/000775/run.json:19 says native.log retains the initial setup failure. That file contains the successful measured run; initial-setup-failure.log contains the failure. This is the only incorrect evidence-file reference found in the window. Correct the manifest note.
          family: evidence-reference-accuracy
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e4ee2189-1000-4da4-aa88-9ddd545cc800
            repository: github.com/xianxu/pair
            issue_id: "000775"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/521044732775f127
            reviewed_head: 6515126f10857f851ca746ca5295e70ef91967d2
            slot: pair:5
            requested_at: "2026-10-21T19:02:54.338212Z"
          phase: finished
          pid: 89849
          started_at: "2026-10-21T19:02:54.486838Z"
          finished_at: "2026-10-21T19:05:36.126992Z"
          session_id: 01a15fd8-526d-7e71-840d-6ddebc26ced6
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 6515126f10857f851ca746ca5295e70ef91967d2
        tracker_head: bc99b272e1d6d6e68ef5a1b896407da4d8988f28
        branch_point: eaa758a6f0cd38837655be5334400098849410ec
        since: "2026-10-21T10:56:40-07:00"
        until: "2026-10-21T12:02:05-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T10:56:40-07:00"
              end: "2026-10-21T12:02:05-07:00"
              began: "2026-10-21T10:56:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 125226b7a4514d51e3a8d5af3dc30c8128b828b781e5c0b4fc0074a5b2e8e266
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 64358629
              windows:
                - start: "2026-10-21T10:56:40-07:00"
                  end: "2026-10-21T12:02:05-07:00"
            - harness: codex
              id: 01a15fcf-6ef8-7773-bd25-bcdb05d954fd
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 8b3ba0d2fa149da5c6b354f2c7a56fd53c9251eb0733d32e8d491b5766e303ef
              started: "2026-10-21T18:53:12.419Z"
              spawned: true
              bytes: 313406
              windows:
                - start: "2026-10-21T18:53:12.419Z"
                  end: "2026-10-21T12:02:05-07:00"
        queue_waits:
            - attempt_id: 1854c8857c9461ff2e2db655f22ab7f1
              slot: pair:5
              record_digest: b373e1f39283314bbda49da7e134d97c2ec3c508b1294d9658b6f6b5df1cef51
              start: "2026-10-21T17:56:40Z"
              end: "2026-10-21T18:30:45.832701Z"
              queue_seconds: 2045.832701
              raw_seconds: 2444.526379
              exact: true
        cuts:
            - dispatched: "2026-10-21T11:48:47-07:00"
              finalized: "2026-10-21T11:50:20-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T12:05:37-07:00"
      dispatched: "2026-10-21T12:02:51-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Both signatures call exact_error, which compares complete bytes. All 10 tests pass; replacing that helper with the old grep matcher produces eight failing subcases and zero errors. Exact positives and confirmed/unconfirmed exit1 outcomes are exercised.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The pinned correction adds the runnable README.md usage pointer and atlas/zellij-reply-safety.md:290-311 documentation of private roots, counters and liveness limits, supported by scripts/repro/zellij-false-miss.sh.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The pinned before/after diff changes run.json's reference from native.log to initial-setup-failure.log. The latter contains the setup refusal and cleanup result; native.log contains the successful measured run.
          round: 2
      findings:
        - id: BR-4
          severity: Important
          title: Numeric arguments undergo octal interpretation and integer overflow
          detail: scripts/repro/zellij-false-miss.sh:38-40 accepts digit strings directly into Bash arithmetic. A traced native invocation with arguments 010 1 exits successfully, reports queries=010, and executes only eight queries. Invocations with 18446744073709551617 as either argument also succeed and execute a one-query/one-worker workload because the value wraps to one. Both numeric inputs, n and parallel, are the complete family in this window; their unchecked representations feed worker and launch-loop arithmetic. Validate bounded decimal strings before arithmetic, normalize accepted values consistently for execution/reporting, and add regressions for both arguments covering leading zeros, overflow, limits and one-past-limit refusal.
          family: bounded-decimal-input
          round: 2
        - id: BR-5
          severity: Minor
          title: Atlas overstates the liveness response validation
          detail: 'atlas/zellij-reply-safety.md:301-302 says the response begins with layout { and malformed replies remain unconfirmed, but scripts/repro/zellij-false-miss.sh:27 accepts a matching line anywhere. An actual-worker check with "unrelated diagnostic\nlayout { pane; }\n" increments the confirmed counter. This is the 2nd finding in family new-surface-documentation. Apply the rule that documented predicates must match enforced predicates across all four new documentation consumers: README.md, scripts/repro/README.md, atlas/index.md and atlas/zellij-reply-safety.md. The inaccurate predicate claims are confined to atlas lines 301-302; describe the implemented line-presence check accurately.'
          family: new-surface-documentation
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 087979cd-61a7-4798-9873-b14dd1884ebe
            repository: github.com/xianxu/pair
            issue_id: "000775"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e7651d89c6c2b72a
            reviewed_head: b88131bf4093ef81422cd19f71f18d328863c2c0
            slot: pair:5
            requested_at: "2026-10-21T19:16:11.966725Z"
          phase: finished
          pid: 91910
          started_at: "2026-10-21T19:16:12.142327Z"
          finished_at: "2026-10-21T19:17:40.011644Z"
          session_id: 01a15fe4-7e74-79e1-bf91-ba9aafeda67e
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: b88131bf4093ef81422cd19f71f18d328863c2c0
        tracker_head: 2e76067bd30230e4780016414b5e2eba5a52f5e2
        branch_point: eaa758a6f0cd38837655be5334400098849410ec
        since: "2026-10-21T10:56:40-07:00"
        until: "2026-10-21T12:15:20-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T10:56:40-07:00"
              end: "2026-10-21T12:15:20-07:00"
              began: "2026-10-21T10:56:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 5d22bfa7f8187ebeba03de2408bd93d05b8c95f45a94bff97ed7ed458e2eea10
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 65477962
              windows:
                - start: "2026-10-21T10:56:40-07:00"
                  end: "2026-10-21T12:15:20-07:00"
            - harness: codex
              id: 01a15fcf-6ef8-7773-bd25-bcdb05d954fd
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 9720b8c017b8c632640cbf195ae8bec649748d7aabe9ccd77ae3da7340ae5bcf
              started: "2026-10-21T18:53:12.419Z"
              spawned: true
              bytes: 489393
              windows:
                - start: "2026-10-21T18:53:12.419Z"
                  end: "2026-10-21T12:15:20-07:00"
        queue_waits:
            - attempt_id: 1854c8857c9461ff2e2db655f22ab7f1
              slot: pair:5
              record_digest: b373e1f39283314bbda49da7e134d97c2ec3c508b1294d9658b6f6b5df1cef51
              start: "2026-10-21T17:56:40Z"
              end: "2026-10-21T18:30:45.832701Z"
              queue_seconds: 2045.832701
              raw_seconds: 2444.526379
              exact: true
        cuts:
            - dispatched: "2026-10-21T11:48:47-07:00"
              finalized: "2026-10-21T11:50:20-07:00"
              finalizing: false
            - dispatched: "2026-10-21T12:02:51-07:00"
              finalized: "2026-10-21T12:05:37-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-21T12:17:41-07:00"
      dispatched: "2026-10-21T12:16:09-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Both signatures use complete-byte comparison. Restoring the old grep matcher produces eight failing regression subcases.
          round: 3
        - id: BR-2
          disposition: addressed
          note: README.md:2435 provides the invocation; atlas/zellij-reply-safety.md:290 documents the standalone diagnostic, counters and liveness limits.
          round: 3
        - id: BR-3
          disposition: addressed
          note: run.json names initial-setup-failure.log, which contains the setup refusal and passing cleanup; native.log contains the successful measured run.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Both arguments use bounded_decimal before arithmetic. Tests cover normalization, overflow, exact limits, one-past-limit refusal and actual request counts. Restoring the old CLI validation produces nine failing subcases.
          round: 3
        - id: BR-5
          disposition: addressed
          note: The pinned correction replaces beginning-of-response and malformed-layout claims with line-presence wording at atlas/zellij-reply-safety.md:301-304, matching the worker at scripts/repro/zellij-false-miss.sh:35. All four documentation consumers were checked.
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: b88131bf4093ef81422cd19f71f18d328863c2c0
      blocked: false
---

# Gate ledger — pair#775 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T11:50:20-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `exact-diagnostic-contract` Both signature checks accept additional stderr instead of requiring exact replies
  scripts/repro/zellij-false-miss.sh:57 and :60 use grep -Fxq, which matches any complete line rather than the complete stderr stream. These are both instances in this window. Executing the actual worker with the expected diagnostic followed by "unrelated error" produced counts "1 0 0 0" for exit0 and "0 0 1 0" for exit1. This violates the Spec's exact-signature counting contract (ARCH-PURPOSE). Compare the complete stderr bytes and add regressions covering exact positives and extra-line negatives for both signatures, including confirmed and unconfirmed exit1 liveness.
- **BR-2** [Important] `new-surface-documentation` The standalone reproduction lacks top-level README and atlas updates
  scripts/repro/README.md:3 introduces a runnable diagnostic with query/worker arguments, private-root conventions and named counters, but neither README.md nor atlas/ changes in this range. These are the two missing documentation consumers for this new surface. Add a top-level usage pointer and document the standalone diagnostic and its liveness limits in atlas/zellij-reply-safety.md.
- **BR-3** [Minor] `evidence-reference-accuracy` The run manifest points to the wrong log for the initial setup failure
  workshop/evidence/000775/run.json:19 says native.log retains the initial setup failure. That file contains the successful measured run; initial-setup-failure.log contains the failure. This is the only incorrect evidence-file reference found in the window. Correct the manifest note.

## Round 2 — 2026-10-21T12:05:37-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Both signatures call exact_error, which compares complete bytes. All 10 tests pass; replacing that helper with the old grep matcher produces eight failing subcases and zero errors. Exact positives and confirmed/unconfirmed exit1 outcomes are exercised.
- BR-2 — addressed — The pinned correction adds the runnable README.md usage pointer and atlas/zellij-reply-safety.md:290-311 documentation of private roots, counters and liveness limits, supported by scripts/repro/zellij-false-miss.sh.
- BR-3 — addressed — The pinned before/after diff changes run.json's reference from native.log to initial-setup-failure.log. The latter contains the setup refusal and cleanup result; native.log contains the successful measured run.

### Raised

- **BR-4** [Important] `bounded-decimal-input` Numeric arguments undergo octal interpretation and integer overflow
  scripts/repro/zellij-false-miss.sh:38-40 accepts digit strings directly into Bash arithmetic. A traced native invocation with arguments 010 1 exits successfully, reports queries=010, and executes only eight queries. Invocations with 18446744073709551617 as either argument also succeed and execute a one-query/one-worker workload because the value wraps to one. Both numeric inputs, n and parallel, are the complete family in this window; their unchecked representations feed worker and launch-loop arithmetic. Validate bounded decimal strings before arithmetic, normalize accepted values consistently for execution/reporting, and add regressions for both arguments covering leading zeros, overflow, limits and one-past-limit refusal.
- **BR-5** [Minor] `new-surface-documentation` Atlas overstates the liveness response validation
  atlas/zellij-reply-safety.md:301-302 says the response begins with layout { and malformed replies remain unconfirmed, but scripts/repro/zellij-false-miss.sh:27 accepts a matching line anywhere. An actual-worker check with "unrelated diagnostic\nlayout { pane; }\n" increments the confirmed counter. This is the 2nd finding in family new-surface-documentation. Apply the rule that documented predicates must match enforced predicates across all four new documentation consumers: README.md, scripts/repro/README.md, atlas/index.md and atlas/zellij-reply-safety.md. The inaccurate predicate claims are confined to atlas lines 301-302; describe the implemented line-presence check accurately.

## Round 3 — 2026-10-21T12:17:41-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Both signatures use complete-byte comparison. Restoring the old grep matcher produces eight failing regression subcases.
- BR-2 — addressed — README.md:2435 provides the invocation; atlas/zellij-reply-safety.md:290 documents the standalone diagnostic, counters and liveness limits.
- BR-3 — addressed — run.json names initial-setup-failure.log, which contains the setup refusal and passing cleanup; native.log contains the successful measured run.
- BR-4 — addressed — Both arguments use bounded_decimal before arithmetic. Tests cover normalization, overflow, exact limits, one-past-limit refusal and actual request counts. Restoring the old CLI validation produces nine failing subcases.
- BR-5 — addressed — The pinned correction replaces beginning-of-response and malformed-layout claims with line-presence wording at atlas/zellij-reply-safety.md:301-304, matching the worker at scripts/repro/zellij-false-miss.sh:35. All four documentation consumers were checked.

## Open findings

(none — every finding has been disposed)
