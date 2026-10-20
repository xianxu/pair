---
gate: boundary-review
issue: 730
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4647af9f-143b-422a-a62d-20131a1ee87f
            repository: github.com/xianxu/pair
            issue_id: "000730"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/1b2275b46ca2c4e2
            reviewed_head: 4d5e387c09e72ec67ba9c4db75c477a5e5c8d2f0
            slot: pair:4
            requested_at: "2026-10-20T18:10:50.589596Z"
          phase: finished
          pid: 83362
          started_at: "2026-10-20T18:10:50.786434Z"
          finished_at: "2026-10-20T18:11:39.267803Z"
          session_id: 01a15a82-4c99-7000-be42-c50942484eda
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4d5e387c09e72ec67ba9c4db75c477a5e5c8d2f0
        tracker_head: abe3eac4533e690b0342fcdd503b09e48bacb057
        branch_point: 0aa4bee4ad83c49eea27226d718236e01698ad26
        since: "2026-10-20T10:08:52-07:00"
        until: "2026-10-20T11:10:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T10:08:52-07:00"
              end: "2026-10-20T11:10:19-07:00"
              began: "2026-10-20T10:08:52-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 8bdf0be8da59ac559b9d63458597cc8d9b65741c81bc78538699bf8d524e49e5
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 1887269
              windows:
                - start: "2026-10-20T10:08:52-07:00"
                  end: "2026-10-20T11:10:19-07:00"
      "n": 1
      timestamp: "2026-10-20T11:11:40-07:00"
      dispatched: "2026-10-20T11:10:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: README still describes switcher hubs as reversed
          detail: README.md:1238 says both the hub tab and switcher row use reversed colours, contradicting menu_render.go:695–705 and the corrected atlas/couch.md:1748–1750. Update the paragraph to distinguish the reversed tab chip from the underlined switcher hub, with reverse reserved for selection. Family sweep found this single stale description in README.md and atlas/.
          family: user-facing-doc-consistency
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#730 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T11:11:40-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-facing-doc-consistency` (kind: defect) README still describes switcher hubs as reversed
  README.md:1238 says both the hub tab and switcher row use reversed colours, contradicting menu_render.go:695–705 and the corrected atlas/couch.md:1748–1750. Update the paragraph to distinguish the reversed tab chip from the underlined switcher hub, with reverse reserved for selection. Family sweep found this single stale description in README.md and atlas/.

## Open findings

- **BR-1** [Important] `user-facing-doc-consistency` (kind: defect) README still describes switcher hubs as reversed
