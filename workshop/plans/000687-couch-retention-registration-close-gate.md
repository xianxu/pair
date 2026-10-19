---
gate: boundary-review
issue: 687
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a996d582-a0d5-417d-8313-53923948b521
            repository: github.com/xianxu/pair
            issue_id: "000687"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ac20ddb4b7aad3b3
            reviewed_head: 230a100ed9601fbb70209de6ad677f19ea8af004
            slot: pair:3
            requested_at: "2026-10-19T04:48:00.014053Z"
          phase: finished
          pid: 90383
          started_at: "2026-10-19T04:48:00.793076Z"
          finished_at: "2026-10-19T04:50:59.348486Z"
          session_id: 01a1527c-ed66-78c1-b57d-444794477e6b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 230a100ed9601fbb70209de6ad677f19ea8af004
        tracker_head: a46cbc5b5569913aac51dd924df2bb54df032b63
        branch_point: 213f82ae5b9471b81d75bb9b8cd4e550b0274e1d
        since: "2026-10-18T21:13:11-07:00"
        until: "2026-10-18T21:46:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T21:13:11-07:00"
              end: "2026-10-18T21:46:45-07:00"
              began: "2026-10-18T21:13:11-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 0e6c9006ae492564993d4822533e2734a98448d8acd93846307b6a927e9f8849
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 97103380
              windows:
                - start: "2026-10-18T21:13:11-07:00"
                  end: "2026-10-18T21:46:45-07:00"
            - harness: codex
              id: 01a1525e-0fab-7162-987c-6fc807387573
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: bb8cc76b1c2d6e8a7d2eb10b53c7fbd523f0de870ecc618b93557f1ea846b893
              started: "2026-10-19T04:14:18.649Z"
              spawned: true
              bytes: 510287
              windows:
                - start: "2026-10-19T04:14:18.649Z"
                  end: "2026-10-18T21:46:45-07:00"
            - harness: codex
              id: 01a15267-8ed3-7322-af10-a634acb97ee6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: bc598dbc34a5385d0ec6c2f95bfc17203733d2eb34211e6ceff6fe4d98eba439
              started: "2026-10-19T04:24:42.055Z"
              spawned: false
              bytes: 584936
              windows:
                - start: "2026-10-19T04:24:42.055Z"
                  end: "2026-10-18T21:46:45-07:00"
            - harness: codex
              id: 01a15268-9cbe-7e23-ab0f-2aa2bd992cfa
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 5675acb69afcb4056cde39a222580eae140acffab7a8cdc50839aa4b5eb46436
              started: "2026-10-19T04:25:51.146Z"
              spawned: false
              bytes: 459116
              windows:
                - start: "2026-10-19T04:25:51.146Z"
                  end: "2026-10-18T21:46:45-07:00"
            - harness: codex
              id: 01a15269-932d-77b0-885c-5bbd692ea880
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 73dd690729861ea2f09bc9636248369909d5108a7dccd8c0ed65921c84108802
              started: "2026-10-19T04:26:53.897Z"
              spawned: false
              bytes: 454494
              windows:
                - start: "2026-10-19T04:26:53.897Z"
                  end: "2026-10-18T21:46:45-07:00"
            - harness: codex
              id: 01a1526a-0344-71e1-b4c9-946c5d6f6a7a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 8320e6ade9cebddcff9a51c763db840c8b8a699da4774dddf49fbe07b6c3168d
              started: "2026-10-19T04:27:22.555Z"
              spawned: false
              bytes: 406195
              windows:
                - start: "2026-10-19T04:27:22.555Z"
                  end: "2026-10-18T21:46:45-07:00"
            - harness: codex
              id: 01a1526a-aacd-7992-9e08-602a181cd6fa
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 073a3a54ffcb0b77f4e13afd6a9c59f95178c1bd49b936a5f0674afffdcfc94b
              started: "2026-10-19T04:28:04.838Z"
              spawned: true
              bytes: 679396
              windows:
                - start: "2026-10-19T04:28:04.838Z"
                  end: "2026-10-18T21:46:45-07:00"
      "n": 1
      timestamp: "2026-10-18T21:51:01-07:00"
      dispatched: "2026-10-18T21:47:56-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: 'ARCH-PURE: the PURE participation predicate is tested through filesystem IO and a fake probe'
          detail: cmd/internal/diagnosticlog/runtime_registry_test.go:48 calls supervisorFixture, which creates a temporary coordinator, injects leaseProcessWorld, and publishes runtimes.json; line 49 then reads that file. This violates the supplied PURE-entity test requirement for the predicate classified PURE in the plan at line 33. Construct ProcessRegistration and Registration literals directly for this table test, leaving registry IO and process doubles in the existing integration tests.
          family: pure-entity-test-isolation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: cebb990f-ab08-443c-ba8d-c19db34fc24b
            repository: github.com/xianxu/pair
            issue_id: "000687"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/add8c30cf075554b
            reviewed_head: f479a6e80c2849d687539c6c4377d31b0fdc12c5
            slot: pair:3
            requested_at: "2026-10-19T04:53:05.130222Z"
          phase: finished
          pid: 85088
          started_at: "2026-10-19T04:53:05.336179Z"
          finished_at: "2026-10-19T04:55:26.035434Z"
          session_id: 01a15281-925d-7410-9a95-0c6c8c7496cb
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: f479a6e80c2849d687539c6c4377d31b0fdc12c5
        tracker_head: a46cbc5b5569913aac51dd924df2bb54df032b63
        branch_point: 213f82ae5b9471b81d75bb9b8cd4e550b0274e1d
        since: "2026-10-18T21:13:11-07:00"
        until: "2026-10-18T21:51:50-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T21:13:11-07:00"
              end: "2026-10-18T21:51:50-07:00"
              began: "2026-10-18T21:13:11-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: e307604ab807db1bb958e7998b6f5ffdf769ced8d34417118f21d843f272790a
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 97358045
              windows:
                - start: "2026-10-18T21:13:11-07:00"
                  end: "2026-10-18T21:51:50-07:00"
            - harness: codex
              id: 01a1525e-0fab-7162-987c-6fc807387573
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: bb8cc76b1c2d6e8a7d2eb10b53c7fbd523f0de870ecc618b93557f1ea846b893
              started: "2026-10-19T04:14:18.649Z"
              spawned: true
              bytes: 510287
              windows:
                - start: "2026-10-19T04:14:18.649Z"
                  end: "2026-10-18T21:51:50-07:00"
            - harness: codex
              id: 01a15267-8ed3-7322-af10-a634acb97ee6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: bc598dbc34a5385d0ec6c2f95bfc17203733d2eb34211e6ceff6fe4d98eba439
              started: "2026-10-19T04:24:42.055Z"
              spawned: false
              bytes: 584936
              windows:
                - start: "2026-10-19T04:24:42.055Z"
                  end: "2026-10-18T21:51:50-07:00"
            - harness: codex
              id: 01a15268-9cbe-7e23-ab0f-2aa2bd992cfa
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 5675acb69afcb4056cde39a222580eae140acffab7a8cdc50839aa4b5eb46436
              started: "2026-10-19T04:25:51.146Z"
              spawned: false
              bytes: 459116
              windows:
                - start: "2026-10-19T04:25:51.146Z"
                  end: "2026-10-18T21:51:50-07:00"
            - harness: codex
              id: 01a15269-932d-77b0-885c-5bbd692ea880
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 73dd690729861ea2f09bc9636248369909d5108a7dccd8c0ed65921c84108802
              started: "2026-10-19T04:26:53.897Z"
              spawned: false
              bytes: 454494
              windows:
                - start: "2026-10-19T04:26:53.897Z"
                  end: "2026-10-18T21:51:50-07:00"
            - harness: codex
              id: 01a1526a-0344-71e1-b4c9-946c5d6f6a7a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 8320e6ade9cebddcff9a51c763db840c8b8a699da4774dddf49fbe07b6c3168d
              started: "2026-10-19T04:27:22.555Z"
              spawned: false
              bytes: 406195
              windows:
                - start: "2026-10-19T04:27:22.555Z"
                  end: "2026-10-18T21:51:50-07:00"
            - harness: codex
              id: 01a1526a-aacd-7992-9e08-602a181cd6fa
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 073a3a54ffcb0b77f4e13afd6a9c59f95178c1bd49b936a5f0674afffdcfc94b
              started: "2026-10-19T04:28:04.838Z"
              spawned: true
              bytes: 679396
              windows:
                - start: "2026-10-19T04:28:04.838Z"
                  end: "2026-10-18T21:51:50-07:00"
        cuts:
            - dispatched: "2026-10-18T21:47:56-07:00"
              finalized: "2026-10-18T21:51:01-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T21:55:26-07:00"
      dispatched: "2026-10-18T21:53:02-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: runtime_registry_test.go:47 now tests matchesRuntimeRegistration using value literals only. The pinned correction removes filesystem setup, registry reads and the fake probe; the test passes. This changes test isolation, not production behavior.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: f479a6e80c2849d687539c6c4377d31b0fdc12c5
      blocked: false
---

# Gate ledger — pair#687 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T21:51:01-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `pure-entity-test-isolation` ARCH-PURE: the PURE participation predicate is tested through filesystem IO and a fake probe
  cmd/internal/diagnosticlog/runtime_registry_test.go:48 calls supervisorFixture, which creates a temporary coordinator, injects leaseProcessWorld, and publishes runtimes.json; line 49 then reads that file. This violates the supplied PURE-entity test requirement for the predicate classified PURE in the plan at line 33. Construct ProcessRegistration and Registration literals directly for this table test, leaving registry IO and process doubles in the existing integration tests.

## Round 2 — 2026-10-18T21:55:26-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — runtime_registry_test.go:47 now tests matchesRuntimeRegistration using value literals only. The pinned correction removes filesystem setup, registry reads and the fake probe; the test passes. This changes test isolation, not production behavior.

## Open findings

(none — every finding has been disposed)
