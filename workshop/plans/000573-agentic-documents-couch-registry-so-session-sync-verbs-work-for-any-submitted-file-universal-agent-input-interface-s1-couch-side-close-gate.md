---
gate: boundary-review
issue: 573
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 15ce5ed6-357a-4e41-8c31-1db418eb5595
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7ef127e48f5f4414
            reviewed_head: 52a0ff84621e6a41f057af5a3b8d1c4d03c0d560
            slot: pair:5
            requested_at: "2026-10-15T19:57:01.859517Z"
          phase: finished
          pid: 98632
          started_at: "2026-10-15T19:57:02.10891Z"
          finished_at: "2026-10-15T19:59:40.001749Z"
          session_id: 01a14123-b8c4-76a1-b0ee-e416704ea4bc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 52a0ff84621e6a41f057af5a3b8d1c4d03c0d560
        tracker_head: 09e2c47977b577821fb8eac6a7fce89431044b4c
        branch_point: d6da7d4b809729db8a8766bd71258ca8f3326d9a
        since: "2026-10-15T10:56:32-07:00"
        until: "2026-10-15T12:56:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:56:32-07:00"
              end: "2026-10-15T12:56:24-07:00"
              began: "2026-10-15T10:56:32-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 1f502758993f760c5dc442232077eb26ca20ab91e84477626922874f26d9856a
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 13738297
              windows:
                - start: "2026-10-15T10:56:32-07:00"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140b5-e539-7301-b7cf-0082310a7bc4
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 3b18b67365f6828afd7347bbda17a9e4df2576434f07234af887cd60f970f542
              started: "2026-10-15T17:57:05.01Z"
              spawned: true
              bytes: 947597
              windows:
                - start: "2026-10-15T17:57:05.01Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140ba-1c00-70d0-9c5b-08da3d2bf056
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 0a78fbc8d4db263c1e4a431e451be1c198b022cec2d1ca8640de3ab950699039
              started: "2026-10-15T18:01:42.668Z"
              spawned: false
              bytes: 606915
              windows:
                - start: "2026-10-15T18:01:42.668Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140bc-d433-7991-9270-798d625318dd
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: bf573990502088b7b88da236f02a9d544220fb72c54899cb21d01e95a563a43b
              started: "2026-10-15T18:04:41.148Z"
              spawned: false
              bytes: 461974
              windows:
                - start: "2026-10-15T18:04:41.148Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140bd-fca4-7711-990d-cc9e591b005e
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: eb8a9d2483c94214c247a57805ce651731e481599ccf9767cbf940957863d0e5
              started: "2026-10-15T18:05:56.635Z"
              spawned: false
              bytes: 443958
              windows:
                - start: "2026-10-15T18:05:56.635Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140be-6458-7a91-9764-caea54f76cbb
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 61f03c9b48eb8551cd697bd3cf2330ab6774c0c33ba24be2f29c63b71169939b
              started: "2026-10-15T18:06:22.518Z"
              spawned: false
              bytes: 384768
              windows:
                - start: "2026-10-15T18:06:22.518Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140bf-dd65-7cf3-a13b-795ffd04e917
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: e7601d47b02be43a0c9468969b1751a2c8eda33a612ee217635096dc7fcb20f4
              started: "2026-10-15T18:07:58.475Z"
              spawned: true
              bytes: 499863
              windows:
                - start: "2026-10-15T18:07:58.475Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140c9-1723-7412-87c2-1f5333b222f5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 034fcf83107a5311e22f8c8e118e42779edb601c63156112a14dc89a5499eafd
              started: "2026-10-15T18:18:03.165Z"
              spawned: true
              bytes: 308941
              windows:
                - start: "2026-10-15T18:18:03.165Z"
                  end: "2026-10-15T12:56:24-07:00"
      "n": 1
      timestamp: "2026-10-15T12:59:41-07:00"
      dispatched: "2026-10-15T12:56:59-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Repeated registration discards additional aliases and permits writes under their operator locks
          detail: 'cmd/internal/sessionsync/registry.go:256 returns from existing-registration handling without retaining the requested spelling; documentLockPaths at line 529 can inspect only the canonical path, current request, and stored spellings. Reproduced by registering FILE, registering a symlink ALIAS to FILE, creating ALIAS.lock, then writing FILE: the write succeeds despite the operator lock. Preserve all admitted live spellings through the registry transition model, with bounded growth, and use them consistently for lock checks and legacy handoff. ARCH-ORDER / ARCH-PURPOSE.'
          family: canonical-document-lock-authority
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Shadow writers exceed the registered reader limit and strand successful operations
          detail: 'cmd/internal/sessionsync/sync.go:417 bounds only the resulting document; line 432 can publish a larger section-preserving shadow, while registry.go:496 refuses shadows over 4 MiB. Reproduced with an approximately 2 MiB baseline section, an operator deletion elsewhere, and an approximately 2 MiB section write: Write succeeds, then Read fails with registry-limit. Independently, registry.go:287-311 imports an oversized legacy shadow without validation, returns registration success, and leaves subsequent reads failing. Validate complete prospective metadata before document mutation or registry publication; sweep every metadata writer against its reader limits and add regressions for both paths. ARCH-CONSTRAINTS / ARCH-FUNERAL.'
          family: persisted-writer-reader-envelope
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#573 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T12:59:41-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `canonical-document-lock-authority` (kind: defect) Repeated registration discards additional aliases and permits writes under their operator locks
  cmd/internal/sessionsync/registry.go:256 returns from existing-registration handling without retaining the requested spelling; documentLockPaths at line 529 can inspect only the canonical path, current request, and stored spellings. Reproduced by registering FILE, registering a symlink ALIAS to FILE, creating ALIAS.lock, then writing FILE: the write succeeds despite the operator lock. Preserve all admitted live spellings through the registry transition model, with bounded growth, and use them consistently for lock checks and legacy handoff. ARCH-ORDER / ARCH-PURPOSE.
- **BR-2** [Critical] `persisted-writer-reader-envelope` (kind: defect) Shadow writers exceed the registered reader limit and strand successful operations
  cmd/internal/sessionsync/sync.go:417 bounds only the resulting document; line 432 can publish a larger section-preserving shadow, while registry.go:496 refuses shadows over 4 MiB. Reproduced with an approximately 2 MiB baseline section, an operator deletion elsewhere, and an approximately 2 MiB section write: Write succeeds, then Read fails with registry-limit. Independently, registry.go:287-311 imports an oversized legacy shadow without validation, returns registration success, and leaves subsequent reads failing. Validate complete prospective metadata before document mutation or registry publication; sweep every metadata writer against its reader limits and add regressions for both paths. ARCH-CONSTRAINTS / ARCH-FUNERAL.

## Open findings

- **BR-1** [Critical] `canonical-document-lock-authority` (kind: defect) Repeated registration discards additional aliases and permits writes under their operator locks
- **BR-2** [Critical] `persisted-writer-reader-envelope` (kind: defect) Shadow writers exceed the registered reader limit and strand successful operations
