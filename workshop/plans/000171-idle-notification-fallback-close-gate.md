---
gate: boundary-review
issue: 171
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-09T02:36:02-07:00"
      agent: sdlc
      findings:
        - id: BR-1
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
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: purpose-vs-covered-population
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-09T02:36:02-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Critical
          title: Idle expiry reads its epoch token after the drain, so an opener drained at expiry time gets a false alert and loses its floor
          detail: |-
            wrap.go:2715 reads p.idleTimerToken after the 2701-2708 drain, but the drain runs
            processLifecycleObservation -> syncIdleTimer -> resetIdleTimer, which advances
            p.idleTimerToken in lockstep with the reducer's new IdleToken. A submission or
            bare return drained there therefore matches the stale expiry: reproduced
            deterministically as gen1=1 gen2=2 alerted=true, emitting
            "no agent output for 60s" against a turn opened microseconds earlier and setting
            IdleNotified=true so that turn is left with no floor at all. Snapshot
            token := p.idleTimerToken before the drain loop. ARCH-ORDER.
          family: expiry-token-snapshot-after-mutation
          round: 2
        - id: BR-3
          severity: Important
          title: The test named for the injected expiry/boundary ordering never enters the idle branch; deleting the whole drain keeps every test green
          detail: |-
            Instrumented, TestIdleFloorExpiryYieldsToABoundaryQueuedBeforeIt
            (idle_floor_test.go:134) enters case <-idleTimer.C zero times - the top-level
            lifecycleEvents case consumes the completion 40ms early and syncIdleTimer
            disarms the timer. Removing wrap.go:2701-2708 entirely leaves go test -run Idle
            -count=5 passing, so PQ-2's drain deliverable is pinned by nothing. Extract the
            branch body (e.g. applyIdleExpiry(token uint64)) so the interleaving can be
            injected; that also gives the Critical above a natural failing test. ARCH-ORDER.
          family: unfalsifiable-race-test
          round: 2
        - id: BR-4
          severity: Important
          title: The lifecycle fuzz excludes both new observation kinds, and the invariant it asserts is now false for the idle alert
          detail: |-
            notification_lifecycle_test.go:161 derives kinds from
            raw%byte(ObservationGraceExpired)+1, yielding 1..10, so ObservationIdleExpired
            and ObservationBareReturn are unreachable - a hand-maintained restatement of the
            enum that silently stops covering every kind added after it. Widening it also
            requires restating the property: the idle alert deliberately breaks "notify at
            most once per generation" (alert plus a later real completion). Add an
            observationKindCount sentinel, feed state.IdleToken for the new tokenized kind,
            and split the invariant into one completion plus at most one alert per
            generation. ARCH-PURPOSE.
          family: hand-maintained-enumeration
          round: 2
        - id: BR-5
          severity: Important
          title: README update appears missing for the now-always-on idle notification and PAIR_WRAP_IDLE_S
          detail: |-
            The diff converts an inert knob into an always-armed notification for every
            agent. README's "## Notifications" section describes what Pair forwards and the
            keybindings table documents the comparable PAIR_WRAP_REMAP_RETURN=0 opt-out, but
            neither mentions the idle floor or that PAIR_WRAP_IDLE_S=0 disables it, so a user
            who starts receiving "no agent output for 60s" has no documented way to
            understand or silence it.
          family: docs-gate-user-surface
          round: 2
        - id: BR-6
          severity: Minor
          title: Idle and watchdog deadlines can be co-ready with no precedence rule, so the operator may get the less-informative message
          detail: |-
            Both are 60s and live in separate select cases, so Go picks randomly when they
            expire together; the drain covers channel events, not the lifecycle timer. The
            0.5s emitOuter limiter suppresses the duplicate, so the effect is "no agent
            output for 60s" displacing "agent stopped working". A non-blocking select on
            lifecycleTimer.C before applying the expiry gives the completion precedence.
          family: expiry-ignores-coready-boundary
          round: 2
        - id: BR-7
          severity: Minor
          title: emitPlainCR's nil-profile branch publishes ObservationBareReturn outside the pure decision and without the overlay check
          detail: |-
            wrap.go:1776-1779 hardcodes the submits predicate that decidePlainReturn now
            owns for its four exits, and returns before the pickerActive swap - so if that
            defensive branch ever became reachable it would open a turn on a picker confirm,
            exactly the case submits:false was added to prevent. Dead in production today
            (hasReturnRemap gates every call site). ARCH-DRY.
          family: submits-rule-second-home
          round: 2
        - id: BR-8
          severity: Minor
          title: The timer Stop+drain idiom is now written three times
          detail: |-
            notification_lifecycle.go:284-291, :299-305 and :318-324 are the same block;
            extract func drainStop(t *time.Timer). ARCH-DRY.
          family: duplicated-idiom
          round: 2
        - id: BR-9
          severity: Minor
          title: NotificationLifecycle now carries five booleans whose legal combinations are unwritten
          detail: |-
            Active, Completed, ActivitySeen, GracePending and IdleNotified declare 32 states
            and mean about four; Completed is provably redundant with !Active except to
            distinguish "never opened", which is why the guards disagree across cases
            (Active && !Completed at :146, !Active || Completed at :193, bare Active at
            :140). Collapse to turnState{none, open, closed} with the open-substate flags
            hanging off open. Pre-existing, extended by this diff. ARCH-ORDER.
          family: boolean-constellation-needs-tagged-enum
          round: 2
        - id: BR-10
          severity: Minor
          title: '"no agent output for Ns" reports the configured interval, and non-output observations also reset the deadline'
          detail: |-
            wrap.go:2710 formats p.idleS, and syncIdleTimer resets the deadline on every
            reduced observation - including journal records that arrive with no pane output -
            so the reported silence can be longer than the actual byte-quiet window. Also
            renders as "for 0s" at sub-second intervals.
          family: message-overstates-measurement
          round: 2
        - id: BR-11
          severity: Minor
          title: The idle harness suppresses the real pair slug subprocess with a 1s wall-clock debounce rather than an injected seam
          detail: |-
            idle_floor_test.go:51 sets lastSlug: time.Now(); this is the established repo
            pattern, but it is the first test whose emit happens after a live timer in a
            goroutine, so on a loaded machine the ~960ms margin can lapse and the test spawns
            a real slug run (model call plus session-state write) against the operator's
            machine. An injected spawnSlug func() would make it deterministic. ARCH-MOCK.
          family: test-avoids-seam-not-injects-it
          round: 2
        - id: BR-12
          severity: Minor
          title: Atlas retains the old at-most-once invariant and claims a drain guarantee the code does not provide
          detail: |-
            atlas/architecture.md:783 still says terminals "notify at most once per
            generation" unqualified, which the alert-plus-completion path contradicts; :785
            states the expiry branch drains "so a just-reported turn is never alerted
            against", which the Critical finding shows is false for drained openers. Both
            need a clause once the token fix lands.
          family: atlas-overstates-guarantee
          round: 2
      blocked: true
---

# Gate ledger — pair#171 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T02:36:02-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `purpose-vs-covered-population` Rule for ObservationBareReturn's publish site: submission-reaching branches, not every bare-CR bypass
  2nd in this family, so the deliverable is the rule, not the instance. Rule: publish the
  turn-opening observation only where the CR actually reaches the agent as a submission.
  decidePlainReturn has three bare-CR adapt.Bypass returns (harness_tty.go:91-98
  overlay-active, :120-126 composer-inactive, :128-132 composer-unknown) plus the
  ttyProfile == nil early return at wrap.go:1769-1771 that yields no decision; the plan
  names one. Keying the publish in emitPlainCR (wrap.go:1786-1788) on Bypass alone would
  open a turn on a pair-local picker confirm that submits nothing, producing a spurious
  "no agent output for 60s" — the untrustworthy-notification noise the Spec targets.
  Prevalence: 4 branches, 1 named.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-09T02:36:02-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `expiry-token-snapshot-after-mutation` Idle expiry reads its epoch token after the drain, so an opener drained at expiry time gets a false alert and loses its floor
  wrap.go:2715 reads p.idleTimerToken after the 2701-2708 drain, but the drain runs
  processLifecycleObservation -> syncIdleTimer -> resetIdleTimer, which advances
  p.idleTimerToken in lockstep with the reducer's new IdleToken. A submission or
  bare return drained there therefore matches the stale expiry: reproduced
  deterministically as gen1=1 gen2=2 alerted=true, emitting
  "no agent output for 60s" against a turn opened microseconds earlier and setting
  IdleNotified=true so that turn is left with no floor at all. Snapshot
  token := p.idleTimerToken before the drain loop. ARCH-ORDER.
- **BR-3** [Important] `unfalsifiable-race-test` The test named for the injected expiry/boundary ordering never enters the idle branch; deleting the whole drain keeps every test green
  Instrumented, TestIdleFloorExpiryYieldsToABoundaryQueuedBeforeIt
  (idle_floor_test.go:134) enters case <-idleTimer.C zero times - the top-level
  lifecycleEvents case consumes the completion 40ms early and syncIdleTimer
  disarms the timer. Removing wrap.go:2701-2708 entirely leaves go test -run Idle
  -count=5 passing, so PQ-2's drain deliverable is pinned by nothing. Extract the
  branch body (e.g. applyIdleExpiry(token uint64)) so the interleaving can be
  injected; that also gives the Critical above a natural failing test. ARCH-ORDER.
- **BR-4** [Important] `hand-maintained-enumeration` The lifecycle fuzz excludes both new observation kinds, and the invariant it asserts is now false for the idle alert
  notification_lifecycle_test.go:161 derives kinds from
  raw%byte(ObservationGraceExpired)+1, yielding 1..10, so ObservationIdleExpired
  and ObservationBareReturn are unreachable - a hand-maintained restatement of the
  enum that silently stops covering every kind added after it. Widening it also
  requires restating the property: the idle alert deliberately breaks "notify at
  most once per generation" (alert plus a later real completion). Add an
  observationKindCount sentinel, feed state.IdleToken for the new tokenized kind,
  and split the invariant into one completion plus at most one alert per
  generation. ARCH-PURPOSE.
- **BR-5** [Important] `docs-gate-user-surface` README update appears missing for the now-always-on idle notification and PAIR_WRAP_IDLE_S
  The diff converts an inert knob into an always-armed notification for every
  agent. README's "## Notifications" section describes what Pair forwards and the
  keybindings table documents the comparable PAIR_WRAP_REMAP_RETURN=0 opt-out, but
  neither mentions the idle floor or that PAIR_WRAP_IDLE_S=0 disables it, so a user
  who starts receiving "no agent output for 60s" has no documented way to
  understand or silence it.
- **BR-6** [Minor] `expiry-ignores-coready-boundary` Idle and watchdog deadlines can be co-ready with no precedence rule, so the operator may get the less-informative message
  Both are 60s and live in separate select cases, so Go picks randomly when they
  expire together; the drain covers channel events, not the lifecycle timer. The
  0.5s emitOuter limiter suppresses the duplicate, so the effect is "no agent
  output for 60s" displacing "agent stopped working". A non-blocking select on
  lifecycleTimer.C before applying the expiry gives the completion precedence.
- **BR-7** [Minor] `submits-rule-second-home` emitPlainCR's nil-profile branch publishes ObservationBareReturn outside the pure decision and without the overlay check
  wrap.go:1776-1779 hardcodes the submits predicate that decidePlainReturn now
  owns for its four exits, and returns before the pickerActive swap - so if that
  defensive branch ever became reachable it would open a turn on a picker confirm,
  exactly the case submits:false was added to prevent. Dead in production today
  (hasReturnRemap gates every call site). ARCH-DRY.
- **BR-8** [Minor] `duplicated-idiom` The timer Stop+drain idiom is now written three times
  notification_lifecycle.go:284-291, :299-305 and :318-324 are the same block;
  extract func drainStop(t *time.Timer). ARCH-DRY.
- **BR-9** [Minor] `boolean-constellation-needs-tagged-enum` NotificationLifecycle now carries five booleans whose legal combinations are unwritten
  Active, Completed, ActivitySeen, GracePending and IdleNotified declare 32 states
  and mean about four; Completed is provably redundant with !Active except to
  distinguish "never opened", which is why the guards disagree across cases
  (Active && !Completed at :146, !Active || Completed at :193, bare Active at
  :140). Collapse to turnState{none, open, closed} with the open-substate flags
  hanging off open. Pre-existing, extended by this diff. ARCH-ORDER.
- **BR-10** [Minor] `message-overstates-measurement` "no agent output for Ns" reports the configured interval, and non-output observations also reset the deadline
  wrap.go:2710 formats p.idleS, and syncIdleTimer resets the deadline on every
  reduced observation - including journal records that arrive with no pane output -
  so the reported silence can be longer than the actual byte-quiet window. Also
  renders as "for 0s" at sub-second intervals.
- **BR-11** [Minor] `test-avoids-seam-not-injects-it` The idle harness suppresses the real pair slug subprocess with a 1s wall-clock debounce rather than an injected seam
  idle_floor_test.go:51 sets lastSlug: time.Now(); this is the established repo
  pattern, but it is the first test whose emit happens after a live timer in a
  goroutine, so on a loaded machine the ~960ms margin can lapse and the test spawns
  a real slug run (model call plus session-state write) against the operator's
  machine. An injected spawnSlug func() would make it deterministic. ARCH-MOCK.
- **BR-12** [Minor] `atlas-overstates-guarantee` Atlas retains the old at-most-once invariant and claims a drain guarantee the code does not provide
  atlas/architecture.md:783 still says terminals "notify at most once per
  generation" unqualified, which the alert-plus-completion path contradicts; :785
  states the expiry branch drains "so a just-reported turn is never alerted
  against", which the Critical finding shows is false for drained openers. Both
  need a clause once the token fix lands.

## Open findings

- **BR-1** [Minor] `purpose-vs-covered-population` Rule for ObservationBareReturn's publish site: submission-reaching branches, not every bare-CR bypass
- **BR-2** [Critical] `expiry-token-snapshot-after-mutation` Idle expiry reads its epoch token after the drain, so an opener drained at expiry time gets a false alert and loses its floor
- **BR-3** [Important] `unfalsifiable-race-test` The test named for the injected expiry/boundary ordering never enters the idle branch; deleting the whole drain keeps every test green
- **BR-4** [Important] `hand-maintained-enumeration` The lifecycle fuzz excludes both new observation kinds, and the invariant it asserts is now false for the idle alert
- **BR-5** [Important] `docs-gate-user-surface` README update appears missing for the now-always-on idle notification and PAIR_WRAP_IDLE_S
- **BR-6** [Minor] `expiry-ignores-coready-boundary` Idle and watchdog deadlines can be co-ready with no precedence rule, so the operator may get the less-informative message
- **BR-7** [Minor] `submits-rule-second-home` emitPlainCR's nil-profile branch publishes ObservationBareReturn outside the pure decision and without the overlay check
- **BR-8** [Minor] `duplicated-idiom` The timer Stop+drain idiom is now written three times
- **BR-9** [Minor] `boolean-constellation-needs-tagged-enum` NotificationLifecycle now carries five booleans whose legal combinations are unwritten
- **BR-10** [Minor] `message-overstates-measurement` "no agent output for Ns" reports the configured interval, and non-output observations also reset the deadline
- **BR-11** [Minor] `test-avoids-seam-not-injects-it` The idle harness suppresses the real pair slug subprocess with a 1s wall-clock debounce rather than an injected seam
- **BR-12** [Minor] `atlas-overstates-guarantee` Atlas retains the old at-most-once invariant and claims a drain guarantee the code does not provide
