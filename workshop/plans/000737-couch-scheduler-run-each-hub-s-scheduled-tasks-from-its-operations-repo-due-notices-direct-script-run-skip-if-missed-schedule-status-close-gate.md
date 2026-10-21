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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: bce4ef5d-9dfc-49ed-9f85-e4c114d7300e
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/f599858499d90cc7
            reviewed_head: e34a33888d9441e83bc815b5a8f4f6ddcf28d979
            slot: pair:1
            requested_at: "2026-10-21T03:36:38.96183Z"
          phase: finished
          pid: 65864
          started_at: "2026-10-21T03:36:39.154577Z"
          finished_at: "2026-10-21T03:40:38.048196Z"
          session_id: 01a15c88-4f9b-72f2-ae3a-e3f25dcac1a4
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e34a33888d9441e83bc815b5a8f4f6ddcf28d979
        tracker_head: c8cda69ed09dbb3642d80d6393fe40637fd184a5
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-20T20:35:29-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-20T20:35:29-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: db0f6dc181d70724d46705f2e65b91f8f7c0381dcd3ab53fe7053f3862324857
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 11591960
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-20T20:35:29-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 2383b96e4f38c6623aa4660b5b4164d9719d5c03dd473db3660d6519bd2d27af
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 3239041
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-20T20:35:29-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 8094b48b6cd0b9fec9bea82194dece9f6ef73c5e1acc79ea1f0bc711a41cc76b
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 3688433
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-20T20:35:29-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 47d0ceba06f58ebf40e6c061354f31a29514158dcedb881b60e2e50356ab8576
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 2972930
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-20T20:35:29-07:00"
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
                  end: "2026-10-20T20:35:29-07:00"
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
                  end: "2026-10-20T20:35:29-07:00"
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
                  end: "2026-10-20T20:35:29-07:00"
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
                  end: "2026-10-20T20:35:29-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-20T20:40:39-07:00"
      dispatched: "2026-10-20T20:36:36-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Socket/CLI refusal-isolation and valid-manual-run regressions pass. Disabling diagnostic-entry validation support makes TestScheduleMixedRefusalStatusAndHistoryThroughSocket fail with invalid schedule status task identity.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Cache recovery covers publication, legacy staging, touch and eviction interruptions with abrupt subprocess exits. Disabling recoverCacheLocked makes TestCacheAbruptMutationRecovery/publication-task-written fail on retained aged residue. The new finding concerns staging outside this repaired cache path.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Encode/decode and Advance share phase validation. Source/stage/companion regressions pass; removing persistence-boundary validation makes TestStateJSONRefusesInconsistentCompanions/notice_missing_summary fail.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Inspected the before/after inventory and source declarations. The Core concepts tables now name TaskKey in model.go, Catalog, Builder and scheduleCoordinator, and explicitly label scheduleNotice and FinalizeNotice as planned M2. The Revisions passage records that distinction.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Skill examples now use runnable landing-plan commands and explain optional flags separately. TestSkillDocumentsRecovery passes at head; overlaying the previous skill makes it fail on couch --schedule --status [--json].
          round: 2
        - id: BR-6
          disposition: addressed
          note: Manifest and input codecs preserve unknown members through reuse and rebuilding. TestCacheFutureManifestMetadataSurvivesReuseAndRebuild passes; restoring unknown-member rejection makes the regression fail.
          round: 2
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: Build captures and ledger staging still have only deferred cleanup
          detail: cmd/internal/couchschedule/build.go:86 creates couch-build-* directories outside the managed cache, and store_unix.go:132 creates .state-*.tmp snapshots; both rely solely on deferred removal. Temporary overlay subprocess tests exiting abruptly reproduced both abandoned shapes, and a subsequent successful StateStore.Update left the old snapshot untouched. This is the 2nd finding in family durable-artifact-crash-recovery, with two remaining staging families found. State the rule that every scheduler-owned staging artifact needs owner-safe crash cleanup, enumerate all staging writers, and sweep the class in this round. Add identity-fenced recovery under the appropriate locks and abrupt-death regressions; build captures and repeated ledger residues otherwise accumulate outside the declared retention bounds. ARCH-FUNERAL, ARCH-PURPOSE.
          family: durable-artifact-crash-recovery
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b74f92ea-7f5f-4db0-bfe8-5441c44b7545
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/7b9140d58209a636
            reviewed_head: 2a71a0fad074c43b370b1d8a66bfdfadfe97c17e
            slot: pair:1
            requested_at: "2026-10-21T04:10:13.606693Z"
          phase: finished
          pid: 30358
          started_at: "2026-10-21T04:10:13.708019Z"
          finished_at: "2026-10-21T04:12:58.646653Z"
          session_id: 01a15ca7-0c4a-7b32-9494-0b30657baaca
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2a71a0fad074c43b370b1d8a66bfdfadfe97c17e
        tracker_head: a5e16637f11beadfef44c626fd594baffa32997a
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-20T21:09:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-20T21:09:28-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: e9905ab495725773a7d067e6efae1e7b652d2f0caebcd8e2b29d4ca8b39a5da3
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 13339568
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-20T21:09:28-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 84f6a41df52983e16bee241ca605b481f13aa5df2904241564e3200f4a36856b
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 4382534
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-20T21:09:28-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 919ee469d1e3a35e14406d8df492331353327481c60693bcf1be422065040cbc
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 4998826
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-20T21:09:28-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 4d2222da17d24cc7612d70ad585754b0d70b27cbd03c04adff1c93e5c17e440d
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 3483876
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-20T21:09:28-07:00"
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
                  end: "2026-10-20T21:09:28-07:00"
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
                  end: "2026-10-20T21:09:28-07:00"
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
                  end: "2026-10-20T21:09:28-07:00"
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
                  end: "2026-10-20T21:09:28-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T20:36:36-07:00"
              finalized: "2026-10-20T20:40:39-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-20T21:12:59-07:00"
      dispatched: "2026-10-20T21:10:12-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-7
          disposition: addressed
          note: Build captures now use owner-held slots, ledger residues recover under the state lock, and operational collection staging recovers under the task lock. The native crash tests pass; independently disabling capture, ledger, and collection recovery in a disposable pinned-head copy makes the corresponding regressions fail.
          round: 3
      findings:
        - kind: defect
          id: BR-8
          severity: Critical
          title: Ledger publication promotes a substituted staging inode into authoritative state
          detail: 'cmd/internal/couchschedule/store_unix.go:169 renames the temporary pathname without checking that it still identifies the file originally opened by publishSchedule. A disposable pinned-head regression replaced that pathname at the existing pre-rename hook: StateStore.Update returned nil and state.json contained the replacement''s empty tasks map instead of the intended ledger. This can discard retained execution authority. This is the 3rd finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; enforce the rule that every staging publication and deletion must verify its captured ownership identity, and sweep the complete writer inventory. Add a descriptor-relative identity check before ledger rename and a regression asserting refusal, unchanged prior authority, and preservation of the replacement. ARCH-SECURE, ARCH-PURPOSE.'
          family: durable-artifact-crash-recovery
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1c842a53-f8bc-4def-8be4-8e625598d203
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/8588b74eadfc1e4e
            reviewed_head: 10788616469fcb3405b066c24f6397e22dc87732
            slot: pair:1
            requested_at: "2026-10-21T04:21:35.420948Z"
          phase: finished
          pid: 88761
          started_at: "2026-10-21T04:21:35.582052Z"
          finished_at: "2026-10-21T04:23:47.199708Z"
          session_id: 01a15cb1-7559-7242-bb41-a279882c2195
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 10788616469fcb3405b066c24f6397e22dc87732
        tracker_head: 4331b07c27022f410e5276ce430b614cb63eba56
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-20T21:20:51-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-20T21:20:51-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5fb0393b4b24005f431daeb378254c32a5e5e03d29383aed2c42a4fc769179c3
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 14704548
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-20T21:20:51-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 84f6a41df52983e16bee241ca605b481f13aa5df2904241564e3200f4a36856b
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 4382534
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-20T21:20:51-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 919ee469d1e3a35e14406d8df492331353327481c60693bcf1be422065040cbc
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 4998826
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-20T21:20:51-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 4d2222da17d24cc7612d70ad585754b0d70b27cbd03c04adff1c93e5c17e440d
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 3483876
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-20T21:20:51-07:00"
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
                  end: "2026-10-20T21:20:51-07:00"
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
                  end: "2026-10-20T21:20:51-07:00"
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
                  end: "2026-10-20T21:20:51-07:00"
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
                  end: "2026-10-20T21:20:51-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T20:36:36-07:00"
              finalized: "2026-10-20T20:40:39-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:10:12-07:00"
              finalized: "2026-10-20T21:12:59-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-20T21:23:47-07:00"
      dispatched: "2026-10-20T21:21:32-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-8
          disposition: addressed
          note: store_unix.go:169 checks captured staging identity immediately before Renameat, sharing the predicate with error cleanup. TestLedgerPublicationRefusesReplacement at store_recovery_test.go:374 passes on head; removing that guard in a disposable copy fails refusal, unchanged prior authority, and replacement preservation assertions. Ledger/Store race tests pass. The disposable copy was removed.
          round: 4
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 10788616469fcb3405b066c24f6397e22dc87732
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: bc0371b0-1975-48a7-b8fc-e851b329c8f4
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/4cd79521962a6ba7
            reviewed_head: 11db21abcaa8e29ad26e7914f8c6ecf86aec6d68
            slot: pair:1
            requested_at: "2026-10-21T07:01:06.065439Z"
          phase: finished
          pid: 25993
          started_at: "2026-10-21T07:01:06.294104Z"
          finished_at: "2026-10-21T07:07:18.426758Z"
          session_id: 01a15d43-7dda-7903-a57d-6d47ed804ece
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 11db21abcaa8e29ad26e7914f8c6ecf86aec6d68
        tracker_head: 8ad72e59f205333629047a953ea2e6eaa42cefe1
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-21T00:00:01-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-21T00:00:01-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 53aafcca6ff90d876eab50f4fca2749e7172390f8ee1afe087431a9242a53131
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 19032561
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-21T00:00:01-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 56cb851cc0c000731264aea96f8cdb07ca78ed6fd637908f32e482409f1ea486
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 6217909
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-21T00:00:01-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: c5a168bbfda1a53fd7eda6ec03ea3ecc08dca8852bda4f1868992570919a4e85
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 6803546
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-21T00:00:01-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: e3452498ecaeb9c010124a71304f43b0f3a8abbd469e3eca0f7dcebd2bb76bdd
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 4829922
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-21T00:00:01-07:00"
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
                  end: "2026-10-21T00:00:01-07:00"
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
                  end: "2026-10-21T00:00:01-07:00"
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
                  end: "2026-10-21T00:00:01-07:00"
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
                  end: "2026-10-21T00:00:01-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T20:36:36-07:00"
              finalized: "2026-10-20T20:40:39-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:10:12-07:00"
              finalized: "2026-10-20T21:12:59-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:21:32-07:00"
              finalized: "2026-10-20T21:23:47-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-21T00:07:19-07:00"
      dispatched: "2026-10-21T00:01:03-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-9
          severity: Critical
          title: Notice admission can proceed after its deadline or committed cancellation
          detail: 'cmd/internal/couchcmd/schedule_automatic.go:650-654 checks authority before calling Admit, while schedule_notice.go:45 then waits for holdHubEffect using the service context. There is no persisted-deadline cancellation or task-authority recheck after that wait. Two deterministic probes using the production scheduler, real broker, held ownership gate, and injected clock reproduced broker admission after NoticeDeadline and after catalogue removal had committed, finalized history, and cleared Active. ARCH-ORDER and ARCH-CONSTRAINTS: enforce one rule across queued and blocked effects—admission must remain authorized until the actual broker boundary. Bound gate/admission work by the persisted deadline and serialize or cancel pending admission against declaration cancellation, preserving uncertainty for already accepted effects. Add both interleavings as permanent regression tests and return this ordering gap to the TL under the approved transaction revision.'
          family: effect-authorization-fencing
          round: 5
        - kind: defect
          id: BR-10
          severity: Critical
          title: One damaged task history prevents every scheduler worker from starting
          detail: 'cmd/internal/couchcmd/schedule_automatic.go:151-153 returns a task-local passive-history error from the entire recovery transaction; run starts workers only after recovery succeeds. A probe created offline-skip history for two hubs, truncated one task''s record.json, and reopened the scheduler: recover returned the JSON error, preventing all workers from starting. Similar global propagation remains in scheduled completion and notice-history paths. This is the 2nd finding in family task-refusal-isolation. Earlier corrections handled instances; do not patch only recover. State and enforce the rule that task-history failures preserve that task''s authority and expose a local refusal, while shared authority-store failures block globally. Enumerate startup recovery, passive repair, completion reconciliation, cancellation finalization, and notice finalization; sweep those consumers and test that a healthy second hub continues. ARCH-PURPOSE.'
          family: task-refusal-isolation
          round: 5
        - kind: defect
          id: BR-11
          severity: Critical
          title: Same-occurrence recovery silently deletes unknown summary fields
          detail: 'cmd/internal/couchschedule/model.go:235 and :260 replace Last with the recovered summary, while cmd/internal/couchcmd/schedule_records.go:14-15 constructs that summary without the existing Last.Unknown members. A persisted scheduled-delivery summary containing future_summary_field={"value":42} lost that field after automaticScheduler.recover processed the same occurrence. Codec preservation tests do not cover this rewrite. This is the 2nd finding in family additive-persisted-format-compatibility. Do not fix only this caller: enforce preservation of unknown members whenever an existing persisted object is updated for the same identity, keeping distinct schema namespaces separate. Sweep manual, scheduled, passive, and uncertain canonical-adoption paths, with reopen-and-rewrite regressions. ARCH-EVOLVE.'
          family: additive-persisted-format-compatibility
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d5538866-60e8-4bb3-84a5-4ffa0b7360fe
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/f4da20a12c709819
            reviewed_head: 8e26782e53cbabcf4eb7c86d81803045582dc4c0
            slot: pair:1
            requested_at: "2026-10-21T15:48:31.12272Z"
          phase: finished
          pid: 68627
          started_at: "2026-10-21T15:48:31.224053Z"
          finished_at: "2026-10-21T15:53:10.190743Z"
          session_id: 01a15f26-5a7b-7ac3-aea5-4a432178e00d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 8e26782e53cbabcf4eb7c86d81803045582dc4c0
        tracker_head: 54bd775ea278d1bb2427b420f998b19134cc3fce
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-21T08:47:29-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-21T08:47:29-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 6410f914ba87b4de52933ce4e09be3bbfba7fb38d3ed3d42425714389d9d2bc2
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 20289941
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-21T08:47:29-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 56cb851cc0c000731264aea96f8cdb07ca78ed6fd637908f32e482409f1ea486
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 6217909
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-21T08:47:29-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: c5a168bbfda1a53fd7eda6ec03ea3ecc08dca8852bda4f1868992570919a4e85
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 6803546
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-21T08:47:29-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: e3452498ecaeb9c010124a71304f43b0f3a8abbd469e3eca0f7dcebd2bb76bdd
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 4829922
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-21T08:47:29-07:00"
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
                  end: "2026-10-21T08:47:29-07:00"
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
                  end: "2026-10-21T08:47:29-07:00"
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
                  end: "2026-10-21T08:47:29-07:00"
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
                  end: "2026-10-21T08:47:29-07:00"
            - harness: codex
              id: 01a15d4c-2f70-7da2-8ce7-e8b02379a07a
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 0984d57d1df7801854c004570922c87fb4f68469dfc5a8622686fa9b5e7dd98f
              started: "2026-10-21T07:10:36.509Z"
              spawned: true
              bytes: 745859
              windows:
                - start: "2026-10-21T07:10:36.509Z"
                  end: "2026-10-21T08:47:29-07:00"
            - harness: codex
              id: 01a15d4c-6139-7bf1-b67d-d736f64e6185
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: ccecb51b8872a94a020713567a54630ef240d9653610185a1f3d7e6c17900c63
              started: "2026-10-21T07:10:49.26Z"
              spawned: true
              bytes: 1130310
              windows:
                - start: "2026-10-21T07:10:49.26Z"
                  end: "2026-10-21T08:47:29-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T20:36:36-07:00"
              finalized: "2026-10-20T20:40:39-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:10:12-07:00"
              finalized: "2026-10-20T21:12:59-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:21:32-07:00"
              finalized: "2026-10-20T21:23:47-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-21T00:01:03-07:00"
              finalized: "2026-10-21T00:07:19-07:00"
              finalizing: false
      "n": 6
      timestamp: "2026-10-21T08:53:10-07:00"
      dispatched: "2026-10-21T08:48:29-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-9
          disposition: addressed
          note: Held-gate deadline and cancellation regressions pass and fail when the admission fence is removed; additional tests cover deadline-bounded gate and broker waits.
          round: 6
        - id: BR-10
          disposition: addressed
          note: Seven history-consumer isolation cases pass and fail without local quarantine; manual refusal, pending-evidence retry, and shared-store failure tests also pass.
          round: 6
        - id: BR-11
          disposition: addressed
          note: Manual and scheduled recovery regressions pass and lose persisted summary metadata when the central same-identity merge is removed; broader rewrite tests cover passive and uncertain paths.
          round: 6
      findings:
        - kind: defect
          id: BR-12
          severity: Critical
          title: Cached canonical recovery is ignored after a failed authority commit
          detail: 'cmd/internal/couchcmd/schedule_automatic.go:250 constructs ResultRecovered without At, so model.go:162-163 silently ignores it. A production-path overlay probe published canonical passive history with different timestamps, failed the following ledger commit, and retried: the ledger retained EndedAt=12:00:00 instead of canonical 11:59:50. ARCH-ORDER: cached and freshly read evidence must produce equivalent recovery transitions, including after failed persistence. This is the 4th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; state and enforce that rule across cached/direct adoption and failed-commit retries, rather than patching only this call. Supply the canonical event time and add a permanent regression asserting the reopened ledger matches canonical history.'
          family: durable-artifact-crash-recovery
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a0fddd69-28c0-484d-95d6-6e386f230e12
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9f972f0a3a987cb9
            reviewed_head: e72bfd54f67bed8dd8b2947281286bfa61fb008e
            slot: pair:1
            requested_at: "2026-10-21T16:01:26.215877Z"
          phase: finished
          pid: 48930
          started_at: "2026-10-21T16:01:26.350844Z"
          finished_at: "2026-10-21T16:06:34.248954Z"
          session_id: 01a15f32-2ea4-72d3-a489-82499f39ec34
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e72bfd54f67bed8dd8b2947281286bfa61fb008e
        tracker_head: 0bea86a43ab871e67ded085e0ef7244baeacdeb9
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-21T09:00:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-21T09:00:10-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 90e4c9c496d04b98413dc03a80c76c60c190e3639c952b61dbc406cd07078ebe
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 20823778
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-21T09:00:10-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 56cb851cc0c000731264aea96f8cdb07ca78ed6fd637908f32e482409f1ea486
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 6217909
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-21T09:00:10-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: c5a168bbfda1a53fd7eda6ec03ea3ecc08dca8852bda4f1868992570919a4e85
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 6803546
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-21T09:00:10-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: e3452498ecaeb9c010124a71304f43b0f3a8abbd469e3eca0f7dcebd2bb76bdd
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 4829922
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-21T09:00:10-07:00"
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
                  end: "2026-10-21T09:00:10-07:00"
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
                  end: "2026-10-21T09:00:10-07:00"
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
                  end: "2026-10-21T09:00:10-07:00"
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
                  end: "2026-10-21T09:00:10-07:00"
            - harness: codex
              id: 01a15d4c-2f70-7da2-8ce7-e8b02379a07a
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 0984d57d1df7801854c004570922c87fb4f68469dfc5a8622686fa9b5e7dd98f
              started: "2026-10-21T07:10:36.509Z"
              spawned: true
              bytes: 745859
              windows:
                - start: "2026-10-21T07:10:36.509Z"
                  end: "2026-10-21T09:00:10-07:00"
            - harness: codex
              id: 01a15d4c-6139-7bf1-b67d-d736f64e6185
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: ccecb51b8872a94a020713567a54630ef240d9653610185a1f3d7e6c17900c63
              started: "2026-10-21T07:10:49.26Z"
              spawned: true
              bytes: 1130310
              windows:
                - start: "2026-10-21T07:10:49.26Z"
                  end: "2026-10-21T09:00:10-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T20:36:36-07:00"
              finalized: "2026-10-20T20:40:39-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:10:12-07:00"
              finalized: "2026-10-20T21:12:59-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:21:32-07:00"
              finalized: "2026-10-20T21:23:47-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-21T00:01:03-07:00"
              finalized: "2026-10-21T00:07:19-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-21T08:48:29-07:00"
              finalized: "2026-10-21T08:53:10-07:00"
              finalizing: false
      "n": 7
      timestamp: "2026-10-21T09:06:35-07:00"
      dispatched: "2026-10-21T09:01:23-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-12
          disposition: addressed
          note: TestScheduleCachedCanonicalRecoveryAfterFailedCommit passes at head and fails for skipped, refused, and completed outcomes with the old missing-At producer restored through a disposable overlay. All production recovery producers use NewRecoveryEvent; direct and cached scheduled adoption share adoptScheduledSummary.
          round: 7
        - id: BR-9
          disposition: addressed
          note: Admission retains authority through bounded broker admission after acquiring the hub gate. Deadline/cancellation regression tests pass under race.
          round: 7
        - id: BR-10
          disposition: addressed
          note: Shared quarantine handling preserves task authority and isolates history failures. Startup, consumer-isolation, and pending-evidence regressions pass under race.
          round: 7
        - id: BR-11
          disposition: addressed
          note: Same-ID reducer rewrites preserve summary unknown members separately from operational-record metadata. Codec, consumer, and detached-snapshot regressions pass under race.
          round: 7
      findings:
        - kind: defect
          id: BR-13
          severity: Critical
          title: Recovery clears authority using history whose durability remains unconfirmed
          detail: 'schedule_automatic.go:190 reads history through Store.Read and adopts it at line 213 without completing publication durability. A disposable regression injected directory-sync failure after bundle rename: Publish returned an error, the bundle remained readable, and startup recovery nevertheless cleared Active while the same sync still failed. The same regression failed through schedule_manual.go:173 and schedule_service.go:102. The notify-only adoption path at schedule_automatic.go:468 also bypasses durable confirmation. This violates the history-before-authority-release contract and can leave committed completion authority referring to output whose publication was never confirmed durable (ARCH-ORDER, ARCH-PURPOSE). This is the 5th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; do NOT fix only this instance. Enforce one rule across every adoption consumer: readable history becomes authoritative only after its publication/finalization durability is confirmed under the task lock. Reuse a shared durable adoption boundary, retain authority on failure, and add permanent regressions across the consumer inventory.'
          family: durable-artifact-crash-recovery
          round: 7
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8395756c-448f-4d66-b843-07bbdbe3ce8e
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/8e89470ef8f200df
            reviewed_head: c999468484f4832f68db1d3ac41d639a3fd66e5b
            slot: pair:1
            requested_at: "2026-10-21T16:41:30.991914Z"
          phase: finished
          pid: 24795
          started_at: "2026-10-21T16:41:31.090011Z"
          finished_at: "2026-10-21T17:31:24.711052Z"
          session_id: 01a15f56-dfdb-7f01-9f8f-741d4a6d77df
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c999468484f4832f68db1d3ac41d639a3fd66e5b
        tracker_head: 89d7b94a0f4c394efad33f83b72afceb04d6dbcf
        branch_point: 2d298efea85f7a623d0a8d4a65b3260e0c626f18
        since: "2026-10-20T17:17:42-07:00"
        until: "2026-10-21T09:40:21-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T17:17:42-07:00"
              end: "2026-10-21T09:40:21-07:00"
              began: "2026-10-20T17:17:42-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 8f6eef4fd9e9f065e35fed2fd47ba1e1844f4e7d804d68f06bc4f944298f9792
              started: "2026-10-20T17:14:55.765Z"
              spawned: false
              bytes: 21918693
              windows:
                - start: "2026-10-20T17:17:42-07:00"
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15bd2-f5c9-7251-9c7e-cba357006b38
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 56cb851cc0c000731264aea96f8cdb07ca78ed6fd637908f32e482409f1ea486
              started: "2026-10-21T00:18:34.683Z"
              spawned: true
              bytes: 6217909
              windows:
                - start: "2026-10-21T00:18:34.683Z"
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15bd3-91ad-7770-8227-419b821d5986
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: c5a168bbfda1a53fd7eda6ec03ea3ecc08dca8852bda4f1868992570919a4e85
              started: "2026-10-21T00:19:14.859Z"
              spawned: true
              bytes: 6803546
              windows:
                - start: "2026-10-21T00:19:14.859Z"
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15be0-4dec-70a0-a17e-61b630b10dca
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: e3452498ecaeb9c010124a71304f43b0f3a8abbd469e3eca0f7dcebd2bb76bdd
              started: "2026-10-21T00:33:09.272Z"
              spawned: true
              bytes: 4829922
              windows:
                - start: "2026-10-21T00:33:09.272Z"
                  end: "2026-10-21T09:40:21-07:00"
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
                  end: "2026-10-21T09:40:21-07:00"
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
                  end: "2026-10-21T09:40:21-07:00"
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
                  end: "2026-10-21T09:40:21-07:00"
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
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15d4c-2f70-7da2-8ce7-e8b02379a07a
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 0984d57d1df7801854c004570922c87fb4f68469dfc5a8622686fa9b5e7dd98f
              started: "2026-10-21T07:10:36.509Z"
              spawned: true
              bytes: 745859
              windows:
                - start: "2026-10-21T07:10:36.509Z"
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15d4c-6139-7bf1-b67d-d736f64e6185
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: ccecb51b8872a94a020713567a54630ef240d9653610185a1f3d7e6c17900c63
              started: "2026-10-21T07:10:49.26Z"
              spawned: true
              bytes: 1130310
              windows:
                - start: "2026-10-21T07:10:49.26Z"
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15f3b-cf13-78c2-84b2-ffd459d211cb
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: 06163cd1b5205c1e4371453ec1f5a28a522def4658906c7d87d83fbde3881e21
              started: "2026-10-21T16:11:57.726Z"
              spawned: true
              bytes: 582808
              windows:
                - start: "2026-10-21T16:11:57.726Z"
                  end: "2026-10-21T09:40:21-07:00"
            - harness: codex
              id: 01a15f3c-8f28-7152-948c-0e75b1fe25d1
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a15a4d-a5f6-7e32-9f09-082ad1e172f0
              digest: ea3a01494b537b744358fa9c14dc0c504f67abe66db9b2806c7521d512ef4984
              started: "2026-10-21T16:12:46.936Z"
              spawned: true
              bytes: 865866
              windows:
                - start: "2026-10-21T16:12:46.936Z"
                  end: "2026-10-21T09:40:21-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T20:08:11-07:00"
              finalized: "2026-10-20T20:12:31-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T20:36:36-07:00"
              finalized: "2026-10-20T20:40:39-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:10:12-07:00"
              finalized: "2026-10-20T21:12:59-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T21:21:32-07:00"
              finalized: "2026-10-20T21:23:47-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-21T00:01:03-07:00"
              finalized: "2026-10-21T00:07:19-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-21T08:48:29-07:00"
              finalized: "2026-10-21T08:53:10-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-21T09:01:23-07:00"
              finalized: "2026-10-21T09:06:35-07:00"
              finalizing: false
      "n": 8
      timestamp: "2026-10-21T10:31:26-07:00"
      dispatched: "2026-10-21T09:41:29-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-13
          disposition: addressed
          note: ReadDurable shares the task-locked read/validation/sync boundary; authority consumers use it, and finalization confirms directory durability. TestScheduleDurableAdoptionConsumerInventory passes at HEAD. A disposable Go overlay replacing ReadDurable with Read makes the consumer regression fail, including startup, completion, manual recovery, reserved execution and notify adoption.
          round: 8
        - id: BR-9
          disposition: addressed
          note: 'Retained disposition verified: admission holds authority after acquiring the hub gate, rechecks the persisted deadline and tuple, and bounds broker waits. Admission regressions pass, including under race detection.'
          round: 8
        - id: BR-10
          disposition: addressed
          note: 'Retained disposition verified: history failures quarantine the named task while preserving authority; consumer-isolation and healthy-peer dispatch tests pass.'
          round: 8
        - id: BR-11
          disposition: addressed
          note: 'Retained disposition verified: Advance preserves same-occurrence summary metadata, and recovery keeps record and summary namespaces separate. Preservation regressions pass.'
          round: 8
        - id: BR-12
          disposition: addressed
          note: 'Retained disposition verified: direct and cached adoption share timestamped recovery construction. The failed-commit canonical recovery regression passes.'
          round: 8
      findings:
        - kind: defect
          id: BR-14
          severity: Critical
          title: Catalogue cancellation strands a notice whose terminal history already committed
          detail: 'cmd/internal/couchcmd/schedule_service.go:258-262 attempts dropped finalization without first adopting existing terminal history. A disposable production-path regression lets receipt polling finalize submitted, injects BeforePersist failure, then removes or revises the declaration before retry. Both cases remain stuck after three refresh/step cycles: FinalizeNotice returns "operational notice is not pending", Active remains delivery, Last remains pending, and task quarantine prevents further reconciliation at schedule_automatic.go:451-453. This is the 6th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; do NOT fix only this instance. Enforce the rule that durably confirmed canonical terminal history wins before any cancellation or replacement transition. Sweep catalogue removal/revision/owner changes and every competing finalization consumer, preserve terminal facts, and add sequence regressions for failed authority commits followed by cancellation. ARCH-ORDER, ARCH-PURPOSE.'
          family: durable-artifact-crash-recovery
          round: 8
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a5e8c7b1-7912-45bd-ad5e-a2ca9706713d
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/014ee292d49e2cff
            reviewed_head: b7b6ebb2335e030df26295135431a5c5bd3b9a48
            slot: pair:1
            requested_at: "2026-10-21T21:05:14.151845Z"
          phase: finished
          pid: 66135
          started_at: "2026-10-21T21:05:14.321707Z"
          finished_at: "2026-10-21T21:10:08.785125Z"
          session_id: 01a16048-51fd-7f62-ad04-cc8dff90fd6f
          outcome: returned
      "n": 9
      timestamp: "2026-10-21T14:10:10-07:00"
      dispatched: "2026-10-21T14:05:10-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-15
          severity: Critical
          title: Pending catalogue edits strand undispatched reservations indefinitely
          detail: 'cmd/internal/couchcmd/schedule_owner_dispatch.go:132-135 skips reserved work whenever o.busy is set, while schedule_owner_batch.go:236-243 provides no deadline settlement for those reservations. With both workers occupied, reserve another task and edit its declaration before dispatch: refresh marks it busy, but neither subsequent ticks nor refreshes can settle it, even after its deadline. A temporary test overlay reproduced this in all six scheduled cases: script/notify-only crossed with revision/removal/owner-policy edits. ARCH-ORDER, ARCH-CONSTRAINTS and ARCH-PURPOSE: this is the 2nd finding in family `persisted-state-invariants`. Do NOT fix only this instance; establish the rule that every undispatched reservation has a bounded settlement path independent of dispatch eligibility, and sweep manual, scripted and notify-only reservations across edit, expiry and cancellation sequences. Preserve canonical-evidence precedence and settle known-unstarted work through the pure transition model before applying the edit.'
          family: persisted-state-invariants
          round: 9
        - kind: defect
          id: BR-16
          severity: Critical
          title: The replacement core-concepts table mislocates StateOwner
          detail: 'workshop/plans/000737-couch-scheduler-plan.md:634 lists StateOwner in cmd/internal/couchschedule/{store,store_unix}.go, but neither file declares it; the type and its capability methods are implemented in cmd/internal/couchschedule/owner.go:13 onward. This contradicts the required core-concepts cross-check. ARCH-PURPOSE: this is the 2nd finding in family `core-concept-traceability`. Reconcile every active replacement-table row against its production declaration and mark superseded entries explicitly, rather than repairing only this row. Record the corrected implementation map in a Revisions entry.'
          family: core-concept-traceability
          round: 9
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 96709ecd-e829-4c93-bd0c-2b2c49841dda
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a4851ab06959ac89
            reviewed_head: e7044d90e3d1cda81cc0b99b137f6680a792a6c8
            slot: pair:1
            requested_at: "2026-10-21T21:24:41.937989Z"
          phase: finished
          pid: 84320
          started_at: "2026-10-21T21:24:42.033184Z"
          finished_at: "2026-10-21T21:27:44.751636Z"
          session_id: 01a1605a-22bd-7cc1-86a3-bbcb03574089
          outcome: returned
      "n": 10
      timestamp: "2026-10-21T14:27:45-07:00"
      dispatched: "2026-10-21T14:24:40-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-15
          disposition: addressed
          note: schedule_owner_batch.go:236-255 settles expired undispatched reservations independently of busy/policy/capacity. The reservation matrix passes; removing this branch through a temporary overlay makes manual/revision/expiry fail. Removing the companion timeout-recovery correction also fails TestUnstartedTimeoutSummaryCanBeRecovered.
          round: 10
        - id: BR-16
          disposition: addressed
          note: The plan's BR-16 revision at line 770 explicitly supersedes earlier location tables and supplies the complete current map. StateOwner now points to owner.go, where its declaration and capability methods exist; the production-anchor and plan/atlas equality checks pass.
          round: 10
      findings:
        - kind: defect
          id: BR-17
          severity: Critical
          title: Ordinary ticks erase unresolved catalogue-discovery errors
          detail: 'cmd/internal/couchcmd/schedule_owner_loop.go:474-482 ignores refresh''s returned error and immediately runs tick; schedule_owner.go:344-352 then publishes a fresh snapshot with an empty Error. Renaming a configured hub away makes refresh report its discovery failure, but the subsequent successful tick clears that error without rediscovery succeeding. A temporary-overlay regression reproduced this on the pinned head. ARCH-ORDER and ARCH-SECURE: retain each failure until evidence from the corresponding operation resolves it; successful ledger reads do not establish successful catalogue discovery. Preserve settlement progress while retaining the discovery error, and add a failure -> tick -> successful rediscovery regression.'
          family: unresolved-failure-visibility
          round: 10
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5cf5c526-6dab-4280-b373-7bba87ce9b16
            repository: github.com/xianxu/pair
            issue_id: "000737"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/5858f8e2c508260c
            reviewed_head: 16589ba4c7b896b51f50b2761a8160be58e6720f
            slot: pair:1
            requested_at: "2026-10-21T21:39:02.074151Z"
          phase: finished
          pid: 69298
          started_at: "2026-10-21T21:39:02.176578Z"
          finished_at: "2026-10-21T21:41:45.738683Z"
          session_id: 01a16067-42c4-71a1-8aba-072c4cc31373
          outcome: returned
      "n": 11
      timestamp: "2026-10-21T14:41:46-07:00"
      dispatched: "2026-10-21T14:39:00-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-17
          disposition: addressed
          note: schedule_owner.go retains catalogue failure evidence through tick, publication and terminal settlement until successful refresh. Catalogue-health regressions pass; removing the retained-error projection in a disposable checkout makes TestScheduleOwnerCatalogueHealthRetainedWhileFactsSettle fail with the original erased-diagnostic symptom.
          round: 11
        - id: BR-15
          disposition: addressed
          note: settleObservedBatch expires known-undispatched reservations independently of pending catalogue edits. The reservation matrix passes; removing that branch makes manual/revision/expiry fail because the reservation remains stranded.
          round: 11
        - id: BR-16
          disposition: addressed
          note: The plan explicitly supersedes earlier location tables. Its active map and atlas correctly locate StateOwner in couchschedule/owner.go and identify current transport and discovery declarations. Production-anchor and map-equality checks pass.
          round: 11
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      reviewed: 16589ba4c7b896b51f50b2761a8160be58e6720f
      blocked: false
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

## Round 2 — 2026-10-20T20:40:39-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Socket/CLI refusal-isolation and valid-manual-run regressions pass. Disabling diagnostic-entry validation support makes TestScheduleMixedRefusalStatusAndHistoryThroughSocket fail with invalid schedule status task identity.
- BR-2 — addressed — Cache recovery covers publication, legacy staging, touch and eviction interruptions with abrupt subprocess exits. Disabling recoverCacheLocked makes TestCacheAbruptMutationRecovery/publication-task-written fail on retained aged residue. The new finding concerns staging outside this repaired cache path.
- BR-3 — addressed — Encode/decode and Advance share phase validation. Source/stage/companion regressions pass; removing persistence-boundary validation makes TestStateJSONRefusesInconsistentCompanions/notice_missing_summary fail.
- BR-4 — addressed — Inspected the before/after inventory and source declarations. The Core concepts tables now name TaskKey in model.go, Catalog, Builder and scheduleCoordinator, and explicitly label scheduleNotice and FinalizeNotice as planned M2. The Revisions passage records that distinction.
- BR-5 — addressed — Skill examples now use runnable landing-plan commands and explain optional flags separately. TestSkillDocumentsRecovery passes at head; overlaying the previous skill makes it fail on couch --schedule --status [--json].
- BR-6 — addressed — Manifest and input codecs preserve unknown members through reuse and rebuilding. TestCacheFutureManifestMetadataSurvivesReuseAndRebuild passes; restoring unknown-member rejection makes the regression fail.

### Raised

- **BR-7** [Important] `durable-artifact-crash-recovery` (kind: defect) Build captures and ledger staging still have only deferred cleanup
  cmd/internal/couchschedule/build.go:86 creates couch-build-* directories outside the managed cache, and store_unix.go:132 creates .state-*.tmp snapshots; both rely solely on deferred removal. Temporary overlay subprocess tests exiting abruptly reproduced both abandoned shapes, and a subsequent successful StateStore.Update left the old snapshot untouched. This is the 2nd finding in family durable-artifact-crash-recovery, with two remaining staging families found. State the rule that every scheduler-owned staging artifact needs owner-safe crash cleanup, enumerate all staging writers, and sweep the class in this round. Add identity-fenced recovery under the appropriate locks and abrupt-death regressions; build captures and repeated ledger residues otherwise accumulate outside the declared retention bounds. ARCH-FUNERAL, ARCH-PURPOSE.

## Round 3 — 2026-10-20T21:12:59-07:00 (codex) — BLOCKED

### Disposed

- BR-7 — addressed — Build captures now use owner-held slots, ledger residues recover under the state lock, and operational collection staging recovers under the task lock. The native crash tests pass; independently disabling capture, ledger, and collection recovery in a disposable pinned-head copy makes the corresponding regressions fail.

### Raised

- **BR-8** [Critical] `durable-artifact-crash-recovery` (kind: defect) Ledger publication promotes a substituted staging inode into authoritative state
  cmd/internal/couchschedule/store_unix.go:169 renames the temporary pathname without checking that it still identifies the file originally opened by publishSchedule. A disposable pinned-head regression replaced that pathname at the existing pre-rename hook: StateStore.Update returned nil and state.json contained the replacement's empty tasks map instead of the intended ledger. This can discard retained execution authority. This is the 3rd finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; enforce the rule that every staging publication and deletion must verify its captured ownership identity, and sweep the complete writer inventory. Add a descriptor-relative identity check before ledger rename and a regression asserting refusal, unchanged prior authority, and preservation of the replacement. ARCH-SECURE, ARCH-PURPOSE.

## Round 4 — 2026-10-20T21:23:47-07:00 (codex) — passed

### Disposed

- BR-8 — addressed — store_unix.go:169 checks captured staging identity immediately before Renameat, sharing the predicate with error cleanup. TestLedgerPublicationRefusesReplacement at store_recovery_test.go:374 passes on head; removing that guard in a disposable copy fails refusal, unchanged prior authority, and replacement preservation assertions. Ledger/Store race tests pass. The disposable copy was removed.

## Round 5 — 2026-10-21T00:07:19-07:00 (codex) — BLOCKED

### Raised

- **BR-9** [Critical] `effect-authorization-fencing` (kind: defect) Notice admission can proceed after its deadline or committed cancellation
  cmd/internal/couchcmd/schedule_automatic.go:650-654 checks authority before calling Admit, while schedule_notice.go:45 then waits for holdHubEffect using the service context. There is no persisted-deadline cancellation or task-authority recheck after that wait. Two deterministic probes using the production scheduler, real broker, held ownership gate, and injected clock reproduced broker admission after NoticeDeadline and after catalogue removal had committed, finalized history, and cleared Active. ARCH-ORDER and ARCH-CONSTRAINTS: enforce one rule across queued and blocked effects—admission must remain authorized until the actual broker boundary. Bound gate/admission work by the persisted deadline and serialize or cancel pending admission against declaration cancellation, preserving uncertainty for already accepted effects. Add both interleavings as permanent regression tests and return this ordering gap to the TL under the approved transaction revision.
- **BR-10** [Critical] `task-refusal-isolation` (kind: defect) One damaged task history prevents every scheduler worker from starting
  cmd/internal/couchcmd/schedule_automatic.go:151-153 returns a task-local passive-history error from the entire recovery transaction; run starts workers only after recovery succeeds. A probe created offline-skip history for two hubs, truncated one task's record.json, and reopened the scheduler: recover returned the JSON error, preventing all workers from starting. Similar global propagation remains in scheduled completion and notice-history paths. This is the 2nd finding in family task-refusal-isolation. Earlier corrections handled instances; do not patch only recover. State and enforce the rule that task-history failures preserve that task's authority and expose a local refusal, while shared authority-store failures block globally. Enumerate startup recovery, passive repair, completion reconciliation, cancellation finalization, and notice finalization; sweep those consumers and test that a healthy second hub continues. ARCH-PURPOSE.
- **BR-11** [Critical] `additive-persisted-format-compatibility` (kind: defect) Same-occurrence recovery silently deletes unknown summary fields
  cmd/internal/couchschedule/model.go:235 and :260 replace Last with the recovered summary, while cmd/internal/couchcmd/schedule_records.go:14-15 constructs that summary without the existing Last.Unknown members. A persisted scheduled-delivery summary containing future_summary_field={"value":42} lost that field after automaticScheduler.recover processed the same occurrence. Codec preservation tests do not cover this rewrite. This is the 2nd finding in family additive-persisted-format-compatibility. Do not fix only this caller: enforce preservation of unknown members whenever an existing persisted object is updated for the same identity, keeping distinct schema namespaces separate. Sweep manual, scheduled, passive, and uncertain canonical-adoption paths, with reopen-and-rewrite regressions. ARCH-EVOLVE.

## Round 6 — 2026-10-21T08:53:10-07:00 (codex) — BLOCKED

### Disposed

- BR-9 — addressed — Held-gate deadline and cancellation regressions pass and fail when the admission fence is removed; additional tests cover deadline-bounded gate and broker waits.
- BR-10 — addressed — Seven history-consumer isolation cases pass and fail without local quarantine; manual refusal, pending-evidence retry, and shared-store failure tests also pass.
- BR-11 — addressed — Manual and scheduled recovery regressions pass and lose persisted summary metadata when the central same-identity merge is removed; broader rewrite tests cover passive and uncertain paths.

### Raised

- **BR-12** [Critical] `durable-artifact-crash-recovery` (kind: defect) Cached canonical recovery is ignored after a failed authority commit
  cmd/internal/couchcmd/schedule_automatic.go:250 constructs ResultRecovered without At, so model.go:162-163 silently ignores it. A production-path overlay probe published canonical passive history with different timestamps, failed the following ledger commit, and retried: the ledger retained EndedAt=12:00:00 instead of canonical 11:59:50. ARCH-ORDER: cached and freshly read evidence must produce equivalent recovery transitions, including after failed persistence. This is the 4th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; state and enforce that rule across cached/direct adoption and failed-commit retries, rather than patching only this call. Supply the canonical event time and add a permanent regression asserting the reopened ledger matches canonical history.

## Round 7 — 2026-10-21T09:06:35-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — addressed — TestScheduleCachedCanonicalRecoveryAfterFailedCommit passes at head and fails for skipped, refused, and completed outcomes with the old missing-At producer restored through a disposable overlay. All production recovery producers use NewRecoveryEvent; direct and cached scheduled adoption share adoptScheduledSummary.
- BR-9 — addressed — Admission retains authority through bounded broker admission after acquiring the hub gate. Deadline/cancellation regression tests pass under race.
- BR-10 — addressed — Shared quarantine handling preserves task authority and isolates history failures. Startup, consumer-isolation, and pending-evidence regressions pass under race.
- BR-11 — addressed — Same-ID reducer rewrites preserve summary unknown members separately from operational-record metadata. Codec, consumer, and detached-snapshot regressions pass under race.

### Raised

- **BR-13** [Critical] `durable-artifact-crash-recovery` (kind: defect) Recovery clears authority using history whose durability remains unconfirmed
  schedule_automatic.go:190 reads history through Store.Read and adopts it at line 213 without completing publication durability. A disposable regression injected directory-sync failure after bundle rename: Publish returned an error, the bundle remained readable, and startup recovery nevertheless cleared Active while the same sync still failed. The same regression failed through schedule_manual.go:173 and schedule_service.go:102. The notify-only adoption path at schedule_automatic.go:468 also bypasses durable confirmation. This violates the history-before-authority-release contract and can leave committed completion authority referring to output whose publication was never confirmed durable (ARCH-ORDER, ARCH-PURPOSE). This is the 5th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; do NOT fix only this instance. Enforce one rule across every adoption consumer: readable history becomes authoritative only after its publication/finalization durability is confirmed under the task lock. Reuse a shared durable adoption boundary, retain authority on failure, and add permanent regressions across the consumer inventory.

## Round 8 — 2026-10-21T10:31:26-07:00 (codex) — BLOCKED

### Disposed

- BR-13 — addressed — ReadDurable shares the task-locked read/validation/sync boundary; authority consumers use it, and finalization confirms directory durability. TestScheduleDurableAdoptionConsumerInventory passes at HEAD. A disposable Go overlay replacing ReadDurable with Read makes the consumer regression fail, including startup, completion, manual recovery, reserved execution and notify adoption.
- BR-9 — addressed — Retained disposition verified: admission holds authority after acquiring the hub gate, rechecks the persisted deadline and tuple, and bounds broker waits. Admission regressions pass, including under race detection.
- BR-10 — addressed — Retained disposition verified: history failures quarantine the named task while preserving authority; consumer-isolation and healthy-peer dispatch tests pass.
- BR-11 — addressed — Retained disposition verified: Advance preserves same-occurrence summary metadata, and recovery keeps record and summary namespaces separate. Preservation regressions pass.
- BR-12 — addressed — Retained disposition verified: direct and cached adoption share timestamped recovery construction. The failed-commit canonical recovery regression passes.

### Raised

- **BR-14** [Critical] `durable-artifact-crash-recovery` (kind: defect) Catalogue cancellation strands a notice whose terminal history already committed
  cmd/internal/couchcmd/schedule_service.go:258-262 attempts dropped finalization without first adopting existing terminal history. A disposable production-path regression lets receipt polling finalize submitted, injects BeforePersist failure, then removes or revises the declaration before retry. Both cases remain stuck after three refresh/step cycles: FinalizeNotice returns "operational notice is not pending", Active remains delivery, Last remains pending, and task quarantine prevents further reconciliation at schedule_automatic.go:451-453. This is the 6th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; do NOT fix only this instance. Enforce the rule that durably confirmed canonical terminal history wins before any cancellation or replacement transition. Sweep catalogue removal/revision/owner changes and every competing finalization consumer, preserve terminal facts, and add sequence regressions for failed authority commits followed by cancellation. ARCH-ORDER, ARCH-PURPOSE.

## Round 9 — 2026-10-21T14:10:10-07:00 (codex) — BLOCKED

### Raised

- **BR-15** [Critical] `persisted-state-invariants` (kind: defect) Pending catalogue edits strand undispatched reservations indefinitely
  cmd/internal/couchcmd/schedule_owner_dispatch.go:132-135 skips reserved work whenever o.busy is set, while schedule_owner_batch.go:236-243 provides no deadline settlement for those reservations. With both workers occupied, reserve another task and edit its declaration before dispatch: refresh marks it busy, but neither subsequent ticks nor refreshes can settle it, even after its deadline. A temporary test overlay reproduced this in all six scheduled cases: script/notify-only crossed with revision/removal/owner-policy edits. ARCH-ORDER, ARCH-CONSTRAINTS and ARCH-PURPOSE: this is the 2nd finding in family `persisted-state-invariants`. Do NOT fix only this instance; establish the rule that every undispatched reservation has a bounded settlement path independent of dispatch eligibility, and sweep manual, scripted and notify-only reservations across edit, expiry and cancellation sequences. Preserve canonical-evidence precedence and settle known-unstarted work through the pure transition model before applying the edit.
- **BR-16** [Critical] `core-concept-traceability` (kind: defect) The replacement core-concepts table mislocates StateOwner
  workshop/plans/000737-couch-scheduler-plan.md:634 lists StateOwner in cmd/internal/couchschedule/{store,store_unix}.go, but neither file declares it; the type and its capability methods are implemented in cmd/internal/couchschedule/owner.go:13 onward. This contradicts the required core-concepts cross-check. ARCH-PURPOSE: this is the 2nd finding in family `core-concept-traceability`. Reconcile every active replacement-table row against its production declaration and mark superseded entries explicitly, rather than repairing only this row. Record the corrected implementation map in a Revisions entry.

## Round 10 — 2026-10-21T14:27:45-07:00 (codex) — BLOCKED

### Disposed

- BR-15 — addressed — schedule_owner_batch.go:236-255 settles expired undispatched reservations independently of busy/policy/capacity. The reservation matrix passes; removing this branch through a temporary overlay makes manual/revision/expiry fail. Removing the companion timeout-recovery correction also fails TestUnstartedTimeoutSummaryCanBeRecovered.
- BR-16 — addressed — The plan's BR-16 revision at line 770 explicitly supersedes earlier location tables and supplies the complete current map. StateOwner now points to owner.go, where its declaration and capability methods exist; the production-anchor and plan/atlas equality checks pass.

### Raised

- **BR-17** [Critical] `unresolved-failure-visibility` (kind: defect) Ordinary ticks erase unresolved catalogue-discovery errors
  cmd/internal/couchcmd/schedule_owner_loop.go:474-482 ignores refresh's returned error and immediately runs tick; schedule_owner.go:344-352 then publishes a fresh snapshot with an empty Error. Renaming a configured hub away makes refresh report its discovery failure, but the subsequent successful tick clears that error without rediscovery succeeding. A temporary-overlay regression reproduced this on the pinned head. ARCH-ORDER and ARCH-SECURE: retain each failure until evidence from the corresponding operation resolves it; successful ledger reads do not establish successful catalogue discovery. Preserve settlement progress while retaining the discovery error, and add a failure -> tick -> successful rediscovery regression.

## Round 11 — 2026-10-21T14:41:46-07:00 (codex) — passed

### Disposed

- BR-17 — addressed — schedule_owner.go retains catalogue failure evidence through tick, publication and terminal settlement until successful refresh. Catalogue-health regressions pass; removing the retained-error projection in a disposable checkout makes TestScheduleOwnerCatalogueHealthRetainedWhileFactsSettle fail with the original erased-diagnostic symptom.
- BR-15 — addressed — settleObservedBatch expires known-undispatched reservations independently of pending catalogue edits. The reservation matrix passes; removing that branch makes manual/revision/expiry fail because the reservation remains stranded.
- BR-16 — addressed — The plan explicitly supersedes earlier location tables. Its active map and atlas correctly locate StateOwner in couchschedule/owner.go and identify current transport and discovery declarations. Production-anchor and map-equality checks pass.

## Open findings

- **BR-14** [Critical] `durable-artifact-crash-recovery` (kind: defect) Catalogue cancellation strands a notice whose terminal history already committed
