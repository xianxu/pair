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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ff41bbe4-d232-453e-ace3-310896fa8f77
            repository: github.com/xianxu/pair
            issue_id: "000647"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9c75f4d5e27995bc
            reviewed_head: daac3160b931f3344bae2adc31afd21c8ec07153
            slot: pair:1
            requested_at: "2026-10-17T20:01:49.243621Z"
          phase: finished
          pid: 90232
          started_at: "2026-10-17T20:01:49.461273Z"
          finished_at: "2026-10-17T20:05:39.167824Z"
          session_id: 01a14b74-d389-7ed3-b767-c5c1be0263a0
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: daac3160b931f3344bae2adc31afd21c8ec07153
        tracker_head: 3b8fd7963879ee79e8d425016b9044ee3f120f2f
        branch_point: c0841e1082d39731781d8a2972eeeaed41d07ade
        since: "2026-10-17T11:09:41-07:00"
        until: "2026-10-17T13:01:13-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T11:09:41-07:00"
              end: "2026-10-17T13:01:13-07:00"
              began: "2026-10-17T11:09:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a14367-3aab-7a22-b89b-0908f54a9846
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 592e38463932b527d87d8b079ef5b0820776937787fb1934ded2aee7fdc68ffc
              started: "2026-10-16T06:30:01.402Z"
              spawned: false
              bytes: 54031550
              windows:
                - start: "2026-10-17T11:09:41-07:00"
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
                  end: "2026-10-17T13:01:13-07:00"
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
        excluded_brain_minutes: 21.398933333333332
        cuts:
            - boundary: M1
              dispatched: "2026-10-17T12:20:48-07:00"
              finalized: "2026-10-17T12:23:54-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-17T13:05:40-07:00"
      dispatched: "2026-10-17T13:01:46-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The retry is removed. TestCouchSessionAttachStopsAtFirstUnknownListing passes at HEAD and fails at all five production/exported attach fences when the retry is restored in a disposable copy.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Native and cached short/full listings now require exact identity-set agreement before projection or client queries. Removing reconciliation makes the contradictory-listing and production RunLaunch regressions fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The shared fixture now emits native-valid full listings and is_plugin fields. Listed-detached and composed-launch consumers pass with isolated Git discovery; reverting the fixture makes TestRegistrationTerminalFixtureDistinguishesListingForms fail.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Critical
          title: Absence diagnostics erase query-process cleanup failures
          detail: 'cmd/internal/launcher/zellij.go:227-230 and cmd/internal/launcher/osruntime.go:175-178 use errors.As to recognize an exit-1 diagnostic, then return success even when zellijexec.Run joined that ExitError with a cleanup failure at cmd/internal/zellijexec/run.go:45-53. Injecting EIO at the collector boundary reproduced nil errors from both production consumers. This violates ARCH-ORDER: a failed query boundary becomes successful absence or name-acceptance evidence. This is the 2nd finding in family unknown-observation-is-terminal. Do NOT fix only one instance: state and enforce the rule that diagnostic normalization may consume only the subprocess exit failure, never cancellation or collection failures. The sweep found this pattern in both of the runner''s two production consumers. Preserve cleanup status separately and add regressions for both consumers.'
          family: unknown-observation-is-terminal
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 25b927cd-c808-4b0c-8daa-b7932a475749
            repository: github.com/xianxu/pair
            issue_id: "000647"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ebe4614c77573ff9
            reviewed_head: dad70e903c4abaa882f4d38b3fa84680fa8c7bba
            slot: pair:1
            requested_at: "2026-10-17T20:17:31.970583Z"
          phase: finished
          pid: 16919
          started_at: "2026-10-17T20:17:32.108502Z"
          finished_at: "2026-10-17T20:19:46.599697Z"
          session_id: 01a14b83-3509-7ce1-b876-4e83c0953d92
          outcome: returned
      "n": 3
      timestamp: "2026-10-17T13:19:48-07:00"
      dispatched: "2026-10-17T13:17:29-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Terminal unknown remains final across launcher and Couch attach fences; production-path and controlled helper-exit regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Native and cached short/full listings must agree on exact identities before granting authority; contradictory and reordered controls pass.
          round: 3
        - id: BR-3
          disposition: addressed
          note: The registration fixture distinguishes short/full listing forms and supplies typed pane evidence; selected registration controls pass.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Both consumers normalize only direct ExitError values and preserve collector plus cancellation causes. TestQueryCollectionFailuresCannotBecomeDiagnosticEvidence passes at head; restoring pre-fix handling in a disposable copy produces six failing subtests. The copy was removed.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: dad70e903c4abaa882f4d38b3fa84680fa8c7bba
      blocked: false
---

# Gate ledger — pair647-query-fix#647 (boundary-review)

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

## Round 2 — 2026-10-17T13:05:40-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The retry is removed. TestCouchSessionAttachStopsAtFirstUnknownListing passes at HEAD and fails at all five production/exported attach fences when the retry is restored in a disposable copy.
- BR-2 — addressed — Native and cached short/full listings now require exact identity-set agreement before projection or client queries. Removing reconciliation makes the contradictory-listing and production RunLaunch regressions fail.
- BR-3 — addressed — The shared fixture now emits native-valid full listings and is_plugin fields. Listed-detached and composed-launch consumers pass with isolated Git discovery; reverting the fixture makes TestRegistrationTerminalFixtureDistinguishesListingForms fail.

### Raised

- **BR-4** [Critical] `unknown-observation-is-terminal` (kind: defect) Absence diagnostics erase query-process cleanup failures
  cmd/internal/launcher/zellij.go:227-230 and cmd/internal/launcher/osruntime.go:175-178 use errors.As to recognize an exit-1 diagnostic, then return success even when zellijexec.Run joined that ExitError with a cleanup failure at cmd/internal/zellijexec/run.go:45-53. Injecting EIO at the collector boundary reproduced nil errors from both production consumers. This violates ARCH-ORDER: a failed query boundary becomes successful absence or name-acceptance evidence. This is the 2nd finding in family unknown-observation-is-terminal. Do NOT fix only one instance: state and enforce the rule that diagnostic normalization may consume only the subprocess exit failure, never cancellation or collection failures. The sweep found this pattern in both of the runner's two production consumers. Preserve cleanup status separately and add regressions for both consumers.

## Round 3 — 2026-10-17T13:19:48-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Terminal unknown remains final across launcher and Couch attach fences; production-path and controlled helper-exit regressions pass.
- BR-2 — addressed — Native and cached short/full listings must agree on exact identities before granting authority; contradictory and reordered controls pass.
- BR-3 — addressed — The registration fixture distinguishes short/full listing forms and supplies typed pane evidence; selected registration controls pass.
- BR-4 — addressed — Both consumers normalize only direct ExitError values and preserve collector plus cancellation causes. TestQueryCollectionFailuresCannotBecomeDiagnosticEvidence passes at head; restoring pre-fix handling in a disposable copy produces six failing subtests. The copy was removed.

## Open findings

(none — every finding has been disposed)
