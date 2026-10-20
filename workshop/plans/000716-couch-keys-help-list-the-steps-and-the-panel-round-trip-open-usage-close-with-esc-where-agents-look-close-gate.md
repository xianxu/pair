---
gate: boundary-review
issue: 716
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9e9f7ffa-eb36-47d8-9766-12df26aeeb92
            repository: github.com/xianxu/pair
            issue_id: "000716"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/eccb048f8ef41b9d
            reviewed_head: 5dddd69efbe9733a203d68128f3d55f7c8401b51
            slot: pair:2
            requested_at: "2026-10-20T03:05:39.332059Z"
          phase: finished
          pid: 42621
          started_at: "2026-10-20T03:05:39.714697Z"
          finished_at: "2026-10-20T03:09:40.780623Z"
          session_id: 15bf7af5-29c4-447c-b16d-cedb07d31717
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 5dddd69efbe9733a203d68128f3d55f7c8401b51
        tracker_head: ee843b01decf9b9e5deaf950d8ff718036619d70
        branch_point: 4b3105cb2d8d4b4a99ec29e8e7d91a3974de820a
        since: "2026-10-19T19:14:56-07:00"
        until: "2026-10-19T19:23:08-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T19:14:56-07:00"
              end: "2026-10-19T19:23:08-07:00"
              began: "2026-10-19T19:14:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 13a7ea79206731ceb6f4348a119fac1cd8e7a2616fcc8a576bb5ccf5182af03f
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 49688114
              windows:
                - start: "2026-10-19T19:14:56-07:00"
                  end: "2026-10-19T19:23:08-07:00"
      "n": 1
      timestamp: "2026-10-19T20:09:42-07:00"
      dispatched: "2026-10-19T20:05:35-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: Panel refusal hint tells a refused lone esc that only a lone esc is admitted
          detail: Admission.Refusal (recipe.go:95) only looks at the admission. A DismissOnly request refused for an incomplete screen (wrapper) or an empty Instance (agent_drive_service.go:109) still gets "only a lone esc is admitted ... then retry". Consider keeping the generic wording when the request is already DismissOnly.
          family: refusal-wording-ignores-request
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Step meaning and skill prose hardcode limits that the help renders from constants
          detail: The wait:MS meaning in agentdrive.Steps says "0–1000 ms, at most 5 seconds" and SKILL.md says "64 steps, 8 KiB". If MaxKey* changes, neither is checked. Render them with fmt from the constants, or test them against the constants.
          family: limit-prose-restates-constants
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: 5dddd69efbe9733a203d68128f3d55f7c8401b51
      blocked: false
---

# Gate ledger — pair#716 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T20:09:42-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `refusal-wording-ignores-request` (kind: defect) Panel refusal hint tells a refused lone esc that only a lone esc is admitted
  Admission.Refusal (recipe.go:95) only looks at the admission. A DismissOnly request refused for an incomplete screen (wrapper) or an empty Instance (agent_drive_service.go:109) still gets "only a lone esc is admitted ... then retry". Consider keeping the generic wording when the request is already DismissOnly.
- **BR-2** [Minor] `limit-prose-restates-constants` (kind: defect) Step meaning and skill prose hardcode limits that the help renders from constants
  The wait:MS meaning in agentdrive.Steps says "0–1000 ms, at most 5 seconds" and SKILL.md says "64 steps, 8 KiB". If MaxKey* changes, neither is checked. Render them with fmt from the constants, or test them against the constants.

## Open findings

- **BR-1** [Minor] `refusal-wording-ignores-request` (kind: defect) Panel refusal hint tells a refused lone esc that only a lone esc is admitted
- **BR-2** [Minor] `limit-prose-restates-constants` (kind: defect) Step meaning and skill prose hardcode limits that the help renders from constants
