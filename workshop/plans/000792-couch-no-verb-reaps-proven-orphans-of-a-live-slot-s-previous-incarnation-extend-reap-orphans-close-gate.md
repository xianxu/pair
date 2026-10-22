---
gate: boundary-review
issue: 792
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9d544d22-8f43-4561-97a1-2a6e3417385f
            repository: github.com/xianxu/pair
            issue_id: "000792"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/60f67f47a9451408
            reviewed_head: dfd00c602b36faf734bc87b3964fe139792d376a
            slot: pair:2
            requested_at: "2026-10-22T09:36:55.914965Z"
          phase: finished
          pid: 22555
          started_at: "2026-10-22T09:36:56.087931Z"
          finished_at: "2026-10-22T09:40:13.193829Z"
          session_id: 01a162f8-84e2-7e13-9ab9-29b95aa08559
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: dfd00c602b36faf734bc87b3964fe139792d376a
        tracker_head: c069147fea8ce3dfa1b08348384d304ca8578820
        branch_point: 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f
        since: "2026-10-22T00:59:38-07:00"
        until: "2026-10-22T02:35:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T00:59:38-07:00"
              end: "2026-10-22T02:35:54-07:00"
              began: "2026-10-22T00:59:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 1678d5564f63bede96f8c713d7e637619d718d6f9bfc2d56eb2f1202e8674f8a
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 42640457
              windows:
                - start: "2026-10-22T00:59:38-07:00"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162a0-6690-79e3-9b5a-619de5ccaa8b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 0f9210202d37a08e6da07bddeff96861e8b690fc03f1ec84bcce50739a1bb079
              started: "2026-10-22T08:00:41.809Z"
              spawned: true
              bytes: 829943
              windows:
                - start: "2026-10-22T08:00:41.809Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162aa-fb4e-7970-91a6-42b9a6a9962e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 506543e9015d020fe1467994d6a43f96be26f341883044e5919b66645853e3f6
              started: "2026-10-22T08:12:15.436Z"
              spawned: false
              bytes: 672111
              windows:
                - start: "2026-10-22T08:12:15.436Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162b2-0701-72e3-afe2-d6bf7832e7c0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8c9d4f06007864e64d2cd1ed9b62bec64b0f274f1a24363b8aefe776caaf7251
              started: "2026-10-22T08:19:57.343Z"
              spawned: false
              bytes: 472250
              windows:
                - start: "2026-10-22T08:19:57.343Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162b7-7095-7f20-9646-101dd7e10f0b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2949391f4fd89b32dfc4645845c66181542120724d93c8c95a1e3d51c7124274
              started: "2026-10-22T08:25:51.993Z"
              spawned: false
              bytes: 457013
              windows:
                - start: "2026-10-22T08:25:51.993Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162b8-2620-7f21-8b19-af572c33611a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: f0f0d3fa15536c04b0d579eb9cbac47145cb6b43eafea4d07d96016236ab9c55
              started: "2026-10-22T08:26:38.369Z"
              spawned: false
              bytes: 394000
              windows:
                - start: "2026-10-22T08:26:38.369Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162b9-18b9-77a0-b3c1-6f8f773de441
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 0b4bcb4c77aa23f5254aaf99e17f84b16fb0483476815833e8aff6a2fa98bb83
              started: "2026-10-22T08:27:40.154Z"
              spawned: true
              bytes: 3430048
              windows:
                - start: "2026-10-22T08:27:40.154Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162ba-f349-76d0-be3b-8d82d59eeaf5
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: c17d31218dce0d0e48f2de0b95896aa2848a9547dd157f4298627e8c3a45ff59
              started: "2026-10-22T08:29:41.717Z"
              spawned: true
              bytes: 1276240
              windows:
                - start: "2026-10-22T08:29:41.717Z"
                  end: "2026-10-22T02:35:54-07:00"
            - harness: codex
              id: 01a162bb-5658-7632-b1b2-6414b758cf4e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 5573a1f91fc0669683e653a141c7232b4147c1cb9bc68fcec70a8ca83233ca8e
              started: "2026-10-22T08:30:07.029Z"
              spawned: true
              bytes: 941165
              windows:
                - start: "2026-10-22T08:30:07.029Z"
                  end: "2026-10-22T02:35:54-07:00"
      "n": 1
      timestamp: "2026-10-22T02:40:16-07:00"
      dispatched: "2026-10-22T02:36:52-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Response projection can route proven orphans through unguarded legacy tree reaping
          detail: cmd/internal/couchcmd/message_service.go:980 projects the confirmed sweep before CLI admission. projectResourceView independently truncates Orphans and LeftRunning (resources_projection.go:66-72), but provenOrphanAdmission protects only PIDs in the received Orphans rows (reap_orphans.go:340-341). With enough orphan rows to exceed their section budget and a single LeftRunning entry overlapping an omitted orphan, that entry survives projection and reaches reapLeftRunning at line 123. It then uses local expanding-tree reaping without owning-hub or current-ownership checks, potentially signalling a foreign orphan and current or unproven descendants. ARCH-PURPOSE, ARCH-SECURE, and ARCH-CONSTRAINTS require omission to preserve protection. Remove all orphan overlaps from legacy families on the supervisor's complete view before projection, or preserve an independently complete exclusion set and refuse when it cannot fit. Add a regression through actual projection and CLI execution that proves zero legacy signals for an omitted overlapping orphan.
          family: incomplete-evidence-must-not-widen-authority
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5cd3e3bb-e9a6-4903-9dd3-e7a96e39c4df
            repository: github.com/xianxu/pair
            issue_id: "000792"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/02e5fea098bb3fd7
            reviewed_head: 03d0c96e4f24b3ce6fcf82df47f334b573929766
            slot: pair:2
            requested_at: "2026-10-22T09:48:06.682902Z"
          phase: finished
          pid: 37480
          started_at: "2026-10-22T09:48:06.827115Z"
          finished_at: "2026-10-22T09:54:11.4431Z"
          session_id: 01a16302-c0e2-7523-847c-b3a427221d31
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 03d0c96e4f24b3ce6fcf82df47f334b573929766
        tracker_head: c069147fea8ce3dfa1b08348384d304ca8578820
        branch_point: 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f
        since: "2026-10-22T00:59:38-07:00"
        until: "2026-10-22T02:47:22-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T00:59:38-07:00"
              end: "2026-10-22T02:47:22-07:00"
              began: "2026-10-22T00:59:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 1771cebe12f77cb1edf64b7be16bc92cec0f0ac743a25847c19e55b55fd20c22
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 43169153
              windows:
                - start: "2026-10-22T00:59:38-07:00"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162a0-6690-79e3-9b5a-619de5ccaa8b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 0f9210202d37a08e6da07bddeff96861e8b690fc03f1ec84bcce50739a1bb079
              started: "2026-10-22T08:00:41.809Z"
              spawned: true
              bytes: 829943
              windows:
                - start: "2026-10-22T08:00:41.809Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162aa-fb4e-7970-91a6-42b9a6a9962e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 506543e9015d020fe1467994d6a43f96be26f341883044e5919b66645853e3f6
              started: "2026-10-22T08:12:15.436Z"
              spawned: false
              bytes: 672111
              windows:
                - start: "2026-10-22T08:12:15.436Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162b2-0701-72e3-afe2-d6bf7832e7c0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8c9d4f06007864e64d2cd1ed9b62bec64b0f274f1a24363b8aefe776caaf7251
              started: "2026-10-22T08:19:57.343Z"
              spawned: false
              bytes: 472250
              windows:
                - start: "2026-10-22T08:19:57.343Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162b7-7095-7f20-9646-101dd7e10f0b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2949391f4fd89b32dfc4645845c66181542120724d93c8c95a1e3d51c7124274
              started: "2026-10-22T08:25:51.993Z"
              spawned: false
              bytes: 457013
              windows:
                - start: "2026-10-22T08:25:51.993Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162b8-2620-7f21-8b19-af572c33611a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: f0f0d3fa15536c04b0d579eb9cbac47145cb6b43eafea4d07d96016236ab9c55
              started: "2026-10-22T08:26:38.369Z"
              spawned: false
              bytes: 394000
              windows:
                - start: "2026-10-22T08:26:38.369Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162b9-18b9-77a0-b3c1-6f8f773de441
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 02771ff534751bb26bdc672f847ba9a289de965837931675c2c00d45ad7f372e
              started: "2026-10-22T08:27:40.154Z"
              spawned: true
              bytes: 3513840
              windows:
                - start: "2026-10-22T08:27:40.154Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162ba-f349-76d0-be3b-8d82d59eeaf5
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: fd358504726f78e8db2a52cc1a0ef634a356f5bc1ceed6412db47ed077ffd569
              started: "2026-10-22T08:29:41.717Z"
              spawned: true
              bytes: 1386710
              windows:
                - start: "2026-10-22T08:29:41.717Z"
                  end: "2026-10-22T02:47:22-07:00"
            - harness: codex
              id: 01a162bb-5658-7632-b1b2-6414b758cf4e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 5573a1f91fc0669683e653a141c7232b4147c1cb9bc68fcec70a8ca83233ca8e
              started: "2026-10-22T08:30:07.029Z"
              spawned: true
              bytes: 941165
              windows:
                - start: "2026-10-22T08:30:07.029Z"
                  end: "2026-10-22T02:47:22-07:00"
        cuts:
            - dispatched: "2026-10-22T02:36:52-07:00"
              finalized: "2026-10-22T02:40:16-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-22T02:54:12-07:00"
      dispatched: "2026-10-22T02:48:03-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: resources_projection.go filters both legacy families before truncation. The projection-to-CLI regressions pass at head; removing that filter in a disposable checkout makes all four overlap cases signal both the old PID and its current descendant. The disposable checkout was removed.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: Selection truncation discards orphan protection before legacy admission
          detail: reap_orphans.go:107 derives protection only from returned Orphans. Before projection, SelectResourceProcesses (resource.go:349-351) drops orphan rows when 8192 current-owned rows fill its selection, and AdvanceResources replaces remembered provenance with that truncated selection. A disposable regression through the real sampler, service and confirmed CLI produced zero orphan rows, retained the overlapping LeftRunning row, signalled old PID 11 and unproven child 12 through legacy reaping, and exited 0 despite coverage errors. This is the 2nd finding in family incomplete-evidence-must-not-widen-authority. Earlier rounds fixed instances; do not fix only this instance. Enforce the rule that incomplete evidence never widens signalling authority, including across sampler passes. Sweep table/slot limits and incomplete census, selection truncation/conflicts/unknown classification, cached/RPC projection, CLI admission and both legacy families. Preserve conservative exclusions independently of lossy selection, refusing when protection cannot be retained, and add sequence regressions.
          family: incomplete-evidence-must-not-widen-authority
          round: 2
        - kind: defect
          id: BR-3
          severity: Important
          title: New RPC fixture exceeds the Unix socket path limit with long TMPDIR
          detail: 'agentfixture/resources_test.go:42 creates its isolated root beneath inherited TMPDIR. In this review environment, TestResourceObservationUsesExistingPrivateRPCAndCallerCancellation fails at line 59 with “isolated message socket exceeds 103 bytes”; the same test passes with TMPDIR=/tmp. ARCH-CONSTRAINTS: use a short private fixture root, following couchmessage/transport_test.go:23, retain cleanup, and verify under a long TMPDIR.'
          family: fixtures-must-enforce-platform-bounds
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 15dc2879-fa83-4221-8abd-c5e1be9cca5c
            repository: github.com/xianxu/pair
            issue_id: "000792"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9f6bb5d4834b8b2c
            reviewed_head: aeb3e97b256c63c2f6e5abc853c625258631dc2f
            slot: pair:2
            requested_at: "2026-10-22T10:34:49.00592Z"
          phase: finished
          pid: 5207
          started_at: "2026-10-22T10:34:49.137164Z"
          finished_at: "2026-10-22T10:37:33.456253Z"
          session_id: 01a1632d-833a-7f00-9c7a-fce83ee674ec
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: aeb3e97b256c63c2f6e5abc853c625258631dc2f
        tracker_head: b87f659c8435410eb11b9dca5a982fab8864f8be
        branch_point: 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f
        since: "2026-10-22T00:59:38-07:00"
        until: "2026-10-22T03:34:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T00:59:38-07:00"
              end: "2026-10-22T03:34:04-07:00"
              began: "2026-10-22T00:59:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 9fd1be0c7766d843d0c9b87fc3078a27a971e7376af5159029e6d61bfd7cefe1
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 45041449
              windows:
                - start: "2026-10-22T00:59:38-07:00"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162a0-6690-79e3-9b5a-619de5ccaa8b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 0f9210202d37a08e6da07bddeff96861e8b690fc03f1ec84bcce50739a1bb079
              started: "2026-10-22T08:00:41.809Z"
              spawned: true
              bytes: 829943
              windows:
                - start: "2026-10-22T08:00:41.809Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162aa-fb4e-7970-91a6-42b9a6a9962e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 506543e9015d020fe1467994d6a43f96be26f341883044e5919b66645853e3f6
              started: "2026-10-22T08:12:15.436Z"
              spawned: false
              bytes: 672111
              windows:
                - start: "2026-10-22T08:12:15.436Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162b2-0701-72e3-afe2-d6bf7832e7c0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8c9d4f06007864e64d2cd1ed9b62bec64b0f274f1a24363b8aefe776caaf7251
              started: "2026-10-22T08:19:57.343Z"
              spawned: false
              bytes: 472250
              windows:
                - start: "2026-10-22T08:19:57.343Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162b7-7095-7f20-9646-101dd7e10f0b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2949391f4fd89b32dfc4645845c66181542120724d93c8c95a1e3d51c7124274
              started: "2026-10-22T08:25:51.993Z"
              spawned: false
              bytes: 457013
              windows:
                - start: "2026-10-22T08:25:51.993Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162b8-2620-7f21-8b19-af572c33611a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: f0f0d3fa15536c04b0d579eb9cbac47145cb6b43eafea4d07d96016236ab9c55
              started: "2026-10-22T08:26:38.369Z"
              spawned: false
              bytes: 394000
              windows:
                - start: "2026-10-22T08:26:38.369Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162b9-18b9-77a0-b3c1-6f8f773de441
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: cc39620bec1952925dd61a79a6529862c51b30ddd8525a398c3db99bbf4fcb25
              started: "2026-10-22T08:27:40.154Z"
              spawned: true
              bytes: 4569276
              windows:
                - start: "2026-10-22T08:27:40.154Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162ba-f349-76d0-be3b-8d82d59eeaf5
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 19469680a5d5ab0d0af2b6af53595f09e336a61eaa7af6246e4c589274b38019
              started: "2026-10-22T08:29:41.717Z"
              spawned: true
              bytes: 1776551
              windows:
                - start: "2026-10-22T08:29:41.717Z"
                  end: "2026-10-22T03:34:04-07:00"
            - harness: codex
              id: 01a162bb-5658-7632-b1b2-6414b758cf4e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: b44bc246e0662e6414847c9753c674c061604baa2a872545d5b8dee8e6ee1f40
              started: "2026-10-22T08:30:07.029Z"
              spawned: true
              bytes: 1673995
              windows:
                - start: "2026-10-22T08:30:07.029Z"
                  end: "2026-10-22T03:34:04-07:00"
        cuts:
            - dispatched: "2026-10-22T02:36:52-07:00"
              finalized: "2026-10-22T02:40:16-07:00"
              finalizing: false
            - dispatched: "2026-10-22T02:48:03-07:00"
              finalized: "2026-10-22T02:54:12-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-22T03:37:34-07:00"
      dispatched: "2026-10-22T03:34:46-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Pre-projection overlap exclusion remains present for both legacy families; the projection regressions pass. No reopening.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: 'cmd/internal/couchcmd/reap_orphans.go:101 snapshots legacyRefused before executing exact effects at line 126, then uses that stale value to authorize both legacy families at lines 130–131. A disposable pinned-head regression admitted a healthy classification, injected census failure before the exact signal request, and confirmed the real sampler latched legacy refusal. The exact target was refused, but legacy nvim PID 51 and registered PID 52 both received TERM and were reported reaped. ARCH-PURPOSE/ARCH-ORDER/ARCH-SECURE: the rule must cover outstanding invocations, not only classifications requested after evidence loss. Keep this existing family open; enforce revocation at the legacy effect boundary and sweep loss during exact dispatch, waits, and legacy execution across both families.'
          round: 3
        - id: BR-3
          disposition: addressed
          note: agentfixture/resources_test.go:41–70 creates and cleans a short private socket root independently of inherited TMPDIR. The regression passes with its deliberately long TMPDIR; replacing the short-root call with t.TempDir in a disposable checkout makes the same test fail with “isolated message socket exceeds 103 bytes”.
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 044472a5-e968-48cc-8e82-952295e581cd
            repository: github.com/xianxu/pair
            issue_id: "000792"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/0b9d11461d29653b
            reviewed_head: c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934
            slot: pair:2
            requested_at: "2026-10-22T11:03:53.822313Z"
          phase: finished
          pid: 46193
          started_at: "2026-10-22T11:03:53.972037Z"
          finished_at: "2026-10-22T11:10:21.302608Z"
          session_id: 01a16348-22f2-7b62-8548-95b9bb096a14
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934
        tracker_head: b87f659c8435410eb11b9dca5a982fab8864f8be
        branch_point: 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f
        since: "2026-10-22T00:59:38-07:00"
        until: "2026-10-22T04:02:53-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T00:59:38-07:00"
              end: "2026-10-22T04:02:53-07:00"
              began: "2026-10-22T00:59:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 420a47aac883ddfeed1a4406563de023b79cd8226ed69abb1e7b265d56d5b5bc
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 47196464
              windows:
                - start: "2026-10-22T00:59:38-07:00"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162a0-6690-79e3-9b5a-619de5ccaa8b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 0f9210202d37a08e6da07bddeff96861e8b690fc03f1ec84bcce50739a1bb079
              started: "2026-10-22T08:00:41.809Z"
              spawned: true
              bytes: 829943
              windows:
                - start: "2026-10-22T08:00:41.809Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162aa-fb4e-7970-91a6-42b9a6a9962e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 506543e9015d020fe1467994d6a43f96be26f341883044e5919b66645853e3f6
              started: "2026-10-22T08:12:15.436Z"
              spawned: false
              bytes: 672111
              windows:
                - start: "2026-10-22T08:12:15.436Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162b2-0701-72e3-afe2-d6bf7832e7c0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 8c9d4f06007864e64d2cd1ed9b62bec64b0f274f1a24363b8aefe776caaf7251
              started: "2026-10-22T08:19:57.343Z"
              spawned: false
              bytes: 472250
              windows:
                - start: "2026-10-22T08:19:57.343Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162b7-7095-7f20-9646-101dd7e10f0b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2949391f4fd89b32dfc4645845c66181542120724d93c8c95a1e3d51c7124274
              started: "2026-10-22T08:25:51.993Z"
              spawned: false
              bytes: 457013
              windows:
                - start: "2026-10-22T08:25:51.993Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162b8-2620-7f21-8b19-af572c33611a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: f0f0d3fa15536c04b0d579eb9cbac47145cb6b43eafea4d07d96016236ab9c55
              started: "2026-10-22T08:26:38.369Z"
              spawned: false
              bytes: 394000
              windows:
                - start: "2026-10-22T08:26:38.369Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162b9-18b9-77a0-b3c1-6f8f773de441
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: bfe4dab928dbc96dc96470f03ce5a08eef8c22ada3fa3cd63accd2eedc02b253
              started: "2026-10-22T08:27:40.154Z"
              spawned: true
              bytes: 5045611
              windows:
                - start: "2026-10-22T08:27:40.154Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162ba-f349-76d0-be3b-8d82d59eeaf5
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 9b164d7e2fc48b3c7df93c8f9d3aa49295f3951bd4df2d2a1266477e97ce96d7
              started: "2026-10-22T08:29:41.717Z"
              spawned: true
              bytes: 2256827
              windows:
                - start: "2026-10-22T08:29:41.717Z"
                  end: "2026-10-22T04:02:53-07:00"
            - harness: codex
              id: 01a162bb-5658-7632-b1b2-6414b758cf4e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 03fe99c8758b01c4dd2a2b8dec7f63805c90753f546d976e892d9cc7adb48351
              started: "2026-10-22T08:30:07.029Z"
              spawned: true
              bytes: 2309715
              windows:
                - start: "2026-10-22T08:30:07.029Z"
                  end: "2026-10-22T04:02:53-07:00"
        cuts:
            - dispatched: "2026-10-22T02:36:52-07:00"
              finalized: "2026-10-22T02:40:16-07:00"
              finalizing: false
            - dispatched: "2026-10-22T02:48:03-07:00"
              finalized: "2026-10-22T02:54:12-07:00"
              finalizing: false
            - dispatched: "2026-10-22T03:34:46-07:00"
              finalized: "2026-10-22T03:37:34-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-22T04:10:22-07:00"
      dispatched: "2026-10-22T04:03:50-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-2
          disposition: addressed
          note: resources.go:261–272 latches incomplete ownership evidence; reap_legacy_signal.go:49–88 checks that latch under the effect gate through identity rereads and signalling. Sequence and late-loss regressions pass. In disposable pinned-head copies, removing the latch reproduced signals to protected PIDs 11/12; removing the effect-time check reproduced late TERM/KILL signals across both legacy families. Both disposable copies were removed.
          round: 4
        - id: BR-1
          disposition: addressed
          note: Protection still precedes lossy projection in resources_projection.go. Projection-to-CLI regressions pass for both legacy families, including incomplete witnesses and repeated projection.
          round: 4
        - id: BR-3
          disposition: addressed
          note: agentfixture/resources_test.go:41–70 retains a short private socket root independent of TMPDIR, with cleanup. Its deliberately long-TMPDIR regression passes.
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934
      blocked: false
---

# Gate ledger — pair#792 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T02:40:16-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `incomplete-evidence-must-not-widen-authority` (kind: defect) Response projection can route proven orphans through unguarded legacy tree reaping
  cmd/internal/couchcmd/message_service.go:980 projects the confirmed sweep before CLI admission. projectResourceView independently truncates Orphans and LeftRunning (resources_projection.go:66-72), but provenOrphanAdmission protects only PIDs in the received Orphans rows (reap_orphans.go:340-341). With enough orphan rows to exceed their section budget and a single LeftRunning entry overlapping an omitted orphan, that entry survives projection and reaches reapLeftRunning at line 123. It then uses local expanding-tree reaping without owning-hub or current-ownership checks, potentially signalling a foreign orphan and current or unproven descendants. ARCH-PURPOSE, ARCH-SECURE, and ARCH-CONSTRAINTS require omission to preserve protection. Remove all orphan overlaps from legacy families on the supervisor's complete view before projection, or preserve an independently complete exclusion set and refuse when it cannot fit. Add a regression through actual projection and CLI execution that proves zero legacy signals for an omitted overlapping orphan.

## Round 2 — 2026-10-22T02:54:12-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — resources_projection.go filters both legacy families before truncation. The projection-to-CLI regressions pass at head; removing that filter in a disposable checkout makes all four overlap cases signal both the old PID and its current descendant. The disposable checkout was removed.

### Raised

- **BR-2** [Critical] `incomplete-evidence-must-not-widen-authority` (kind: defect) Selection truncation discards orphan protection before legacy admission
  reap_orphans.go:107 derives protection only from returned Orphans. Before projection, SelectResourceProcesses (resource.go:349-351) drops orphan rows when 8192 current-owned rows fill its selection, and AdvanceResources replaces remembered provenance with that truncated selection. A disposable regression through the real sampler, service and confirmed CLI produced zero orphan rows, retained the overlapping LeftRunning row, signalled old PID 11 and unproven child 12 through legacy reaping, and exited 0 despite coverage errors. This is the 2nd finding in family incomplete-evidence-must-not-widen-authority. Earlier rounds fixed instances; do not fix only this instance. Enforce the rule that incomplete evidence never widens signalling authority, including across sampler passes. Sweep table/slot limits and incomplete census, selection truncation/conflicts/unknown classification, cached/RPC projection, CLI admission and both legacy families. Preserve conservative exclusions independently of lossy selection, refusing when protection cannot be retained, and add sequence regressions.
- **BR-3** [Important] `fixtures-must-enforce-platform-bounds` (kind: defect) New RPC fixture exceeds the Unix socket path limit with long TMPDIR
  agentfixture/resources_test.go:42 creates its isolated root beneath inherited TMPDIR. In this review environment, TestResourceObservationUsesExistingPrivateRPCAndCallerCancellation fails at line 59 with “isolated message socket exceeds 103 bytes”; the same test passes with TMPDIR=/tmp. ARCH-CONSTRAINTS: use a short private fixture root, following couchmessage/transport_test.go:23, retain cleanup, and verify under a long TMPDIR.

## Round 3 — 2026-10-22T03:37:34-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Pre-projection overlap exclusion remains present for both legacy families; the projection regressions pass. No reopening.
- BR-2 — not-addressed — cmd/internal/couchcmd/reap_orphans.go:101 snapshots legacyRefused before executing exact effects at line 126, then uses that stale value to authorize both legacy families at lines 130–131. A disposable pinned-head regression admitted a healthy classification, injected census failure before the exact signal request, and confirmed the real sampler latched legacy refusal. The exact target was refused, but legacy nvim PID 51 and registered PID 52 both received TERM and were reported reaped. ARCH-PURPOSE/ARCH-ORDER/ARCH-SECURE: the rule must cover outstanding invocations, not only classifications requested after evidence loss. Keep this existing family open; enforce revocation at the legacy effect boundary and sweep loss during exact dispatch, waits, and legacy execution across both families.
- BR-3 — addressed — agentfixture/resources_test.go:41–70 creates and cleans a short private socket root independently of inherited TMPDIR. The regression passes with its deliberately long TMPDIR; replacing the short-root call with t.TempDir in a disposable checkout makes the same test fail with “isolated message socket exceeds 103 bytes”.

## Round 4 — 2026-10-22T04:10:22-07:00 (codex) — passed

### Disposed

- BR-2 — addressed — resources.go:261–272 latches incomplete ownership evidence; reap_legacy_signal.go:49–88 checks that latch under the effect gate through identity rereads and signalling. Sequence and late-loss regressions pass. In disposable pinned-head copies, removing the latch reproduced signals to protected PIDs 11/12; removing the effect-time check reproduced late TERM/KILL signals across both legacy families. Both disposable copies were removed.
- BR-1 — addressed — Protection still precedes lossy projection in resources_projection.go. Projection-to-CLI regressions pass for both legacy families, including incomplete witnesses and repeated projection.
- BR-3 — addressed — agentfixture/resources_test.go:41–70 retains a short private socket root independent of TMPDIR, with cleanup. Its deliberately long-TMPDIR regression passes.

## Open findings

(none — every finding has been disposed)
