---
gate: boundary-review
issue: 579
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c86498bf-dff7-434c-820d-d058f439327e
            repository: github.com/xianxu/pair
            issue_id: "000579"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/252bef2d378bc31d
            reviewed_head: b13ee4118af26fb38f981c319255d7f7bc4e0a13
            slot: pair:3
            requested_at: "2026-10-15T23:04:52.074265Z"
          phase: finished
          pid: 83077
          started_at: "2026-10-15T23:04:52.230515Z"
          finished_at: "2026-10-15T23:07:46.430936Z"
          session_id: 01a141cf-b25b-7fa3-8966-1276c481fff8
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b13ee4118af26fb38f981c319255d7f7bc4e0a13
        tracker_head: 189d1219f8b96523d662d7aad3ec68eef8dd05c7
        branch_point: 32a0818a32ffb6ae4764aa587a3193959a4ad295
        since: "2026-10-15T15:07:36-07:00"
        until: "2026-10-15T16:04:20-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T15:07:36-07:00"
              end: "2026-10-15T16:04:20-07:00"
              began: "2026-10-15T15:07:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a13b1c-d331-7843-9659-349166408f6a
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 1c2eb01881ce2ad81775190b756672600e768677df78578996f7cd0a191fcead
              started: "2026-10-14T15:51:55.914Z"
              spawned: false
              bytes: 18993990
              windows:
                - start: "2026-10-15T15:07:36-07:00"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a1419c-2ee2-7c50-8a0e-e7b7aaeb911a
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: 5460e40eee05f0f66056f8ac068b947adf38baacf477a7e671b16476b4412194
              started: "2026-10-15T22:08:37.171Z"
              spawned: true
              bytes: 1422428
              windows:
                - start: "2026-10-15T22:08:37.171Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141ac-daf7-7032-bbcc-e6b476eca0b7
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 3227328a1285ef02c985c34d087d2a717c37a0bfb3e272d687916181624a7443
              started: "2026-10-15T22:26:51.056Z"
              spawned: false
              bytes: 747864
              windows:
                - start: "2026-10-15T22:26:51.056Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141ae-da56-79e0-a4e2-5ab971fc5793
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d1c7f8e6fd3001343dc5bc156df924d12678f1725396abaa8c543d12b42fb8cb
              started: "2026-10-15T22:29:03.092Z"
              spawned: false
              bytes: 461537
              windows:
                - start: "2026-10-15T22:29:03.092Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141b0-22e9-7491-978e-42ac3c0a7c62
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 2d54ab9a00f18a9b5d6f0977c062201157af5f2dcbc38fba4e417ba3008d4e30
              started: "2026-10-15T22:30:26.425Z"
              spawned: false
              bytes: 481392
              windows:
                - start: "2026-10-15T22:30:26.425Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141b0-5194-7760-9a2c-9568e3c720d0
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: 3f95268d9e5c23333ba685b7c7202f23e0edd23d6af947f9a0326d6063e95693
              started: "2026-10-15T22:30:36.78Z"
              spawned: true
              bytes: 1487052
              windows:
                - start: "2026-10-15T22:30:36.78Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141b0-6634-7060-9775-9205e269605e
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: fd7071d39fa058673a0124383b4249b7a17238bbb4a546cf950966f0ee72174f
              started: "2026-10-15T22:30:42.049Z"
              spawned: true
              bytes: 875171
              windows:
                - start: "2026-10-15T22:30:42.049Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141b0-8f06-7e72-8ed9-64544f5f52b1
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: e384c6699051c1af64d02f24f7205c6b7c4e71154aced5fb51b31c0015536557
              started: "2026-10-15T22:30:53.827Z"
              spawned: false
              bytes: 462408
              windows:
                - start: "2026-10-15T22:30:53.827Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141bc-9fc5-7d30-8bbf-663e7fd398f2
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 40404e344153083320fdff3c3a21109a0c70f94c5ca9d660aa371f6a4f3b73df
              started: "2026-10-15T22:44:03.439Z"
              spawned: false
              bytes: 436934
              windows:
                - start: "2026-10-15T22:44:03.439Z"
                  end: "2026-10-15T16:04:20-07:00"
            - harness: codex
              id: 01a141bc-fa2d-78b1-8e54-4d85c4f3a21b
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 554c808b6bb64fb2a812693bf6f6cc29a2ee9061e1e791e44c5200f7a5d0dc71
              started: "2026-10-15T22:44:26.527Z"
              spawned: false
              bytes: 459979
              windows:
                - start: "2026-10-15T22:44:26.527Z"
                  end: "2026-10-15T16:04:20-07:00"
      "n": 1
      timestamp: "2026-10-15T16:07:47-07:00"
      dispatched: "2026-10-15T16:04:49-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Drain capture before stopping its flush owner
          detail: cmd/internal/wrapcmd/wrap.go:3144 stops the capture ticker when masterPump returns, but capture.Close runs only after run returns, beyond cmd.Wait at line 3041. A child that closes its PTY and keeps running leaves accepted capture bytes buffered indefinitely despite healthy IO, violating the declared 250ms envelope (ARCH-CONSTRAINTS, ARCH-ORDER). A temporary overlay regression reproduced pump completion with offset=12 and an empty raw sink. Flush capture on every pump exit before waiting for the child, report failures, and retain a deterministic EOF-before-tick regression.
          family: drain-before-owner-stop
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 02b08413-075b-4baf-8715-7ad5e37fd6fd
            repository: github.com/xianxu/pair
            issue_id: "000579"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/3b6b0ac70846a5ed
            reviewed_head: aa22ef251aaa6d0510e4e9a9c480cb506dd005b2
            slot: pair:3
            requested_at: "2026-10-15T23:10:04.444581Z"
          phase: finished
          pid: 86366
          started_at: "2026-10-15T23:10:04.629976Z"
          finished_at: "2026-10-15T23:12:41.627737Z"
          session_id: 01a141d4-7466-7cd2-ad0d-b9e233dae31a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: aa22ef251aaa6d0510e4e9a9c480cb506dd005b2
        tracker_head: 189d1219f8b96523d662d7aad3ec68eef8dd05c7
        branch_point: 32a0818a32ffb6ae4764aa587a3193959a4ad295
        since: "2026-10-15T15:07:36-07:00"
        until: "2026-10-15T16:09:33-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T15:07:36-07:00"
              end: "2026-10-15T16:09:33-07:00"
              began: "2026-10-15T15:07:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a13b1c-d331-7843-9659-349166408f6a
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 996d3c036e30c5b3b16a5fb46f990fa80070927012916d2cab3981c1336bae1d
              started: "2026-10-14T15:51:55.914Z"
              spawned: false
              bytes: 19364589
              windows:
                - start: "2026-10-15T15:07:36-07:00"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a1419c-2ee2-7c50-8a0e-e7b7aaeb911a
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: 5460e40eee05f0f66056f8ac068b947adf38baacf477a7e671b16476b4412194
              started: "2026-10-15T22:08:37.171Z"
              spawned: true
              bytes: 1422428
              windows:
                - start: "2026-10-15T22:08:37.171Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141ac-daf7-7032-bbcc-e6b476eca0b7
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 3227328a1285ef02c985c34d087d2a717c37a0bfb3e272d687916181624a7443
              started: "2026-10-15T22:26:51.056Z"
              spawned: false
              bytes: 747864
              windows:
                - start: "2026-10-15T22:26:51.056Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141ae-da56-79e0-a4e2-5ab971fc5793
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d1c7f8e6fd3001343dc5bc156df924d12678f1725396abaa8c543d12b42fb8cb
              started: "2026-10-15T22:29:03.092Z"
              spawned: false
              bytes: 461537
              windows:
                - start: "2026-10-15T22:29:03.092Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141b0-22e9-7491-978e-42ac3c0a7c62
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 2d54ab9a00f18a9b5d6f0977c062201157af5f2dcbc38fba4e417ba3008d4e30
              started: "2026-10-15T22:30:26.425Z"
              spawned: false
              bytes: 481392
              windows:
                - start: "2026-10-15T22:30:26.425Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141b0-5194-7760-9a2c-9568e3c720d0
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: 3f95268d9e5c23333ba685b7c7202f23e0edd23d6af947f9a0326d6063e95693
              started: "2026-10-15T22:30:36.78Z"
              spawned: true
              bytes: 1487052
              windows:
                - start: "2026-10-15T22:30:36.78Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141b0-6634-7060-9775-9205e269605e
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a13b1c-d331-7843-9659-349166408f6a
              digest: fd7071d39fa058673a0124383b4249b7a17238bbb4a546cf950966f0ee72174f
              started: "2026-10-15T22:30:42.049Z"
              spawned: true
              bytes: 875171
              windows:
                - start: "2026-10-15T22:30:42.049Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141b0-8f06-7e72-8ed9-64544f5f52b1
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: e384c6699051c1af64d02f24f7205c6b7c4e71154aced5fb51b31c0015536557
              started: "2026-10-15T22:30:53.827Z"
              spawned: false
              bytes: 462408
              windows:
                - start: "2026-10-15T22:30:53.827Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141bc-9fc5-7d30-8bbf-663e7fd398f2
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 40404e344153083320fdff3c3a21109a0c70f94c5ca9d660aa371f6a4f3b73df
              started: "2026-10-15T22:44:03.439Z"
              spawned: false
              bytes: 436934
              windows:
                - start: "2026-10-15T22:44:03.439Z"
                  end: "2026-10-15T16:09:33-07:00"
            - harness: codex
              id: 01a141bc-fa2d-78b1-8e54-4d85c4f3a21b
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 554c808b6bb64fb2a812693bf6f6cc29a2ee9061e1e791e44c5200f7a5d0dc71
              started: "2026-10-15T22:44:26.527Z"
              spawned: false
              bytes: 459979
              windows:
                - start: "2026-10-15T22:44:26.527Z"
                  end: "2026-10-15T16:09:33-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T16:04:49-07:00"
              finalized: "2026-10-15T16:07:47-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T16:13:02-07:00"
      dispatched: "2026-10-15T16:10:02-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: wrap.go:3077 flushes capture on every pump exit. TestMasterPumpDrainsCaptureBeforeReturningOnEOF passes at head; removing the defer in a temporary overlay makes both drainage and error-reporting cases fail.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Important
          title: Add the promised registry durability-retry regression
          detail: cmd/internal/diagnosticlog/registry_test.go:247 checks unchanged declarations and inode preservation, but never exercises publication followed by failed directory sync and retry. The M1 plan explicitly requires this case at workshop/plans/000579-wrapper-log-io-plan.md:120. Replacing registry.go:62's durability reconfirmation with return nil still passes every TestRegistry test. Add an injected sync-failure sequence proving an equal declaration reconfirms durability, propagates sync failure, and avoids content replacement.
          family: durability-reconfirmation
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#579 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T16:07:47-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `drain-before-owner-stop` (kind: defect) Drain capture before stopping its flush owner
  cmd/internal/wrapcmd/wrap.go:3144 stops the capture ticker when masterPump returns, but capture.Close runs only after run returns, beyond cmd.Wait at line 3041. A child that closes its PTY and keeps running leaves accepted capture bytes buffered indefinitely despite healthy IO, violating the declared 250ms envelope (ARCH-CONSTRAINTS, ARCH-ORDER). A temporary overlay regression reproduced pump completion with offset=12 and an empty raw sink. Flush capture on every pump exit before waiting for the child, report failures, and retain a deterministic EOF-before-tick regression.

## Round 2 — 2026-10-15T16:13:02-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — wrap.go:3077 flushes capture on every pump exit. TestMasterPumpDrainsCaptureBeforeReturningOnEOF passes at head; removing the defer in a temporary overlay makes both drainage and error-reporting cases fail.

### Raised

- **BR-2** [Important] `durability-reconfirmation` (kind: defect) Add the promised registry durability-retry regression
  cmd/internal/diagnosticlog/registry_test.go:247 checks unchanged declarations and inode preservation, but never exercises publication followed by failed directory sync and retry. The M1 plan explicitly requires this case at workshop/plans/000579-wrapper-log-io-plan.md:120. Replacing registry.go:62's durability reconfirmation with return nil still passes every TestRegistry test. Add an injected sync-failure sequence proving an equal declaration reconfirms durability, propagates sync failure, and avoids content replacement.

## Open findings

- **BR-2** [Important] `durability-reconfirmation` (kind: defect) Add the promised registry durability-retry regression
