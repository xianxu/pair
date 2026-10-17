---
gate: boundary-review
issue: 632
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 792dfcf0-b995-4283-b4f3-9e46390fb2a2
            repository: github.com/xianxu/pair
            issue_id: "000632"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a9cef0961c4907f5
            reviewed_head: 36a0c2599c93803fb74838b344fb0a0c72651488
            slot: pair:6
            requested_at: "2026-10-17T07:31:09.284679Z"
          phase: finished
          pid: 23489
          started_at: "2026-10-17T07:31:09.52236Z"
          finished_at: "2026-10-17T07:34:45.761704Z"
          session_id: 01a148c5-91c8-7a60-a38f-bf8ef1076c42
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 36a0c2599c93803fb74838b344fb0a0c72651488
        tracker_head: 4c666784abf8c7adce884b3f6bb3375e21974e41
        branch_point: 316474acd6fdd7f61fc8c06dd51b96e2dc3ac0aa
        since: "2026-10-16T22:52:10-07:00"
        until: "2026-10-17T00:29:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T22:52:10-07:00"
              end: "2026-10-17T00:29:26-07:00"
              began: "2026-10-16T22:52:10-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14367-664d-7850-a55c-bb707864cce5
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 1b9fa532e4f76b9df2a36ca9ac2ae16f73d730dacf17fc3e916b3758c070bce7
              started: "2026-10-16T06:30:12.567Z"
              spawned: false
              bytes: 33666671
              windows:
                - start: "2026-10-16T22:52:10-07:00"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a1486b-265e-78c3-a41b-c74e13e10c23
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 420c60a994a3dfc025804c37abce3c5e7eec820cb381b332e090bc13476c10d9
              started: "2026-10-17T05:52:24.235Z"
              spawned: true
              bytes: 787781
              windows:
                - start: "2026-10-17T05:52:24.235Z"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a1486e-480a-7ed1-93c9-db98a4033fcd
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: a51dda37a68547fab9f0dffe1c9d4258a96800dc6085280f42597e72fb715f36
              started: "2026-10-17T05:55:49.911Z"
              spawned: false
              bytes: 585779
              windows:
                - start: "2026-10-17T05:55:49.911Z"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a1486f-511e-7f82-b866-0f5b7e11a11f
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: ed66b6cd5f81c6b9e31fffbb57c2fd6ea73830a136ae540cdec3365a1dca2781
              started: "2026-10-17T05:56:57.743Z"
              spawned: false
              bytes: 422915
              windows:
                - start: "2026-10-17T05:56:57.743Z"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a14870-5559-7d63-953e-9653cce6c783
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: e907ef69e9bd434319d4f3aaa8c13f2127e0bd0fc21daa6019dd8c8699c8971e
              started: "2026-10-17T05:58:04.554Z"
              spawned: false
              bytes: 392521
              windows:
                - start: "2026-10-17T05:58:04.554Z"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a14870-a652-7dc1-9da9-be9eb2e4e762
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2240be14d1c5df4f0083a682ff8dd4398793f671b79618839568ae5ae458a00e
              started: "2026-10-17T05:58:25.082Z"
              spawned: false
              bytes: 383314
              windows:
                - start: "2026-10-17T05:58:25.082Z"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a14871-8b3c-7b63-85f0-7c26ee9cd4ad
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 426ffae79ce6008c5b95b466c1163461a01c01f703d82af558b722eddca25046
              started: "2026-10-17T05:59:23.34Z"
              spawned: true
              bytes: 455388
              windows:
                - start: "2026-10-17T05:59:23.34Z"
                  end: "2026-10-17T00:29:26-07:00"
            - harness: codex
              id: 01a14874-2b91-7252-865b-c8294d1fa010
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 9cde7019f04524fc233bc6fdf79dad52cc88775948540ad5074ed6f39d4abb27
              started: "2026-10-17T06:02:15.449Z"
              spawned: true
              bytes: 605154
              windows:
                - start: "2026-10-17T06:02:15.449Z"
                  end: "2026-10-17T00:29:26-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 58.11745
      "n": 1
      timestamp: "2026-10-17T00:34:47-07:00"
      dispatched: "2026-10-17T00:31:06-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Require a qualified complete footer before reporting known-none
          detail: 'cmd/internal/wrapcmd/peer_composer.go:29-55 accepts any line beginning with “⏵⏵ ” below the composer. If no field contains the complete substring “shell”, it returns false: both “⏵⏵ garbage” and a truncated “… · 1 she” become known-none instead of unknown. This violates the plan’s incomplete/unqualified-footer contract; an active-to-false report can also fabricate completion through registry.go:228-231. Validate recognized no-shell grammar and return nil for malformed or truncated fields. Add negative tests for these cases. ARCH-SECURE, ARCH-ORDER.'
          family: unknown-evidence-preservation
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Correct the Core concepts table’s Console test modification claim
          detail: workshop/plans/000632-background-work-fade-plan.md:49 marks cmd/internal/couchtty/console_activity_test.go as modified, but the pinned name-status and patch contain no change to that file. The revision at line 111 explains intentional reuse, yet leaves the table contradictory. Remove this row from the changed-entity inventory and document it as reused validation, with a Revisions entry reconciling the table. This is Critical under the requested Core concepts cross-check. ARCH-PURPOSE.
          family: core-concept-change-traceability
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9cd92dd5-4610-42dd-8a0b-807af05bcddc
            repository: github.com/xianxu/pair
            issue_id: "000632"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a6b00cd3c16bb1e7
            reviewed_head: c4519837b3f7ddd3d2a8527b07c73b035601efc7
            slot: pair:6
            requested_at: "2026-10-17T08:21:51.200247Z"
          phase: finished
          pid: 56429
          started_at: "2026-10-17T08:21:51.438878Z"
          finished_at: "2026-10-17T08:25:41.455625Z"
          session_id: 01a148f3-fc92-7c82-8b42-a1a75fb1aba9
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c4519837b3f7ddd3d2a8527b07c73b035601efc7
        tracker_head: f0fab12c0a1df69be9544bde71ebceda49830a19
        branch_point: 316474acd6fdd7f61fc8c06dd51b96e2dc3ac0aa
        since: "2026-10-16T22:52:10-07:00"
        until: "2026-10-17T01:21:12-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T22:52:10-07:00"
              end: "2026-10-17T01:21:12-07:00"
              began: "2026-10-16T22:52:10-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14367-664d-7850-a55c-bb707864cce5
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 964ef067a0f83a930ffde14d5ba6a53a3ed04192f52fb2e3aea7474f9dc1656f
              started: "2026-10-16T06:30:12.567Z"
              spawned: false
              bytes: 34607885
              windows:
                - start: "2026-10-16T22:52:10-07:00"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a1486b-265e-78c3-a41b-c74e13e10c23
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 420c60a994a3dfc025804c37abce3c5e7eec820cb381b332e090bc13476c10d9
              started: "2026-10-17T05:52:24.235Z"
              spawned: true
              bytes: 787781
              windows:
                - start: "2026-10-17T05:52:24.235Z"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a1486e-480a-7ed1-93c9-db98a4033fcd
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: a51dda37a68547fab9f0dffe1c9d4258a96800dc6085280f42597e72fb715f36
              started: "2026-10-17T05:55:49.911Z"
              spawned: false
              bytes: 585779
              windows:
                - start: "2026-10-17T05:55:49.911Z"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a1486f-511e-7f82-b866-0f5b7e11a11f
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: ed66b6cd5f81c6b9e31fffbb57c2fd6ea73830a136ae540cdec3365a1dca2781
              started: "2026-10-17T05:56:57.743Z"
              spawned: false
              bytes: 422915
              windows:
                - start: "2026-10-17T05:56:57.743Z"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a14870-5559-7d63-953e-9653cce6c783
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: e907ef69e9bd434319d4f3aaa8c13f2127e0bd0fc21daa6019dd8c8699c8971e
              started: "2026-10-17T05:58:04.554Z"
              spawned: false
              bytes: 392521
              windows:
                - start: "2026-10-17T05:58:04.554Z"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a14870-a652-7dc1-9da9-be9eb2e4e762
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 2240be14d1c5df4f0083a682ff8dd4398793f671b79618839568ae5ae458a00e
              started: "2026-10-17T05:58:25.082Z"
              spawned: false
              bytes: 383314
              windows:
                - start: "2026-10-17T05:58:25.082Z"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a14871-8b3c-7b63-85f0-7c26ee9cd4ad
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 426ffae79ce6008c5b95b466c1163461a01c01f703d82af558b722eddca25046
              started: "2026-10-17T05:59:23.34Z"
              spawned: true
              bytes: 455388
              windows:
                - start: "2026-10-17T05:59:23.34Z"
                  end: "2026-10-17T01:21:12-07:00"
            - harness: codex
              id: 01a14874-2b91-7252-865b-c8294d1fa010
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14367-664d-7850-a55c-bb707864cce5
              digest: 9cde7019f04524fc233bc6fdf79dad52cc88775948540ad5074ed6f39d4abb27
              started: "2026-10-17T06:02:15.449Z"
              spawned: true
              bytes: 605154
              windows:
                - start: "2026-10-17T06:02:15.449Z"
                  end: "2026-10-17T01:21:12-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 102.90711666666667
        cuts:
            - dispatched: "2026-10-17T00:31:06-07:00"
              finalized: "2026-10-17T00:34:47-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-17T01:25:42-07:00"
      dispatched: "2026-10-17T01:21:47-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: peer_composer.go:32-64 now validates complete permission/footer grammar. TestBackgroundWorkUnqualifiedFooterStaysUnknown passes at pinned head; restoring the pre-fix parser in a disposable copy causes all ten cases to fail behaviorally. The copy was removed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The correction diff removes the unchanged Console test from the Core concepts inventory. Plan line 116 explicitly describes it as reused validation; pinned name-status confirms no modification, and console_activity_test.go:285 supplies the referenced activity/aging coverage.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: c4519837b3f7ddd3d2a8527b07c73b035601efc7
      blocked: false
---

# Gate ledger — pair#632 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T00:34:47-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `unknown-evidence-preservation` (kind: defect) Require a qualified complete footer before reporting known-none
  cmd/internal/wrapcmd/peer_composer.go:29-55 accepts any line beginning with “⏵⏵ ” below the composer. If no field contains the complete substring “shell”, it returns false: both “⏵⏵ garbage” and a truncated “… · 1 she” become known-none instead of unknown. This violates the plan’s incomplete/unqualified-footer contract; an active-to-false report can also fabricate completion through registry.go:228-231. Validate recognized no-shell grammar and return nil for malformed or truncated fields. Add negative tests for these cases. ARCH-SECURE, ARCH-ORDER.
- **BR-2** [Critical] `core-concept-change-traceability` (kind: defect) Correct the Core concepts table’s Console test modification claim
  workshop/plans/000632-background-work-fade-plan.md:49 marks cmd/internal/couchtty/console_activity_test.go as modified, but the pinned name-status and patch contain no change to that file. The revision at line 111 explains intentional reuse, yet leaves the table contradictory. Remove this row from the changed-entity inventory and document it as reused validation, with a Revisions entry reconciling the table. This is Critical under the requested Core concepts cross-check. ARCH-PURPOSE.

## Round 2 — 2026-10-17T01:25:42-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — peer_composer.go:32-64 now validates complete permission/footer grammar. TestBackgroundWorkUnqualifiedFooterStaysUnknown passes at pinned head; restoring the pre-fix parser in a disposable copy causes all ten cases to fail behaviorally. The copy was removed.
- BR-2 — addressed — The correction diff removes the unchanged Console test from the Core concepts inventory. Plan line 116 explicitly describes it as reused validation; pinned name-status confirms no modification, and console_activity_test.go:285 supplies the referenced activity/aging coverage.

## Open findings

(none — every finding has been disposed)
