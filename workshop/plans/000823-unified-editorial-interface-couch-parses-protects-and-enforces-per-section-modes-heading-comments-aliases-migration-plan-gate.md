---
gate: plan-quality
issue: 823
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 240d2e2b-c716-4655-a274-8b1816830711
            repository: github.com/xianxu/pair
            issue_id: "000823"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot9/pair
            reviewed_head: 9be1517dc688ef198ad5884eb66cd20dcd7c83ea
            slot: pair:9
            requested_at: "2026-10-23T10:36:48.402809Z"
          phase: finished
          pid: 32988
          started_at: "2026-10-23T10:36:48.595628Z"
          finished_at: "2026-10-23T10:38:51.402996Z"
          session_id: 01a16855-b204-7ea1-aa9c-807df3683ad5
          outcome: returned
      "n": 1
      timestamp: "2026-10-23T03:38:51-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Replace test-case inventories with named risky-function strategies
          detail: Tasks 1–5 enumerate prose test cases while leaving most compiler, classifier, scoped-hash and codec functions unnamed. Name the unit-tested entry points and give each risky function one adversarial-input/mechanical-guard strategy, including malformed-input fuzzing, migration preservation properties and controlled transaction ordering; compress the case inventories.
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Important
          title: Specify duplicate-key rejection for persisted presentation metadata
          detail: 'ARCH-EVOLVE: the round/span/review-record contract specifies known types, enums and extension preservation but omits duplicate-key handling. Current decodeRound uses json.Unmarshal, which accepts duplicates; name the strict decoding boundary and require duplicate rejection alongside preservation of unknown nested members through rewrites.'
          family: strict-persisted-format-validation
          round: 1
      blocked: true
---

# Gate ledger — pair#823 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T03:38:51-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named risky-function strategies
  Tasks 1–5 enumerate prose test cases while leaving most compiler, classifier, scoped-hash and codec functions unnamed. Name the unit-tested entry points and give each risky function one adversarial-input/mechanical-guard strategy, including malformed-input fuzzing, migration preservation properties and controlled transaction ordering; compress the case inventories.
- **PQ-2** [Important] `strict-persisted-format-validation` Specify duplicate-key rejection for persisted presentation metadata
  ARCH-EVOLVE: the round/span/review-record contract specifies known types, enums and extension preservation but omits duplicate-key handling. Current decodeRound uses json.Unmarshal, which accepts duplicates; name the strict decoding boundary and require duplicate rejection alongside preservation of unknown nested members through rewrites.

## Open findings

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named risky-function strategies
- **PQ-2** [Important] `strict-persisted-format-validation` Specify duplicate-key rejection for persisted presentation metadata
