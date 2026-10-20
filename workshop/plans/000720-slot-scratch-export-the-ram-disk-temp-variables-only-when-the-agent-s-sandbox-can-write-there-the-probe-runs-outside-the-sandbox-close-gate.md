---
gate: boundary-review
issue: 720
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8a945867-1589-45dc-b29c-29780582f2d2
            repository: github.com/xianxu/pair
            issue_id: "000720"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e0b2d34759bed559
            reviewed_head: 7d772f18a1e923e9ced69d630e8c142f52d43cc2
            slot: pair:7
            requested_at: "2026-10-20T04:23:39.668926Z"
          phase: finished
          pid: 77636
          started_at: "2026-10-20T04:23:39.761125Z"
          finished_at: "2026-10-20T04:26:36.442624Z"
          session_id: 01a1578c-fcf1-7a12-af5d-793d2abf31bc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 7d772f18a1e923e9ced69d630e8c142f52d43cc2
        tracker_head: 138db9496a4248a63a9d3d0951c434fd2c2a0b3c
        branch_point: 7d9ce7ca7988743e3ef201221a19c985c8cecec7
        since: "2026-10-19T21:04:58-07:00"
        until: "2026-10-19T21:23:21-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T21:04:58-07:00"
              end: "2026-10-19T21:23:21-07:00"
              began: "2026-10-19T21:04:58-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a677842155287b382c2520c401f10bb5532c52a66cbad7016aa15fcc0abbdcab
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 180380506
              windows:
                - start: "2026-10-19T21:04:58-07:00"
                  end: "2026-10-19T21:23:21-07:00"
            - harness: codex
              id: 01a1577f-ba8d-77e1-b41c-0a7b9dc983bc
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 47db2390379917bdce316583865314d83e32a269f17c715369f93cf06a314ab5
              started: "2026-10-20T04:09:11.634Z"
              spawned: false
              bytes: 597930
              windows:
                - start: "2026-10-20T04:09:11.634Z"
                  end: "2026-10-19T21:23:21-07:00"
            - harness: codex
              id: 01a15781-8cdd-7350-8c1d-49943a68f97b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 9d1ee0673acd7c0ec3bfea98daa94d998e91fe0274a9dc1728994637a2b8b33f
              started: "2026-10-20T04:11:10.85Z"
              spawned: false
              bytes: 431568
              windows:
                - start: "2026-10-20T04:11:10.85Z"
                  end: "2026-10-19T21:23:21-07:00"
            - harness: codex
              id: 01a15781-e425-74d3-87d3-12627b51e5fe
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6b3fcdbaae6f0187def40613f844fee4925c0fb9dd457a32f87ed9061c7c01bf
              started: "2026-10-20T04:11:33.495Z"
              spawned: false
              bytes: 420976
              windows:
                - start: "2026-10-20T04:11:33.495Z"
                  end: "2026-10-19T21:23:21-07:00"
            - harness: codex
              id: 01a15782-94b2-71a2-b265-1115743a74de
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 6d74c80e800cb61589746e4e256f7aac23307a4158a34215086db6c0949a4d0e
              started: "2026-10-20T04:12:18.165Z"
              spawned: true
              bytes: 434925
              windows:
                - start: "2026-10-20T04:12:18.165Z"
                  end: "2026-10-19T21:23:21-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.3002666666666667
      "n": 1
      timestamp: "2026-10-19T21:26:39-07:00"
      dispatched: "2026-10-19T21:23:38-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Claude admission ignores permission rules that deny sandbox writes
          detail: cmd/internal/couchscratch/sandbox_settings.go:29-55 projects only sandbox.filesystem and ignores permissions.deny. With sandbox.enabled=true, allowWrite=["/Volumes/TestScratch"], and permissions.deny=["Edit(//Volumes/TestScratch/**)"], ScratchAllowed returns true although Claude applies Edit denies to sandbox access. Project these restrictions, or return unknown when unsupported write-denial rules occur; add a regression proving that this configuration produces no scratch exports.
          family: effective-policy-denials
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Claude local settings discovery misses repository and main-checkout policy
          detail: cmd/internal/couchscratch/sandbox_read.go:59-62 reads settings.local.json only beneath in.Project, supplied from record.StartingPath at cmd/internal/couchcmd/scratch.go:175. Claude can load local settings from the repository root when launched in a subdirectory, and from the main checkout for a worktree. A denying local policy there is skipped and absence is treated as sandbox disabled, allowing all three scratch exports. Resolve the native local-settings location or conservatively decline unresolved layouts; cover subdirectory and linked-worktree fixtures through the production reader.
          family: effective-policy-source-discovery
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 46562d83-f3fa-4e4b-8971-7013743b2b50
            repository: github.com/xianxu/pair
            issue_id: "000720"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/0adfd7a456d01ded
            reviewed_head: ba3584f4f35642e621411abf34f4cc59badeeee6
            slot: pair:7
            requested_at: "2026-10-20T04:39:02.17052Z"
          phase: finished
          pid: 72773
          started_at: "2026-10-20T04:39:02.465877Z"
          finished_at: "2026-10-20T04:41:01.645412Z"
          session_id: 01a1579b-1219-72b2-9457-364cefd2f4c5
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: ba3584f4f35642e621411abf34f4cc59badeeee6
        tracker_head: 344b4f90c74e8203476886db116167dc3fdf4e5d
        branch_point: 7d9ce7ca7988743e3ef201221a19c985c8cecec7
        since: "2026-10-19T21:04:58-07:00"
        until: "2026-10-19T21:38:03-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T21:04:58-07:00"
              end: "2026-10-19T21:38:03-07:00"
              began: "2026-10-19T21:04:58-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: f73ff5ea9c0e8222dc637c8008724f80da626032b66900b9cea3fa812c9550ec
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 182365752
              windows:
                - start: "2026-10-19T21:04:58-07:00"
                  end: "2026-10-19T21:38:03-07:00"
            - harness: codex
              id: 01a1577f-ba8d-77e1-b41c-0a7b9dc983bc
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 47db2390379917bdce316583865314d83e32a269f17c715369f93cf06a314ab5
              started: "2026-10-20T04:09:11.634Z"
              spawned: false
              bytes: 597930
              windows:
                - start: "2026-10-20T04:09:11.634Z"
                  end: "2026-10-19T21:38:03-07:00"
            - harness: codex
              id: 01a15781-8cdd-7350-8c1d-49943a68f97b
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 9d1ee0673acd7c0ec3bfea98daa94d998e91fe0274a9dc1728994637a2b8b33f
              started: "2026-10-20T04:11:10.85Z"
              spawned: false
              bytes: 431568
              windows:
                - start: "2026-10-20T04:11:10.85Z"
                  end: "2026-10-19T21:38:03-07:00"
            - harness: codex
              id: 01a15781-e425-74d3-87d3-12627b51e5fe
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 6b3fcdbaae6f0187def40613f844fee4925c0fb9dd457a32f87ed9061c7c01bf
              started: "2026-10-20T04:11:33.495Z"
              spawned: false
              bytes: 420976
              windows:
                - start: "2026-10-20T04:11:33.495Z"
                  end: "2026-10-19T21:38:03-07:00"
            - harness: codex
              id: 01a15782-94b2-71a2-b265-1115743a74de
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 6d74c80e800cb61589746e4e256f7aac23307a4158a34215086db6c0949a4d0e
              started: "2026-10-20T04:12:18.165Z"
              spawned: true
              bytes: 434925
              windows:
                - start: "2026-10-20T04:12:18.165Z"
                  end: "2026-10-19T21:38:03-07:00"
            - harness: codex
              id: 01a15793-8e40-7741-9b17-442d2fa818a3
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 90db1e1c748df235eab95cb6f25679797d32943bfe8c52e547041df9c7f822a7
              started: "2026-10-20T04:30:51.053Z"
              spawned: false
              bytes: 470690
              windows:
                - start: "2026-10-20T04:30:51.053Z"
                  end: "2026-10-19T21:38:03-07:00"
            - harness: codex
              id: 01a15793-fcf8-7db0-a372-e38cc8ce464e
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d76a133c5cbf8acdd01a814d5bf58e2f20f6c82ed54d7aa0a6c1b9c6dd0af9c2
              started: "2026-10-20T04:31:19.423Z"
              spawned: false
              bytes: 374931
              windows:
                - start: "2026-10-20T04:31:19.423Z"
                  end: "2026-10-19T21:38:03-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.3002666666666667
        cuts:
            - dispatched: "2026-10-19T21:23:38-07:00"
              finalized: "2026-10-19T21:26:39-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-19T21:41:03-07:00"
      dispatched: "2026-10-19T21:38:59-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The authoritative V2 revision replaces policy inference with an explicit declaration. Native-policy readers are absent; the production composition regression rejects exports without the marker and fails when the admission guard is removed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: V2 removes native local-settings discovery entirely. Admission uses the injected XDG/HOME declaration path; lifecycle, routing, and isolation tests pass through the production admission seam.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Important
          title: Slot-only show omits the promised scratch diagnostic
          detail: cmd/internal/couchcore/operationdispatch.go:251-255 projects scratch status only over matched threads. A valid slot with no thread therefore has no scratch status, contrary to the V2 show contract (ARCH-PURPOSE). Extending TestShowReportsASlotWithoutAThread with an injected ScratchDiagnostic and a nonempty Scratch assertion fails. Project admission for slot-only targets too, and retain that regression.
          family: scratch-diagnostics-cover-show-targets
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#720 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T21:26:39-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `effective-policy-denials` (kind: defect) Claude admission ignores permission rules that deny sandbox writes
  cmd/internal/couchscratch/sandbox_settings.go:29-55 projects only sandbox.filesystem and ignores permissions.deny. With sandbox.enabled=true, allowWrite=["/Volumes/TestScratch"], and permissions.deny=["Edit(//Volumes/TestScratch/**)"], ScratchAllowed returns true although Claude applies Edit denies to sandbox access. Project these restrictions, or return unknown when unsupported write-denial rules occur; add a regression proving that this configuration produces no scratch exports.
- **BR-2** [Critical] `effective-policy-source-discovery` (kind: defect) Claude local settings discovery misses repository and main-checkout policy
  cmd/internal/couchscratch/sandbox_read.go:59-62 reads settings.local.json only beneath in.Project, supplied from record.StartingPath at cmd/internal/couchcmd/scratch.go:175. Claude can load local settings from the repository root when launched in a subdirectory, and from the main checkout for a worktree. A denying local policy there is skipped and absence is treated as sandbox disabled, allowing all three scratch exports. Resolve the native local-settings location or conservatively decline unresolved layouts; cover subdirectory and linked-worktree fixtures through the production reader.

## Round 2 — 2026-10-19T21:41:03-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The authoritative V2 revision replaces policy inference with an explicit declaration. Native-policy readers are absent; the production composition regression rejects exports without the marker and fails when the admission guard is removed.
- BR-2 — addressed — V2 removes native local-settings discovery entirely. Admission uses the injected XDG/HOME declaration path; lifecycle, routing, and isolation tests pass through the production admission seam.

### Raised

- **BR-3** [Important] `scratch-diagnostics-cover-show-targets` (kind: defect) Slot-only show omits the promised scratch diagnostic
  cmd/internal/couchcore/operationdispatch.go:251-255 projects scratch status only over matched threads. A valid slot with no thread therefore has no scratch status, contrary to the V2 show contract (ARCH-PURPOSE). Extending TestShowReportsASlotWithoutAThread with an injected ScratchDiagnostic and a nonempty Scratch assertion fails. Project admission for slot-only targets too, and retain that regression.

## Open findings

- **BR-3** [Important] `scratch-diagnostics-cover-show-targets` (kind: defect) Slot-only show omits the promised scratch diagnostic
