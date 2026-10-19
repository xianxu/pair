---
gate: plan-quality
issue: 687
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: a291a253-0545-49eb-8ee5-1dc7470c6c33
            repository: github.com/xianxu/pair
            issue_id: "000687"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot3/pair
            reviewed_head: c1c1f13b1c000c79708b5d4c827c6aaf26ee69d3
            slot: pair:3
            requested_at: "2026-10-19T04:24:39.554749Z"
          phase: finished
          pid: 80206
          started_at: "2026-10-19T04:24:40.078977Z"
          finished_at: "2026-10-19T04:25:21.192584Z"
          session_id: 01a15267-8ed3-7322-af10-a634acb97ee6
          outcome: returned
      "n": 1
      timestamp: "2026-10-18T21:25:21-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Replace prose test inventories with named-function strategies
          detail: The plan's test rows at workshop/plans/000687-couch-retention-registration-plan.md:83–95 enumerate malformed-input and event cases but do not name the new codec, participation predicate, or adapter functions that will be tested. Name each risky production function and give one adversarial-input or event-sequence strategy plus its mechanical guard, covering ReadRuntimes/RegisterRuntime and production composition as well; compress the existing inventories instead of appending another checklist (ARCH-PURE, ARCH-SECURE, ARCH-ORDER).
          family: function-level-test-contract
          round: 1
      blocked: true
---

# Gate ledger — pair#687 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T21:25:21-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-contract` Replace prose test inventories with named-function strategies
  The plan's test rows at workshop/plans/000687-couch-retention-registration-plan.md:83–95 enumerate malformed-input and event cases but do not name the new codec, participation predicate, or adapter functions that will be tested. Name each risky production function and give one adversarial-input or event-sequence strategy plus its mechanical guard, covering ReadRuntimes/RegisterRuntime and production composition as well; compress the existing inventories instead of appending another checklist (ARCH-PURE, ARCH-SECURE, ARCH-ORDER).

## Open findings

- **PQ-1** [Important] `function-level-test-contract` Replace prose test inventories with named-function strategies
