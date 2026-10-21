---
gate: plan-quality
issue: 759
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 084e967e-ddf6-4fc4-aef4-19263968cd2f
            repository: github.com/xianxu/pair
            issue_id: "000759"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: 9ae7ea3e488b0cd97e2aa9f9bb0270e9e6920589
            slot: pair:6
            requested_at: "2026-10-21T16:08:52.379049Z"
          phase: finished
          pid: 75656
          started_at: "2026-10-21T16:08:52.565349Z"
          finished_at: "2026-10-21T16:10:02.581245Z"
          session_id: 01a15f38-fdb4-7303-987f-6135f4bab39b
          outcome: returned
      "n": 1
      timestamp: "2026-10-21T09:10:02-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Specify preservation and mixed-version behavior for every changed persisted format.
          detail: 'ARCH-EVOLVE: The plan promises unknown-field preservation without naming its codec or read-modify-write mechanism; existing sessionsync readRows decodes into Entry (ledger.go:198), and Ledger.trim rewrites typed rows (ledger.go:148), dropping unknown fields. Name the preservation mechanism for the latch, origin store and changed ledger records, including nested members, strict known-field/required-field/duplicate-key validation, and the older-writer compatibility or justified refusal boundary; require a newer-record round trip through the actual rewrite path.'
          family: persisted-format-preservation-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose case inventories with strategies attached to named functions.
          detail: The Risk functions table names functions only selectively, while M1–M3 enumerate JWT, lifecycle, navigation and provenance cases in prose. Compress these into one adversarial-class/mechanical-guard line per named risky function, including verification, durable publication, session/replay admission, document handoff, reset and provenance projection; retain the generated-sequence invariants and operation-count checks.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#759 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T09:10:02-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `persisted-format-preservation-contract` Specify preservation and mixed-version behavior for every changed persisted format.
  ARCH-EVOLVE: The plan promises unknown-field preservation without naming its codec or read-modify-write mechanism; existing sessionsync readRows decodes into Entry (ledger.go:198), and Ledger.trim rewrites typed rows (ledger.go:148), dropping unknown fields. Name the preservation mechanism for the latch, origin store and changed ledger records, including nested members, strict known-field/required-field/duplicate-key validation, and the older-writer compatibility or justified refusal boundary; require a newer-record round trip through the actual rewrite path.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose case inventories with strategies attached to named functions.
  The Risk functions table names functions only selectively, while M1–M3 enumerate JWT, lifecycle, navigation and provenance cases in prose. Compress these into one adversarial-class/mechanical-guard line per named risky function, including verification, durable publication, session/replay admission, document handoff, reset and provenance projection; retain the generated-sequence invariants and operation-count checks.

## Open findings

- **PQ-1** [Important] `persisted-format-preservation-contract` Specify preservation and mixed-version behavior for every changed persisted format.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose case inventories with strategies attached to named functions.
