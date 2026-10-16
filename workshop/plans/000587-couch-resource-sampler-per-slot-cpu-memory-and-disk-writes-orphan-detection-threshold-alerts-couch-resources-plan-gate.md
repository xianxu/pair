---
gate: plan-quality
issue: 587
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 70af708a-39af-42c9-beb8-bcc2d5acb4c7
            repository: github.com/xianxu/pair
            issue_id: "000587"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot1/pair
            reviewed_head: 6755511e25e3e1680ea34b8974d84faadc9ee773
            slot: pair:1
            requested_at: "2026-10-16T00:02:52.147318Z"
          phase: finished
          pid: 13529
          started_at: "2026-10-16T00:02:52.33455Z"
          finished_at: "2026-10-16T00:03:56.528375Z"
          session_id: 01a14204-ca62-74d0-8a42-3e78dea248ce
          outcome: returned
      "n": 1
      timestamp: "2026-10-15T17:03:56-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define alert authority when the originating slot is disconnected or retired
          detail: 'ARCH-PURPOSE / ARCH-MOCK: sendGenerated delegates to Broker.Send, which requires a connected sender (cmd/internal/couchcmd/supervision.go:120; cmd/internal/couchmessage/broker.go:385,528), while proven orphans can outlive their originating slot. Name the authorized sender and hub-resolution policy for this state, preserving original process provenance without impersonating a replacement slot; verify delivery through the real broker with the original binding disconnected.'
          family: supervisor-alert-authority
          round: 1
        - id: PQ-2
          severity: Important
          title: Specify the immutable delivery envelope and reconciliation policy for uncertain sends
          detail: 'ARCH-ORDER: the plan combines same-ID retries with current hub resolution and changing metric text, but Broker.Send rejects a reused ID whose sender, body or route differs (cmd/internal/couchmessage/broker.go:532). Specify what envelope is persisted before submission, how receipts are recovered after restart, and what happens when ownership changes or receipt evidence is unavailable; missing evidence must not authorize a duplicate delivery.'
          family: stable-delivery-envelope
          round: 1
        - id: PQ-3
          severity: Important
          title: Replace prose test-case inventories with named function strategies
          detail: Tasks 1–5 and the revision enumerate test cases, while only AdvanceResources is clearly named as a function under test. Compress these inventories into the risky function or method names with one adversarial-input or event-sequence strategy and mechanical guard per function, covering the native adapter, reducer, store validation, policy parsing and worker lifecycle; retain separate integration and live qualification requirements.
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#587 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T17:03:56-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `supervisor-alert-authority` Define alert authority when the originating slot is disconnected or retired
  ARCH-PURPOSE / ARCH-MOCK: sendGenerated delegates to Broker.Send, which requires a connected sender (cmd/internal/couchcmd/supervision.go:120; cmd/internal/couchmessage/broker.go:385,528), while proven orphans can outlive their originating slot. Name the authorized sender and hub-resolution policy for this state, preserving original process provenance without impersonating a replacement slot; verify delivery through the real broker with the original binding disconnected.
- **PQ-2** [Important] `stable-delivery-envelope` Specify the immutable delivery envelope and reconciliation policy for uncertain sends
  ARCH-ORDER: the plan combines same-ID retries with current hub resolution and changing metric text, but Broker.Send rejects a reused ID whose sender, body or route differs (cmd/internal/couchmessage/broker.go:532). Specify what envelope is persisted before submission, how receipts are recovered after restart, and what happens when ownership changes or receipt evidence is unavailable; missing evidence must not authorize a duplicate delivery.
- **PQ-3** [Important] `function-level-test-strategy` Replace prose test-case inventories with named function strategies
  Tasks 1–5 and the revision enumerate test cases, while only AdvanceResources is clearly named as a function under test. Compress these inventories into the risky function or method names with one adversarial-input or event-sequence strategy and mechanical guard per function, covering the native adapter, reducer, store validation, policy parsing and worker lifecycle; retain separate integration and live qualification requirements.

## Open findings

- **PQ-1** [Important] `supervisor-alert-authority` Define alert authority when the originating slot is disconnected or retired
- **PQ-2** [Important] `stable-delivery-envelope` Specify the immutable delivery envelope and reconciliation policy for uncertain sends
- **PQ-3** [Important] `function-level-test-strategy` Replace prose test-case inventories with named function strategies
