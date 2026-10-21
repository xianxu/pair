---
gate: boundary-review
issue: 759
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7e66947f-6b48-4c5d-8579-93ce17324e03
            repository: github.com/xianxu/pair
            issue_id: "000759"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/bfcc938d287cd6e8
            reviewed_head: 1f260e9e9f1aa95d41ea540ea24a33d1c2d2168b
            slot: pair:6
            requested_at: "2026-10-21T17:47:24.851454Z"
          phase: finished
          pid: 91773
          started_at: "2026-10-21T17:47:24.973804Z"
          finished_at: "2026-10-21T17:49:43.716296Z"
          session_id: 01a15f93-34d4-7861-b862-01965ab91876
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1f260e9e9f1aa95d41ea540ea24a33d1c2d2168b
        tracker_head: 2421c921f0505b9f0fe77fad3439d4ed0398afe6
        branch_point: 9404e4e27c3cdd6eda6d2974f2aad57439a18e2c
        since: "2026-10-21T01:17:25-07:00"
        until: "2026-10-21T10:46:31-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T01:17:25-07:00"
              end: "2026-10-21T10:46:31-07:00"
              began: "2026-10-21T01:17:25-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 7094687d14c149bf8a3eed8d7758ae6d9f72ea2dd179b1fee63c1081dd12d513
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 81426286
              windows:
                - start: "2026-10-21T01:17:25-07:00"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15d89-a169-7cd2-9406-165d31bb6bfc
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 229f06a727fc87097b39cb5bf3eeb0dd5421fb1ee4ecf391b3fb77a12cb2df39
              started: "2026-10-21T08:17:43.497Z"
              spawned: true
              bytes: 907607
              windows:
                - start: "2026-10-21T08:17:43.497Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15d89-c0d9-73b3-ad3e-30eb355cce91
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 3b682eadb3c40b674e9f0f1be401d16eafaa05b29a575894acb3f3a9ecd381d1
              started: "2026-10-21T08:17:51.497Z"
              spawned: true
              bytes: 776410
              windows:
                - start: "2026-10-21T08:17:51.497Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15d92-c2c0-7452-9005-9f0932709c33
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 9b41ad0acc298dcf0ee5bf7d7fb9495b312313eb4e5f9427bd11797cb77294fa
              started: "2026-10-21T08:27:41.722Z"
              spawned: true
              bytes: 314006
              windows:
                - start: "2026-10-21T08:27:41.722Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f24-5859-7ec1-82e7-fb5e8400f798
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 12e9e5f81d984dbaf271ac59fa845f90bb96083c778e1641c509519ceab9e14b
              started: "2026-10-21T15:46:19.964Z"
              spawned: true
              bytes: 730135
              windows:
                - start: "2026-10-21T15:46:19.964Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f38-fdb4-7303-987f-6135f4bab39b
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: e267beb1c1f03db3a63daeed9c69841371e77333cafdbba9b66695aa6581393e
              started: "2026-10-21T16:08:53.326Z"
              spawned: false
              bytes: 718469
              windows:
                - start: "2026-10-21T16:08:53.326Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f3d-2459-7730-b0c5-4c4653a1f610
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: eaab9e42186411b1ee977c4c7f2f7d882f362bc9525036f9d56ff9b092418589
              started: "2026-10-21T16:13:25.886Z"
              spawned: false
              bytes: 557947
              windows:
                - start: "2026-10-21T16:13:25.886Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f3f-2ab4-75b0-9b7e-09a8c722eed8
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c0a5d1a77fe6a020b287d4b477f265ade84ba239fb1fd4bab1fc84524bf7ba32
              started: "2026-10-21T16:15:38.113Z"
              spawned: false
              bytes: 498949
              windows:
                - start: "2026-10-21T16:15:38.113Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f55-59bd-7770-a306-ac6bcff40a16
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: ddced05de91003e084dfe223c2111af8d5f653607c735afdc4381b405e69e7ae
              started: "2026-10-21T16:39:52.197Z"
              spawned: false
              bytes: 513630
              windows:
                - start: "2026-10-21T16:39:52.197Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f55-cc00-7781-a1ad-cded8f1e13ae
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 49c12f1dddd3b1ba45b76ca35fd082b16cb14b01299a5681cec229b3d9f2b606
              started: "2026-10-21T16:40:21.051Z"
              spawned: false
              bytes: 478993
              windows:
                - start: "2026-10-21T16:40:21.051Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f56-fc31-7962-82c4-5df123cb9eee
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: d79025ea7f3fab4a41b3f10c4161f8da54c3a1e2ce922ec06cc58e38d486bcd1
              started: "2026-10-21T16:41:38.96Z"
              spawned: false
              bytes: 503867
              windows:
                - start: "2026-10-21T16:41:38.96Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f57-bdca-78c1-9aa2-c8215b2a5ada
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: c40d48a254a41f95f4d73443d5d778c2c05693dad5ecf00e688c551c67830249
              started: "2026-10-21T16:42:28.372Z"
              spawned: true
              bytes: 470181
              windows:
                - start: "2026-10-21T16:42:28.372Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f6f-6cb2-76a1-9f8d-ad03228a1dba
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fc65f564ea69e073d74bdc167bc008d875890b4d73acddc884435abd68198265
              started: "2026-10-21T17:08:20.452Z"
              spawned: true
              bytes: 558180
              windows:
                - start: "2026-10-21T17:08:20.452Z"
                  end: "2026-10-21T10:46:31-07:00"
            - harness: codex
              id: 01a15f86-bb18-7030-a44c-3328ff9d0870
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: a771fa8fe52cd10613e45c7b4a892ea11b5516be4db6b495088644d4ace20536
              started: "2026-10-21T17:33:47.966Z"
              spawned: true
              bytes: 406808
              windows:
                - start: "2026-10-21T17:33:47.966Z"
                  end: "2026-10-21T10:46:31-07:00"
      "n": 1
      timestamp: "2026-10-21T10:49:45-07:00"
      dispatched: "2026-10-21T10:47:22-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Session admission does not match its PURE classification.
          detail: 'workshop/plans/000759-remote-security-plan.md:94 lists a pure SessionState in session.go, but no such entity exists. Sessions.Handshake directly reads OS randomness and mutates synchronized state (session.go:43-94); its tests exercise that integration path. ARCH-PURE: promote Sessions to INTEGRATION and identify any actual pure admission values/functions, or extract the promised pure core. Add a Revisions entry correcting the classification.'
          family: core-concept-classification
          round: 1
        - id: BR-2
          severity: Important
          title: Session qualification lacks the promised generated sequences and controlled ordering.
          detail: 'The plan''s Sessions strategy at workshop/plans/000759-remote-security-plan.md:259 requires fake-clock generated sequences and a race rendezvous. session_test.go contains three hand-written tests; TestRemoteSessionsConcurrentCap at line 60 launches goroutines without controlling their ordering and never interleaves admission with revocation, logout or expiry reclamation. ARCH-ORDER: add reproducible sequence coverage with independent capacity/revocation invariants and held admission boundaries.'
          family: deterministic-state-sequence-coverage
          round: 1
        - id: BR-3
          severity: Important
          title: Interrupted atomic publications leave unbounded remote-state staging files.
          detail: 'store.go:67, origin_store.go:53 and audit.go:37 create unique .remote-latch-*, .remote-origin-* and .remote-audit-* files through durablefile.WriteAtomic. durablefile/write.go removes them only through deferred cleanup; process death before rename leaves them behind, and no opener or retention routine collects them. ARCH-FUNERAL: add ownership-safe recovery cleanup or a bounded reusable staging scheme, with interrupted-publication tests; otherwise repeated crashes bypass the stated latch/origin/audit disk bounds.'
          family: durable-artifact-lifecycle
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 49b2908e-9f38-4c2b-a5a1-c156d32efe53
            repository: github.com/xianxu/pair
            issue_id: "000759"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/160aefea4c6cd3b6
            reviewed_head: 1811e47e668e236b6ce31de774954ce6759c7e2d
            slot: pair:6
            requested_at: "2026-10-21T17:58:39.193255Z"
          phase: finished
          pid: 7746
          started_at: "2026-10-21T17:58:39.359077Z"
          finished_at: "2026-10-21T18:00:48.648989Z"
          session_id: 01a15f9d-7fdb-7723-b305-56594b190ef3
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1811e47e668e236b6ce31de774954ce6759c7e2d
        tracker_head: b6bd5d48b11e6af5a6ec9e678ef2cc362c115e84
        branch_point: 9404e4e27c3cdd6eda6d2974f2aad57439a18e2c
        since: "2026-10-21T01:17:25-07:00"
        until: "2026-10-21T10:57:41-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T01:17:25-07:00"
              end: "2026-10-21T10:57:41-07:00"
              began: "2026-10-21T01:17:25-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: bcb401b64e237fe562c5b7949e77eaf3b52d15d4f9a3749b14cb410f472a8ec7
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 82073663
              windows:
                - start: "2026-10-21T01:17:25-07:00"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15d89-a169-7cd2-9406-165d31bb6bfc
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 229f06a727fc87097b39cb5bf3eeb0dd5421fb1ee4ecf391b3fb77a12cb2df39
              started: "2026-10-21T08:17:43.497Z"
              spawned: true
              bytes: 907607
              windows:
                - start: "2026-10-21T08:17:43.497Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15d89-c0d9-73b3-ad3e-30eb355cce91
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 3b682eadb3c40b674e9f0f1be401d16eafaa05b29a575894acb3f3a9ecd381d1
              started: "2026-10-21T08:17:51.497Z"
              spawned: true
              bytes: 776410
              windows:
                - start: "2026-10-21T08:17:51.497Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15d92-c2c0-7452-9005-9f0932709c33
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 9b41ad0acc298dcf0ee5bf7d7fb9495b312313eb4e5f9427bd11797cb77294fa
              started: "2026-10-21T08:27:41.722Z"
              spawned: true
              bytes: 314006
              windows:
                - start: "2026-10-21T08:27:41.722Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f24-5859-7ec1-82e7-fb5e8400f798
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 12e9e5f81d984dbaf271ac59fa845f90bb96083c778e1641c509519ceab9e14b
              started: "2026-10-21T15:46:19.964Z"
              spawned: true
              bytes: 730135
              windows:
                - start: "2026-10-21T15:46:19.964Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f38-fdb4-7303-987f-6135f4bab39b
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: e267beb1c1f03db3a63daeed9c69841371e77333cafdbba9b66695aa6581393e
              started: "2026-10-21T16:08:53.326Z"
              spawned: false
              bytes: 718469
              windows:
                - start: "2026-10-21T16:08:53.326Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f3d-2459-7730-b0c5-4c4653a1f610
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: eaab9e42186411b1ee977c4c7f2f7d882f362bc9525036f9d56ff9b092418589
              started: "2026-10-21T16:13:25.886Z"
              spawned: false
              bytes: 557947
              windows:
                - start: "2026-10-21T16:13:25.886Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f3f-2ab4-75b0-9b7e-09a8c722eed8
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: c0a5d1a77fe6a020b287d4b477f265ade84ba239fb1fd4bab1fc84524bf7ba32
              started: "2026-10-21T16:15:38.113Z"
              spawned: false
              bytes: 498949
              windows:
                - start: "2026-10-21T16:15:38.113Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f55-59bd-7770-a306-ac6bcff40a16
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: ddced05de91003e084dfe223c2111af8d5f653607c735afdc4381b405e69e7ae
              started: "2026-10-21T16:39:52.197Z"
              spawned: false
              bytes: 513630
              windows:
                - start: "2026-10-21T16:39:52.197Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f55-cc00-7781-a1ad-cded8f1e13ae
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 49c12f1dddd3b1ba45b76ca35fd082b16cb14b01299a5681cec229b3d9f2b606
              started: "2026-10-21T16:40:21.051Z"
              spawned: false
              bytes: 478993
              windows:
                - start: "2026-10-21T16:40:21.051Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f56-fc31-7962-82c4-5df123cb9eee
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: d79025ea7f3fab4a41b3f10c4161f8da54c3a1e2ce922ec06cc58e38d486bcd1
              started: "2026-10-21T16:41:38.96Z"
              spawned: false
              bytes: 503867
              windows:
                - start: "2026-10-21T16:41:38.96Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f57-bdca-78c1-9aa2-c8215b2a5ada
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: c40d48a254a41f95f4d73443d5d778c2c05693dad5ecf00e688c551c67830249
              started: "2026-10-21T16:42:28.372Z"
              spawned: true
              bytes: 470181
              windows:
                - start: "2026-10-21T16:42:28.372Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f6f-6cb2-76a1-9f8d-ad03228a1dba
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fc65f564ea69e073d74bdc167bc008d875890b4d73acddc884435abd68198265
              started: "2026-10-21T17:08:20.452Z"
              spawned: true
              bytes: 558180
              windows:
                - start: "2026-10-21T17:08:20.452Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f86-bb18-7030-a44c-3328ff9d0870
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: a771fa8fe52cd10613e45c7b4a892ea11b5516be4db6b495088644d4ace20536
              started: "2026-10-21T17:33:47.966Z"
              spawned: true
              bytes: 406808
              windows:
                - start: "2026-10-21T17:33:47.966Z"
                  end: "2026-10-21T10:57:41-07:00"
            - harness: codex
              id: 01a15f96-84fc-7261-97a7-553e48f2d938
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 2b74832f998118784954d72126330594da77f6aee2aa94def679c0f94d58db40
              started: "2026-10-21T17:51:02.624Z"
              spawned: true
              bytes: 469098
              windows:
                - start: "2026-10-21T17:51:02.624Z"
                  end: "2026-10-21T10:57:41-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-21T10:47:22-07:00"
              finalized: "2026-10-21T10:49:45-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T11:00:49-07:00"
      dispatched: "2026-10-21T10:58:35-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The pinned correction removes nonexistent SessionState, classifies Sessions as INTEGRATION at plan line 101, and identifies pure identity predicates at line 93. session.go confirms entropy and mutex ownership; policy.go contains the value predicates. The Revisions entry records this correction.
          round: 2
        - id: BR-2
          disposition: addressed
          note: session_sequence_test.go adds 24 seeded lifecycle sequences, held handshake boundaries interleaving revoke/logout/expiry, and rendezvous capacity waves of 8/16 callers. Independent ownership, historical-cookie and cancellation assertions exercise production Sessions methods; normal and race runs pass.
          round: 2
        - id: BR-3
          disposition: addressed
          note: All four remote records use bounded reusable stages with opener recovery. Publication tests cover retained authority, foreign siblings and unsafe slots. In a disposable pinned-head copy, disabling reclaimRemoteStage made the recovery tests fail; the original tests passed. The disposable copy was removed.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Critical
          title: Core-concepts table assigns DocumentResult to the wrong file.
          detail: 'workshop/plans/000759-remote-security-plan.md:94 places DocumentResult in protocol.go, but its declaration is cmd/internal/remotesession/documents.go:18. This is the 2nd finding in family core-concept-classification. ARCH-PURE / ARCH-PURPOSE: apply the rule that every delivered table entity must match its actual declaration, classification and path; sweep all delivered rows, split this row by location, and record the correction under Revisions. Critical follows the review contract’s explicit severity for table/code contradictions.'
          family: core-concept-classification
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#759 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T10:49:45-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `core-concept-classification` Session admission does not match its PURE classification.
  workshop/plans/000759-remote-security-plan.md:94 lists a pure SessionState in session.go, but no such entity exists. Sessions.Handshake directly reads OS randomness and mutates synchronized state (session.go:43-94); its tests exercise that integration path. ARCH-PURE: promote Sessions to INTEGRATION and identify any actual pure admission values/functions, or extract the promised pure core. Add a Revisions entry correcting the classification.
- **BR-2** [Important] `deterministic-state-sequence-coverage` Session qualification lacks the promised generated sequences and controlled ordering.
  The plan's Sessions strategy at workshop/plans/000759-remote-security-plan.md:259 requires fake-clock generated sequences and a race rendezvous. session_test.go contains three hand-written tests; TestRemoteSessionsConcurrentCap at line 60 launches goroutines without controlling their ordering and never interleaves admission with revocation, logout or expiry reclamation. ARCH-ORDER: add reproducible sequence coverage with independent capacity/revocation invariants and held admission boundaries.
- **BR-3** [Important] `durable-artifact-lifecycle` Interrupted atomic publications leave unbounded remote-state staging files.
  store.go:67, origin_store.go:53 and audit.go:37 create unique .remote-latch-*, .remote-origin-* and .remote-audit-* files through durablefile.WriteAtomic. durablefile/write.go removes them only through deferred cleanup; process death before rename leaves them behind, and no opener or retention routine collects them. ARCH-FUNERAL: add ownership-safe recovery cleanup or a bounded reusable staging scheme, with interrupted-publication tests; otherwise repeated crashes bypass the stated latch/origin/audit disk bounds.

## Round 2 — 2026-10-21T11:00:49-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The pinned correction removes nonexistent SessionState, classifies Sessions as INTEGRATION at plan line 101, and identifies pure identity predicates at line 93. session.go confirms entropy and mutex ownership; policy.go contains the value predicates. The Revisions entry records this correction.
- BR-2 — addressed — session_sequence_test.go adds 24 seeded lifecycle sequences, held handshake boundaries interleaving revoke/logout/expiry, and rendezvous capacity waves of 8/16 callers. Independent ownership, historical-cookie and cancellation assertions exercise production Sessions methods; normal and race runs pass.
- BR-3 — addressed — All four remote records use bounded reusable stages with opener recovery. Publication tests cover retained authority, foreign siblings and unsafe slots. In a disposable pinned-head copy, disabling reclaimRemoteStage made the recovery tests fail; the original tests passed. The disposable copy was removed.

### Raised

- **BR-4** [Critical] `core-concept-classification` (kind: defect) Core-concepts table assigns DocumentResult to the wrong file.
  workshop/plans/000759-remote-security-plan.md:94 places DocumentResult in protocol.go, but its declaration is cmd/internal/remotesession/documents.go:18. This is the 2nd finding in family core-concept-classification. ARCH-PURE / ARCH-PURPOSE: apply the rule that every delivered table entity must match its actual declaration, classification and path; sweep all delivered rows, split this row by location, and record the correction under Revisions. Critical follows the review contract’s explicit severity for table/code contradictions.

## Open findings

- **BR-4** [Critical] `core-concept-classification` (kind: defect) Core-concepts table assigns DocumentResult to the wrong file.
