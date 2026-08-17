---
gate: plan-quality
issue: 115
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-16T13:39:44-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Readiness-gated persistence is left as an optional/ambiguous implementation choice
          detail: The spec requires repo-agent defaults to persist only after launch readiness, but Task 3 Step 4 says to use the existing blocking `LaunchSession` return if no readiness seam exists, or maybe port readiness as a follow-up. Current `LaunchSession` is documented as a blocking fork+wait child in `cmd/internal/launcher/runtime.go:29`, and `runCreate` reaches it only at `cmd/internal/launcher/createflow.go:488`; that return is session exit, not target readiness. The plan needs a concrete readiness boundary and fake/OS seam for this milestone, or it cannot satisfy the Done-when. ARCH-PURPOSE, ARCH-MOCK.
          round: 1
        - id: PQ-2
          severity: Important
          title: Test plan is a prose case inventory instead of the required function strategy contract
          detail: Tasks 1-3 list parser, codec, precedence, and create-flow cases in prose, but the gate requires the functions unit-tested by name plus one line of strategy per risky function. Compress the test plan around functions such as `ParseArgs`, the `AgentDefault` codec helpers, `ScopedPaths.AgentDefault`, `DecideLaunchArgs`, and `runCreate`/`AgentDefaultOps`, with adversarial input classes and mechanical guards instead of enumerated examples.
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-16T13:42:39-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The plan now defines nonce-bound readiness as an M1 deliverable with a shared ReadyRecord, ReadinessOps, wrap-side publication after PTY start, stale-record removal, matching identity/nonce/PID checks, and failed/timeout no-persist behavior.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The plan now names the tested functions and gives one-line adversarial/mechanical test strategies instead of a prose case inventory.
          round: 2
      blocked: false
content_hash: c0b17d51a2c52211f9432a452a1a91b816b00d9bb8a743e46b9a6132d9794adc
---

# Gate ledger — pair#115 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-16T13:39:44-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] Readiness-gated persistence is left as an optional/ambiguous implementation choice
  The spec requires repo-agent defaults to persist only after launch readiness, but Task 3 Step 4 says to use the existing blocking `LaunchSession` return if no readiness seam exists, or maybe port readiness as a follow-up. Current `LaunchSession` is documented as a blocking fork+wait child in `cmd/internal/launcher/runtime.go:29`, and `runCreate` reaches it only at `cmd/internal/launcher/createflow.go:488`; that return is session exit, not target readiness. The plan needs a concrete readiness boundary and fake/OS seam for this milestone, or it cannot satisfy the Done-when. ARCH-PURPOSE, ARCH-MOCK.
- **PQ-2** [Important] Test plan is a prose case inventory instead of the required function strategy contract
  Tasks 1-3 list parser, codec, precedence, and create-flow cases in prose, but the gate requires the functions unit-tested by name plus one line of strategy per risky function. Compress the test plan around functions such as `ParseArgs`, the `AgentDefault` codec helpers, `ScopedPaths.AgentDefault`, `DecideLaunchArgs`, and `runCreate`/`AgentDefaultOps`, with adversarial input classes and mechanical guards instead of enumerated examples.

## Round 2 — 2026-08-16T13:42:39-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The plan now defines nonce-bound readiness as an M1 deliverable with a shared ReadyRecord, ReadinessOps, wrap-side publication after PTY start, stale-record removal, matching identity/nonce/PID checks, and failed/timeout no-persist behavior.
- PQ-2 — addressed — The plan now names the tested functions and gives one-line adversarial/mechanical test strategies instead of a prose case inventory.

## Open findings

(none — every finding has been disposed)
