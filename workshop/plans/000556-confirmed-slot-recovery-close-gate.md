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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ae868893-e886-4c83-a394-f1a854b708b0
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/48fa1712f7ee7662
            reviewed_head: a682dc8e4ccfa2ccb6717e9592e30bccc88db27f
            slot: pair:2
            requested_at: "2026-10-15T15:54:27.718592Z"
          phase: finished
          pid: 84093
          started_at: "2026-10-15T15:54:27.80741Z"
          finished_at: "2026-10-15T16:00:19.180974Z"
          session_id: 01a14045-a338-7a30-9746-d1e554a0e978
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: a682dc8e4ccfa2ccb6717e9592e30bccc88db27f
        tracker_head: 8e64700a93e0474ac9bc463d10a685ce5a69936d
        branch_point: d6a797451cc6ded68c890f10f294db28b94d1708
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T08:54:03-07:00"
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
              end: "2026-10-15T08:54:03-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4c32becc1c1a402f659d2681c914b5ff746fa5dff81689389672c7255a4f4dbd
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 70040653
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T08:54:03-07:00"
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
                  end: "2026-10-15T08:54:03-07:00"
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
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 48b645ecae3feb601d0797d83414e43e5834d1bd281ec5a798de2660000e34b0
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 6590774
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 25337c61cbe8c20b54699b46f799e5cc192ed8f72445487cf828c842a6c3f34f
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 5315331
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T08:54:03-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 12c78e0ea342e7db503687050515eb33b931970893d3bf1175376378aca61509
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 6436863
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T08:54:03-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T07:25:54-07:00"
              finalized: "2026-10-15T07:31:14-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T09:00:19-07:00"
      dispatched: "2026-10-15T08:54:25-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: recovery_prepared.go:477–484 selects read-only readiness when source absence is unproved. The strengthened foreign-survivor test asserts zero normal readiness calls and unchanged process/session identities; it passed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: retireConfirmed now signals accepted generations leaf-first and independently observes their absence. Passing regressions cover survivors, reparenting, replacement, false signal success and final-listing races; settled-owner tests exercise ParkExpected and refusal after park failure.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README.md:447–455 documents the standalone token command; lines 493–520 explain exact-stop acceptance, two-minute expiry, Fresh availability and bounded Starting. These passages agree with the CLI, confirmation owner and stale-start decision.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Important
          title: Avoid copying the live Couch for read-only profile resolution
          detail: cmd/internal/couchcore/recovery_preflight.go:82 copies the entire Couch, including recoveryMu, regMu and gateOnce; recovery_preflight_test.go:337 repeats the pattern. Copying an active synchronized owner violates its synchronization contract and can race concurrent operations. go vet ./cmd/internal/couchcore exits 1 with both copylocks diagnostics. Pass the read-only ThreadStore into profile-resolution helpers instead of cloning Couch, and construct the test fixture without copying its locks. ARCH-ORDER.
          family: synchronized-owners-must-not-be-copied
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1bb5cde0-e4f1-4dd4-bb7e-8e511095b0bd
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/99fba32e73b76d2a
            reviewed_head: dead1b665bddf4d053df5faba06b54a73420296a
            slot: pair:2
            requested_at: "2026-10-15T16:13:15.841739Z"
          phase: finished
          pid: 66084
          started_at: "2026-10-15T16:13:15.934778Z"
          finished_at: "2026-10-15T16:18:43.595767Z"
          session_id: 01a14056-da01-7461-94a5-7592a3cda1c7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: dead1b665bddf4d053df5faba06b54a73420296a
        tracker_head: 8e64700a93e0474ac9bc463d10a685ce5a69936d
        branch_point: d6a797451cc6ded68c890f10f294db28b94d1708
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T09:12:57-07:00"
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
              end: "2026-10-15T09:12:57-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 1355590379cfb8398575c9add6762805eaa4b926e059f01dc17f6c9f5793a627
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 71105997
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T09:12:57-07:00"
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
                  end: "2026-10-15T09:12:57-07:00"
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
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 5ef7bc6caca1bdd40e551380ea277f2645679b045366143be89130989f41dcd3
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 6887723
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: d19521de83a6b374d343c9139d8288e2899b13e1fad6b810258ba5446c392980
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 5372550
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T09:12:57-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 836c253b7e7919ef7b500f449eb0ffc46d8daf567cd70d495234c41147c37243
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 6886986
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T09:12:57-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T07:25:54-07:00"
              finalized: "2026-10-15T07:31:14-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T08:54:25-07:00"
              finalized: "2026-10-15T09:00:19-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-15T09:18:44-07:00"
      dispatched: "2026-10-15T09:13:14-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Survivor-safe readiness remains implemented in recovery_prepared.go; retained-record and recordless Fresh regression tests pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Confirmed retirement independently proves accepted process absence; survivor, replacement, reparenting and final-listing regressions pass.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README.md documents standalone confirmation, expiry, Fresh availability and bounded Starting, consistent with the implemented CLI and recovery owners.
          round: 3
        - id: BR-4
          disposition: addressed
          note: recovery_preflight.go:82 copies only ThreadStore and passes it through the existing resolver; the fixture constructs Couch independently. Core vet passes; a temporary pre-fix overlay reproduces both copylocks diagnostics. Profile/preflight race tests pass.
          round: 3
      findings:
        - kind: defect
          id: BR-5
          severity: Critical
          title: Recognize keyed Fresh before applying native-resume admission guards
          detail: cmd/internal/couchcore/recovery_prepared.go:94–115 resolves the native profile and checks park, continuation and start claims before lines 125–130 recognize a Fresh candidate key. ResumeCandidate always enters as RecoveryInferred; menu_recovery.go:109–120 and the printed CLI command use that route for keyed Fresh. A temporary regression matrix reproduces failures for unfinished park, missing saved profile and young start claim; explicit RecoveryFresh succeeds in all three controls. This violates always-reachable Fresh (ARCH-PURPOSE). Normalize the validated choice before mode-specific admission, sweep every native-only guard, and add equivalence regressions through the advertised menu/CLI routes.
          family: fresh-selection-precedes-native-guards
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5cfe790c-12a5-4715-8bc0-a34bf76088e3
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/dc3b0e3198edcea8
            reviewed_head: 646c4a5bdb214ee4c755133049c3af386dc679f0
            slot: pair:2
            requested_at: "2026-10-15T16:33:46.692911Z"
          phase: finished
          pid: 51612
          started_at: "2026-10-15T16:33:46.781371Z"
          finished_at: "2026-10-15T16:36:05.093062Z"
          session_id: 01a14069-a1f6-7241-8712-481a63a68263
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 646c4a5bdb214ee4c755133049c3af386dc679f0
        tracker_head: 8e64700a93e0474ac9bc463d10a685ce5a69936d
        branch_point: d6a797451cc6ded68c890f10f294db28b94d1708
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T09:33:28-07:00"
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
              end: "2026-10-15T09:33:28-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 13ed5bf7be5ba1d85632f689f03bd7e5efeed578b37c28558bfbdfb95f3f3c95
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 72204416
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T09:33:28-07:00"
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
                  end: "2026-10-15T09:33:28-07:00"
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
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 0ff886d9d3dc1645a112a6103c6ef6d4d58675a44122451d8ef9a4217c5d6d45
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 7213711
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: d19521de83a6b374d343c9139d8288e2899b13e1fad6b810258ba5446c392980
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 5372550
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T09:33:28-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: a89b7023804dac0f251b8724f9096e6b7c112ba24b867a0adf56ba50368bc06e
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 7134411
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T09:33:28-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T07:25:54-07:00"
              finalized: "2026-10-15T07:31:14-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T08:54:25-07:00"
              finalized: "2026-10-15T09:00:19-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:13:14-07:00"
              finalized: "2026-10-15T09:18:44-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-15T09:36:05-07:00"
      dispatched: "2026-10-15T09:33:44-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Retained and recordless Fresh use read-only checkout readiness beside unproved owners. The retained-owner regression passed and asserts unchanged workspace bytes and no normal provisioning or signals.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Confirmed retirement independently proves every accepted process absent, with generation checks before signals and after final listing. Survivor, replacement, reparenting and delayed-absence regressions passed under race.
          round: 4
        - id: BR-3
          disposition: addressed
          note: The pinned README recovery and Reboot passages document standalone confirmation, expiry and survivor-safe Fresh; atlas/couch.md describes the corresponding prepared executor and retirement owners.
          round: 4
        - id: BR-4
          disposition: addressed
          note: Fresh profile resolution passes a read-only ThreadStore into existing resolvers without copying Couch. Profile regressions and go vet -copylocks passed.
          round: 4
        - id: BR-5
          disposition: addressed
          note: Validated keyed Fresh is normalized before profile and native-only guards. The four-scenario, three-route matrix passes on head; restoring the pre-fix executor in a disposable copy produces eight candidate/operation failures. Menu and printed-CLI transport regressions also pass.
          round: 4
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 646c4a5bdb214ee4c755133049c3af386dc679f0
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c262336d-f2ad-4642-99cd-a177748fab4a
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/25a87b5a1d4049fb
            reviewed_head: fe5275365b340df9a3f6d85b1b500625d30497fb
            slot: pair:2
            requested_at: "2026-10-15T17:46:59.430908Z"
          phase: finished
          pid: 82130
          started_at: "2026-10-15T17:46:59.761281Z"
          finished_at: "2026-10-15T17:51:12.801204Z"
          session_id: 01a140ac-ab14-73e3-8b9d-b70aa6eb0f2d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: fe5275365b340df9a3f6d85b1b500625d30497fb
        tracker_head: 8ea8f94c293645b55930dd8f27b4c20138349084
        branch_point: f3cd4848a3fc5924e81635da9ddd0f835a83ea72
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T10:46:16-07:00"
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
              end: "2026-10-15T10:46:16-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 1dbd4509cec935e5807858f459a36fb1b06de698ce70c44765121f437e15e3b5
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 77403521
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T10:46:16-07:00"
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
                  end: "2026-10-15T10:46:16-07:00"
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
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 76f1c789afc3df9a2b879784e16867d3b1964515c2c3d0a705feaf01238c8290
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 8478208
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 2e001aae715be8d2d064f146a86cc6c9ebc49d950f5e19602ddfdcd4557de56f
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 6769470
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T10:46:16-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 2706f6460aea090beec662d10c1aed7e15d4b26135c69f8f308119c661edc5d9
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 8797848
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T10:46:16-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.03448333333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T07:25:54-07:00"
              finalized: "2026-10-15T07:31:14-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T08:54:25-07:00"
              finalized: "2026-10-15T09:00:19-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:13:14-07:00"
              finalized: "2026-10-15T09:18:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:33:44-07:00"
              finalized: "2026-10-15T09:36:05-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-15T10:51:14-07:00"
      dispatched: "2026-10-15T10:46:56-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-6
          severity: Critical
          title: Classify every existing-session selector before permitting native spawn
          detail: cmd/internal/launcher/native_resume_selector.go:72 marks Claude --teleport as Remote without selecting a resume kind, and --from-pr is unclassified; both remain NativeResumeNone and reach start(cmd) without admission at cmd/internal/wrapcmd/native_admission.go:28–33. Installed claude --help confirms both resume existing sessions. Enumerate existing-session selectors using the existing Fresh authority, refuse unsupported mappings before spawn, and add regressions for both wrapper entry forms. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
          family: existing-session-selectors-require-admission
          round: 5
        - kind: defect
          id: BR-7
          severity: Important
          title: Provide the promised fenced recovery path for interrupted reserved grants
          detail: cmd/internal/launcher/native_admission_os.go:257 excludes reserved entries from collection, while native_admission.go:167–171 permits cancellation only by the original helper; no retirement integration supplies a fencing/removal transition. A helper dying after reservation permanently blocks that native ID even after its pending launch is fenced and its tree proved absent, contrary to the plan at lines 90 and 223–235, and repeated failures consume global capacity. Add exact-attempt fencing and reclamation under the store lock with a sequence test proving replacement admission succeeds and the old grant cannot execute. ARCH-ORDER, ARCH-FUNERAL.
          family: fenced-admissions-must-be-reclaimable
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8993afa2-605d-436e-ac2a-a58154e6ba97
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/cafcc7ec02f27c00
            reviewed_head: 451da7b2d3a7c062ead312ee8ef32d3a31e991e9
            slot: pair:2
            requested_at: "2026-10-15T18:33:24.122137Z"
          phase: finished
          pid: 75006
          started_at: "2026-10-15T18:33:24.377152Z"
          finished_at: "2026-10-15T18:36:38.189937Z"
          session_id: 01a140d7-285a-7170-b3d7-88696ec61a73
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 451da7b2d3a7c062ead312ee8ef32d3a31e991e9
        tracker_head: 50f8c8cb91f2f1246ceeaf541fea530c21e26165
        branch_point: f3cd4848a3fc5924e81635da9ddd0f835a83ea72
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T11:32:43-07:00"
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
              end: "2026-10-15T11:32:43-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a441b1d93cdf66c439c11108b4b6567290d6c00aefb8eef2657cc3a2fc4b154e
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 78549915
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T11:32:43-07:00"
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
                  end: "2026-10-15T11:32:43-07:00"
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
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: c9d06048b88f3e85afdc17de0f074e8440846b4dc8b71396b1bee5452c72505b
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 8986997
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 3cd8aa7267d66dbfb44ee3a9f6fd8d5d8e89e3ba648101eb1db90925c3f09cd3
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 7221777
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T11:32:43-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 2706f6460aea090beec662d10c1aed7e15d4b26135c69f8f308119c661edc5d9
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 8797848
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T11:32:43-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.03448333333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T07:25:54-07:00"
              finalized: "2026-10-15T07:31:14-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T08:54:25-07:00"
              finalized: "2026-10-15T09:00:19-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:13:14-07:00"
              finalized: "2026-10-15T09:18:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:33:44-07:00"
              finalized: "2026-10-15T09:36:05-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-15T10:46:56-07:00"
              finalized: "2026-10-15T10:51:14-07:00"
              finalizing: false
      "n": 6
      timestamp: "2026-10-15T11:36:39-07:00"
      dispatched: "2026-10-15T11:33:20-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: native_resume_selector.go:163 reuses Fresh's forbidden-selector authority. Removing this guard makes selector tests and real-wrapper tests for --teleport and --from-pr fail in both argv and encoded-command forms.
          round: 6
        - id: BR-7
          disposition: addressed
          note: native_admission_os.go:258 fences exact dead-helper reserved grants under the Claim lock. Removing reclamation makes replacement-admission and controlled lock-order tests fail; consumed spawning entries remain retained.
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 451da7b2d3a7c062ead312ee8ef32d3a31e991e9
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4baea88d-7fad-4c12-af47-4ab9d680d06b
            repository: github.com/xianxu/pair
            issue_id: "000556"
            recipe: milestone-review
            round: 7
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/78f82bba5b2e9485
            reviewed_head: 6bba5141ed4966662bd2d966188acb14bae0cf3f
            slot: pair:2
            requested_at: "2026-10-15T18:38:59.003709Z"
          phase: finished
          pid: 9743
          started_at: "2026-10-15T18:38:59.200567Z"
          finished_at: "2026-10-15T18:43:10.739264Z"
          session_id: 01a140dc-43d7-7461-ba55-4e00fdedd1e4
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 6bba5141ed4966662bd2d966188acb14bae0cf3f
        tracker_head: 50f8c8cb91f2f1246ceeaf541fea530c21e26165
        branch_point: f3cd4848a3fc5924e81635da9ddd0f835a83ea72
        since: "2026-10-14T15:10:56-07:00"
        until: "2026-10-15T11:38:26-07:00"
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
              end: "2026-10-15T11:38:26-07:00"
              began: "2026-10-15T05:25:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 55a5d9ed50386fa0e66cdbbf26b450239bcb7f8452af01e117ad6c6fa504f75a
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 78721581
              windows:
                - start: "2026-10-14T15:10:56-07:00"
                  end: "2026-10-14T15:14:20-07:00"
                - start: "2026-10-15T05:25:38-07:00"
                  end: "2026-10-15T11:38:26-07:00"
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
                  end: "2026-10-15T11:38:26-07:00"
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
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f93-ba32-71d2-8258-83e78285838e
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 468b366b1838d80815904d33e90c384f5dd2a8f3757e20e84324b8858acb021e
              started: "2026-10-15T12:40:11.384Z"
              spawned: false
              bytes: 874129
              windows:
                - start: "2026-10-15T12:40:11.384Z"
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f96-c4d5-7e33-bbb6-1db19a76ea4c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bc5280144f946b797cec25cf25c7432dfe38cf27b44ffe397f429ec15305acfe
              started: "2026-10-15T12:43:29.43Z"
              spawned: false
              bytes: 465487
              windows:
                - start: "2026-10-15T12:43:29.43Z"
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f98-b84b-7ae1-9c78-3b4a9be020a5
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bac5c8a02b3200b8163888f72b9147a79356e4f0baaafccc4496ea53020d1b8f
              started: "2026-10-15T12:45:36.668Z"
              spawned: false
              bytes: 470273
              windows:
                - start: "2026-10-15T12:45:36.668Z"
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f99-5581-7320-b0b0-500317f35f68
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2ac34b43830c16776c23e74d2b2a7083c63a85eccc3e1d3653c2e46f9b93d669
              started: "2026-10-15T12:46:17.557Z"
              spawned: false
              bytes: 471335
              windows:
                - start: "2026-10-15T12:46:17.557Z"
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f9a-21b2-7b82-b6ba-25e1f19f39fc
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: c9d06048b88f3e85afdc17de0f074e8440846b4dc8b71396b1bee5452c72505b
              started: "2026-10-15T12:47:08.358Z"
              spawned: true
              bytes: 8986997
              windows:
                - start: "2026-10-15T12:47:08.358Z"
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f9b-57c0-7e72-a872-467a305cda3b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 3cd8aa7267d66dbfb44ee3a9f6fd8d5d8e89e3ba648101eb1db90925c3f09cd3
              started: "2026-10-15T12:48:27.78Z"
              spawned: true
              bytes: 7221777
              windows:
                - start: "2026-10-15T12:48:27.78Z"
                  end: "2026-10-15T11:38:26-07:00"
            - harness: codex
              id: 01a13f9f-1efe-7480-b4d1-a97513916fd3
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 2706f6460aea090beec662d10c1aed7e15d4b26135c69f8f308119c661edc5d9
              started: "2026-10-15T12:52:35.36Z"
              spawned: true
              bytes: 8797848
              windows:
                - start: "2026-10-15T12:52:35.36Z"
                  end: "2026-10-15T11:38:26-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.03448333333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-15T07:25:54-07:00"
              finalized: "2026-10-15T07:31:14-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T08:54:25-07:00"
              finalized: "2026-10-15T09:00:19-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:13:14-07:00"
              finalized: "2026-10-15T09:18:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-15T09:33:44-07:00"
              finalized: "2026-10-15T09:36:05-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-15T10:46:56-07:00"
              finalized: "2026-10-15T10:51:14-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-15T11:33:20-07:00"
              finalized: "2026-10-15T11:36:39-07:00"
              finalizing: true
      "n": 7
      timestamp: "2026-10-15T11:43:13-07:00"
      dispatched: "2026-10-15T11:38:56-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Fresh selects read-only workspace readiness while owner absence is unproved; survivor and absent-source control tests pass.
          round: 7
        - id: BR-2
          disposition: addressed
          note: Confirmed retirement signals leaf-first and independently observes every accepted owner absent; survivor, replacement, false-success, and final-listing tests pass.
          round: 7
        - id: BR-3
          disposition: addressed
          note: README recovery passages document standalone confirmation, token lifetime, bounded Starting, and survivor-safe Fresh, matching the executor and confirmation owner; atlas changes cover the same flow.
          round: 7
        - id: BR-4
          disposition: addressed
          note: Profile resolution passes preview stores through the original Couch owner rather than copying Couch; profile tests and go vet pass.
          round: 7
        - id: BR-5
          disposition: addressed
          note: Keyed Fresh resolves before native-only profile and lifecycle guards; the explicit, candidate, and operation regression matrix passes.
          round: 7
        - id: BR-6
          disposition: not-addressed
          note: 'ARCH-PURPOSE and ARCH-SECURE: native_resume_selector.go:47 drops positional identity parsing after --. Codex resume -- <full UUID> becomes a picker and enters the unguarded compatibility exception. Temporary regressions fail for resume and exec resume; both production wrapper transports spawn synthetic children without admission. This is the 2nd finding in family existing-session-selectors-require-admission: enforce native positional grammar across the complete selector enumeration rather than patching only this spelling.'
          round: 7
        - id: BR-7
          disposition: addressed
          note: Reserve fences exact dead-helper reserved grants under the Claim lock; both lock-order tests, stale-grant rejection, and live/unknown/spawning retention controls pass.
          round: 7
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

## Round 2 — 2026-10-15T09:00:19-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — recovery_prepared.go:477–484 selects read-only readiness when source absence is unproved. The strengthened foreign-survivor test asserts zero normal readiness calls and unchanged process/session identities; it passed.
- BR-2 — addressed — retireConfirmed now signals accepted generations leaf-first and independently observes their absence. Passing regressions cover survivors, reparenting, replacement, false signal success and final-listing races; settled-owner tests exercise ParkExpected and refusal after park failure.
- BR-3 — addressed — README.md:447–455 documents the standalone token command; lines 493–520 explain exact-stop acceptance, two-minute expiry, Fresh availability and bounded Starting. These passages agree with the CLI, confirmation owner and stale-start decision.

### Raised

- **BR-4** [Important] `synchronized-owners-must-not-be-copied` (kind: defect) Avoid copying the live Couch for read-only profile resolution
  cmd/internal/couchcore/recovery_preflight.go:82 copies the entire Couch, including recoveryMu, regMu and gateOnce; recovery_preflight_test.go:337 repeats the pattern. Copying an active synchronized owner violates its synchronization contract and can race concurrent operations. go vet ./cmd/internal/couchcore exits 1 with both copylocks diagnostics. Pass the read-only ThreadStore into profile-resolution helpers instead of cloning Couch, and construct the test fixture without copying its locks. ARCH-ORDER.

## Round 3 — 2026-10-15T09:18:44-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Survivor-safe readiness remains implemented in recovery_prepared.go; retained-record and recordless Fresh regression tests pass.
- BR-2 — addressed — Confirmed retirement independently proves accepted process absence; survivor, replacement, reparenting and final-listing regressions pass.
- BR-3 — addressed — README.md documents standalone confirmation, expiry, Fresh availability and bounded Starting, consistent with the implemented CLI and recovery owners.
- BR-4 — addressed — recovery_preflight.go:82 copies only ThreadStore and passes it through the existing resolver; the fixture constructs Couch independently. Core vet passes; a temporary pre-fix overlay reproduces both copylocks diagnostics. Profile/preflight race tests pass.

### Raised

- **BR-5** [Critical] `fresh-selection-precedes-native-guards` (kind: defect) Recognize keyed Fresh before applying native-resume admission guards
  cmd/internal/couchcore/recovery_prepared.go:94–115 resolves the native profile and checks park, continuation and start claims before lines 125–130 recognize a Fresh candidate key. ResumeCandidate always enters as RecoveryInferred; menu_recovery.go:109–120 and the printed CLI command use that route for keyed Fresh. A temporary regression matrix reproduces failures for unfinished park, missing saved profile and young start claim; explicit RecoveryFresh succeeds in all three controls. This violates always-reachable Fresh (ARCH-PURPOSE). Normalize the validated choice before mode-specific admission, sweep every native-only guard, and add equivalence regressions through the advertised menu/CLI routes.

## Round 4 — 2026-10-15T09:36:05-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retained and recordless Fresh use read-only checkout readiness beside unproved owners. The retained-owner regression passed and asserts unchanged workspace bytes and no normal provisioning or signals.
- BR-2 — addressed — Confirmed retirement independently proves every accepted process absent, with generation checks before signals and after final listing. Survivor, replacement, reparenting and delayed-absence regressions passed under race.
- BR-3 — addressed — The pinned README recovery and Reboot passages document standalone confirmation, expiry and survivor-safe Fresh; atlas/couch.md describes the corresponding prepared executor and retirement owners.
- BR-4 — addressed — Fresh profile resolution passes a read-only ThreadStore into existing resolvers without copying Couch. Profile regressions and go vet -copylocks passed.
- BR-5 — addressed — Validated keyed Fresh is normalized before profile and native-only guards. The four-scenario, three-route matrix passes on head; restoring the pre-fix executor in a disposable copy produces eight candidate/operation failures. Menu and printed-CLI transport regressions also pass.

## Round 5 — 2026-10-15T10:51:14-07:00 (codex) — BLOCKED

### Raised

- **BR-6** [Critical] `existing-session-selectors-require-admission` (kind: defect) Classify every existing-session selector before permitting native spawn
  cmd/internal/launcher/native_resume_selector.go:72 marks Claude --teleport as Remote without selecting a resume kind, and --from-pr is unclassified; both remain NativeResumeNone and reach start(cmd) without admission at cmd/internal/wrapcmd/native_admission.go:28–33. Installed claude --help confirms both resume existing sessions. Enumerate existing-session selectors using the existing Fresh authority, refuse unsupported mappings before spawn, and add regressions for both wrapper entry forms. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
- **BR-7** [Important] `fenced-admissions-must-be-reclaimable` (kind: defect) Provide the promised fenced recovery path for interrupted reserved grants
  cmd/internal/launcher/native_admission_os.go:257 excludes reserved entries from collection, while native_admission.go:167–171 permits cancellation only by the original helper; no retirement integration supplies a fencing/removal transition. A helper dying after reservation permanently blocks that native ID even after its pending launch is fenced and its tree proved absent, contrary to the plan at lines 90 and 223–235, and repeated failures consume global capacity. Add exact-attempt fencing and reclamation under the store lock with a sequence test proving replacement admission succeeds and the old grant cannot execute. ARCH-ORDER, ARCH-FUNERAL.

## Round 6 — 2026-10-15T11:36:39-07:00 (codex) — passed

### Disposed

- BR-6 — addressed — native_resume_selector.go:163 reuses Fresh's forbidden-selector authority. Removing this guard makes selector tests and real-wrapper tests for --teleport and --from-pr fail in both argv and encoded-command forms.
- BR-7 — addressed — native_admission_os.go:258 fences exact dead-helper reserved grants under the Claim lock. Removing reclamation makes replacement-admission and controlled lock-order tests fail; consumed spawning entries remain retained.

## Round 7 — 2026-10-15T11:43:13-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Fresh selects read-only workspace readiness while owner absence is unproved; survivor and absent-source control tests pass.
- BR-2 — addressed — Confirmed retirement signals leaf-first and independently observes every accepted owner absent; survivor, replacement, false-success, and final-listing tests pass.
- BR-3 — addressed — README recovery passages document standalone confirmation, token lifetime, bounded Starting, and survivor-safe Fresh, matching the executor and confirmation owner; atlas changes cover the same flow.
- BR-4 — addressed — Profile resolution passes preview stores through the original Couch owner rather than copying Couch; profile tests and go vet pass.
- BR-5 — addressed — Keyed Fresh resolves before native-only profile and lifecycle guards; the explicit, candidate, and operation regression matrix passes.
- BR-6 — not-addressed — ARCH-PURPOSE and ARCH-SECURE: native_resume_selector.go:47 drops positional identity parsing after --. Codex resume -- <full UUID> becomes a picker and enters the unguarded compatibility exception. Temporary regressions fail for resume and exec resume; both production wrapper transports spawn synthetic children without admission. This is the 2nd finding in family existing-session-selectors-require-admission: enforce native positional grammar across the complete selector enumeration rather than patching only this spelling.
- BR-7 — addressed — Reserve fences exact dead-helper reserved grants under the Claim lock; both lock-order tests, stale-grant rejection, and live/unknown/spawning retention controls pass.

## Open findings

- **BR-6** [Critical] `existing-session-selectors-require-admission` (kind: defect) Classify every existing-session selector before permitting native spawn
