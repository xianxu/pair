---
gate: boundary-review
issue: 737
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: beea5dbc-2dca-42f7-ba3d-461ee3c6c965
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/21f3e34c061adac5
            reviewed_head: e5547ba789573abd77c9af33d67c993b8ca5a636
            slot: pair:1
            requested_at: "2026-10-21T03:08:13.834329Z"
          phase: finished
          pid: 8223
          started_at: "2026-10-21T03:08:14.053418Z"
          finished_at: "2026-10-21T03:12:30.290329Z"
          session_id: 01a15c6e-4b34-7e53-be6a-cc43c9dd732c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e5547ba789573abd77c9af33d67c993b8ca5a636
        tracker_head: 5073c42ee30deb9ad7a64fa2d10628b5779d05c7
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-20T20:05:05-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-20T20:05:05-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a396f18becc8a0b14c2509e639e2760c66b7a78e10199c05ce3f79e23fe424c
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 9326362
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 2e937bf4dd9d58cabbedb48b75c23d405af91c28c0e759fd806d60ff77528a80
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 2473885
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: ed4fd9319c53cce5de03301fede9c8ac36422b6ee8f82132441524768c179e36
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 2895127
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 9bf82f7a556bf9600b18c91a96ae9254ffc54810a6273010dad9e4deeb68e7ce
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 2307226
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15c40-c87b-7d42-8375-bfd5ecbd7a53
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: a3b269236061b4495ff4f5eb5c065aa81cd5d6dfbaf9222bc67300f4a25c3bfc
              started: "2026-10-21T02:18:32.36Z"
              spawned: false
              bytes: 731519
              windows:
                - start: "2026-10-21T02:18:32.36Z"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15c43-ba56-74b0-90bf-eba6239eef02
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: c535f95d699a28f090a389e2df5a02d209b9a7922aa84ff5c20e32475399b95b
              started: "2026-10-21T02:21:45.369Z"
              spawned: false
              bytes: 518971
              windows:
                - start: "2026-10-21T02:21:45.369Z"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15c45-809b-7fb3-a021-892d99ec3b1d
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: b261c2a1db7e2401a2a58ce9c12df197749550821c9655313b509884af09b668
              started: "2026-10-21T02:23:41.668Z"
              spawned: false
              bytes: 487844
              windows:
                - start: "2026-10-21T02:23:41.668Z"
                  end: "2026-10-20T20:05:05-07:00"
            - harness: codex
              id: 01a15c45-e52f-7763-8852-144173152199
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: b766e2ab9af6806b5e6afb3df7c118c516bfaf10ab272731bdcc89753d815897
              started: "2026-10-21T02:24:07.329Z"
              spawned: false
              bytes: 520797
              windows:
                - start: "2026-10-21T02:24:07.329Z"
                  end: "2026-10-20T20:05:05-07:00"
      "n": 1
      timestamp: "2026-10-20T20:12:31-07:00"
      dispatched: "2026-10-20T20:08:11-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: One invalid task name makes status and manual commands unusable
          detail: cmd/internal/couchschedule/status.go:86 rejects invalid names even for refused tasks intentionally retained by Catalog.Discover at catalog.go:111. A temporary overlay test discovering BadTask reproduced “invalid schedule status task identity”; runScheduleCLI validates this snapshot before status, history, or manual execution. Represent malformed-entry refusals safely without invalidating the whole snapshot, and test refusal isolation through the CLI/protocol boundary. ARCH-PURPOSE, ARCH-SECURE.
          family: task-refusal-isolation
          round: 1
        - id: BR-2
          severity: Critical
          title: An abandoned cache publication blocks subsequent builds across the namespace
          detail: 'cmd/internal/couchschedule/cache.go:304 creates .publication-* directories whose only cleanup is deferred, while inspectCache at line 137 rejects them as unknown entries. A process crash before rename leaves this production-created shape behind; an overlay test reproduced subsequent Build failure with “unknown cache entry: .publication-123456789”. Add identity-safe recovery under the maintenance lock and sweep interrupted publication, manifest-update, and eviction shapes; test abrupt termination rather than only returned errors. ARCH-ORDER, ARCH-FUNERAL.'
          family: durable-artifact-crash-recovery
          round: 1
        - id: BR-3
          severity: Critical
          title: Accepted persisted phase combinations can panic the supervisor during recovery
          detail: cmd/internal/couchschedule/state_json.go:78 validates stage names independently of source and required companion state. EncodeLedger and DecodeLedger accept a manual notifying attempt with Last=nil; Advance(OwnerUnknown) then dereferences state.Last at model.go:310, reachable from schedule_service.go:135. An overlay test reproduced the nil-pointer panic. Validate legal state combinations at decode and enforce them through transition-owned state updates; malformed storage must visibly block scheduling instead of crashing Couch. ARCH-SECURE, ARCH-ORDER.
          family: persisted-state-invariants
          round: 1
        - id: BR-4
          severity: Critical
          title: The core-concepts inventory does not match the delivered M1 entities
          detail: workshop/plans/000737-couch-scheduler-plan.md:33 places TaskKey in spec.go although it is defined in model.go; lines 35–36 list ScheduleState, Occurrence, and BuildKey, which have no corresponding definitions. Integration rows similarly use TaskCatalog/TaskBuilder instead of Catalog/Builder. Correct the greppable inventory and explicitly distinguish delivered M1 entities from M2 entities in a Revisions entry. This is a plan correction, classified Critical under the requested core-concepts gate. ARCH-PURPOSE.
          family: core-concept-traceability
          round: 1
        - id: BR-5
          severity: Important
          title: New skill examples break the existing command-validity test
          detail: cmd/internal/couchcmd/skills/couch/SKILL.md:86–93 embeds literal optional-argument notation and uppercase TASK placeholders in couch commands. Both the full package run and isolated TestSkillDocumentsRecovery fail because “couch --schedule --status [--json]” does not parse; TASK also violates ValidTaskName. Use runnable examples with a valid task slug, explain optional flags separately, and rerun the existing test. ARCH-PURPOSE.
          family: executable-documentation-validity
          round: 1
        - id: BR-6
          severity: Important
          title: Cache manifests reject additive fields and block both reuse and rebuilding
          detail: 'cmd/internal/couchschedule/cache.go:149 decodes BuildManifest with strictjson.Decode, and maintainCache rewrites it through json.Marshal without unknown-member preservation. An overlay test adding future_metadata to an otherwise valid manifest reproduced “json: unknown field future_metadata” on the next Build. Preserve unknown members at the manifest and input-entry levels while retaining strict owned-field validation, and test older-reader read-modify-write compatibility. ARCH-EVOLVE.'
          family: additive-persisted-format-compatibility
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#737 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T20:12:31-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `task-refusal-isolation` One invalid task name makes status and manual commands unusable
  cmd/internal/couchschedule/status.go:86 rejects invalid names even for refused tasks intentionally retained by Catalog.Discover at catalog.go:111. A temporary overlay test discovering BadTask reproduced “invalid schedule status task identity”; runScheduleCLI validates this snapshot before status, history, or manual execution. Represent malformed-entry refusals safely without invalidating the whole snapshot, and test refusal isolation through the CLI/protocol boundary. ARCH-PURPOSE, ARCH-SECURE.
- **BR-2** [Critical] `durable-artifact-crash-recovery` An abandoned cache publication blocks subsequent builds across the namespace
  cmd/internal/couchschedule/cache.go:304 creates .publication-* directories whose only cleanup is deferred, while inspectCache at line 137 rejects them as unknown entries. A process crash before rename leaves this production-created shape behind; an overlay test reproduced subsequent Build failure with “unknown cache entry: .publication-123456789”. Add identity-safe recovery under the maintenance lock and sweep interrupted publication, manifest-update, and eviction shapes; test abrupt termination rather than only returned errors. ARCH-ORDER, ARCH-FUNERAL.
- **BR-3** [Critical] `persisted-state-invariants` Accepted persisted phase combinations can panic the supervisor during recovery
  cmd/internal/couchschedule/state_json.go:78 validates stage names independently of source and required companion state. EncodeLedger and DecodeLedger accept a manual notifying attempt with Last=nil; Advance(OwnerUnknown) then dereferences state.Last at model.go:310, reachable from schedule_service.go:135. An overlay test reproduced the nil-pointer panic. Validate legal state combinations at decode and enforce them through transition-owned state updates; malformed storage must visibly block scheduling instead of crashing Couch. ARCH-SECURE, ARCH-ORDER.
- **BR-4** [Critical] `core-concept-traceability` The core-concepts inventory does not match the delivered M1 entities
  workshop/plans/000737-couch-scheduler-plan.md:33 places TaskKey in spec.go although it is defined in model.go; lines 35–36 list ScheduleState, Occurrence, and BuildKey, which have no corresponding definitions. Integration rows similarly use TaskCatalog/TaskBuilder instead of Catalog/Builder. Correct the greppable inventory and explicitly distinguish delivered M1 entities from M2 entities in a Revisions entry. This is a plan correction, classified Critical under the requested core-concepts gate. ARCH-PURPOSE.
- **BR-5** [Important] `executable-documentation-validity` New skill examples break the existing command-validity test
  cmd/internal/couchcmd/skills/couch/SKILL.md:86–93 embeds literal optional-argument notation and uppercase TASK placeholders in couch commands. Both the full package run and isolated TestSkillDocumentsRecovery fail because “couch --schedule --status [--json]” does not parse; TASK also violates ValidTaskName. Use runnable examples with a valid task slug, explain optional flags separately, and rerun the existing test. ARCH-PURPOSE.
- **BR-6** [Important] `additive-persisted-format-compatibility` Cache manifests reject additive fields and block both reuse and rebuilding
  cmd/internal/couchschedule/cache.go:149 decodes BuildManifest with strictjson.Decode, and maintainCache rewrites it through json.Marshal without unknown-member preservation. An overlay test adding future_metadata to an otherwise valid manifest reproduced “json: unknown field future_metadata” on the next Build. Preserve unknown members at the manifest and input-entry levels while retaining strict owned-field validation, and test older-reader read-modify-write compatibility. ARCH-EVOLVE.

## Open findings

- **BR-1** [Critical] `task-refusal-isolation` One invalid task name makes status and manual commands unusable
- **BR-2** [Critical] `durable-artifact-crash-recovery` An abandoned cache publication blocks subsequent builds across the namespace
- **BR-3** [Critical] `persisted-state-invariants` Accepted persisted phase combinations can panic the supervisor during recovery
- **BR-4** [Critical] `core-concept-traceability` The core-concepts inventory does not match the delivered M1 entities
- **BR-5** [Important] `executable-documentation-validity` New skill examples break the existing command-validity test
- **BR-6** [Important] `additive-persisted-format-compatibility` Cache manifests reject additive fields and block both reuse and rebuilding
