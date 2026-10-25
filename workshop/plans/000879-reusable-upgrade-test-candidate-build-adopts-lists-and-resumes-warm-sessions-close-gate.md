---
gate: boundary-review
issue: 879
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6252fb16-3a54-49f9-9be1-f4c3f4481874
            repository: github.com/xianxu/pair
            issue_id: "000879"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/61e7d34fcc74b1be
            reviewed_head: ee8116e1ac241a77254c4ea015ffd57594c72427
            slot: pair:9
            requested_at: "2026-10-25T01:05:38.322254Z"
          phase: finished
          pid: 58544
          started_at: "2026-10-25T01:05:38.432676Z"
          finished_at: "2026-10-25T01:07:15.882042Z"
          session_id: 01a17097-7efc-7e51-93bc-cb52a8308dbc
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/pair
        code_head: ee8116e1ac241a77254c4ea015ffd57594c72427
        tracker_head: 1af5d8fe6c069eead268901744db91291547adfa
        branch_point: 597d8b49bc89498fd6fa37094b3aac0723f1c360
        since: "2026-10-24T17:15:49-07:00"
        until: "2026-10-24T17:59:49-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-24T17:15:49-07:00"
              end: "2026-10-24T17:59:49-07:00"
              began: "2026-10-24T17:15:49-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16e79-60b5-7b50-8db7-c97e0f54ec41
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 2afc341889f86bc7cb05ad84e744ef103e2a72f1f688afd90ab21bc049174f91
              started: "2026-10-24T15:13:32.628Z"
              spawned: false
              bytes: 10315075
              windows:
                - start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
              dispatch_windows:
                - start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-24T17:59:49-07:00"
        branches:
            - root: /Users/xianxu/workspace/worktree/pair-slot9/pair
              path: /Users/xianxu/workspace/pair/.git/worktrees/pair8/logs/HEAD
              head: ee8116e1ac241a77254c4ea015ffd57594c72427
              branch: 000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions
              digest: 0f5a8d674d2d64ea140023168423df92f1d1b6c0fb6c72cc3abbfbe6fe8b9283
              bytes: 70258
              intervals:
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15c54-f5ad-7701-bb23-643678932dc5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc4-b370-7d60-9efa-989c5b30869b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cc8-fb76-78e2-8c9a-cdf39be72681
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-866d-7c81-a3c6-38ba2cd75a01
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15cca-fa91-74b0-8ae6-64d332f5fa13
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d72-f362-71f2-ac46-571622add1dd
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-07d8-7e30-b172-25de09195a37
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15d78-9f81-7ee3-bed1-6016d32c5c39
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a15e49-d730-7d33-b4b3-809814e36de7
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-1ff0-7c41-8e18-3d145d14b1df
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16014-81a1-7001-9e7e-d93eed68c583
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601e-da52-7c40-a985-f00415b625a1
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1601f-7ced-7603-80df-24b55016bcd8
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16024-7679-7323-be47-5288a6d96993
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16025-4e19-7c81-b2f7-0259a80e5bfd
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615b-b3b2-71d3-878e-78e13474733a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1615f-3510-79f2-9cae-31cf36613788
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-37d2-7071-bfc4-bdb6afa7db78
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16161-e237-7f33-bb66-192582ab999d
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-593c-72c1-8c56-0ab2fb2d8d38
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16479-d353-7592-b10b-f52ec506bc33
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-645c-7e42-9827-fa4c62413563
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1647b-e38b-7623-aee0-a413c4337961
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d0-bf92-7a21-807c-ac181f0f9609
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a164d1-75c0-76b2-9860-a7072090d766
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-6420-7633-808f-be531aa7cac6
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1657a-e09e-7d83-84b2-14be06e54f9b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16855-b204-7ea1-aa9c-807df3683ad5
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16859-6f9d-7f41-b16c-a59b0f09c784
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685b-959d-72f1-b83a-9ac59140f8ae
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a1685c-1fd0-7683-938e-3900db4ac004
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16862-faa4-76e2-bce3-5bc33019d381
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16862-faa4-76e2-bce3-5bc33019d381
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16e3f-96da-77e2-9e7e-a2e539093730
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16e3f-96da-77e2-9e7e-a2e539093730
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16e42-5d03-7223-90f5-ee5ac13cda7a
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16e42-5d03-7223-90f5-ee5ac13cda7a
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16e44-5fe8-7922-8aeb-5f3633e5f47b
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16e44-5fe8-7922-8aeb-5f3633e5f47b
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source: codex/01a16e45-1372-7320-905e-20e435093633
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source: codex/01a16e45-1372-7320-905e-20e435093633
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T17:15:49-07:00"
                  end: "2026-10-25T00:16:24Z"
                  branch: main-slot9
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T00:16:24Z"
                  end: "2026-10-24T17:59:49-07:00"
                  kind: branch
      "n": 1
      timestamp: "2026-10-24T18:07:17-07:00"
      dispatched: "2026-10-24T18:05:35-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: The reusable upgrade oracle requires readiness markers to remain absent.
          detail: 'cmd/couch/upgrade_acceptance_test.go:295-299 rejects any nonempty zellij-ready-v1 directory on every snapshot, including candidate snapshots. A baseline that legitimately writes readiness receipts, or a compatible candidate that establishes them during adoption, therefore fails regardless of successful adoption/list/resume/attach. ARCH-PURPOSE: separate the historical pre-#866 absence proof from generic cohort snapshots; accept genuine marker-bearing baselines and legitimate candidate publication without manufacturing or deleting receipts. Add regression coverage for both cases.'
          family: qualification-baseline-independence
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: The core-concepts table names the wrong file for the pure cohort comparison.
          detail: 'workshop/plans/000879-native-upgrade-plan.md:21 places the comparison in cmd/internal/testcouch/upgrade.go, but that file contains replacement IO; compareUpgradeCohort actually lives at cmd/couch/upgrade_acceptance_test.go:304. The later Revisions passage acknowledges the move but leaves the authoritative table contradictory. ARCH-PURE: update the table to the actual function and location, retaining its PURE classification. This is Critical under the requested core-concepts cross-check.'
          family: core-concept-table-code-agreement
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: README guidance is missing for the new upgrade qualification invocation.
          detail: atlas/couch-testing.md:200 introduces TestDisposableCouchBuildUpgrade and PAIR_NATIVE_COUCH_UPGRADE, PAIR_COUCH_UPGRADE_BASE_SOURCE, and PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE, but README.md is unchanged. Add a discoverable invocation or concise link beside the existing disposable-fleet guidance at README.md:2694, including the generated-runtime preparation requirement.
          family: user-surface-readme-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#879 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T18:07:17-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `qualification-baseline-independence` (kind: defect) The reusable upgrade oracle requires readiness markers to remain absent.
  cmd/couch/upgrade_acceptance_test.go:295-299 rejects any nonempty zellij-ready-v1 directory on every snapshot, including candidate snapshots. A baseline that legitimately writes readiness receipts, or a compatible candidate that establishes them during adoption, therefore fails regardless of successful adoption/list/resume/attach. ARCH-PURPOSE: separate the historical pre-#866 absence proof from generic cohort snapshots; accept genuine marker-bearing baselines and legitimate candidate publication without manufacturing or deleting receipts. Add regression coverage for both cases.
- **BR-2** [Critical] `core-concept-table-code-agreement` (kind: defect) The core-concepts table names the wrong file for the pure cohort comparison.
  workshop/plans/000879-native-upgrade-plan.md:21 places the comparison in cmd/internal/testcouch/upgrade.go, but that file contains replacement IO; compareUpgradeCohort actually lives at cmd/couch/upgrade_acceptance_test.go:304. The later Revisions passage acknowledges the move but leaves the authoritative table contradictory. ARCH-PURE: update the table to the actual function and location, retaining its PURE classification. This is Critical under the requested core-concepts cross-check.
- **BR-3** [Important] `user-surface-readme-coverage` (kind: defect) README guidance is missing for the new upgrade qualification invocation.
  atlas/couch-testing.md:200 introduces TestDisposableCouchBuildUpgrade and PAIR_NATIVE_COUCH_UPGRADE, PAIR_COUCH_UPGRADE_BASE_SOURCE, and PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE, but README.md is unchanged. Add a discoverable invocation or concise link beside the existing disposable-fleet guidance at README.md:2694, including the generated-runtime preparation requirement.

## Open findings

- **BR-1** [Critical] `qualification-baseline-independence` (kind: defect) The reusable upgrade oracle requires readiness markers to remain absent.
- **BR-2** [Critical] `core-concept-table-code-agreement` (kind: defect) The core-concepts table names the wrong file for the pure cohort comparison.
- **BR-3** [Important] `user-surface-readme-coverage` (kind: defect) README guidance is missing for the new upgrade qualification invocation.
