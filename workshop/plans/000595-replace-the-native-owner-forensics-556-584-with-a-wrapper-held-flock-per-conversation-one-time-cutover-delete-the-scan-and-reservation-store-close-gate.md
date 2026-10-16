---
gate: boundary-review
issue: 595
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 0312b5fe-d8a9-46d1-868f-608248ac6572
            repository: github.com/xianxu/pair
            issue_id: "000595"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/db47110353867fed
            reviewed_head: 71b211cc49811f3df0676c657e181578765cd7f6
            slot: pair:2
            requested_at: "2026-10-16T10:51:45.547552Z"
          phase: finished
          pid: 22006
          started_at: "2026-10-16T10:51:46.012238Z"
          finished_at: "2026-10-16T11:07:20.651513Z"
          session_id: 9963600e-ec74-4ba4-9605-b9280c0825ca
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 71b211cc49811f3df0676c657e181578765cd7f6
        tracker_head: 975471e76eb89b8e8347dd0cba66b2566c62e1f0
        branch_point: 1e93c3d9e0309d35f9993e3b52f5339480074081
        since: "2026-10-15T19:45:21-07:00"
        until: "2026-10-16T03:50:47-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T19:45:21-07:00"
              end: "2026-10-15T20:45:21-07:00"
              began: "2026-10-15T19:45:21-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-16T02:59:59-07:00"
              end: "2026-10-16T03:50:47-07:00"
              began: "2026-10-16T02:59:59-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4d9aaac1c2280cafce056e7f08e02684a5297091192d5e52ff75916caf8d0efe
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 7128928
              windows:
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5737d584cf35265469b124fac2e9b0f83d6e07ffce5e088e4065e64d2b12668b
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 100097492
              windows:
                - start: "2026-10-15T19:45:21-07:00"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a1429e-ebc8-7610-b5df-8a4d6e94c9a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 303856442c5abae077043b1dddf05234698a6272e89f17d4add55b736976a625
              started: "2026-10-16T02:51:13.807Z"
              spawned: true
              bytes: 637944
              windows:
                - start: "2026-10-16T02:51:13.807Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ab-0b8a-7e51-8b96-2bd081084538
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 3d4af4fd9200a90586f21a8d46f3f4562e2bf3d84e7809767ce0b9749ac9e07e
              started: "2026-10-16T03:04:28.545Z"
              spawned: false
              bytes: 605962
              windows:
                - start: "2026-10-16T03:04:28.545Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142ab-8f17-75d0-9d16-9f5cb036d5e3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 38d33ef511e7fc987d122ca62ae9610f13cdfb430ac9ffa7bab67f9d8de0d811
              started: "2026-10-16T03:05:02.084Z"
              spawned: true
              bytes: 850604
              windows:
                - start: "2026-10-16T03:05:02.084Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142ad-01a9-7c12-af5c-200906e8cdca
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ad55ba4d74274be3fa6970cbb4955997b27eb5f808a7484d3ad8d07d5c434dcb
              started: "2026-10-16T03:06:37.159Z"
              spawned: false
              bytes: 456151
              windows:
                - start: "2026-10-16T03:06:37.159Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142ae-6330-7783-9f09-31d7ff6d2a7a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b71b546d1e6d10cbbda97b487230c878ffb28afe56dfcb9486d550d6615af80d
              started: "2026-10-16T03:08:07.966Z"
              spawned: false
              bytes: 446084
              windows:
                - start: "2026-10-16T03:08:07.966Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142ae-d92a-7ee2-824b-b23fbbac5e37
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8497a635a85846893095cc0bb14dec6d92791e01a5b3b66f0333f3e18adc7b34
              started: "2026-10-16T03:08:37.905Z"
              spawned: false
              bytes: 397998
              windows:
                - start: "2026-10-16T03:08:37.905Z"
                  end: "2026-10-15T20:45:21-07:00"
                - start: "2026-10-16T02:59:59-07:00"
                  end: "2026-10-16T03:50:47-07:00"
            - harness: codex
              id: 01a142af-ad08-7e40-80ec-d227c3009b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 715da457ab79624da6910aa60f79e0029d3a7696674e47abf1a4f7de1c5c6c2d
              started: "2026-10-16T03:09:31.922Z"
              spawned: true
              bytes: 642617
              windows:
                - start: "2026-10-16T03:09:31.922Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142af-eee0-7690-a5d6-7665399ff88b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: c5861c3be06e7a31b005f2628f7729989e532d2bf5bbbcd1032cfaa094eb33aa
              started: "2026-10-16T03:09:48.791Z"
              spawned: true
              bytes: 704235
              windows:
                - start: "2026-10-16T03:09:48.791Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b3-1520-7c92-a180-31e3993bd485
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b94fc705dc5704b435b1465cd05b14ef71067d82623f540f15bb6c5f3b63d24c
              started: "2026-10-16T03:13:15.222Z"
              spawned: true
              bytes: 1266036
              windows:
                - start: "2026-10-16T03:13:15.222Z"
                  end: "2026-10-15T20:45:21-07:00"
            - harness: codex
              id: 01a142b8-e0ea-7662-a597-05926edba15b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: f007c9977d91413c421707c79fc5e86901c9e9c9b9109091e54c3174964de81c
              started: "2026-10-16T03:19:35.005Z"
              spawned: true
              bytes: 623158
              windows:
                - start: "2026-10-16T03:19:35.005Z"
                  end: "2026-10-15T20:45:21-07:00"
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
                  end: "2026-10-15T20:45:21-07:00"
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
                  end: "2026-10-15T20:45:21-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.0299
      "n": 1
      timestamp: "2026-10-16T04:07:39-07:00"
      dispatched: "2026-10-16T03:51:41-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Spec, Done-when and plan Core concepts still require the dropped capability cutover gate
          detail: Plan rows name launcher/native_lock_capability.go and readiness/record.go (absent/unchanged); issue Spec line 186 and Done-when bullet 4 still promise "relaunch older slots first" refusal and legacy-capability tests; the re-plan has no plan Revisions entry and the issue Log says "awaiting OK" while atlas cites TL e93b109d. Update plan/issue to the approved re-plan.
          family: plan-claims-undelivered-surface
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Late binding notice after a restart has started (SIGUSR2) turns the agent restart into a wrapper failure
          detail: masterPump calls Discover after locks.Stop(); Advance rejects "discovery after stop", the error is stored in nativeLockErr and returned before the restartFresh check, so the wrapper exits 1 instead of re-execing. An ID valid for validBindingNotice but rejected by NewKey likewise kills the agent group. Ignore/log rejected notices while stopping and add a wrapper-level test.
          family: stopping-rejection-treated-as-lock-failure
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: ownedGroup always waits the full 1s grace, adding a second to every wrapper exit and restart
          detail: StartStop only skips the timer on ESRCH; with only the unreaped leader left, Darwin returns EPERM (measured on this host) and Linux succeeds against zombies, so group.Wait always blocks 1s before SIGKILL and reap. The test uses a 1ms grace and cannot see it. Finish early once the group is empty.
          family: fixed-grace-instead-of-bound
          round: 1
        - kind: defect
          id: BR-4
          severity: Important
          title: nativeLockHolder synchronously execs sdlc workspace --json (up to 2s) before every agent spawn
          detail: It runs unconditionally, even for non-native wraps and unknown-ID Fresh launches, only to produce a diagnostic label, with no seam or fake (ARCH-CONSTRAINTS/ARCH-MOCK). Resolve lazily when a key is first acquired, from launch environment, behind an injectable seam.
          family: external-probe-on-startup-path
          round: 1
        - kind: defect
          id: BR-5
          severity: Important
          title: No wrapper-level tests for late collision, notice-channel handling or controlled signal/restart exits
          detail: Done-when bullets 2-3 are covered only by the primitives; nothing drives masterPump notice to Discover to StartStop to nativeLockErr, or the SIGTERM/HUP/SIGUSR2 paths in run (ARCH-ORDER), which is how the restart bug above shipped.
          family: wrapper-ordering-untested
          round: 1
        - kind: defect
          id: BR-6
          severity: Minor
          title: atlas/couch.md Owners list still names the deleted launcher/native_lock_capability.go
          family: plan-claims-undelivered-surface
          round: 1
        - kind: defect
          id: BR-7
          severity: Minor
          title: Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
          family: truthful-lock-status-notice
          round: 1
        - kind: defect
          id: BR-8
          severity: Minor
          title: Acquire fails outright on an invalid holder label instead of falling back to the generic label
          family: diagnostic-label-blocks-lock
          round: 1
        - kind: defect
          id: BR-9
          severity: Minor
          title: Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule
          family: shared-identifier-validation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#595 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T04:07:39-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `plan-claims-undelivered-surface` (kind: defect) Spec, Done-when and plan Core concepts still require the dropped capability cutover gate
  Plan rows name launcher/native_lock_capability.go and readiness/record.go (absent/unchanged); issue Spec line 186 and Done-when bullet 4 still promise "relaunch older slots first" refusal and legacy-capability tests; the re-plan has no plan Revisions entry and the issue Log says "awaiting OK" while atlas cites TL e93b109d. Update plan/issue to the approved re-plan.
- **BR-2** [Important] `stopping-rejection-treated-as-lock-failure` (kind: defect) Late binding notice after a restart has started (SIGUSR2) turns the agent restart into a wrapper failure
  masterPump calls Discover after locks.Stop(); Advance rejects "discovery after stop", the error is stored in nativeLockErr and returned before the restartFresh check, so the wrapper exits 1 instead of re-execing. An ID valid for validBindingNotice but rejected by NewKey likewise kills the agent group. Ignore/log rejected notices while stopping and add a wrapper-level test.
- **BR-3** [Important] `fixed-grace-instead-of-bound` (kind: defect) ownedGroup always waits the full 1s grace, adding a second to every wrapper exit and restart
  StartStop only skips the timer on ESRCH; with only the unreaped leader left, Darwin returns EPERM (measured on this host) and Linux succeeds against zombies, so group.Wait always blocks 1s before SIGKILL and reap. The test uses a 1ms grace and cannot see it. Finish early once the group is empty.
- **BR-4** [Important] `external-probe-on-startup-path` (kind: defect) nativeLockHolder synchronously execs sdlc workspace --json (up to 2s) before every agent spawn
  It runs unconditionally, even for non-native wraps and unknown-ID Fresh launches, only to produce a diagnostic label, with no seam or fake (ARCH-CONSTRAINTS/ARCH-MOCK). Resolve lazily when a key is first acquired, from launch environment, behind an injectable seam.
- **BR-5** [Important] `wrapper-ordering-untested` (kind: defect) No wrapper-level tests for late collision, notice-channel handling or controlled signal/restart exits
  Done-when bullets 2-3 are covered only by the primitives; nothing drives masterPump notice to Discover to StartStop to nativeLockErr, or the SIGTERM/HUP/SIGUSR2 paths in run (ARCH-ORDER), which is how the restart bug above shipped.
- **BR-6** [Minor] `plan-claims-undelivered-surface` (kind: defect) atlas/couch.md Owners list still names the deleted launcher/native_lock_capability.go
- **BR-7** [Minor] `truthful-lock-status-notice` (kind: defect) Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
- **BR-8** [Minor] `diagnostic-label-blocks-lock` (kind: defect) Acquire fails outright on an invalid holder label instead of falling back to the generic label
- **BR-9** [Minor] `shared-identifier-validation` (kind: defect) Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule

## Open findings

- **BR-1** [Critical] `plan-claims-undelivered-surface` (kind: defect) Spec, Done-when and plan Core concepts still require the dropped capability cutover gate
- **BR-2** [Important] `stopping-rejection-treated-as-lock-failure` (kind: defect) Late binding notice after a restart has started (SIGUSR2) turns the agent restart into a wrapper failure
- **BR-3** [Important] `fixed-grace-instead-of-bound` (kind: defect) ownedGroup always waits the full 1s grace, adding a second to every wrapper exit and restart
- **BR-4** [Important] `external-probe-on-startup-path` (kind: defect) nativeLockHolder synchronously execs sdlc workspace --json (up to 2s) before every agent spawn
- **BR-5** [Important] `wrapper-ordering-untested` (kind: defect) No wrapper-level tests for late collision, notice-channel handling or controlled signal/restart exits
- **BR-6** [Minor] `plan-claims-undelivered-surface` (kind: defect) atlas/couch.md Owners list still names the deleted launcher/native_lock_capability.go
- **BR-7** [Minor] `truthful-lock-status-notice` (kind: defect) Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
- **BR-8** [Minor] `diagnostic-label-blocks-lock` (kind: defect) Acquire fails outright on an invalid holder label instead of falling back to the generic label
- **BR-9** [Minor] `shared-identifier-validation` (kind: defect) Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule
