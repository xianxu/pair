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

## Open findings

- **BR-1** [Critical] `effect-outcome-truthfulness` Notify reports success when the presenter drops its output during a stall
- **BR-2** [Important] `acceptance-scope-traceability` Done-when omits the approved mixed-version compatibility behavior
