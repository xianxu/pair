---
gate: boundary-review
issue: 587
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 17675bd2-ffe6-4f8a-a3d4-0071384371a5
            repository: github.com/xianxu/pair
            issue_id: "000587"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/192561dfb76ffa63
            reviewed_head: 0da6c7c91b78394c755bf593d735c5cdef1275cc
            slot: pair:1
            requested_at: "2026-10-16T00:31:58.515171Z"
          phase: finished
          pid: 98119
          started_at: "2026-10-16T00:31:58.613029Z"
          finished_at: "2026-10-16T00:38:20.215669Z"
          session_id: 01a1421f-6f6f-7c32-8689-4af719225b42
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 0da6c7c91b78394c755bf593d735c5cdef1275cc
        tracker_head: 6f59b1d966560e05c0bb3887995ce3fb23c436e5
        branch_point: 99e11ecb8c72b7e5325a58692c1efd679e50710c
        since: "2026-10-15T16:54:15-07:00"
        until: "2026-10-15T17:31:29-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T16:54:15-07:00"
              end: "2026-10-15T17:31:29-07:00"
              began: "2026-10-15T16:54:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a13337-c560-79d0-b42a-1aa85da46014
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: f3da988cdb82020c0acec99b7ff64233d24c58794aecbbfce499ad7f2778b348
              started: "2026-10-13T03:05:03.782Z"
              spawned: false
              bytes: 31931466
              windows:
                - start: "2026-10-15T16:54:15-07:00"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a141fd-16f7-7221-8fb6-b1bee1d65678
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: ec2cbccae9eb05ab4ce9569dfa5125008d5e1b24fc63cf742c52553fd5830190
              started: "2026-10-15T23:54:28.223Z"
              spawned: true
              bytes: 1713561
              windows:
                - start: "2026-10-15T23:54:28.223Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a14204-ca62-74d0-8a42-3e78dea248ce
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 72565e13ca5ec209280660a8798d4a4d88af2aad721b5a6b5fd13c600fc4bf40
              started: "2026-10-16T00:02:53.264Z"
              spawned: false
              bytes: 624471
              windows:
                - start: "2026-10-16T00:02:53.264Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a14206-c1a3-7872-acc2-24437b8e5916
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 9e8585ee70ca53f43ade1a542efa2cbcdfee990ab30ae0623787b409d2aecf49
              started: "2026-10-16T00:05:01.776Z"
              spawned: false
              bytes: 453030
              windows:
                - start: "2026-10-16T00:05:01.776Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a14208-28be-7471-9672-7394cbe9bb2e
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 003f802ad713170d548b6984142fa1d1c8b2cb70f6bb4809e3a484bf6956aad1
              started: "2026-10-16T00:06:33.937Z"
              spawned: false
              bytes: 426772
              windows:
                - start: "2026-10-16T00:06:33.937Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a1420b-4952-73b3-bf95-e4e25f95036e
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 747fae07e1c3a17bdc9c5451bcce13097cfe5badfb53b596234e66081dbfaea2
              started: "2026-10-16T00:09:58.847Z"
              spawned: false
              bytes: 460452
              windows:
                - start: "2026-10-16T00:09:58.847Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a1420b-c507-7382-b464-844cfa8fdf88
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 9866084d14015f829a69e8f9ac028eaa2073fe080dc8fd90817352b4b85ab9a5
              started: "2026-10-16T00:10:30.329Z"
              spawned: false
              bytes: 429106
              windows:
                - start: "2026-10-16T00:10:30.329Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a1420c-5e85-7da0-9ee1-37f6626f0f37
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 9299838b0a0b3dded4619c50bfe3d97482e82d858ccd579d9bec364374c96086
              started: "2026-10-16T00:11:09.407Z"
              spawned: true
              bytes: 1088010
              windows:
                - start: "2026-10-16T00:11:09.407Z"
                  end: "2026-10-15T17:31:29-07:00"
            - harness: codex
              id: 01a1420c-c0bb-7523-a341-15c117477e45
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a13337-c560-79d0-b42a-1aa85da46014
              digest: 49f38e09be4cd3fbc0dcae95b4f14eb106af57d471d58d9000f7a6f56d34251f
              started: "2026-10-16T00:11:34.552Z"
              spawned: true
              bytes: 940148
              windows:
                - start: "2026-10-16T00:11:34.552Z"
                  end: "2026-10-15T17:31:29-07:00"
      "n": 1
      timestamp: "2026-10-15T17:38:28-07:00"
      dispatched: "2026-10-15T17:31:57-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Resource responses exceed the existing socket frame limit within the supported process envelope
          detail: cmd/internal/couchcmd/message_service.go:773 returns the entire ResourceView without a transport byte bound. ResourceMaxProcesses permits 8192 process rows, while couchmessage.MaxFrameBytes is 590848 bytes; a conservative serialization containing only a subset of the required process fields already occupies 1760214 bytes at that count. transport.go:213 rejects the response, so both resources CLI forms fail precisely when a large process fleet needs inspection. Add pagination or an explicitly bounded projection that preserves slot aggregates and discloses omitted details, and test maximum-size views through the real socket. ARCH-CONSTRAINTS.
          family: response-envelope-bounds
          round: 1
        - id: BR-2
          severity: Critical
          title: A corrupt history file permanently disables alerts despite a valid checkpoint
          detail: cmd/internal/couchcore/resource_store.go:32 discards the successfully read checkpoint when history validation fails. cmd/internal/couchcmd/resources_persist.go:64 then returns on the latched loadError on every subsequent pass, with no reload or repair path; neither history nor alerts recover during that supervisor lifetime. This contradicts the plan's fresh-baseline recovery contract. Load the artifacts independently, preserve valid checkpoint suppression state, and rebuild invalid aggregate history with visible continuity loss. Add a regression using valid checkpoint state plus corrupt history that proves subsequent persistence and eligible alert delivery recover without duplicate admission. ARCH-PURPOSE.
          family: snapshot-failure-recovery
          round: 1
        - id: BR-3
          severity: Critical
          title: Incomplete slot inventory is treated as confirmed ownership loss
          detail: cmd/internal/couchcore/resource.go:240 treats a missing slot as known ownership loss. At startup, message_service.go:474 starts inventory discovery concurrently with the immediately sampling worker, so persisted live processes can be reported as proven orphans before discovery finishes. resources_inventory.go:109 and :199 also silently omit slots beyond the cap, hiding incompleteness from the reducer and its overflow diagnostics. Carry explicit inventory completeness and omission evidence into sampling; missing owners must remain unknown until authoritative inventory establishes absence. Add barrier-controlled restart and over-cap inventory regressions. ARCH-ORDER, ARCH-CONSTRAINTS.
          family: ownership-absence-requires-evidence
          round: 1
        - id: BR-4
          severity: Critical
          title: The core-concepts table names a nonexistent platform adapter
          detail: workshop/plans/000587-couch-resource-sampler-plan.md:41 names resource_other.go, but the pinned implementation supplies cmd/internal/procutil/resource_unsupported.go. Under the requested core-concepts gate this table/code contradiction requires correction. Update the table and record the actual fallback path in Revisions; the implemented unsupported capability itself exists.
          family: core-concept-traceability
          round: 1
        - id: BR-5
          severity: Important
          title: Production alert admission bypasses the pure episode transition model
          detail: cmd/internal/couchcmd/resources_alert.go:145, :152 and :172 assign episode phases directly. The Admitted event in couchcore.AdvanceResourceEpisode is exercised only by tests, so production admission does not pass through the model those tests validate. Move delivery-outcome transitions into the pure model, have the dispatcher return outcomes through that boundary, and prevent direct authoritative phase mutation. Cover admission, refusal, uncertainty and restart reconciliation through the production transition path. ARCH-ORDER, ARCH-PURE, ARCH-DRY.
          family: authoritative-state-transition-boundary
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#587 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T17:38:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `response-envelope-bounds` Resource responses exceed the existing socket frame limit within the supported process envelope
  cmd/internal/couchcmd/message_service.go:773 returns the entire ResourceView without a transport byte bound. ResourceMaxProcesses permits 8192 process rows, while couchmessage.MaxFrameBytes is 590848 bytes; a conservative serialization containing only a subset of the required process fields already occupies 1760214 bytes at that count. transport.go:213 rejects the response, so both resources CLI forms fail precisely when a large process fleet needs inspection. Add pagination or an explicitly bounded projection that preserves slot aggregates and discloses omitted details, and test maximum-size views through the real socket. ARCH-CONSTRAINTS.
- **BR-2** [Critical] `snapshot-failure-recovery` A corrupt history file permanently disables alerts despite a valid checkpoint
  cmd/internal/couchcore/resource_store.go:32 discards the successfully read checkpoint when history validation fails. cmd/internal/couchcmd/resources_persist.go:64 then returns on the latched loadError on every subsequent pass, with no reload or repair path; neither history nor alerts recover during that supervisor lifetime. This contradicts the plan's fresh-baseline recovery contract. Load the artifacts independently, preserve valid checkpoint suppression state, and rebuild invalid aggregate history with visible continuity loss. Add a regression using valid checkpoint state plus corrupt history that proves subsequent persistence and eligible alert delivery recover without duplicate admission. ARCH-PURPOSE.
- **BR-3** [Critical] `ownership-absence-requires-evidence` Incomplete slot inventory is treated as confirmed ownership loss
  cmd/internal/couchcore/resource.go:240 treats a missing slot as known ownership loss. At startup, message_service.go:474 starts inventory discovery concurrently with the immediately sampling worker, so persisted live processes can be reported as proven orphans before discovery finishes. resources_inventory.go:109 and :199 also silently omit slots beyond the cap, hiding incompleteness from the reducer and its overflow diagnostics. Carry explicit inventory completeness and omission evidence into sampling; missing owners must remain unknown until authoritative inventory establishes absence. Add barrier-controlled restart and over-cap inventory regressions. ARCH-ORDER, ARCH-CONSTRAINTS.
- **BR-4** [Critical] `core-concept-traceability` The core-concepts table names a nonexistent platform adapter
  workshop/plans/000587-couch-resource-sampler-plan.md:41 names resource_other.go, but the pinned implementation supplies cmd/internal/procutil/resource_unsupported.go. Under the requested core-concepts gate this table/code contradiction requires correction. Update the table and record the actual fallback path in Revisions; the implemented unsupported capability itself exists.
- **BR-5** [Important] `authoritative-state-transition-boundary` Production alert admission bypasses the pure episode transition model
  cmd/internal/couchcmd/resources_alert.go:145, :152 and :172 assign episode phases directly. The Admitted event in couchcore.AdvanceResourceEpisode is exercised only by tests, so production admission does not pass through the model those tests validate. Move delivery-outcome transitions into the pure model, have the dispatcher return outcomes through that boundary, and prevent direct authoritative phase mutation. Cover admission, refusal, uncertainty and restart reconciliation through the production transition path. ARCH-ORDER, ARCH-PURE, ARCH-DRY.

## Open findings

- **BR-1** [Critical] `response-envelope-bounds` Resource responses exceed the existing socket frame limit within the supported process envelope
- **BR-2** [Critical] `snapshot-failure-recovery` A corrupt history file permanently disables alerts despite a valid checkpoint
- **BR-3** [Critical] `ownership-absence-requires-evidence` Incomplete slot inventory is treated as confirmed ownership loss
- **BR-4** [Critical] `core-concept-traceability` The core-concepts table names a nonexistent platform adapter
- **BR-5** [Important] `authoritative-state-transition-boundary` Production alert admission bypasses the pure episode transition model
