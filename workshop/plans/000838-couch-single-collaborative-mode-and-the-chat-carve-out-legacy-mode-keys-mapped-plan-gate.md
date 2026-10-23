---
gate: plan-quality
issue: 838
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: e5882ded-f915-43df-958d-c101f7442397
            repository: github.com/xianxu/pair
            issue_id: "000838"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot3/pair
            reviewed_head: 6d5858a863c893a3e65ccc6d9c7835ec378743e4
            slot: pair:3
            requested_at: "2026-10-23T21:24:59.436235Z"
          phase: finished
          pid: 47731
          started_at: "2026-10-23T21:24:59.5829Z"
          finished_at: "2026-10-23T21:25:57.921733Z"
          session_id: 01a16aa7-1fd8-7630-9171-e18b111edef2
          outcome: returned
      "n": 1
      timestamp: "2026-10-23T14:26:21-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Resolve shared scanner provenance loss before relying on it for item grammar
          detail: 'ARCH-DRY / ARCH-PURPOSE: the cited #575 candidate''s pkg/sidedoc/chat.go:141–160 emits only Source "tag" when a heading also has a legacy map/comment authority, losing the legacy provenance required by the plan''s bullet-item contract. Specify a shared API correction and its dependency gate that preserves applicable legacy authority alongside tag ownership, with an independent grammar invariant; #576''s stated notice/performance scope does not cover this correction.'
          family: shared-authority-preserves-provenance
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with per-function adversarial strategies
          detail: The execution tasks at workshop/plans/000838-collaborative-chat-plan.md:115, :123 and :125 enumerate test cases instead of providing the required function-scoped strategies. Compress them into named production surfaces, including Compile, ValidateEdits, selectChatItems, PlanMigration, sectionPolicyHash and the changed round codec functions, each paired with its adversarial input class and mechanical guard; retain fixture replay and verification commands.
          family: function-scoped-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#838 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T14:26:21-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `shared-authority-preserves-provenance` Resolve shared scanner provenance loss before relying on it for item grammar
  ARCH-DRY / ARCH-PURPOSE: the cited #575 candidate's pkg/sidedoc/chat.go:141–160 emits only Source "tag" when a heading also has a legacy map/comment authority, losing the legacy provenance required by the plan's bullet-item contract. Specify a shared API correction and its dependency gate that preserves applicable legacy authority alongside tag ownership, with an independent grammar invariant; #576's stated notice/performance scope does not cover this correction.
- **PQ-2** [Important] `function-scoped-test-strategy` Replace prose test-case inventories with per-function adversarial strategies
  The execution tasks at workshop/plans/000838-collaborative-chat-plan.md:115, :123 and :125 enumerate test cases instead of providing the required function-scoped strategies. Compress them into named production surfaces, including Compile, ValidateEdits, selectChatItems, PlanMigration, sectionPolicyHash and the changed round codec functions, each paired with its adversarial input class and mechanical guard; retain fixture replay and verification commands.

## Open findings

- **PQ-1** [Important] `shared-authority-preserves-provenance` Resolve shared scanner provenance loss before relying on it for item grammar
- **PQ-2** [Important] `function-scoped-test-strategy` Replace prose test-case inventories with per-function adversarial strategies
