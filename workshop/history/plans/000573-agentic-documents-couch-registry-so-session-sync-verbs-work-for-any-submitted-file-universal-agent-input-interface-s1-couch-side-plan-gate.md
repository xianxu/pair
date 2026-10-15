---
reviewer_runs:
    - version: 1
      intent:
        id: 6baebbe1-c5fa-46ec-9952-ffe740b7d04a
        repository: github.com/xianxu/pair
        issue_id: "000573"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: codex
        root: /Users/xianxu/workspace/worktree/pair-slot5/pair
        reviewed_head: bf583f84042b7492a88da2d7cfb2fa5aa6004900
        slot: pair:5
        requested_at: "2026-10-15T18:06:21.072079Z"
      phase: finished
      pid: 90250
      started_at: "2026-10-15T18:06:21.454399Z"
      finished_at: "2026-10-15T18:06:49.057475Z"
      session_id: 01a140be-6458-7a91-9764-caea54f76cbb
      outcome: returned
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
    - reviewer_runs:
        - version: 1
          intent:
            id: 01c0e060-257d-44dd-8684-69d0fa8c2f18
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair
            reviewed_head: bf583f84042b7492a88da2d7cfb2fa5aa6004900
            slot: pair:5
            requested_at: "2026-10-15T18:04:38.674538Z"
          phase: finished
          pid: 62286
          started_at: "2026-10-15T18:04:38.977137Z"
          finished_at: "2026-10-15T18:05:09.855769Z"
          session_id: 01a140bc-d433-7991-9270-798d625318dd
          outcome: returned
      "n": 2
      timestamp: "2026-10-15T11:05:10-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Current metadata and history are durably exported at the current path before registry removal, with central authority retained through interrupted handoffs.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Historical intervals persist until forget; capacity refuses further renames without evicting retained evidence or changing ledger retention.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Named function strategies cover adversarial inputs, malformed persisted data, controlled interleavings and publication failures.
          round: 2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: bb0cb94d-d24f-4e4b-92fe-81de74485a65
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair
            reviewed_head: bf583f84042b7492a88da2d7cfb2fa5aa6004900
            slot: pair:5
            requested_at: "2026-10-15T18:05:54.568723Z"
          phase: finished
          pid: 82768
          started_at: "2026-10-15T18:05:54.877103Z"
          finished_at: "2026-10-15T18:06:19.396629Z"
          session_id: 01a140bd-fca4-7711-990d-cc9e591b005e
          outcome: returned
      "n": 3
      timestamp: "2026-10-15T11:06:19-07:00"
      agent: codex
      blocked: false
content_hash: 58e1586912ac84edc90130d9cd8aeac6dd16229610a2c7e61e38050712b4a321
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

## Round 2 — 2026-10-15T11:05:10-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Current metadata and history are durably exported at the current path before registry removal, with central authority retained through interrupted handoffs.
- PQ-2 — addressed — Historical intervals persist until forget; capacity refuses further renames without evicting retained evidence or changing ledger retention.
- PQ-3 — addressed — Named function strategies cover adversarial inputs, malformed persisted data, controlled interleavings and publication failures.

## Round 3 — 2026-10-15T11:06:19-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
