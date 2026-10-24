---
gate: boundary-review
issue: 860
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 26ceec7f-df70-453d-a570-bea58f9fe642
            repository: github.com/xianxu/pair
            issue_id: "000860"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9dae7d0c560f4a31
            reviewed_head: 17ce2edb890108642629051661d7866b103882a0
            slot: pair:9
            requested_at: "2026-10-24T07:47:33.703031Z"
          phase: finished
          pid: 2096
          started_at: "2026-10-24T07:47:33.84405Z"
          finished_at: "2026-10-24T07:49:33.89961Z"
          session_id: 01a16ce1-1aa1-79a0-b7bb-40fb6f5364ec
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 17ce2edb890108642629051661d7866b103882a0
        tracker_head: 9522e4b2d46cfb4b81530d9a824d8f7a887cc1a3
        branch_point: 99846a5015f753f05edaf4778618849f377a2f71
        since: "2026-10-24T00:22:54-07:00"
        until: "2026-10-24T00:46:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-24T00:22:54-07:00"
              end: "2026-10-24T00:46:55-07:00"
              began: "2026-10-24T00:22:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 9716b7a7b4e12f509f14fe884b442bfa58441bf9c64078376c61be479e848ec8
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 31825969
              windows:
                - start: "2026-10-24T00:22:54-07:00"
                  end: "2026-10-24T00:46:55-07:00"
            - harness: codex
              id: 01a16ccd-42bf-7241-9b61-3796ba74a6cf
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: c07d7c9f93a59a64644afe80ee29fa0f7137d4709f1c58b3a0ab2dc0404dba56
              started: "2026-10-24T07:25:53.817Z"
              spawned: true
              bytes: 664918
              windows:
                - start: "2026-10-24T07:25:53.817Z"
                  end: "2026-10-24T00:46:55-07:00"
      "n": 1
      timestamp: "2026-10-24T00:49:36-07:00"
      dispatched: "2026-10-24T00:47:31-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: 'ARCH-PURPOSE: child-proof publication sweep leaves native-exit fixtures racing'
          detail: 'The child-proof extension at cmd/internal/couchcmd/worker_cue_coordinator_test.go:62 fixes newWorkerCueRig but leaves the same live-wrapper pattern at native_exit_test.go:41, :147 and :167. Running go test -race ./cmd/internal/couchcmd -run ''^TestNativeExit(AdmitsOnlyTheCurrentChild|WithoutChildEvidenceIsUnsupported|ReachesTheHubAsOneLifecycleNotice)$'' -count=20 -timeout=120s failed: newExitRig''s write at :41 races messageService.execute at message_service.go:896, where the value-receiver call copies authority. These are all remaining connected-fixture child assignments; worker_cue_coordinator_test.go:309 is the documented no-wrapper control. Compose all three native-exit variants before service construction through the existing decorator, preserve their assertions, and repeat this regression command. Add a Revisions entry completing the demonstrated child-proof family.'
          family: child-authority-before-readers
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Done-when does not cover the added child-proof contract
          detail: workshop/issues/000860-multi-hub-fixture-authority-race.md:53-57 covers the original notification race, multi-hub startup policy and hub/hubs assignments, but neither child-proof revision at :96 and :104 has corresponding acceptance criteria. These are both instances of the same contract extension, implemented in worker_cue_coordinator_test.go. Update Done-when to require pre-start child-proof composition and repeated race coverage preserving the default, role-change, committed-report and bounded-proof controls, including the native-exit siblings identified above.
          family: acceptance-criteria-track-contract
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6cd55195-e21a-4508-a3fa-f2d9474724d6
            repository: github.com/xianxu/pair
            issue_id: "000860"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/b6dc7b87fc54ff65
            reviewed_head: 67a53ea2b4d26e3ed053150decb566c473e45068
            slot: pair:9
            requested_at: "2026-10-24T07:58:01.103807Z"
          phase: finished
          pid: 20255
          started_at: "2026-10-24T07:58:01.283889Z"
          finished_at: "2026-10-24T08:03:13.892754Z"
          session_id: 01a16cea-adc5-78b3-9958-bc4742719fa7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 67a53ea2b4d26e3ed053150decb566c473e45068
        tracker_head: 9522e4b2d46cfb4b81530d9a824d8f7a887cc1a3
        branch_point: 99846a5015f753f05edaf4778618849f377a2f71
        since: "2026-10-24T00:22:54-07:00"
        until: "2026-10-24T00:57:06-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-24T00:22:54-07:00"
              end: "2026-10-24T00:57:06-07:00"
              began: "2026-10-24T00:22:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 87dc176f2d7e418369f15145d2519b23f80e7690fac54db122e8ee454fb5821b
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 32364506
              windows:
                - start: "2026-10-24T00:22:54-07:00"
                  end: "2026-10-24T00:57:06-07:00"
            - harness: codex
              id: 01a16ccd-42bf-7241-9b61-3796ba74a6cf
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a16862-faa4-76e2-bce3-5bc33019d381
              digest: c07d7c9f93a59a64644afe80ee29fa0f7137d4709f1c58b3a0ab2dc0404dba56
              started: "2026-10-24T07:25:53.817Z"
              spawned: true
              bytes: 664918
              windows:
                - start: "2026-10-24T07:25:53.817Z"
                  end: "2026-10-24T00:57:06-07:00"
        cuts:
            - dispatched: "2026-10-24T00:47:31-07:00"
              finalized: "2026-10-24T00:49:36-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-24T01:04:02-07:00"
      dispatched: "2026-10-24T00:57:58-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: All three native-exit assignments now use pre-construction decorators (native_exit_test.go:38, :152, :166). Twenty race repetitions passed. The new source guard passes on head and fails against the prior native-exit source, identifying exactly the three original writes. The issue's Revisions section records the complete family.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Done-when at workshop/issues/000860-multi-hub-fixture-authority-race.md:53-54 now requires pre-construction child authority, the named proof and native-exit controls, and mechanical enumeration. These requirements match the worker-cue/native-exit fixtures and authority_source_test.go; repeated race checks preserve their assertions.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 67a53ea2b4d26e3ed053150decb566c473e45068
      blocked: false
---

# Gate ledger — pair#860 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T00:49:36-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `child-authority-before-readers` (kind: defect) ARCH-PURPOSE: child-proof publication sweep leaves native-exit fixtures racing
  The child-proof extension at cmd/internal/couchcmd/worker_cue_coordinator_test.go:62 fixes newWorkerCueRig but leaves the same live-wrapper pattern at native_exit_test.go:41, :147 and :167. Running go test -race ./cmd/internal/couchcmd -run '^TestNativeExit(AdmitsOnlyTheCurrentChild|WithoutChildEvidenceIsUnsupported|ReachesTheHubAsOneLifecycleNotice)$' -count=20 -timeout=120s failed: newExitRig's write at :41 races messageService.execute at message_service.go:896, where the value-receiver call copies authority. These are all remaining connected-fixture child assignments; worker_cue_coordinator_test.go:309 is the documented no-wrapper control. Compose all three native-exit variants before service construction through the existing decorator, preserve their assertions, and repeat this regression command. Add a Revisions entry completing the demonstrated child-proof family.
- **BR-2** [Important] `acceptance-criteria-track-contract` (kind: defect) Done-when does not cover the added child-proof contract
  workshop/issues/000860-multi-hub-fixture-authority-race.md:53-57 covers the original notification race, multi-hub startup policy and hub/hubs assignments, but neither child-proof revision at :96 and :104 has corresponding acceptance criteria. These are both instances of the same contract extension, implemented in worker_cue_coordinator_test.go. Update Done-when to require pre-start child-proof composition and repeated race coverage preserving the default, role-change, committed-report and bounded-proof controls, including the native-exit siblings identified above.

## Round 2 — 2026-10-24T01:04:02-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — All three native-exit assignments now use pre-construction decorators (native_exit_test.go:38, :152, :166). Twenty race repetitions passed. The new source guard passes on head and fails against the prior native-exit source, identifying exactly the three original writes. The issue's Revisions section records the complete family.
- BR-2 — addressed — Done-when at workshop/issues/000860-multi-hub-fixture-authority-race.md:53-54 now requires pre-construction child authority, the named proof and native-exit controls, and mechanical enumeration. These requirements match the worker-cue/native-exit fixtures and authority_source_test.go; repeated race checks preserve their assertions.

## Open findings

(none — every finding has been disposed)
