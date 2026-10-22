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
---

# Gate ledger — pair#792 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T02:40:16-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `incomplete-evidence-must-not-widen-authority` (kind: defect) Response projection can route proven orphans through unguarded legacy tree reaping
  cmd/internal/couchcmd/message_service.go:980 projects the confirmed sweep before CLI admission. projectResourceView independently truncates Orphans and LeftRunning (resources_projection.go:66-72), but provenOrphanAdmission protects only PIDs in the received Orphans rows (reap_orphans.go:340-341). With enough orphan rows to exceed their section budget and a single LeftRunning entry overlapping an omitted orphan, that entry survives projection and reaches reapLeftRunning at line 123. It then uses local expanding-tree reaping without owning-hub or current-ownership checks, potentially signalling a foreign orphan and current or unproven descendants. ARCH-PURPOSE, ARCH-SECURE, and ARCH-CONSTRAINTS require omission to preserve protection. Remove all orphan overlaps from legacy families on the supervisor's complete view before projection, or preserve an independently complete exclusion set and refuse when it cannot fit. Add a regression through actual projection and CLI execution that proves zero legacy signals for an omitted overlapping orphan.

## Open findings

- **BR-1** [Critical] `incomplete-evidence-must-not-widen-authority` (kind: defect) Response projection can route proven orphans through unguarded legacy tree reaping
