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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 96ffc194-c1b3-4a55-8f0e-235a5498f95c
            repository: github.com/xianxu/pair
            issue_id: "000571"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a5eabb22d0c795a8
            reviewed_head: 7611621eff5cedd7df3160d62a6fa8b8f93b326b
            slot: pair:7
            requested_at: "2026-10-16T00:33:14.646291Z"
          phase: finished
          pid: 8625
          started_at: "2026-10-16T00:33:14.753069Z"
          finished_at: "2026-10-16T00:37:27.201852Z"
          session_id: 01a14220-98ec-7001-9b98-3951b8611aee
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 7611621eff5cedd7df3160d62a6fa8b8f93b326b
        tracker_head: 6f59b1d966560e05c0bb3887995ce3fb23c436e5
        branch_point: 99e11ecb8c72b7e5325a58692c1efd679e50710c
        since: "2026-10-15T10:22:13-07:00"
        until: "2026-10-15T17:32:46-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:22:13-07:00"
              end: "2026-10-15T17:32:46-07:00"
              began: "2026-10-15T10:22:13-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 02e6e0b26c1279669099bae7955d13ccb1ebd674736aaf5a19bc182ae96a873f
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 71785258
              windows:
                - start: "2026-10-15T10:22:13-07:00"
                  end: "2026-10-15T17:32:46-07:00"
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
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a14097-c211-7ce0-b051-366dd9fff257
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 63874921497c8b76dd0f883cf9682ea5294a235a1a3f124a4d3ab1354550e800
              started: "2026-10-15T17:24:09.928Z"
              spawned: true
              bytes: 3289274
              windows:
                - start: "2026-10-15T17:24:09.928Z"
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a14177-aa85-7cb3-bedb-b2a85e52af11
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43bbb8baac5acc685a555545857159490869cd449d01fb7111da69c67b3b9dc5
              started: "2026-10-15T21:28:47.768Z"
              spawned: false
              bytes: 667373
              windows:
                - start: "2026-10-15T21:28:47.768Z"
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a1417a-a6e2-7912-8b67-8458dfe9c87c
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6e13a1d39624297d15c0f7a0388e25cd74a6176b9aaa00d492195956f70113e5
              started: "2026-10-15T21:32:00.527Z"
              spawned: false
              bytes: 550567
              windows:
                - start: "2026-10-15T21:32:00.527Z"
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a1417c-1628-7022-a694-967d63271267
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 907486ef0551b775f583f9c072d40e7d8c85f2dd011c2822fd58257375430eaf
              started: "2026-10-15T21:33:35.501Z"
              spawned: false
              bytes: 535544
              windows:
                - start: "2026-10-15T21:33:35.501Z"
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a1417c-c082-7122-a0fa-e0e3da0cc318
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 794442b6bc357a71c594717d55a25d15f708bcd42b0d8343224557b0c36d6d2f
              started: "2026-10-15T21:34:18.189Z"
              spawned: false
              bytes: 413877
              windows:
                - start: "2026-10-15T21:34:18.189Z"
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a1417d-adca-7cd1-a945-d5965c8d2446
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 95af9c7ca3afea5eb6f56b04b944f2c9c6b2986bcc94eef43b7a6a7252f85a13
              started: "2026-10-15T21:35:18.067Z"
              spawned: true
              bytes: 2308333
              windows:
                - start: "2026-10-15T21:35:18.067Z"
                  end: "2026-10-15T17:32:46-07:00"
            - harness: codex
              id: 01a1417e-6595-70e1-9dbc-1eedb485d020
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 6a700a1e1bb3950ef80823813a995e3e758d3bd264e31b9d0100f3f045e2aa2a
              started: "2026-10-15T21:36:05.122Z"
              spawned: true
              bytes: 1951703
              windows:
                - start: "2026-10-15T21:36:05.122Z"
                  end: "2026-10-15T17:32:46-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.064
        cuts:
            - dispatched: "2026-10-15T16:35:33-07:00"
              finalized: "2026-10-15T16:39:24-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-15T17:37:50-07:00"
      dispatched: "2026-10-15T17:33:13-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Native leaf replacement is implemented separately from layer merging. The linked-checkout regression checks projected scope and selected ID; a temporary overlay restoring recursive table merging makes it fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Both session metadata and turn-context authority now require native eligibility. Positive fixtures contain required native fields. Removing the eligibility guards through a temporary overlay makes the malformed-schema regressions select the invalid newer ID and fail.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Critical
          title: Trust-specific alias rejection silently narrows native worktree selection
          detail: 'cmd/internal/sessioninventory/codex_git_scope.go:63 rejects symlinked admin targets for both trust and worktree discovery, and appendRoot at line 148 silently excludes that checkout. Native worktree repository_identity canonicalizes these targets, although native trust discovery rejects them. With API-only configuration, an older current-checkout conversation and a newer linked-checkout conversation whose .git points through an admin alias, a private production-selector regression resolves the older ID instead of the newer native match. ARCH-PURPOSE / ARCH-SECURE: separate trust eligibility from worktree discovery semantics, and sweep current/primary/linked discovery so unsupported observations yield compatibility rather than a smaller authoritative scope.'
          family: native-worktree-scope-fidelity
          round: 2
        - kind: defect
          id: BR-4
          severity: Critical
          title: Read-only latest selection creates native SQLite shared-memory files
          detail: 'cmd/internal/sessioninventory/codex_latest.go:103 queries the live native database through QueryNativeSelectorSQLite, whose sqlite3 -readonly invocation at native_selector_os.go:78 still creates WAL shared-memory sidecars. A private WAL fixture containing state_5.sqlite and state_5.sqlite-wal but no SHM returned a resolved ID and acquired a new 32768-byte state_5.sqlite-shm. This violates the issue''s no-native-mutation contract. ARCH-PURPOSE / ARCH-MOCK / ARCH-FUNERAL: use a bounded, coherent private snapshot or another genuinely non-mutating observation mechanism, retain compatibility when safe observation is unavailable, and add WAL-backed no-mutation regression coverage.'
          family: native-read-only-observation
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1012d4e3-b66e-43ed-a040-b9b8fa5e21a1
            repository: github.com/xianxu/pair
            issue_id: "000571"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/2d3f1ba50376b04a
            reviewed_head: 2084334f6e1ecef502f755df75507d0443373189
            slot: pair:7
            requested_at: "2026-10-16T01:42:49.985484Z"
          phase: finished
          pid: 13326
          started_at: "2026-10-16T01:42:50.079819Z"
          finished_at: "2026-10-16T01:46:52.36869Z"
          session_id: 01a14260-4ebf-73a3-9ab9-b47c486ecece
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2084334f6e1ecef502f755df75507d0443373189
        tracker_head: c91a739979ba6239958bf2a9411c02c45b54a138
        branch_point: 4c6fcc8d437ce52c708b0a6bdef09ce42b86917b
        since: "2026-10-15T10:22:13-07:00"
        until: "2026-10-15T18:42:23-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:22:13-07:00"
              end: "2026-10-15T18:42:23-07:00"
              began: "2026-10-15T10:22:13-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 0fb5550ab97d5be02663ddc5b6e2f40478222fae9c170b5f10a8c504afbe0fad
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 75166887
              windows:
                - start: "2026-10-15T10:22:13-07:00"
                  end: "2026-10-15T18:42:23-07:00"
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
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a14097-c211-7ce0-b051-366dd9fff257
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 8ee1e389d29207757e676b4fbe995ad12a40da006a9ec08cffb0b504420c443b
              started: "2026-10-15T17:24:09.928Z"
              spawned: true
              bytes: 3966706
              windows:
                - start: "2026-10-15T17:24:09.928Z"
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a14177-aa85-7cb3-bedb-b2a85e52af11
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43bbb8baac5acc685a555545857159490869cd449d01fb7111da69c67b3b9dc5
              started: "2026-10-15T21:28:47.768Z"
              spawned: false
              bytes: 667373
              windows:
                - start: "2026-10-15T21:28:47.768Z"
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a1417a-a6e2-7912-8b67-8458dfe9c87c
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6e13a1d39624297d15c0f7a0388e25cd74a6176b9aaa00d492195956f70113e5
              started: "2026-10-15T21:32:00.527Z"
              spawned: false
              bytes: 550567
              windows:
                - start: "2026-10-15T21:32:00.527Z"
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a1417c-1628-7022-a694-967d63271267
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 907486ef0551b775f583f9c072d40e7d8c85f2dd011c2822fd58257375430eaf
              started: "2026-10-15T21:33:35.501Z"
              spawned: false
              bytes: 535544
              windows:
                - start: "2026-10-15T21:33:35.501Z"
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a1417c-c082-7122-a0fa-e0e3da0cc318
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 794442b6bc357a71c594717d55a25d15f708bcd42b0d8343224557b0c36d6d2f
              started: "2026-10-15T21:34:18.189Z"
              spawned: false
              bytes: 413877
              windows:
                - start: "2026-10-15T21:34:18.189Z"
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a1417d-adca-7cd1-a945-d5965c8d2446
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: d0caa4d080a01390e98c2b84e6e1785d7cdc689dad9ff1a29c4335c0f1ac0713
              started: "2026-10-15T21:35:18.067Z"
              spawned: true
              bytes: 4019461
              windows:
                - start: "2026-10-15T21:35:18.067Z"
                  end: "2026-10-15T18:42:23-07:00"
            - harness: codex
              id: 01a1417e-6595-70e1-9dbc-1eedb485d020
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b1ed1977413ef1ffe9d4ea6c8d89c95c68414638654ac30521eaa510595914c3
              started: "2026-10-15T21:36:05.122Z"
              spawned: true
              bytes: 2769446
              windows:
                - start: "2026-10-15T21:36:05.122Z"
                  end: "2026-10-15T18:42:23-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.064
        cuts:
            - dispatched: "2026-10-15T16:35:33-07:00"
              finalized: "2026-10-15T16:39:24-07:00"
              finalizing: false
            - dispatched: "2026-10-15T17:33:13-07:00"
              finalized: "2026-10-15T17:37:50-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-15T18:46:59-07:00"
      dispatched: "2026-10-15T18:42:48-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Leaf-replacement logic and its projection/selected-ID regressions remain present; the targeted configuration tests passed.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Native record eligibility checks remain reachable before selection authority; malformed metadata and turn-context regressions passed. The new finding concerns filtering valid records, not decoding invalid records.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Current/primary/linked discovery uncertainty now yields whole-projection compatibility. Restoring the pre-fix Git implementation through a temporary Go overlay made TestCodexLatestGitAdminAliasCannotSelectOlderConversation fail by selecting the older ID; pinned-head tests passed.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Production selection uses an owned private DB/WAL snapshot. Bypassing snapshot dispatch through temporary overlays made both TUI and exec WAL regressions fail with a newly created 32768-byte native SHM file; pinned-head non-mutation, cleanup and race tests passed.
          round: 3
      findings:
        - kind: defect
          id: BR-5
          severity: Critical
          title: Exec latest can pin Custom or Internal conversations excluded by native selection
          detail: 'cmd/internal/sessioninventory/codex_latest.go:181-190 filters only by provider and never applies native source-kind filtering. In rust-v0.162.1, exec all_thread_source_kinds feeds app-server source_kind_matches, which excludes SessionSource::Custom and SessionSource::Internal. A temporary production-selector regression with an older eligible cli conversation and a newer matching conversation reproduced wrong-ID resolution for custom atlas, custom chatgpt, internal guardian and internal memory_consolidation sources. ARCH-PURPOSE / ARCH-SECURE: decode and filter persisted DB source values according to the native selection rule, or retain compatibility when their meaning is unproven. Sweep the source-kind mapping across both collectors and add selected-ID regressions; valid transcript decoding alone does not establish selector eligibility.'
          family: native-selector-filter-fidelity
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7ce7e0a5-aaf8-4265-a71c-cf42d766388a
            repository: github.com/xianxu/pair
            issue_id: "000571"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/d079df4c329f5341
            reviewed_head: 1e3129bcbe93bb7aaf8d27b6b33785f5757ca7f3
            slot: pair:7
            requested_at: "2026-10-16T02:28:35.857691Z"
          phase: finished
          pid: 2099
          started_at: "2026-10-16T02:28:35.949395Z"
          finished_at: "2026-10-16T02:32:49.087685Z"
          session_id: 01a1428a-34c2-7f83-a48d-5fd7444e9bdb
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1e3129bcbe93bb7aaf8d27b6b33785f5757ca7f3
        tracker_head: 7a7cdeee23fe98f0fdeda9acc2f48cda1043445c
        branch_point: 0169e962f95a932fd9884b20b2d5dd07b73022f1
        since: "2026-10-15T10:22:13-07:00"
        until: "2026-10-15T19:28:20-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:22:13-07:00"
              end: "2026-10-15T19:28:20-07:00"
              began: "2026-10-15T10:22:13-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 4d8eb79287c8603832233344f903ad57936377c674ac862fa95043cf43288dc8
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 78016902
              windows:
                - start: "2026-10-15T10:22:13-07:00"
                  end: "2026-10-15T19:28:20-07:00"
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
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a14097-c211-7ce0-b051-366dd9fff257
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 2ea842eea9446200addfb65af85b1f8a6d3e742ab26bed90ad523fa5eedd0f28
              started: "2026-10-15T17:24:09.928Z"
              spawned: true
              bytes: 4723189
              windows:
                - start: "2026-10-15T17:24:09.928Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a14177-aa85-7cb3-bedb-b2a85e52af11
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43bbb8baac5acc685a555545857159490869cd449d01fb7111da69c67b3b9dc5
              started: "2026-10-15T21:28:47.768Z"
              spawned: false
              bytes: 667373
              windows:
                - start: "2026-10-15T21:28:47.768Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a1417a-a6e2-7912-8b67-8458dfe9c87c
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6e13a1d39624297d15c0f7a0388e25cd74a6176b9aaa00d492195956f70113e5
              started: "2026-10-15T21:32:00.527Z"
              spawned: false
              bytes: 550567
              windows:
                - start: "2026-10-15T21:32:00.527Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a1417c-1628-7022-a694-967d63271267
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 907486ef0551b775f583f9c072d40e7d8c85f2dd011c2822fd58257375430eaf
              started: "2026-10-15T21:33:35.501Z"
              spawned: false
              bytes: 535544
              windows:
                - start: "2026-10-15T21:33:35.501Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a1417c-c082-7122-a0fa-e0e3da0cc318
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 794442b6bc357a71c594717d55a25d15f708bcd42b0d8343224557b0c36d6d2f
              started: "2026-10-15T21:34:18.189Z"
              spawned: false
              bytes: 413877
              windows:
                - start: "2026-10-15T21:34:18.189Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a1417d-adca-7cd1-a945-d5965c8d2446
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 2ee67c0bafbd34deed4a036e45da542678d185b2f4640ba440fddc04977dd17a
              started: "2026-10-15T21:35:18.067Z"
              spawned: true
              bytes: 5091628
              windows:
                - start: "2026-10-15T21:35:18.067Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a1417e-6595-70e1-9dbc-1eedb485d020
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 1b5c9f426c219cd8746e9e83eebf452e7a5319e9e9d825a0ae6ef4269f14aaae
              started: "2026-10-15T21:36:05.122Z"
              spawned: true
              bytes: 3644516
              windows:
                - start: "2026-10-15T21:36:05.122Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a14276-5f69-78a1-ba4b-f58121438b6a
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: dce0b10c629da8d3a28aeeed669e0e0bb919ff129e9f2f54d82a659fbb30adc6
              started: "2026-10-16T02:06:56.581Z"
              spawned: false
              bytes: 551795
              windows:
                - start: "2026-10-16T02:06:56.581Z"
                  end: "2026-10-15T19:28:20-07:00"
            - harness: codex
              id: 01a14276-ecef-7310-a963-13ecb96db6e6
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 1848ebd5f6e6e1b16c6279a5ac20562b4d36c2b4aaf2c967419e25d09fe5a8b0
              started: "2026-10-16T02:07:32.798Z"
              spawned: false
              bytes: 455821
              windows:
                - start: "2026-10-16T02:07:32.798Z"
                  end: "2026-10-15T19:28:20-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.064
        cuts:
            - dispatched: "2026-10-15T16:35:33-07:00"
              finalized: "2026-10-15T16:39:24-07:00"
              finalizing: false
            - dispatched: "2026-10-15T17:33:13-07:00"
              finalized: "2026-10-15T17:37:50-07:00"
              finalizing: false
            - dispatched: "2026-10-15T18:42:48-07:00"
              finalized: "2026-10-15T18:46:59-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-15T19:32:49-07:00"
      dispatched: "2026-10-15T19:28:34-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The configuration projection and override machinery are removed; indirect Codex invocations remain native.
          round: 4
        - id: BR-2
          disposition: addressed
          note: The native-selector transcript eligibility machinery is removed; Pair no longer predicts indirect Codex targets.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Codex worktree-scope emulation is removed; indirect requests return unsupported before scope observation.
          round: 4
        - id: BR-4
          disposition: addressed
          note: Codex SQLite collectors and snapshot machinery are removed; exact-ID admission uses transcript validation.
          round: 4
        - id: BR-5
          disposition: addressed
          note: Both Codex collectors are absent and indirect selection is fenced at pure and IO boundaries. Removing each guard independently made TestNativeSelectorCodexIndirectNeverProbesEvenWithResolvedConfiguration fail.
          round: 4
      findings:
        - kind: defect
          id: BR-6
          severity: Critical
          title: A Codex home error blocks unrelated agents sharing the ledger
          detail: 'cmd/internal/launcher/osruntime.go:705-715 returns a whole-ledger error when any Codex owner has an unavailable home. With bound Codex and Claude entries in one tag and CODEX_HOME naming a missing directory, a temporary regression calling readSavedConfigForTag for Claude failed with "cannot read native session authority ... Codex native home unavailable". Valid relative CODEX_HOME also cannot resolve here because startup cwd is deliberately unavailable. ARCH-PURPOSE: preserve per-agent availability; represent the Codex target as unavailable without discarding usable Claude/Qoder entries, and add mixed-agent ledger launch/restart regressions.'
          family: native-agent-error-isolation
          round: 4
        - kind: defect
          id: BR-7
          severity: Critical
          title: Recovery discovery and validation use different Codex transcript homes
          detail: 'cmd/internal/launcher/recovery_authority_os.go:167-171 now validates candidates under CODEX_HOME, but cmd/internal/couchcore/recovery_evidence_os.go:199-200 still discovers them through NewOSRuntime(HomeDir, DataDir), which scans HOME/.codex. A temporary production-reader regression found one valid candidate in the default home, then found zero after moving the same tree to an absolute CODEX_HOME. This contradicts the active plan''s exact/ledger/recovery consistency requirement. ARCH-PURPOSE / ARCH-DRY: share transcript namespace construction across discovery and validation, sweep remaining recovery root constructors, and add an end-to-end custom-home recovery regression.'
          family: native-transcript-namespace-consistency
          round: 4
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d4fe7e93-13da-43e6-be5d-ef0d33aa5acc
            repository: github.com/xianxu/pair
            issue_id: "000571"
            recipe: milestone-review
            round: 5
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9285354d67c930ab
            reviewed_head: 701270af8026d19ea079bca8d86b45bf526e03e2
            slot: pair:7
            requested_at: "2026-10-16T03:23:35.684432Z"
          phase: finished
          pid: 53116
          started_at: "2026-10-16T03:23:35.790001Z"
          finished_at: "2026-10-16T03:28:59.988633Z"
          session_id: 01a142bc-8ef1-7e01-841e-58103e6c4cfd
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 701270af8026d19ea079bca8d86b45bf526e03e2
        tracker_head: 3f43c23ac3fc49d6a7a0dffd3a6902ada551a003
        branch_point: 5414ef47b64388d7a7db9db8448f3a836bc62f6a
        since: "2026-10-15T10:22:13-07:00"
        until: "2026-10-15T20:23:15-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:22:13-07:00"
              end: "2026-10-15T20:23:15-07:00"
              began: "2026-10-15T10:22:13-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: bca0b023f3627b802880cb87aad3b04e2c4f81888967053d9ee093dbdac1e446
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 79533189
              windows:
                - start: "2026-10-15T10:22:13-07:00"
                  end: "2026-10-15T20:23:15-07:00"
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
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a14097-c211-7ce0-b051-366dd9fff257
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 4435e4596e541bb09a95f5356393835df1becd7b095b15efe38b51bf5b3ee7d4
              started: "2026-10-15T17:24:09.928Z"
              spawned: true
              bytes: 5440527
              windows:
                - start: "2026-10-15T17:24:09.928Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a14177-aa85-7cb3-bedb-b2a85e52af11
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 43bbb8baac5acc685a555545857159490869cd449d01fb7111da69c67b3b9dc5
              started: "2026-10-15T21:28:47.768Z"
              spawned: false
              bytes: 667373
              windows:
                - start: "2026-10-15T21:28:47.768Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a1417a-a6e2-7912-8b67-8458dfe9c87c
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6e13a1d39624297d15c0f7a0388e25cd74a6176b9aaa00d492195956f70113e5
              started: "2026-10-15T21:32:00.527Z"
              spawned: false
              bytes: 550567
              windows:
                - start: "2026-10-15T21:32:00.527Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a1417c-1628-7022-a694-967d63271267
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 907486ef0551b775f583f9c072d40e7d8c85f2dd011c2822fd58257375430eaf
              started: "2026-10-15T21:33:35.501Z"
              spawned: false
              bytes: 535544
              windows:
                - start: "2026-10-15T21:33:35.501Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a1417c-c082-7122-a0fa-e0e3da0cc318
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 794442b6bc357a71c594717d55a25d15f708bcd42b0d8343224557b0c36d6d2f
              started: "2026-10-15T21:34:18.189Z"
              spawned: false
              bytes: 413877
              windows:
                - start: "2026-10-15T21:34:18.189Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a1417d-adca-7cd1-a945-d5965c8d2446
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 2ee67c0bafbd34deed4a036e45da542678d185b2f4640ba440fddc04977dd17a
              started: "2026-10-15T21:35:18.067Z"
              spawned: true
              bytes: 5091628
              windows:
                - start: "2026-10-15T21:35:18.067Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a1417e-6595-70e1-9dbc-1eedb485d020
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 087ba968c9bd5212dcaf2e8362592ddffb0090109b8647ef014fe88ea1de98a7
              started: "2026-10-15T21:36:05.122Z"
              spawned: true
              bytes: 4206118
              windows:
                - start: "2026-10-15T21:36:05.122Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a14276-5f69-78a1-ba4b-f58121438b6a
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: dce0b10c629da8d3a28aeeed669e0e0bb919ff129e9f2f54d82a659fbb30adc6
              started: "2026-10-16T02:06:56.581Z"
              spawned: false
              bytes: 551795
              windows:
                - start: "2026-10-16T02:06:56.581Z"
                  end: "2026-10-15T20:23:15-07:00"
            - harness: codex
              id: 01a14276-ecef-7310-a963-13ecb96db6e6
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 1848ebd5f6e6e1b16c6279a5ac20562b4d36c2b4aaf2c967419e25d09fe5a8b0
              started: "2026-10-16T02:07:32.798Z"
              spawned: false
              bytes: 455821
              windows:
                - start: "2026-10-16T02:07:32.798Z"
                  end: "2026-10-15T20:23:15-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.064
        cuts:
            - dispatched: "2026-10-15T16:35:33-07:00"
              finalized: "2026-10-15T16:39:24-07:00"
              finalizing: false
            - dispatched: "2026-10-15T17:33:13-07:00"
              finalized: "2026-10-15T17:37:50-07:00"
              finalizing: false
            - dispatched: "2026-10-15T18:42:48-07:00"
              finalized: "2026-10-15T18:46:59-07:00"
              finalizing: false
            - dispatched: "2026-10-15T19:28:34-07:00"
              finalized: "2026-10-15T19:32:49-07:00"
              finalizing: false
      "n": 5
      timestamp: "2026-10-15T20:29:02-07:00"
      dispatched: "2026-10-15T20:23:34-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Remains disposed; the superseding native-pass-through design removes Codex configuration projection.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Remains disposed; Codex indirect transcript selection is removed.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Remains disposed; native worktree selection is delegated to Codex.
          round: 5
        - id: BR-4
          disposition: addressed
          note: Remains disposed; the Codex SQLite collector is removed and indirect requests return before IO.
          round: 5
        - id: BR-5
          disposition: addressed
          note: Remains disposed; Pair no longer applies Codex indirect source filters.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: Mixed typed ledgers now isolate unavailable Codex homes correctly, and reverting the fix makes the committed Claude/Qoder regressions fail. However, osruntime.go:705-720 populates unavailable only through parseLedger's typed-owner callback. A compatibility-only Codex ledger never invokes it. A disposable-head regression with invalid CODEX_HOME and saved configuration returned nil error, emitted “starting fresh,” and deleted config-work-codex.json. Validate namespace availability for every represented agent, including compatibility-only rows, and add saved-config/restart regressions for that case.
          round: 5
        - id: BR-7
          disposition: addressed
          note: Recovery discovery now uses NewNativeTranscriptRuntime, matching validation and legacy held-file checks. Targeted normal/race tests pass; restoring the former discovery constructor makes the committed custom/invalid/relative-home cases fail.
          round: 5
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

## Round 2 — 2026-10-15T17:37:50-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Native leaf replacement is implemented separately from layer merging. The linked-checkout regression checks projected scope and selected ID; a temporary overlay restoring recursive table merging makes it fail.
- BR-2 — addressed — Both session metadata and turn-context authority now require native eligibility. Positive fixtures contain required native fields. Removing the eligibility guards through a temporary overlay makes the malformed-schema regressions select the invalid newer ID and fail.

### Raised

- **BR-3** [Critical] `native-worktree-scope-fidelity` (kind: defect) Trust-specific alias rejection silently narrows native worktree selection
  cmd/internal/sessioninventory/codex_git_scope.go:63 rejects symlinked admin targets for both trust and worktree discovery, and appendRoot at line 148 silently excludes that checkout. Native worktree repository_identity canonicalizes these targets, although native trust discovery rejects them. With API-only configuration, an older current-checkout conversation and a newer linked-checkout conversation whose .git points through an admin alias, a private production-selector regression resolves the older ID instead of the newer native match. ARCH-PURPOSE / ARCH-SECURE: separate trust eligibility from worktree discovery semantics, and sweep current/primary/linked discovery so unsupported observations yield compatibility rather than a smaller authoritative scope.
- **BR-4** [Critical] `native-read-only-observation` (kind: defect) Read-only latest selection creates native SQLite shared-memory files
  cmd/internal/sessioninventory/codex_latest.go:103 queries the live native database through QueryNativeSelectorSQLite, whose sqlite3 -readonly invocation at native_selector_os.go:78 still creates WAL shared-memory sidecars. A private WAL fixture containing state_5.sqlite and state_5.sqlite-wal but no SHM returned a resolved ID and acquired a new 32768-byte state_5.sqlite-shm. This violates the issue's no-native-mutation contract. ARCH-PURPOSE / ARCH-MOCK / ARCH-FUNERAL: use a bounded, coherent private snapshot or another genuinely non-mutating observation mechanism, retain compatibility when safe observation is unavailable, and add WAL-backed no-mutation regression coverage.

## Round 3 — 2026-10-15T18:46:59-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Leaf-replacement logic and its projection/selected-ID regressions remain present; the targeted configuration tests passed.
- BR-2 — addressed — Native record eligibility checks remain reachable before selection authority; malformed metadata and turn-context regressions passed. The new finding concerns filtering valid records, not decoding invalid records.
- BR-3 — addressed — Current/primary/linked discovery uncertainty now yields whole-projection compatibility. Restoring the pre-fix Git implementation through a temporary Go overlay made TestCodexLatestGitAdminAliasCannotSelectOlderConversation fail by selecting the older ID; pinned-head tests passed.
- BR-4 — addressed — Production selection uses an owned private DB/WAL snapshot. Bypassing snapshot dispatch through temporary overlays made both TUI and exec WAL regressions fail with a newly created 32768-byte native SHM file; pinned-head non-mutation, cleanup and race tests passed.

### Raised

- **BR-5** [Critical] `native-selector-filter-fidelity` (kind: defect) Exec latest can pin Custom or Internal conversations excluded by native selection
  cmd/internal/sessioninventory/codex_latest.go:181-190 filters only by provider and never applies native source-kind filtering. In rust-v0.162.1, exec all_thread_source_kinds feeds app-server source_kind_matches, which excludes SessionSource::Custom and SessionSource::Internal. A temporary production-selector regression with an older eligible cli conversation and a newer matching conversation reproduced wrong-ID resolution for custom atlas, custom chatgpt, internal guardian and internal memory_consolidation sources. ARCH-PURPOSE / ARCH-SECURE: decode and filter persisted DB source values according to the native selection rule, or retain compatibility when their meaning is unproven. Sweep the source-kind mapping across both collectors and add selected-ID regressions; valid transcript decoding alone does not establish selector eligibility.

## Round 4 — 2026-10-15T19:32:49-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The configuration projection and override machinery are removed; indirect Codex invocations remain native.
- BR-2 — addressed — The native-selector transcript eligibility machinery is removed; Pair no longer predicts indirect Codex targets.
- BR-3 — addressed — Codex worktree-scope emulation is removed; indirect requests return unsupported before scope observation.
- BR-4 — addressed — Codex SQLite collectors and snapshot machinery are removed; exact-ID admission uses transcript validation.
- BR-5 — addressed — Both Codex collectors are absent and indirect selection is fenced at pure and IO boundaries. Removing each guard independently made TestNativeSelectorCodexIndirectNeverProbesEvenWithResolvedConfiguration fail.

### Raised

- **BR-6** [Critical] `native-agent-error-isolation` (kind: defect) A Codex home error blocks unrelated agents sharing the ledger
  cmd/internal/launcher/osruntime.go:705-715 returns a whole-ledger error when any Codex owner has an unavailable home. With bound Codex and Claude entries in one tag and CODEX_HOME naming a missing directory, a temporary regression calling readSavedConfigForTag for Claude failed with "cannot read native session authority ... Codex native home unavailable". Valid relative CODEX_HOME also cannot resolve here because startup cwd is deliberately unavailable. ARCH-PURPOSE: preserve per-agent availability; represent the Codex target as unavailable without discarding usable Claude/Qoder entries, and add mixed-agent ledger launch/restart regressions.
- **BR-7** [Critical] `native-transcript-namespace-consistency` (kind: defect) Recovery discovery and validation use different Codex transcript homes
  cmd/internal/launcher/recovery_authority_os.go:167-171 now validates candidates under CODEX_HOME, but cmd/internal/couchcore/recovery_evidence_os.go:199-200 still discovers them through NewOSRuntime(HomeDir, DataDir), which scans HOME/.codex. A temporary production-reader regression found one valid candidate in the default home, then found zero after moving the same tree to an absolute CODEX_HOME. This contradicts the active plan's exact/ledger/recovery consistency requirement. ARCH-PURPOSE / ARCH-DRY: share transcript namespace construction across discovery and validation, sweep remaining recovery root constructors, and add an end-to-end custom-home recovery regression.

## Round 5 — 2026-10-15T20:29:02-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Remains disposed; the superseding native-pass-through design removes Codex configuration projection.
- BR-2 — addressed — Remains disposed; Codex indirect transcript selection is removed.
- BR-3 — addressed — Remains disposed; native worktree selection is delegated to Codex.
- BR-4 — addressed — Remains disposed; the Codex SQLite collector is removed and indirect requests return before IO.
- BR-5 — addressed — Remains disposed; Pair no longer applies Codex indirect source filters.
- BR-6 — not-addressed — Mixed typed ledgers now isolate unavailable Codex homes correctly, and reverting the fix makes the committed Claude/Qoder regressions fail. However, osruntime.go:705-720 populates unavailable only through parseLedger's typed-owner callback. A compatibility-only Codex ledger never invokes it. A disposable-head regression with invalid CODEX_HOME and saved configuration returned nil error, emitted “starting fresh,” and deleted config-work-codex.json. Validate namespace availability for every represented agent, including compatibility-only rows, and add saved-config/restart regressions for that case.
- BR-7 — addressed — Recovery discovery now uses NewNativeTranscriptRuntime, matching validation and legacy held-file checks. Targeted normal/race tests pass; restoring the former discovery constructor makes the committed custom/invalid/relative-home cases fail.

## Open findings

- **BR-6** [Critical] `native-agent-error-isolation` (kind: defect) A Codex home error blocks unrelated agents sharing the ledger
