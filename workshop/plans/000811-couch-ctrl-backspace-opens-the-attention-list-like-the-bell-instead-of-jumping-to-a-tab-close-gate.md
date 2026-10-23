---
gate: boundary-review
issue: 811
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8ff7d4a9-744e-4cbe-8515-ee33fb009d45
            repository: github.com/xianxu/pair
            issue_id: "000811"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/d9306bb642ebaf1b
            reviewed_head: 18da8382860fd1cb4e545f0873ad4e2fedea4fe7
            slot: pair:3
            requested_at: "2026-10-23T16:39:15.079344Z"
          phase: finished
          pid: 64877
          started_at: "2026-10-23T16:39:15.220533Z"
          finished_at: "2026-10-23T16:41:26.859414Z"
          session_id: 01a169a1-85e2-7830-9621-b5cc85869590
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 18da8382860fd1cb4e545f0873ad4e2fedea4fe7
        tracker_head: 99f3152fc9a9e4484f0d1248246dfd2271b061d6
        branch_point: 454b965ade23bb9d2fae30a1c75693c0e7ff1a10
        since: "2026-10-23T04:07:36-07:00"
        until: "2026-10-23T09:38:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T04:07:36-07:00"
              end: "2026-10-23T09:38:04-07:00"
              began: "2026-10-23T04:07:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a16004-9432-7d31-86ac-fb5d8d41b0e6
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: fc21a4bdb840b9a51d1e44283d764d18a2009c5be8183904850a5eb2b89fb6f6
              started: "2026-10-21T19:51:15.615Z"
              spawned: false
              bytes: 47524563
              windows:
                - start: "2026-10-23T04:07:36-07:00"
                  end: "2026-10-23T09:38:04-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.1062
      "n": 1
      timestamp: "2026-10-23T09:41:30-07:00"
      dispatched: "2026-10-23T09:39:12-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Important
          title: Unsupported-terminal guidance recommends another enhanced-only shortcut
          detail: README.md:1427-1429 and atlas/couch.md:3203-3204 recommend Ctrl+Alt+a on terminals without enhanced keys. However, cmd/internal/couchkeys/couchkeys.go:98-101 explicitly declares that shortcut enhanced-only, with no legacy encoding; terminal/input.go:351-357 preserves its legacy bytes rather than converting them into the declared chord. Both documentation passages therefore recommend a fallback that cannot work in the stated environment. These are the two explicit instances in this window; also clarify the ambiguous “bell/attention shortcut” fallback comment at couchkeys.go:31. Recommend clicking the bell on unsupported terminals and qualify Ctrl+Alt+a as requiring enhanced encoding.
          family: fallback-matches-terminal-capabilities
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#811 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T09:41:30-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `fallback-matches-terminal-capabilities` Unsupported-terminal guidance recommends another enhanced-only shortcut
  README.md:1427-1429 and atlas/couch.md:3203-3204 recommend Ctrl+Alt+a on terminals without enhanced keys. However, cmd/internal/couchkeys/couchkeys.go:98-101 explicitly declares that shortcut enhanced-only, with no legacy encoding; terminal/input.go:351-357 preserves its legacy bytes rather than converting them into the declared chord. Both documentation passages therefore recommend a fallback that cannot work in the stated environment. These are the two explicit instances in this window; also clarify the ambiguous “bell/attention shortcut” fallback comment at couchkeys.go:31. Recommend clicking the bell on unsupported terminals and qualify Ctrl+Alt+a as requiring enhanced encoding.

## Open findings

- **BR-1** [Important] `fallback-matches-terminal-capabilities` Unsupported-terminal guidance recommends another enhanced-only shortcut
