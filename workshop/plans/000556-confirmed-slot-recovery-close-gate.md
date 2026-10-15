---
gate: boundary-review
issue: 556
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2d69997a-f595-4ed7-a0d9-c81cd05e1271
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0d4255c869952fe8
            reviewed_head: a2b67ce4127cd90c06961204a4feed4c213dd1b2
            slot: pair:2
            requested_at: "2026-10-15T14:25:55.8153Z"
          phase: finished
          pid: 14804
          started_at: "2026-10-15T14:25:55.907008Z"
          finished_at: "2026-10-15T14:31:13.677955Z"
          session_id: 01a13ff4-9593-7ac2-b900-fea83c073860
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: a2b67ce4127cd90c06961204a4feed4c213dd1b2
        tracker_head: 0419f748d42598ca675ca1dbab87518bcccaaf96
        branch_point: 28d9e6816fd5139ad895ea22d585d7cd6aac9ed9
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T07:25:31-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-14T15:10:56-07:00"
              end: "2026-10-14T15:14:20-07:00"
              began: "2026-10-14T15:10:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
            - start: "2026-10-15T05:25:38-07:00"
              end: "2026-10-15T07:25:31-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bd5c34f2bcc27d7655095237f35da1c2f533ac07d3a504cae1e3effbef0d0e8c
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 66321192
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f89-c505-75d1-8c3f-68cad2c5cb47
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: bbeeacd1afcdaa1b33d42ed3d8c7410752357d17cf162f91cf77df320c971cd7
              started: "2026-10-15T12:29:16.072Z"
              spawned: true
              bytes: 1014136
              windows:
                - start: "2026-10-15T12:29:16.072Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f89-dd68-7b23-a41e-cab0cafa6d0a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b4e8e1faa6bea920f7a1a1865d173c6b275d09debbbf7ae03d95539abd6df5f6
              started: "2026-10-15T12:29:22.379Z"
              spawned: true
              bytes: 718460
              windows:
                - start: "2026-10-15T12:29:22.379Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: fa55a48129568c719233a3ed585e963ef91535b8089c86864b6ad5475371ec41
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 5383307
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 2a740f2b9a54f885799ddcb975c34d7e6c50279c3d488bc31c5dd9d58828c0a2
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 4441280
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T07:25:31-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: a525f61cccbed3c97f5acc2109d28ebd11964aecde4b7b0a940bd2c7820b3cbf
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 5647457
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T07:25:31-07:00"
      "n": 1
      timestamp: "2026-10-15T07:31:14-07:00"
      dispatched: "2026-10-15T07:25:54-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Retained-record Fresh still reconciles beneath foreign owners
          detail: cmd/internal/couchcore/recovery_prepared.go:330 unconditionally calls prepareTrackedWorkspace, which invokes Workspaces.Ensure for numbered slots. launchRecoveryFreshAddress reaches this call even when confirmRecoveryAbsent failed and the old owners were deliberately left running (lines 414–465). Ensure invokes workspace reconciliation, so allocating a new tag does not isolate those owners from setup/repair effects. Use the read-only readiness adapter already implemented for recordless recovery whenever source absence is unproved. Extend TestPreparedRecoveryForeignFreshLeavesRemnantsAndUsesNewTag to reject any normal readiness call; its fixture currently increments prepared without asserting zero. ARCH-SECURE, ARCH-PURPOSE.
          family: recovery-preserves-surviving-workspaces
          round: 1
        - id: BR-2
          severity: Critical
          title: Confirmed retirement does not prove the advertised stop set stopped
          detail: cmd/internal/launcher/recovery_terminal.go:137–141 sends confirmed plans through server-only quiescence. The only signal seam is KillServer, while checkConfirmed returns snapshot.Servers (recovery_terminal_confirmed.go:291) even when accepted wrappers/native agents remain. The existing confirmed-retirement test succeeds although recoveryTerminalWorld.KillServer removes only Servers and leaves Processes intact. A surviving or reparented native owner can therefore strand Resume after the server was killed, and the promised settled-wrapper ParkExpected path is absent. Use normal park where available, implement bounded exact-owner retirement for damaged sources, and require independently observed absence of the named owners before declaring retirement complete. Add survivor/reparenting sequence assertions. ARCH-ORDER, ARCH-PURPOSE, ARCH-MOCK.
          family: retirement-requires-owner-absence
          round: 1
        - id: BR-3
          severity: Important
          title: README update appears missing for recovery confirmation and changed Fresh behavior
          detail: cmd/internal/couchcmd/cli.go:342 introduces standalone --recovery-confirmation TOKEN, and run.go:1386 changes reboot usage and confirmation semantics, but README.md is unchanged in the pinned range. README.md:452 still advertises reboot --force-unknown, and lines 495–499 and 1410–1412 retain the earlier restricted Fresh/Starting description. Update the CLI reference and recovery instructions to describe exact-stop confirmation, expiry, always-offered Fresh and bounded Starting. Atlas updates are present.
          family: user-facing-docs-follow-surface
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#556 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T07:31:14-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `recovery-preserves-surviving-workspaces` Retained-record Fresh still reconciles beneath foreign owners
  cmd/internal/couchcore/recovery_prepared.go:330 unconditionally calls prepareTrackedWorkspace, which invokes Workspaces.Ensure for numbered slots. launchRecoveryFreshAddress reaches this call even when confirmRecoveryAbsent failed and the old owners were deliberately left running (lines 414–465). Ensure invokes workspace reconciliation, so allocating a new tag does not isolate those owners from setup/repair effects. Use the read-only readiness adapter already implemented for recordless recovery whenever source absence is unproved. Extend TestPreparedRecoveryForeignFreshLeavesRemnantsAndUsesNewTag to reject any normal readiness call; its fixture currently increments prepared without asserting zero. ARCH-SECURE, ARCH-PURPOSE.
- **BR-2** [Critical] `retirement-requires-owner-absence` Confirmed retirement does not prove the advertised stop set stopped
  cmd/internal/launcher/recovery_terminal.go:137–141 sends confirmed plans through server-only quiescence. The only signal seam is KillServer, while checkConfirmed returns snapshot.Servers (recovery_terminal_confirmed.go:291) even when accepted wrappers/native agents remain. The existing confirmed-retirement test succeeds although recoveryTerminalWorld.KillServer removes only Servers and leaves Processes intact. A surviving or reparented native owner can therefore strand Resume after the server was killed, and the promised settled-wrapper ParkExpected path is absent. Use normal park where available, implement bounded exact-owner retirement for damaged sources, and require independently observed absence of the named owners before declaring retirement complete. Add survivor/reparenting sequence assertions. ARCH-ORDER, ARCH-PURPOSE, ARCH-MOCK.
- **BR-3** [Important] `user-facing-docs-follow-surface` README update appears missing for recovery confirmation and changed Fresh behavior
  cmd/internal/couchcmd/cli.go:342 introduces standalone --recovery-confirmation TOKEN, and run.go:1386 changes reboot usage and confirmation semantics, but README.md is unchanged in the pinned range. README.md:452 still advertises reboot --force-unknown, and lines 495–499 and 1410–1412 retain the earlier restricted Fresh/Starting description. Update the CLI reference and recovery instructions to describe exact-stop confirmation, expiry, always-offered Fresh and bounded Starting. Atlas updates are present.

## Open findings

- **BR-1** [Critical] `recovery-preserves-surviving-workspaces` Retained-record Fresh still reconciles beneath foreign owners
- **BR-2** [Critical] `retirement-requires-owner-absence` Confirmed retirement does not prove the advertised stop set stopped
- **BR-3** [Important] `user-facing-docs-follow-surface` README update appears missing for recovery confirmation and changed Fresh behavior
