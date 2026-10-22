---
gate: boundary-review
issue: 758
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7a1f5416-a281-4580-9698-8074dacfef93
            repository: github.com/xianxu/pair
            issue_id: "000758"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/653fd1bc9ae6485d
            reviewed_head: 1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf
            slot: pair:6
            requested_at: "2026-10-22T12:58:08.115608Z"
          phase: finished
          pid: 5890
          started_at: "2026-10-22T12:58:08.24957Z"
          finished_at: "2026-10-22T13:01:46.159896Z"
          session_id: 01a163b0-b992-73d2-ba88-164aae0a719c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf
        tracker_head: 5c6b4ccca2c31c8883b5d8ad1d1168952d600881
        branch_point: 1e70379fba4a3653258f5542ac1d1aecf0ca1b43
        since: "2026-10-22T04:59:40-07:00"
        until: "2026-10-22T05:57:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T04:59:40-07:00"
              end: "2026-10-22T05:57:24-07:00"
              began: "2026-10-22T04:59:40-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c8d8c5d9f6970d84722ef9b8d933844e0ca624c9f075efa8b6925db73fb7011c
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 106608768
              windows:
                - start: "2026-10-22T04:59:40-07:00"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a16385-a739-7d02-bb3c-91b494692f62
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 0d5890be200455c6b30bf6f3fecf916917cbe0a51b25858342abd7ac97ddef05
              started: "2026-10-22T12:11:06.635Z"
              spawned: false
              bytes: 606483
              windows:
                - start: "2026-10-22T12:11:06.635Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a16388-246e-7f63-8a56-8030faac80db
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 42404cb3df8e525a88c7e7e06ca1e705434b6ad04081d93389964744e7becef9
              started: "2026-10-22T12:13:49.312Z"
              spawned: false
              bytes: 468029
              windows:
                - start: "2026-10-22T12:13:49.312Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a16389-e7e3-7b83-b2e5-be7e7000c3af
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 53270051d542b69ffa68ac63946416d67fc9809925a5b6d0da5578031da113f9
              started: "2026-10-22T12:15:46.091Z"
              spawned: false
              bytes: 483664
              windows:
                - start: "2026-10-22T12:15:46.091Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638a-810e-70a2-ae02-305e486d3f94
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 94d4f24c41960c868664ddbc1d3ace5d790b646d75a369de42ab7d1c6c8696d4
              started: "2026-10-22T12:16:24.11Z"
              spawned: false
              bytes: 484388
              windows:
                - start: "2026-10-22T12:16:24.11Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638c-09b6-7bd3-bbef-7872745d3d2a
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: dd1fc8a332153f760232991b4d5c0853dc0ed1edeaeb173845d18802257df7f1
              started: "2026-10-22T12:18:04.437Z"
              spawned: true
              bytes: 1220334
              windows:
                - start: "2026-10-22T12:18:04.437Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638c-40b3-7432-9bb5-491da977da77
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 2ba61ee1b44c7ae2867e104a6b22c58adbfb589876ba0c93365b75f98a16c38c
              started: "2026-10-22T12:18:18.521Z"
              spawned: true
              bytes: 1222539
              windows:
                - start: "2026-10-22T12:18:18.521Z"
                  end: "2026-10-22T05:57:24-07:00"
            - harness: codex
              id: 01a1638c-c122-73f0-84e1-124985c62b54
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 1758fbb99c3a3bf81c0c7bd0372e8b9a18183a5b96c058e94f1af888c6fa6c0f
              started: "2026-10-22T12:18:51.407Z"
              spawned: true
              bytes: 1163023
              windows:
                - start: "2026-10-22T12:18:51.407Z"
                  end: "2026-10-22T05:57:24-07:00"
      "n": 1
      timestamp: "2026-10-22T06:01:46-07:00"
      dispatched: "2026-10-22T05:58:05-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: A late negative lookup reports confirmed absence despite a current successor
          detail: cmd/internal/editorendpoint/owner.go:414-424 fences retirement against the captured binding but unconditionally returns Code=ok with EditorEndpoint=null afterward. If B replaces A while A's probe is pending, a subsequent nil response for A cannot establish absence of B. Recheck current authority under the same lock as retirement and return stale or validate the successor. Extend owner_test.go:234 to assert the delayed lookup response; line 260 currently discards it. ARCH-ORDER, ARCH-SECURE.
          family: observation-generation-fencing
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
          detail: cmd/internal/editorendpoint/owner.go:129-131 treats every Store.Publish error as uncertain publication. However, store.go:112-114 can reject encoding before invoking any writer, including codec.go:139-140 rejecting a candidate above 8 MiB. A valid near-capacity registry therefore becomes unavailable for lookup, unregister, and cleanup until restart after one oversized admission. Preflight candidate encoding and classify size exhaustion as capacity without poisoning or changing authority; reserve publication uncertainty for attempted writes. Add an owner-level regression proving no writer call and continued operation after refusal. ARCH-CONSTRAINTS, ARCH-ORDER.
          family: prepublication-refusal-isolation
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
          detail: cmd/internal/editorendpoint/owner_test.go:69-73 only appends notices and always succeeds; couchcmd/editor_endpoint_service_test.go:124-125 is a no-op. The native callback at nvim_live_test.go:65-68 only writes a token. None can hold or fail Superseded, although plan line 168 marks held-notification qualification complete. Add controllable notification completion/failure through the production seam and verify publication precedes delivery, failure preserves the successor, and shutdown cancels and joins an in-flight notification. Model pending/current attachment identity in the editor double. ARCH-MOCK, ARCH-ORDER, ARCH-PURPOSE.
          family: external-lifecycle-qualification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#758 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T06:01:46-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `observation-generation-fencing` (kind: defect) A late negative lookup reports confirmed absence despite a current successor
  cmd/internal/editorendpoint/owner.go:414-424 fences retirement against the captured binding but unconditionally returns Code=ok with EditorEndpoint=null afterward. If B replaces A while A's probe is pending, a subsequent nil response for A cannot establish absence of B. Recheck current authority under the same lock as retirement and return stale or validate the successor. Extend owner_test.go:234 to assert the delayed lookup response; line 260 currently discards it. ARCH-ORDER, ARCH-SECURE.
- **BR-2** [Critical] `prepublication-refusal-isolation` (kind: defect) Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
  cmd/internal/editorendpoint/owner.go:129-131 treats every Store.Publish error as uncertain publication. However, store.go:112-114 can reject encoding before invoking any writer, including codec.go:139-140 rejecting a candidate above 8 MiB. A valid near-capacity registry therefore becomes unavailable for lookup, unregister, and cleanup until restart after one oversized admission. Preflight candidate encoding and classify size exhaustion as capacity without poisoning or changing authority; reserve publication uncertainty for attempted writes. Add an owner-level regression proving no writer call and continued operation after refusal. ARCH-CONSTRAINTS, ARCH-ORDER.
- **BR-3** [Important] `external-lifecycle-qualification` (kind: defect) Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
  cmd/internal/editorendpoint/owner_test.go:69-73 only appends notices and always succeeds; couchcmd/editor_endpoint_service_test.go:124-125 is a no-op. The native callback at nvim_live_test.go:65-68 only writes a token. None can hold or fail Superseded, although plan line 168 marks held-notification qualification complete. Add controllable notification completion/failure through the production seam and verify publication precedes delivery, failure preserves the successor, and shutdown cancels and joins an in-flight notification. Model pending/current attachment identity in the editor double. ARCH-MOCK, ARCH-ORDER, ARCH-PURPOSE.

## Open findings

- **BR-1** [Critical] `observation-generation-fencing` (kind: defect) A late negative lookup reports confirmed absence despite a current successor
- **BR-2** [Critical] `prepublication-refusal-isolation` (kind: defect) Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
- **BR-3** [Important] `external-lifecycle-qualification` (kind: defect) Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
