---
gate: boundary-review
issue: 891
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-25T13:36:11-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: Prepended --no-daemon is not in codexBoolGlobalOption, so the wrapper fresh start keeps resume <id>
          detail: |-
            codexResumeCommandIndex (agentargs.go:75) stops at an unknown leading flag, and stripCodexResumeSubcommand's fallback checks only args[0]. freshAgentInvocation (wrap.go:2620) runs FreshAgentArgs over the encoded commandArgs, so "--no-daemon --disable shell_snapshot resume id" keeps the resume binding. Add --no-daemon to codexBoolGlobalOption, sweep the other readers of that table (fresh_args.go freshValueOption, native_resume_selector), and add a FreshAgentArgs test with the flag leading.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: codex-global-flag-table-single-source
          round: 1
        - id: BR-2
          severity: Minor
          title: The per-launch codex --version probe has a 3s bound but no measured typical cost
          detail: |-
            It runs on the launch/startup path for every Codex launch; record a measured latency in the Log.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: launch-path-latency-budget
          round: 1
        - id: BR-3
          severity: Minor
          title: The agent_version.go snippet omits the time, strings and strconv imports
          detail: (carried from plan-quality PQ-3, deferred to the boundary review)
          family: plan-snippet-compiles
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 42e51784-1983-4fc4-954d-3323952a4d14
            repository: github.com/xianxu/pair
            issue_id: "000891"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/5d945210df0140cf
            reviewed_head: 4a31184eff5b60725c7e3b46662ec39f0f3b989a
            slot: pair:8
            requested_at: "2026-10-25T20:34:25.005803Z"
          phase: finished
          pid: 19397
          started_at: "2026-10-25T20:34:25.117794Z"
          finished_at: "2026-10-25T20:36:07.471694Z"
          session_id: a0b79cf9-8828-46f1-9040-1339502c6eee
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-queue-v1
        repository: github.com/xianxu/pair
        code_head: 4a31184eff5b60725c7e3b46662ec39f0f3b989a
        tracker_head: f2e6f7c7a1e1fd529a54090d618c95e6f1264c61
        branch_point: 802880a417ae347054f71959961ef7a860bb4e36
        since: "2026-10-25T09:36:24-07:00"
        until: "2026-10-25T13:25:52-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-25T09:36:24-07:00"
              end: "2026-10-25T13:25:52-07:00"
              began: "2026-10-25T09:36:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: claude
              id: 111274fc-89c5-4440-91df-adc1a07cfe70
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: fa062a53557809a167e3bcc0a41a1671b64d4f83667ddbad761a4c3f45234d2e
              started: "2026-10-25T16:49:27.738Z"
              spawned: false
              bytes: 711378
              windows:
                - start: "2026-10-25T16:49:27.738Z"
                  end: "2026-10-25T13:25:52-07:00"
              dispatch_windows:
                - start: "2026-10-25T16:49:27.738Z"
                  end: "2026-10-25T13:25:52-07:00"
            - harness: qoder
              id: 67fc755c-6be9-468a-8b48-8dc8b7ad729b
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: 45154ae2d608de9182ac76242c971aa1157482ce85a4576a26ab95897132dd45
              started: "2026-10-23T14:04:55.968Z"
              spawned: false
              bytes: 6611422
              windows:
                - start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T13:25:52-07:00"
              dispatch_windows:
                - start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T13:25:52-07:00"
        queue_waits:
            - attempt_id: 111a1928a151ff2bd6d090179c0b7ce1
              slot: pair:8
              record_digest: 597e2fbdb18cd397af692372110e0c40f6ffccbe6bc341dd36c96acafaff4194
              start: "2026-10-25T17:23:13.129091Z"
              end: "2026-10-25T18:13:53.038877Z"
              queue_seconds: 3039.909786
              raw_seconds: 3039.909786
              exact: true
            - attempt_id: 0985c36590f84c8aee9e7c6c9a64b7c0
              slot: pair:8
              record_digest: a5b29488d90dc72078f68e326d6ba0eae2d4a01f8c6d55939c8eaf7a0580c6dd
              start: "2026-10-25T18:35:01.638365Z"
              end: "2026-10-25T19:12:59.070822Z"
              queue_seconds: 2277.432457
              raw_seconds: 2277.432457
              exact: true
            - attempt_id: 69df5e1c75d18c0757498bbbafb12dbe
              slot: pair:8
              record_digest: c996e64814d5385fd5c87a9cdd5e44e7798c6426dabe3ec8a14ab2c335887172
              start: "2026-10-25T19:39:21.864931Z"
              end: "2026-10-25T20:09:30.679148Z"
              queue_seconds: 1808.814217
              raw_seconds: 1808.814217
              exact: true
        branches:
            - root: /Users/xianxu/workspace/worktree/pair-slot8/pair
              path: /Users/xianxu/workspace/pair/.git/worktrees/pair7/logs/HEAD
              head: 4a31184eff5b60725c7e3b46662ec39f0f3b989a
              branch: 000891-codex-0-164-s-shared-background-server-prompt-stalls-every-wrapped-start-launch
              digest: c25d90d5be52c91583509dd077647df79ea23491c7e75fa4284e253e088bcba2
              bytes: 20983
              intervals:
                - source_index: 0
                  start: "2026-10-25T16:49:27.738Z"
                  end: "2026-10-25T13:25:52-07:00"
                  kind: branch
                - source: qoder/62ca9f18-5914-4138-9ef3-cbfd57888793
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source: qoder/62ca9f18-5914-4138-9ef3-cbfd57888793
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T13:25:52-07:00"
                  kind: branch
                - source_index: 1
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T13:25:52-07:00"
                  kind: branch
                - source: qoder/7febf963-ecac-409f-82f9-c1438842f2a5
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source: qoder/7febf963-ecac-409f-82f9-c1438842f2a5
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T13:25:52-07:00"
                  kind: branch
                - source: qoder/d0d6b474-f0bb-4817-a59c-ca5442029afe
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source: qoder/d0d6b474-f0bb-4817-a59c-ca5442029afe
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T13:25:52-07:00"
                  kind: branch
      "n": 2
      timestamp: "2026-10-25T13:36:11-07:00"
      dispatched: "2026-10-25T13:34:22-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-4
          severity: Important
          title: PAIR_AGENT_VERSION is only set on a successful codex probe and never cleared, so an inherited stale value reaches the ready record
          detail: 'createflow.go:831-834 calls rt.SetEnv only when the probe returned a version, and only for codex; runcli.go does not unset it, and wrap.go:664 copies os.Getenv into agent_version for every agent. A pair launch from a shell inside a Codex pane (env already carries 0.164.0) records that version for a claude launch or a failed codex probe. Fix: SetEnv unconditionally for all agents (empty when unknown), and add a launcher test that pre-seeds rt.env with a stale value.'
          family: launch-env-must-overwrite-inherited
          round: 2
        - kind: defect
          id: BR-5
          severity: Minor
          title: codex --version probe timeout is not enforced when a grandchild holds stdout (no cmd.WaitDelay)
          detail: osruntime.go:1331-1334 uses exec.CommandContext(...).Output() with no WaitDelay; the npm codex shim forks the native binary, so killing the parent can leave Output blocked past the stated 3s bound. It also uses context.Background instead of the launch startup context.
          family: declared-bound-not-enforced
          round: 2
        - kind: defect
          id: BR-6
          severity: Minor
          title: No launch test pins the --no-daemon gate on a non-Couch codex launch, though the plan applies it to all codex launches
          family: policy-scope-untested
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c7b1684a-f277-48e8-9289-422b5c64b79d
            repository: github.com/xianxu/pair
            issue_id: "000891"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/14b0f5505c92bd72
            reviewed_head: 2fd584aa8b1dae3960d6b9fb135ca90940e5b75f
            slot: pair:8
            requested_at: "2026-10-25T22:22:12.067867Z"
          phase: finished
          pid: 18292
          started_at: "2026-10-25T22:22:12.32679Z"
          finished_at: "2026-10-25T22:29:06.948842Z"
          session_id: 24b01096-64a7-4f3c-9545-869cca77d5a2
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-queue-v1
        repository: github.com/xianxu/pair
        code_head: 2fd584aa8b1dae3960d6b9fb135ca90940e5b75f
        tracker_head: fdf5b3e9402c3b8dc79cf21131006ff84cf02776
        branch_point: 774eb26bf56b367e0b54c212bc5bc2680179a3df
        since: "2026-10-25T09:36:24-07:00"
        until: "2026-10-25T15:12:53-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-25T09:36:24-07:00"
              end: "2026-10-25T15:12:53-07:00"
              began: "2026-10-25T09:36:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: claude
              id: 111274fc-89c5-4440-91df-adc1a07cfe70
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: fa062a53557809a167e3bcc0a41a1671b64d4f83667ddbad761a4c3f45234d2e
              started: "2026-10-25T16:49:27.738Z"
              spawned: false
              bytes: 711378
              windows:
                - start: "2026-10-25T16:49:27.738Z"
                  end: "2026-10-25T15:12:53-07:00"
              dispatch_windows:
                - start: "2026-10-25T16:49:27.738Z"
                  end: "2026-10-25T15:12:53-07:00"
            - harness: qoder
              id: 67fc755c-6be9-468a-8b48-8dc8b7ad729b
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: 745b71d0d6f8167e3f8013901335049b6f51782ad7fac6e608ed03f4eebb92a6
              started: "2026-10-23T14:04:55.968Z"
              spawned: false
              bytes: 6965588
              windows:
                - start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T15:12:53-07:00"
              dispatch_windows:
                - start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T15:12:53-07:00"
        queue_waits:
            - attempt_id: 111a1928a151ff2bd6d090179c0b7ce1
              slot: pair:8
              record_digest: 597e2fbdb18cd397af692372110e0c40f6ffccbe6bc341dd36c96acafaff4194
              start: "2026-10-25T17:23:13.129091Z"
              end: "2026-10-25T18:13:53.038877Z"
              queue_seconds: 3039.909786
              raw_seconds: 3039.909786
              exact: true
            - attempt_id: 0985c36590f84c8aee9e7c6c9a64b7c0
              slot: pair:8
              record_digest: a5b29488d90dc72078f68e326d6ba0eae2d4a01f8c6d55939c8eaf7a0580c6dd
              start: "2026-10-25T18:35:01.638365Z"
              end: "2026-10-25T19:12:59.070822Z"
              queue_seconds: 2277.432457
              raw_seconds: 2277.432457
              exact: true
            - attempt_id: 69df5e1c75d18c0757498bbbafb12dbe
              slot: pair:8
              record_digest: c996e64814d5385fd5c87a9cdd5e44e7798c6426dabe3ec8a14ab2c335887172
              start: "2026-10-25T19:39:21.864931Z"
              end: "2026-10-25T20:09:30.679148Z"
              queue_seconds: 1808.814217
              raw_seconds: 1808.814217
              exact: true
            - attempt_id: 2e0407d2fe3cd7b86670381ea604430d
              slot: pair:8
              record_digest: fb94ed1b63a4e7a5ca0d7b30668988266702a484887176ddfd97934f6379e115
              start: "2026-10-25T20:39:32.155217Z"
              end: "2026-10-25T21:59:03.417882Z"
              queue_seconds: 4771.262665
              raw_seconds: 4771.262665
              exact: true
        cuts:
            - dispatched: "2026-10-25T13:34:22-07:00"
              finalized: "2026-10-25T13:36:11-07:00"
              finalizing: false
        branches:
            - root: /Users/xianxu/workspace/worktree/pair-slot8/pair
              path: /Users/xianxu/workspace/pair/.git/worktrees/pair7/logs/HEAD
              head: 2fd584aa8b1dae3960d6b9fb135ca90940e5b75f
              branch: 000891-codex-0-164-s-shared-background-server-prompt-stalls-every-wrapped-start-launch
              digest: 89392d2e6bb5ef653b33b0819a387e1d33d45082d821d0ef43e18d2d1e026a5f
              bytes: 21984
              intervals:
                - source_index: 0
                  start: "2026-10-25T16:49:27.738Z"
                  end: "2026-10-25T15:12:53-07:00"
                  kind: branch
                - source: qoder/62ca9f18-5914-4138-9ef3-cbfd57888793
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source: qoder/62ca9f18-5914-4138-9ef3-cbfd57888793
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T15:12:53-07:00"
                  kind: branch
                - source_index: 1
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source_index: 1
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T15:12:53-07:00"
                  kind: branch
                - source: qoder/7febf963-ecac-409f-82f9-c1438842f2a5
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source: qoder/7febf963-ecac-409f-82f9-c1438842f2a5
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T15:12:53-07:00"
                  kind: branch
                - source: qoder/d0d6b474-f0bb-4817-a59c-ca5442029afe
                  start: "2026-10-25T09:36:24-07:00"
                  end: "2026-10-25T16:36:46Z"
                  branch: main-slot8
                  kind: ownership-only
                - source: qoder/d0d6b474-f0bb-4817-a59c-ca5442029afe
                  start: "2026-10-25T16:36:46Z"
                  end: "2026-10-25T15:12:53-07:00"
                  kind: branch
      "n": 3
      timestamp: "2026-10-25T15:29:14-07:00"
      dispatched: "2026-10-25T15:22:08-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: withdrawn
          note: 'Evidence holds: --no-daemon is at agentargs.go:109; removing it makes TestFreshAgentArgsWalksPastNoDaemon fail with resume id1 retained (re-run in this checkout).'
          round: 3
        - id: BR-2
          disposition: addressed
          note: Issue Log records codex --version at 0.11-0.15 s real over 3 runs against the 3 s bound.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Plan snippet (plan line 201-206) now imports regexp, strconv, strings, time, matching agent_version.go.
          round: 3
        - id: BR-4
          disposition: addressed
          note: createflow.go:833-834 calls SetEnv unconditionally for all agents; TestAgentVersionEnvOverwritesAnInheritedValue seeds a stale value for codex and claude and asserts it is cleared.
          round: 3
        - id: BR-5
          disposition: addressed
          note: osruntime.go sets cmd.WaitDelay = 1s so the bound holds past a grandchild holding stdout; the context.Background vs launch-context point remains a minor nit.
          round: 3
        - id: BR-6
          disposition: addressed
          note: TestStandaloneCodexLaunchOptsOutOfTheSharedServer pins --no-daemon on a non-Couch codex launch.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 2fd584aa8b1dae3960d6b9fb135ca90940e5b75f
      blocked: false
disputes:
    - id: BR-1
      rationale: 'The finding is factually wrong: the diff adds --no-daemon to codexBoolGlobalOption and tests the fresh-start strip behind it. The wrapper fresh-start path is wrap.go:2626 -> FreshAgentArgs -> persistedConfigArgs -> stripCodexResumeSubcommand -> codexResumeCommandIndex -> codexGlobalOptionWidth -> codexBoolGlobalOption; consuming the leading flag at width 1 lets the scan reach and drop resume <id>. TestCodexLaunchOptsOutOfTheSharedServer additionally pins the launcher carrying the flag through launch into the encoded argv.'
      evidence: cmd/internal/launcher/agentargs.go:109 lists --no-daemon in codexBoolGlobalOption (added in 13b06a465); cmd/internal/launcher/agentargs_test.go:68 TestFreshAgentArgsWalksPastNoDaemon expects ["--no-daemon","--disable","shell_snapshot"]; go test ./cmd/internal/launcher -run TestFreshAgentArgsWalksPastNoDaemon -count=1 -> PASS. Without the table entry the strip stops at the first token and resume id1 remains, so the test would fail.
      after: 2
      timestamp: "2026-10-25T20:38:18Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:8 (/Users/xianxu/workspace/worktree/pair-slot8/pair)
---

# Gate ledger — pair#891 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-25T13:36:11-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `codex-global-flag-table-single-source` Prepended --no-daemon is not in codexBoolGlobalOption, so the wrapper fresh start keeps resume <id>
  codexResumeCommandIndex (agentargs.go:75) stops at an unknown leading flag, and stripCodexResumeSubcommand's fallback checks only args[0]. freshAgentInvocation (wrap.go:2620) runs FreshAgentArgs over the encoded commandArgs, so "--no-daemon --disable shell_snapshot resume id" keeps the resume binding. Add --no-daemon to codexBoolGlobalOption, sweep the other readers of that table (fresh_args.go freshValueOption, native_resume_selector), and add a FreshAgentArgs test with the flag leading.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `launch-path-latency-budget` The per-launch codex --version probe has a 3s bound but no measured typical cost
  It runs on the launch/startup path for every Codex launch; record a measured latency in the Log.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `plan-snippet-compiles` The agent_version.go snippet omits the time, strings and strconv imports
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-10-25T13:36:11-07:00 (claude) — BLOCKED

### Raised

- **BR-4** [Important] `launch-env-must-overwrite-inherited` (kind: defect) PAIR_AGENT_VERSION is only set on a successful codex probe and never cleared, so an inherited stale value reaches the ready record
  createflow.go:831-834 calls rt.SetEnv only when the probe returned a version, and only for codex; runcli.go does not unset it, and wrap.go:664 copies os.Getenv into agent_version for every agent. A pair launch from a shell inside a Codex pane (env already carries 0.164.0) records that version for a claude launch or a failed codex probe. Fix: SetEnv unconditionally for all agents (empty when unknown), and add a launcher test that pre-seeds rt.env with a stale value.
- **BR-5** [Minor] `declared-bound-not-enforced` (kind: defect) codex --version probe timeout is not enforced when a grandchild holds stdout (no cmd.WaitDelay)
  osruntime.go:1331-1334 uses exec.CommandContext(...).Output() with no WaitDelay; the npm codex shim forks the native binary, so killing the parent can leave Output blocked past the stated 3s bound. It also uses context.Background instead of the launch startup context.
- **BR-6** [Minor] `policy-scope-untested` (kind: defect) No launch test pins the --no-daemon gate on a non-Couch codex launch, though the plan applies it to all codex launches

## Round 3 — 2026-10-25T15:29:14-07:00 (claude) — passed

### Disposed

- BR-1 — withdrawn — Evidence holds: --no-daemon is at agentargs.go:109; removing it makes TestFreshAgentArgsWalksPastNoDaemon fail with resume id1 retained (re-run in this checkout).
- BR-2 — addressed — Issue Log records codex --version at 0.11-0.15 s real over 3 runs against the 3 s bound.
- BR-3 — addressed — Plan snippet (plan line 201-206) now imports regexp, strconv, strings, time, matching agent_version.go.
- BR-4 — addressed — createflow.go:833-834 calls SetEnv unconditionally for all agents; TestAgentVersionEnvOverwritesAnInheritedValue seeds a stale value for codex and claude and asserts it is cleared.
- BR-5 — addressed — osruntime.go sets cmd.WaitDelay = 1s so the bound holds past a grandchild holding stdout; the context.Background vs launch-context point remains a minor nit.
- BR-6 — addressed — TestStandaloneCodexLaunchOptsOutOfTheSharedServer pins --no-daemon on a non-Couch codex launch.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-1** — after round 2 (2026-10-25T20:38:18Z, Xian Xu on Xian’s MacBook Pro at pair:8 (/Users/xianxu/workspace/worktree/pair-slot8/pair))
  rationale: The finding is factually wrong: the diff adds --no-daemon to codexBoolGlobalOption and tests the fresh-start strip behind it. The wrapper fresh-start path is wrap.go:2626 -> FreshAgentArgs -> persistedConfigArgs -> stripCodexResumeSubcommand -> codexResumeCommandIndex -> codexGlobalOptionWidth -> codexBoolGlobalOption; consuming the leading flag at width 1 lets the scan reach and drop resume <id>. TestCodexLaunchOptsOutOfTheSharedServer additionally pins the launcher carrying the flag through launch into the encoded argv.
  evidence: cmd/internal/launcher/agentargs.go:109 lists --no-daemon in codexBoolGlobalOption (added in 13b06a465); cmd/internal/launcher/agentargs_test.go:68 TestFreshAgentArgsWalksPastNoDaemon expects ["--no-daemon","--disable","shell_snapshot"]; go test ./cmd/internal/launcher -run TestFreshAgentArgsWalksPastNoDaemon -count=1 -> PASS. Without the table entry the strip stops at the first token and resume id1 remains, so the test would fail.

## Open findings

(none — every finding has been disposed)
