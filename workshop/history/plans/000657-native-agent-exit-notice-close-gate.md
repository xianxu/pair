---
gate: boundary-review
issue: 657
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 52374d32-34c7-4dd0-b4e9-a31a8cf1d3d6
            repository: github.com/xianxu/pair
            issue_id: "000657"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/3b422318883afdc6
            reviewed_head: 7698b84659486513ac5d678bfaf01e0012441193
            slot: pair:2
            requested_at: "2026-10-18T15:01:33.065301Z"
          phase: finished
          pid: 30026
          started_at: "2026-10-18T15:01:33.314312Z"
          finished_at: "2026-10-18T15:06:24.378716Z"
          session_id: dc56a28a-1b04-4fac-aae6-0079ab47af65
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 7698b84659486513ac5d678bfaf01e0012441193
        tracker_head: 5f8f1371b1593a4fed90c3b47cbce3592e604c44
        branch_point: a5bc8e64a82776647060495425ec311dee235b23
        since: "2026-10-18T07:05:50-07:00"
        until: "2026-10-18T08:00:57-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T07:05:50-07:00"
              end: "2026-10-18T08:00:57-07:00"
              began: "2026-10-18T07:05:50-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 402211c23e062d3429985ad5025666e4a2bd9648ec976112dcb306adf6598103
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 32229656
              windows:
                - start: "2026-10-18T07:05:50-07:00"
                  end: "2026-10-18T08:00:57-07:00"
        queue_waits:
            - attempt_id: a3c1d97cc13e87d2e60a949f998fbfe5
              slot: pair:2
              record_digest: 47f2ebbbba2a9e855b45a715847f047139dd79d1a5d3805c437b169cf917ebe2
              start: "2026-10-18T14:27:35.604267Z"
              end: "2026-10-18T14:48:13.089401Z"
              queue_seconds: 1237.485134
              raw_seconds: 1237.485134
              exact: true
      "n": 1
      timestamp: "2026-10-18T08:06:25-07:00"
      dispatched: "2026-10-18T08:01:29-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: wrap.go native-exit call site untested; conformance test re-enacts instead of driving run, and the SIGUSR2 case is missing
          detail: Replacing p.reportNativeExit(peerRun, code) at cmd/internal/wrapcmd/wrap.go:3102 with a no-op leaves the whole wrapcmd suite green. Extract the post-Wait decision into a helper that run calls, and test it with real owned children for self-exit, SIGHUP, SIGUSR2 and no peer runtime; otherwise revise the plan's claim of a real wrapper run.
          family: production-seam-untested-by-reenactment
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: No test covers the report-to-hub-notice chain that the Done-when requires
          detail: 'The startMessageService exited closure (message_service.go:517-521) builds the binding-to-ThreadAddress mapping and the incarnation string, and no test runs it. A bad mapping silently drops the notice because the observation ends up with an empty Path. Add one rig test that sends a current report and asserts the published hub body "slot: exited (code N)".'
          family: done-when-chain-not-integrated
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: The orphan tree reaper SIGTERMs the native child before the wrapper, so the wrapper may report a self exit
          detail: PlanReap (launcher/session_reap.go:76) signals the deepest processes first, so the child can die with code 143 before stoppedByWrapper is set. The report is probably dropped by slot routing for an orphan, but this is plausible rather than confirmed and no test covers it.
          family: external-stop-misread-as-self-exit
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: Plan says Incarnation is the binding key plus the ordinal; the code uses "ordinal:identity"
          detail: message_service.go:518. The value is opaque, so only the plan text needs a Revisions entry.
          family: plan-code-drift
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: The native-exit dedupe and capacity block copies the automatic-input-stall handler
          detail: native_exit.go:53-65 against automatic_input_stall.go:43-55. A shared per-slot occurrence-record helper would remove the copy (ARCH-DRY).
          family: duplicated-occurrence-record
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 669b8f29-d935-4a4c-9856-ee71a2f08c88
            repository: github.com/xianxu/pair
            issue_id: "000657"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/cf611fd3701e918d
            reviewed_head: d1002ef25dda88d3f8055616e82b7bef2555551a
            slot: pair:2
            requested_at: "2026-10-18T15:10:42.81061Z"
          phase: finished
          pid: 83900
          started_at: "2026-10-18T15:10:42.904881Z"
          finished_at: "2026-10-18T15:23:20.494783Z"
          session_id: f6e754a6-823f-477b-a6f0-c37e847d68a2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: d1002ef25dda88d3f8055616e82b7bef2555551a
        tracker_head: 5f8f1371b1593a4fed90c3b47cbce3592e604c44
        branch_point: a5bc8e64a82776647060495425ec311dee235b23
        since: "2026-10-18T07:05:50-07:00"
        until: "2026-10-18T08:10:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T07:05:50-07:00"
              end: "2026-10-18T08:10:24-07:00"
              began: "2026-10-18T07:05:50-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: c2ed64927495365a5ab7d507d9aff208ec75cccae25f6842df98e16941613003
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 32480627
              windows:
                - start: "2026-10-18T07:05:50-07:00"
                  end: "2026-10-18T08:10:24-07:00"
        queue_waits:
            - attempt_id: a3c1d97cc13e87d2e60a949f998fbfe5
              slot: pair:2
              record_digest: 47f2ebbbba2a9e855b45a715847f047139dd79d1a5d3805c437b169cf917ebe2
              start: "2026-10-18T14:27:35.604267Z"
              end: "2026-10-18T14:48:13.089401Z"
              queue_seconds: 1237.485134
              raw_seconds: 1237.485134
              exact: true
        cuts:
            - dispatched: "2026-10-18T08:01:29-07:00"
              finalized: "2026-10-18T08:06:25-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T08:23:21-07:00"
      dispatched: "2026-10-18T08:10:41-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: afterChildReaped holds the whole post-Wait decision; run calls it unconditionally; the test covers self exit, SIGHUP, SIGUSR2, a wait error and no peer runtime, and goes red when the classify guard is reverted.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestNativeExitReachesTheHubAsOneLifecycleNotice runs the handler, wireNativeExitNotices, a real Console and the publisher; restoring the pre-fix lifecycleMetadata(address, "") call makes it FAIL (no lifecycle message).
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: No change or test in round 1; the PlanReap deepest-first SIGTERM can still give a 143 self-exit report; still plausible and Minor.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Plan line 56 still says binding key plus ordinal (the code uses ordinal:identity); line 73 still claims a real wrapper run (the test drives afterChildReaped); the plan checkboxes are unticked. Needs one Revisions entry.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: native_exit.go:58-69 still copies the automatic-input-stall dedupe and capacity block; no shared helper.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: d1002ef25dda88d3f8055616e82b7bef2555551a
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d2234b76-1287-407f-8bab-2d9507c49691
            repository: github.com/xianxu/pair
            issue_id: "000657"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/6d9970b7843fa05a
            reviewed_head: 137152dbba61d1b5ccded1affccc45725c92a21e
            slot: pair:2
            requested_at: "2026-10-18T16:48:46.545523Z"
          phase: finished
          pid: 26469
          started_at: "2026-10-18T16:48:46.649717Z"
          finished_at: "2026-10-18T16:49:49.479561Z"
          session_id: 0136130e-ab9d-4ed9-b9b7-bea01d22ef39
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 137152dbba61d1b5ccded1affccc45725c92a21e
        tracker_head: f2daca97ddf4cc27e7399777a71a16a12ff14d81
        branch_point: 18935a4b30e733382b3d321cb7da6117e7a3dfc8
        since: "2026-10-18T07:05:50-07:00"
        until: "2026-10-18T09:46:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T07:05:50-07:00"
              end: "2026-10-18T09:46:55-07:00"
              began: "2026-10-18T07:05:50-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 47403deede102fc7c6f25eb2fb1d7700d9c87f64aec7ab51ac364ebd1e4604cf
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 33227494
              windows:
                - start: "2026-10-18T07:05:50-07:00"
                  end: "2026-10-18T09:46:55-07:00"
        queue_waits:
            - attempt_id: a3c1d97cc13e87d2e60a949f998fbfe5
              slot: pair:2
              record_digest: 47f2ebbbba2a9e855b45a715847f047139dd79d1a5d3805c437b169cf917ebe2
              start: "2026-10-18T14:27:35.604267Z"
              end: "2026-10-18T14:48:13.089401Z"
              queue_seconds: 1237.485134
              raw_seconds: 1237.485134
              exact: true
            - attempt_id: 1b2b7c3d69d680d0691c9e1395a07610
              slot: pair:2
              record_digest: 0d3aed4552a265f5e99b2a81c6bb9f22c3d9480293b807339fae1036856bc52c
              start: "2026-10-18T16:04:28.656922Z"
              end: "2026-10-18T16:26:57.195067Z"
              queue_seconds: 1348.538145
              raw_seconds: 1348.538145
              exact: true
        cuts:
            - dispatched: "2026-10-18T08:01:29-07:00"
              finalized: "2026-10-18T08:06:25-07:00"
              finalizing: false
            - dispatched: "2026-10-18T08:10:41-07:00"
              finalized: "2026-10-18T08:23:21-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-18T09:49:50-07:00"
      dispatched: "2026-10-18T09:48:45-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: Interdiff touches only the conformance test; reaper signal ordering and coverage unchanged (Minor, non-blocking).
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Plan file unchanged in the window; no Revisions entry records the ordinal:identity Incarnation format.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: native_exit.go dedupe/capacity block still duplicates automatic_input_stall.go; no shared helper in the window (ARCH-DRY, Minor).
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 137152dbba61d1b5ccded1affccc45725c92a21e
      blocked: false
---

# Gate ledger — pair#657 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T08:06:25-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `production-seam-untested-by-reenactment` (kind: defect) wrap.go native-exit call site untested; conformance test re-enacts instead of driving run, and the SIGUSR2 case is missing
  Replacing p.reportNativeExit(peerRun, code) at cmd/internal/wrapcmd/wrap.go:3102 with a no-op leaves the whole wrapcmd suite green. Extract the post-Wait decision into a helper that run calls, and test it with real owned children for self-exit, SIGHUP, SIGUSR2 and no peer runtime; otherwise revise the plan's claim of a real wrapper run.
- **BR-2** [Important] `done-when-chain-not-integrated` (kind: defect) No test covers the report-to-hub-notice chain that the Done-when requires
  The startMessageService exited closure (message_service.go:517-521) builds the binding-to-ThreadAddress mapping and the incarnation string, and no test runs it. A bad mapping silently drops the notice because the observation ends up with an empty Path. Add one rig test that sends a current report and asserts the published hub body "slot: exited (code N)".
- **BR-3** [Minor] `external-stop-misread-as-self-exit` (kind: defect) The orphan tree reaper SIGTERMs the native child before the wrapper, so the wrapper may report a self exit
  PlanReap (launcher/session_reap.go:76) signals the deepest processes first, so the child can die with code 143 before stoppedByWrapper is set. The report is probably dropped by slot routing for an orphan, but this is plausible rather than confirmed and no test covers it.
- **BR-4** [Minor] `plan-code-drift` (kind: defect) Plan says Incarnation is the binding key plus the ordinal; the code uses "ordinal:identity"
  message_service.go:518. The value is opaque, so only the plan text needs a Revisions entry.
- **BR-5** [Minor] `duplicated-occurrence-record` (kind: defect) The native-exit dedupe and capacity block copies the automatic-input-stall handler
  native_exit.go:53-65 against automatic_input_stall.go:43-55. A shared per-slot occurrence-record helper would remove the copy (ARCH-DRY).

## Round 2 — 2026-10-18T08:23:21-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — afterChildReaped holds the whole post-Wait decision; run calls it unconditionally; the test covers self exit, SIGHUP, SIGUSR2, a wait error and no peer runtime, and goes red when the classify guard is reverted.
- BR-2 — addressed — TestNativeExitReachesTheHubAsOneLifecycleNotice runs the handler, wireNativeExitNotices, a real Console and the publisher; restoring the pre-fix lifecycleMetadata(address, "") call makes it FAIL (no lifecycle message).
- BR-3 — not-addressed — No change or test in round 1; the PlanReap deepest-first SIGTERM can still give a 143 self-exit report; still plausible and Minor.
- BR-4 — not-addressed — Plan line 56 still says binding key plus ordinal (the code uses ordinal:identity); line 73 still claims a real wrapper run (the test drives afterChildReaped); the plan checkboxes are unticked. Needs one Revisions entry.
- BR-5 — not-addressed — native_exit.go:58-69 still copies the automatic-input-stall dedupe and capacity block; no shared helper.

## Round 3 — 2026-10-18T09:49:50-07:00 (claude) — passed

### Disposed

- BR-3 — not-addressed — Interdiff touches only the conformance test; reaper signal ordering and coverage unchanged (Minor, non-blocking).
- BR-4 — not-addressed — Plan file unchanged in the window; no Revisions entry records the ordinal:identity Incarnation format.
- BR-5 — not-addressed — native_exit.go dedupe/capacity block still duplicates automatic_input_stall.go; no shared helper in the window (ARCH-DRY, Minor).

## Open findings

- **BR-3** [Minor] `external-stop-misread-as-self-exit` (kind: defect) The orphan tree reaper SIGTERMs the native child before the wrapper, so the wrapper may report a self exit
- **BR-4** [Minor] `plan-code-drift` (kind: defect) Plan says Incarnation is the binding key plus the ordinal; the code uses "ordinal:identity"
- **BR-5** [Minor] `duplicated-occurrence-record` (kind: defect) The native-exit dedupe and capacity block copies the automatic-input-stall handler
