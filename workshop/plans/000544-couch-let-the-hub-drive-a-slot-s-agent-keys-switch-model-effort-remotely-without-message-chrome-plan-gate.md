---
gate: plan-quality
issue: 544
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-14T02:06:18-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define caller cancellation independently of short-lived RPC connections
          detail: 'ARCH-ORDER: the plan requires caller-disconnect cancellation, but every existing RPC normally closes its connection and cancels its handler context after one exchange. Name the operation context owner and the observable cancellation mechanism, distinguishing normal admission/poll closure from caller abandonment without sacrificing read-only lost-reply recovery.'
          family: operation-lifetime-independent-of-rpc
          round: 1
        - id: PQ-2
          severity: Important
          title: Specify isolated authentication for model capture and conformance
          detail: 'ARCH-SECURE: peerLiveLocalAuth restores Claude''s real HOME and native configuration directory at cmd/internal/wrapcmd/peer_live_test.go:420–433. Name the authentication mode or harness change that keeps model-selection writes in disposable configuration, and refuse the native-config fallback for these scenarios before gathering captures.'
          family: live-tests-isolate-writable-user-state
          round: 1
        - id: PQ-3
          severity: Important
          title: Replace prose test-case inventories with named-function strategies
          detail: The test tasks enumerate cases instead of giving one adversarial-input class and mechanical guard per risky function. Compress them into strategies for ParseKeySequence, ReadModelScreen, ReduceDrive, ApplyReceiptEvent and classifyWait; use malformed-input fuzzing and controlled sequence testing with independent invariants, retaining the stateful integration harness.
          family: function-level-test-strategies
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T02:09:22-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Console/wrapper lifetime ownership and registered-session cancellation are explicitly independent of short-lived RPC contexts.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Model capture requires disposable authentication/configuration and explicitly rejects native-HOME/config fallback before launch.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Named-function adversarial strategies and independent invariants supersede the prose inventories while preserving the stateful integration harness.
          round: 2
      blocked: false
content_hash: 857be186189a5d0c76ca7fe22e60da23faa1bb9ac03e60c3b11429746d14b80b
---

# Gate ledger — pair#544 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T02:06:18-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `operation-lifetime-independent-of-rpc` Define caller cancellation independently of short-lived RPC connections
  ARCH-ORDER: the plan requires caller-disconnect cancellation, but every existing RPC normally closes its connection and cancels its handler context after one exchange. Name the operation context owner and the observable cancellation mechanism, distinguishing normal admission/poll closure from caller abandonment without sacrificing read-only lost-reply recovery.
- **PQ-2** [Important] `live-tests-isolate-writable-user-state` Specify isolated authentication for model capture and conformance
  ARCH-SECURE: peerLiveLocalAuth restores Claude's real HOME and native configuration directory at cmd/internal/wrapcmd/peer_live_test.go:420–433. Name the authentication mode or harness change that keeps model-selection writes in disposable configuration, and refuse the native-config fallback for these scenarios before gathering captures.
- **PQ-3** [Important] `function-level-test-strategies` Replace prose test-case inventories with named-function strategies
  The test tasks enumerate cases instead of giving one adversarial-input class and mechanical guard per risky function. Compress them into strategies for ParseKeySequence, ReadModelScreen, ReduceDrive, ApplyReceiptEvent and classifyWait; use malformed-input fuzzing and controlled sequence testing with independent invariants, retaining the stateful integration harness.

## Round 2 — 2026-10-14T02:09:22-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Console/wrapper lifetime ownership and registered-session cancellation are explicitly independent of short-lived RPC contexts.
- PQ-2 — addressed — Model capture requires disposable authentication/configuration and explicitly rejects native-HOME/config fallback before launch.
- PQ-3 — addressed — Named-function adversarial strategies and independent invariants supersede the prose inventories while preserving the stateful integration harness.

## Open findings

(none — every finding has been disposed)
