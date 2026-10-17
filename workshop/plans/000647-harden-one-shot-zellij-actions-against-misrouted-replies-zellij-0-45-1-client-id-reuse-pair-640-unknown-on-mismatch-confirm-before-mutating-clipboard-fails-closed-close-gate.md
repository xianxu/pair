---
gate: boundary-review
issue: 647
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1812c841-1c00-42c8-9326-8058ce3f0839
            repository: github.com/xianxu/pair
            issue_id: "000647"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/dc3c9f8c8c58455f
            reviewed_head: b4236f4ef378d88e5bee66906b828d0f28f298ce
            slot: pair:1
            requested_at: "2026-10-17T19:20:50.357935Z"
          phase: finished
          pid: 42448
          started_at: "2026-10-17T19:20:50.468522Z"
          finished_at: "2026-10-17T19:23:49.481621Z"
          session_id: 01a14b4f-4d0d-7c71-bca5-f3b997f089c3
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: b4236f4ef378d88e5bee66906b828d0f28f298ce
        tracker_head: 0850755fa273c21c71f30503b4dafdb0ff095118
        branch_point: c0841e1082d39731781d8a2972eeeaed41d07ade
        since: "2026-10-17T11:09:41-07:00"
        until: "2026-10-17T12:20:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T11:09:41-07:00"
              end: "2026-10-17T12:20:24-07:00"
              began: "2026-10-17T11:09:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a14367-3aab-7a22-b89b-0908f54a9846
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: d44f963acffe001398c2d0803292672a3c0d005ded5c90226e6d9cac8b2d4d00
              started: "2026-10-16T06:30:01.402Z"
              spawned: false
              bytes: 50968369
              windows:
                - start: "2026-10-17T11:09:41-07:00"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b16-22e0-7d71-b7cf-dd6d298acc07
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: a10d3fdd9f3f8f52d18b6669bfd73202ec2ac9383eeff7e9e64e790663687dda
              started: "2026-10-17T18:18:24.815Z"
              spawned: false
              bytes: 669765
              windows:
                - start: "2026-10-17T18:18:24.815Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b1a-ca03-7672-8af5-23bfb7b3ae31
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 7d9baec896379ebacfcf35681f375babd28cc59822a61d4269e9d55dc51dd69c
              started: "2026-10-17T18:23:29.721Z"
              spawned: false
              bytes: 495668
              windows:
                - start: "2026-10-17T18:23:29.721Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b1c-04fb-71c0-a664-55b826acf5ad
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: db426a236f0a35a1baeadcdd881e4c6f34a7f26857c7eb5c79eeea5c2e0858d2
              started: "2026-10-17T18:24:50.437Z"
              spawned: false
              bytes: 447881
              windows:
                - start: "2026-10-17T18:24:50.437Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b1c-61b8-78a0-a243-f2e794ef47b3
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 8fab00af97f195cbf2312c17dc0d28e36f9d9a3bdc8bdb15494d01af4969d9a4
              started: "2026-10-17T18:25:14.067Z"
              spawned: false
              bytes: 452686
              windows:
                - start: "2026-10-17T18:25:14.067Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b1d-52ba-7730-abf6-0067df228cad
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: 1ce6d9cedfed3c96f161e982e9454cade37cd2096cc31effeb858bef732bcf40
              started: "2026-10-17T18:26:15.671Z"
              spawned: true
              bytes: 501418
              windows:
                - start: "2026-10-17T18:26:15.671Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b1d-781d-7fa2-b666-bfeba707f5cb
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: 2cc7bd3852e982668cab809ff9a009457b3ca25f05a3d740164a5bdf1f08121b
              started: "2026-10-17T18:26:25.034Z"
              spawned: true
              bytes: 508430
              windows:
                - start: "2026-10-17T18:26:25.034Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b21-5374-7d40-b911-a7c8855612df
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: 3c3d8c983eb723a1c8c176ce02af9397ad3fe1a5bccc4e388f260a7e73f950c9
              started: "2026-10-17T18:30:37.799Z"
              spawned: true
              bytes: 491935
              windows:
                - start: "2026-10-17T18:30:37.799Z"
                  end: "2026-10-17T12:20:24-07:00"
            - harness: codex
              id: 01a14b28-ecf4-7ab3-a74a-48d1aa733056
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: d8d38d9b7f47d01222b902241d72ab8e2f051e7e055a5a3fd0eef98d9dc3fca9
              started: "2026-10-17T18:38:55.891Z"
              spawned: true
              bytes: 825834
              windows:
                - start: "2026-10-17T18:38:55.891Z"
                  end: "2026-10-17T12:20:24-07:00"
        queue_waits:
            - attempt_id: 3c2d468ef5dbd6af994b5071e7c1e3ec
              slot: pair:1
              record_digest: f999cfb089ac0b1c1a7c8b11f0e79562319f7546a721da587e8d02a2f1cced74
              start: "2026-10-17T18:11:36.368971Z"
              end: "2026-10-17T18:16:18.343583Z"
              queue_seconds: 281.974612
              raw_seconds: 281.974612
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.13475
      "n": 1
      timestamp: "2026-10-17T12:23:54-07:00"
      dispatched: "2026-10-17T12:20:48-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Launcher Couch attachment still retries unknown observations into authority
          detail: cmd/internal/launcher/couch_session.go:149 retries every SessionLiveness error. A malformed or wrong-kind first reply now rejected by the strict parser can therefore be replaced by a valid second reply, allowing verifyCouchSessionAttach to succeed and attachment to continue. This violates the approved no-retry contract and ARCH-PURPOSE/ARCH-ORDER. Remove this retry and add a production-path regression asserting that unknown first evidence prevents attachment even when the next reply would be valid.
          family: unknown-observation-is-terminal
          round: 1
        - id: BR-2
          severity: Critical
          title: Individually valid but contradictory session listings still grant live authority
          detail: cmd/internal/launcher/zellij.go:116-142 validates short and full listings independently without reconciling their session identities. For example, short output containing pair-a and full output containing only pair-b both pass; snapshot then reports pair-a as SessionLive at line 98 because omission from the full listing is treated as absence of an EXITED flag. ListedPairSessionContext consumes this projection as positive attachment evidence. Under ARCH-SECURE/ARCH-ORDER, conflicting observations must remain unknown. Reconcile the decision-relevant listing facts before one-shot admission and cover contradictory listings with zero-effect regressions.
          family: contradictory-evidence-is-unknown
          round: 1
        - id: BR-3
          severity: Important
          title: Existing terminal lifecycle fixture fails the new listing contract
          detail: cmd/internal/couchcore/terminal_registration_fixture_test.go:147 emits only a session name for both list-sessions forms. Consequently, go test ./cmd/internal/couchcore -run '^TestListedDetachedTerminalAttachesWithoutNewServer$' -count=1 fails deterministically at terminal_warm_fixture_test.go:171 with “session listing row 1 lacks creation metadata”. Update the shared fixture to emit native-valid full listings and owner-pane fields, then rerun its lifecycle consumers without weakening production validation.
          family: strict-contract-fixture-conformance
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#647 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T12:23:54-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `unknown-observation-is-terminal` Launcher Couch attachment still retries unknown observations into authority
  cmd/internal/launcher/couch_session.go:149 retries every SessionLiveness error. A malformed or wrong-kind first reply now rejected by the strict parser can therefore be replaced by a valid second reply, allowing verifyCouchSessionAttach to succeed and attachment to continue. This violates the approved no-retry contract and ARCH-PURPOSE/ARCH-ORDER. Remove this retry and add a production-path regression asserting that unknown first evidence prevents attachment even when the next reply would be valid.
- **BR-2** [Critical] `contradictory-evidence-is-unknown` Individually valid but contradictory session listings still grant live authority
  cmd/internal/launcher/zellij.go:116-142 validates short and full listings independently without reconciling their session identities. For example, short output containing pair-a and full output containing only pair-b both pass; snapshot then reports pair-a as SessionLive at line 98 because omission from the full listing is treated as absence of an EXITED flag. ListedPairSessionContext consumes this projection as positive attachment evidence. Under ARCH-SECURE/ARCH-ORDER, conflicting observations must remain unknown. Reconcile the decision-relevant listing facts before one-shot admission and cover contradictory listings with zero-effect regressions.
- **BR-3** [Important] `strict-contract-fixture-conformance` Existing terminal lifecycle fixture fails the new listing contract
  cmd/internal/couchcore/terminal_registration_fixture_test.go:147 emits only a session name for both list-sessions forms. Consequently, go test ./cmd/internal/couchcore -run '^TestListedDetachedTerminalAttachesWithoutNewServer$' -count=1 fails deterministically at terminal_warm_fixture_test.go:171 with “session listing row 1 lacks creation metadata”. Update the shared fixture to emit native-valid full listings and owner-pane fields, then rerun its lifecycle consumers without weakening production validation.

## Open findings

- **BR-1** [Critical] `unknown-observation-is-terminal` Launcher Couch attachment still retries unknown observations into authority
- **BR-2** [Critical] `contradictory-evidence-is-unknown` Individually valid but contradictory session listings still grant live authority
- **BR-3** [Important] `strict-contract-fixture-conformance` Existing terminal lifecycle fixture fails the new listing contract
