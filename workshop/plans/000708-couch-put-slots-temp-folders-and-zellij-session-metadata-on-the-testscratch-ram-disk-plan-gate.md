---
gate: plan-quality
issue: 708
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 307ed10d-c1cc-499e-982c-a09290cd5712
            repository: github.com/xianxu/pair
            issue_id: "000708"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot3/pair
            reviewed_head: db7f93bb167b0ace183ec0848a999bc9b05521a0
            slot: pair:3
            requested_at: "2026-10-19T17:58:30.701195Z"
          phase: finished
          pid: 59140
          started_at: "2026-10-19T17:58:31.347477Z"
          finished_at: "2026-10-19T18:00:01.697509Z"
          session_id: 01a15550-a959-73f1-bee6-53bae1611707
          outcome: returned
      "n": 1
      timestamp: "2026-10-19T11:00:02-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Define how collection proves detached jobs no longer need scratch
          detail: 'ARCH-ORDER/ARCH-FUNERAL: the current cleanup contract requires retaining detached jobs but names no mechanism that attributes a reparented job with no currently open scratch file to its slot. Specify the evidence source and lifetime, how uncertainty prevents deletion, and a Collect sequence-test strategy that controls parent exit and delayed scratch use; current ancestry and open-file snapshots alone cannot establish this invariant.'
          family: cleanup-requires-lifetime-evidence
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: ab5ee9a9-30ad-41cb-af16-acab06e10309
            repository: github.com/xianxu/pair
            issue_id: "000708"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot3/pair
            reviewed_head: 6feea16666bb4b884895d76b22e572821c3a3f48
            slot: pair:3
            requested_at: "2026-10-19T18:02:05.875731Z"
          phase: finished
          pid: 80840
          started_at: "2026-10-19T18:02:06.070158Z"
          finished_at: "2026-10-19T18:02:39.388708Z"
          session_id: 01a15553-ef35-7412-8cd8-123b550f5afb
          outcome: returned
      "n": 2
      timestamp: "2026-10-19T11:02:39-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: 'R2 supplies environment attribution and controlled delayed-use tests, but its nonempty-environment exclusion still does not prove that a detached process no longer needs scratch. State the rule for every potentially relevant process generation: missing or different environment markers cannot establish unrelatedness after environment replacement. Either retain those generations conservatively or name lifetime evidence that survives marker removal, and exercise that rule with a reparented child that retains the old pathname while presenting a readable nonempty replacement environment. This remains the existing cleanup-requires-lifetime-evidence finding under ARCH-ORDER/ARCH-FUNERAL; no new finding is raised.'
          round: 2
      blocked: true
---

# Gate ledger — pair#708 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T11:00:02-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `cleanup-requires-lifetime-evidence` Define how collection proves detached jobs no longer need scratch
  ARCH-ORDER/ARCH-FUNERAL: the current cleanup contract requires retaining detached jobs but names no mechanism that attributes a reparented job with no currently open scratch file to its slot. Specify the evidence source and lifetime, how uncertainty prevents deletion, and a Collect sequence-test strategy that controls parent exit and delayed scratch use; current ancestry and open-file snapshots alone cannot establish this invariant.

## Round 2 — 2026-10-19T11:02:39-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — not-addressed — R2 supplies environment attribution and controlled delayed-use tests, but its nonempty-environment exclusion still does not prove that a detached process no longer needs scratch. State the rule for every potentially relevant process generation: missing or different environment markers cannot establish unrelatedness after environment replacement. Either retain those generations conservatively or name lifetime evidence that survives marker removal, and exercise that rule with a reparented child that retains the old pathname while presenting a readable nonempty replacement environment. This remains the existing cleanup-requires-lifetime-evidence finding under ARCH-ORDER/ARCH-FUNERAL; no new finding is raised.

## Open findings

- **PQ-1** [Important] `cleanup-requires-lifetime-evidence` Define how collection proves detached jobs no longer need scratch
