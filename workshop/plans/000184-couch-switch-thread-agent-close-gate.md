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
    - "n": 2
      timestamp: "2026-09-13T14:13:32-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestOrientationDueSubmitSurvivesPrioritizedInput deterministically queues input after timer receipt. It passes at HEAD and fails when pending-event preservation is removed through a scratch overlay.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The live-source acceptance exercises cleanup, actual archive preservation, verified park, launch transport and wrapper delivery. Removing completion.Scrollback transfer makes it fail on both park metadata and prompt contents.
          round: 2
      findings:
        - id: BR-3
          severity: Important
          title: Retry-history lookup is unbounded and ignores cancellation
          detail: cmd/internal/pairlifecycle/store.go:174 scans every integer below the persisted attempt while holding the cleanup lock; missing files continue the scan without checking context. Any positive attempt is accepted. A bounded scratch test with attempt 1000000000 confirmed ten reads after cancellation before an injected error stopped it. Bound history discovery independently of the numeric counter, honor cancellation throughout, and add sparse-history/cancellation regressions (ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER).
          family: persisted-counters-bound-work
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-13T14:21:57-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: store.go:188-223 limits prior-completion reads to 32 and checks cancellation before further reads or cleanup. Sparse-counter and cancellation regressions pass at HEAD and fail using the parent store.go through a scratch Go overlay. Lifecycle race tests pass.
          round: 3
        - id: BR-1
          disposition: addressed
          note: Retained prior disposition. Pending submission survives solicited terminal replies; scheduler regression coverage remains present and the wrapper suite passes.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Retained prior disposition. Full-chain acceptance includes owned live-source cleanup, exact archive transfer, verified park, and delivery through launcher/layout/wrapper; acceptance passes.
          round: 3
      blocked: false
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

## Round 2 — 2026-09-13T14:13:32-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — TestOrientationDueSubmitSurvivesPrioritizedInput deterministically queues input after timer receipt. It passes at HEAD and fails when pending-event preservation is removed through a scratch overlay.
- BR-2 — addressed — The live-source acceptance exercises cleanup, actual archive preservation, verified park, launch transport and wrapper delivery. Removing completion.Scrollback transfer makes it fail on both park metadata and prompt contents.

### Raised

- **BR-3** [Important] `persisted-counters-bound-work` Retry-history lookup is unbounded and ignores cancellation
  cmd/internal/pairlifecycle/store.go:174 scans every integer below the persisted attempt while holding the cleanup lock; missing files continue the scan without checking context. Any positive attempt is accepted. A bounded scratch test with attempt 1000000000 confirmed ten reads after cancellation before an injected error stopped it. Bound history discovery independently of the numeric counter, honor cancellation throughout, and add sparse-history/cancellation regressions (ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER).

## Round 3 — 2026-09-13T14:21:57-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — store.go:188-223 limits prior-completion reads to 32 and checks cancellation before further reads or cleanup. Sparse-counter and cancellation regressions pass at HEAD and fail using the parent store.go through a scratch Go overlay. Lifecycle race tests pass.
- BR-1 — addressed — Retained prior disposition. Pending submission survives solicited terminal replies; scheduler regression coverage remains present and the wrapper suite passes.
- BR-2 — addressed — Retained prior disposition. Full-chain acceptance includes owned live-source cleanup, exact archive transfer, verified park, and delivery through launcher/layout/wrapper; acceptance passes.

## Open findings

(none — every finding has been disposed)
