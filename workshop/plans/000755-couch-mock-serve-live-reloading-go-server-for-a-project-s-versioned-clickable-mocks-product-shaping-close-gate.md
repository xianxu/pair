---
gate: boundary-review
issue: 755
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8a325c2d-ac07-4f9e-8efb-c570bad9fb2d
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/da1e6dd27e86a5d5
            reviewed_head: 5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d
            slot: pair:4
            requested_at: "2026-10-21T06:18:18.297399Z"
          phase: finished
          pid: 75745
          started_at: "2026-10-21T06:18:18.43672Z"
          finished_at: "2026-10-21T06:23:05.697153Z"
          session_id: 01a15d1c-4f3d-76d3-a0ec-7b5763f68316
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d
        tracker_head: 1f186b33f779b6063716a9ab3401de9226e23af9
        branch_point: 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92
        since: "2026-10-20T22:40:54-07:00"
        until: "2026-10-20T23:17:51-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:54-07:00"
              end: "2026-10-20T23:17:51-07:00"
              began: "2026-10-20T22:40:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: c9fcc62e2f3fe43991463769530d64402867cc5510e7e98a97ba50f5d77a4d76
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 15600169
              windows:
                - start: "2026-10-20T22:40:54-07:00"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15cfa-72c5-7572-8b02-f2056ee548f6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5401f0f3bd86c869cb32d5df44c87f0fd8258346cda55aaceb20581356217caf
              started: "2026-10-21T05:41:19.814Z"
              spawned: true
              bytes: 652919
              windows:
                - start: "2026-10-21T05:41:19.814Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6e1e1b3acdc4e7f6c4b3ee9921c792a7dbc8ab590b6fc7513e8a02c4152ccab9
              started: "2026-10-21T05:55:18.212Z"
              spawned: false
              bytes: 545233
              windows:
                - start: "2026-10-21T05:55:18.212Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d09-e16e-7973-869b-447feb0de7d4
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 7666bc2cfa2b5266548b5bf7684a18adcdc8aa89a2cb40d27161f6eb7b38336f
              started: "2026-10-21T05:58:11.397Z"
              spawned: false
              bytes: 505167
              windows:
                - start: "2026-10-21T05:58:11.397Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0b-c808-77a0-8a74-b058604cf1e8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 775a5ebd6ce47b7647d64baca6dc6d6075af73394146897950b4df76ba0ae6fb
              started: "2026-10-21T06:00:16.2Z"
              spawned: false
              bytes: 518046
              windows:
                - start: "2026-10-21T06:00:16.2Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0c-4f3e-7c23-9eb6-c0e3bfaee8ee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 68125bf8491fa1a9237a8a23b25136ec9a96c11f2527fd3bb588f3a4b2fad9d3
              started: "2026-10-21T06:00:50.659Z"
              spawned: false
              bytes: 447230
              windows:
                - start: "2026-10-21T06:00:50.659Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0d-9ec0-70d1-a196-c8d913b2f3c9
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: a3213c5c1610c6162b59f80e20c949d2754f33f228bb0a53adea5038abd561b8
              started: "2026-10-21T06:02:16.226Z"
              spawned: true
              bytes: 399198
              windows:
                - start: "2026-10-21T06:02:16.226Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d0e-032a-7401-bb22-542ec2559bee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 3d980590c5f4ba3ff9eceab50c9eb0f1ce8e3ce4a2ab2bc2ac0e58587b65bfcc
              started: "2026-10-21T06:02:41.947Z"
              spawned: true
              bytes: 396746
              windows:
                - start: "2026-10-21T06:02:41.947Z"
                  end: "2026-10-20T23:17:51-07:00"
            - harness: codex
              id: 01a15d12-f90c-7480-af77-512cb85c5c24
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: b3af1b23f47d568ec119d8edf4e2a83b475d100a1b6e3d6a8cf9ac9c1b41ac8c
              started: "2026-10-21T06:08:07.079Z"
              spawned: true
              bytes: 545002
              windows:
                - start: "2026-10-21T06:08:07.079Z"
                  end: "2026-10-20T23:17:51-07:00"
      "n": 1
      timestamp: "2026-10-20T23:23:07-07:00"
      dispatched: "2026-10-20T23:18:15-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Ancestor symlinks bypass the private-root HTTP boundary
          detail: 'cmd/internal/mockserve/project.go:90-97 checks only the final path component with Lstat; os.Root.Open follows ancestor symlinks whose targets remain inside the project root. After starting a valid project, creating mock-1/private -> .. and requesting /mock-1/private/design.md returned HTTP 200 with the private root document before the next scan rejected the symlink. ARCH-SECURE: enforce race-safe, no-follow path resolution at the read boundary rather than relying on an earlier scan. Sweep document, asset and scanner reads, and add a deterministic post-initial-scan ancestor-replacement regression that fails without the fix.'
          family: filesystem-route-isolation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 78250bae-041e-45e0-a9e3-c96c7bae9bb2
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/96421f6ff64905db
            reviewed_head: 6026591abfcf2590379f7e149e70df3c0617de6f
            slot: pair:4
            requested_at: "2026-10-21T06:33:20.321511Z"
          phase: finished
          pid: 1179
          started_at: "2026-10-21T06:33:20.460896Z"
          finished_at: "2026-10-21T06:37:25.850331Z"
          session_id: 01a15d2a-1253-7ec3-924c-15e8513605b8
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 6026591abfcf2590379f7e149e70df3c0617de6f
        tracker_head: deb9ee1ae285970fca7df7cb1a4bba599ffa91d3
        branch_point: 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92
        since: "2026-10-20T22:40:54-07:00"
        until: "2026-10-20T23:32:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:54-07:00"
              end: "2026-10-20T23:32:26-07:00"
              began: "2026-10-20T22:40:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 5d8a9a34534ac3890734344a10e511e92a47ecfb53bb2223ce7daa4522085065
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 16115041
              windows:
                - start: "2026-10-20T22:40:54-07:00"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15cfa-72c5-7572-8b02-f2056ee548f6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5401f0f3bd86c869cb32d5df44c87f0fd8258346cda55aaceb20581356217caf
              started: "2026-10-21T05:41:19.814Z"
              spawned: true
              bytes: 652919
              windows:
                - start: "2026-10-21T05:41:19.814Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6e1e1b3acdc4e7f6c4b3ee9921c792a7dbc8ab590b6fc7513e8a02c4152ccab9
              started: "2026-10-21T05:55:18.212Z"
              spawned: false
              bytes: 545233
              windows:
                - start: "2026-10-21T05:55:18.212Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d09-e16e-7973-869b-447feb0de7d4
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 7666bc2cfa2b5266548b5bf7684a18adcdc8aa89a2cb40d27161f6eb7b38336f
              started: "2026-10-21T05:58:11.397Z"
              spawned: false
              bytes: 505167
              windows:
                - start: "2026-10-21T05:58:11.397Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d0b-c808-77a0-8a74-b058604cf1e8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 775a5ebd6ce47b7647d64baca6dc6d6075af73394146897950b4df76ba0ae6fb
              started: "2026-10-21T06:00:16.2Z"
              spawned: false
              bytes: 518046
              windows:
                - start: "2026-10-21T06:00:16.2Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d0c-4f3e-7c23-9eb6-c0e3bfaee8ee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 68125bf8491fa1a9237a8a23b25136ec9a96c11f2527fd3bb588f3a4b2fad9d3
              started: "2026-10-21T06:00:50.659Z"
              spawned: false
              bytes: 447230
              windows:
                - start: "2026-10-21T06:00:50.659Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d0d-9ec0-70d1-a196-c8d913b2f3c9
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: a3213c5c1610c6162b59f80e20c949d2754f33f228bb0a53adea5038abd561b8
              started: "2026-10-21T06:02:16.226Z"
              spawned: true
              bytes: 399198
              windows:
                - start: "2026-10-21T06:02:16.226Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d0e-032a-7401-bb22-542ec2559bee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 3d980590c5f4ba3ff9eceab50c9eb0f1ce8e3ce4a2ab2bc2ac0e58587b65bfcc
              started: "2026-10-21T06:02:41.947Z"
              spawned: true
              bytes: 396746
              windows:
                - start: "2026-10-21T06:02:41.947Z"
                  end: "2026-10-20T23:32:26-07:00"
            - harness: codex
              id: 01a15d12-f90c-7480-af77-512cb85c5c24
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: b3af1b23f47d568ec119d8edf4e2a83b475d100a1b6e3d6a8cf9ac9c1b41ac8c
              started: "2026-10-21T06:08:07.079Z"
              spawned: true
              bytes: 545002
              windows:
                - start: "2026-10-21T06:08:07.079Z"
                  end: "2026-10-20T23:32:26-07:00"
        cuts:
            - dispatched: "2026-10-20T23:18:15-07:00"
              finalized: "2026-10-20T23:23:07-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-20T23:37:26-07:00"
      dispatched: "2026-10-20T23:33:18-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: project.go now pins and verifies directory and file identities across HTTP, selection and scanner reads. Post-scan alias and controlled replacement tests pass. In disposable pinned-head copies, removing directory pinning reproduced HTTP 200 exposure of design.md in both ancestor-alias cases; disabling identity guards failed directory, in-parent file and scanner-file regressions.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: Back/forward-cache restoration permanently disables live reload
          detail: 'cmd/internal/mockserve/reload.js:70-75 sets stopped=true and clears polling on pagehide, but never handles pageshow. A document restored from the browser''s back/forward cache retains that stopped state without rerunning its scripts, so subsequent edits and version switches are never observed. ARCH-ORDER/ARCH-PURPOSE: model suspension and restoration explicitly, resume exactly one poll on restoration, and retain cancellation on subsequent departures. A deterministic test added to the existing JS harness dispatched persisted pagehide/pageshow and failed because zero polling timers remained.'
          family: document-lifecycle-restartability
          round: 2
        - kind: defect
          id: BR-3
          severity: Important
          title: Concurrent selection regression fails under the new identity checks
          detail: 'cmd/internal/mockserve/project_test.go:139-169 assumes every concurrent pointer read succeeds, although a legitimate atomic rename between Lstat and Open now produces "file changed during open: .mock-current". Running go test ./cmd/internal/mockserve -run ''TestUseAtomicSelectionReaders|TestProjectRejectsReplacementDuringOpen|TestHTTPRejectsInRootAncestorLinksAfterScan'' -count=20 reproduced that failure. The fatal exit also skips wg.Wait, allowing project cleanup to close the root while the writer still runs; the same run reported "file already closed". ARCH-ORDER: define the permitted replacement/refusal outcome, exercise it deterministically, assert whole values for successful reads, and join the writer on every exit without weakening confinement.'
          family: concurrency-test-contract
          round: 2
        - kind: unverifiable
          id: BR-4
          severity: Important
          title: Real-browser acceptance behavior could not be qualified in this environment
          detail: 'WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run ''TestMockBrowserReload|TestSampleBrowser'' -count=1 failed all initial DOM checks with CDP read EOF. A separate private-profile Chrome diagnostic exited with signal 5 and "sandbox initialization failed: Operation not permitted". This prevents verification of the planned browser click/edit/switch/recovery behavior; provide passing output for the pinned reviewed head on a host where Chrome can initialize its sandbox, or enable that capability in the review environment. ARCH-PURPOSE.'
          family: browser-qualification-evidence
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: af847332-9ef2-46cb-b622-d83f57b31291
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/f338fd9be858bcae
            reviewed_head: 897dbb6cdc7c683b03f6e99ccc0a1c7decd1ad62
            slot: pair:4
            requested_at: "2026-10-21T06:49:00.415541Z"
          phase: finished
          pid: 43045
          started_at: "2026-10-21T06:49:00.5162Z"
          finished_at: "2026-10-21T06:52:29.481412Z"
          session_id: 01a15d38-6a5f-7dd1-ae46-046dfff405d2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 897dbb6cdc7c683b03f6e99ccc0a1c7decd1ad62
        tracker_head: deb9ee1ae285970fca7df7cb1a4bba599ffa91d3
        branch_point: 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92
        since: "2026-10-20T22:40:54-07:00"
        until: "2026-10-20T23:48:22-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:54-07:00"
              end: "2026-10-20T23:48:22-07:00"
              began: "2026-10-20T22:40:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ac4f6223080617f9289e430c44cf5b58f8870b888cbc0a24f38938e60ccf5871
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 17114026
              windows:
                - start: "2026-10-20T22:40:54-07:00"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15cfa-72c5-7572-8b02-f2056ee548f6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5401f0f3bd86c869cb32d5df44c87f0fd8258346cda55aaceb20581356217caf
              started: "2026-10-21T05:41:19.814Z"
              spawned: true
              bytes: 652919
              windows:
                - start: "2026-10-21T05:41:19.814Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6e1e1b3acdc4e7f6c4b3ee9921c792a7dbc8ab590b6fc7513e8a02c4152ccab9
              started: "2026-10-21T05:55:18.212Z"
              spawned: false
              bytes: 545233
              windows:
                - start: "2026-10-21T05:55:18.212Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d09-e16e-7973-869b-447feb0de7d4
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 7666bc2cfa2b5266548b5bf7684a18adcdc8aa89a2cb40d27161f6eb7b38336f
              started: "2026-10-21T05:58:11.397Z"
              spawned: false
              bytes: 505167
              windows:
                - start: "2026-10-21T05:58:11.397Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d0b-c808-77a0-8a74-b058604cf1e8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 775a5ebd6ce47b7647d64baca6dc6d6075af73394146897950b4df76ba0ae6fb
              started: "2026-10-21T06:00:16.2Z"
              spawned: false
              bytes: 518046
              windows:
                - start: "2026-10-21T06:00:16.2Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d0c-4f3e-7c23-9eb6-c0e3bfaee8ee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 68125bf8491fa1a9237a8a23b25136ec9a96c11f2527fd3bb588f3a4b2fad9d3
              started: "2026-10-21T06:00:50.659Z"
              spawned: false
              bytes: 447230
              windows:
                - start: "2026-10-21T06:00:50.659Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d0d-9ec0-70d1-a196-c8d913b2f3c9
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: a3213c5c1610c6162b59f80e20c949d2754f33f228bb0a53adea5038abd561b8
              started: "2026-10-21T06:02:16.226Z"
              spawned: true
              bytes: 399198
              windows:
                - start: "2026-10-21T06:02:16.226Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d0e-032a-7401-bb22-542ec2559bee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 0272946db24a77027059fa1d2499a9e4300652ebd31f14effb6ccfbfb574c63a
              started: "2026-10-21T06:02:41.947Z"
              spawned: true
              bytes: 554824
              windows:
                - start: "2026-10-21T06:02:41.947Z"
                  end: "2026-10-20T23:48:22-07:00"
            - harness: codex
              id: 01a15d12-f90c-7480-af77-512cb85c5c24
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: b3af1b23f47d568ec119d8edf4e2a83b475d100a1b6e3d6a8cf9ac9c1b41ac8c
              started: "2026-10-21T06:08:07.079Z"
              spawned: true
              bytes: 545002
              windows:
                - start: "2026-10-21T06:08:07.079Z"
                  end: "2026-10-20T23:48:22-07:00"
        cuts:
            - dispatched: "2026-10-20T23:18:15-07:00"
              finalized: "2026-10-20T23:23:07-07:00"
              finalizing: false
            - dispatched: "2026-10-20T23:33:18-07:00"
              finalized: "2026-10-20T23:37:26-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-20T23:52:30-07:00"
      dispatched: "2026-10-20T23:48:58-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-2
          disposition: addressed
          note: The production client handles persisted pageshow with generation/request ownership fencing. All 11 client tests pass; running the current harness against the pre-fix script in memory produces six lifecycle failures, including restoration and stale-completion cases.
          round: 3
        - id: BR-3
          disposition: addressed
          note: TestUseAtomicSelectionReaders accepts only whole values or typed ErrPathChanged refusal and unconditionally joins its writer. TestConcurrentSelectionReadRefusesChangedIdentity deterministically exercises replacement and verifies the settled selection. Twenty race-enabled repetitions of the selection/confinement tests passed.
          round: 3
        - id: BR-4
          disposition: withdrawn
          note: workshop/plans/000755-mock-browser-evidence.md records the exact browser command, passing subtests, exit zero, and sandbox-enabled private-profile qualification on 8fefb5c78182993499354d9227052209570022ea. Git inspection confirms the reviewed head differs only in documentation. Local qualification still fails initial DOM predicates with CDP read EOF; the supplied host measurement answers the requested invariant for identical executable code.
          round: 3
      findings:
        - kind: defect
          id: BR-5
          severity: Important
          title: Interrupted selection writers leave staging files without a reclamation path
          detail: 'cmd/internal/mockserve/project.go:227-244 creates a fresh .mock-current-<nonce>.tmp for every selection, with removal owned only by a defer. Process termination before rename/defer leaves that file permanently; no startup, subsequent writer, or scanner path reclaims abandoned staging files. These files consume the scanner''s 1,024-entry allowance (scan.go:70-72), so repeated interrupted commands eventually prevent serving an otherwise valid project. ARCH-FUNERAL: define bounded, concurrency-safe staging ownership/reclamation, preserve active writers and the published pointer, and add interrupted-writer recovery coverage.'
          family: transient-artifact-reclamation
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 99fb5c24-3b89-4f75-8065-11e1da34a1aa
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/f6904dd752437243
            reviewed_head: 1a41927bc3d9689c62c58d032a8c7bef9b6c69a9
            slot: pair:4
            requested_at: "2026-10-21T07:00:00.660471Z"
          phase: finished
          pid: 84688
          started_at: "2026-10-21T07:00:00.857703Z"
          finished_at: "2026-10-21T07:03:00.080032Z"
          session_id: 01a15d42-7f37-7fc1-b91a-c7c75e6ec32c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1a41927bc3d9689c62c58d032a8c7bef9b6c69a9
        tracker_head: 8ad72e59f205333629047a953ea2e6eaa42cefe1
        branch_point: 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92
        since: "2026-10-20T22:40:54-07:00"
        until: "2026-10-20T23:59:06-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:54-07:00"
              end: "2026-10-20T23:59:06-07:00"
              began: "2026-10-20T22:40:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 9938a7e6dcb353417ee35faddf00413a1e24b0b9089cd6b10911861e59dd82d9
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 17657583
              windows:
                - start: "2026-10-20T22:40:54-07:00"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15cfa-72c5-7572-8b02-f2056ee548f6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5401f0f3bd86c869cb32d5df44c87f0fd8258346cda55aaceb20581356217caf
              started: "2026-10-21T05:41:19.814Z"
              spawned: true
              bytes: 652919
              windows:
                - start: "2026-10-21T05:41:19.814Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6e1e1b3acdc4e7f6c4b3ee9921c792a7dbc8ab590b6fc7513e8a02c4152ccab9
              started: "2026-10-21T05:55:18.212Z"
              spawned: false
              bytes: 545233
              windows:
                - start: "2026-10-21T05:55:18.212Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d09-e16e-7973-869b-447feb0de7d4
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 7666bc2cfa2b5266548b5bf7684a18adcdc8aa89a2cb40d27161f6eb7b38336f
              started: "2026-10-21T05:58:11.397Z"
              spawned: false
              bytes: 505167
              windows:
                - start: "2026-10-21T05:58:11.397Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d0b-c808-77a0-8a74-b058604cf1e8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 775a5ebd6ce47b7647d64baca6dc6d6075af73394146897950b4df76ba0ae6fb
              started: "2026-10-21T06:00:16.2Z"
              spawned: false
              bytes: 518046
              windows:
                - start: "2026-10-21T06:00:16.2Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d0c-4f3e-7c23-9eb6-c0e3bfaee8ee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 68125bf8491fa1a9237a8a23b25136ec9a96c11f2527fd3bb588f3a4b2fad9d3
              started: "2026-10-21T06:00:50.659Z"
              spawned: false
              bytes: 447230
              windows:
                - start: "2026-10-21T06:00:50.659Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d0d-9ec0-70d1-a196-c8d913b2f3c9
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 3b23ab7dc82e8fb206a9ffe0a5d32549fa160a7d69fe7991920bcd1a0b0435e2
              started: "2026-10-21T06:02:16.226Z"
              spawned: true
              bytes: 634480
              windows:
                - start: "2026-10-21T06:02:16.226Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d0e-032a-7401-bb22-542ec2559bee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 0272946db24a77027059fa1d2499a9e4300652ebd31f14effb6ccfbfb574c63a
              started: "2026-10-21T06:02:41.947Z"
              spawned: true
              bytes: 554824
              windows:
                - start: "2026-10-21T06:02:41.947Z"
                  end: "2026-10-20T23:59:06-07:00"
            - harness: codex
              id: 01a15d12-f90c-7480-af77-512cb85c5c24
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: b3af1b23f47d568ec119d8edf4e2a83b475d100a1b6e3d6a8cf9ac9c1b41ac8c
              started: "2026-10-21T06:08:07.079Z"
              spawned: true
              bytes: 545002
              windows:
                - start: "2026-10-21T06:08:07.079Z"
                  end: "2026-10-20T23:59:06-07:00"
        cuts:
            - dispatched: "2026-10-20T23:18:15-07:00"
              finalized: "2026-10-20T23:23:07-07:00"
              finalizing: false
            - dispatched: "2026-10-20T23:33:18-07:00"
              finalized: "2026-10-20T23:37:26-07:00"
              finalizing: false
            - dispatched: "2026-10-20T23:48:58-07:00"
              finalized: "2026-10-20T23:52:30-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-21T00:03:01-07:00"
      dispatched: "2026-10-20T23:59:57-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Identity-checked directory/file reads remain shared by HTTP and scanner paths; ancestor-link and replacement regressions pass.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Generation/request ownership fencing remains reachable; repeated restoration and stale response, body, rejection and timer tests pass.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Concurrent readers accept only complete values or ErrPathChanged, verify the settled value, and join writers; package race tests pass.
          round: 4
        - id: BR-4
          disposition: withdrawn
          note: Retain the prior withdrawal. Recorded sandboxed browser qualification targets 584a03cf62abf80bad6e7041f5426a6fbd1cf906; its diff to the reviewed head changes documentation only.
          round: 4
        - id: BR-5
          disposition: addressed
          note: Fixed staging is reclaimed only under the persistent OS lock and excluded from scan content/caps. The killed-writer recovery regression passes; disabling locking makes it fail. Removing reserved-name scan exclusion independently fails its regression. Both negative controls ran in a disposable copy, subsequently removed.
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: 1a41927bc3d9689c62c58d032a8c7bef9b6c69a9
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5d504d12-915f-4835-85d7-ab49f4f4fb53
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: fix-review
            round: 5
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9f930db67db5a909
            reviewed_head: 438872758e60a6f4212cad05b293f677dabae63e
            slot: pair:4
            requested_at: "2026-10-21T08:39:06.92159Z"
          phase: finished
          pid: 99298
          started_at: "2026-10-21T08:39:07.107061Z"
          finished_at: "2026-10-21T08:41:58.198419Z"
          session_id: 01a15d9d-3a2c-7321-9ec4-2088b00ed4e0
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 438872758e60a6f4212cad05b293f677dabae63e
        tracker_head: 0488795041f347ec4e234200fd5e593513279de8
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-20T22:40:54-07:00"
        until: "2026-10-21T00:47:43-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T22:40:54-07:00"
              end: "2026-10-21T00:47:43-07:00"
              began: "2026-10-20T22:40:54-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: d67a6177cd029548f94b01839848eac17c25043f9a9d403266663fdfdf26405d
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 18952989
              windows:
                - start: "2026-10-20T22:40:54-07:00"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15cfa-72c5-7572-8b02-f2056ee548f6
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 5401f0f3bd86c869cb32d5df44c87f0fd8258346cda55aaceb20581356217caf
              started: "2026-10-21T05:41:19.814Z"
              spawned: true
              bytes: 652919
              windows:
                - start: "2026-10-21T05:41:19.814Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 6e1e1b3acdc4e7f6c4b3ee9921c792a7dbc8ab590b6fc7513e8a02c4152ccab9
              started: "2026-10-21T05:55:18.212Z"
              spawned: false
              bytes: 545233
              windows:
                - start: "2026-10-21T05:55:18.212Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d09-e16e-7973-869b-447feb0de7d4
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 7666bc2cfa2b5266548b5bf7684a18adcdc8aa89a2cb40d27161f6eb7b38336f
              started: "2026-10-21T05:58:11.397Z"
              spawned: false
              bytes: 505167
              windows:
                - start: "2026-10-21T05:58:11.397Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d0b-c808-77a0-8a74-b058604cf1e8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 775a5ebd6ce47b7647d64baca6dc6d6075af73394146897950b4df76ba0ae6fb
              started: "2026-10-21T06:00:16.2Z"
              spawned: false
              bytes: 518046
              windows:
                - start: "2026-10-21T06:00:16.2Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d0c-4f3e-7c23-9eb6-c0e3bfaee8ee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 68125bf8491fa1a9237a8a23b25136ec9a96c11f2527fd3bb588f3a4b2fad9d3
              started: "2026-10-21T06:00:50.659Z"
              spawned: false
              bytes: 447230
              windows:
                - start: "2026-10-21T06:00:50.659Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d0d-9ec0-70d1-a196-c8d913b2f3c9
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 3b23ab7dc82e8fb206a9ffe0a5d32549fa160a7d69fe7991920bcd1a0b0435e2
              started: "2026-10-21T06:02:16.226Z"
              spawned: true
              bytes: 634480
              windows:
                - start: "2026-10-21T06:02:16.226Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d0e-032a-7401-bb22-542ec2559bee
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: 0272946db24a77027059fa1d2499a9e4300652ebd31f14effb6ccfbfb574c63a
              started: "2026-10-21T06:02:41.947Z"
              spawned: true
              bytes: 554824
              windows:
                - start: "2026-10-21T06:02:41.947Z"
                  end: "2026-10-21T00:47:43-07:00"
            - harness: codex
              id: 01a15d12-f90c-7480-af77-512cb85c5c24
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: b3af1b23f47d568ec119d8edf4e2a83b475d100a1b6e3d6a8cf9ac9c1b41ac8c
              started: "2026-10-21T06:08:07.079Z"
              spawned: true
              bytes: 545002
              windows:
                - start: "2026-10-21T06:08:07.079Z"
                  end: "2026-10-21T00:47:43-07:00"
        queue_waits:
            - attempt_id: b6b1d1d560ec5ce9860a56bbe0aa1492
              slot: pair:4
              record_digest: 72ec7c6f54225916035eaffa0abef151ae7a64d9f4628822e3a2fa1c4749903b
              start: "2026-10-21T06:59:32.347001Z"
              end: "2026-10-21T07:34:43.614045Z"
              queue_seconds: 2111.267044
              raw_seconds: 2111.267044
              exact: true
        cuts:
            - dispatched: "2026-10-20T23:18:15-07:00"
              finalized: "2026-10-20T23:23:07-07:00"
              finalizing: false
            - dispatched: "2026-10-20T23:33:18-07:00"
              finalized: "2026-10-20T23:37:26-07:00"
              finalizing: false
            - dispatched: "2026-10-20T23:48:58-07:00"
              finalized: "2026-10-20T23:52:30-07:00"
              finalizing: false
            - dispatched: "2026-10-20T23:59:57-07:00"
              finalized: "2026-10-21T00:03:01-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-21T01:42:00-07:00"
      dispatched: "2026-10-21T01:39:04-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Retain the finalized disposition; filesystem implementation and regression tests are unchanged in this window.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Retain the finalized disposition; reload lifecycle implementation and regression tests are unchanged in this window.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Retain the finalized disposition; concurrent-selection tests are unchanged in this window.
          round: 5
        - id: BR-4
          disposition: withdrawn
          note: Retain the prior withdrawal; this window changes neither browser behavior nor the previously accepted qualification evidence.
          round: 5
        - id: BR-5
          disposition: addressed
          note: Retain the finalized disposition; staging ownership, reclamation, scan exclusions and their regression tests are unchanged in this window.
          round: 5
      recipe: fix-review
      isolation: fenced
      reviewed: 438872758e60a6f4212cad05b293f677dabae63e
      blocked: false
---

# Gate ledger — pair#755 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T23:23:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `filesystem-route-isolation` (kind: defect) Ancestor symlinks bypass the private-root HTTP boundary
  cmd/internal/mockserve/project.go:90-97 checks only the final path component with Lstat; os.Root.Open follows ancestor symlinks whose targets remain inside the project root. After starting a valid project, creating mock-1/private -> .. and requesting /mock-1/private/design.md returned HTTP 200 with the private root document before the next scan rejected the symlink. ARCH-SECURE: enforce race-safe, no-follow path resolution at the read boundary rather than relying on an earlier scan. Sweep document, asset and scanner reads, and add a deterministic post-initial-scan ancestor-replacement regression that fails without the fix.

## Round 2 — 2026-10-20T23:37:26-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — project.go now pins and verifies directory and file identities across HTTP, selection and scanner reads. Post-scan alias and controlled replacement tests pass. In disposable pinned-head copies, removing directory pinning reproduced HTTP 200 exposure of design.md in both ancestor-alias cases; disabling identity guards failed directory, in-parent file and scanner-file regressions.

### Raised

- **BR-2** [Critical] `document-lifecycle-restartability` (kind: defect) Back/forward-cache restoration permanently disables live reload
  cmd/internal/mockserve/reload.js:70-75 sets stopped=true and clears polling on pagehide, but never handles pageshow. A document restored from the browser's back/forward cache retains that stopped state without rerunning its scripts, so subsequent edits and version switches are never observed. ARCH-ORDER/ARCH-PURPOSE: model suspension and restoration explicitly, resume exactly one poll on restoration, and retain cancellation on subsequent departures. A deterministic test added to the existing JS harness dispatched persisted pagehide/pageshow and failed because zero polling timers remained.
- **BR-3** [Important] `concurrency-test-contract` (kind: defect) Concurrent selection regression fails under the new identity checks
  cmd/internal/mockserve/project_test.go:139-169 assumes every concurrent pointer read succeeds, although a legitimate atomic rename between Lstat and Open now produces "file changed during open: .mock-current". Running go test ./cmd/internal/mockserve -run 'TestUseAtomicSelectionReaders|TestProjectRejectsReplacementDuringOpen|TestHTTPRejectsInRootAncestorLinksAfterScan' -count=20 reproduced that failure. The fatal exit also skips wg.Wait, allowing project cleanup to close the root while the writer still runs; the same run reported "file already closed". ARCH-ORDER: define the permitted replacement/refusal outcome, exercise it deterministically, assert whole values for successful reads, and join the writer on every exit without weakening confinement.
- **BR-4** [Important] `browser-qualification-evidence` (kind: unverifiable) Real-browser acceptance behavior could not be qualified in this environment
  WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run 'TestMockBrowserReload|TestSampleBrowser' -count=1 failed all initial DOM checks with CDP read EOF. A separate private-profile Chrome diagnostic exited with signal 5 and "sandbox initialization failed: Operation not permitted". This prevents verification of the planned browser click/edit/switch/recovery behavior; provide passing output for the pinned reviewed head on a host where Chrome can initialize its sandbox, or enable that capability in the review environment. ARCH-PURPOSE.

## Round 3 — 2026-10-20T23:52:30-07:00 (codex) — BLOCKED

### Disposed

- BR-2 — addressed — The production client handles persisted pageshow with generation/request ownership fencing. All 11 client tests pass; running the current harness against the pre-fix script in memory produces six lifecycle failures, including restoration and stale-completion cases.
- BR-3 — addressed — TestUseAtomicSelectionReaders accepts only whole values or typed ErrPathChanged refusal and unconditionally joins its writer. TestConcurrentSelectionReadRefusesChangedIdentity deterministically exercises replacement and verifies the settled selection. Twenty race-enabled repetitions of the selection/confinement tests passed.
- BR-4 — withdrawn — workshop/plans/000755-mock-browser-evidence.md records the exact browser command, passing subtests, exit zero, and sandbox-enabled private-profile qualification on 8fefb5c78182993499354d9227052209570022ea. Git inspection confirms the reviewed head differs only in documentation. Local qualification still fails initial DOM predicates with CDP read EOF; the supplied host measurement answers the requested invariant for identical executable code.

### Raised

- **BR-5** [Important] `transient-artifact-reclamation` (kind: defect) Interrupted selection writers leave staging files without a reclamation path
  cmd/internal/mockserve/project.go:227-244 creates a fresh .mock-current-<nonce>.tmp for every selection, with removal owned only by a defer. Process termination before rename/defer leaves that file permanently; no startup, subsequent writer, or scanner path reclaims abandoned staging files. These files consume the scanner's 1,024-entry allowance (scan.go:70-72), so repeated interrupted commands eventually prevent serving an otherwise valid project. ARCH-FUNERAL: define bounded, concurrency-safe staging ownership/reclamation, preserve active writers and the published pointer, and add interrupted-writer recovery coverage.

## Round 4 — 2026-10-21T00:03:01-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Identity-checked directory/file reads remain shared by HTTP and scanner paths; ancestor-link and replacement regressions pass.
- BR-2 — addressed — Generation/request ownership fencing remains reachable; repeated restoration and stale response, body, rejection and timer tests pass.
- BR-3 — addressed — Concurrent readers accept only complete values or ErrPathChanged, verify the settled value, and join writers; package race tests pass.
- BR-4 — withdrawn — Retain the prior withdrawal. Recorded sandboxed browser qualification targets 584a03cf62abf80bad6e7041f5426a6fbd1cf906; its diff to the reviewed head changes documentation only.
- BR-5 — addressed — Fixed staging is reclaimed only under the persistent OS lock and excluded from scan content/caps. The killed-writer recovery regression passes; disabling locking makes it fail. Removing reserved-name scan exclusion independently fails its regression. Both negative controls ran in a disposable copy, subsequently removed.

## Round 5 — 2026-10-21T01:42:00-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retain the finalized disposition; filesystem implementation and regression tests are unchanged in this window.
- BR-2 — addressed — Retain the finalized disposition; reload lifecycle implementation and regression tests are unchanged in this window.
- BR-3 — addressed — Retain the finalized disposition; concurrent-selection tests are unchanged in this window.
- BR-4 — withdrawn — Retain the prior withdrawal; this window changes neither browser behavior nor the previously accepted qualification evidence.
- BR-5 — addressed — Retain the finalized disposition; staging ownership, reclamation, scan exclusions and their regression tests are unchanged in this window.

## Open findings

(none — every finding has been disposed)
