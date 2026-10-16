---
gate: boundary-review
issue: 583
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 495b4eaf-50ee-4ed6-bbf5-df138e05b085
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/96dbfd4db73a2e74
            reviewed_head: 95db2c14bf53651a4f6d9c7d249480808de89ec0
            slot: pair:4
            requested_at: "2026-10-16T20:40:18.634276Z"
          phase: finished
          pid: 30392
          started_at: "2026-10-16T20:40:18.768151Z"
          finished_at: "2026-10-16T20:44:03.690059Z"
          session_id: 26785c50-cd8e-43da-9092-614db9ec4874
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 95db2c14bf53651a4f6d9c7d249480808de89ec0
        tracker_head: 4271b0ad47f61b78ea866e78abe804eb754ebdb3
        branch_point: 758c4b72216ed740e7167f241e8287d2d6db25c6
        since: "2026-10-16T10:00:03-07:00"
        until: "2026-10-16T13:37:21-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T10:00:03-07:00"
              end: "2026-10-16T13:37:21-07:00"
              began: "2026-10-16T10:00:03-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: claude
              id: 1ff324a8-f519-463e-91c6-eed2bb7295d3
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: bb690dfb3f054cd9b6e7b831e86003e242e20023175ac2b6fcd0e2bc4fcc0f1a
              started: "2026-10-16T18:00:34.517Z"
              spawned: false
              bytes: 613947
              windows:
                - start: "2026-10-16T18:00:34.517Z"
                  end: "2026-10-16T13:37:21-07:00"
            - harness: claude
              id: 5c6faf1f-ecbe-4e42-8a3e-5f18673a4399
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4c80faadfd6cff7705e00943475d8280d3c4da71743a4c89aa86ed51ff212500
              started: "2026-10-16T17:35:33.908Z"
              spawned: false
              bytes: 672305
              windows:
                - start: "2026-10-16T17:35:33.908Z"
                  end: "2026-10-16T13:37:21-07:00"
            - harness: claude
              id: a845f300-2968-4cb8-b745-15e1c37f59df
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ff595bab0159fd79c6c095b041ee34276cfb920a34245a085adc1c1d219baa99
              started: "2026-10-16T18:01:06.246Z"
              spawned: false
              bytes: 399901
              windows:
                - start: "2026-10-16T18:01:06.246Z"
                  end: "2026-10-16T13:37:21-07:00"
            - harness: claude
              id: ea13117c-335f-4574-9a74-7f6e505c13e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 837a5683d4857e30ba3acb3f625e5c12de3abc79988376d6d02418c2ead4e222
              started: "2026-10-16T17:52:16.809Z"
              spawned: false
              bytes: 623751
              windows:
                - start: "2026-10-16T17:52:16.809Z"
                  end: "2026-10-16T13:37:21-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: bd533a109ff594aa705b72791111cefc750259f09b4dffadd42ad48f95b076ee
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 9575867
              windows:
                - start: "2026-10-16T10:00:03-07:00"
                  end: "2026-10-16T13:37:21-07:00"
      "n": 1
      timestamp: "2026-10-16T13:44:04-07:00"
      dispatched: "2026-10-16T13:40:16-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: The rule "live owner but reparented to launchd → kill" is never reached for tag-keyed kinds; the sweep drops live tags
          detail: 'deadTagSidecars (nvim_reap_os.go:159) and argvOnlyEmbeds (:232) both drop live tags, and the quit path passes OwnerLive:false, so no draft/scrollback/review row ever has OwnerLive:true and reapReasonReparented only fires for changelog viewers and the hand-built test target at nvim_reap_os_test.go:170. Plan Task 2 names this "the closed-pane case" and Task 5 says OwnerLive is live[tag]. Fix: emit live-tag rows with OwnerLive:true (the classifier still requires ppid 1 and full argv attribution) and add a sweep case, or add a plan revision withdrawing the claim.'
          family: plan-claim-unreachable-in-production
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: A failed openlock write lets another session's sweep reap a live changelog viewer
          detail: openerLive treats an absent .openlock as a dead owner, but opener/run.go ignores the WriteFile error on the openlock. Consider also checking that the viewer's grandparent pid is the opener before killing.
          family: absent-evidence-read-as-owner-death
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Quit-path ReapNvim lost its argv fallback; an overwritten scrollback sidecar's viewer waits for the next launch sweep
          detail: The old pattern kill caught every scrollback viewer of the tag at quit (and over-matched); the new path reads only pid records. Document this in the plan's Non-goals or Revisions.
          family: quit-path-coverage-narrowed
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: byPID is built twice per sweep and Command(pid) is read twice for each argv-only row
          detail: sweepOrphanNvim:133 and reapNvimRows:287 each build the map, and argvOnlyEmbeds:227 and decideNvimSidecar:355 each read argv. Pass the map through and keep the argv already read.
          family: repeated-per-pass-work
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 1e65fd4f-c96f-4bbe-8e52-f3d6737efd28
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/e3c9306e50bcc941
            reviewed_head: 42045692d9a2a272484f7ee9af80f882035ab67a
            slot: pair:4
            requested_at: "2026-10-16T21:05:16.860739Z"
          phase: finished
          pid: 92494
          started_at: "2026-10-16T21:05:16.999741Z"
          finished_at: "2026-10-16T21:07:23.811937Z"
          session_id: bb63606c-29e5-442f-892d-59ec33dd8347
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 42045692d9a2a272484f7ee9af80f882035ab67a
        tracker_head: d2aba2205e2cca653ac0d1a1f86fad094b6a1e16
        branch_point: 758c4b72216ed740e7167f241e8287d2d6db25c6
        since: "2026-10-16T10:00:03-07:00"
        until: "2026-10-16T14:04:14-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T10:00:03-07:00"
              end: "2026-10-16T14:04:14-07:00"
              began: "2026-10-16T10:00:03-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: claude
              id: 1ff324a8-f519-463e-91c6-eed2bb7295d3
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: bb690dfb3f054cd9b6e7b831e86003e242e20023175ac2b6fcd0e2bc4fcc0f1a
              started: "2026-10-16T18:00:34.517Z"
              spawned: false
              bytes: 613947
              windows:
                - start: "2026-10-16T18:00:34.517Z"
                  end: "2026-10-16T14:04:14-07:00"
            - harness: claude
              id: 5c6faf1f-ecbe-4e42-8a3e-5f18673a4399
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4c80faadfd6cff7705e00943475d8280d3c4da71743a4c89aa86ed51ff212500
              started: "2026-10-16T17:35:33.908Z"
              spawned: false
              bytes: 672305
              windows:
                - start: "2026-10-16T17:35:33.908Z"
                  end: "2026-10-16T14:04:14-07:00"
            - harness: claude
              id: a845f300-2968-4cb8-b745-15e1c37f59df
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ff595bab0159fd79c6c095b041ee34276cfb920a34245a085adc1c1d219baa99
              started: "2026-10-16T18:01:06.246Z"
              spawned: false
              bytes: 399901
              windows:
                - start: "2026-10-16T18:01:06.246Z"
                  end: "2026-10-16T14:04:14-07:00"
            - harness: claude
              id: ea13117c-335f-4574-9a74-7f6e505c13e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 837a5683d4857e30ba3acb3f625e5c12de3abc79988376d6d02418c2ead4e222
              started: "2026-10-16T17:52:16.809Z"
              spawned: false
              bytes: 623751
              windows:
                - start: "2026-10-16T17:52:16.809Z"
                  end: "2026-10-16T14:04:14-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 39da18279c7199702cb00b697a8bc196578375fafd5f94143f50596140229e5d
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 10641918
              windows:
                - start: "2026-10-16T10:00:03-07:00"
                  end: "2026-10-16T14:04:14-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-16T13:40:16-07:00"
              finalized: "2026-10-16T13:44:04-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T14:07:24-07:00"
      dispatched: "2026-10-16T14:05:15-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: tagSidecars/argvOnlyEmbeds now pass live[tag]; reverting the two filters turns the two new reparented sweep cases red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The openlock write is checked (run.go:197) and TestRunChangelogRefusesAViewerItCannotClaim goes red (code = 0, want 1) on revert; the regression it introduced is raised separately below.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Recorded in the plan Non-goals at :551-555; the prose matches ReapNvim, which reads pid records only (osruntime.go:946).
          round: 2
        - id: BR-4
          disposition: addressed
          note: byPID is built once (indexByPID); the second argv read is justified as authorizing evidence read beside the identity fence, with the reason commented at nvim_reap_os.go:336.
          round: 2
      findings:
        - kind: defect
          id: BR-5
          severity: Critical
          title: The BR-2 fix's stderr string breaks TestProductionArtifactReferencesAreExactlyClassified; the suite is red at head
          detail: 'cmd/internal/opener/run.go:194 adds "pair-changelog-open: openlock: %v\n", which artifactpath/coverage_test.go:205 rejects as unlisted changelog vocabulary; the test passes with run.go reverted to bff93e1e9^. Reword the message without the artifact token, e.g. "claim viewer", and run the full cmd/internal suite. The rule: a fix commit is verified against every package''s tests, not only the new targeted test.'
          family: fix-verified-by-targeted-test-only
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c21e99db-6c8a-4925-be27-23b1ffc2ca14
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/afeb1ecc6a482a45
            reviewed_head: 283e2314a2baae3ace0e1e981c1f034e2fd5d4cc
            slot: pair:4
            requested_at: "2026-10-16T21:20:31.703494Z"
          phase: finished
          pid: 31489
          started_at: "2026-10-16T21:20:31.889529Z"
          finished_at: "2026-10-16T21:36:48.883475Z"
          session_id: 85886e49-ec5a-4130-861f-c0df95e294b7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 283e2314a2baae3ace0e1e981c1f034e2fd5d4cc
        tracker_head: d2aba2205e2cca653ac0d1a1f86fad094b6a1e16
        branch_point: 758c4b72216ed740e7167f241e8287d2d6db25c6
        since: "2026-10-16T10:00:03-07:00"
        until: "2026-10-16T14:19:58-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T10:00:03-07:00"
              end: "2026-10-16T14:19:58-07:00"
              began: "2026-10-16T10:00:03-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: claude
              id: 1ff324a8-f519-463e-91c6-eed2bb7295d3
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: bb690dfb3f054cd9b6e7b831e86003e242e20023175ac2b6fcd0e2bc4fcc0f1a
              started: "2026-10-16T18:00:34.517Z"
              spawned: false
              bytes: 613947
              windows:
                - start: "2026-10-16T18:00:34.517Z"
                  end: "2026-10-16T14:19:58-07:00"
            - harness: claude
              id: 5c6faf1f-ecbe-4e42-8a3e-5f18673a4399
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4c80faadfd6cff7705e00943475d8280d3c4da71743a4c89aa86ed51ff212500
              started: "2026-10-16T17:35:33.908Z"
              spawned: false
              bytes: 672305
              windows:
                - start: "2026-10-16T17:35:33.908Z"
                  end: "2026-10-16T14:19:58-07:00"
            - harness: claude
              id: a845f300-2968-4cb8-b745-15e1c37f59df
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ff595bab0159fd79c6c095b041ee34276cfb920a34245a085adc1c1d219baa99
              started: "2026-10-16T18:01:06.246Z"
              spawned: false
              bytes: 399901
              windows:
                - start: "2026-10-16T18:01:06.246Z"
                  end: "2026-10-16T14:19:58-07:00"
            - harness: claude
              id: ea13117c-335f-4574-9a74-7f6e505c13e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 837a5683d4857e30ba3acb3f625e5c12de3abc79988376d6d02418c2ead4e222
              started: "2026-10-16T17:52:16.809Z"
              spawned: false
              bytes: 623751
              windows:
                - start: "2026-10-16T17:52:16.809Z"
                  end: "2026-10-16T14:19:58-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 22e3e9537b4ce7350856dbe2ca530f38fb513892de6348773bcc5e8809b86a49
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 11250984
              windows:
                - start: "2026-10-16T10:00:03-07:00"
                  end: "2026-10-16T14:19:58-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-16T13:40:16-07:00"
              finalized: "2026-10-16T13:44:04-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-16T14:05:15-07:00"
              finalized: "2026-10-16T14:07:24-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-16T14:36:49-07:00"
      dispatched: "2026-10-16T14:20:30-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: 'manifest.go lists "pair-changelog-open: write %s: %v\n"; reverting manifest.go to 283e2314a^ turns TestProductionArtifactReferencesAreExactlyClassified red, and it is green at head; every touched package passes.'
          round: 3
      findings:
        - kind: defect
          id: BR-6
          severity: Important
          title: The review kind's argv check matches any tag's review pane, so a stale review record can kill another live tag's pane after pid reuse
          detail: 'nvim_reap_os.go:101 sets ExpectedAssetBase "review.lua", which every pair review embed carries. Pid records store no start stamp, so the identity fence (nvim_reap_os.go:368) only catches reuse after the snapshot. Failure: a dead tag''s surviving nvim-pid-A-review names a pid that tag B''s live review embed now holds; the sweep reads owner A as dead, every check passes, and B''s pane is SIGKILLed. Fix: attribute review to its owner (argv must carry the file in the tag''s own review state), or store and compare the start stamp in the record. Add a cross-tag reuse regression test.'
          family: kill-attribution-must-name-owner
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 621c58ec-0a1a-49db-9776-37749417dd44
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/4598158c594d57c4
            reviewed_head: b2c4283f9abffcd28acd487636e04583c9af7b8b
            slot: pair:4
            requested_at: "2026-10-16T22:18:45.001104Z"
          phase: finished
          pid: 50387
          started_at: "2026-10-16T22:18:45.786389Z"
          finished_at: "2026-10-16T22:21:52.184304Z"
          session_id: 1f29e64f-c400-4d16-9fee-663a1cb3e426
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b2c4283f9abffcd28acd487636e04583c9af7b8b
        tracker_head: 3c4253905905a8e960b150e25e9b48340c6f4068
        branch_point: 758c4b72216ed740e7167f241e8287d2d6db25c6
        since: "2026-10-16T10:00:03-07:00"
        until: "2026-10-16T15:16:57-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T10:00:03-07:00"
              end: "2026-10-16T15:16:57-07:00"
              began: "2026-10-16T10:00:03-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: claude
              id: 1ff324a8-f519-463e-91c6-eed2bb7295d3
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: bb690dfb3f054cd9b6e7b831e86003e242e20023175ac2b6fcd0e2bc4fcc0f1a
              started: "2026-10-16T18:00:34.517Z"
              spawned: false
              bytes: 613947
              windows:
                - start: "2026-10-16T18:00:34.517Z"
                  end: "2026-10-16T15:16:57-07:00"
            - harness: claude
              id: 5c6faf1f-ecbe-4e42-8a3e-5f18673a4399
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4c80faadfd6cff7705e00943475d8280d3c4da71743a4c89aa86ed51ff212500
              started: "2026-10-16T17:35:33.908Z"
              spawned: false
              bytes: 672305
              windows:
                - start: "2026-10-16T17:35:33.908Z"
                  end: "2026-10-16T15:16:57-07:00"
            - harness: claude
              id: a845f300-2968-4cb8-b745-15e1c37f59df
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ff595bab0159fd79c6c095b041ee34276cfb920a34245a085adc1c1d219baa99
              started: "2026-10-16T18:01:06.246Z"
              spawned: false
              bytes: 399901
              windows:
                - start: "2026-10-16T18:01:06.246Z"
                  end: "2026-10-16T15:16:57-07:00"
            - harness: claude
              id: ea13117c-335f-4574-9a74-7f6e505c13e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 837a5683d4857e30ba3acb3f625e5c12de3abc79988376d6d02418c2ead4e222
              started: "2026-10-16T17:52:16.809Z"
              spawned: false
              bytes: 623751
              windows:
                - start: "2026-10-16T17:52:16.809Z"
                  end: "2026-10-16T15:16:57-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 3e846171966a20a808ec417ede0fce1ccb174183f5e73c6f93c081d238415357
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 12845225
              windows:
                - start: "2026-10-16T10:00:03-07:00"
                  end: "2026-10-16T15:16:57-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 15.0931
        cuts:
            - boundary: M1
              dispatched: "2026-10-16T13:40:16-07:00"
              finalized: "2026-10-16T13:44:04-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-16T14:05:15-07:00"
              finalized: "2026-10-16T14:07:24-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-16T14:20:30-07:00"
              finalized: "2026-10-16T14:36:49-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-16T15:21:57-07:00"
      dispatched: "2026-10-16T15:18:38-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: Cross-tag kill with distinct files now skips. The sweep row with a live parent fails without the fix (asset base plus dead owner used to mean KILL). The remaining same-file pid-reuse case should be recorded in the plan's Revisions.
          round: 4
      findings:
        - kind: defect
          id: BR-7
          severity: Critical
          title: Review attribution reads a mutable review-target, so a retargeted pane's own orphan is never killed and its record is deleted
          detail: 'nvim_reap_os.go:358 (reviewAttribution) sets ExpectedArgPath from review-target-<tag>.json, but restore_client.lua:137 (Alt+C activation of a live pane) and init.lua propose (:PairReview) both rewrite that file while the pane''s argv still names its spawn file. A reparented or quit-path embed therefore classifies "argv serves another path", reapKeepsRecord drops the record, and no later pass can find the pid (review has no argv-only fallback). That is the issue''s orphan class, reopened. This is the 2nd finding in this family. The rule: attribution evidence must be bound to the pid when the record is written and must not change for its lifetime; a mismatch against mutable state keeps the record and never drops it. Fix: review.lua writes its served file into the pid record (line 2) at publish, the reaper compares process argv with that line, and a legacy one-line record is attribution unknown (kept). Add sweep and quit rows: a reparented embed after a retarget must be KILLed; a cross-tag live pane with a different file must be skipped.'
          family: kill-attribution-must-name-owner
          round: 4
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d9f387c4-9145-4676-9d33-83e73cdabd12
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: milestone-review
            round: 5
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0c245756041611c0
            reviewed_head: 25ae5ed5385381dde0c9cab7e788e147bf5f0b8a
            slot: pair:4
            requested_at: "2026-10-16T23:05:16.659949Z"
          phase: finished
          pid: 72053
          started_at: "2026-10-16T23:05:16.931938Z"
          finished_at: "2026-10-16T23:18:06.411263Z"
          session_id: b5dcb61d-5ea2-4c51-9e18-88997eb4276d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 25ae5ed5385381dde0c9cab7e788e147bf5f0b8a
        tracker_head: b964604c9088c1b178a46a61b114cfdc44df80ea
        branch_point: 758c4b72216ed740e7167f241e8287d2d6db25c6
        since: "2026-10-16T10:00:03-07:00"
        until: "2026-10-16T16:03:55-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T10:00:03-07:00"
              end: "2026-10-16T16:03:55-07:00"
              began: "2026-10-16T10:00:03-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: claude
              id: 1ff324a8-f519-463e-91c6-eed2bb7295d3
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: bb690dfb3f054cd9b6e7b831e86003e242e20023175ac2b6fcd0e2bc4fcc0f1a
              started: "2026-10-16T18:00:34.517Z"
              spawned: false
              bytes: 613947
              windows:
                - start: "2026-10-16T18:00:34.517Z"
                  end: "2026-10-16T16:03:55-07:00"
            - harness: claude
              id: 5c6faf1f-ecbe-4e42-8a3e-5f18673a4399
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 4c80faadfd6cff7705e00943475d8280d3c4da71743a4c89aa86ed51ff212500
              started: "2026-10-16T17:35:33.908Z"
              spawned: false
              bytes: 672305
              windows:
                - start: "2026-10-16T17:35:33.908Z"
                  end: "2026-10-16T16:03:55-07:00"
            - harness: claude
              id: a845f300-2968-4cb8-b745-15e1c37f59df
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ff595bab0159fd79c6c095b041ee34276cfb920a34245a085adc1c1d219baa99
              started: "2026-10-16T18:01:06.246Z"
              spawned: false
              bytes: 399901
              windows:
                - start: "2026-10-16T18:01:06.246Z"
                  end: "2026-10-16T16:03:55-07:00"
            - harness: claude
              id: ea13117c-335f-4574-9a74-7f6e505c13e2
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 837a5683d4857e30ba3acb3f625e5c12de3abc79988376d6d02418c2ead4e222
              started: "2026-10-16T17:52:16.809Z"
              spawned: false
              bytes: 623751
              windows:
                - start: "2026-10-16T17:52:16.809Z"
                  end: "2026-10-16T16:03:55-07:00"
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 2e7be0e2695d2175691ef524917c04d6c0a2bd37f117e136c3f9ed713dedd456
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 14372598
              windows:
                - start: "2026-10-16T10:00:03-07:00"
                  end: "2026-10-16T16:03:55-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 40.262566666666665
        cuts:
            - boundary: M1
              dispatched: "2026-10-16T13:40:16-07:00"
              finalized: "2026-10-16T13:44:04-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-16T14:05:15-07:00"
              finalized: "2026-10-16T14:07:24-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-16T14:20:30-07:00"
              finalized: "2026-10-16T14:36:49-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-16T15:18:38-07:00"
              finalized: "2026-10-16T15:21:57-07:00"
              finalizing: false
      "n": 5
      timestamp: "2026-10-16T16:18:07-07:00"
      dispatched: "2026-10-16T16:05:12-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: review.lua binds argv's served file into record line 2; reaper reads it via sidecarRecord (nvim_reap_os.go:330-337) and never the review-target state; mutating it out reddens 9 rows including the retarget reap/sweep rows and the cross-tag-live-pane row; lifecycle test pins line 2 (21/21 green).
          round: 5
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 25ae5ed5385381dde0c9cab7e788e147bf5f0b8a
      blocked: false
---

# Gate ledger — pair#583 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T13:44:04-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `plan-claim-unreachable-in-production` (kind: defect) The rule "live owner but reparented to launchd → kill" is never reached for tag-keyed kinds; the sweep drops live tags
  deadTagSidecars (nvim_reap_os.go:159) and argvOnlyEmbeds (:232) both drop live tags, and the quit path passes OwnerLive:false, so no draft/scrollback/review row ever has OwnerLive:true and reapReasonReparented only fires for changelog viewers and the hand-built test target at nvim_reap_os_test.go:170. Plan Task 2 names this "the closed-pane case" and Task 5 says OwnerLive is live[tag]. Fix: emit live-tag rows with OwnerLive:true (the classifier still requires ppid 1 and full argv attribution) and add a sweep case, or add a plan revision withdrawing the claim.
- **BR-2** [Minor] `absent-evidence-read-as-owner-death` (kind: defect) A failed openlock write lets another session's sweep reap a live changelog viewer
  openerLive treats an absent .openlock as a dead owner, but opener/run.go ignores the WriteFile error on the openlock. Consider also checking that the viewer's grandparent pid is the opener before killing.
- **BR-3** [Minor] `quit-path-coverage-narrowed` (kind: defect) Quit-path ReapNvim lost its argv fallback; an overwritten scrollback sidecar's viewer waits for the next launch sweep
  The old pattern kill caught every scrollback viewer of the tag at quit (and over-matched); the new path reads only pid records. Document this in the plan's Non-goals or Revisions.
- **BR-4** [Minor] `repeated-per-pass-work` (kind: defect) byPID is built twice per sweep and Command(pid) is read twice for each argv-only row
  sweepOrphanNvim:133 and reapNvimRows:287 each build the map, and argvOnlyEmbeds:227 and decideNvimSidecar:355 each read argv. Pass the map through and keep the argv already read.

## Round 2 — 2026-10-16T14:07:24-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — tagSidecars/argvOnlyEmbeds now pass live[tag]; reverting the two filters turns the two new reparented sweep cases red.
- BR-2 — addressed — The openlock write is checked (run.go:197) and TestRunChangelogRefusesAViewerItCannotClaim goes red (code = 0, want 1) on revert; the regression it introduced is raised separately below.
- BR-3 — addressed — Recorded in the plan Non-goals at :551-555; the prose matches ReapNvim, which reads pid records only (osruntime.go:946).
- BR-4 — addressed — byPID is built once (indexByPID); the second argv read is justified as authorizing evidence read beside the identity fence, with the reason commented at nvim_reap_os.go:336.

### Raised

- **BR-5** [Critical] `fix-verified-by-targeted-test-only` (kind: defect) The BR-2 fix's stderr string breaks TestProductionArtifactReferencesAreExactlyClassified; the suite is red at head
  cmd/internal/opener/run.go:194 adds "pair-changelog-open: openlock: %v\n", which artifactpath/coverage_test.go:205 rejects as unlisted changelog vocabulary; the test passes with run.go reverted to bff93e1e9^. Reword the message without the artifact token, e.g. "claim viewer", and run the full cmd/internal suite. The rule: a fix commit is verified against every package's tests, not only the new targeted test.

## Round 3 — 2026-10-16T14:36:49-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — addressed — manifest.go lists "pair-changelog-open: write %s: %v\n"; reverting manifest.go to 283e2314a^ turns TestProductionArtifactReferencesAreExactlyClassified red, and it is green at head; every touched package passes.

### Raised

- **BR-6** [Important] `kill-attribution-must-name-owner` (kind: defect) The review kind's argv check matches any tag's review pane, so a stale review record can kill another live tag's pane after pid reuse
  nvim_reap_os.go:101 sets ExpectedAssetBase "review.lua", which every pair review embed carries. Pid records store no start stamp, so the identity fence (nvim_reap_os.go:368) only catches reuse after the snapshot. Failure: a dead tag's surviving nvim-pid-A-review names a pid that tag B's live review embed now holds; the sweep reads owner A as dead, every check passes, and B's pane is SIGKILLed. Fix: attribute review to its owner (argv must carry the file in the tag's own review state), or store and compare the start stamp in the record. Add a cross-tag reuse regression test.

## Round 4 — 2026-10-16T15:21:57-07:00 (claude) — BLOCKED

### Disposed

- BR-6 — addressed — Cross-tag kill with distinct files now skips. The sweep row with a live parent fails without the fix (asset base plus dead owner used to mean KILL). The remaining same-file pid-reuse case should be recorded in the plan's Revisions.

### Raised

- **BR-7** [Critical] `kill-attribution-must-name-owner` (kind: defect) Review attribution reads a mutable review-target, so a retargeted pane's own orphan is never killed and its record is deleted
  nvim_reap_os.go:358 (reviewAttribution) sets ExpectedArgPath from review-target-<tag>.json, but restore_client.lua:137 (Alt+C activation of a live pane) and init.lua propose (:PairReview) both rewrite that file while the pane's argv still names its spawn file. A reparented or quit-path embed therefore classifies "argv serves another path", reapKeepsRecord drops the record, and no later pass can find the pid (review has no argv-only fallback). That is the issue's orphan class, reopened. This is the 2nd finding in this family. The rule: attribution evidence must be bound to the pid when the record is written and must not change for its lifetime; a mismatch against mutable state keeps the record and never drops it. Fix: review.lua writes its served file into the pid record (line 2) at publish, the reaper compares process argv with that line, and a legacy one-line record is attribution unknown (kept). Add sweep and quit rows: a reparented embed after a retarget must be KILLed; a cross-tag live pane with a different file must be skipped.

## Round 5 — 2026-10-16T16:18:07-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — review.lua binds argv's served file into record line 2; reaper reads it via sidecarRecord (nvim_reap_os.go:330-337) and never the review-target state; mutating it out reddens 9 rows including the retarget reap/sweep rows and the cross-tag-live-pane row; lifecycle test pins line 2 (21/21 green).

## Open findings

(none — every finding has been disposed)
