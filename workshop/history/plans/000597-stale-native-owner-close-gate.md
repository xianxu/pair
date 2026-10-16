---
gate: boundary-review
issue: 597
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7eb99c96-b29b-43f8-a649-949a8f9865d6
            repository: github.com/xianxu/pair
            issue_id: "000597"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/fa569a328969e5bb
            reviewed_head: 50597c9ccf496628f74aa8b6c1de229852f0eb64
            slot: pair:2
            requested_at: "2026-10-16T04:49:29.662811Z"
          phase: finished
          pid: 80249
          started_at: "2026-10-16T04:49:29.752688Z"
          finished_at: "2026-10-16T04:51:45.983678Z"
          session_id: b2f87f73-e6be-489d-9843-b447db52ebc2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 50597c9ccf496628f74aa8b6c1de229852f0eb64
        tracker_head: adc329f1a26330a15b107dce074b958a14d84960
        branch_point: 7d96738f8a9034af12459699e30c80eb02f146bc
        since: "2026-10-15T20:32:41-07:00"
        until: "2026-10-15T21:34:00-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:32:41-07:00"
              end: "2026-10-15T20:44:09-07:00"
              began: "2026-10-15T20:32:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-15T20:47:56-07:00"
              end: "2026-10-15T21:34:00-07:00"
              began: "2026-10-15T20:47:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 38f6253328070a3d0a5792e16f6fefa39711f941af854dd873b655990e314e74
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 2214391
              windows:
                - start: "2026-10-15T20:47:56-07:00"
                  end: "2026-10-15T21:34:00-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T20:32:41-07:00"
                  end: "2026-10-15T20:44:09-07:00"
                - start: "2026-10-15T20:47:56-07:00"
                  end: "2026-10-15T21:34:00-07:00"
            - harness: codex
              id: 01a142c6-124f-7491-a52b-beb2cb26f82f
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 9ca992bb23dafdecad96a56dbbb59e90918b18dc1b663443da393b4efe43467b
              started: "2026-10-16T03:33:59.704Z"
              spawned: true
              bytes: 909366
              windows:
                - start: "2026-10-16T03:33:59.704Z"
                  end: "2026-10-15T20:44:09-07:00"
            - harness: codex
              id: 01a142c8-0a54-7ae0-b8e6-c5d900157a3d
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: db2455471dce5c23ba032d6016d44fcec459b83f4dbaaa2dffff17405c4b7a73
              started: "2026-10-16T03:36:08.698Z"
              spawned: true
              bytes: 859979
              windows:
                - start: "2026-10-16T03:36:08.698Z"
                  end: "2026-10-15T20:44:09-07:00"
      "n": 1
      timestamp: "2026-10-15T21:51:46-07:00"
      dispatched: "2026-10-15T21:49:28-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Native admission bypass gated on couchOwned, not Recovery, strips cross-slot protection from ordinary Couch cold resume
          detail: 'createflow.go:824 skips ReserveNativeResume for any Couch-owned explicit resume. couchcore/resume.go:599 (BuildCouchResumeLaunchProfile, Recovery nil) now launches with an empty grant, wrapper startAgentPTY bypasses admission, and that path never calls checkRecoveryConversationInUse, so a conversation live in another slot can be duplicated. The plan says "Couch recovery only". Fix: gate on couchOwned && opts.Args.Recovery != nil, like the Fresh branch at line 658. Family sweep: createflow.go:824; createflow_test.go:541-557 (pins the overbroad behaviour without Recovery); stale comments at createflow.go:821-823 and wrapcmd/native_admission.go:20-23 claiming the registry was checked.'
          family: recovery-only-scope-gate
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: No gate-level test that an exact registry collision refuses at both recovery gates
          detail: 'The old both-gates refusal tests were deleted. recovery_resume_integration_test.go now covers only the non-refusing stale-evidence case, and refusal is pinned only by the unit table. Add preflight and after-prepare cases with the same conversation in another live slot, asserting the "in use by" error, zero Runner ops, and no effects before the first gate. Same family: the Done-when "remains retryable" clause has no test of a refused Fresh followed by a successful retry on the same reserved marker (recovery_fresh_claim_test.go).'
          family: done-when-both-states-coverage
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e3703c8a-ab89-425e-8c99-e297da7fc459
            repository: github.com/xianxu/pair
            issue_id: "000597"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ef117cb7f0a30f23
            reviewed_head: ca8e2ee5d5cbdd713dd5c9878505418a19e0644d
            slot: pair:2
            requested_at: "2026-10-16T05:10:57.205364Z"
          phase: finished
          pid: 39160
          started_at: "2026-10-16T05:10:57.299607Z"
          finished_at: "2026-10-16T05:15:41.026525Z"
          session_id: 0da56c1e-8543-47c9-bc82-083bd679b991
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: ca8e2ee5d5cbdd713dd5c9878505418a19e0644d
        tracker_head: adc329f1a26330a15b107dce074b958a14d84960
        branch_point: 6f50978e0165d6f9ad4e6f17bf598ac0a780a194
        since: "2026-10-15T20:32:41-07:00"
        until: "2026-10-15T21:55:23-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:32:41-07:00"
              end: "2026-10-15T20:44:09-07:00"
              began: "2026-10-15T20:32:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-15T20:47:56-07:00"
              end: "2026-10-15T21:55:23-07:00"
              began: "2026-10-15T20:47:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: d226350feeb96ac7d5b349d1a42c17862563606b80b1cafa2b88fc3a7e232d55
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 2656479
              windows:
                - start: "2026-10-15T20:47:56-07:00"
                  end: "2026-10-15T21:55:23-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T20:32:41-07:00"
                  end: "2026-10-15T20:44:09-07:00"
                - start: "2026-10-15T20:47:56-07:00"
                  end: "2026-10-15T21:55:23-07:00"
            - harness: codex
              id: 01a142c6-124f-7491-a52b-beb2cb26f82f
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 9ca992bb23dafdecad96a56dbbb59e90918b18dc1b663443da393b4efe43467b
              started: "2026-10-16T03:33:59.704Z"
              spawned: true
              bytes: 909366
              windows:
                - start: "2026-10-16T03:33:59.704Z"
                  end: "2026-10-15T20:44:09-07:00"
            - harness: codex
              id: 01a142c8-0a54-7ae0-b8e6-c5d900157a3d
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: db2455471dce5c23ba032d6016d44fcec459b83f4dbaaa2dffff17405c4b7a73
              started: "2026-10-16T03:36:08.698Z"
              spawned: true
              bytes: 859979
              windows:
                - start: "2026-10-16T03:36:08.698Z"
                  end: "2026-10-15T20:44:09-07:00"
        cuts:
            - dispatched: "2026-10-15T21:49:28-07:00"
              finalized: "2026-10-15T21:51:46-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T22:15:41-07:00"
      dispatched: "2026-10-15T22:10:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'The named cold-resume instance is fixed and verified (removing the check at resume.go:601 makes TestColdResumeRefusesConversationLiveInAnotherSlot launch couch-fake-1). The class remains open. createflow.go:824 still skips ReserveNativeResume for every Couch-owned launch with a resolved selector. A Couch spawn built by BuildCouchLaunchProfile (spawnResolved / launchClaimedThread with explicit or path argv) carrying --resume ID, --continue or a picker gets no admission and no registry check; a probe at head gave code=0 reserved=0 launches=1 grantenv="" (base reserved). Rule: skip admission only for launches whose profile proves the registry check ran, i.e. couchOwned && (opts.Args.ResumeRequired || opts.Args.Recovery != nil). Every other Couch-owned selector launch keeps ReserveNativeResume, so the wrapper gets a grant and startAgentPTY goes through startNativePTY. Sites: createflow.go:824 gate; createflow.go:821-823 and wrapcmd/native_admission.go:20-23 comments ("every Couch resume"). Add a createflow test: BuildCouchLaunchProfile plus claude --resume <uuid> under Couch env must reserve admission.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates passes, and its after-prepare case fails with the recovery_prepared.go:388 gate disabled (":3 resumed a conversation live in :2: <nil>"). The preflight case asserts zero prepares. The "retry after refusal" scenario in recovery_fresh_claim_test.go covers the retryable clause with a byte-identical marker check.'
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Minor
          title: atlas/couch.md:93 still presents native admission as the concurrent-writer boundary for Couch
          detail: 'Couch-owned resume and recovery now skip native admission and rely on checkRecoveryConversationInUse (in use by repo:N). Add one sentence naming the registry collision check (#597) and noting that ordinary Pair keeps admission until #595.'
          family: docs-surface-sync
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9644e68c-4bbc-4899-b634-617411dfa540
            repository: github.com/xianxu/pair
            issue_id: "000597"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/d0400c5bf07ca9ea
            reviewed_head: 7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2
            slot: pair:2
            requested_at: "2026-10-16T05:31:44.401955Z"
          phase: finished
          pid: 71499
          started_at: "2026-10-16T05:31:44.502194Z"
          finished_at: "2026-10-16T05:37:48.822164Z"
          session_id: 4950f52c-517f-426c-9748-e0bc7dd51d5e
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2
        tracker_head: adc329f1a26330a15b107dce074b958a14d84960
        branch_point: 6f50978e0165d6f9ad4e6f17bf598ac0a780a194
        since: "2026-10-15T20:32:41-07:00"
        until: "2026-10-15T22:17:27-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T20:32:41-07:00"
              end: "2026-10-15T20:44:09-07:00"
              began: "2026-10-15T20:32:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-15T20:47:56-07:00"
              end: "2026-10-15T22:17:27-07:00"
              began: "2026-10-15T20:47:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 22a6052d3550bf1ef7eaa48d65a6de0c286ea16d5796959d8a7dd03f0cc57ff8
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 2912271
              windows:
                - start: "2026-10-15T20:47:56-07:00"
                  end: "2026-10-15T22:17:27-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T20:32:41-07:00"
                  end: "2026-10-15T20:44:09-07:00"
                - start: "2026-10-15T20:47:56-07:00"
                  end: "2026-10-15T22:17:27-07:00"
            - harness: codex
              id: 01a142c6-124f-7491-a52b-beb2cb26f82f
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 9ca992bb23dafdecad96a56dbbb59e90918b18dc1b663443da393b4efe43467b
              started: "2026-10-16T03:33:59.704Z"
              spawned: true
              bytes: 909366
              windows:
                - start: "2026-10-16T03:33:59.704Z"
                  end: "2026-10-15T20:44:09-07:00"
            - harness: codex
              id: 01a142c8-0a54-7ae0-b8e6-c5d900157a3d
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: db2455471dce5c23ba032d6016d44fcec459b83f4dbaaa2dffff17405c4b7a73
              started: "2026-10-16T03:36:08.698Z"
              spawned: true
              bytes: 859979
              windows:
                - start: "2026-10-16T03:36:08.698Z"
                  end: "2026-10-15T20:44:09-07:00"
        cuts:
            - dispatched: "2026-10-15T21:49:28-07:00"
              finalized: "2026-10-15T21:51:46-07:00"
              finalizing: false
            - dispatched: "2026-10-15T22:10:55-07:00"
              finalized: "2026-10-15T22:15:41-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-15T22:37:49-07:00"
      dispatched: "2026-10-15T22:31:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: createflow.go:825 gates on couchOwned && (ResumeRequired || Recovery != nil); both builders' sole callers run checkRecoveryConversationInUse first (resume.go:601, recovery_prepared.go:388); wrapper bypass narrowed to pinned selector; both regression tests verified red on revert.
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'atlas/couch.md:93-97 now names the registry collision check, the in-use message, and that other launches plus ordinary Pair keep admission until #595.'
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: 7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2
      blocked: false
---

# Gate ledger — pair#597 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T21:51:46-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `recovery-only-scope-gate` (kind: defect) Native admission bypass gated on couchOwned, not Recovery, strips cross-slot protection from ordinary Couch cold resume
  createflow.go:824 skips ReserveNativeResume for any Couch-owned explicit resume. couchcore/resume.go:599 (BuildCouchResumeLaunchProfile, Recovery nil) now launches with an empty grant, wrapper startAgentPTY bypasses admission, and that path never calls checkRecoveryConversationInUse, so a conversation live in another slot can be duplicated. The plan says "Couch recovery only". Fix: gate on couchOwned && opts.Args.Recovery != nil, like the Fresh branch at line 658. Family sweep: createflow.go:824; createflow_test.go:541-557 (pins the overbroad behaviour without Recovery); stale comments at createflow.go:821-823 and wrapcmd/native_admission.go:20-23 claiming the registry was checked.
- **BR-2** [Important] `done-when-both-states-coverage` (kind: defect) No gate-level test that an exact registry collision refuses at both recovery gates
  The old both-gates refusal tests were deleted. recovery_resume_integration_test.go now covers only the non-refusing stale-evidence case, and refusal is pinned only by the unit table. Add preflight and after-prepare cases with the same conversation in another live slot, asserting the "in use by" error, zero Runner ops, and no effects before the first gate. Same family: the Done-when "remains retryable" clause has no test of a refused Fresh followed by a successful retry on the same reserved marker (recovery_fresh_claim_test.go).

## Round 2 — 2026-10-15T22:15:41-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The named cold-resume instance is fixed and verified (removing the check at resume.go:601 makes TestColdResumeRefusesConversationLiveInAnotherSlot launch couch-fake-1). The class remains open. createflow.go:824 still skips ReserveNativeResume for every Couch-owned launch with a resolved selector. A Couch spawn built by BuildCouchLaunchProfile (spawnResolved / launchClaimedThread with explicit or path argv) carrying --resume ID, --continue or a picker gets no admission and no registry check; a probe at head gave code=0 reserved=0 launches=1 grantenv="" (base reserved). Rule: skip admission only for launches whose profile proves the registry check ran, i.e. couchOwned && (opts.Args.ResumeRequired || opts.Args.Recovery != nil). Every other Couch-owned selector launch keeps ReserveNativeResume, so the wrapper gets a grant and startAgentPTY goes through startNativePTY. Sites: createflow.go:824 gate; createflow.go:821-823 and wrapcmd/native_admission.go:20-23 comments ("every Couch resume"). Add a createflow test: BuildCouchLaunchProfile plus claude --resume <uuid> under Couch env must reserve admission.
- BR-2 — addressed — TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates passes, and its after-prepare case fails with the recovery_prepared.go:388 gate disabled (":3 resumed a conversation live in :2: <nil>"). The preflight case asserts zero prepares. The "retry after refusal" scenario in recovery_fresh_claim_test.go covers the retryable clause with a byte-identical marker check.

### Raised

- **BR-3** [Minor] `docs-surface-sync` (kind: defect) atlas/couch.md:93 still presents native admission as the concurrent-writer boundary for Couch
  Couch-owned resume and recovery now skip native admission and rely on checkRecoveryConversationInUse (in use by repo:N). Add one sentence naming the registry collision check (#597) and noting that ordinary Pair keeps admission until #595.

## Round 3 — 2026-10-15T22:37:49-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — createflow.go:825 gates on couchOwned && (ResumeRequired || Recovery != nil); both builders' sole callers run checkRecoveryConversationInUse first (resume.go:601, recovery_prepared.go:388); wrapper bypass narrowed to pinned selector; both regression tests verified red on revert.
- BR-3 — addressed — atlas/couch.md:93-97 now names the registry collision check, the in-use message, and that other launches plus ordinary Pair keep admission until #595.

## Open findings

(none — every finding has been disposed)
