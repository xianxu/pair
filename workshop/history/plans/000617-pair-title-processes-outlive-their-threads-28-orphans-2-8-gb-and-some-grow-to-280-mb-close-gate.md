---
gate: boundary-review
issue: 617
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ce415ae0-4d4e-48d7-825c-f72a9cde21a3
            repository: github.com/xianxu/pair
            issue_id: "000617"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0bdf4acd453d6a0b
            reviewed_head: 21453132e5af1a4fe4fb296a8530d21d41b30aa4
            slot: pair:6
            requested_at: "2026-10-17T01:07:06.892533Z"
          phase: finished
          pid: 66217
          started_at: "2026-10-17T01:07:07.021857Z"
          finished_at: "2026-10-17T01:09:09.022683Z"
          session_id: 01a14765-f819-7dc3-a9e2-6b6d58692155
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 21453132e5af1a4fe4fb296a8530d21d41b30aa4
        tracker_head: be2f921f157cce849404fdb13c03c33104deb600
        branch_point: e16f1373fd78c912d0a3be90e68972baa3cd6347
        since: "2026-10-16T12:54:47-07:00"
        until: "2026-10-16T18:06:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T12:54:47-07:00"
              end: "2026-10-16T18:06:45-07:00"
              began: "2026-10-16T12:54:47-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14367-664d-7850-a55c-bb707864cce5
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 54d48493986610d3a8d5a13a174e0435ef148208c465f3b5560809b9a048d70c
              started: "2026-10-16T06:30:12.567Z"
              spawned: false
              bytes: 20745597
              windows:
                - start: "2026-10-16T12:54:47-07:00"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a14649-013b-7201-8290-bd63f624a692
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: c17a0397274c942f751046da70cd8a3e7464e8d30a3c4c673171f4fbdc8ae916
              started: "2026-10-16T19:55:52.08Z"
              spawned: true
              bytes: 3766308
              windows:
                - start: "2026-10-16T19:55:52.08Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146a5-de47-7dc2-9a63-1092559b3b5a
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 059b9b5e9795cbdfd3c4227ae1dad0c09f01725edf3083b957900d260559c064
              started: "2026-10-16T21:37:18.463Z"
              spawned: false
              bytes: 827884
              windows:
                - start: "2026-10-16T21:37:18.463Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146a8-55fc-7532-acae-87650647d022
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 4084bd029c032f657f9cfc6fc119f01661c65a70cfab0da62d4f45a5735ca37d
              started: "2026-10-16T21:40:00.15Z"
              spawned: false
              bytes: 446289
              windows:
                - start: "2026-10-16T21:40:00.15Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146a9-265d-76c2-b37e-af5db394f4e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: b56cb474dd55ed9eb7e5afa3b00b053c1b79aede1f03be9d68a707be6feeebf9
              started: "2026-10-16T21:40:53.616Z"
              spawned: false
              bytes: 451408
              windows:
                - start: "2026-10-16T21:40:53.616Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146a9-8342-7bb0-ad16-f7c0a57fde8f
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 6b0fbb6a69db97fda3d8d63943912a0461f7c7fbe8ff244c4a35076eb15b1a2b
              started: "2026-10-16T21:41:17.142Z"
              spawned: false
              bytes: 402657
              windows:
                - start: "2026-10-16T21:41:17.142Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146be-c40f-7260-8010-82bec0adaa3e
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: a7f25608804b966b5f3a6c1afa7608eb80a4a17ca23a9aefb328de0e6eeb126e
              started: "2026-10-16T22:04:29.708Z"
              spawned: true
              bytes: 2208360
              windows:
                - start: "2026-10-16T22:04:29.708Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146be-f19f-7552-88ca-b1c4e79d1e14
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 059b704bbc1b12aa07883768e606e490cc4705b0d46db65daefa92408cffd01a
              started: "2026-10-16T22:04:41.408Z"
              spawned: true
              bytes: 3052098
              windows:
                - start: "2026-10-16T22:04:41.408Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146e2-2dd2-7e02-bd2f-bf9c3a961c59
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f992fcb0fe63520ca49bd95bf2d7f5203c9d0e8c6a8e0cbccd95a7946ef06ce5
              started: "2026-10-16T22:43:11.045Z"
              spawned: false
              bytes: 476389
              windows:
                - start: "2026-10-16T22:43:11.045Z"
                  end: "2026-10-16T18:06:45-07:00"
            - harness: codex
              id: 01a146e2-9bd2-7193-bd12-00016f7810e4
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 8e6f6a67aa794dde5f928859158f26e469edfaf4493ff567ee5c151c3b573b4a
              started: "2026-10-16T22:43:39.504Z"
              spawned: false
              bytes: 406012
              windows:
                - start: "2026-10-16T22:43:39.504Z"
                  end: "2026-10-16T18:06:45-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 55.30478333333333
      "n": 1
      timestamp: "2026-10-16T18:09:11-07:00"
      dispatched: "2026-10-16T18:07:04-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Atomic title publications leave unrecoverable temporary artifacts
          detail: 'cmd/internal/titlepoller/runtime.go:50 now routes PID and workspace-owner writes through osfs.FS.WriteAtomic, which creates sibling names such as title-pid-TAG.123456789. Lease EOF can SIGKILL the title between creation and rename; rename failure also leaves the temporary file. These names have no recovery path: artifactpath/gc.go recognizes the final PID path, while storagegc/stateio.go:116 recognizes only .pending-* staging names. storagegc/inventory.go:240 treats the leftovers as unknown artifacts and blocks collection. ARCH-FUNERAL: provide fenced recovery for both publication families and test interruption after temporary-file creation, including preservation of active publishers.'
          family: interrupted-publication-recovery
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2cfbfc7e-e458-4f72-8e0f-4e3f8920859a
            repository: github.com/xianxu/pair
            issue_id: "000617"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ae28cf6c4bc224ca
            reviewed_head: 20e3a811d207d6f6cd73dbb2df2df18733bd6837
            slot: pair:6
            requested_at: "2026-10-17T01:33:08.486677Z"
          phase: finished
          pid: 19887
          started_at: "2026-10-17T01:33:08.666176Z"
          finished_at: "2026-10-17T01:34:57.973586Z"
          session_id: 01a1477d-cdbc-7f32-8594-5115e80a7b4b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 20e3a811d207d6f6cd73dbb2df2df18733bd6837
        tracker_head: 709a9bd8f89551023189eb527e58afcd5bf602c6
        branch_point: e16f1373fd78c912d0a3be90e68972baa3cd6347
        since: "2026-10-16T12:54:47-07:00"
        until: "2026-10-16T18:32:07-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T12:54:47-07:00"
              end: "2026-10-16T18:32:07-07:00"
              began: "2026-10-16T12:54:47-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14367-664d-7850-a55c-bb707864cce5
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: df76c1e89348a2f0b6e889e9c13e2a0528162fabb729517c47ed61b880e6a3cd
              started: "2026-10-16T06:30:12.567Z"
              spawned: false
              bytes: 22077660
              windows:
                - start: "2026-10-16T12:54:47-07:00"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a14649-013b-7201-8290-bd63f624a692
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: c17a0397274c942f751046da70cd8a3e7464e8d30a3c4c673171f4fbdc8ae916
              started: "2026-10-16T19:55:52.08Z"
              spawned: true
              bytes: 3766308
              windows:
                - start: "2026-10-16T19:55:52.08Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146a5-de47-7dc2-9a63-1092559b3b5a
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 059b9b5e9795cbdfd3c4227ae1dad0c09f01725edf3083b957900d260559c064
              started: "2026-10-16T21:37:18.463Z"
              spawned: false
              bytes: 827884
              windows:
                - start: "2026-10-16T21:37:18.463Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146a8-55fc-7532-acae-87650647d022
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 4084bd029c032f657f9cfc6fc119f01661c65a70cfab0da62d4f45a5735ca37d
              started: "2026-10-16T21:40:00.15Z"
              spawned: false
              bytes: 446289
              windows:
                - start: "2026-10-16T21:40:00.15Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146a9-265d-76c2-b37e-af5db394f4e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: b56cb474dd55ed9eb7e5afa3b00b053c1b79aede1f03be9d68a707be6feeebf9
              started: "2026-10-16T21:40:53.616Z"
              spawned: false
              bytes: 451408
              windows:
                - start: "2026-10-16T21:40:53.616Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146a9-8342-7bb0-ad16-f7c0a57fde8f
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 6b0fbb6a69db97fda3d8d63943912a0461f7c7fbe8ff244c4a35076eb15b1a2b
              started: "2026-10-16T21:41:17.142Z"
              spawned: false
              bytes: 402657
              windows:
                - start: "2026-10-16T21:41:17.142Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146be-c40f-7260-8010-82bec0adaa3e
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 1d7921ced64db77814a215e2e0bf1710386b8a5990d30365ef7a26f8f2e9244a
              started: "2026-10-16T22:04:29.708Z"
              spawned: true
              bytes: 2604827
              windows:
                - start: "2026-10-16T22:04:29.708Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146be-f19f-7552-88ca-b1c4e79d1e14
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 059b704bbc1b12aa07883768e606e490cc4705b0d46db65daefa92408cffd01a
              started: "2026-10-16T22:04:41.408Z"
              spawned: true
              bytes: 3052098
              windows:
                - start: "2026-10-16T22:04:41.408Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146e2-2dd2-7e02-bd2f-bf9c3a961c59
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f992fcb0fe63520ca49bd95bf2d7f5203c9d0e8c6a8e0cbccd95a7946ef06ce5
              started: "2026-10-16T22:43:11.045Z"
              spawned: false
              bytes: 476389
              windows:
                - start: "2026-10-16T22:43:11.045Z"
                  end: "2026-10-16T18:32:07-07:00"
            - harness: codex
              id: 01a146e2-9bd2-7193-bd12-00016f7810e4
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 8e6f6a67aa794dde5f928859158f26e469edfaf4493ff567ee5c151c3b573b4a
              started: "2026-10-16T22:43:39.504Z"
              spawned: false
              bytes: 406012
              windows:
                - start: "2026-10-16T22:43:39.504Z"
                  end: "2026-10-16T18:32:07-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 55.30478333333333
        cuts:
            - dispatched: "2026-10-16T18:07:04-07:00"
              finalized: "2026-10-16T18:09:11-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T18:34:59-07:00"
      dispatched: "2026-10-16T18:33:06-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Both title publication families now use root-locked private pending staging through PublishMetadata. Production CLI interruption tests verify preservation of the previous target, exclusion of active publishers, and recovery after EOF termination; Collector.applyPage invokes that recovery automatically. Restoring the old writer in a disposable overlay made TestTitlePublicationRenameFailureLeavesNoUnclassifiedSibling fail for both families with sibling residue.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 20e3a811d207d6f6cd73dbb2df2df18733bd6837
      blocked: false
---

# Gate ledger — pair#617 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T18:09:11-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `interrupted-publication-recovery` (kind: defect) Atomic title publications leave unrecoverable temporary artifacts
  cmd/internal/titlepoller/runtime.go:50 now routes PID and workspace-owner writes through osfs.FS.WriteAtomic, which creates sibling names such as title-pid-TAG.123456789. Lease EOF can SIGKILL the title between creation and rename; rename failure also leaves the temporary file. These names have no recovery path: artifactpath/gc.go recognizes the final PID path, while storagegc/stateio.go:116 recognizes only .pending-* staging names. storagegc/inventory.go:240 treats the leftovers as unknown artifacts and blocks collection. ARCH-FUNERAL: provide fenced recovery for both publication families and test interruption after temporary-file creation, including preservation of active publishers.

## Round 2 — 2026-10-16T18:34:59-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Both title publication families now use root-locked private pending staging through PublishMetadata. Production CLI interruption tests verify preservation of the previous target, exclusion of active publishers, and recovery after EOF termination; Collector.applyPage invokes that recovery automatically. Restoring the old writer in a disposable overlay made TestTitlePublicationRenameFailureLeavesNoUnclassifiedSibling fail for both families with sibling residue.

## Open findings

(none — every finding has been disposed)
