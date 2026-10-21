---
gate: boundary-review
issue: 744
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8fc913c6-1521-4840-bd9f-dbaa7ee7dd93
            repository: github.com/xianxu/pair
            issue_id: "000744"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/029e8c37ac78628c
            reviewed_head: 6d584dc279b5745064cabd5a8e59f01da3c6e8d6
            slot: pair:5
            requested_at: "2026-10-21T07:36:31.057793Z"
          phase: finished
          pid: 1743
          started_at: "2026-10-21T07:36:31.151456Z"
          finished_at: "2026-10-21T07:41:28.860899Z"
          session_id: 01a15d63-e9ae-7793-8a4c-0c0d633d2a1f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 6d584dc279b5745064cabd5a8e59f01da3c6e8d6
        tracker_head: ebec37c3d5a0909023c5f71fac27c237fe739e68
        branch_point: 6df5e89ee15015626ed2e72f99cd26f3b66b1dbe
        since: "2026-10-20T14:46:17-07:00"
        until: "2026-10-21T00:35:43-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T14:46:17-07:00"
              end: "2026-10-21T00:35:43-07:00"
              began: "2026-10-20T14:46:17-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 12ee1b912d50bded710fb5b9247834f2a0ceca85c22ff1c3c305f302fb68e422
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 41658474
              windows:
                - start: "2026-10-20T14:46:17-07:00"
                  end: "2026-10-21T00:35:43-07:00"
            - harness: codex
              id: 01a15bb4-d9dc-77d2-b060-2c53819119cb
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: c126cd6662734a949076eec6a6ac256475ad19958332a515e900ec82cd819479
              started: "2026-10-20T23:45:42.133Z"
              spawned: false
              bytes: 650271
              windows:
                - start: "2026-10-20T23:45:42.133Z"
                  end: "2026-10-21T00:35:43-07:00"
            - harness: codex
              id: 01a15bb8-0b6d-7f13-b9af-97cac5c1efeb
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: cf7d84d0cdde49e1540fe4a20e24e7e2f047ae98691cd71b6401e2633667ccd4
              started: "2026-10-20T23:49:11.262Z"
              spawned: false
              bytes: 451427
              windows:
                - start: "2026-10-20T23:49:11.262Z"
                  end: "2026-10-21T00:35:43-07:00"
            - harness: codex
              id: 01a15bb9-aa49-76f1-8e85-aca18bb08618
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: ed963e2ce2eab8523538f4ae7db2fb83ce607214674e0effa139f3ef6b3bc29c
              started: "2026-10-20T23:50:57.738Z"
              spawned: false
              bytes: 468665
              windows:
                - start: "2026-10-20T23:50:57.738Z"
                  end: "2026-10-21T00:35:43-07:00"
            - harness: codex
              id: 01a15bba-35bc-73f3-8319-d2dee53a04e3
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b09d277600703840ac84217aeebf9d2bd2b3863be3a1098c34847efc0d1591c8
              started: "2026-10-20T23:51:33.073Z"
              spawned: false
              bytes: 468576
              windows:
                - start: "2026-10-20T23:51:33.073Z"
                  end: "2026-10-21T00:35:43-07:00"
            - harness: codex
              id: 01a15d19-969d-7671-b91b-54b0d8f464ec
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 11e6a8e57d5f82b796cfc1020f020054e8cb27b0cbfd345e4ebe5b74a6a93182
              started: "2026-10-21T06:15:20.547Z"
              spawned: true
              bytes: 1449336
              windows:
                - start: "2026-10-21T06:15:20.547Z"
                  end: "2026-10-21T00:35:43-07:00"
            - harness: codex
              id: 01a15d19-dd03-7df3-ab88-b40ad1ddbf52
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 735e45bfb2b70250b0509d1e06543321de81c7da49d35cdde3bfbf83ff344c43
              started: "2026-10-21T06:15:38.583Z"
              spawned: true
              bytes: 1466001
              windows:
                - start: "2026-10-21T06:15:38.583Z"
                  end: "2026-10-21T00:35:43-07:00"
      "n": 1
      timestamp: "2026-10-21T00:41:29-07:00"
      dispatched: "2026-10-21T00:36:29-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: The new socket-length requirement breaks existing isolated consumers
          detail: 'cmd/internal/couchmessage/transport.go:65 rejects existing test roots. The ordinary package run failed TestAttentionServiceIgnoresLegacyListAndDoesNotPersistEvents, TestScratchServiceDrivesMaintenanceAndJoinsOnClose, and TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub with the socket-length error; the first two reproduced independently. ARCH-PURPOSE: sweep isolated consumers and give them owned short roots while preserving private-parent confinement and overflow refusal.'
          family: isolation-contract-consumer-coverage
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: ActorCLI cancellation leaves published commands and children running
          detail: 'cmd/internal/testcouch/actor.go:78-94 returns when the caller context ends without canceling or retiring the published request. agentfixture/run.go:369 executes that request under the fixture-wide context, with its relative timeout starting when execution begins. Consequently canceled queued work can start later, active work can continue mutating the fleet, and abandoned results consume mailbox capacity. ARCH-ORDER: add correlated cancellation and acknowledgment, join active children, suppress canceled queued execution, and reclaim terminal request state.'
          family: request-cancellation-joins-effects
          round: 1
        - kind: defect
          id: BR-3
          severity: Critical
          title: The Core concepts table classifies filesystem-backed correlation as PURE
          detail: 'workshop/plans/000744-shared-native-couch-plan.md:30 lists request correlation as PURE, but its implementation in agentfixture/protocol.go:214 and :241 reads and removes mailbox files, and its coverage uses filesystem fixtures. ARCH-PURE: classify mailbox correlation as INTEGRATION or extract an independently tested pure correlation function. Correct the paint-adapter location to paint.go and align the Command/Result names in the same table revision.'
          family: core-concept-classification
          round: 1
        - kind: defect
          id: BR-4
          severity: Important
          title: Ownership-manifest failures are silently discarded
          detail: 'cmd/internal/testcouch/fleet.go:157-163 ignores ownership.json write errors and rewrites the file in place. A failed or partial write can leave cleanup-failure or crash recovery without the promised readable ownership evidence while execution continues normally. ARCH-FUNERAL: publish complete snapshots atomically, surface persistence failures, and preserve cleanup ownership in memory; cover a failed publication.'
          family: recovery-evidence-write-errors
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#744 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T00:41:29-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `isolation-contract-consumer-coverage` (kind: defect) The new socket-length requirement breaks existing isolated consumers
  cmd/internal/couchmessage/transport.go:65 rejects existing test roots. The ordinary package run failed TestAttentionServiceIgnoresLegacyListAndDoesNotPersistEvents, TestScratchServiceDrivesMaintenanceAndJoinsOnClose, and TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub with the socket-length error; the first two reproduced independently. ARCH-PURPOSE: sweep isolated consumers and give them owned short roots while preserving private-parent confinement and overflow refusal.
- **BR-2** [Critical] `request-cancellation-joins-effects` (kind: defect) ActorCLI cancellation leaves published commands and children running
  cmd/internal/testcouch/actor.go:78-94 returns when the caller context ends without canceling or retiring the published request. agentfixture/run.go:369 executes that request under the fixture-wide context, with its relative timeout starting when execution begins. Consequently canceled queued work can start later, active work can continue mutating the fleet, and abandoned results consume mailbox capacity. ARCH-ORDER: add correlated cancellation and acknowledgment, join active children, suppress canceled queued execution, and reclaim terminal request state.
- **BR-3** [Critical] `core-concept-classification` (kind: defect) The Core concepts table classifies filesystem-backed correlation as PURE
  workshop/plans/000744-shared-native-couch-plan.md:30 lists request correlation as PURE, but its implementation in agentfixture/protocol.go:214 and :241 reads and removes mailbox files, and its coverage uses filesystem fixtures. ARCH-PURE: classify mailbox correlation as INTEGRATION or extract an independently tested pure correlation function. Correct the paint-adapter location to paint.go and align the Command/Result names in the same table revision.
- **BR-4** [Important] `recovery-evidence-write-errors` (kind: defect) Ownership-manifest failures are silently discarded
  cmd/internal/testcouch/fleet.go:157-163 ignores ownership.json write errors and rewrites the file in place. A failed or partial write can leave cleanup-failure or crash recovery without the promised readable ownership evidence while execution continues normally. ARCH-FUNERAL: publish complete snapshots atomically, surface persistence failures, and preserve cleanup ownership in memory; cover a failed publication.

## Open findings

- **BR-1** [Critical] `isolation-contract-consumer-coverage` (kind: defect) The new socket-length requirement breaks existing isolated consumers
- **BR-2** [Critical] `request-cancellation-joins-effects` (kind: defect) ActorCLI cancellation leaves published commands and children running
- **BR-3** [Critical] `core-concept-classification` (kind: defect) The Core concepts table classifies filesystem-backed correlation as PURE
- **BR-4** [Important] `recovery-evidence-write-errors` (kind: defect) Ownership-manifest failures are silently discarded
