---
gate: plan-quality
issue: 171
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-09T01:52:38-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Routing the idle expiry through complete() spends the floor on a mid-turn pause and disables it for the rest of the turn
          detail: |-
            complete() (notification_lifecycle.go:72-80) sets Active=false/Completed=true, so an
            un-recognized turn that pauses 60s on a long tool call fires once and is then swallowed
            by the Active && !Completed guard forever after — including when the agent actually
            needs input. The Log's own measurement (14.1% of minute-intervals show >=60s byte
            silence) says this is the common case. Whether a timeout is a completion or a
            repeatable alert that leaves the turn open is a change to the reducer's completion
            contract and must be decided in the plan, not in the diff.
          family: timeout-is-not-completion
          round: 1
        - id: PQ-2
          severity: Important
          title: Plan specifies the reducer side of the idle observation but never when the idle timer is armed relative to turn state
          detail: |-
            The timer's epoch is last-byte, not turn-open: nothing re-arms it on
            ObservationUserSubmission and the idleFired latch (wrap.go:2691,2698) leaves it stopped
            after a fire, so a turn opened while the timer is spent has no floor. The
            case <-idleTimer.C branch also lacks the lifecycleEvents drain the chunk branch does
            deliberately at wrap.go:2671-2679, so a queued submission can reduce after the expiry.
            And every other timer observation here is tokenized (notification_lifecycle.go:140,144)
            with arming owned by syncLifecycleTimer; the plan should say why this one is exempt.
          family: timer-arming-not-tied-to-state
          round: 1
        - id: PQ-3
          severity: Important
          title: Floor covers only Alt+Enter submissions; bare-CR submissions open no turn and are neither covered nor named a non-goal
          detail: |-
            ObservationUserSubmission is published only at wrap.go:1836 and wrap.go:1855, both
            Alt+Enter forms. Plain Enter goes through emitPlainCR (wrap.go:1842,1848,1898) and
            emits a bare CR with no observation when the composer gate reports inactive
            (harness_tty.go:120-126) — the menu-answer case from the 2026-09-02 Log. With
            ObservationWorking usually absent (measurement 3), nothing opens a turn there. Cover it
            or record it as an explicit non-goal with the reason.
          family: purpose-vs-covered-population
          round: 1
        - id: PQ-4
          severity: Important
          title: Test rows name no function under test and no seam for the acceptance test
          detail: |-
            "unit tests for both cases" does not name Reduce, and the unrecognized-turn row states
            an outcome with no mechanism — read literally it is a 60s wall-clock or live-pane test.
            Reuse the existing in-process seams: lifecycle_journal_test.go:279-321 drives
            p.masterPump() over an os.Pipe ptmx, and notification_rewriter_test.go:194 shows the
            outerTTYFile + writeTTY emit seam. Add one strategy line for the risky surface — the
            master-loop select — naming its adversarial class: idle expiry racing a queued
            submission and a completion, with arrival order injected rather than sampled.
          family: test-surface-unnamed
          round: 1
        - id: PQ-5
          severity: Minor
          title: No budget stated for the per-chunk timer Stop/Reset the always-armed path adds for every agent
          detail: |-
            The plan cites ARCH-CONSTRAINTS to reject screen-diff hashing on the keystroke-latency
            path but leaves unstated that wrap.go:2681-2692 now runs a Stop+drain+Reset per chunk
            for every agent, where previously idleS==0 skipped it. Negligible by the Log's own
            203-chunks/123s figure, but it should be asserted rather than assumed.
          family: constraint-budget-unstated
          round: 1
        - id: PQ-6
          severity: Minor
          title: No atlas step for the reducer/notify-mode surface this change alters
          detail: |-
            atlas/architecture.md:783 enumerates the reducer's openers and terminals and :965
            describes "idle/native OSC for codex/agy"; both go stale when the "idle" mode value is
            deleted and ObservationIdleExpired is added. The close gate requires the atlas update.
          family: atlas-drift-unplanned
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T01:59:58-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Plan decides idle expiry as an alert (IdleNotified, turn stays open) with rationale.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: syncIdleTimer() + IdleToken + lifecycleEvents drain; idleFired latch deleted.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: ObservationBareReturn covers the composer-inactive bare-CR path.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Reduce named; both existing in-process seams named; injected arrival order.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Budget stated with measured basis (1.7 chunks/s, sub-microsecond Stop/Reset).
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Atlas step names architecture.md:783 and :965; both verified stale-on-change.
          round: 2
      findings:
        - id: PQ-7
          severity: Minor
          title: 'Rule for ObservationBareReturn''s publish site: submission-reaching branches, not every bare-CR bypass'
          detail: |-
            2nd in this family, so the deliverable is the rule, not the instance. Rule: publish the
            turn-opening observation only where the CR actually reaches the agent as a submission.
            decidePlainReturn has three bare-CR adapt.Bypass returns (harness_tty.go:91-98
            overlay-active, :120-126 composer-inactive, :128-132 composer-unknown) plus the
            ttyProfile == nil early return at wrap.go:1769-1771 that yields no decision; the plan
            names one. Keying the publish in emitPlainCR (wrap.go:1786-1788) on Bypass alone would
            open a turn on a pair-local picker confirm that submits nothing, producing a spurious
            "no agent output for 60s" — the untrustworthy-notification noise the Spec targets.
            Prevalence: 4 branches, 1 named.
          family: purpose-vs-covered-population
          round: 2
      blocked: false
content_hash: 89c18d0b981ebb332071afccd6d804a7d58523d149f62d36f49b07d97971406f
---

# Gate ledger — pair#171 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T01:52:38-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `timeout-is-not-completion` Routing the idle expiry through complete() spends the floor on a mid-turn pause and disables it for the rest of the turn
  complete() (notification_lifecycle.go:72-80) sets Active=false/Completed=true, so an
  un-recognized turn that pauses 60s on a long tool call fires once and is then swallowed
  by the Active && !Completed guard forever after — including when the agent actually
  needs input. The Log's own measurement (14.1% of minute-intervals show >=60s byte
  silence) says this is the common case. Whether a timeout is a completion or a
  repeatable alert that leaves the turn open is a change to the reducer's completion
  contract and must be decided in the plan, not in the diff.
- **PQ-2** [Important] `timer-arming-not-tied-to-state` Plan specifies the reducer side of the idle observation but never when the idle timer is armed relative to turn state
  The timer's epoch is last-byte, not turn-open: nothing re-arms it on
  ObservationUserSubmission and the idleFired latch (wrap.go:2691,2698) leaves it stopped
  after a fire, so a turn opened while the timer is spent has no floor. The
  case <-idleTimer.C branch also lacks the lifecycleEvents drain the chunk branch does
  deliberately at wrap.go:2671-2679, so a queued submission can reduce after the expiry.
  And every other timer observation here is tokenized (notification_lifecycle.go:140,144)
  with arming owned by syncLifecycleTimer; the plan should say why this one is exempt.
- **PQ-3** [Important] `purpose-vs-covered-population` Floor covers only Alt+Enter submissions; bare-CR submissions open no turn and are neither covered nor named a non-goal
  ObservationUserSubmission is published only at wrap.go:1836 and wrap.go:1855, both
  Alt+Enter forms. Plain Enter goes through emitPlainCR (wrap.go:1842,1848,1898) and
  emits a bare CR with no observation when the composer gate reports inactive
  (harness_tty.go:120-126) — the menu-answer case from the 2026-09-02 Log. With
  ObservationWorking usually absent (measurement 3), nothing opens a turn there. Cover it
  or record it as an explicit non-goal with the reason.
- **PQ-4** [Important] `test-surface-unnamed` Test rows name no function under test and no seam for the acceptance test
  "unit tests for both cases" does not name Reduce, and the unrecognized-turn row states
  an outcome with no mechanism — read literally it is a 60s wall-clock or live-pane test.
  Reuse the existing in-process seams: lifecycle_journal_test.go:279-321 drives
  p.masterPump() over an os.Pipe ptmx, and notification_rewriter_test.go:194 shows the
  outerTTYFile + writeTTY emit seam. Add one strategy line for the risky surface — the
  master-loop select — naming its adversarial class: idle expiry racing a queued
  submission and a completion, with arrival order injected rather than sampled.
- **PQ-5** [Minor] `constraint-budget-unstated` No budget stated for the per-chunk timer Stop/Reset the always-armed path adds for every agent
  The plan cites ARCH-CONSTRAINTS to reject screen-diff hashing on the keystroke-latency
  path but leaves unstated that wrap.go:2681-2692 now runs a Stop+drain+Reset per chunk
  for every agent, where previously idleS==0 skipped it. Negligible by the Log's own
  203-chunks/123s figure, but it should be asserted rather than assumed.
- **PQ-6** [Minor] `atlas-drift-unplanned` No atlas step for the reducer/notify-mode surface this change alters
  atlas/architecture.md:783 enumerates the reducer's openers and terminals and :965
  describes "idle/native OSC for codex/agy"; both go stale when the "idle" mode value is
  deleted and ObservationIdleExpired is added. The close gate requires the atlas update.

## Round 2 — 2026-09-09T01:59:58-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Plan decides idle expiry as an alert (IdleNotified, turn stays open) with rationale.
- PQ-2 — addressed — syncIdleTimer() + IdleToken + lifecycleEvents drain; idleFired latch deleted.
- PQ-3 — addressed — ObservationBareReturn covers the composer-inactive bare-CR path.
- PQ-4 — addressed — Reduce named; both existing in-process seams named; injected arrival order.
- PQ-5 — addressed — Budget stated with measured basis (1.7 chunks/s, sub-microsecond Stop/Reset).
- PQ-6 — addressed — Atlas step names architecture.md:783 and :965; both verified stale-on-change.

### Raised

- **PQ-7** [Minor] `purpose-vs-covered-population` Rule for ObservationBareReturn's publish site: submission-reaching branches, not every bare-CR bypass
  2nd in this family, so the deliverable is the rule, not the instance. Rule: publish the
  turn-opening observation only where the CR actually reaches the agent as a submission.
  decidePlainReturn has three bare-CR adapt.Bypass returns (harness_tty.go:91-98
  overlay-active, :120-126 composer-inactive, :128-132 composer-unknown) plus the
  ttyProfile == nil early return at wrap.go:1769-1771 that yields no decision; the plan
  names one. Keying the publish in emitPlainCR (wrap.go:1786-1788) on Bypass alone would
  open a turn on a pair-local picker confirm that submits nothing, producing a spurious
  "no agent output for 60s" — the untrustworthy-notification noise the Spec targets.
  Prevalence: 4 branches, 1 named.

## Open findings

- **PQ-7** [Minor] `purpose-vs-covered-population` Rule for ObservationBareReturn's publish site: submission-reaching branches, not every bare-CR bypass
