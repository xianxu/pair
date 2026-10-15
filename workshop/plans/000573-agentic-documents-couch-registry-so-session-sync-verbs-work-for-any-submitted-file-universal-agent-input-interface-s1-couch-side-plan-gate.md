---
gate: plan-quality
issue: 573
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 48fefd54-1ffc-49be-940e-0e610e35ca12
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair
            reviewed_head: 5ed33952a812c1390f1512fec064b2301879ce54
            slot: pair:5
            requested_at: "2026-10-15T18:01:40.657019Z"
          phase: finished
          pid: 38651
          started_at: "2026-10-15T18:01:40.879309Z"
          finished_at: "2026-10-15T18:02:50.991758Z"
          session_id: 01a140ba-1c00-70d0-9c5b-08da3d2bf056
          outcome: returned
      "n": 1
      timestamp: "2026-10-15T11:02:51-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Preserve current metadata when forgetting a registered legacy document
          detail: 'ARCH-ORDER: The plan imports legacy state/shadow, leaves the original companions untouched, then deletes central state on forget while allowing typed session-sync files to continue. This reactivates stale folds, gap acceptance, LastWrite and shadow. Specify a crash-safe handoff of current metadata back to the legacy route, including after rename, before deleting its authoritative central copy.'
          family: authority-handoff-preserves-state
          round: 1
        - id: PQ-2
          severity: Important
          title: Historical-path pruning cannot assume ledger rows expire after fourteen days
          detail: 'ARCH-PURPOSE / ARCH-FUNERAL: ledger.go:124 only trims above the size threshold; ledger.go:171 and owed.go:58 do not expire rows by age. Define pruning against actual retained evidence, or explicitly design an expiration-policy change, so removing aliases cannot silently remove still-owed historical submits.'
          family: retention-follows-live-evidence
          round: 1
        - id: PQ-3
          severity: Important
          title: Replace prose test inventories with named function strategies
          detail: Tasks 1–3 enumerate test cases, while registered submit matching and operator lock age remain unnamed functions and the persisted-input parser has no malformed-input strategy. Compress these lists into named unit-test targets with one adversarial input class and mechanical guard per risky function; retain controlled barriers for ordering and publication fault injection.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#573 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T11:02:51-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `authority-handoff-preserves-state` Preserve current metadata when forgetting a registered legacy document
  ARCH-ORDER: The plan imports legacy state/shadow, leaves the original companions untouched, then deletes central state on forget while allowing typed session-sync files to continue. This reactivates stale folds, gap acceptance, LastWrite and shadow. Specify a crash-safe handoff of current metadata back to the legacy route, including after rename, before deleting its authoritative central copy.
- **PQ-2** [Important] `retention-follows-live-evidence` Historical-path pruning cannot assume ledger rows expire after fourteen days
  ARCH-PURPOSE / ARCH-FUNERAL: ledger.go:124 only trims above the size threshold; ledger.go:171 and owed.go:58 do not expire rows by age. Define pruning against actual retained evidence, or explicitly design an expiration-policy change, so removing aliases cannot silently remove still-owed historical submits.
- **PQ-3** [Important] `function-level-test-strategy` Replace prose test inventories with named function strategies
  Tasks 1–3 enumerate test cases, while registered submit matching and operator lock age remain unnamed functions and the persisted-input parser has no malformed-input strategy. Compress these lists into named unit-test targets with one adversarial input class and mechanical guard per risky function; retain controlled barriers for ordering and publication fault injection.

## Open findings

- **PQ-1** [Important] `authority-handoff-preserves-state` Preserve current metadata when forgetting a registered legacy document
- **PQ-2** [Important] `retention-follows-live-evidence` Historical-path pruning cannot assume ledger rows expire after fourteen days
- **PQ-3** [Important] `function-level-test-strategy` Replace prose test inventories with named function strategies
