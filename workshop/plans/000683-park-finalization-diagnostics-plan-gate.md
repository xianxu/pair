---
gate: plan-quality
issue: 683
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: ebbb13e4-f575-4dc6-9c48-7ee27188829b
            repository: github.com/xianxu/pair
            issue_id: "000683"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair
            reviewed_head: 02cf1e8e4ec798e60f32b0b0ba98dcef91f494c7
            slot: pair:5
            requested_at: "2026-10-19T09:20:06.051435Z"
          phase: finished
          pid: 31254
          started_at: "2026-10-19T09:20:06.177206Z"
          finished_at: "2026-10-19T09:21:28.136191Z"
          session_id: 01a15376-0707-71a2-8e30-2442c7675920
          outcome: returned
      "n": 1
      timestamp: "2026-10-19T02:21:28-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Define late failure publication after a newer successful attempt
          detail: 'ARCH-ORDER: Plan sections 4–5 allow asynchronous terminal completion but supersede only failures already present. If older attempt A awaits readiness, newer B succeeds with no unresolved history and writes nothing, then A fails, RecordFailure can expose stale unresolved evidence. Name the authoritative ordering and bounded state that makes this sequence deterministic; specify how late publication becomes superseded without relying on timestamp resolution.'
          family: terminal-order-supersession
          round: 1
        - id: PQ-2
          severity: Important
          title: Reconcile existing launch capture writers with the sole Finish recorder
          detail: 'ARCH-DRY/ARCH-ORDER: Plan section 4 requires sole publication through OperationAttempt.Finish and exactly one publication, yet also enriches an existing captured launch row. captureLaunchFailure saves at cmd/internal/couchcore/launchfailure_capture.go:129 and finishSwitchFailure saves again at :157. Specify their evidence-only/deferred behavior under an owned attempt, any necessary standalone fallback, and how the immutable operation nonce relates to the launch nonce.'
          family: single-publication-ownership
          round: 1
        - id: PQ-3
          severity: Important
          title: Define safe holder command evidence before persisting it
          detail: 'ARCH-SECURE: Plan section 3 introduces persisted holder command lines with clipping but no policy excluding credentials or conversational arguments, despite the no-secrets contract. Specify an allowlisted command summary or another structurally safe representation, identify where it is produced and enforced, and name its adversarial test strategy; length bounds alone do not establish safe provenance.'
          family: diagnostic-secret-boundary
          round: 1
        - id: PQ-4
          severity: Important
          title: Replace prose test inventories with named function strategies
          detail: Section 7 Task 1 enumerates cases beginning “Cover 64 maximum-size entries...” while the strategy table leaves several functions unnamed as “preserving decoder/encoder, normalize/clone” and “cached projection / inventory.” Compress the repeated inventories into one adversarial strategy and mechanical guard per risky named function, retaining integration entry points and verification commands.
          family: function-test-strategy-contract
          round: 1
      blocked: true
---

# Gate ledger — pair#683 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T02:21:28-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `terminal-order-supersession` Define late failure publication after a newer successful attempt
  ARCH-ORDER: Plan sections 4–5 allow asynchronous terminal completion but supersede only failures already present. If older attempt A awaits readiness, newer B succeeds with no unresolved history and writes nothing, then A fails, RecordFailure can expose stale unresolved evidence. Name the authoritative ordering and bounded state that makes this sequence deterministic; specify how late publication becomes superseded without relying on timestamp resolution.
- **PQ-2** [Important] `single-publication-ownership` Reconcile existing launch capture writers with the sole Finish recorder
  ARCH-DRY/ARCH-ORDER: Plan section 4 requires sole publication through OperationAttempt.Finish and exactly one publication, yet also enriches an existing captured launch row. captureLaunchFailure saves at cmd/internal/couchcore/launchfailure_capture.go:129 and finishSwitchFailure saves again at :157. Specify their evidence-only/deferred behavior under an owned attempt, any necessary standalone fallback, and how the immutable operation nonce relates to the launch nonce.
- **PQ-3** [Important] `diagnostic-secret-boundary` Define safe holder command evidence before persisting it
  ARCH-SECURE: Plan section 3 introduces persisted holder command lines with clipping but no policy excluding credentials or conversational arguments, despite the no-secrets contract. Specify an allowlisted command summary or another structurally safe representation, identify where it is produced and enforced, and name its adversarial test strategy; length bounds alone do not establish safe provenance.
- **PQ-4** [Important] `function-test-strategy-contract` Replace prose test inventories with named function strategies
  Section 7 Task 1 enumerates cases beginning “Cover 64 maximum-size entries...” while the strategy table leaves several functions unnamed as “preserving decoder/encoder, normalize/clone” and “cached projection / inventory.” Compress the repeated inventories into one adversarial strategy and mechanical guard per risky named function, retaining integration entry points and verification commands.

## Open findings

- **PQ-1** [Important] `terminal-order-supersession` Define late failure publication after a newer successful attempt
- **PQ-2** [Important] `single-publication-ownership` Reconcile existing launch capture writers with the sole Finish recorder
- **PQ-3** [Important] `diagnostic-secret-boundary` Define safe holder command evidence before persisting it
- **PQ-4** [Important] `function-test-strategy-contract` Replace prose test inventories with named function strategies
