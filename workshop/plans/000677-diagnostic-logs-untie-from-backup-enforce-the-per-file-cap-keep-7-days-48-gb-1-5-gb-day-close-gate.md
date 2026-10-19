---
gate: boundary-review
issue: 677
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 98845410-e886-4819-8666-cf53bf2967ba
            repository: github.com/xianxu/pair
            issue_id: "000677"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/1326dc8662ea9d21
            reviewed_head: 687e59af511ceca9516f0755171c2876277d5de2
            slot: pair:3
            requested_at: "2026-10-19T02:07:23.288681Z"
          phase: finished
          pid: 59117
          started_at: "2026-10-19T02:07:23.585476Z"
          finished_at: "2026-10-19T02:11:05.675623Z"
          session_id: 01a151e9-df70-7791-b337-aa02e2d8a952
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 687e59af511ceca9516f0755171c2876277d5de2
        tracker_head: 2da03a0fd4e2c39f436a2e6b4c941e9fe401d201
        branch_point: 9950fa540423ab382e682437fc0564fadc104f29
        since: "2026-10-18T16:54:38-07:00"
        until: "2026-10-18T19:05:34-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T16:54:38-07:00"
              end: "2026-10-18T19:05:34-07:00"
              began: "2026-10-18T16:54:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: b527cbfce38f66d1e1ea8c72a4d16a428ca615598c8b67e96a9ebd4e8a4307e7
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 89401035
              windows:
                - start: "2026-10-18T16:54:38-07:00"
                  end: "2026-10-18T19:05:34-07:00"
            - harness: codex
              id: 01a15170-8a14-7b32-a5de-ed0b9b21fc18
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 385a927d3f4151bd301fee430df20d5a304e867f9117e504e4bc9f020d220fed
              started: "2026-10-18T23:54:52.431Z"
              spawned: true
              bytes: 306138
              windows:
                - start: "2026-10-18T23:54:52.431Z"
                  end: "2026-10-18T19:05:34-07:00"
            - harness: codex
              id: 01a15179-8362-7531-999e-6456914a2df0
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 29d3326e982f6cc7d2bfc0f91e3315ed8c2948cae8c8cd2dbdcdad7dee2b093e
              started: "2026-10-19T00:04:41.045Z"
              spawned: false
              bytes: 573922
              windows:
                - start: "2026-10-19T00:04:41.045Z"
                  end: "2026-10-18T19:05:34-07:00"
            - harness: codex
              id: 01a1517b-aef9-7e10-9124-c956a0c3a8ed
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: a348a6dae756cddb48f7f6fab0911e2d6d8e70c421a5e0f5ab3994817964a134
              started: "2026-10-19T00:07:03.133Z"
              spawned: false
              bytes: 433389
              windows:
                - start: "2026-10-19T00:07:03.133Z"
                  end: "2026-10-18T19:05:34-07:00"
            - harness: codex
              id: 01a1517e-ac15-7c52-9c78-994b9d137543
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 1565671cf6851dbe59deca7c9b6ad0552f3630bd69e722b36f571dfce9168254
              started: "2026-10-19T00:10:18.895Z"
              spawned: false
              bytes: 428746
              windows:
                - start: "2026-10-19T00:10:18.895Z"
                  end: "2026-10-18T19:05:34-07:00"
            - harness: codex
              id: 01a1517f-0797-70d1-828b-c5eb69511568
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 2abeae12b15fc7277b922ccb95077bd0c946da47e04bce540ee125b98e13927f
              started: "2026-10-19T00:10:42.324Z"
              spawned: false
              bytes: 417567
              windows:
                - start: "2026-10-19T00:10:42.324Z"
                  end: "2026-10-18T19:05:34-07:00"
            - harness: codex
              id: 01a15187-002b-7771-bbe9-1d77fd9fa7f1
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 33c9885456267433c718e3e852b66d06ed85395b8cfbc6eb5a515ad5ad119702
              started: "2026-10-19T00:19:24.429Z"
              spawned: true
              bytes: 568785
              windows:
                - start: "2026-10-19T00:19:24.429Z"
                  end: "2026-10-18T19:05:34-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
      "n": 1
      timestamp: "2026-10-18T19:11:07-07:00"
      dispatched: "2026-10-18T19:07:19-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Writer-side deletion replay does not inspect the segment being deleted
          detail: cmd/internal/diagnosticlog/collect.go:192 removes the backup fence from recoverDeletion, but writer.go:231-236 still authorizes replay using proof of the current path only; Write and maintain have the same pattern. After interruption at delete-intent, a reader can open the closed segment and a subsequent writer reopen can unlink it without inspecting that holder. Enforce exact-target proof across every recovery caller, including ordinary writers without ReaderProof, and add held-segment replay regressions. ARCH-ORDER, ARCH-SECURE.
          family: exact-target-deletion-proof
          round: 1
        - id: BR-2
          severity: Critical
          title: A missing-parent declaration terminates scheduled diagnostic collection
          detail: registry.go:150 now permits structurally valid missing-parent declarations, but cmd/internal/gcruntime/schedule.go:90-97 propagates their canonicalization error. storagegc.ScheduleWorker terminates on that error and preserves its prior cursor, so subsequent eligible paths remain unreachable across restarts. Isolate expected per-path blockers, preserve them visibly, and advance the bounded cursor without recreating or deleting the stale declaration. Test a stale declaration followed by an eligible path through the scheduler. ARCH-PURPOSE, ARCH-FUNERAL.
          family: diagnostic-traversal-blocker-isolation
          round: 1
        - id: BR-3
          severity: Critical
          title: The Core concepts tables contradict the implemented boundaries and locations
          detail: workshop/plans/000677-diagnostic-log-retention-plan.md:34 classifies diagnostic-only CLI scope as PURE, although runWithService performs service/filesystem IO and its tests require temporary directories and injected services. Line 43 claims modified measurement tests in wrap_events_test.go and diagnostic_buffer_test.go, neither changed in this range; the new implementation is diagnostic_turn_budget_test.go. Promote CLI scope to INTEGRATION and correct the measurement location/status in a Revisions entry. ARCH-PURE.
          family: core-concept-contract-accuracy
          round: 1
        - id: BR-4
          severity: Important
          title: Apply omits managed blockers and drops completed work when a page errors
          detail: cmd/internal/diagnosticlog/collect.go:145-187 skips noneligible managed rows and returns only removals, so ApplyDiagnostics can report no blockers while managed payloads remain blocked. Separately, gcruntime/runtime.go:314-317 discards rows returned alongside an error, losing already-completed deletions from byte accounting. Preserve retained/blocked results and completed effects independently of errors; test mixed managed rows and failure after one completed deletion. ARCH-PURPOSE, ARCH-ORDER.
          family: diagnostic-result-completeness
          round: 1
        - id: BR-5
          severity: Important
          title: The required artifact ownership guard fails on both changed source surfaces
          detail: 'go test ./cmd/internal/artifactpath fails TestProductionArtifactReferencesAreExactlyClassified: diagnosticlog/proof.go remains a NonArtifactSource despite its new artifactpath import, and diagnosticlog/sweep.go is absent from the exhaustive inventory. Update cmd/internal/artifactpath/manifest.go with accurate source classifications and rerun the guard; do not expand the known-violations allowance. ARCH-PURPOSE.'
          family: artifact-source-classification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: de4f7aaf-4757-4c1a-8c19-ba3953bf2a04
            repository: github.com/xianxu/pair
            issue_id: "000677"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/e94da22159af4b75
            reviewed_head: 8238fd8577e5a9244600bdbaa9913e90173bf74a
            slot: pair:3
            requested_at: "2026-10-19T02:20:49.684649Z"
          phase: finished
          pid: 8052
          started_at: "2026-10-19T02:20:49.978767Z"
          finished_at: "2026-10-19T02:23:35.581654Z"
          session_id: 01a151f6-2d4d-7572-ad08-f9a096bab008
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 8238fd8577e5a9244600bdbaa9913e90173bf74a
        tracker_head: 51f04bc5c359c8bc430d63572c7414d68bf99871
        branch_point: 9950fa540423ab382e682437fc0564fadc104f29
        since: "2026-10-18T16:54:38-07:00"
        until: "2026-10-18T19:19:51-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T16:54:38-07:00"
              end: "2026-10-18T19:19:51-07:00"
              began: "2026-10-18T16:54:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 629f6e319b8735718ecbe2470f6a8e3861d343329dafbf9ad073dde349bb8fc8
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 90930320
              windows:
                - start: "2026-10-18T16:54:38-07:00"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a15170-8a14-7b32-a5de-ed0b9b21fc18
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 385a927d3f4151bd301fee430df20d5a304e867f9117e504e4bc9f020d220fed
              started: "2026-10-18T23:54:52.431Z"
              spawned: true
              bytes: 306138
              windows:
                - start: "2026-10-18T23:54:52.431Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a15179-8362-7531-999e-6456914a2df0
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 29d3326e982f6cc7d2bfc0f91e3315ed8c2948cae8c8cd2dbdcdad7dee2b093e
              started: "2026-10-19T00:04:41.045Z"
              spawned: false
              bytes: 573922
              windows:
                - start: "2026-10-19T00:04:41.045Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a1517b-aef9-7e10-9124-c956a0c3a8ed
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: a348a6dae756cddb48f7f6fab0911e2d6d8e70c421a5e0f5ab3994817964a134
              started: "2026-10-19T00:07:03.133Z"
              spawned: false
              bytes: 433389
              windows:
                - start: "2026-10-19T00:07:03.133Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a1517e-ac15-7c52-9c78-994b9d137543
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 1565671cf6851dbe59deca7c9b6ad0552f3630bd69e722b36f571dfce9168254
              started: "2026-10-19T00:10:18.895Z"
              spawned: false
              bytes: 428746
              windows:
                - start: "2026-10-19T00:10:18.895Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a1517f-0797-70d1-828b-c5eb69511568
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 2abeae12b15fc7277b922ccb95077bd0c946da47e04bce540ee125b98e13927f
              started: "2026-10-19T00:10:42.324Z"
              spawned: false
              bytes: 417567
              windows:
                - start: "2026-10-19T00:10:42.324Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a15187-002b-7771-bbe9-1d77fd9fa7f1
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 33c9885456267433c718e3e852b66d06ed85395b8cfbc6eb5a515ad5ad119702
              started: "2026-10-19T00:19:24.429Z"
              spawned: true
              bytes: 568785
              windows:
                - start: "2026-10-19T00:19:24.429Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a151ee-b185-7051-8c26-d5ec2259e7d2
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 1393170e6dec4cb5f141390c06e5bf784cd2287a518d25b8371d0ca53ec5ddb7
              started: "2026-10-19T02:12:40.047Z"
              spawned: true
              bytes: 648869
              windows:
                - start: "2026-10-19T02:12:40.047Z"
                  end: "2026-10-18T19:19:51-07:00"
            - harness: codex
              id: 01a151ee-ca5d-7d41-a5dc-523a15a6805f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 092a1b851e07ba5f392072702f32d37aedf5a5d981864b96e8ee700774f28716
              started: "2026-10-19T02:12:46.393Z"
              spawned: true
              bytes: 807601
              windows:
                - start: "2026-10-19T02:12:46.393Z"
                  end: "2026-10-18T19:19:51-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.11616666666666667
        cuts:
            - dispatched: "2026-10-18T19:07:19-07:00"
              finalized: "2026-10-18T19:11:07-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T19:23:36-07:00"
      dispatched: "2026-10-18T19:20:46-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: collect.go:209 now proves the exact deletion target in shared recoverDeletion. The regression covers Open, Write, maintain and direct recovery; all four fail when the correction is removed through a temporary overlay.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The real-scheduler missing-parent regression passes and fails without the schedule.go correction. It verifies later collection, unchanged declaration bytes, no parent recreation and visible preview blocking. A remaining sibling in the broader isolation family is reported below.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The plan's Core concepts table now classifies CLI scope under Integration point at line 41 and identifies diagnostic_turn_budget_test.go as a new test at line 42. Both match the implementation; the Revisions passage records the correction.
          round: 2
        - id: BR-4
          disposition: addressed
          note: ObserveCollection preserves managed retained/blocked decisions, and runtime.go:323 appends completed rows before propagating errors. Both mixed-result and second-deletion-fault regressions pass at head and fail with the prior implementations overlaid.
          round: 2
        - id: BR-5
          disposition: addressed
          note: manifest.go classifies proof.go as a resolved consumer and sweep.go as a non-artifact source. The ownership guard passes at head and fails with the previous manifest; the known-violations allowance is unchanged.
          round: 2
      findings:
        - kind: defect
          id: BR-6
          severity: Critical
          title: Other per-path diagnostic blockers still terminate scheduled collection
          detail: 'cmd/internal/gcruntime/schedule.go:96-110 isolates only missing registered paths. A temporary regression using reportingFixture, an external append to its managed current log, and a later expired wrap-events-z-good.jsonl fails on scheduler batch 2 with "diagnostic reservation identity or size changed"; the later log remains uncollected. storagegc/schedule.go:117-118 propagates that error, preserving the cursor, and its worker exits. This is the 2nd finding in family diagnostic-traversal-blocker-isolation: state and enforce the rule across all path-local certificate/recovery blockers, retaining them visibly while advancing past that path. Enumerate and test the class rather than adding another message-specific exception; preserve cancellation, busy handling and global inventory failures. ARCH-PURPOSE, ARCH-FUNERAL.'
          family: diagnostic-traversal-blocker-isolation
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#677 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T19:11:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `exact-target-deletion-proof` Writer-side deletion replay does not inspect the segment being deleted
  cmd/internal/diagnosticlog/collect.go:192 removes the backup fence from recoverDeletion, but writer.go:231-236 still authorizes replay using proof of the current path only; Write and maintain have the same pattern. After interruption at delete-intent, a reader can open the closed segment and a subsequent writer reopen can unlink it without inspecting that holder. Enforce exact-target proof across every recovery caller, including ordinary writers without ReaderProof, and add held-segment replay regressions. ARCH-ORDER, ARCH-SECURE.
- **BR-2** [Critical] `diagnostic-traversal-blocker-isolation` A missing-parent declaration terminates scheduled diagnostic collection
  registry.go:150 now permits structurally valid missing-parent declarations, but cmd/internal/gcruntime/schedule.go:90-97 propagates their canonicalization error. storagegc.ScheduleWorker terminates on that error and preserves its prior cursor, so subsequent eligible paths remain unreachable across restarts. Isolate expected per-path blockers, preserve them visibly, and advance the bounded cursor without recreating or deleting the stale declaration. Test a stale declaration followed by an eligible path through the scheduler. ARCH-PURPOSE, ARCH-FUNERAL.
- **BR-3** [Critical] `core-concept-contract-accuracy` The Core concepts tables contradict the implemented boundaries and locations
  workshop/plans/000677-diagnostic-log-retention-plan.md:34 classifies diagnostic-only CLI scope as PURE, although runWithService performs service/filesystem IO and its tests require temporary directories and injected services. Line 43 claims modified measurement tests in wrap_events_test.go and diagnostic_buffer_test.go, neither changed in this range; the new implementation is diagnostic_turn_budget_test.go. Promote CLI scope to INTEGRATION and correct the measurement location/status in a Revisions entry. ARCH-PURE.
- **BR-4** [Important] `diagnostic-result-completeness` Apply omits managed blockers and drops completed work when a page errors
  cmd/internal/diagnosticlog/collect.go:145-187 skips noneligible managed rows and returns only removals, so ApplyDiagnostics can report no blockers while managed payloads remain blocked. Separately, gcruntime/runtime.go:314-317 discards rows returned alongside an error, losing already-completed deletions from byte accounting. Preserve retained/blocked results and completed effects independently of errors; test mixed managed rows and failure after one completed deletion. ARCH-PURPOSE, ARCH-ORDER.
- **BR-5** [Important] `artifact-source-classification` The required artifact ownership guard fails on both changed source surfaces
  go test ./cmd/internal/artifactpath fails TestProductionArtifactReferencesAreExactlyClassified: diagnosticlog/proof.go remains a NonArtifactSource despite its new artifactpath import, and diagnosticlog/sweep.go is absent from the exhaustive inventory. Update cmd/internal/artifactpath/manifest.go with accurate source classifications and rerun the guard; do not expand the known-violations allowance. ARCH-PURPOSE.

## Round 2 — 2026-10-18T19:23:36-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — collect.go:209 now proves the exact deletion target in shared recoverDeletion. The regression covers Open, Write, maintain and direct recovery; all four fail when the correction is removed through a temporary overlay.
- BR-2 — addressed — The real-scheduler missing-parent regression passes and fails without the schedule.go correction. It verifies later collection, unchanged declaration bytes, no parent recreation and visible preview blocking. A remaining sibling in the broader isolation family is reported below.
- BR-3 — addressed — The plan's Core concepts table now classifies CLI scope under Integration point at line 41 and identifies diagnostic_turn_budget_test.go as a new test at line 42. Both match the implementation; the Revisions passage records the correction.
- BR-4 — addressed — ObserveCollection preserves managed retained/blocked decisions, and runtime.go:323 appends completed rows before propagating errors. Both mixed-result and second-deletion-fault regressions pass at head and fail with the prior implementations overlaid.
- BR-5 — addressed — manifest.go classifies proof.go as a resolved consumer and sweep.go as a non-artifact source. The ownership guard passes at head and fails with the previous manifest; the known-violations allowance is unchanged.

### Raised

- **BR-6** [Critical] `diagnostic-traversal-blocker-isolation` (kind: defect) Other per-path diagnostic blockers still terminate scheduled collection
  cmd/internal/gcruntime/schedule.go:96-110 isolates only missing registered paths. A temporary regression using reportingFixture, an external append to its managed current log, and a later expired wrap-events-z-good.jsonl fails on scheduler batch 2 with "diagnostic reservation identity or size changed"; the later log remains uncollected. storagegc/schedule.go:117-118 propagates that error, preserving the cursor, and its worker exits. This is the 2nd finding in family diagnostic-traversal-blocker-isolation: state and enforce the rule across all path-local certificate/recovery blockers, retaining them visibly while advancing past that path. Enumerate and test the class rather than adding another message-specific exception; preserve cancellation, busy handling and global inventory failures. ARCH-PURPOSE, ARCH-FUNERAL.

## Open findings

- **BR-6** [Critical] `diagnostic-traversal-blocker-isolation` (kind: defect) Other per-path diagnostic blockers still terminate scheduled collection
