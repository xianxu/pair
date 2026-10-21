---
gate: boundary-review
issue: 743
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 12905d93-bd0d-4524-a31e-02c22afb0b59
            repository: github.com/xianxu/pair
            issue_id: "000743"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a14ad98fe2ac673c
            reviewed_head: 784562d8fa7168ae60e053935f6dc59d3af47290
            slot: pair:3
            requested_at: "2026-10-21T05:00:12.090035Z"
          phase: finished
          pid: 26028
          started_at: "2026-10-21T05:00:12.248648Z"
          finished_at: "2026-10-21T05:03:33.021964Z"
          session_id: 01a15cd4-cdc5-7e83-b5cf-b71fdc577ede
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 784562d8fa7168ae60e053935f6dc59d3af47290
        tracker_head: f2e9d56cf18d6f58cadea5f3f9e4f1e4de005e30
        branch_point: e30ea629818a733f65741bfc95c11de8e1868f34
        since: "2026-10-20T14:15:16-07:00"
        until: "2026-10-20T21:58:40-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T14:15:16-07:00"
              end: "2026-10-20T21:58:40-07:00"
              began: "2026-10-20T14:15:16-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 71dfb8c43ebb17e72745d652ddbb636635a66516f34a95b2cfbba8c5c2b74f0f
              started: "2026-10-20T09:33:46.808Z"
              spawned: false
              bytes: 19210641
              windows:
                - start: "2026-10-20T14:15:16-07:00"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b2b-9845-71f3-b6dd-4bef6f6724eb
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 4a4f05fd6a18dd01bb0bd44ef87689adcfde5401e1e4d80c5021643bde460a84
              started: "2026-10-20T21:15:46.291Z"
              spawned: true
              bytes: 730506
              windows:
                - start: "2026-10-20T21:15:46.291Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b2b-ac07-7d63-b9e6-1c18554def37
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: e5cbac97670ef3153d96535fa79b3e6ec70e9e052386a08131a1820c44865b89
              started: "2026-10-20T21:15:51.351Z"
              spawned: true
              bytes: 656968
              windows:
                - start: "2026-10-20T21:15:51.351Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b31-ece1-7830-98f3-1fa9fb8a5d92
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 2c7558c7623b153d4eebe2a97163a6732d27bf94cf9d6d2487800beb9e52b35a
              started: "2026-10-20T21:22:41.204Z"
              spawned: true
              bytes: 421969
              windows:
                - start: "2026-10-20T21:22:41.204Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b36-98e6-78b3-8e9b-4624ada9685b
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 6b05824663e21299311a78ff5d9ad59711e650d24ea94622c8d99eb2fef6dd63
              started: "2026-10-20T21:27:47.553Z"
              spawned: false
              bytes: 585139
              windows:
                - start: "2026-10-20T21:27:47.553Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b38-ee9d-7dd2-b72d-627123cc86c4
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 1c2fd30c96f2e651a3acda247ad3fcae196b5a9c16d50a31732e7d8f4e690a9e
              started: "2026-10-20T21:30:20.567Z"
              spawned: false
              bytes: 434988
              windows:
                - start: "2026-10-20T21:30:20.567Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b3a-36e5-7f83-9e8c-40fc7a574de7
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 292bc899bce90be0c8b8f2bca9b23e995944044f00bab34fde0148b5a745112b
              started: "2026-10-20T21:31:44.598Z"
              spawned: false
              bytes: 465191
              windows:
                - start: "2026-10-20T21:31:44.598Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b3a-9b8e-7681-b768-e336d4012d7c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 6b7598c0813ecb82101bb2adc3aa1813a6261fb6fd9ce028ba2b349a31c5edfc
              started: "2026-10-20T21:32:10.42Z"
              spawned: false
              bytes: 406507
              windows:
                - start: "2026-10-20T21:32:10.42Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b3c-1cd7-7f82-89ba-164d5f69becd
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 06972463ca050ec46ea057080f0b9f731bf1b223ce4aa30904b1ef64eee40f97
              started: "2026-10-20T21:33:48.745Z"
              spawned: true
              bytes: 392945
              windows:
                - start: "2026-10-20T21:33:48.745Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b3e-dc8c-7d71-b3ce-bc2fba4b9c09
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 209c3c4b414e3fa4362790479e63907a3d620206b454dbcc071ad7675c530a2b
              started: "2026-10-20T21:36:48.878Z"
              spawned: true
              bytes: 374412
              windows:
                - start: "2026-10-20T21:36:48.878Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b3f-8490-7362-b166-7ce05d4f0c3f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 2e5148d49347e8e6214e989b1f63f42f2077ca982e8102089c05c56d89618ffc
              started: "2026-10-20T21:37:31.89Z"
              spawned: true
              bytes: 297928
              windows:
                - start: "2026-10-20T21:37:31.89Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b40-7436-7653-a157-6712ebb38b07
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 7e4598da5be6d91a5131ede1c477dc41c3604079d44adfa9b55fbd830b4611ae
              started: "2026-10-20T21:38:33.237Z"
              spawned: true
              bytes: 421949
              windows:
                - start: "2026-10-20T21:38:33.237Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b41-dd9c-7781-937e-e347ce239915
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 6545ccd94ef550c6d5a650c5de51f0bcf467d2450632fc2e2118fc9c95e8ba60
              started: "2026-10-20T21:40:05.754Z"
              spawned: true
              bytes: 402938
              windows:
                - start: "2026-10-20T21:40:05.754Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b44-a87a-7571-a797-c5eb0191f50c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 5a9a8ff598e6100f53cb4c1e2ccfac210304738233d35507876056215dde13a3
              started: "2026-10-20T21:43:08.819Z"
              spawned: true
              bytes: 505254
              windows:
                - start: "2026-10-20T21:43:08.819Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b46-b9b7-7d90-a21b-5805637fb71e
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: a906d9fd2684a2f963ed64523e550897167b508e8b07a9467619102ea45a6381
              started: "2026-10-20T21:45:24.246Z"
              spawned: true
              bytes: 302914
              windows:
                - start: "2026-10-20T21:45:24.246Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b4c-77c4-7b83-b014-e243f7398311
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a158a8-e5f4-7681-8bd6-2773c44f4444
              digest: 68894618e0c5591d1868f2941900cebcb9edf12c4a9ddb3021c6aa6a7b8db698
              started: "2026-10-20T21:51:40.538Z"
              spawned: true
              bytes: 487921
              windows:
                - start: "2026-10-20T21:51:40.538Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: a9ac6a303da7b025980881a56b4a35c6a17c64f72a26c00836be57c702882cad
              started: "2026-10-20T21:55:01.064Z"
              spawned: false
              bytes: 9893770
              windows:
                - start: "2026-10-20T21:55:01.064Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15bb6-91ad-7e82-be0e-60bb2692e4bd
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              digest: 94fd67d574e2132a75e5c688414f16e45df411c7d15e6001a3ea563edd988204
              started: "2026-10-20T23:47:34.021Z"
              spawned: true
              bytes: 940608
              windows:
                - start: "2026-10-20T23:47:34.021Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15bb8-0416-7580-b82c-9e15bd953a80
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              digest: 31d9ec7858ed99006cb637c8d918b79e037f7859d311a7424fe26e48ded8e5e9
              started: "2026-10-20T23:49:08.892Z"
              spawned: true
              bytes: 550987
              windows:
                - start: "2026-10-20T23:49:08.892Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15bbb-65d9-7473-a327-686b0e3cb7e9
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              digest: c8a7c580b4dc776fecda1100dffe3092ee043353a8334edc4698693472312c51
              started: "2026-10-20T23:52:50.475Z"
              spawned: true
              bytes: 1455203
              windows:
                - start: "2026-10-20T23:52:50.475Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15bc6-97c6-7611-b87a-1303bbc97e1b
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: dec60d9dbe4c3a558b7701843d8d1f19a47944a6004370488d532f4d0870b99e
              started: "2026-10-21T00:05:05.925Z"
              spawned: false
              bytes: 502372
              windows:
                - start: "2026-10-21T00:05:05.925Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15bc7-2132-7471-8253-5860ca9d63b1
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: b6e5768c9a9ac30b06d32c163a7c50a2b80c0a856ccbb1f4f66ab3f2b6607006
              started: "2026-10-21T00:05:40.247Z"
              spawned: false
              bytes: 516946
              windows:
                - start: "2026-10-21T00:05:40.247Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15bd2-95d0-7313-82c6-b36443b66182
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              digest: 9029806a907c4a511883d419eb8e24b9fb527f3f6718df18ab675dbbe7425fb8
              started: "2026-10-21T00:18:10.369Z"
              spawned: true
              bytes: 729638
              windows:
                - start: "2026-10-21T00:18:10.369Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15be1-7f3d-7fe1-b320-918261b6457c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15b4f-8616-7e93-b16c-446f5ace13a6
              digest: 2cee4db4aa1921ba94a9c741e860c7600a1cac4b0e3368b0e02085693bcfb6a3
              started: "2026-10-21T00:34:27.441Z"
              spawned: true
              bytes: 1866907
              windows:
                - start: "2026-10-21T00:34:27.441Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15cae-ee73-7ad3-bb95-fb0e3f89f4f1
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 4d87212ee266f676110ccad424124357d6e09c1cfbd82b90d7df9d5a4cd72cfa
              started: "2026-10-21T04:18:51.096Z"
              spawned: false
              bytes: 3185870
              windows:
                - start: "2026-10-21T04:18:51.096Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15caf-fe1c-72c0-9ade-476641487797
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15cae-ee73-7ad3-bb95-fb0e3f89f4f1
              digest: 2a25fc44595a59b550fd43a3c698362df889e851008b76e40c846a6b29e23d87
              started: "2026-10-21T04:20:00.259Z"
              spawned: true
              bytes: 1067074
              windows:
                - start: "2026-10-21T04:20:00.259Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15cb4-26db-7d50-821d-b950884b834d
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15cae-ee73-7ad3-bb95-fb0e3f89f4f1
              digest: 675275b770e45fcdf63b6bed0cd802d06dbaef68e4db93b991f376f7e398ae19
              started: "2026-10-21T04:24:32.779Z"
              spawned: true
              bytes: 658211
              windows:
                - start: "2026-10-21T04:24:32.779Z"
                  end: "2026-10-20T21:58:40-07:00"
            - harness: codex
              id: 01a15cb8-78db-77e3-9609-d6ffbba91890
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15cae-ee73-7ad3-bb95-fb0e3f89f4f1
              digest: 8fc7740b61aaf14bb014bc36f234b762e4730e2e708d05eb139d0ae7ca2ef891
              started: "2026-10-21T04:29:15.953Z"
              spawned: true
              bytes: 759338
              windows:
                - start: "2026-10-21T04:29:15.953Z"
                  end: "2026-10-20T21:58:40-07:00"
        queue_waits:
            - attempt_id: 56bcc341feb3c2d6bfc6649666d2c845
              slot: pair:3
              record_digest: 9db9effa834a8395ced3fca85904c86e247870028cf307bc6cdf6223847a04fd
              start: "2026-10-21T02:29:55.344413Z"
              end: "2026-10-21T02:41:28.22562Z"
              queue_seconds: 692.881207
              raw_seconds: 692.881207
              exact: true
            - attempt_id: c1312d85288ac119c62be2def139c5f5
              slot: pair:3
              record_digest: c158ae1e645bc5757e0b4c410af9f7c4ad0a54e6cfe6a9806dc786354732d4ac
              start: "2026-10-21T02:49:40.716634Z"
              end: "2026-10-21T03:15:46.771462Z"
              queue_seconds: 1566.054828
              raw_seconds: 1566.054828
              exact: true
            - attempt_id: 47f425cdb75065b0c55278178856c569
              slot: pair:3
              record_digest: 30fc019dfc4af2a803664df10dad7b90adf94aa778103ac44c4bd95506d2d507
              start: "2026-10-21T03:34:31.551086Z"
              end: "2026-10-21T04:01:29.338653Z"
              queue_seconds: 1617.787567
              raw_seconds: 1617.787567
              exact: true
        protected:
            - issue: "748"
              slot: pair:3
              start: "2026-10-21T00:33:59.272Z"
              end: "2026-10-21T00:55:02.534Z"
            - issue: "748"
              slot: pair:3
              start: "2026-10-21T00:34:30.891Z"
              end: "2026-10-21T00:53:04.867Z"
            - issue: "748"
              slot: pair:3
              start: "2026-10-21T02:07:50.64Z"
              end: "2026-10-20T19:20:01-07:00"
            - issue: "748"
              slot: pair:3
              start: "2026-10-20T19:20:01-07:00"
              end: "2026-10-20T19:29:24-07:00"
            - issue: "748"
              slot: pair:3
              start: "2026-10-20T19:35:32-07:00"
              end: "2026-10-21T02:48:09.597Z"
            - issue: "748"
              slot: pair:3
              start: "2026-10-21T02:09:02.922Z"
              end: "2026-10-21T02:14:52.42Z"
      "n": 1
      timestamp: "2026-10-20T22:03:34-07:00"
      dispatched: "2026-10-20T22:00:09-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Cancelled provisioning can still activate the new image
          detail: 'cmd/internal/vmcmd/provision.go:87-100 decides sealing and activation from collected/executed/stopConfirmed flags, without the RunState cancellation outcome. Cancel during successful Stop: runtime.go:131-133 records cancellation, but Release receives the detached cleanup context and can still SaveImage and publish the manifest. This violates the failed-provision-preserves-active-image contract and ARCH-ORDER. Make activation eligibility an explicit state-owned decision; add a controlled cancellation regression asserting nonzero exit, unchanged active manifest, and no activation.'
          family: activation-requires-success-authority
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Image publication can install an unreadable active manifest
          detail: cmd/internal/vmcmd/provision.go:212-229 writes and renames encoded image metadata without enforcing the 64 KiB limit imposed by readMachineImage at lines 113-119. An individually valid active manifest with large preserved unknown metadata, combined with an individually valid new tool report, can exceed that limit. Provisioning then succeeds while subsequent runs reject image.json. Enforce the shared size limit before publication, preserving the previous manifest on overflow; test exact-limit and over-limit publication with preserved unknown fields. ARCH-CONSTRAINTS.
          family: persisted-writer-reader-bounds
          round: 1
        - kind: defect
          id: BR-3
          severity: Critical
          title: M1 core-concepts locations contradict the delivered files
          detail: workshop/plans/000743-local-vm-conformance-plan.md:59-60 names runtime_darwin.go, runtime_other.go, and positions.go, which do not exist at the pinned head. Native runtime responsibilities are implemented in host.go and driver_darwin.go/driver_other.go; positions are in positions_unix.go/positions_unsupported.go. Correct the greppable table and add a Revisions entry mapping these M1 entities to their actual files. This is Critical under the requested Core concepts consistency rule; the functionality itself is present.
          family: core-concept-code-traceability
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#743 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T22:03:34-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `activation-requires-success-authority` (kind: defect) Cancelled provisioning can still activate the new image
  cmd/internal/vmcmd/provision.go:87-100 decides sealing and activation from collected/executed/stopConfirmed flags, without the RunState cancellation outcome. Cancel during successful Stop: runtime.go:131-133 records cancellation, but Release receives the detached cleanup context and can still SaveImage and publish the manifest. This violates the failed-provision-preserves-active-image contract and ARCH-ORDER. Make activation eligibility an explicit state-owned decision; add a controlled cancellation regression asserting nonzero exit, unchanged active manifest, and no activation.
- **BR-2** [Critical] `persisted-writer-reader-bounds` (kind: defect) Image publication can install an unreadable active manifest
  cmd/internal/vmcmd/provision.go:212-229 writes and renames encoded image metadata without enforcing the 64 KiB limit imposed by readMachineImage at lines 113-119. An individually valid active manifest with large preserved unknown metadata, combined with an individually valid new tool report, can exceed that limit. Provisioning then succeeds while subsequent runs reject image.json. Enforce the shared size limit before publication, preserving the previous manifest on overflow; test exact-limit and over-limit publication with preserved unknown fields. ARCH-CONSTRAINTS.
- **BR-3** [Critical] `core-concept-code-traceability` (kind: defect) M1 core-concepts locations contradict the delivered files
  workshop/plans/000743-local-vm-conformance-plan.md:59-60 names runtime_darwin.go, runtime_other.go, and positions.go, which do not exist at the pinned head. Native runtime responsibilities are implemented in host.go and driver_darwin.go/driver_other.go; positions are in positions_unix.go/positions_unsupported.go. Correct the greppable table and add a Revisions entry mapping these M1 entities to their actual files. This is Critical under the requested Core concepts consistency rule; the functionality itself is present.

## Open findings

- **BR-1** [Critical] `activation-requires-success-authority` (kind: defect) Cancelled provisioning can still activate the new image
- **BR-2** [Critical] `persisted-writer-reader-bounds` (kind: defect) Image publication can install an unreadable active manifest
- **BR-3** [Critical] `core-concept-code-traceability` (kind: defect) M1 core-concepts locations contradict the delivered files
