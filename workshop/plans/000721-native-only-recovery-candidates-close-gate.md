---
gate: boundary-review
issue: 721
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 72b02fcc-f55a-42be-97ed-104b4efefab5
            repository: github.com/xianxu/pair
            issue_id: "000721"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/34724b7e71d0d65b
            reviewed_head: 0d8f276deb35fa5f2fa201cad88ab884644195c6
            slot: pair:3
            requested_at: "2026-10-20T07:25:57.510744Z"
          phase: finished
          pid: 97106
          started_at: "2026-10-20T07:25:57.735707Z"
          finished_at: "2026-10-20T07:29:18.452112Z"
          session_id: 01a15833-e457-7780-96bd-9d7334c0d38d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 0d8f276deb35fa5f2fa201cad88ab884644195c6
        tracker_head: c47ecf82459be419b5add46697edee3bd5fcbace
        branch_point: 3a1418f40e77f77d5a6e9e619efa46f8cbe36c8b
        since: "2026-10-19T21:10:41-07:00"
        until: "2026-10-20T00:25:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T21:10:41-07:00"
              end: "2026-10-20T00:25:10-07:00"
              began: "2026-10-19T21:10:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a15678-2c82-7682-8812-61f85b526f0a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: d297189423602c210660bbb10f315f0e36469be4d9f8d06c053b67a7bb82fab1
              started: "2026-10-19T23:21:19.095Z"
              spawned: false
              bytes: 12235358
              windows:
                - start: "2026-10-19T21:10:41-07:00"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a15810-571b-7123-9206-e124cade8c8a
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 83157abcf491ddb250a3b7f98d76a736f167b5cc1e9ece575f102332f736f739
              started: "2026-10-20T06:47:08.671Z"
              spawned: false
              bytes: 3580746
              windows:
                - start: "2026-10-20T06:47:08.671Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a15810-d79a-77b0-ac36-0f24ac2103da
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15810-571b-7123-9206-e124cade8c8a
              digest: 30a4121b3b1397a7cbbbeda8ea893b32cf0164014573157227709a6d1ba39d6f
              started: "2026-10-20T06:47:41.317Z"
              spawned: true
              bytes: 1376373
              windows:
                - start: "2026-10-20T06:47:41.317Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a15815-474b-7612-818e-9abbeb4f0af9
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 155751c1754fdf74d38a04059a208796d26543aa5fe01f8f12e86fba9dd37b4b
              started: "2026-10-20T06:52:32.423Z"
              spawned: false
              bytes: 639644
              windows:
                - start: "2026-10-20T06:52:32.423Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a15817-0137-73a1-a82b-b5f5722e166e
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 7747ab0dd5d4654237f4886c85b1ef2aad0b6e0a639a5be10aa0d0d22b7be8ad
              started: "2026-10-20T06:54:25.507Z"
              spawned: false
              bytes: 439119
              windows:
                - start: "2026-10-20T06:54:25.507Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a15818-b842-7983-b766-ccb6bc22c5e2
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 8d8ee34755d941ab3ee37d8a6e38e04150ce15e3636dea0a09f2067a9f1d6d6e
              started: "2026-10-20T06:56:18.138Z"
              spawned: false
              bytes: 518198
              windows:
                - start: "2026-10-20T06:56:18.138Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a15819-958c-7f31-879f-3e9cc5944f8f
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: ce2d4cf2737b020c6b2abba0f935efecc9675cca05b3ec1eb050ac4fba75f66d
              started: "2026-10-20T06:57:14.813Z"
              spawned: false
              bytes: 422672
              windows:
                - start: "2026-10-20T06:57:14.813Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a1581a-ca17-7851-9db7-b0f9d435706e
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15810-571b-7123-9206-e124cade8c8a
              digest: 72467ead3223fc28c44b8addf41cc3ded0bf7bb9b6d4fcde324c7d709a8f2d73
              started: "2026-10-20T06:58:33.202Z"
              spawned: true
              bytes: 920312
              windows:
                - start: "2026-10-20T06:58:33.202Z"
                  end: "2026-10-20T00:25:10-07:00"
            - harness: codex
              id: 01a1581a-fea0-7173-b5ec-8bc1149c5109
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a15810-571b-7123-9206-e124cade8c8a
              digest: 4a98d8d53876bafcc621ebf4ad2bb6fb2e4ec8c4ddf60eb55e1ce183b480bca8
              started: "2026-10-20T06:58:46.649Z"
              spawned: true
              bytes: 785884
              windows:
                - start: "2026-10-20T06:58:46.649Z"
                  end: "2026-10-20T00:25:10-07:00"
        queue_waits:
            - attempt_id: f362417d7963a354d4add24c0f3358f2
              slot: pair:3
              record_digest: d3b5bfc2c00ca464d2a02f636de51c827ff928c712240641bd6e7217e87dc8bf
              start: "2026-10-20T04:23:57.388355Z"
              end: "2026-10-20T04:48:43.982663Z"
              queue_seconds: 1486.594308
              raw_seconds: 1486.594308
              exact: true
            - attempt_id: dc6266e5245eb2c4939d554b12b0f278
              slot: pair:3
              record_digest: 1b1d1273505fe04db62a30ff4fef152b574b6d94411b1be573ecafac723f7a03
              start: "2026-10-20T06:15:38.841775Z"
              end: "2026-10-20T06:27:48.233515Z"
              queue_seconds: 729.39174
              raw_seconds: 729.39174
              exact: true
        protected:
            - issue: "714"
              slot: pair:3
              start: "2026-10-19T21:10:41-07:00"
              end: "2026-10-20T04:23:57.388355Z"
            - issue: "714"
              slot: pair:3
              start: "2026-10-19T21:10:41-07:00"
              end: "2026-10-20T04:18:58.596Z"
            - issue: "714"
              slot: pair:3
              start: "2026-10-20T04:11:18.667Z"
              end: "2026-10-20T04:12:15.059Z"
            - issue: "714"
              slot: pair:3
              start: "2026-10-20T04:13:01.451Z"
              end: "2026-10-20T04:13:20.295Z"
            - issue: "714"
              slot: pair:3
              start: "2026-10-20T04:14:09.422Z"
              end: "2026-10-20T04:14:32.759Z"
            - issue: "714"
              slot: pair:3
              start: "2026-10-20T04:14:38.231Z"
              end: "2026-10-20T04:14:59.489Z"
            - issue: "714"
              slot: pair:3
              start: "2026-10-20T04:48:43.982663Z"
              end: "2026-10-20T06:15:13.801Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.16611666666666666
      "n": 1
      timestamp: "2026-10-20T00:29:19-07:00"
      dispatched: "2026-10-20T00:25:54-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Native-only evidence is not revalidated before retirement and claim
          detail: cmd/internal/couchcore/recovery_prepared.go:292 skips capture preservation for native-only selections, then line 308 rebuilds authority from the earlier observation without reading the artifact again. Persistence and retirement follow at lines 335–356, and CommitStartClaim at line 412 precedes the next evidence acquisition at line 434. A transcript changed after initial acquisition can therefore cause source retirement and lifecycle writes before selection refuses, contrary to the plan's pre-claim recheck and the Spec's refusal-before-effects contract (ARCH-ORDER). Revalidate the original selection before persistence/retirement and before claim admission, retaining the later checks. Add a controllable mutation regression asserting refusal before retirement or claim.
          family: recovery-revalidation-before-effects
          round: 1
        - id: BR-2
          severity: Critical
          title: Oversized screens consume native scan budget despite never being read
          detail: cmd/internal/couchcore/recovery_evidence_os.go:255–261 checks only the 256 MiB aggregate limit before reserving the entire raw capture. Replay independently rejects raw files above 64 MiB before opening them (cmd/internal/scrollbackcmd/bounded.go:14,55). Thus a 200 MiB screen charges roughly 200 MiB without reading its raw bytes, leaving insufficient budget for an otherwise valid 65 MiB native inventory; retry carries that charge forward and candidates remain unavailable. This violates the plan's oversized-screen preflight and remaining-budget contract (ARCH-CONSTRAINTS, ARCH-PURPOSE). Share the replay bounds at preflight and charge only reads that can occur; test captures between 64 and 256 MiB with native evidence that fits the actual remaining budget.
          family: recovery-budget-charges-attempted-io
          round: 1
        - id: BR-3
          severity: Critical
          title: Early screen failures discard independently readable activity
          detail: cmd/internal/couchcore/recovery_evidence_os.go:228–241 returns on archive validation, missing events, or retained identity mismatch before assigning out.Capture and out.Screen at lines 252–253. Consequently a readable raw file with a missing events file, or a retained post-reboot capture rejected for changed device identity, produces no activity despite available descriptive metadata. This contradicts the Spec's readable-activity and partial-observation requirements (ARCH-PURPOSE). Observe and retain explicitly sourced descriptive activity independently of capture authorization, while preserving strict identity refusal. Add OS-reader regressions asserting unavailable-screen reason plus retained activity for these failures.
          family: recovery-retains-independent-observations
          round: 1
        - id: BR-4
          severity: Important
          title: README update appears missing for native-only candidate selection
          detail: README.md:531–534 still describes resume-candidates as listing the best choice and alternatives with TTY-tail evidence. The range changes that user-facing command to offer unconfirmed native-only choices without tail scoring and introduces a five-minute selection lifetime, but updates only atlas pages. Update the README command description and recovery usage passage with the source distinction, explicit selection, and expiry/refresh behavior required by the docs gate.
          family: user-surface-documentation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#721 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T00:29:19-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `recovery-revalidation-before-effects` Native-only evidence is not revalidated before retirement and claim
  cmd/internal/couchcore/recovery_prepared.go:292 skips capture preservation for native-only selections, then line 308 rebuilds authority from the earlier observation without reading the artifact again. Persistence and retirement follow at lines 335–356, and CommitStartClaim at line 412 precedes the next evidence acquisition at line 434. A transcript changed after initial acquisition can therefore cause source retirement and lifecycle writes before selection refuses, contrary to the plan's pre-claim recheck and the Spec's refusal-before-effects contract (ARCH-ORDER). Revalidate the original selection before persistence/retirement and before claim admission, retaining the later checks. Add a controllable mutation regression asserting refusal before retirement or claim.
- **BR-2** [Critical] `recovery-budget-charges-attempted-io` Oversized screens consume native scan budget despite never being read
  cmd/internal/couchcore/recovery_evidence_os.go:255–261 checks only the 256 MiB aggregate limit before reserving the entire raw capture. Replay independently rejects raw files above 64 MiB before opening them (cmd/internal/scrollbackcmd/bounded.go:14,55). Thus a 200 MiB screen charges roughly 200 MiB without reading its raw bytes, leaving insufficient budget for an otherwise valid 65 MiB native inventory; retry carries that charge forward and candidates remain unavailable. This violates the plan's oversized-screen preflight and remaining-budget contract (ARCH-CONSTRAINTS, ARCH-PURPOSE). Share the replay bounds at preflight and charge only reads that can occur; test captures between 64 and 256 MiB with native evidence that fits the actual remaining budget.
- **BR-3** [Critical] `recovery-retains-independent-observations` Early screen failures discard independently readable activity
  cmd/internal/couchcore/recovery_evidence_os.go:228–241 returns on archive validation, missing events, or retained identity mismatch before assigning out.Capture and out.Screen at lines 252–253. Consequently a readable raw file with a missing events file, or a retained post-reboot capture rejected for changed device identity, produces no activity despite available descriptive metadata. This contradicts the Spec's readable-activity and partial-observation requirements (ARCH-PURPOSE). Observe and retain explicitly sourced descriptive activity independently of capture authorization, while preserving strict identity refusal. Add OS-reader regressions asserting unavailable-screen reason plus retained activity for these failures.
- **BR-4** [Important] `user-surface-documentation` README update appears missing for native-only candidate selection
  README.md:531–534 still describes resume-candidates as listing the best choice and alternatives with TTY-tail evidence. The range changes that user-facing command to offer unconfirmed native-only choices without tail scoring and introduces a five-minute selection lifetime, but updates only atlas pages. Update the README command description and recovery usage passage with the source distinction, explicit selection, and expiry/refresh behavior required by the docs gate.

## Open findings

- **BR-1** [Critical] `recovery-revalidation-before-effects` Native-only evidence is not revalidated before retirement and claim
- **BR-2** [Critical] `recovery-budget-charges-attempted-io` Oversized screens consume native scan budget despite never being read
- **BR-3** [Critical] `recovery-retains-independent-observations` Early screen failures discard independently readable activity
- **BR-4** [Important] `user-surface-documentation` README update appears missing for native-only candidate selection
