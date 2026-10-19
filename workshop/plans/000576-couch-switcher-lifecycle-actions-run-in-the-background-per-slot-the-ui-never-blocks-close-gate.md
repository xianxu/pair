---
gate: boundary-review
issue: 576
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: cef4afb4-572f-42b9-976b-8261d302ead8
            repository: github.com/xianxu/pair
            issue_id: "000576"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/426e924c03adc9c0
            reviewed_head: 1fc259181880f60916941eb7a09c88c6d40e9d47
            slot: pair:7
            requested_at: "2026-10-19T00:23:49.351744Z"
          phase: finished
          pid: 32875
          started_at: "2026-10-19T00:23:49.461135Z"
          finished_at: "2026-10-19T00:26:57.732742Z"
          session_id: 01a1518b-0d95-7a43-b1a7-e69a735e41e1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 1fc259181880f60916941eb7a09c88c6d40e9d47
        tracker_head: 0704e4beb830d5d1963c948fc0fc2240346a1bda
        branch_point: 1065d70ca3796af3bb97892eaa99f5c1bcc38e87
        since: "2026-10-17T19:21:23-07:00"
        until: "2026-10-18T17:23:31-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T19:21:23-07:00"
              end: "2026-10-18T17:23:31-07:00"
              began: "2026-10-17T19:21:23-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 492bd2e1ed20426a9787a4497600449bfb751d71a15c57c1262802480c0c018b
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 134340521
              windows:
                - start: "2026-10-17T19:21:23-07:00"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a14cd0-c02f-7483-9015-a1001cc269c0
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 249b2eae98e8ac8af0d639aa6a552610d21a02359a4d2132bc140e16f61b162a
              started: "2026-10-18T02:21:51.643Z"
              spawned: true
              bytes: 943832
              windows:
                - start: "2026-10-18T02:21:51.643Z"
                  end: "2026-10-18T17:23:31-07:00"
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
                  end: "2026-10-18T17:23:31-07:00"
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
                  end: "2026-10-18T17:23:31-07:00"
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
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a15129-04f5-7432-ae9a-7cec38fa6a7d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: ba12e995fef41a60f13884f765039b73c9e42faa949fb5785c4a9c8d7f44769c
              started: "2026-10-18T22:36:45.308Z"
              spawned: true
              bytes: 2654203
              windows:
                - start: "2026-10-18T22:36:45.308Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a15129-f16e-7a71-89df-9152add4970e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6f62e883f3b6dbdd6ca8aee0bd133e96b938bd02d47127b91910e796759930a9
              started: "2026-10-18T22:37:46.272Z"
              spawned: false
              bytes: 608946
              windows:
                - start: "2026-10-18T22:37:46.272Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a1512b-903e-7393-80d7-91aedbac84a8
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 54a49317f3724b5ca5a9ee4c03f5d3cee6abde95a7571886bd8ba28bab56f071
              started: "2026-10-18T22:39:32.52Z"
              spawned: false
              bytes: 460651
              windows:
                - start: "2026-10-18T22:39:32.52Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a1512b-bdb0-7710-b492-5e8530e22d4f
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e348a651f5da59443f89ee1d809b3246fc43dabc8c6afbc9a439d466180bc186
              started: "2026-10-18T22:39:43.649Z"
              spawned: true
              bytes: 2212130
              windows:
                - start: "2026-10-18T22:39:43.649Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a1512d-0e7f-7df2-a078-4ed5b766d023
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 287df6fbcf0d088ff9bd4b74a6ff17216f7277bd6cba5c6fe79e4c102b6dd3f7
              started: "2026-10-18T22:41:10.222Z"
              spawned: false
              bytes: 423650
              windows:
                - start: "2026-10-18T22:41:10.222Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a1512e-9b54-72f3-aa78-6265513a1e4b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a7eca60eb27f42585805aa3f3c1fb43818a70f66d281e9568b70d7111b4e7624
              started: "2026-10-18T22:42:51.805Z"
              spawned: false
              bytes: 432594
              windows:
                - start: "2026-10-18T22:42:51.805Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a1512f-0b3b-7af2-bc39-d419fd807583
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: bf6106215be0554ddb2538c129ed949a440730b216371c5319ffa7937245d400
              started: "2026-10-18T22:43:20.475Z"
              spawned: false
              bytes: 405607
              windows:
                - start: "2026-10-18T22:43:20.475Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a1513f-4f93-7df3-9a5d-a396207d6f3a
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e0990b7d3aa1be48288b8764080e374e4901f724d37b50d5a12405346925dc1f
              started: "2026-10-18T23:01:06.14Z"
              spawned: true
              bytes: 639288
              windows:
                - start: "2026-10-18T23:01:06.14Z"
                  end: "2026-10-18T17:23:31-07:00"
            - harness: codex
              id: 01a15165-7823-7611-91c9-ca9cf32c6e98
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 30ff34ada125807995427395f9d443e339886dcbf780a9960b812078872a6c67
              started: "2026-10-18T23:42:46.934Z"
              spawned: true
              bytes: 601410
              windows:
                - start: "2026-10-18T23:42:46.934Z"
                  end: "2026-10-18T17:23:31-07:00"
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
            - attempt_id: cdcd182f5eafd616f61d77300bb8d421
              slot: pair:7
              record_digest: 4b5c67a55906e6867b1627fc1ccbd3549c3777e0d62b69ce530c0258cfea6f67
              start: "2026-10-18T19:09:05.37459Z"
              end: "2026-10-18T19:45:37.846547Z"
              queue_seconds: 2192.471957
              raw_seconds: 2192.471957
              exact: true
            - attempt_id: 337e96abb8056df574bdefeb29d257a1
              slot: pair:7
              record_digest: c8d3e1ecec22414a0a7a2091e259e0a60a0d223fcc1c5b8b3dd2a94a6e786ace
              start: "2026-10-18T23:45:46.259498Z"
              end: "2026-10-18T23:45:46.31539Z"
              queue_seconds: 0.055892
              raw_seconds: 0.055892
              exact: true
            - attempt_id: 9edee006603c6a4f05d38fb526062d7d
              slot: pair:7
              record_digest: 4762dc13019ac6d7b2b90ad442327c29bf7aa39473be899eae1c1e9d52a9fd13
              start: "2026-10-18T23:58:59.625414Z"
              end: "2026-10-19T00:07:05.371352Z"
              queue_seconds: 485.745938
              raw_seconds: 485.745938
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 121.22536666666666
      "n": 1
      timestamp: "2026-10-18T17:27:42-07:00"
      dispatched: "2026-10-18T17:23:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: A newly listed conversation bypasses its running start reservation
          detail: 'cmd/internal/couchtty/menu_operations.go:121 reserves starts without an exact target under StartPath, but admission at lines 164–165 compares only exact keys. Row lookup and rendering likewise resolve only RowKey or ResultAddress, which is assigned at final completion. If inventory lists the new conversation while start/adoption remains running, its row has no job progress and a second action is admitted instead of returning busy: start. A temporary Go-overlay regression reproduced both failures on the pinned head. Preserve one reservation across temporary-path and authoritative-row identities, and share that resolution across admission, projection, busy selection and address lookup; cover inventory arrival during held adoption. This violates ARCH-ORDER and ARCH-PURPOSE.'
          family: target-ownership-survives-identity-materialization
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4fd4dc87-33e7-497c-8cc7-a2b9742f31bb
            repository: github.com/xianxu/pair
            issue_id: "000576"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/c5371658d78435ad
            reviewed_head: 9f58ddf535ce1cd4c3ad09a97a30bf1c527adef9
            slot: pair:7
            requested_at: "2026-10-19T00:54:39.599527Z"
          phase: finished
          pid: 5496
          started_at: "2026-10-19T00:54:39.890404Z"
          finished_at: "2026-10-19T00:57:03.164571Z"
          session_id: 01a151a7-4a68-7e63-95f4-712cba8fd13f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 9f58ddf535ce1cd4c3ad09a97a30bf1c527adef9
        tracker_head: b995d11356308b513df7b5048432f5474283adb5
        branch_point: 1065d70ca3796af3bb97892eaa99f5c1bcc38e87
        since: "2026-10-17T19:21:23-07:00"
        until: "2026-10-18T17:53:57-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T19:21:23-07:00"
              end: "2026-10-18T17:53:57-07:00"
              began: "2026-10-17T19:21:23-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 41156ee78135c78c6454bb8df7d929d79734a153c70a9bc8beda44540e3ea851
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 136774434
              windows:
                - start: "2026-10-17T19:21:23-07:00"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a14cd0-c02f-7483-9015-a1001cc269c0
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 249b2eae98e8ac8af0d639aa6a552610d21a02359a4d2132bc140e16f61b162a
              started: "2026-10-18T02:21:51.643Z"
              spawned: true
              bytes: 943832
              windows:
                - start: "2026-10-18T02:21:51.643Z"
                  end: "2026-10-18T17:53:57-07:00"
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
                  end: "2026-10-18T17:53:57-07:00"
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
                  end: "2026-10-18T17:53:57-07:00"
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
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a15129-04f5-7432-ae9a-7cec38fa6a7d
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: ba12e995fef41a60f13884f765039b73c9e42faa949fb5785c4a9c8d7f44769c
              started: "2026-10-18T22:36:45.308Z"
              spawned: true
              bytes: 2654203
              windows:
                - start: "2026-10-18T22:36:45.308Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a15129-f16e-7a71-89df-9152add4970e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6f62e883f3b6dbdd6ca8aee0bd133e96b938bd02d47127b91910e796759930a9
              started: "2026-10-18T22:37:46.272Z"
              spawned: false
              bytes: 608946
              windows:
                - start: "2026-10-18T22:37:46.272Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a1512b-903e-7393-80d7-91aedbac84a8
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 54a49317f3724b5ca5a9ee4c03f5d3cee6abde95a7571886bd8ba28bab56f071
              started: "2026-10-18T22:39:32.52Z"
              spawned: false
              bytes: 460651
              windows:
                - start: "2026-10-18T22:39:32.52Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a1512b-bdb0-7710-b492-5e8530e22d4f
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: e348a651f5da59443f89ee1d809b3246fc43dabc8c6afbc9a439d466180bc186
              started: "2026-10-18T22:39:43.649Z"
              spawned: true
              bytes: 2212130
              windows:
                - start: "2026-10-18T22:39:43.649Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a1512d-0e7f-7df2-a078-4ed5b766d023
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 287df6fbcf0d088ff9bd4b74a6ff17216f7277bd6cba5c6fe79e4c102b6dd3f7
              started: "2026-10-18T22:41:10.222Z"
              spawned: false
              bytes: 423650
              windows:
                - start: "2026-10-18T22:41:10.222Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a1512e-9b54-72f3-aa78-6265513a1e4b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a7eca60eb27f42585805aa3f3c1fb43818a70f66d281e9568b70d7111b4e7624
              started: "2026-10-18T22:42:51.805Z"
              spawned: false
              bytes: 432594
              windows:
                - start: "2026-10-18T22:42:51.805Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a1512f-0b3b-7af2-bc39-d419fd807583
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: bf6106215be0554ddb2538c129ed949a440730b216371c5319ffa7937245d400
              started: "2026-10-18T22:43:20.475Z"
              spawned: false
              bytes: 405607
              windows:
                - start: "2026-10-18T22:43:20.475Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a1513f-4f93-7df3-9a5d-a396207d6f3a
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 7a59dd0f2949e30af28dd3bd46163a9d95037a591a0ae897e5991c63141f447a
              started: "2026-10-18T23:01:06.14Z"
              spawned: true
              bytes: 803272
              windows:
                - start: "2026-10-18T23:01:06.14Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a15165-7823-7611-91c9-ca9cf32c6e98
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 30ff34ada125807995427395f9d443e339886dcbf780a9960b812078872a6c67
              started: "2026-10-18T23:42:46.934Z"
              spawned: true
              bytes: 601410
              windows:
                - start: "2026-10-18T23:42:46.934Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a15192-6a0f-7980-b3b9-971241e18a0c
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 46581581019c976a2e6169b205587bb982e061176548c5261e9713d6733d6820
              started: "2026-10-19T00:31:52.471Z"
              spawned: true
              bytes: 1093244
              windows:
                - start: "2026-10-19T00:31:52.471Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a15195-39ca-7df2-b0dd-f1073b3082d4
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 381817a312ba135590439bb5a591375673f8b4ca5d0c0722606ece2971237f2e
              started: "2026-10-19T00:34:57.217Z"
              spawned: false
              bytes: 469022
              windows:
                - start: "2026-10-19T00:34:57.217Z"
                  end: "2026-10-18T17:53:57-07:00"
            - harness: codex
              id: 01a15195-a8a3-7b63-841c-4c490e98c582
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: f6cbe2aac696d2dc3171aefff68b5988f2c38880783ec6f50197e8ca7d652906
              started: "2026-10-19T00:35:25.614Z"
              spawned: false
              bytes: 431504
              windows:
                - start: "2026-10-19T00:35:25.614Z"
                  end: "2026-10-18T17:53:57-07:00"
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
            - attempt_id: cdcd182f5eafd616f61d77300bb8d421
              slot: pair:7
              record_digest: 4b5c67a55906e6867b1627fc1ccbd3549c3777e0d62b69ce530c0258cfea6f67
              start: "2026-10-18T19:09:05.37459Z"
              end: "2026-10-18T19:45:37.846547Z"
              queue_seconds: 2192.471957
              raw_seconds: 2192.471957
              exact: true
            - attempt_id: 337e96abb8056df574bdefeb29d257a1
              slot: pair:7
              record_digest: c8d3e1ecec22414a0a7a2091e259e0a60a0d223fcc1c5b8b3dd2a94a6e786ace
              start: "2026-10-18T23:45:46.259498Z"
              end: "2026-10-18T23:45:46.31539Z"
              queue_seconds: 0.055892
              raw_seconds: 0.055892
              exact: true
            - attempt_id: 9edee006603c6a4f05d38fb526062d7d
              slot: pair:7
              record_digest: 4762dc13019ac6d7b2b90ad442327c29bf7aa39473be899eae1c1e9d52a9fd13
              start: "2026-10-18T23:58:59.625414Z"
              end: "2026-10-19T00:07:05.371352Z"
              queue_seconds: 485.745938
              raw_seconds: 485.745938
              exact: true
            - attempt_id: 648263354a17ffbc53c1040762991a8c
              slot: pair:7
              record_digest: bd524f5920c4fb64b81a046beecafff0004655ca985f96c4f713f0be3a58617f
              start: "2026-10-19T00:39:31.213085Z"
              end: "2026-10-19T00:39:31.26667Z"
              queue_seconds: 0.053585
              raw_seconds: 0.053585
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 121.22536666666666
        cuts:
            - dispatched: "2026-10-18T17:23:47-07:00"
              finalized: "2026-10-18T17:27:42-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T17:57:04-07:00"
      dispatched: "2026-10-18T17:54:36-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Original reservations survive identity materialization through derived aliases shared by admission, row lookup, busy selection and address lookup. Both materialization regressions pass at head; replacing only menu_operations.go with its pre-fix version through a temporary Go overlay makes both fail, including the real Console.Run held-adoption case.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 9f58ddf535ce1cd4c3ad09a97a30bf1c527adef9
      blocked: false
---

# Gate ledger — pair#576 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T17:27:42-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `target-ownership-survives-identity-materialization` (kind: defect) A newly listed conversation bypasses its running start reservation
  cmd/internal/couchtty/menu_operations.go:121 reserves starts without an exact target under StartPath, but admission at lines 164–165 compares only exact keys. Row lookup and rendering likewise resolve only RowKey or ResultAddress, which is assigned at final completion. If inventory lists the new conversation while start/adoption remains running, its row has no job progress and a second action is admitted instead of returning busy: start. A temporary Go-overlay regression reproduced both failures on the pinned head. Preserve one reservation across temporary-path and authoritative-row identities, and share that resolution across admission, projection, busy selection and address lookup; cover inventory arrival during held adoption. This violates ARCH-ORDER and ARCH-PURPOSE.

## Round 2 — 2026-10-18T17:57:04-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Original reservations survive identity materialization through derived aliases shared by admission, row lookup, busy selection and address lookup. Both materialization regressions pass at head; replacing only menu_operations.go with its pre-fix version through a temporary Go overlay makes both fail, including the real Console.Run held-adoption case.

## Open findings

(none — every finding has been disposed)
