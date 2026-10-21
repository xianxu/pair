---
gate: boundary-review
issue: 769
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d3461ec5-f3c6-4f19-ba7c-82b682a20008
            repository: github.com/xianxu/pair
            issue_id: "000769"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/603593012770156a
            reviewed_head: 2119e58202baca40d25389862df0a3166e164bc4
            slot: pair:4
            requested_at: "2026-10-21T14:26:51.490456Z"
          phase: finished
          pid: 71190
          started_at: "2026-10-21T14:26:51.642855Z"
          finished_at: "2026-10-21T14:28:55.795577Z"
          session_id: 01a15edb-97dd-7423-9aac-cc18c464843f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2119e58202baca40d25389862df0a3166e164bc4
        tracker_head: 2453bbce235751bcdbd0391ed330a228de9322b7
        branch_point: 7fabbeb93589b7c0c44a43fa699e83b09082bd5e
        since: "2026-10-21T06:43:45-07:00"
        until: "2026-10-21T07:25:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T06:43:45-07:00"
              end: "2026-10-21T07:25:54-07:00"
              began: "2026-10-21T06:43:45-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: eddd9fb62afad98f660fafea3ba76afaaae996ec28036a46335e937c3f5874fb
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 29931387
              windows:
                - start: "2026-10-21T06:43:45-07:00"
                  end: "2026-10-21T07:25:54-07:00"
            - harness: codex
              id: 01a15eb4-d650-7bf1-92c3-672d1e0c8e6e
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: fdd108bec4069a521c45f3d157c515f73d27757ca9e4b7da897249aecc287c1f
              started: "2026-10-21T13:44:32.143Z"
              spawned: true
              bytes: 1616828
              windows:
                - start: "2026-10-21T13:44:32.143Z"
                  end: "2026-10-21T07:25:54-07:00"
            - harness: codex
              id: 01a15eba-e6e7-7c90-859b-7d708807eeb8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 85338e4376e043c18a37800d36907876f3e71f588c7e1e47ad838562735ba64e
              started: "2026-10-21T13:51:09.932Z"
              spawned: false
              bytes: 587179
              windows:
                - start: "2026-10-21T13:51:09.932Z"
                  end: "2026-10-21T07:25:54-07:00"
            - harness: codex
              id: 01a15ebd-c1d6-7983-a527-7d2c80b25047
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 62c55c6ca24f2e2e47da34517d31fab691980d125fe62592b1fbe3b88781d77e
              started: "2026-10-21T13:54:17.184Z"
              spawned: false
              bytes: 406148
              windows:
                - start: "2026-10-21T13:54:17.184Z"
                  end: "2026-10-21T07:25:54-07:00"
            - harness: codex
              id: 01a15ebf-436b-7dd2-8662-c4d29bf2ec59
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ba1299fc962df826dc941321bd633f1f632052971b1dc0b52fc43b3fe197c0b8
              started: "2026-10-21T13:55:55.795Z"
              spawned: false
              bytes: 430746
              windows:
                - start: "2026-10-21T13:55:55.795Z"
                  end: "2026-10-21T07:25:54-07:00"
            - harness: codex
              id: 01a15ebf-cc02-7742-b092-1134e63ff069
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: b21c090f0aa1d83d2dd7f09016fffb30026d39dd8b263cb06968031e36b997b4
              started: "2026-10-21T13:56:30.605Z"
              spawned: false
              bytes: 394674
              windows:
                - start: "2026-10-21T13:56:30.605Z"
                  end: "2026-10-21T07:25:54-07:00"
      "n": 1
      timestamp: "2026-10-21T07:28:57-07:00"
      dispatched: "2026-10-21T07:26:48-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: README update appears missing for wrap help and Darwin lifetime behavior.
          detail: cmd/internal/wrapcmd/wrap.go:2776 introduces pair wrap --help and -h, and lines 2816–2819 describe the changed closed-stdio lifetime policy, but README.md is unchanged in the pinned range. Add a concise README usage entry linking the help command and explaining that Darwin targets closing stdio remain owned until exit or explicit stop; atlas documentation alone does not satisfy the README update gate.
          family: user-surface-readme-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ba70e633-df61-4c0c-8f26-093b0a9fabfe
            repository: github.com/xianxu/pair
            issue_id: "000769"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/d80a1b641c93be44
            reviewed_head: dfc195a1b30268c577cbe3e8d2e21123f60c19b3
            slot: pair:4
            requested_at: "2026-10-21T14:31:00.317841Z"
          phase: finished
          pid: 46138
          started_at: "2026-10-21T14:31:00.424877Z"
          finished_at: "2026-10-21T14:33:29.853799Z"
          session_id: 01a15edf-6349-7123-bee1-9b52ddaa6ff2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: dfc195a1b30268c577cbe3e8d2e21123f60c19b3
        tracker_head: 2453bbce235751bcdbd0391ed330a228de9322b7
        branch_point: 7fabbeb93589b7c0c44a43fa699e83b09082bd5e
        since: "2026-10-21T06:43:45-07:00"
        until: "2026-10-21T07:30:29-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T06:43:45-07:00"
              end: "2026-10-21T07:30:29-07:00"
              began: "2026-10-21T06:43:45-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: codex
              id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 2ae21945eba1b7166901a371deecfa662b5f799de4fbd987bca7236a11d1e89e
              started: "2026-10-20T17:08:30.923Z"
              spawned: false
              bytes: 30236632
              windows:
                - start: "2026-10-21T06:43:45-07:00"
                  end: "2026-10-21T07:30:29-07:00"
            - harness: codex
              id: 01a15eb4-d650-7bf1-92c3-672d1e0c8e6e
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              parent_id: 01a15a49-099a-71f3-872b-0c2ad747d4bc
              digest: fdd108bec4069a521c45f3d157c515f73d27757ca9e4b7da897249aecc287c1f
              started: "2026-10-21T13:44:32.143Z"
              spawned: true
              bytes: 1616828
              windows:
                - start: "2026-10-21T13:44:32.143Z"
                  end: "2026-10-21T07:30:29-07:00"
            - harness: codex
              id: 01a15eba-e6e7-7c90-859b-7d708807eeb8
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 85338e4376e043c18a37800d36907876f3e71f588c7e1e47ad838562735ba64e
              started: "2026-10-21T13:51:09.932Z"
              spawned: false
              bytes: 587179
              windows:
                - start: "2026-10-21T13:51:09.932Z"
                  end: "2026-10-21T07:30:29-07:00"
            - harness: codex
              id: 01a15ebd-c1d6-7983-a527-7d2c80b25047
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 62c55c6ca24f2e2e47da34517d31fab691980d125fe62592b1fbe3b88781d77e
              started: "2026-10-21T13:54:17.184Z"
              spawned: false
              bytes: 406148
              windows:
                - start: "2026-10-21T13:54:17.184Z"
                  end: "2026-10-21T07:30:29-07:00"
            - harness: codex
              id: 01a15ebf-436b-7dd2-8662-c4d29bf2ec59
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: ba1299fc962df826dc941321bd633f1f632052971b1dc0b52fc43b3fe197c0b8
              started: "2026-10-21T13:55:55.795Z"
              spawned: false
              bytes: 430746
              windows:
                - start: "2026-10-21T13:55:55.795Z"
                  end: "2026-10-21T07:30:29-07:00"
            - harness: codex
              id: 01a15ebf-cc02-7742-b092-1134e63ff069
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: b21c090f0aa1d83d2dd7f09016fffb30026d39dd8b263cb06968031e36b997b4
              started: "2026-10-21T13:56:30.605Z"
              spawned: false
              bytes: 394674
              windows:
                - start: "2026-10-21T13:56:30.605Z"
                  end: "2026-10-21T07:30:29-07:00"
        cuts:
            - dispatched: "2026-10-21T07:26:48-07:00"
              finalized: "2026-10-21T07:28:57-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T07:33:38-07:00"
      dispatched: "2026-10-21T07:30:58-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:417 now documents pair wrap --help/-h and ownership until target exit or explicit stop. This matches wrap.go:2776 and :2814, the retained-slave implementation in pty_darwin.go:50, and atlas/terminal.md:189–199. The correction is prose-only; native lifecycle tests independently passed.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: dfc195a1b30268c577cbe3e8d2e21123f60c19b3
      blocked: false
---

# Gate ledger — pair#769 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T07:28:57-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-surface-readme-coverage` (kind: defect) README update appears missing for wrap help and Darwin lifetime behavior.
  cmd/internal/wrapcmd/wrap.go:2776 introduces pair wrap --help and -h, and lines 2816–2819 describe the changed closed-stdio lifetime policy, but README.md is unchanged in the pinned range. Add a concise README usage entry linking the help command and explaining that Darwin targets closing stdio remain owned until exit or explicit stop; atlas documentation alone does not satisfy the README update gate.

## Round 2 — 2026-10-21T07:33:38-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — README.md:417 now documents pair wrap --help/-h and ownership until target exit or explicit stop. This matches wrap.go:2776 and :2814, the retained-slave implementation in pty_darwin.go:50, and atlas/terminal.md:189–199. The correction is prose-only; native lifecycle tests independently passed.

## Open findings

(none — every finding has been disposed)
