---
gate: plan-quality
issue: 755
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: bc1d3891-3f03-4ba0-bbc2-5d1b79df4fd5
            repository: github.com/xianxu/pair
            issue_id: "000755"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot4/pair
            reviewed_head: 6a2869e12192a75894f978aaa647135180bc28bf
            slot: pair:4
            requested_at: "2026-10-21T05:55:17.35279Z"
          phase: finished
          pid: 72959
          started_at: "2026-10-21T05:55:17.499688Z"
          finished_at: "2026-10-21T05:56:45.287469Z"
          session_id: 01a15d07-3ccf-7023-ade8-9f85cedd4787
          outcome: returned
      "n": 1
      timestamp: "2026-10-20T22:56:45-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Define how a loaded document establishes its revision baseline
          detail: 'ARCH-ORDER: The plan specifies server revision transitions but no document bootstrap contract. Define how served HTML carries a conservative revision/version baseline and how the client compares its first observation, so an edit or selection change before the first poll cannot be silently adopted without reload/navigation; name the production decision function and a controllable-ordering strategy that verifies this invariant.'
          family: document-revision-ordering
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace the prose case inventory with production function strategies
          detail: The test table at workshop/plans/000755-mock-server-plan.md:72–83 enumerates cases under Test names, while the gate requires the production functions to be unit-tested by name. Compress it to one adversarial-input class and mechanical guard per risky function, including parsing, HTML transformation, revision transitions, bounded scanning and browser reload decisions; retain integration qualification separately.
          family: function-level-test-strategy
          round: 1
        - id: PQ-3
          severity: Minor
          title: Bound visited directory entries as well as regular-file contents
          detail: 'ARCH-CONSTRAINTS: The envelope limits regular files and content bytes, but an arbitrarily large empty-directory tree can remain within both limits. State the traversal scope and an entry/directory budget with visible refusal and recovery, and include visited entries in the scanner''s operation-count oracle.'
          family: scan-work-budget
          round: 1
      blocked: true
---

# Gate ledger — pair#755 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T22:56:45-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `document-revision-ordering` Define how a loaded document establishes its revision baseline
  ARCH-ORDER: The plan specifies server revision transitions but no document bootstrap contract. Define how served HTML carries a conservative revision/version baseline and how the client compares its first observation, so an edit or selection change before the first poll cannot be silently adopted without reload/navigation; name the production decision function and a controllable-ordering strategy that verifies this invariant.
- **PQ-2** [Important] `function-level-test-strategy` Replace the prose case inventory with production function strategies
  The test table at workshop/plans/000755-mock-server-plan.md:72–83 enumerates cases under Test names, while the gate requires the production functions to be unit-tested by name. Compress it to one adversarial-input class and mechanical guard per risky function, including parsing, HTML transformation, revision transitions, bounded scanning and browser reload decisions; retain integration qualification separately.
- **PQ-3** [Minor] `scan-work-budget` Bound visited directory entries as well as regular-file contents
  ARCH-CONSTRAINTS: The envelope limits regular files and content bytes, but an arbitrarily large empty-directory tree can remain within both limits. State the traversal scope and an entry/directory budget with visible refusal and recovery, and include visited entries in the scanner's operation-count oracle.

## Open findings

- **PQ-1** [Important] `document-revision-ordering` Define how a loaded document establishes its revision baseline
- **PQ-2** [Important] `function-level-test-strategy` Replace the prose case inventory with production function strategies
- **PQ-3** [Minor] `scan-work-budget` Bound visited directory entries as well as regular-file contents
