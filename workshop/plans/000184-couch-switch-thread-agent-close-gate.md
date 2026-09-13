---
gate: boundary-review
issue: 184
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-13T13:30:28-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Forwarding a terminal reply can lose the orientation submit deadline
          detail: cmd/internal/wrapcmd/wrap.go:1498 consumes the settle tick before jumping to queued input. A solicited reply does not cancel delivery, but forwarding it never rearms the timer, leaving DeliverySettling until timeout. Preserve the pending settle event and test simultaneous reply/deadline readiness deterministically (ARCH-ORDER).
          family: input-priority-preserves-pending-events
          round: 1
        - id: BR-2
          severity: Important
          title: Full-chain acceptance omits successful live-source switching
          detail: cmd/internal/couchcmd/switchagent_acceptance_test.go:22 seeds an already verified park, bypassing teardown and exact TTY capture transfer. Add a successful owned-live-source case carrying the actual park descriptor through launch and wrapper delivery, with preserved tag-owned artifacts (ARCH-PURPOSE, ARCH-MOCK).
          family: acceptance-covers-lifecycle-composition
          round: 1
      blocked: true
---

# Gate ledger — pair#184 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T13:30:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `input-priority-preserves-pending-events` Forwarding a terminal reply can lose the orientation submit deadline
  cmd/internal/wrapcmd/wrap.go:1498 consumes the settle tick before jumping to queued input. A solicited reply does not cancel delivery, but forwarding it never rearms the timer, leaving DeliverySettling until timeout. Preserve the pending settle event and test simultaneous reply/deadline readiness deterministically (ARCH-ORDER).
- **BR-2** [Important] `acceptance-covers-lifecycle-composition` Full-chain acceptance omits successful live-source switching
  cmd/internal/couchcmd/switchagent_acceptance_test.go:22 seeds an already verified park, bypassing teardown and exact TTY capture transfer. Add a successful owned-live-source case carrying the actual park descriptor through launch and wrapper delivery, with preserved tag-owned artifacts (ARCH-PURPOSE, ARCH-MOCK).

## Open findings

- **BR-1** [Critical] `input-priority-preserves-pending-events` Forwarding a terminal reply can lose the orientation submit deadline
- **BR-2** [Important] `acceptance-covers-lifecycle-composition` Full-chain acceptance omits successful live-source switching
