---
gate: boundary-review
issue: 571
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5313b275-fbc3-463c-8549-9eb73e49f8d9
            repository: github.com/xianxu/pair
            issue_id: "000571"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/6c18a209f836610c
            reviewed_head: 73a3a6a321de20044b092c8469be71b3a82b31a4
            slot: pair:7
            requested_at: "2026-10-15T23:35:36.173076Z"
          phase: finished
          pid: 77446
          started_at: "2026-10-15T23:35:36.382206Z"
          finished_at: "2026-10-15T23:39:17.961252Z"
          session_id: 01a141eb-d423-7cd1-a193-69bff2fb0560
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 73a3a6a321de20044b092c8469be71b3a82b31a4
        tracker_head: 3db8d96fcda44c54d8c9178b76e4a8f5f1cca929
        branch_point: 8792d2b7ee59e86a12bb8471441efe24d963f818
        since: "2026-10-15T10:22:13-07:00"
        until: "2026-10-15T16:35:03-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:22:13-07:00"
              end: "2026-10-15T16:35:03-07:00"
              began: "2026-10-15T10:22:13-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: e057324fe021ed4e8e0dee00a1447ceadfc8d970fa7991cf5ee590351a0de370
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 69514778
              windows:
                - start: "2026-10-15T10:22:13-07:00"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a14096-6170-7e02-ae66-5b9fb3e7b182
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: d480790ffe797d9fe00c5d49e6b4ac95d8c1ba1dbd8ae7c5a9b7ce08f642545f
              started: "2026-10-15T17:22:39.652Z"
              spawned: true
              bytes: 1391734
              windows:
                - start: "2026-10-15T17:22:39.652Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a14097-c211-7ce0-b051-366dd9fff257
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b1dcb2a70e7979533c3a75a0dfdf5c14ee69dd11770bb8bbd8477f1fe3d31dd5
              started: "2026-10-15T17:24:09.928Z"
              spawned: true
              bytes: 2979299
              windows:
                - start: "2026-10-15T17:24:09.928Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a14177-aa85-7cb3-bedb-b2a85e52af11
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43bbb8baac5acc685a555545857159490869cd449d01fb7111da69c67b3b9dc5
              started: "2026-10-15T21:28:47.768Z"
              spawned: false
              bytes: 667373
              windows:
                - start: "2026-10-15T21:28:47.768Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a1417a-a6e2-7912-8b67-8458dfe9c87c
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6e13a1d39624297d15c0f7a0388e25cd74a6176b9aaa00d492195956f70113e5
              started: "2026-10-15T21:32:00.527Z"
              spawned: false
              bytes: 550567
              windows:
                - start: "2026-10-15T21:32:00.527Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a1417c-1628-7022-a694-967d63271267
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 907486ef0551b775f583f9c072d40e7d8c85f2dd011c2822fd58257375430eaf
              started: "2026-10-15T21:33:35.501Z"
              spawned: false
              bytes: 535544
              windows:
                - start: "2026-10-15T21:33:35.501Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a1417c-c082-7122-a0fa-e0e3da0cc318
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 794442b6bc357a71c594717d55a25d15f708bcd42b0d8343224557b0c36d6d2f
              started: "2026-10-15T21:34:18.189Z"
              spawned: false
              bytes: 413877
              windows:
                - start: "2026-10-15T21:34:18.189Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a1417d-adca-7cd1-a945-d5965c8d2446
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e4278ae4a2ae92b16ab5a1619bd73308bbec7ec2bea2fedf8570157d1dcda67f
              started: "2026-10-15T21:35:18.067Z"
              spawned: true
              bytes: 1748698
              windows:
                - start: "2026-10-15T21:35:18.067Z"
                  end: "2026-10-15T16:35:03-07:00"
            - harness: codex
              id: 01a1417e-6595-70e1-9dbc-1eedb485d020
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 59fc21aa2ccdf704be3146d659f5f8c8b581b22b7e89175c738fcf0935075c8b
              started: "2026-10-15T21:36:05.122Z"
              spawned: true
              bytes: 1615469
              windows:
                - start: "2026-10-15T21:36:05.122Z"
                  end: "2026-10-15T16:35:03-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.064
      "n": 1
      timestamp: "2026-10-15T16:39:24-07:00"
      dispatched: "2026-10-15T16:35:33-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: CLI table overrides merge keys that native Codex replaces
          detail: cmd/internal/sessioninventory/codex_config_os.go:292 recursively merges each CLI override. Native apply_toml_override replaces the value at the final dotted-key segment, except for explicitly structured features. With API-only configuration and overrides features={worktrees=false} followed by features={shell_snapshot=false}, Pair retains worktrees=false while native restores the default true. A newer conversation in a corresponding linked checkout is therefore excluded by Pair, allowing it to pin an older current-checkout conversation. Implement native leaf-assignment semantics separately from config-layer merging, sweep other table-valued overrides, and add a regression asserting both projected scope and selected ID.
          family: native-config-override-fidelity
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Exec selection trusts transcript records rejected by native decoding
          detail: cmd/internal/sessioninventory/codex_latest.go:330-343 treats any JSON object containing type=turn_context and payload.cwd as authoritative. Native exec only uses successfully decoded RolloutLine/TurnContextItem records, whose required fields include approval_policy, sandbox_policy, and model. A newer transcript with DB cwd elsewhere and a schema-invalid turn_context naming the current cwd is selected by Pair, while native ignores that record and can select an older conversation. TestCodexLatestTUIAndExecHaveDifferentCWDSources currently asserts this invalid record is authoritative. Validate native record eligibility or retain compatibility when it cannot be established; include validateCodexLatestFirst in the same sweep because its reused inventory validator also accepts incomplete native session_meta records. Replace positive fixtures with native-valid records and add malformed-schema regressions.
          family: native-transcript-schema-fidelity
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#571 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T16:39:24-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `native-config-override-fidelity` (kind: defect) CLI table overrides merge keys that native Codex replaces
  cmd/internal/sessioninventory/codex_config_os.go:292 recursively merges each CLI override. Native apply_toml_override replaces the value at the final dotted-key segment, except for explicitly structured features. With API-only configuration and overrides features={worktrees=false} followed by features={shell_snapshot=false}, Pair retains worktrees=false while native restores the default true. A newer conversation in a corresponding linked checkout is therefore excluded by Pair, allowing it to pin an older current-checkout conversation. Implement native leaf-assignment semantics separately from config-layer merging, sweep other table-valued overrides, and add a regression asserting both projected scope and selected ID.
- **BR-2** [Critical] `native-transcript-schema-fidelity` (kind: defect) Exec selection trusts transcript records rejected by native decoding
  cmd/internal/sessioninventory/codex_latest.go:330-343 treats any JSON object containing type=turn_context and payload.cwd as authoritative. Native exec only uses successfully decoded RolloutLine/TurnContextItem records, whose required fields include approval_policy, sandbox_policy, and model. A newer transcript with DB cwd elsewhere and a schema-invalid turn_context naming the current cwd is selected by Pair, while native ignores that record and can select an older conversation. TestCodexLatestTUIAndExecHaveDifferentCWDSources currently asserts this invalid record is authoritative. Validate native record eligibility or retain compatibility when it cannot be established; include validateCodexLatestFirst in the same sweep because its reused inventory validator also accepts incomplete native session_meta records. Replace positive fixtures with native-valid records and add malformed-schema regressions.

## Open findings

- **BR-1** [Critical] `native-config-override-fidelity` (kind: defect) CLI table overrides merge keys that native Codex replaces
- **BR-2** [Critical] `native-transcript-schema-fidelity` (kind: defect) Exec selection trusts transcript records rejected by native decoding
