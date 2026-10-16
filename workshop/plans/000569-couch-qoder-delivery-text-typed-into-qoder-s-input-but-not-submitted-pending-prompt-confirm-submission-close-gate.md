---
gate: boundary-review
issue: 569
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8061042d-0637-41f6-9d13-b203d22b9d0a
            repository: github.com/xianxu/pair
            issue_id: "000569"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/13e8512e56e7eb33
            reviewed_head: 35c420d4db5f19ea1f573400ebc2614949e9602d
            slot: pair:5
            requested_at: "2026-10-16T00:20:25.462939Z"
          phase: finished
          pid: 3041
          started_at: "2026-10-16T00:20:25.559614Z"
          finished_at: "2026-10-16T00:22:14.348432Z"
          session_id: 01a14214-dc3a-7782-875a-7a5d464f5627
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 35c420d4db5f19ea1f573400ebc2614949e9602d
        tracker_head: 6f59b1d966560e05c0bb3887995ce3fb23c436e5
        branch_point: 99e11ecb8c72b7e5325a58692c1efd679e50710c
        since: "2026-10-15T16:14:38-07:00"
        until: "2026-10-15T17:20:09-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T16:14:38-07:00"
              end: "2026-10-15T17:20:09-07:00"
              began: "2026-10-15T16:14:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 8c4c0a929bc13b8db0a36b1ff2e2fbb90045f4c2c228b753251ac993454ef11d
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 22325721
              windows:
                - start: "2026-10-15T16:14:38-07:00"
                  end: "2026-10-15T17:20:09-07:00"
            - harness: codex
              id: 01a141d9-00d4-7132-96db-bd5040a227cc
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: a2aa3f5463c058f05d0a4af43459bc2d6b49c2cf2823721a2f932c6f14f56375
              started: "2026-10-15T23:15:03.377Z"
              spawned: true
              bytes: 961452
              windows:
                - start: "2026-10-15T23:15:03.377Z"
                  end: "2026-10-15T17:20:09-07:00"
            - harness: codex
              id: 01a141de-3408-77b1-b13f-6e7b5245bb71
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: bb7c1d15c6915085faf91d0a78e5b84e863aeeeef9b56cb830bc0d30e0694008
              started: "2026-10-15T23:20:44.041Z"
              spawned: false
              bytes: 600067
              windows:
                - start: "2026-10-15T23:20:44.041Z"
                  end: "2026-10-15T17:20:09-07:00"
            - harness: codex
              id: 01a141e0-11df-7cb3-a455-ea49113b1da0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 3052f2043733071d96fbfee24f9e902cdb15083bb043d651dddc5cbded35393e
              started: "2026-10-15T23:22:46.379Z"
              spawned: false
              bytes: 429790
              windows:
                - start: "2026-10-15T23:22:46.379Z"
                  end: "2026-10-15T17:20:09-07:00"
            - harness: codex
              id: 01a141e0-dd24-7bd1-a2b7-830a35a14f19
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 1ef92498c78ddcc379ea0adf41831a5e3539c22ac5b8add70dce01201e117606
              started: "2026-10-15T23:23:38.456Z"
              spawned: false
              bytes: 430990
              windows:
                - start: "2026-10-15T23:23:38.456Z"
                  end: "2026-10-15T17:20:09-07:00"
            - harness: codex
              id: 01a141e1-98d3-7063-8f57-e82ee71361f2
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 2e710aa5677883fc47ba3e7ea4452705a12d5e529beb44c4b4c4859765f2d736
              started: "2026-10-15T23:24:26.439Z"
              spawned: false
              bytes: 383674
              windows:
                - start: "2026-10-15T23:24:26.439Z"
                  end: "2026-10-15T17:20:09-07:00"
            - harness: codex
              id: 01a141e2-a62d-7b10-b115-3fb86faaaec0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 6959763d5a9a8bd3c81923765b3601cffdf3dbd3ddd8ab221e09ee667cce83d4
              started: "2026-10-15T23:25:35.284Z"
              spawned: true
              bytes: 493711
              windows:
                - start: "2026-10-15T23:25:35.284Z"
                  end: "2026-10-15T17:20:09-07:00"
      "n": 1
      timestamp: "2026-10-15T17:22:18-07:00"
      dispatched: "2026-10-15T17:20:24-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Recovery notices instruct Enter, which inserts a newline in Qoder
          detail: cmd/internal/couchcmd/automatic_input_stall.go:63 and cmd/internal/couchmessage/delivery.go:89–91 tell the operator to press Enter. The Qoder profile in cmd/internal/wrapcmd/harness_tty.go:90–98 maps plain Enter to backslash-CR for a newline in a recognized composer; Alt+Enter emits the submission CR. Following the notice therefore leaves the automatic message pending. Correct all new recovery instructions and add a regression that connects the prescribed recovery action to production input routing rather than only asserting the notice string. ARCH-PURPOSE.
          family: recovery-instructions-match-input-routing
          round: 1
        - id: BR-2
          severity: Important
          title: README update is missing for changed confirmation and operator recovery
          detail: README.md:724–736 still says a newly opened turn can confirm submission, says the wrapper does not reconstruct the envelope, and gives no quarantine recovery guidance. This range changes all three for Qoder and introduces an operator action, but does not update README.md. Document the Qoder exception, bounded key-only retries, prompt refusal during quarantine, and the correct manual recovery action. atlas/couch.md was updated appropriately.
          family: user-facing-contract-documentation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 87c96b05-0f37-4ab5-9e75-1f51fb8aa40e
            repository: github.com/xianxu/pair
            issue_id: "000569"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/6ab8c5ffa150519f
            reviewed_head: aa4c2240dec53d735a567e90105b45b60c621433
            slot: pair:5
            requested_at: "2026-10-16T01:13:08.03883Z"
          phase: finished
          pid: 17889
          started_at: "2026-10-16T01:13:08.236875Z"
          finished_at: "2026-10-16T01:15:19.710686Z"
          session_id: 01a14245-1f60-7d11-89c4-af4e610d3d9d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: aa4c2240dec53d735a567e90105b45b60c621433
        tracker_head: dc8e92a83f01ab22df28dd0678f2103020b30f35
        branch_point: 99e11ecb8c72b7e5325a58692c1efd679e50710c
        since: "2026-10-15T16:14:38-07:00"
        until: "2026-10-15T18:12:08-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T16:14:38-07:00"
              end: "2026-10-15T18:12:08-07:00"
              began: "2026-10-15T16:14:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: f6a9668a9b7c1eae9a4862e359ca50cd799cd1332f8c00d5e2774cc915c4bbef
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 24102489
              windows:
                - start: "2026-10-15T16:14:38-07:00"
                  end: "2026-10-15T18:12:08-07:00"
            - harness: codex
              id: 01a141d9-00d4-7132-96db-bd5040a227cc
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: a2aa3f5463c058f05d0a4af43459bc2d6b49c2cf2823721a2f932c6f14f56375
              started: "2026-10-15T23:15:03.377Z"
              spawned: true
              bytes: 961452
              windows:
                - start: "2026-10-15T23:15:03.377Z"
                  end: "2026-10-15T18:12:08-07:00"
            - harness: codex
              id: 01a141de-3408-77b1-b13f-6e7b5245bb71
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: bb7c1d15c6915085faf91d0a78e5b84e863aeeeef9b56cb830bc0d30e0694008
              started: "2026-10-15T23:20:44.041Z"
              spawned: false
              bytes: 600067
              windows:
                - start: "2026-10-15T23:20:44.041Z"
                  end: "2026-10-15T18:12:08-07:00"
            - harness: codex
              id: 01a141e0-11df-7cb3-a455-ea49113b1da0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 3052f2043733071d96fbfee24f9e902cdb15083bb043d651dddc5cbded35393e
              started: "2026-10-15T23:22:46.379Z"
              spawned: false
              bytes: 429790
              windows:
                - start: "2026-10-15T23:22:46.379Z"
                  end: "2026-10-15T18:12:08-07:00"
            - harness: codex
              id: 01a141e0-dd24-7bd1-a2b7-830a35a14f19
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 1ef92498c78ddcc379ea0adf41831a5e3539c22ac5b8add70dce01201e117606
              started: "2026-10-15T23:23:38.456Z"
              spawned: false
              bytes: 430990
              windows:
                - start: "2026-10-15T23:23:38.456Z"
                  end: "2026-10-15T18:12:08-07:00"
            - harness: codex
              id: 01a141e1-98d3-7063-8f57-e82ee71361f2
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 2e710aa5677883fc47ba3e7ea4452705a12d5e529beb44c4b4c4859765f2d736
              started: "2026-10-15T23:24:26.439Z"
              spawned: false
              bytes: 383674
              windows:
                - start: "2026-10-15T23:24:26.439Z"
                  end: "2026-10-15T18:12:08-07:00"
            - harness: codex
              id: 01a141e2-a62d-7b10-b115-3fb86faaaec0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 6959763d5a9a8bd3c81923765b3601cffdf3dbd3ddd8ab221e09ee667cce83d4
              started: "2026-10-15T23:25:35.284Z"
              spawned: true
              bytes: 493711
              windows:
                - start: "2026-10-15T23:25:35.284Z"
                  end: "2026-10-15T18:12:08-07:00"
        cuts:
            - dispatched: "2026-10-15T17:20:24-07:00"
              finalized: "2026-10-15T17:22:18-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T18:15:20-07:00"
      dispatched: "2026-10-15T18:13:05-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: AutomaticInputRecoveryHint supplies Alt+Enter advice to both receipts and supervisor notices. TestPeerQoderRecoveryAdviceRoutesToNativeSubmit follows that advice through production translateChunk and verifies consumption and quarantine release. In a disposable pinned-head copy, restoring Enter makes the test fail because routing emits backslash-CR.
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md:724–738 now documents fresh-empty Qoder confirmation, bounded exact-envelope key retries, prompt quarantine refusal and Alt+Enter recovery. README.md:747 distinguishes uncertain body replay. These passages agree with peer_delivery.go, peer_qoder_submission.go and the Qoder keymap in harness_tty.go:90–98.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: aa4c2240dec53d735a567e90105b45b60c621433
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 76e9062f-d57c-485e-bf68-a6c971e8003b
            repository: github.com/xianxu/pair
            issue_id: "000569"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7c4dd260c4d9c99c
            reviewed_head: 277e5d5e7704b3fc18ad3ba9a84850b9fa9d1e72
            slot: pair:5
            requested_at: "2026-10-16T03:12:59.79744Z"
          phase: finished
          pid: 70293
          started_at: "2026-10-16T03:12:59.984093Z"
          finished_at: "2026-10-16T03:14:48.417617Z"
          session_id: 01a142b2-dcab-70c2-854a-12388676da6c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 277e5d5e7704b3fc18ad3ba9a84850b9fa9d1e72
        tracker_head: efdf966b6e15bc508c6947812576740425a16e60
        branch_point: 763eb19fe001266de0a94884e9a5ccaaa94766c3
        since: "2026-10-15T16:14:38-07:00"
        until: "2026-10-15T19:50:22-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T16:14:38-07:00"
              end: "2026-10-15T19:50:22-07:00"
              began: "2026-10-15T16:14:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 38e177727d900ec6fb95339bae2bc3e6ded57b46046c461d5a54fe4a5192fcc7
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 25062902
              windows:
                - start: "2026-10-15T16:14:38-07:00"
                  end: "2026-10-15T19:50:22-07:00"
            - harness: codex
              id: 01a141d9-00d4-7132-96db-bd5040a227cc
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: a2aa3f5463c058f05d0a4af43459bc2d6b49c2cf2823721a2f932c6f14f56375
              started: "2026-10-15T23:15:03.377Z"
              spawned: true
              bytes: 961452
              windows:
                - start: "2026-10-15T23:15:03.377Z"
                  end: "2026-10-15T19:50:22-07:00"
            - harness: codex
              id: 01a141de-3408-77b1-b13f-6e7b5245bb71
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: bb7c1d15c6915085faf91d0a78e5b84e863aeeeef9b56cb830bc0d30e0694008
              started: "2026-10-15T23:20:44.041Z"
              spawned: false
              bytes: 600067
              windows:
                - start: "2026-10-15T23:20:44.041Z"
                  end: "2026-10-15T19:50:22-07:00"
            - harness: codex
              id: 01a141e0-11df-7cb3-a455-ea49113b1da0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 3052f2043733071d96fbfee24f9e902cdb15083bb043d651dddc5cbded35393e
              started: "2026-10-15T23:22:46.379Z"
              spawned: false
              bytes: 429790
              windows:
                - start: "2026-10-15T23:22:46.379Z"
                  end: "2026-10-15T19:50:22-07:00"
            - harness: codex
              id: 01a141e0-dd24-7bd1-a2b7-830a35a14f19
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 1ef92498c78ddcc379ea0adf41831a5e3539c22ac5b8add70dce01201e117606
              started: "2026-10-15T23:23:38.456Z"
              spawned: false
              bytes: 430990
              windows:
                - start: "2026-10-15T23:23:38.456Z"
                  end: "2026-10-15T19:50:22-07:00"
            - harness: codex
              id: 01a141e1-98d3-7063-8f57-e82ee71361f2
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 2e710aa5677883fc47ba3e7ea4452705a12d5e529beb44c4b4c4859765f2d736
              started: "2026-10-15T23:24:26.439Z"
              spawned: false
              bytes: 383674
              windows:
                - start: "2026-10-15T23:24:26.439Z"
                  end: "2026-10-15T19:50:22-07:00"
            - harness: codex
              id: 01a141e2-a62d-7b10-b115-3fb86faaaec0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 6959763d5a9a8bd3c81923765b3601cffdf3dbd3ddd8ab221e09ee667cce83d4
              started: "2026-10-15T23:25:35.284Z"
              spawned: true
              bytes: 493711
              windows:
                - start: "2026-10-15T23:25:35.284Z"
                  end: "2026-10-15T19:50:22-07:00"
        cuts:
            - dispatched: "2026-10-15T17:20:24-07:00"
              finalized: "2026-10-15T17:22:18-07:00"
              finalizing: false
            - dispatched: "2026-10-15T18:13:05-07:00"
              finalized: "2026-10-15T18:15:20-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-15T20:14:50-07:00"
      dispatched: "2026-10-15T20:12:57-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: AutomaticInputRecoveryHint supplies Alt+Enter instructions to receipts and supervisor notices. The routing regression passes; restoring Enter in a prepared disposable checkout makes it fail because translation emits backslash-CR.
          round: 3
        - id: BR-2
          disposition: addressed
          note: README.md:724–738 documents fresh-empty confirmation, bounded key-only retries, quarantine and Alt+Enter recovery; line 747 distinguishes body replay. These passages remain consistent with peer_delivery.go and harness_tty.go:90–98.
          round: 3
      findings:
        - id: BR-3
          severity: Critical
          title: scope changed after close — run a full `sdlc close`
          detail: workshop/plans/000569-qoder-submit-confirmation-plan.md:138–140 adds a Revisions entry authorizing landing before the outstanding native idle/mid-turn/subsequent-delivery checks, superseding the previously reviewed landing hold. This changes the acceptance sequence beyond recording a fix. The fix-only scope rule requires a full sdlc close to assess that revised boundary; this narrow review cannot approve it.
          family: fix-review-preserves-approved-scope
          round: 3
      recipe: fix-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#569 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T17:22:18-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `recovery-instructions-match-input-routing` Recovery notices instruct Enter, which inserts a newline in Qoder
  cmd/internal/couchcmd/automatic_input_stall.go:63 and cmd/internal/couchmessage/delivery.go:89–91 tell the operator to press Enter. The Qoder profile in cmd/internal/wrapcmd/harness_tty.go:90–98 maps plain Enter to backslash-CR for a newline in a recognized composer; Alt+Enter emits the submission CR. Following the notice therefore leaves the automatic message pending. Correct all new recovery instructions and add a regression that connects the prescribed recovery action to production input routing rather than only asserting the notice string. ARCH-PURPOSE.
- **BR-2** [Important] `user-facing-contract-documentation` README update is missing for changed confirmation and operator recovery
  README.md:724–736 still says a newly opened turn can confirm submission, says the wrapper does not reconstruct the envelope, and gives no quarantine recovery guidance. This range changes all three for Qoder and introduces an operator action, but does not update README.md. Document the Qoder exception, bounded key-only retries, prompt refusal during quarantine, and the correct manual recovery action. atlas/couch.md was updated appropriately.

## Round 2 — 2026-10-15T18:15:20-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — AutomaticInputRecoveryHint supplies Alt+Enter advice to both receipts and supervisor notices. TestPeerQoderRecoveryAdviceRoutesToNativeSubmit follows that advice through production translateChunk and verifies consumption and quarantine release. In a disposable pinned-head copy, restoring Enter makes the test fail because routing emits backslash-CR.
- BR-2 — addressed — README.md:724–738 now documents fresh-empty Qoder confirmation, bounded exact-envelope key retries, prompt quarantine refusal and Alt+Enter recovery. README.md:747 distinguishes uncertain body replay. These passages agree with peer_delivery.go, peer_qoder_submission.go and the Qoder keymap in harness_tty.go:90–98.

## Round 3 — 2026-10-15T20:14:50-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — AutomaticInputRecoveryHint supplies Alt+Enter instructions to receipts and supervisor notices. The routing regression passes; restoring Enter in a prepared disposable checkout makes it fail because translation emits backslash-CR.
- BR-2 — addressed — README.md:724–738 documents fresh-empty confirmation, bounded key-only retries, quarantine and Alt+Enter recovery; line 747 distinguishes body replay. These passages remain consistent with peer_delivery.go and harness_tty.go:90–98.

### Raised

- **BR-3** [Critical] `fix-review-preserves-approved-scope` scope changed after close — run a full `sdlc close`
  workshop/plans/000569-qoder-submit-confirmation-plan.md:138–140 adds a Revisions entry authorizing landing before the outstanding native idle/mid-turn/subsequent-delivery checks, superseding the previously reviewed landing hold. This changes the acceptance sequence beyond recording a fix. The fix-only scope rule requires a full sdlc close to assess that revised boundary; this narrow review cannot approve it.

## Open findings

- **BR-3** [Critical] `fix-review-preserves-approved-scope` scope changed after close — run a full `sdlc close`
