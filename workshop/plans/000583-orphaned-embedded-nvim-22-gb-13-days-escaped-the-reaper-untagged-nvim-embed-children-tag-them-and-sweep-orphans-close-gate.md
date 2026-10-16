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

## Open findings

- **BR-1** [Important] `plan-claim-unreachable-in-production` (kind: defect) The rule "live owner but reparented to launchd → kill" is never reached for tag-keyed kinds; the sweep drops live tags
- **BR-2** [Minor] `absent-evidence-read-as-owner-death` (kind: defect) A failed openlock write lets another session's sweep reap a live changelog viewer
- **BR-3** [Minor] `quit-path-coverage-narrowed` (kind: defect) Quit-path ReapNvim lost its argv fallback; an overwritten scrollback sidecar's viewer waits for the next launch sweep
- **BR-4** [Minor] `repeated-per-pass-work` (kind: defect) byPID is built twice per sweep and Command(pid) is read twice for each argv-only row
