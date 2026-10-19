---
gate: boundary-review
issue: 681
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c6c72929-1ec8-404f-b79f-16fa67aa6a2f
            repository: github.com/xianxu/pair
            issue_id: "000681"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/de29162d72cd3890
            reviewed_head: 7a5a81e82ff6a71893d6380211b8bd60a38f37dc
            slot: pair:5
            requested_at: "2026-10-19T01:45:11.520515Z"
          phase: finished
          pid: 67411
          started_at: "2026-10-19T01:45:11.932634Z"
          finished_at: "2026-10-19T01:47:00.534249Z"
          session_id: 01a151d5-8e10-7cb2-8dbb-8e13145a2b17
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 7a5a81e82ff6a71893d6380211b8bd60a38f37dc
        tracker_head: e20e377dfb7be8a1122512c3c58c3f1bd33125e1
        branch_point: 455737dcd073ae5a8610f9b9e7483203c7708f07
        since: "2026-10-18T18:19:31-07:00"
        until: "2026-10-18T18:44:27-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T18:19:31-07:00"
              end: "2026-10-18T18:44:27-07:00"
              began: "2026-10-18T18:19:31-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 5bc34b5a044de0ef7e2d81b0e5a5207ff23e16cf5144af6e034a9367165abfcd
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 62638405
              windows:
                - start: "2026-10-18T18:19:31-07:00"
                  end: "2026-10-18T18:44:27-07:00"
            - harness: codex
              id: 01a151be-a821-7e51-b212-944a82991c3a
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: a614537587a3e9c10dd3cb19fcc2cd9d6d653e839916cb9de6905cee56c3e03f
              started: "2026-10-19T01:20:11.898Z"
              spawned: true
              bytes: 795117
              windows:
                - start: "2026-10-19T01:20:11.898Z"
                  end: "2026-10-18T18:44:27-07:00"
            - harness: codex
              id: 01a151c3-e9a9-7f51-ba7a-7a7087d9ff5d
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 67debc4e70139a501e79cf9ce7fa39078901070b63da208bb26902fe9570b6c8
              started: "2026-10-19T01:25:56.356Z"
              spawned: true
              bytes: 461920
              windows:
                - start: "2026-10-19T01:25:56.356Z"
                  end: "2026-10-18T18:44:27-07:00"
            - harness: codex
              id: 01a151c6-dbc0-7c31-a7aa-19ec3aefb593
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 3d5a0a2d385b0f5279d0bbf889e6dd99468d44b59ee8ebe8e9495a11e61e0022
              started: "2026-10-19T01:29:09.491Z"
              spawned: true
              bytes: 959220
              windows:
                - start: "2026-10-19T01:29:09.491Z"
                  end: "2026-10-18T18:44:27-07:00"
            - harness: codex
              id: 01a151c9-96d6-7961-ac90-592b88e5d301
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 1cddc9c93b44ab42f084cf8d697eba0ae2c8fd2962f3c8046c1e89b1451898d3
              started: "2026-10-19T01:32:08.432Z"
              spawned: true
              bytes: 554954
              windows:
                - start: "2026-10-19T01:32:08.432Z"
                  end: "2026-10-18T18:44:27-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
      "n": 1
      timestamp: "2026-10-18T18:47:16-07:00"
      dispatched: "2026-10-18T18:45:07-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Core concepts table claims an extension in an unchanged file
          detail: workshop/plans/000681-park-recovery-admission-plan.md:45 lists the stateful service/console fixture in cmd/internal/couchcmd/slot_operations_acceptance_test.go as extended, but that file is unchanged in the pinned range. The new fixture is newParkAdmissionRig in cmd/internal/couchcmd/park_recovery_admission_test.go:70. This is the sole location/status contradiction found across the table rows. Correct the row to identify the new fixture and add a Revisions entry recording the implementation location.
          family: core-concept-location-status
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7a225fee-5546-491f-998a-0a249cf5343e
            repository: github.com/xianxu/pair
            issue_id: "000681"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/c5e9c6c4ede0f8b2
            reviewed_head: bd39d8c6b9760ae977e866c3c26eb8604fce30b6
            slot: pair:5
            requested_at: "2026-10-19T01:48:33.040226Z"
          phase: finished
          pid: 1158
          started_at: "2026-10-19T01:48:33.231446Z"
          finished_at: "2026-10-19T01:51:01.34778Z"
          session_id: 01a151d8-a003-7713-b927-89475596e5b0
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: bd39d8c6b9760ae977e866c3c26eb8604fce30b6
        tracker_head: 17b78795f549a3e1979282e864d18ac5fee5a5bc
        branch_point: 455737dcd073ae5a8610f9b9e7483203c7708f07
        since: "2026-10-18T18:19:31-07:00"
        until: "2026-10-18T18:48:01-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T18:19:31-07:00"
              end: "2026-10-18T18:48:01-07:00"
              began: "2026-10-18T18:19:31-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: a2245b37e74ffc8ed73fb9fd069c00cfe157ad3117a9ad94cba6e5ff0590d5ec
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 62824544
              windows:
                - start: "2026-10-18T18:19:31-07:00"
                  end: "2026-10-18T18:48:01-07:00"
            - harness: codex
              id: 01a151be-a821-7e51-b212-944a82991c3a
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: a614537587a3e9c10dd3cb19fcc2cd9d6d653e839916cb9de6905cee56c3e03f
              started: "2026-10-19T01:20:11.898Z"
              spawned: true
              bytes: 795117
              windows:
                - start: "2026-10-19T01:20:11.898Z"
                  end: "2026-10-18T18:48:01-07:00"
            - harness: codex
              id: 01a151c3-e9a9-7f51-ba7a-7a7087d9ff5d
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 67debc4e70139a501e79cf9ce7fa39078901070b63da208bb26902fe9570b6c8
              started: "2026-10-19T01:25:56.356Z"
              spawned: true
              bytes: 461920
              windows:
                - start: "2026-10-19T01:25:56.356Z"
                  end: "2026-10-18T18:48:01-07:00"
            - harness: codex
              id: 01a151c6-dbc0-7c31-a7aa-19ec3aefb593
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 3d5a0a2d385b0f5279d0bbf889e6dd99468d44b59ee8ebe8e9495a11e61e0022
              started: "2026-10-19T01:29:09.491Z"
              spawned: true
              bytes: 959220
              windows:
                - start: "2026-10-19T01:29:09.491Z"
                  end: "2026-10-18T18:48:01-07:00"
            - harness: codex
              id: 01a151c9-96d6-7961-ac90-592b88e5d301
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 1cddc9c93b44ab42f084cf8d697eba0ae2c8fd2962f3c8046c1e89b1451898d3
              started: "2026-10-19T01:32:08.432Z"
              spawned: true
              bytes: 554954
              windows:
                - start: "2026-10-19T01:32:08.432Z"
                  end: "2026-10-18T18:48:01-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
        cuts:
            - dispatched: "2026-10-18T18:45:07-07:00"
              finalized: "2026-10-18T18:47:16-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T18:51:02-07:00"
      dispatched: "2026-10-18T18:48:30-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The pinned correction changes plan line 45 from an extended slot_operations_acceptance_test.go fixture to newParkAdmissionRig in the new park_recovery_admission_test.go, where the fixture exists at line 70. The Revisions entry at line 139 records that implementation choice. This is a prose-only correction.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: Production hub authorization still performs expensive target discovery before receipt admission
          detail: 'cmd/internal/couchcmd/park_recovery_admission_test.go:175 initializes the service without authority.hubs or the production hubTargetResolver, bypassing the pre-admission target authorization path. Production handleHubSlotOperation calls guard before slotOps.handle (hub_guards.go:130-133); that guard invokes WorkspaceReferencePath, repository Discover, and ResolveWorkspace through hubTargetResolver (lines 45-66). A temporary Go overlay enabling guardHubState(t) and hubTargetResolver(r.couch) on the existing fixture passes with its catalog barrier released, but fails with the barrier held: socket recover-park: context deadline exceeded after 2.16s. ARCH-PURPOSE: the documented zero-workspace-probe admission contract is therefore not delivered for configured hubs. The family includes new recover-park admissions and retained duplicate/conflicting-ID requests, all of which traverse this guard before receipt lookup; its workspace IO sites are caller workspace resolution and target resolution/discovery. Preserve caller authentication, derive admission target ownership from the bound cached identity, and retain full queued/dispatch authorization. Extend the transport regression with service-side hub authority enabled and cover retained requests before fresh target lookup.'
          family: receipt-admission-before-workspace-io
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7f7abc49-1682-472d-97fd-37e2964ce29a
            repository: github.com/xianxu/pair
            issue_id: "000681"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7204542613547955
            reviewed_head: 13bab42b594631c392438fcb89aac5837d3dece8
            slot: pair:5
            requested_at: "2026-10-19T01:56:48.941623Z"
          phase: finished
          pid: 33243
          started_at: "2026-10-19T01:56:49.062601Z"
          finished_at: "2026-10-19T01:58:39.260355Z"
          session_id: 01a151e0-3069-7052-a3f2-c7fd01c226c1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 13bab42b594631c392438fcb89aac5837d3dece8
        tracker_head: 21ed560ed5970c926e24389b579f01705958aa65
        branch_point: 455737dcd073ae5a8610f9b9e7483203c7708f07
        since: "2026-10-18T18:19:31-07:00"
        until: "2026-10-18T18:56:21-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T18:19:31-07:00"
              end: "2026-10-18T18:56:21-07:00"
              began: "2026-10-18T18:19:31-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: df86e1be5f42653f9481cdeb2ef0b574a4094be9939c76b0ab52cd746bf35183
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 63480457
              windows:
                - start: "2026-10-18T18:19:31-07:00"
                  end: "2026-10-18T18:56:21-07:00"
            - harness: codex
              id: 01a151be-a821-7e51-b212-944a82991c3a
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: a614537587a3e9c10dd3cb19fcc2cd9d6d653e839916cb9de6905cee56c3e03f
              started: "2026-10-19T01:20:11.898Z"
              spawned: true
              bytes: 795117
              windows:
                - start: "2026-10-19T01:20:11.898Z"
                  end: "2026-10-18T18:56:21-07:00"
            - harness: codex
              id: 01a151c3-e9a9-7f51-ba7a-7a7087d9ff5d
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 67debc4e70139a501e79cf9ce7fa39078901070b63da208bb26902fe9570b6c8
              started: "2026-10-19T01:25:56.356Z"
              spawned: true
              bytes: 461920
              windows:
                - start: "2026-10-19T01:25:56.356Z"
                  end: "2026-10-18T18:56:21-07:00"
            - harness: codex
              id: 01a151c6-dbc0-7c31-a7aa-19ec3aefb593
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 3d5a0a2d385b0f5279d0bbf889e6dd99468d44b59ee8ebe8e9495a11e61e0022
              started: "2026-10-19T01:29:09.491Z"
              spawned: true
              bytes: 959220
              windows:
                - start: "2026-10-19T01:29:09.491Z"
                  end: "2026-10-18T18:56:21-07:00"
            - harness: codex
              id: 01a151c9-96d6-7961-ac90-592b88e5d301
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 1cddc9c93b44ab42f084cf8d697eba0ae2c8fd2962f3c8046c1e89b1451898d3
              started: "2026-10-19T01:32:08.432Z"
              spawned: true
              bytes: 554954
              windows:
                - start: "2026-10-19T01:32:08.432Z"
                  end: "2026-10-18T18:56:21-07:00"
            - harness: codex
              id: 01a151db-69c9-7aa3-9c41-4c58a6780b64
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: af770e0c3a5343629b5fc308cfecee668ad547dc966094fe21c5e64d9dc82e8d
              started: "2026-10-19T01:51:36.494Z"
              spawned: true
              bytes: 832727
              windows:
                - start: "2026-10-19T01:51:36.494Z"
                  end: "2026-10-18T18:56:21-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
        cuts:
            - dispatched: "2026-10-18T18:45:07-07:00"
              finalized: "2026-10-18T18:47:16-07:00"
              finalizing: false
            - dispatched: "2026-10-18T18:48:30-07:00"
              finalized: "2026-10-18T18:51:02-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-18T18:58:40-07:00"
      dispatched: "2026-10-18T18:56:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-2
          disposition: addressed
          note: Production-wired transport enables service hub authority and hubTargetResolver. Restoring the pre-admission full guard through a temporary Go overlay makes TestRecoverParkAdmissionPrecedesSlowQueuedPreparation fail with context deadline exceeded after 2.27s; pinned code passes. Retained duplicate/conflict tests also verify no fresh caller/target workspace probes. Queued and dispatch authorization remain enforced.
          round: 3
        - id: BR-1
          disposition: addressed
          note: The Core concepts row at workshop/plans/000681-park-recovery-admission-plan.md:45 correctly identifies newParkAdmissionRig in the newly added park_recovery_admission_test.go. The pinned diff and fixture definition support the corrected location/status.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 13bab42b594631c392438fcb89aac5837d3dece8
      blocked: false
---

# Gate ledger — pair#681 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T18:47:16-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `core-concept-location-status` (kind: defect) Core concepts table claims an extension in an unchanged file
  workshop/plans/000681-park-recovery-admission-plan.md:45 lists the stateful service/console fixture in cmd/internal/couchcmd/slot_operations_acceptance_test.go as extended, but that file is unchanged in the pinned range. The new fixture is newParkAdmissionRig in cmd/internal/couchcmd/park_recovery_admission_test.go:70. This is the sole location/status contradiction found across the table rows. Correct the row to identify the new fixture and add a Revisions entry recording the implementation location.

## Round 2 — 2026-10-18T18:51:02-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The pinned correction changes plan line 45 from an extended slot_operations_acceptance_test.go fixture to newParkAdmissionRig in the new park_recovery_admission_test.go, where the fixture exists at line 70. The Revisions entry at line 139 records that implementation choice. This is a prose-only correction.

### Raised

- **BR-2** [Critical] `receipt-admission-before-workspace-io` (kind: defect) Production hub authorization still performs expensive target discovery before receipt admission
  cmd/internal/couchcmd/park_recovery_admission_test.go:175 initializes the service without authority.hubs or the production hubTargetResolver, bypassing the pre-admission target authorization path. Production handleHubSlotOperation calls guard before slotOps.handle (hub_guards.go:130-133); that guard invokes WorkspaceReferencePath, repository Discover, and ResolveWorkspace through hubTargetResolver (lines 45-66). A temporary Go overlay enabling guardHubState(t) and hubTargetResolver(r.couch) on the existing fixture passes with its catalog barrier released, but fails with the barrier held: socket recover-park: context deadline exceeded after 2.16s. ARCH-PURPOSE: the documented zero-workspace-probe admission contract is therefore not delivered for configured hubs. The family includes new recover-park admissions and retained duplicate/conflicting-ID requests, all of which traverse this guard before receipt lookup; its workspace IO sites are caller workspace resolution and target resolution/discovery. Preserve caller authentication, derive admission target ownership from the bound cached identity, and retain full queued/dispatch authorization. Extend the transport regression with service-side hub authority enabled and cover retained requests before fresh target lookup.

## Round 3 — 2026-10-18T18:58:40-07:00 (codex) — passed

### Disposed

- BR-2 — addressed — Production-wired transport enables service hub authority and hubTargetResolver. Restoring the pre-admission full guard through a temporary Go overlay makes TestRecoverParkAdmissionPrecedesSlowQueuedPreparation fail with context deadline exceeded after 2.27s; pinned code passes. Retained duplicate/conflict tests also verify no fresh caller/target workspace probes. Queued and dispatch authorization remain enforced.
- BR-1 — addressed — The Core concepts row at workshop/plans/000681-park-recovery-admission-plan.md:45 correctly identifies newParkAdmissionRig in the newly added park_recovery_admission_test.go. The pinned diff and fixture definition support the corrected location/status.

## Open findings

(none — every finding has been disposed)
