---
gate: boundary-review
issue: 659
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 90a121e7-4277-4c77-8846-742c2097d6b4
            repository: github.com/xianxu/pair
            issue_id: "000659"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/95545d166b3092e1
            reviewed_head: 1a1f729229e3539cd0dfca7524ae6d11d5631eb0
            slot: pair:7
            requested_at: "2026-10-18T06:55:28.420392Z"
          phase: finished
          pid: 45524
          started_at: "2026-10-18T06:55:28.638764Z"
          finished_at: "2026-10-18T06:57:07.005056Z"
          session_id: 01a14dcb-42ee-70f3-891a-e307fc9b79b7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 1a1f729229e3539cd0dfca7524ae6d11d5631eb0
        tracker_head: 4e0346c7576c767102c56b3104b9d19a2fa2a015
        branch_point: d6ae7e32b0e128f95f6ca3ae0c1541b79560ac40
        since: "2026-10-17T22:43:28-07:00"
        until: "2026-10-17T23:54:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T22:43:28-07:00"
              end: "2026-10-17T23:54:19-07:00"
              began: "2026-10-17T22:43:28-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 1dfa3b03fca31d679ee90ad56c784d12bcc01e882c94bd79c66493a8effbf129
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 123469860
              windows:
                - start: "2026-10-17T22:43:28-07:00"
                  end: "2026-10-17T23:54:19-07:00"
            - harness: codex
              id: 01a14d89-aac7-71c2-89c5-147a00fb9a67
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 3dd9d336bf8eeadd0f77ca7a8a3130bb6d8d37b0806cd32d6aac262313ab6f62
              started: "2026-10-18T05:43:50.274Z"
              spawned: true
              bytes: 1113788
              windows:
                - start: "2026-10-18T05:43:50.274Z"
                  end: "2026-10-17T23:54:19-07:00"
            - harness: codex
              id: 01a14d89-bf84-7141-a788-8d4cb4d5e4bd
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 78754a84bf387aea43d706edb0aadba627da864cd38c4296836ede968414d2a8
              started: "2026-10-18T05:43:55.585Z"
              spawned: true
              bytes: 937469
              windows:
                - start: "2026-10-18T05:43:55.585Z"
                  end: "2026-10-17T23:54:19-07:00"
            - harness: codex
              id: 01a14d8e-89bf-7b70-b42f-586388f343a7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 035112415c3286ae0449dbd57372a747248e2725d39e4bbadaa42a94e060d73c
              started: "2026-10-18T05:49:09.523Z"
              spawned: true
              bytes: 733773
              windows:
                - start: "2026-10-18T05:49:09.523Z"
                  end: "2026-10-17T23:54:19-07:00"
        queue_waits:
            - attempt_id: db25bcb0f104165571bc9f60bcde47e2
              slot: pair:7
              record_digest: 0a62583d3f8b4bcdf08de5b22146365dcb1065bec167fdbb8ab9ea6e0fa7a0a8
              start: "2026-10-18T05:54:24.588686Z"
              end: "2026-10-18T06:12:34.398198Z"
              queue_seconds: 1089.809512
              raw_seconds: 1089.809512
              exact: true
            - attempt_id: c5ff20b0a514c353698bd714be553db5
              slot: pair:7
              record_digest: 9002e18184514276dbb2114de01eee431f90a50b1769457a9cc7eee8c10e8731
              start: "2026-10-18T06:26:02.440022Z"
              end: "2026-10-18T06:37:53.957156Z"
              queue_seconds: 711.517134
              raw_seconds: 711.517134
              exact: true
      "n": 1
      timestamp: "2026-10-17T23:57:08-07:00"
      dispatched: "2026-10-17T23:55:26-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: 'ARCH-PURE: the probe verdict is classified PURE but depends on IO'
          detail: workshop/plans/000659-test-budgets-plan.md:27 labels the optional/required verdict PURE, but requireLsofObservation reads os.Getenv and can execute HostState (proof_test.go:145–154); its policy tests launch subprocesses (lines 167–201). This is the sole contradictory PURE row in this window. Promote it to INTEGRATION and record a plan Revisions entry, or extract a deterministic classifier taking explicit inputs and test that directly without IO.
          family: core-concept-purity-classification
          round: 1
        - id: BR-2
          severity: Important
          title: Done when retains the superseded load-acceptance contract
          detail: workshop/issues/000659-tests-wall-clock-budgets-under-load.md:104–111 still requires deliberate fleet-load repetitions and a full suite under that same load, while Revisions:194 accepts injected-delay proof supported by observed-load repetitions. The recorded repetition and suite cohorts have different loads. Both load-related checklist clauses belong to this family; update them to express the accepted evidence requirements without implying another load campaign is necessary.
          family: acceptance-checklist-contract-sync
          round: 1
        - id: BR-3
          severity: Important
          title: README update appears missing for required lsof conformance mode
          detail: cmd/internal/diagnosticlog/proof_test.go:151 introduces PAIR_REQUIRE_LSOF_CONFORMANCE=1, and atlas/couch.md:26 documents its invocation, but README.md is unchanged. This is the only new operator-set configuration surface in this window; both native tests consume it through the shared helper. Add the invocation and optional-skip versus required-failure behavior to README's testing guidance.
          family: readme-new-run-surface
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 68b12d5a-f24b-434d-b55f-c42690e9b276
            repository: github.com/xianxu/pair
            issue_id: "000659"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/6d3b5441507ed34f
            reviewed_head: 8f59f920b2064566f2da4baddf36d0f56cd6041f
            slot: pair:7
            requested_at: "2026-10-18T07:00:28.060288Z"
          phase: finished
          pid: 98361
          started_at: "2026-10-18T07:00:28.289018Z"
          finished_at: "2026-10-18T07:01:28.859056Z"
          session_id: 01a14dcf-d5c0-7fa2-8276-2289094a05d9
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 8f59f920b2064566f2da4baddf36d0f56cd6041f
        tracker_head: 4e0346c7576c767102c56b3104b9d19a2fa2a015
        branch_point: d6ae7e32b0e128f95f6ca3ae0c1541b79560ac40
        since: "2026-10-17T22:43:28-07:00"
        until: "2026-10-17T23:59:21-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T22:43:28-07:00"
              end: "2026-10-17T23:59:21-07:00"
              began: "2026-10-17T22:43:28-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a73883f148a504025729eb12d2beff4f68b6b174642ee7570837f67293a21d12
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 123781643
              windows:
                - start: "2026-10-17T22:43:28-07:00"
                  end: "2026-10-17T23:59:21-07:00"
            - harness: codex
              id: 01a14d89-aac7-71c2-89c5-147a00fb9a67
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 3dd9d336bf8eeadd0f77ca7a8a3130bb6d8d37b0806cd32d6aac262313ab6f62
              started: "2026-10-18T05:43:50.274Z"
              spawned: true
              bytes: 1113788
              windows:
                - start: "2026-10-18T05:43:50.274Z"
                  end: "2026-10-17T23:59:21-07:00"
            - harness: codex
              id: 01a14d89-bf84-7141-a788-8d4cb4d5e4bd
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 78754a84bf387aea43d706edb0aadba627da864cd38c4296836ede968414d2a8
              started: "2026-10-18T05:43:55.585Z"
              spawned: true
              bytes: 937469
              windows:
                - start: "2026-10-18T05:43:55.585Z"
                  end: "2026-10-17T23:59:21-07:00"
            - harness: codex
              id: 01a14d8e-89bf-7b70-b42f-586388f343a7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 035112415c3286ae0449dbd57372a747248e2725d39e4bbadaa42a94e060d73c
              started: "2026-10-18T05:49:09.523Z"
              spawned: true
              bytes: 733773
              windows:
                - start: "2026-10-18T05:49:09.523Z"
                  end: "2026-10-17T23:59:21-07:00"
        queue_waits:
            - attempt_id: db25bcb0f104165571bc9f60bcde47e2
              slot: pair:7
              record_digest: 0a62583d3f8b4bcdf08de5b22146365dcb1065bec167fdbb8ab9ea6e0fa7a0a8
              start: "2026-10-18T05:54:24.588686Z"
              end: "2026-10-18T06:12:34.398198Z"
              queue_seconds: 1089.809512
              raw_seconds: 1089.809512
              exact: true
            - attempt_id: c5ff20b0a514c353698bd714be553db5
              slot: pair:7
              record_digest: 9002e18184514276dbb2114de01eee431f90a50b1769457a9cc7eee8c10e8731
              start: "2026-10-18T06:26:02.440022Z"
              end: "2026-10-18T06:37:53.957156Z"
              queue_seconds: 711.517134
              raw_seconds: 711.517134
              exact: true
        cuts:
            - dispatched: "2026-10-17T23:55:26-07:00"
              finalized: "2026-10-17T23:57:08-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T00:01:30-07:00"
      dispatched: "2026-10-18T00:00:24-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan's Core concepts table now classifies requireLsofObservation as integration code, matching its environment reads, testing verdicts, host diagnostics and subprocess checks in proof_test.go:145–201. The Revisions entry records the correction; executable behavior is unchanged.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Both load-related Done when clauses at issue:104–114 and the active plan tasks now match the accepted measured-load Revision: injected-delay proof, observed-load repetitions and a separately recorded passing suite. The superseded clauses remain identified as historical.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: README.md:1848 documents the required-mode invocation and optional-skip versus required-failure behavior. Both native tests consume the shared helper; policy tests cover both modes and wrong answers, and fresh required-mode native tests passed.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 8f59f920b2064566f2da4baddf36d0f56cd6041f
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 35733e86-5955-41e5-ad09-d0e337405b90
            repository: github.com/xianxu/pair
            issue_id: "000659"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/fd15cb41369fa51f
            reviewed_head: 3d1b24fbe73d1e4dfd08826c634fbe6e823bf960
            slot: pair:7
            requested_at: "2026-10-18T18:59:42.552608Z"
          phase: finished
          pid: 23742
          started_at: "2026-10-18T18:59:42.764372Z"
          finished_at: "2026-10-18T19:00:24.409333Z"
          session_id: 01a15062-5278-70f2-9047-ed786db3be16
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 3d1b24fbe73d1e4dfd08826c634fbe6e823bf960
        tracker_head: 09d569a1ca195e013254f9ac5dbd767aae726222
        branch_point: db5ddb4e2bf5af6ec85c3c916e51b17523c12085
        since: "2026-10-17T22:43:28-07:00"
        until: "2026-10-18T11:58:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T22:43:28-07:00"
              end: "2026-10-18T11:58:56-07:00"
              began: "2026-10-17T22:43:28-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 68da14d797e07f0e0f4fe5fc81443816b57e113a2e7c187c64d7bd4490eea984
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 127364748
              windows:
                - start: "2026-10-17T22:43:28-07:00"
                  end: "2026-10-18T11:58:56-07:00"
            - harness: codex
              id: 01a14d89-aac7-71c2-89c5-147a00fb9a67
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 3dd9d336bf8eeadd0f77ca7a8a3130bb6d8d37b0806cd32d6aac262313ab6f62
              started: "2026-10-18T05:43:50.274Z"
              spawned: true
              bytes: 1113788
              windows:
                - start: "2026-10-18T05:43:50.274Z"
                  end: "2026-10-18T11:58:56-07:00"
            - harness: codex
              id: 01a14d89-bf84-7141-a788-8d4cb4d5e4bd
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 78754a84bf387aea43d706edb0aadba627da864cd38c4296836ede968414d2a8
              started: "2026-10-18T05:43:55.585Z"
              spawned: true
              bytes: 937469
              windows:
                - start: "2026-10-18T05:43:55.585Z"
                  end: "2026-10-18T11:58:56-07:00"
            - harness: codex
              id: 01a14d8e-89bf-7b70-b42f-586388f343a7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 035112415c3286ae0449dbd57372a747248e2725d39e4bbadaa42a94e060d73c
              started: "2026-10-18T05:49:09.523Z"
              spawned: true
              bytes: 733773
              windows:
                - start: "2026-10-18T05:49:09.523Z"
                  end: "2026-10-18T11:58:56-07:00"
        queue_waits:
            - attempt_id: db25bcb0f104165571bc9f60bcde47e2
              slot: pair:7
              record_digest: 0a62583d3f8b4bcdf08de5b22146365dcb1065bec167fdbb8ab9ea6e0fa7a0a8
              start: "2026-10-18T05:54:24.588686Z"
              end: "2026-10-18T06:12:34.398198Z"
              queue_seconds: 1089.809512
              raw_seconds: 1089.809512
              exact: true
            - attempt_id: c5ff20b0a514c353698bd714be553db5
              slot: pair:7
              record_digest: 9002e18184514276dbb2114de01eee431f90a50b1769457a9cc7eee8c10e8731
              start: "2026-10-18T06:26:02.440022Z"
              end: "2026-10-18T06:37:53.957156Z"
              queue_seconds: 711.517134
              raw_seconds: 711.517134
              exact: true
            - attempt_id: b6ecb5a8c0c61c236def180e67ec41cc
              slot: pair:7
              record_digest: f962e9784eb4fb42651944817be8e6620636b8d9f8bad55af3a2227da772557f
              start: "2026-10-18T07:50:48.285946Z"
              end: "2026-10-18T08:23:01.488885Z"
              queue_seconds: 1933.202939
              raw_seconds: 1933.202939
              exact: true
            - attempt_id: e3644ec5a5ebc04b7b8fa2219b970aa7
              slot: pair:7
              record_digest: 1010a3a7c4b776b85bdfc87c365bb5ca0307849e10d1bbb8fbc99eb5375e52a2
              start: "2026-10-18T18:18:11.063162Z"
              end: "2026-10-18T18:18:11.12553Z"
              queue_seconds: 0.062368
              raw_seconds: 0.062368
              exact: true
            - attempt_id: 6b9e8c389f3b9c2964d271e047e75235
              slot: pair:7
              record_digest: 4b705da20f3b3a88682c41d8f37624deccdb65cc5dffff67c2b0771c8aee9c84
              start: "2026-10-18T18:22:30.274165Z"
              end: "2026-10-18T18:45:44.547951Z"
              queue_seconds: 1394.273786
              raw_seconds: 1394.273786
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 22.653233333333333
        cuts:
            - dispatched: "2026-10-17T23:55:26-07:00"
              finalized: "2026-10-17T23:57:08-07:00"
              finalizing: false
            - dispatched: "2026-10-18T00:00:24-07:00"
              finalized: "2026-10-18T00:01:30-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-18T12:00:30-07:00"
      dispatched: "2026-10-18T11:59:39-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan's Core concepts row at line 31 correctly classifies requireLsofObservation as integration, matching its environment reads, testing verdicts and host diagnostics in proof_test.go:145–154.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Issue Done when at lines 105–113 and the plan's active verification tasks retain the approved measured-load acceptance. This window adds recovery evidence without changing that contract.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README.md:1892–1901 preserves the fresh required-mode invocation and optional-skip versus required-failure contract, matching proof_test.go:145–164. Required native conformance and verdict-policy tests passed.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 3d1b24fbe73d1e4dfd08826c634fbe6e823bf960
      blocked: false
---

# Gate ledger — pair#659 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T23:57:08-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `core-concept-purity-classification` ARCH-PURE: the probe verdict is classified PURE but depends on IO
  workshop/plans/000659-test-budgets-plan.md:27 labels the optional/required verdict PURE, but requireLsofObservation reads os.Getenv and can execute HostState (proof_test.go:145–154); its policy tests launch subprocesses (lines 167–201). This is the sole contradictory PURE row in this window. Promote it to INTEGRATION and record a plan Revisions entry, or extract a deterministic classifier taking explicit inputs and test that directly without IO.
- **BR-2** [Important] `acceptance-checklist-contract-sync` Done when retains the superseded load-acceptance contract
  workshop/issues/000659-tests-wall-clock-budgets-under-load.md:104–111 still requires deliberate fleet-load repetitions and a full suite under that same load, while Revisions:194 accepts injected-delay proof supported by observed-load repetitions. The recorded repetition and suite cohorts have different loads. Both load-related checklist clauses belong to this family; update them to express the accepted evidence requirements without implying another load campaign is necessary.
- **BR-3** [Important] `readme-new-run-surface` README update appears missing for required lsof conformance mode
  cmd/internal/diagnosticlog/proof_test.go:151 introduces PAIR_REQUIRE_LSOF_CONFORMANCE=1, and atlas/couch.md:26 documents its invocation, but README.md is unchanged. This is the only new operator-set configuration surface in this window; both native tests consume it through the shared helper. Add the invocation and optional-skip versus required-failure behavior to README's testing guidance.

## Round 2 — 2026-10-18T00:01:30-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The plan's Core concepts table now classifies requireLsofObservation as integration code, matching its environment reads, testing verdicts, host diagnostics and subprocess checks in proof_test.go:145–201. The Revisions entry records the correction; executable behavior is unchanged.
- BR-2 — addressed — Both load-related Done when clauses at issue:104–114 and the active plan tasks now match the accepted measured-load Revision: injected-delay proof, observed-load repetitions and a separately recorded passing suite. The superseded clauses remain identified as historical.
- BR-3 — addressed — README.md:1848 documents the required-mode invocation and optional-skip versus required-failure behavior. Both native tests consume the shared helper; policy tests cover both modes and wrong answers, and fresh required-mode native tests passed.

## Round 3 — 2026-10-18T12:00:30-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The plan's Core concepts row at line 31 correctly classifies requireLsofObservation as integration, matching its environment reads, testing verdicts and host diagnostics in proof_test.go:145–154.
- BR-2 — addressed — Issue Done when at lines 105–113 and the plan's active verification tasks retain the approved measured-load acceptance. This window adds recovery evidence without changing that contract.
- BR-3 — addressed — README.md:1892–1901 preserves the fresh required-mode invocation and optional-skip versus required-failure contract, matching proof_test.go:145–164. Required native conformance and verdict-policy tests passed.

## Open findings

(none — every finding has been disposed)
