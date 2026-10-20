---
gate: boundary-review
issue: 717
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 46c1419d-cdb4-460f-b951-d47ecfbdd2db
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/78598fd55144cb8c
            reviewed_head: 55335ce84b2411e558a2785ef9616389f6fe1e22
            slot: pair:2
            requested_at: "2026-10-20T08:25:09.609158Z"
          phase: finished
          pid: 33921
          started_at: "2026-10-20T08:25:09.725163Z"
          finished_at: "2026-10-20T08:27:42.147863Z"
          session_id: 283e0eff-e05d-45ed-ac1d-65a7f24bb212
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 55335ce84b2411e558a2785ef9616389f6fe1e22
        tracker_head: 5f924897f1fe05c1ead6e44425e3908790321373
        branch_point: 0b93f91b541fdba4a7bb52196e5abd8d66bebd52
        since: "2026-10-20T01:12:01-07:00"
        until: "2026-10-20T01:24:35-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T01:12:01-07:00"
              end: "2026-10-20T01:24:35-07:00"
              began: "2026-10-20T01:12:01-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4e1ec141-8d89-49f0-8665-b8191ffad93e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fa2785a7cfaaf3f451dbadb5473fef5653798b54b363415f320397e4c0aac4cb
              started: "2026-10-20T08:20:14.274Z"
              spawned: false
              bytes: 393489
              windows:
                - start: "2026-10-20T08:20:14.274Z"
                  end: "2026-10-20T01:24:35-07:00"
            - harness: claude
              id: 5935e750-db21-494a-ae19-7f77ce8a39fc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 172c0822a9572d9882c4f17b1f7aed34d2a2955171b8a9eea479bddd0e2cf0b6
              started: "2026-10-20T08:18:54.104Z"
              spawned: false
              bytes: 527404
              windows:
                - start: "2026-10-20T08:18:54.104Z"
                  end: "2026-10-20T01:24:35-07:00"
            - harness: claude
              id: 7395260c-761a-4990-8b0d-47a6a2d6d972
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b423a4ba661e0a98bb43fa4a57fe05f23044f51057108640290a4210f780bfb2
              started: "2026-10-20T08:19:58.954Z"
              spawned: false
              bytes: 492806
              windows:
                - start: "2026-10-20T08:19:58.954Z"
                  end: "2026-10-20T01:24:35-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5f775e90386a8cdeebb2285ea9ab462465419798b586f008e4655a6831e16e02
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 52565154
              windows:
                - start: "2026-10-20T01:12:01-07:00"
                  end: "2026-10-20T01:24:35-07:00"
            - harness: claude
              id: ab6148dc-2ee1-41ab-85ce-8be4be326795
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 407bbf48e37a6af04415882970b10fe13a0f1108ce12604b89bec9e002317247
              started: "2026-10-20T08:15:40.587Z"
              spawned: false
              bytes: 555721
              windows:
                - start: "2026-10-20T08:15:40.587Z"
                  end: "2026-10-20T01:24:35-07:00"
            - harness: claude
              id: b6360913-e03e-428d-8115-fbe8d1d46bde
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d8be285cb66264a91ea683d64ff2009e22d787c20ecc873c7001966ac2b3d2e
              started: "2026-10-20T08:17:19.945Z"
              spawned: false
              bytes: 517650
              windows:
                - start: "2026-10-20T08:17:19.945Z"
                  end: "2026-10-20T01:24:35-07:00"
      "n": 1
      timestamp: "2026-10-20T01:27:47-07:00"
      dispatched: "2026-10-20T01:25:07-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: ctrl-c refusal and atlas point at couch --interrupt, which does not exist until M2
          detail: keys.go:57 and atlas/couch.md:2291 direct callers to `couch --interrupt SLOT --confirm`; if M1 lands alone the refusal names a missing verb. Fine if M1 and M2 ship together.
          family: forward-reference-to-unshipped-surface
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Plan M1 claims the skill quick table gained new names; SKILL.md quick table unchanged
          detail: Only the --keys steps sentence was updated (SKILL.md:107-112); either add a quick-table row (e.g. shift-tab) or revise the plan item.
          family: plan-claims-undelivered-item
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: New Steps rows use string-literal KeyKinds beside Key* constants
          detail: keys.go:76-83 use "tab", "shift-tab", etc. while older rows use named constants; cosmetic inconsistency.
          family: mixed-constant-style
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 55335ce84b2411e558a2785ef9616389f6fe1e22
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6076aeca-3b07-4dbc-9bfc-83675a747349
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/8af9025093fe29c0
            reviewed_head: 0b4fa9a21fe77540dc74f54b73edd4380c14b6b3
            slot: pair:2
            requested_at: "2026-10-20T09:16:48.818829Z"
          phase: finished
          pid: 69061
          started_at: "2026-10-20T09:16:48.909576Z"
          finished_at: "2026-10-20T09:26:05.057603Z"
          session_id: 2c6bfc6f-069e-4d85-9047-d930870581b6
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 0b4fa9a21fe77540dc74f54b73edd4380c14b6b3
        tracker_head: 3ac7520410a30125bfb84746d546aa154fc7c5fe
        branch_point: 0b93f91b541fdba4a7bb52196e5abd8d66bebd52
        since: "2026-10-20T01:12:01-07:00"
        until: "2026-10-20T01:44:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T01:12:01-07:00"
              end: "2026-10-20T01:44:26-07:00"
              began: "2026-10-20T01:12:01-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4e1ec141-8d89-49f0-8665-b8191ffad93e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fa2785a7cfaaf3f451dbadb5473fef5653798b54b363415f320397e4c0aac4cb
              started: "2026-10-20T08:20:14.274Z"
              spawned: false
              bytes: 393489
              windows:
                - start: "2026-10-20T08:20:14.274Z"
                  end: "2026-10-20T01:44:26-07:00"
            - harness: claude
              id: 5935e750-db21-494a-ae19-7f77ce8a39fc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 172c0822a9572d9882c4f17b1f7aed34d2a2955171b8a9eea479bddd0e2cf0b6
              started: "2026-10-20T08:18:54.104Z"
              spawned: false
              bytes: 527404
              windows:
                - start: "2026-10-20T08:18:54.104Z"
                  end: "2026-10-20T01:44:26-07:00"
            - harness: claude
              id: 7395260c-761a-4990-8b0d-47a6a2d6d972
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b423a4ba661e0a98bb43fa4a57fe05f23044f51057108640290a4210f780bfb2
              started: "2026-10-20T08:19:58.954Z"
              spawned: false
              bytes: 492806
              windows:
                - start: "2026-10-20T08:19:58.954Z"
                  end: "2026-10-20T01:44:26-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 084c0446d50342a0e1d8a9a2b24ae00b735744932a546aeeff990b7fb9196737
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 53204172
              windows:
                - start: "2026-10-20T01:12:01-07:00"
                  end: "2026-10-20T01:44:26-07:00"
            - harness: claude
              id: ab6148dc-2ee1-41ab-85ce-8be4be326795
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 407bbf48e37a6af04415882970b10fe13a0f1108ce12604b89bec9e002317247
              started: "2026-10-20T08:15:40.587Z"
              spawned: false
              bytes: 555721
              windows:
                - start: "2026-10-20T08:15:40.587Z"
                  end: "2026-10-20T01:44:26-07:00"
            - harness: claude
              id: b6360913-e03e-428d-8115-fbe8d1d46bde
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d8be285cb66264a91ea683d64ff2009e22d787c20ecc873c7001966ac2b3d2e
              started: "2026-10-20T08:17:19.945Z"
              spawned: false
              bytes: 517650
              windows:
                - start: "2026-10-20T08:17:19.945Z"
                  end: "2026-10-20T01:44:26-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T01:25:07-07:00"
              finalized: "2026-10-20T01:27:47-07:00"
              finalizing: true
      "n": 2
      timestamp: "2026-10-20T02:26:05-07:00"
      dispatched: "2026-10-20T02:16:47-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: M2 in this range delivers couch --interrupt (agent_drive_cli.go interrupt branch, ops.go, run.go usage), so the ctrl-c refusal and atlas name a real verb.
          round: 2
        - id: BR-2
          disposition: addressed
          note: SKILL.md quick table gains shift-tab and couch --interrupt rows (SKILL.md:24-25); prose-only, verified against the diff.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: 'Rule the code follows is defensible: a KeyKind gets a named constant only when code outside the table references it (KeyLeft/Right, KeyEsc...); table-only kinds stay literal. No consumer is harmed.'
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Critical
          title: couch --interrupt is admitted and writes Ctrl+C into a slash-command panel with no working turn
          detail: 'wrapcmd/agent_drive.go:71 sets Interruptible = TurnActive && Complete && !picker, but TurnActive is also set while a Claude panel (/usage) shows (#691). Probe in the review checkout (usage panel painted, turnActive set) gave interruptible=true, admission accepted, and a written "\x03"; the drive then confirms that the turn ended. Fix: require positive working-turn evidence (recognized composer, not Dismissable) and add a wrapper test of an interrupt on a panel with the turn flag set.'
          family: admission-proof-from-overloaded-signal
          round: 2
        - kind: defect
          id: BR-5
          severity: Minor
          title: keysUsage repeats StepSyntax's group-once loop instead of sharing a helper
          detail: agent_drive_cli.go:105-113 and keys_help_test.go:24-27 copy keys.go:110-120 (group, else syntax, printed once); expose one agentdrive helper (ARCH-DRY).
          family: duplicated-group-dedupe
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d670bc0d-cf95-4889-902a-8fb36e89a9df
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/125f54a5d9fcbeb4
            reviewed_head: 38665c18cfef50463fa8ad21a618f382f7046d8d
            slot: pair:2
            requested_at: "2026-10-20T10:11:04.203118Z"
          phase: finished
          pid: 9709
          started_at: "2026-10-20T10:11:04.435915Z"
          finished_at: "2026-10-20T10:15:52.62991Z"
          session_id: 35ffdaf5-f5be-4323-802a-13567089a628
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 38665c18cfef50463fa8ad21a618f382f7046d8d
        tracker_head: c5b4a9f472243597b12bb2a1a08a3cab0967385f
        branch_point: 0b93f91b541fdba4a7bb52196e5abd8d66bebd52
        since: "2026-10-20T01:12:01-07:00"
        until: "2026-10-20T02:33:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T01:12:01-07:00"
              end: "2026-10-20T02:33:54-07:00"
              began: "2026-10-20T01:12:01-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4e1ec141-8d89-49f0-8665-b8191ffad93e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fa2785a7cfaaf3f451dbadb5473fef5653798b54b363415f320397e4c0aac4cb
              started: "2026-10-20T08:20:14.274Z"
              spawned: false
              bytes: 393489
              windows:
                - start: "2026-10-20T08:20:14.274Z"
                  end: "2026-10-20T02:33:54-07:00"
            - harness: claude
              id: 5935e750-db21-494a-ae19-7f77ce8a39fc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 172c0822a9572d9882c4f17b1f7aed34d2a2955171b8a9eea479bddd0e2cf0b6
              started: "2026-10-20T08:18:54.104Z"
              spawned: false
              bytes: 527404
              windows:
                - start: "2026-10-20T08:18:54.104Z"
                  end: "2026-10-20T02:33:54-07:00"
            - harness: claude
              id: 7395260c-761a-4990-8b0d-47a6a2d6d972
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b423a4ba661e0a98bb43fa4a57fe05f23044f51057108640290a4210f780bfb2
              started: "2026-10-20T08:19:58.954Z"
              spawned: false
              bytes: 492806
              windows:
                - start: "2026-10-20T08:19:58.954Z"
                  end: "2026-10-20T02:33:54-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b7bf540d3e72a5f51b968ef1e20653cfe8cb96c9cbe09a928474ea32761e8861
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 53443964
              windows:
                - start: "2026-10-20T01:12:01-07:00"
                  end: "2026-10-20T02:33:54-07:00"
            - harness: claude
              id: ab6148dc-2ee1-41ab-85ce-8be4be326795
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 407bbf48e37a6af04415882970b10fe13a0f1108ce12604b89bec9e002317247
              started: "2026-10-20T08:15:40.587Z"
              spawned: false
              bytes: 555721
              windows:
                - start: "2026-10-20T08:15:40.587Z"
                  end: "2026-10-20T02:33:54-07:00"
            - harness: claude
              id: b6360913-e03e-428d-8115-fbe8d1d46bde
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d8be285cb66264a91ea683d64ff2009e22d787c20ecc873c7001966ac2b3d2e
              started: "2026-10-20T08:17:19.945Z"
              spawned: false
              bytes: 517650
              windows:
                - start: "2026-10-20T08:17:19.945Z"
                  end: "2026-10-20T02:33:54-07:00"
        queue_waits:
            - attempt_id: 2caefc480658c640fcaa3b86c6b53759
              slot: pair:2
              record_digest: 3793d9852dd6a1a0b5ba5398aa8bc81c649c4b1eb046752426762de19fdb16d5
              start: "2026-10-20T08:44:33.911984Z"
              end: "2026-10-20T09:07:02.213099Z"
              queue_seconds: 1348.301115
              raw_seconds: 1348.301115
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T01:25:07-07:00"
              finalized: "2026-10-20T01:27:47-07:00"
              finalizing: true
            - dispatched: "2026-10-20T02:16:47-07:00"
              finalized: "2026-10-20T02:26:05-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-20T03:15:53-07:00"
      dispatched: "2026-10-20T03:11:01-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-4
          disposition: addressed
          note: Interruptible now needs turnWorking and a known composer; reverting either guard turns its new subtest red (panel case and no-work case at agent_drive_interrupt_test.go:97).
          round: 3
        - id: BR-5
          disposition: addressed
          note: keys.go StepGroups is the single group-once helper; StepSyntax, keysUsage and keys_help_test all consume it, and the test now asserts every row lands in a help line.
          round: 3
      findings:
        - kind: defect
          id: BR-6
          severity: Critical
          title: couch --interrupt writes Ctrl+C into an idle agent during the 250ms stop-grace window
          detail: '2nd in family. notification_lifecycle.go Stopped keeps Active and ActivitySeen set while GracePending, so turnWorking (line 249) stays true and an idle composer is still known. Probe: Working, then Stopped, then an empty composer gave admission accepted and wrote "\x03", which later becomes Confirmed. Rule: derive Interruptible from one reducer-owned Working() predicate (Active, not Completed, ActivitySeen, not GracePending), not from mirrored flags; add a Stopped-then-interrupt refusal test.'
          family: admission-proof-from-overloaded-signal
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 190d75ac-3ec0-4983-a46d-d148b25aa69b
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/1c1e0955de9c1712
            reviewed_head: 60203319cfd4e0d7fc137c51b151df8986f54377
            slot: pair:2
            requested_at: "2026-10-20T11:23:00.289435Z"
          phase: finished
          pid: 54664
          started_at: "2026-10-20T11:23:00.506314Z"
          finished_at: "2026-10-20T11:30:50.273453Z"
          session_id: 344d278e-309e-44d8-9d3a-314e90ba4d9a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 60203319cfd4e0d7fc137c51b151df8986f54377
        tracker_head: caf1f33c2fb66e18de8e2a9e3c49136732f49009
        branch_point: 0b93f91b541fdba4a7bb52196e5abd8d66bebd52
        since: "2026-10-20T01:12:01-07:00"
        until: "2026-10-20T03:23:16-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T01:12:01-07:00"
              end: "2026-10-20T03:23:16-07:00"
              began: "2026-10-20T01:12:01-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4e1ec141-8d89-49f0-8665-b8191ffad93e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fa2785a7cfaaf3f451dbadb5473fef5653798b54b363415f320397e4c0aac4cb
              started: "2026-10-20T08:20:14.274Z"
              spawned: false
              bytes: 393489
              windows:
                - start: "2026-10-20T08:20:14.274Z"
                  end: "2026-10-20T03:23:16-07:00"
            - harness: claude
              id: 5935e750-db21-494a-ae19-7f77ce8a39fc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 172c0822a9572d9882c4f17b1f7aed34d2a2955171b8a9eea479bddd0e2cf0b6
              started: "2026-10-20T08:18:54.104Z"
              spawned: false
              bytes: 527404
              windows:
                - start: "2026-10-20T08:18:54.104Z"
                  end: "2026-10-20T03:23:16-07:00"
            - harness: claude
              id: 7395260c-761a-4990-8b0d-47a6a2d6d972
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b423a4ba661e0a98bb43fa4a57fe05f23044f51057108640290a4210f780bfb2
              started: "2026-10-20T08:19:58.954Z"
              spawned: false
              bytes: 492806
              windows:
                - start: "2026-10-20T08:19:58.954Z"
                  end: "2026-10-20T03:23:16-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4146d0b2397d77cd3ec6a7c59efbc2e9b67f891591c770c7552fe538b2780668
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 53673968
              windows:
                - start: "2026-10-20T01:12:01-07:00"
                  end: "2026-10-20T03:23:16-07:00"
            - harness: claude
              id: ab6148dc-2ee1-41ab-85ce-8be4be326795
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 407bbf48e37a6af04415882970b10fe13a0f1108ce12604b89bec9e002317247
              started: "2026-10-20T08:15:40.587Z"
              spawned: false
              bytes: 555721
              windows:
                - start: "2026-10-20T08:15:40.587Z"
                  end: "2026-10-20T03:23:16-07:00"
            - harness: claude
              id: b6360913-e03e-428d-8115-fbe8d1d46bde
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d8be285cb66264a91ea683d64ff2009e22d787c20ecc873c7001966ac2b3d2e
              started: "2026-10-20T08:17:19.945Z"
              spawned: false
              bytes: 517650
              windows:
                - start: "2026-10-20T08:17:19.945Z"
                  end: "2026-10-20T03:23:16-07:00"
        queue_waits:
            - attempt_id: 2caefc480658c640fcaa3b86c6b53759
              slot: pair:2
              record_digest: 3793d9852dd6a1a0b5ba5398aa8bc81c649c4b1eb046752426762de19fdb16d5
              start: "2026-10-20T08:44:33.911984Z"
              end: "2026-10-20T09:07:02.213099Z"
              queue_seconds: 1348.301115
              raw_seconds: 1348.301115
              exact: true
            - attempt_id: 4f2f05bcfec7d833f243440ea6c0c3d5
              slot: pair:2
              record_digest: 75620507591580dd643747c871de2663b6689e7deaa7543e9aa8b0909b88dca2
              start: "2026-10-20T09:34:18.084445Z"
              end: "2026-10-20T10:01:23.854084Z"
              queue_seconds: 1625.769639
              raw_seconds: 1625.769639
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T01:25:07-07:00"
              finalized: "2026-10-20T01:27:47-07:00"
              finalizing: true
            - dispatched: "2026-10-20T02:16:47-07:00"
              finalized: "2026-10-20T02:26:05-07:00"
              finalizing: false
            - dispatched: "2026-10-20T03:11:01-07:00"
              finalized: "2026-10-20T03:15:53-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-20T04:30:51-07:00"
      dispatched: "2026-10-20T04:22:57-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-6
          disposition: addressed
          note: Working() (notification_lifecycle.go:95) adds !GracePending and is the sole source of turnWorking; reverting it turns the stop-grace subtest and TestTurnWorkingNeedsReportedActivity red. The gate also requires AdvertisesInterrupt.
          round: 4
      findings:
        - kind: defect
          id: BR-7
          severity: Minor
          title: Plan Core concepts tables omit AdvertisesInterrupt and NotificationLifecycle.Working() added for BR-6
          detail: agentdrive/panel.go AdvertisesInterrupt (pure, new) and wrapcmd/notification_lifecycle.go Working() plus the turnWorking mirror in wrap.go are not rows in either table. Plan line 69 also quotes "no turn to interrupt", but InterruptRefusal says "no working turn to interrupt".
          family: plan-table-omits-delivered-entity
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: 60203319cfd4e0d7fc137c51b151df8986f54377
      blocked: false
---

# Gate ledger — pair#717 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T01:27:47-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `forward-reference-to-unshipped-surface` (kind: defect) ctrl-c refusal and atlas point at couch --interrupt, which does not exist until M2
  keys.go:57 and atlas/couch.md:2291 direct callers to `couch --interrupt SLOT --confirm`; if M1 lands alone the refusal names a missing verb. Fine if M1 and M2 ship together.
- **BR-2** [Minor] `plan-claims-undelivered-item` (kind: defect) Plan M1 claims the skill quick table gained new names; SKILL.md quick table unchanged
  Only the --keys steps sentence was updated (SKILL.md:107-112); either add a quick-table row (e.g. shift-tab) or revise the plan item.
- **BR-3** [Minor] `mixed-constant-style` (kind: defect) New Steps rows use string-literal KeyKinds beside Key* constants
  keys.go:76-83 use "tab", "shift-tab", etc. while older rows use named constants; cosmetic inconsistency.

## Round 2 — 2026-10-20T02:26:05-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — M2 in this range delivers couch --interrupt (agent_drive_cli.go interrupt branch, ops.go, run.go usage), so the ctrl-c refusal and atlas name a real verb.
- BR-2 — addressed — SKILL.md quick table gains shift-tab and couch --interrupt rows (SKILL.md:24-25); prose-only, verified against the diff.
- BR-3 — withdrawn — Rule the code follows is defensible: a KeyKind gets a named constant only when code outside the table references it (KeyLeft/Right, KeyEsc...); table-only kinds stay literal. No consumer is harmed.

### Raised

- **BR-4** [Critical] `admission-proof-from-overloaded-signal` (kind: defect) couch --interrupt is admitted and writes Ctrl+C into a slash-command panel with no working turn
  wrapcmd/agent_drive.go:71 sets Interruptible = TurnActive && Complete && !picker, but TurnActive is also set while a Claude panel (/usage) shows (#691). Probe in the review checkout (usage panel painted, turnActive set) gave interruptible=true, admission accepted, and a written "\x03"; the drive then confirms that the turn ended. Fix: require positive working-turn evidence (recognized composer, not Dismissable) and add a wrapper test of an interrupt on a panel with the turn flag set.
- **BR-5** [Minor] `duplicated-group-dedupe` (kind: defect) keysUsage repeats StepSyntax's group-once loop instead of sharing a helper
  agent_drive_cli.go:105-113 and keys_help_test.go:24-27 copy keys.go:110-120 (group, else syntax, printed once); expose one agentdrive helper (ARCH-DRY).

## Round 3 — 2026-10-20T03:15:53-07:00 (claude) — BLOCKED

### Disposed

- BR-4 — addressed — Interruptible now needs turnWorking and a known composer; reverting either guard turns its new subtest red (panel case and no-work case at agent_drive_interrupt_test.go:97).
- BR-5 — addressed — keys.go StepGroups is the single group-once helper; StepSyntax, keysUsage and keys_help_test all consume it, and the test now asserts every row lands in a help line.

### Raised

- **BR-6** [Critical] `admission-proof-from-overloaded-signal` (kind: defect) couch --interrupt writes Ctrl+C into an idle agent during the 250ms stop-grace window
  2nd in family. notification_lifecycle.go Stopped keeps Active and ActivitySeen set while GracePending, so turnWorking (line 249) stays true and an idle composer is still known. Probe: Working, then Stopped, then an empty composer gave admission accepted and wrote "\x03", which later becomes Confirmed. Rule: derive Interruptible from one reducer-owned Working() predicate (Active, not Completed, ActivitySeen, not GracePending), not from mirrored flags; add a Stopped-then-interrupt refusal test.

## Round 4 — 2026-10-20T04:30:51-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — Working() (notification_lifecycle.go:95) adds !GracePending and is the sole source of turnWorking; reverting it turns the stop-grace subtest and TestTurnWorkingNeedsReportedActivity red. The gate also requires AdvertisesInterrupt.

### Raised

- **BR-7** [Minor] `plan-table-omits-delivered-entity` (kind: defect) Plan Core concepts tables omit AdvertisesInterrupt and NotificationLifecycle.Working() added for BR-6
  agentdrive/panel.go AdvertisesInterrupt (pure, new) and wrapcmd/notification_lifecycle.go Working() plus the turnWorking mirror in wrap.go are not rows in either table. Plan line 69 also quotes "no turn to interrupt", but InterruptRefusal says "no working turn to interrupt".

## Open findings

- **BR-7** [Minor] `plan-table-omits-delivered-entity` (kind: defect) Plan Core concepts tables omit AdvertisesInterrupt and NotificationLifecycle.Working() added for BR-6
