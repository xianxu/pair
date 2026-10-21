---
gate: boundary-review
issue: 757
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 18741f3c-c1cb-4113-a689-b84b0b3ad3eb
            repository: github.com/xianxu/pair
            issue_id: "000757"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e718e163a11db63a
            reviewed_head: ae22a99112f22f9dc85027198aa406c4483fdfcd
            slot: pair:4
            requested_at: "2026-10-21T20:29:13.518874Z"
          phase: finished
          pid: 49251
          started_at: "2026-10-21T20:29:13.658394Z"
          finished_at: "2026-10-21T20:32:59.709781Z"
          session_id: 01a16027-599f-71d2-b910-6a8a77e7815c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: ae22a99112f22f9dc85027198aa406c4483fdfcd
        tracker_head: aa4b71d18e912477694b9316800adf2e315210f7
        branch_point: 9ff2f5735f544de5b29ee7d8f78a224f424eb83d
        since: "2026-10-21T02:15:21-07:00"
        until: "2026-10-21T13:28:11-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T02:15:21-07:00"
              end: "2026-10-21T13:28:11-07:00"
              began: "2026-10-21T02:15:21-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 93abe5c49c9bde074d66954e7e8b92dd5ea9ec3e2d3f196be0cb271656a081db
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 35330654
              windows:
                - start: "2026-10-21T02:15:21-07:00"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a15f38-25f1-79f3-9db1-1071d0a29a9b
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: ff8f64e8191297b10d46a80efde654361717399e8d41bdc3df784bb2a944f591
              started: "2026-10-21T16:07:57.782Z"
              spawned: true
              bytes: 383764
              windows:
                - start: "2026-10-21T16:07:57.782Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a16008-ca05-7963-9603-c72153753beb
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ee1314e0deb785a9c06d234a1ccc807bce100a1644e45f3cbb7af64adcc67cb8
              started: "2026-10-21T19:55:54.328Z"
              spawned: false
              bytes: 598849
              windows:
                - start: "2026-10-21T19:55:54.328Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a1600a-7e4e-71b2-8c76-8f0bfa2f8b2d
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6c60b167861672aba86f1fecd0f17b4c14ff5500b5ff65a6c7b8b0fa44d94682
              started: "2026-10-21T19:57:43.173Z"
              spawned: false
              bytes: 481712
              windows:
                - start: "2026-10-21T19:57:43.173Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a1600b-f289-78f2-9570-0286d2438127
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 8372274b33994a8a5ee5fc0ab08bdbabdee9340a38d70ec842a931d91b405622
              started: "2026-10-21T19:59:18.589Z"
              spawned: false
              bytes: 504954
              windows:
                - start: "2026-10-21T19:59:18.589Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a1600c-55b4-7971-a4d4-c2d387e752ba
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 21f7b7a9cfb026576ebc8396a074d9fee05b35a2960b30278d39dcf168e05990
              started: "2026-10-21T19:59:43.983Z"
              spawned: false
              bytes: 477930
              windows:
                - start: "2026-10-21T19:59:43.983Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a1600d-4e42-7823-a19b-c8094b380ba7
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: c1e66b3047fd36934d8f71ceefd0adc337119654311868306c9e9e8fd5a22e76
              started: "2026-10-21T20:00:47.259Z"
              spawned: true
              bytes: 572917
              windows:
                - start: "2026-10-21T20:00:47.259Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a1600d-a1c2-7b92-a753-d3be1ad0140d
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: ebb0e9e3034e006faf477e2bf8c2e1a55fe4b57871f838a0ce10693ac908aaa1
              started: "2026-10-21T20:01:08.639Z"
              spawned: true
              bytes: 868070
              windows:
                - start: "2026-10-21T20:01:08.639Z"
                  end: "2026-10-21T13:28:11-07:00"
            - harness: codex
              id: 01a16011-7467-79f2-9cca-1dd595575a5d
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 9011f569a611663fe0499241a7a5011051c6a688dd7519094e2f052473b6c4a5
              started: "2026-10-21T20:05:19.183Z"
              spawned: true
              bytes: 405992
              windows:
                - start: "2026-10-21T20:05:19.183Z"
                  end: "2026-10-21T13:28:11-07:00"
        queue_waits:
            - attempt_id: ec24c6a1b6f269341fbdc898b2fe072b
              slot: pair:4
              record_digest: bb028951b7e5d47bfcc5e08ee637c032f26c7d9781742bec118747a9f06ef4f7
              start: "2026-10-21T09:46:31.308701Z"
              end: "2026-10-21T10:01:10.102902Z"
              queue_seconds: 878.794201
              raw_seconds: 878.794201
              exact: true
            - attempt_id: a9d212d80f05d2b976a7982c02f0e4b7
              slot: pair:4
              record_digest: afd80fb6f2df0afc077e21d86bfc9e2afab231eb7d5f9bb4a3031dbb6480c5d4
              start: "2026-10-21T10:27:58.827424Z"
              end: "2026-10-21T10:57:23.460877Z"
              queue_seconds: 1764.633453
              raw_seconds: 1764.633453
              exact: true
            - attempt_id: 2f50987f350712f204f7dcb148a43db0
              slot: pair:4
              record_digest: 10ed2d6fb6a0badf202a49096748b1cd552ac147969ee63ef00dae8d0bbaaf43
              start: "2026-10-21T11:26:57.678267Z"
              end: "2026-10-21T11:59:59.421095Z"
              queue_seconds: 1981.742828
              raw_seconds: 1981.742828
              exact: true
            - attempt_id: 1a55f78d406d509cfe70c95475da4377
              slot: pair:4
              record_digest: 7515f77d051887e6c9dc2de24b6efe461394dc4c41b54889f230af24c09bbcd1
              start: "2026-10-21T12:21:54.797374Z"
              end: "2026-10-21T12:43:43.876142Z"
              queue_seconds: 1309.078768
              raw_seconds: 1309.078768
              exact: true
            - attempt_id: e4f69c4b0a81d2285f916768dcea91be
              slot: pair:4
              record_digest: e4e385beb872c422692bfc37c375d06893ee6b146dd1872a6cb7ed2f84c7a5f8
              start: "2026-10-21T14:24:57.835684Z"
              end: "2026-10-21T14:33:45.698106Z"
              queue_seconds: 527.862422
              raw_seconds: 527.862422
              exact: true
            - attempt_id: ca03ebafeb90170aa289defa07c94b7f
              slot: pair:4
              record_digest: ffc9e46e0175b421aff424760a06b998785561554b1a84f898403c50c95faf81
              start: "2026-10-21T15:18:46.936044Z"
              end: "2026-10-21T15:42:52.617267Z"
              queue_seconds: 1445.681223
              raw_seconds: 1445.681223
              exact: true
        protected:
            - issue: "762"
              slot: pair:4
              start: "2026-10-21T09:33:09.536Z"
              end: "2026-10-21T09:46:19.355Z"
            - issue: "764"
              slot: pair:4
              start: "2026-10-21T10:15:27.297Z"
              end: "2026-10-21T03:28:21-07:00"
            - issue: "764"
              slot: pair:4
              start: "2026-10-21T03:28:21-07:00"
              end: "2026-10-21T10:31:19.457Z"
            - issue: "766"
              slot: pair:4
              start: "2026-10-21T11:13:39.127Z"
              end: "2026-10-21T04:26:37-07:00"
            - issue: "766"
              slot: pair:4
              start: "2026-10-21T11:59:59.421095Z"
              end: "2026-10-21T12:21:42.581Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:43:53.097Z"
              end: "2026-10-21T07:26:48-07:00"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T07:26:48-07:00"
              end: "2026-10-21T14:30:09.267Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:44:32.144Z"
              end: "2026-10-21T14:06:13.206Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:51:10.911Z"
              end: "2026-10-21T13:52:26.282Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:54:18.197Z"
              end: "2026-10-21T13:54:56.996Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:55:56.787Z"
              end: "2026-10-21T13:56:27.076Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:56:31.855Z"
              end: "2026-10-21T13:57:02.112Z"
      "n": 1
      timestamp: "2026-10-21T13:33:00-07:00"
      dispatched: "2026-10-21T13:29:10-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Preserve root-path semantics when rewriting upstream redirects
          detail: cmd/internal/mockserve/front.go:82–86 clears Scheme and Host without supplying a root path. An upstream Location of http://HOST:PORT becomes empty; http://HOST:PORT?next=1 becomes ?next=1, which a browser resolves against the current front path instead of /. This violates redirect preservation. Preserve the resolved destination when constructing the front-relative reference and add regression cases for empty paths, queries, fragments and leading-double-slash paths.
          family: redirect-reference-preservation
          round: 1
        - id: BR-2
          severity: Critical
          title: Reconcile the Core concepts table with actual purity and implementation paths
          detail: workshop/plans/000757-mock-to-real-plan.md:97 lists catalogue.Read as PURE, but examples/mock-to-real/internal/catalogue/catalogue.go:18–20 consumes an arbitrary mutable io.Reader and propagates its IO errors. The integration row at plan line 108 also names internal/sample, which does not exist; implementation lives in internal/app and internal/buildsample. Under the required Core concepts cross-check these contradictions block admission. Reclassify Read as INTEGRATION or extract a pure byte parser, correct the paths, and record a Revisions entry (ARCH-PURE).
          family: core-concept-traceability
          round: 1
        - id: BR-3
          severity: Important
          title: Compare complete endpoint identity when rejecting self-targets
          detail: cmd/internal/mockserve/front.go:262 rejects any upstream sharing the front's port, although the front binds specifically to 127.0.0.1. An app bound only to [::1]:3000 and a front at 127.0.0.1:3000 are distinct supported loopback endpoints, yet startup rejects them as self-targeting. Compare the pinned destination address and port with the listener endpoint; test distinct-address acceptance alongside genuine self-target rejection (ARCH-SECURE).
          family: endpoint-identity-validation
          round: 1
        - id: BR-4
          severity: Important
          title: Inject the production builder's compiler execution boundary
          detail: examples/mock-to-real/internal/buildsample/build.go invokes the concrete run helper for go build and candidate validation; run constructs exec.CommandContext directly at line 140. No injected executor or stateful compiler fake exists, so Build's compiler failure, partial-output and cancellation orderings cannot be driven deterministically through the production boundary. Add that seam and fake, retaining the existing real-compiler tests as conformance evidence (ARCH-MOCK).
          family: external-dependency-fake-seam
          round: 1
        - kind: unverifiable
          id: BR-5
          severity: Important
          title: Browser navigation qualification could not complete in the review environment
          detail: 'WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run ''^TestDevelopmentFrontBrowser$'' -count=1 -timeout=60s failed twice at front_browser_test.go:97 with cdp: Runtime.evaluate: cdp: read: EOF before validating the first real-page DOM. A third run with TMPDIR=/tmp failed identically. Supply successful pinned-head host evidence for the full navigation test, including the delivered-progress negative control, or repair the browser environment and rerun; these failures do not establish an application defect.'
          family: browser-navigation-qualification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e876a452-1a2c-4ada-9600-30ca6dfa5e18
            repository: github.com/xianxu/pair
            issue_id: "000757"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/db2187b0849e1b21
            reviewed_head: 0e4d26c70c40ef990988a2748c9bd219a42549ba
            slot: pair:4
            requested_at: "2026-10-21T20:44:21.532216Z"
          phase: finished
          pid: 95785
          started_at: "2026-10-21T20:44:21.761731Z"
          finished_at: "2026-10-21T20:46:16.240158Z"
          session_id: 01a16035-351a-7251-9d94-0bfa7e23ae61
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 0e4d26c70c40ef990988a2748c9bd219a42549ba
        tracker_head: aebef2fced50518a7a887cc5b7534b4fbd930e93
        branch_point: 9ff2f5735f544de5b29ee7d8f78a224f424eb83d
        since: "2026-10-21T02:15:21-07:00"
        until: "2026-10-21T13:42:48-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T02:15:21-07:00"
              end: "2026-10-21T13:42:48-07:00"
              began: "2026-10-21T02:15:21-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 8fc0f883606b451557389539118c081fc53bd7547a95345475b89b74df3cb96c
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 36650518
              windows:
                - start: "2026-10-21T02:15:21-07:00"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a15f38-25f1-79f3-9db1-1071d0a29a9b
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: ff8f64e8191297b10d46a80efde654361717399e8d41bdc3df784bb2a944f591
              started: "2026-10-21T16:07:57.782Z"
              spawned: true
              bytes: 383764
              windows:
                - start: "2026-10-21T16:07:57.782Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a16008-ca05-7963-9603-c72153753beb
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ee1314e0deb785a9c06d234a1ccc807bce100a1644e45f3cbb7af64adcc67cb8
              started: "2026-10-21T19:55:54.328Z"
              spawned: false
              bytes: 598849
              windows:
                - start: "2026-10-21T19:55:54.328Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a1600a-7e4e-71b2-8c76-8f0bfa2f8b2d
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6c60b167861672aba86f1fecd0f17b4c14ff5500b5ff65a6c7b8b0fa44d94682
              started: "2026-10-21T19:57:43.173Z"
              spawned: false
              bytes: 481712
              windows:
                - start: "2026-10-21T19:57:43.173Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a1600b-f289-78f2-9570-0286d2438127
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 8372274b33994a8a5ee5fc0ab08bdbabdee9340a38d70ec842a931d91b405622
              started: "2026-10-21T19:59:18.589Z"
              spawned: false
              bytes: 504954
              windows:
                - start: "2026-10-21T19:59:18.589Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a1600c-55b4-7971-a4d4-c2d387e752ba
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 21f7b7a9cfb026576ebc8396a074d9fee05b35a2960b30278d39dcf168e05990
              started: "2026-10-21T19:59:43.983Z"
              spawned: false
              bytes: 477930
              windows:
                - start: "2026-10-21T19:59:43.983Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a1600d-4e42-7823-a19b-c8094b380ba7
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: c1e66b3047fd36934d8f71ceefd0adc337119654311868306c9e9e8fd5a22e76
              started: "2026-10-21T20:00:47.259Z"
              spawned: true
              bytes: 572917
              windows:
                - start: "2026-10-21T20:00:47.259Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a1600d-a1c2-7b92-a753-d3be1ad0140d
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: e954f430ae843b7b5b6aef8bfd11960092bb2ec9f978687166cae609e14cb089
              started: "2026-10-21T20:01:08.639Z"
              spawned: true
              bytes: 998801
              windows:
                - start: "2026-10-21T20:01:08.639Z"
                  end: "2026-10-21T13:42:48-07:00"
            - harness: codex
              id: 01a16011-7467-79f2-9cca-1dd595575a5d
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 9011f569a611663fe0499241a7a5011051c6a688dd7519094e2f052473b6c4a5
              started: "2026-10-21T20:05:19.183Z"
              spawned: true
              bytes: 405992
              windows:
                - start: "2026-10-21T20:05:19.183Z"
                  end: "2026-10-21T13:42:48-07:00"
        queue_waits:
            - attempt_id: ec24c6a1b6f269341fbdc898b2fe072b
              slot: pair:4
              record_digest: bb028951b7e5d47bfcc5e08ee637c032f26c7d9781742bec118747a9f06ef4f7
              start: "2026-10-21T09:46:31.308701Z"
              end: "2026-10-21T10:01:10.102902Z"
              queue_seconds: 878.794201
              raw_seconds: 878.794201
              exact: true
            - attempt_id: a9d212d80f05d2b976a7982c02f0e4b7
              slot: pair:4
              record_digest: afd80fb6f2df0afc077e21d86bfc9e2afab231eb7d5f9bb4a3031dbb6480c5d4
              start: "2026-10-21T10:27:58.827424Z"
              end: "2026-10-21T10:57:23.460877Z"
              queue_seconds: 1764.633453
              raw_seconds: 1764.633453
              exact: true
            - attempt_id: 2f50987f350712f204f7dcb148a43db0
              slot: pair:4
              record_digest: 10ed2d6fb6a0badf202a49096748b1cd552ac147969ee63ef00dae8d0bbaaf43
              start: "2026-10-21T11:26:57.678267Z"
              end: "2026-10-21T11:59:59.421095Z"
              queue_seconds: 1981.742828
              raw_seconds: 1981.742828
              exact: true
            - attempt_id: 1a55f78d406d509cfe70c95475da4377
              slot: pair:4
              record_digest: 7515f77d051887e6c9dc2de24b6efe461394dc4c41b54889f230af24c09bbcd1
              start: "2026-10-21T12:21:54.797374Z"
              end: "2026-10-21T12:43:43.876142Z"
              queue_seconds: 1309.078768
              raw_seconds: 1309.078768
              exact: true
            - attempt_id: e4f69c4b0a81d2285f916768dcea91be
              slot: pair:4
              record_digest: e4e385beb872c422692bfc37c375d06893ee6b146dd1872a6cb7ed2f84c7a5f8
              start: "2026-10-21T14:24:57.835684Z"
              end: "2026-10-21T14:33:45.698106Z"
              queue_seconds: 527.862422
              raw_seconds: 527.862422
              exact: true
            - attempt_id: ca03ebafeb90170aa289defa07c94b7f
              slot: pair:4
              record_digest: ffc9e46e0175b421aff424760a06b998785561554b1a84f898403c50c95faf81
              start: "2026-10-21T15:18:46.936044Z"
              end: "2026-10-21T15:42:52.617267Z"
              queue_seconds: 1445.681223
              raw_seconds: 1445.681223
              exact: true
        protected:
            - issue: "762"
              slot: pair:4
              start: "2026-10-21T09:33:09.536Z"
              end: "2026-10-21T09:46:19.355Z"
            - issue: "764"
              slot: pair:4
              start: "2026-10-21T10:15:27.297Z"
              end: "2026-10-21T03:28:21-07:00"
            - issue: "764"
              slot: pair:4
              start: "2026-10-21T03:28:21-07:00"
              end: "2026-10-21T10:31:19.457Z"
            - issue: "766"
              slot: pair:4
              start: "2026-10-21T11:13:39.127Z"
              end: "2026-10-21T04:26:37-07:00"
            - issue: "766"
              slot: pair:4
              start: "2026-10-21T11:59:59.421095Z"
              end: "2026-10-21T12:21:42.581Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:43:53.097Z"
              end: "2026-10-21T07:26:48-07:00"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T07:26:48-07:00"
              end: "2026-10-21T14:30:09.267Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:44:32.144Z"
              end: "2026-10-21T14:06:13.206Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:51:10.911Z"
              end: "2026-10-21T13:52:26.282Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:54:18.197Z"
              end: "2026-10-21T13:54:56.996Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:55:56.787Z"
              end: "2026-10-21T13:56:27.076Z"
            - issue: "769"
              slot: pair:4
              start: "2026-10-21T13:56:31.855Z"
              end: "2026-10-21T13:57:02.112Z"
        cuts:
            - dispatched: "2026-10-21T13:29:10-07:00"
              finalized: "2026-10-21T13:33:00-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T13:46:17-07:00"
      dispatched: "2026-10-21T13:44:18-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: front_origin.go preserves root/query/fragment and leading-double-slash destinations. TestRealRedirectPreservesResolvedDestination passes; removing the preservation guards in a disposable copy fails all five affected cases.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The active Core concepts integration table now classifies catalogue.Read as bounded io.Reader consumption and names internal/app and internal/buildsample. These match the implementation; Revisions — close round1 records the correction. The obsolete proposal is explicitly nonbinding.
          round: 2
        - id: BR-3
          disposition: addressed
          note: targetsListener compares the same pinned address and port used for dialing. TestFrontSelfTargetUsesCompleteEndpoint covers genuine self-targets and distinct loopback addresses; restoring port-only comparison fails the IPv6 and alternate-IPv4 cases.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Build delegates through the injected executor. Stateful compiler tests exercise partial output, validation ordering, cancellation joining and cleanup. Bypassing the executor in a disposable copy makes both failure-phase regressions fail; real-compiler conformance also passes.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The local browser command again fails at the initial real-page assertion with CDP EOF. Retained browser-pass.log records full navigation success, while browser-progress-negative-control.log records failure at the intended delivered-progress assertion. Their cc545350 code head differs from pinned head only in issue-log and evidence files, so the supplied host measurements answer the previously missing qualification.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 0e4d26c70c40ef990988a2748c9bd219a42549ba
      blocked: false
---

# Gate ledger — pair#757 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T13:33:00-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `redirect-reference-preservation` Preserve root-path semantics when rewriting upstream redirects
  cmd/internal/mockserve/front.go:82–86 clears Scheme and Host without supplying a root path. An upstream Location of http://HOST:PORT becomes empty; http://HOST:PORT?next=1 becomes ?next=1, which a browser resolves against the current front path instead of /. This violates redirect preservation. Preserve the resolved destination when constructing the front-relative reference and add regression cases for empty paths, queries, fragments and leading-double-slash paths.
- **BR-2** [Critical] `core-concept-traceability` Reconcile the Core concepts table with actual purity and implementation paths
  workshop/plans/000757-mock-to-real-plan.md:97 lists catalogue.Read as PURE, but examples/mock-to-real/internal/catalogue/catalogue.go:18–20 consumes an arbitrary mutable io.Reader and propagates its IO errors. The integration row at plan line 108 also names internal/sample, which does not exist; implementation lives in internal/app and internal/buildsample. Under the required Core concepts cross-check these contradictions block admission. Reclassify Read as INTEGRATION or extract a pure byte parser, correct the paths, and record a Revisions entry (ARCH-PURE).
- **BR-3** [Important] `endpoint-identity-validation` Compare complete endpoint identity when rejecting self-targets
  cmd/internal/mockserve/front.go:262 rejects any upstream sharing the front's port, although the front binds specifically to 127.0.0.1. An app bound only to [::1]:3000 and a front at 127.0.0.1:3000 are distinct supported loopback endpoints, yet startup rejects them as self-targeting. Compare the pinned destination address and port with the listener endpoint; test distinct-address acceptance alongside genuine self-target rejection (ARCH-SECURE).
- **BR-4** [Important] `external-dependency-fake-seam` Inject the production builder's compiler execution boundary
  examples/mock-to-real/internal/buildsample/build.go invokes the concrete run helper for go build and candidate validation; run constructs exec.CommandContext directly at line 140. No injected executor or stateful compiler fake exists, so Build's compiler failure, partial-output and cancellation orderings cannot be driven deterministically through the production boundary. Add that seam and fake, retaining the existing real-compiler tests as conformance evidence (ARCH-MOCK).
- **BR-5** [Important] `browser-navigation-qualification` (kind: unverifiable) Browser navigation qualification could not complete in the review environment
  WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run '^TestDevelopmentFrontBrowser$' -count=1 -timeout=60s failed twice at front_browser_test.go:97 with cdp: Runtime.evaluate: cdp: read: EOF before validating the first real-page DOM. A third run with TMPDIR=/tmp failed identically. Supply successful pinned-head host evidence for the full navigation test, including the delivered-progress negative control, or repair the browser environment and rerun; these failures do not establish an application defect.

## Round 2 — 2026-10-21T13:46:17-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — front_origin.go preserves root/query/fragment and leading-double-slash destinations. TestRealRedirectPreservesResolvedDestination passes; removing the preservation guards in a disposable copy fails all five affected cases.
- BR-2 — addressed — The active Core concepts integration table now classifies catalogue.Read as bounded io.Reader consumption and names internal/app and internal/buildsample. These match the implementation; Revisions — close round1 records the correction. The obsolete proposal is explicitly nonbinding.
- BR-3 — addressed — targetsListener compares the same pinned address and port used for dialing. TestFrontSelfTargetUsesCompleteEndpoint covers genuine self-targets and distinct loopback addresses; restoring port-only comparison fails the IPv6 and alternate-IPv4 cases.
- BR-4 — addressed — Build delegates through the injected executor. Stateful compiler tests exercise partial output, validation ordering, cancellation joining and cleanup. Bypassing the executor in a disposable copy makes both failure-phase regressions fail; real-compiler conformance also passes.
- BR-5 — addressed — The local browser command again fails at the initial real-page assertion with CDP EOF. Retained browser-pass.log records full navigation success, while browser-progress-negative-control.log records failure at the intended delivered-progress assertion. Their cc545350 code head differs from pinned head only in issue-log and evidence files, so the supplied host measurements answer the previously missing qualification.

## Open findings

(none — every finding has been disposed)
