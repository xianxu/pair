---
gate: boundary-review
issue: 540
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T20:09:56-07:00"
      dispatched: "2026-10-13T20:07:26-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Notify reports success when the presenter drops its output during a stall
          detail: cmd/internal/terminal/presenter.go:1057 delegates to writeUnlessStalled, which returns nil without writing when already stalled and also hides recoverable write-timeout errors. The broker consequently returns ok and the CLI claims the notification was written. Preserve presenter recovery but propagate a non-success notification outcome; add stalled-before-notify and stalls-during-notify regressions with no replay after recovery. ARCH-ORDER, ARCH-PURPOSE.
          family: effect-outcome-truthfulness
          round: 1
        - id: BR-2
          severity: Important
          title: Done-when omits the approved mixed-version compatibility behavior
          detail: workshop/issues/000540-couch-swallow-slots-osc-desktop-notifications-add-couch-notify-so-the-hub-tl-decides-what-reaches-the-operator.md:36 covers only notifications, while workshop/plans/000540-hub-notifications-plan.md:225 adds legacy messaging compatibility and recovery diagnostics. Add acceptance criteria for existing verbs against the old request contract, missing-registration and unsupported-notify guidance, and preservation of one-shot/no-retry behavior. ARCH-PURPOSE.
          family: acceptance-scope-traceability
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T20:28:11-07:00"
      dispatched: "2026-10-13T20:25:29-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: presenter.go:318 preserves stall recovery while returning non-success to Notify. desktop_notify_test.go:148 covers both pre-existing and notification-induced stalls, recovery, and no replay. Routing Notify back through writeUnlessStalled in a temporary pinned-head copy made both regressions fail; restoring the fix made them pass.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Issue #540 Done-when lines 37–42 now require legacy actors/send/status compatibility, accurate restart guidance, single-exchange behavior, and truthful notification outcomes. These match the plan''s mixed-version revision and messages_compat_test.go:34; the compatibility regression passed, and its frozen request shape matches the referenced pre-540 source.'
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 0dcc138b6429a7ef620a27723e15e89420d94efe
      blocked: false
---

# Gate ledger — pair#540 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T20:09:56-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `effect-outcome-truthfulness` Notify reports success when the presenter drops its output during a stall
  cmd/internal/terminal/presenter.go:1057 delegates to writeUnlessStalled, which returns nil without writing when already stalled and also hides recoverable write-timeout errors. The broker consequently returns ok and the CLI claims the notification was written. Preserve presenter recovery but propagate a non-success notification outcome; add stalled-before-notify and stalls-during-notify regressions with no replay after recovery. ARCH-ORDER, ARCH-PURPOSE.
- **BR-2** [Important] `acceptance-scope-traceability` Done-when omits the approved mixed-version compatibility behavior
  workshop/issues/000540-couch-swallow-slots-osc-desktop-notifications-add-couch-notify-so-the-hub-tl-decides-what-reaches-the-operator.md:36 covers only notifications, while workshop/plans/000540-hub-notifications-plan.md:225 adds legacy messaging compatibility and recovery diagnostics. Add acceptance criteria for existing verbs against the old request contract, missing-registration and unsupported-notify guidance, and preservation of one-shot/no-retry behavior. ARCH-PURPOSE.

## Round 2 — 2026-10-13T20:28:11-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — presenter.go:318 preserves stall recovery while returning non-success to Notify. desktop_notify_test.go:148 covers both pre-existing and notification-induced stalls, recovery, and no replay. Routing Notify back through writeUnlessStalled in a temporary pinned-head copy made both regressions fail; restoring the fix made them pass.
- BR-2 — addressed — Issue #540 Done-when lines 37–42 now require legacy actors/send/status compatibility, accurate restart guidance, single-exchange behavior, and truthful notification outcomes. These match the plan's mixed-version revision and messages_compat_test.go:34; the compatibility regression passed, and its frozen request shape matches the referenced pre-540 source.

## Open findings

(none — every finding has been disposed)
