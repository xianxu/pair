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
    - "n": 3
      timestamp: "2026-09-09T02:57:57-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: submits decided in decidePlainReturn and asserted on all eight exits; publish keyed on it, not on adapt.Bypass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Verified by revert — restoring the post-drain read reddens TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Verified by revert — deleting the drain reddens TestIdleExpiryDrainAppliesAQueuedCompletionFirst.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Sentinel and invariant split landed but are unreachable in go test — see Important I-2; also the sentinel guard test is inverted.
          round: 3
        - id: BR-5
          disposition: addressed
          note: README "The idle floor" section documents the message, the once-per-turn bound and PAIR_WRAP_IDLE_S/=0.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Verified by revert — removing the precedence block reddens TestIdleExpiryGivesTheLifecycleDeadlinePrecedenceWhenBothAreReady.
          round: 3
        - id: BR-7
          disposition: addressed
          note: Nil profile now flows through decidePlainReturn as the zero profile; overlay check no longer skipped.
          round: 3
        - id: BR-8
          disposition: addressed
          note: drainStop extracted; a new duplication appeared in the same diff, raised separately.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: Deliberately deferred and tracked as pair#219; pre-existing pattern, Minor, non-blocking.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: Message half is pinned; the behavioural half (idempotent arming) is pinned by nothing — an unconditional resetIdleTimer keeps the whole package green.
          round: 3
        - id: BR-11
          disposition: addressed
          note: spawnSlug seam injected and exercised by both idle harnesses; no wall-clock debounce left.
          round: 3
        - id: BR-12
          disposition: addressed
          note: Atlas now carries the alert-not-terminal exception and the snapshot-before-drain clause.
          round: 3
      findings:
        - id: BR-13
          severity: Important
          title: Three of this round's deliverables can be deleted with the whole wrapcmd package still green, including the mode gate this issue exists to remove
          detail: |-
            This is the 2nd finding in family `unfalsifiable-race-test`. Do not fix the three
            sites named here — write the enumeration. Rule: every `- [x]` row in the issue's
            `## Plan` is complete only when deleting its code turns a test red; sweep the
            checklist by deletion in this round. Measured against the full package (baseline
            11 environmental pty-child failures, identical in every mutation): re-adding
            `if p.notifyModeActive != "idle" { p.idleS = 0 }` near wrap.go:2386 stays green;
            deleting `if decision.submits { publish ObservationBareReturn }` at
            wrap.go:1807-1809 stays green; replacing the epoch check at
            notification_lifecycle.go:328-330 with an unconditional resetIdleTimer stays green.
            Cause: the idle tests build `proxy` directly with idleS/notifyModeActive set, so
            they never cross the arg-parse seam, and no test drives a plain Enter through
            emitPlainCR. Prevalence: 3 of the Plan's 8 rows, measured.
          family: unfalsifiable-race-test
          round: 3
        - id: BR-14
          severity: Important
          title: The floor never arms under PAIR_WRAP_REMAP_RETURN=0 or for agents outside harnessTTYProfiles, and a bare CR mid-turn cannot re-arm a spent floor
          detail: |-
            This is the 2nd finding in family `purpose-vs-covered-population`. Do not fix one
            site — write the enumeration. Rule: enumerate the (agent x configuration x event)
            populations the floor claims, and confirm each has at least one live turn-opening
            publisher and a re-arm path; atlas/architecture.md:785 "It arms for every agent"
            and README:598-612 must be true of that enumeration or be qualified. Openers
            enumerated: keyboard submission (wrap.go:1858,1877) and bare CR (wrap.go:1808),
            both inside translateChunk and reachable only when hasReturnRemap() (wrap.go:1454);
            progress-OSC Working (claude only, progressOSCAuthorized); transcript-started
            (codex journal only); marker completion synthesising Working (claude only).
            So PAIR_WRAP_REMAP_RETURN=0 publishes no keyboard observation at all — README:112
            still documents that var as a keybinding opt-out only — and any agent outside
            {claude, codex, agy, muse} has zero openers, since agentBasename is just
            filepath.Base(argv[0]) (wrap.go:2334). Third instance, event-shaped:
            notification_lifecycle.go:206-210 makes ObservationBareReturn a no-op inside an
            open turn, so once the single alert has fired (IdleNotified=true, timer disarmed)
            the operator's menu answer leaves that turn with no floor, while a mid-turn
            Alt+Enter re-opens and re-arms. Prevalence: 5 openers, 2 configurations with zero,
            1 of 2 keyboard openers with no re-arm.
          family: purpose-vs-covered-population
          round: 3
        - id: BR-15
          severity: Minor
          title: The lifecycle-expiry application is now written twice, and the outer-TTY sidecar test fixture is at seven sites
          detail: |-
            This is the 2nd finding in family `duplicated-idiom`. Do not fix the instance —
            the rule is that a block appearing a second time in the SAME diff gets extracted,
            which is what produced drainStop for BR-8. Instances: the two-line
            `kind, token := p.lifecycleTimerKind, p.lifecycleTimerToken; processLifecycleObservation(...)`
            at wrap.go:2667-2669 and notification_lifecycle.go:296-299 should be one
            applyLifecycleExpiry(); the `dir/outer` + `dir/outer-path` fixture now appears at
            7 test sites, 2 added here (idle_floor_test.go:33-42 and :143-152), and wants a
            shared newLifecycleTestProxy helper. Prevalence: 2 sites and 7 sites, measured.
          family: duplicated-idiom
          round: 3
        - id: BR-16
          severity: Minor
          title: applyIdleExpiry logs IDLE and traces the expiry before the reducer decides whether the alert applies
          detail: |-
            notification_lifecycle.go:303-305 emits p.debug("IDLE", message) and
            traceWrap("idle", ...) unconditionally, so a token-rejected expiry — including one
            the drain or the lifecycle-precedence branch just invalidated — still records a
            completed idle expiry in the debug log and the trace, which is where a future
            operator will go to reconstruct why an alert did or did not fire.
          family: message-overstates-measurement
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-09T03:23:43-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: Bound is now observationKindCount-1 (1..12, both new kinds reachable); seeds reach each; invariant split into completion-per-generation and alert-per-epoch; walk test covers every declared kind; 3.5M execs pass.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: Deliberately deferred to pair#219, which exists with a real spec; pre-existing pattern, Minor, non-blocking.
          round: 4
        - id: BR-10
          disposition: addressed
          note: Verified by mutation — unconditional resetIdleTimer reddens TestSyncIdleTimerDoesNotRestartAnArmedDeadlineWithinTheSameEpoch; sub-second rendering pinned separately.
          round: 4
        - id: BR-13
          disposition: addressed
          note: 'Sweep reproduced independently: 9 of 11 mutations redden a specific named test. One re-addition mutation still survives; carried forward as a Minor, since re-adding deleted code is not the deletion rule BR-13 stated.'
          round: 4
        - id: BR-14
          disposition: addressed
          note: del-passthrough-publish, del-emitplaincr-publish and del-rearm all redden; I re-verified the population enumeration (pass-through is the exclusive else of hasReturnRemap, overlay unreachable there, ptmx has one writer).
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: Both instances still present — notification_lifecycle.go:304 vs wrap.go:2694, and the outer/outer-path fixture at 7 test sites.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: notification_lifecycle.go:310-311 still log and trace the expiry unconditionally, before the reducer decides whether the alert applies.
          round: 4
      findings:
        - id: BR-17
          severity: Important
          title: README and atlas still promise "at most once per turn", which this round's bare-CR re-arm made false
          detail: |-
            This is the 2nd finding in family `atlas-overstates-guarantee`. The rule, not the
            instance: when a state-machine invariant changes, every restatement of it must be
            swept in the same round — here the fuzz invariant was rewritten from per-generation
            to per-epoch (notification_lifecycle_test.go:196-198) while README.md:605 ("It fires
            at most once per turn") and atlas/architecture.md:783 ("a generation can carry one
            alert plus a later real completion") kept the superseded claim. atlas:785 also still
            says IdleToken is "minted once per turn" and then describes the re-arm that mints a
            second one. Enumerated restatements of the once-per invariant: 3 (fuzz, README,
            atlas x2 clauses); 1 swept, 3 stale.
          family: atlas-overstates-guarantee
          round: 4
        - id: BR-18
          severity: Important
          title: The master-loop exactly-once oracle cannot fail — the 500ms emit limiter masks duplicates inside its 400ms window
          detail: |-
            This is the 3rd finding in family `unfalsifiable-race-test`. Do NOT fix this instance
            by widening the sleep. Rule: a notification-count assertion is falsifiable only when
            the observation window exceeds rateLimitS (wrap.go:78, 500ms) or the limiter's clock
            is injected; emitOuter:652 calls time.Now() directly even though the proxy already
            carries an injectable `now func() time.Time`. Measured: with !state.IdleNotified
            dropped from both the reducer guard and syncIdleTimer, idle_floor_test.go:102 passes;
            re-run with settle(1500ms) it fails with "notifications = 3". Prevalence: 3 count
            assertions in idle_floor_test.go (:102, :207, :231) all inside the limiter window;
            :207 and :231 survive on their content checks, :102 has none.
          family: unfalsifiable-race-test
          round: 4
        - id: BR-19
          severity: Minor
          title: No test crosses the startup wiring seam, so the mode gate can be re-added at its original call site with the package green
          detail: |-
            This is the 2nd finding in family `test-avoids-seam-not-injects-it`. Measured:
            inserting `if p.notifyModeActive != "idle" { p.idleS = 0 }` after wrap.go:2416 leaves
            the package GREEN (failure set identical to baseline). resolveNotifyConfig is
            asserted, but nothing tests the path that consumes it, and Run terminates in
            pty.Start so an end-to-end test is not available in this environment. Rule, not
            instance: p.idleS and p.notifyModeActive should have exactly one assignment site fed
            by resolveNotifyConfig, enforced by a source-scanning test — or the residue accepted
            explicitly in the issue Log.
          family: test-avoids-seam-not-injects-it
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-09-09T03:45:57-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: Deferred to pair#219, which states the class (32 states, ~4 meanings, the three disagreeing guards) faithfully; not re-raising on this issue.
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: 'Re-measured at head: lifecycle-expiry application still at wrap.go:2705 + notification_lifecycle.go:304; outer-path fixture still at 7 test sites.'
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: notification_lifecycle.go:310-311 still logs IDLE and traces the expiry unconditionally, before the reducer decides.
          round: 5
        - id: BR-17
          disposition: addressed
          note: The three sites it named are swept and verified by grep; the un-enumerated source-comment class is raised separately below.
          round: 5
        - id: BR-18
          disposition: addressed
          note: 'Verified both ways: duplicate mutation fails with notifications = 9 under the injected clock, passes with now = time.Now.'
          round: 5
        - id: BR-19
          disposition: addressed
          note: 'Verified by revert: re-inserting the mode gate at its original call site reddens the source-scanning test with "assigned at 2 sites".'
          round: 5
      findings:
        - id: BR-20
          severity: Important
          title: Five in-source restatements of the superseded once-per-turn invariant survived the sweep
          detail: |-
            This is the 3rd finding in this family. The rule, not the instance: the enumeration
            of an invariant's restatements must be DERIVED mechanically (grep the phrasing across
            the tree) rather than inherited from the prior finding's bullet list. BR-17's three
            named sites are correctly swept; its enumeration was incomplete. Measured at head:
            notification_lifecycle.go:37 ("inside an open turn it is a no-op" - false, it re-arms),
            :77 ("at most once per turn ... open() clears it"), :80 ("minted once per turn"),
            :320 ("has not already raised its one alert ... a new turn re-arms it"), :351
            ("minted once per turn by the reducer"). 3 swept, 5 stale. Record the grep in the Log
            so the next invariant change reuses the derivation rather than a hand-list.
          family: atlas-overstates-guarantee
          round: 5
        - id: BR-21
          severity: Important
          title: passThroughChunk re-derives the submission predicate inline and has already diverged - a bracketed paste opens a turn and earns a spurious alert
          detail: |-
            This is the 2nd finding in this family. BR-7 established that "does this input submit?"
            is decided once in decidePlainReturn and carried on returnDecision.submits. BR-14's fix
            planted a third home at wrap.go:1595 as a bare `bytes.IndexByte(data, '\r') >= 0`, with
            no bracketed-paste awareness - its caller hard-sets inPaste = false at wrap.go:1492,
            while translateChunk never reaches emitPlainCR inside a paste. Measured with a probe on
            both paths given the same bytes "ESC[200~line one CR line two CR ESC[201~":
            pass-through published [ObservationBareReturn], remap path published nothing.
            Failure scenario: under PAIR_WRAP_REMAP_RETURN=0, or for any agent outside the four
            harnessTTYProfiles entries, the operator pastes a multi-line prompt and steps away
            without submitting; a turn opens on the paste and PAIR_WRAP_IDLE_S later they are told
            "no agent output for 60s" about a pane the agent was never asked anything. Do not add a
            paste tracker to passThroughChunk - that is a third divergence. One pure predicate, both
            stdin paths consult it. Prevalence: 2 divergences from the single predicate in 2 rounds.
          family: submits-rule-second-home
          round: 5
        - id: BR-22
          severity: Minor
          title: The expiry-drain test queues an observation kind that channel never carries in production
          detail: |-
            publishLifecycleObservation has exactly four call sites (wrap.go:1596, 1850, 1900, 1919)
            and they publish only ObservationBareReturn and ObservationUserSubmission; marker,
            native and transcript completions are reduced directly on the master goroutine.
            TestIdleExpiryDrainAppliesAQueuedCompletionFirst (idle_floor_test.go:205) queues an
            ObservationMarkerCompletion on p.lifecycleEvents, and applyIdleExpiry's doc comment
            (notification_lifecycle.go:285) says "a submission or completion already published but
            not yet reduced" - the completion half is unreachable. The drain itself stays pinned by
            the BR-2 test, which uses a reachable kind, so this is fixture realism and a false
            comment rather than a coverage hole.
          family: fixture-not-production-reachable
          round: 5
      blocked: false
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

## Round 3 — 2026-09-09T02:57:57-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — submits decided in decidePlainReturn and asserted on all eight exits; publish keyed on it, not on adapt.Bypass.
- BR-2 — addressed — Verified by revert — restoring the post-drain read reddens TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain.
- BR-3 — addressed — Verified by revert — deleting the drain reddens TestIdleExpiryDrainAppliesAQueuedCompletionFirst.
- BR-4 — not-addressed — Sentinel and invariant split landed but are unreachable in go test — see Important I-2; also the sentinel guard test is inverted.
- BR-5 — addressed — README "The idle floor" section documents the message, the once-per-turn bound and PAIR_WRAP_IDLE_S/=0.
- BR-6 — addressed — Verified by revert — removing the precedence block reddens TestIdleExpiryGivesTheLifecycleDeadlinePrecedenceWhenBothAreReady.
- BR-7 — addressed — Nil profile now flows through decidePlainReturn as the zero profile; overlay check no longer skipped.
- BR-8 — addressed — drainStop extracted; a new duplication appeared in the same diff, raised separately.
- BR-9 — not-addressed — Deliberately deferred and tracked as pair#219; pre-existing pattern, Minor, non-blocking.
- BR-10 — not-addressed — Message half is pinned; the behavioural half (idempotent arming) is pinned by nothing — an unconditional resetIdleTimer keeps the whole package green.
- BR-11 — addressed — spawnSlug seam injected and exercised by both idle harnesses; no wall-clock debounce left.
- BR-12 — addressed — Atlas now carries the alert-not-terminal exception and the snapshot-before-drain clause.

### Raised

- **BR-13** [Important] `unfalsifiable-race-test` Three of this round's deliverables can be deleted with the whole wrapcmd package still green, including the mode gate this issue exists to remove
  This is the 2nd finding in family `unfalsifiable-race-test`. Do not fix the three
  sites named here — write the enumeration. Rule: every `- [x]` row in the issue's
  `## Plan` is complete only when deleting its code turns a test red; sweep the
  checklist by deletion in this round. Measured against the full package (baseline
  11 environmental pty-child failures, identical in every mutation): re-adding
  `if p.notifyModeActive != "idle" { p.idleS = 0 }` near wrap.go:2386 stays green;
  deleting `if decision.submits { publish ObservationBareReturn }` at
  wrap.go:1807-1809 stays green; replacing the epoch check at
  notification_lifecycle.go:328-330 with an unconditional resetIdleTimer stays green.
  Cause: the idle tests build `proxy` directly with idleS/notifyModeActive set, so
  they never cross the arg-parse seam, and no test drives a plain Enter through
  emitPlainCR. Prevalence: 3 of the Plan's 8 rows, measured.
- **BR-14** [Important] `purpose-vs-covered-population` The floor never arms under PAIR_WRAP_REMAP_RETURN=0 or for agents outside harnessTTYProfiles, and a bare CR mid-turn cannot re-arm a spent floor
  This is the 2nd finding in family `purpose-vs-covered-population`. Do not fix one
  site — write the enumeration. Rule: enumerate the (agent x configuration x event)
  populations the floor claims, and confirm each has at least one live turn-opening
  publisher and a re-arm path; atlas/architecture.md:785 "It arms for every agent"
  and README:598-612 must be true of that enumeration or be qualified. Openers
  enumerated: keyboard submission (wrap.go:1858,1877) and bare CR (wrap.go:1808),
  both inside translateChunk and reachable only when hasReturnRemap() (wrap.go:1454);
  progress-OSC Working (claude only, progressOSCAuthorized); transcript-started
  (codex journal only); marker completion synthesising Working (claude only).
  So PAIR_WRAP_REMAP_RETURN=0 publishes no keyboard observation at all — README:112
  still documents that var as a keybinding opt-out only — and any agent outside
  {claude, codex, agy, muse} has zero openers, since agentBasename is just
  filepath.Base(argv[0]) (wrap.go:2334). Third instance, event-shaped:
  notification_lifecycle.go:206-210 makes ObservationBareReturn a no-op inside an
  open turn, so once the single alert has fired (IdleNotified=true, timer disarmed)
  the operator's menu answer leaves that turn with no floor, while a mid-turn
  Alt+Enter re-opens and re-arms. Prevalence: 5 openers, 2 configurations with zero,
  1 of 2 keyboard openers with no re-arm.
- **BR-15** [Minor] `duplicated-idiom` The lifecycle-expiry application is now written twice, and the outer-TTY sidecar test fixture is at seven sites
  This is the 2nd finding in family `duplicated-idiom`. Do not fix the instance —
  the rule is that a block appearing a second time in the SAME diff gets extracted,
  which is what produced drainStop for BR-8. Instances: the two-line
  `kind, token := p.lifecycleTimerKind, p.lifecycleTimerToken; processLifecycleObservation(...)`
  at wrap.go:2667-2669 and notification_lifecycle.go:296-299 should be one
  applyLifecycleExpiry(); the `dir/outer` + `dir/outer-path` fixture now appears at
  7 test sites, 2 added here (idle_floor_test.go:33-42 and :143-152), and wants a
  shared newLifecycleTestProxy helper. Prevalence: 2 sites and 7 sites, measured.
- **BR-16** [Minor] `message-overstates-measurement` applyIdleExpiry logs IDLE and traces the expiry before the reducer decides whether the alert applies
  notification_lifecycle.go:303-305 emits p.debug("IDLE", message) and
  traceWrap("idle", ...) unconditionally, so a token-rejected expiry — including one
  the drain or the lifecycle-precedence branch just invalidated — still records a
  completed idle expiry in the debug log and the trace, which is where a future
  operator will go to reconstruct why an alert did or did not fire.

## Round 4 — 2026-09-09T03:23:43-07:00 (claude) — BLOCKED

### Disposed

- BR-4 — addressed — Bound is now observationKindCount-1 (1..12, both new kinds reachable); seeds reach each; invariant split into completion-per-generation and alert-per-epoch; walk test covers every declared kind; 3.5M execs pass.
- BR-9 — not-addressed — Deliberately deferred to pair#219, which exists with a real spec; pre-existing pattern, Minor, non-blocking.
- BR-10 — addressed — Verified by mutation — unconditional resetIdleTimer reddens TestSyncIdleTimerDoesNotRestartAnArmedDeadlineWithinTheSameEpoch; sub-second rendering pinned separately.
- BR-13 — addressed — Sweep reproduced independently: 9 of 11 mutations redden a specific named test. One re-addition mutation still survives; carried forward as a Minor, since re-adding deleted code is not the deletion rule BR-13 stated.
- BR-14 — addressed — del-passthrough-publish, del-emitplaincr-publish and del-rearm all redden; I re-verified the population enumeration (pass-through is the exclusive else of hasReturnRemap, overlay unreachable there, ptmx has one writer).
- BR-15 — not-addressed — Both instances still present — notification_lifecycle.go:304 vs wrap.go:2694, and the outer/outer-path fixture at 7 test sites.
- BR-16 — not-addressed — notification_lifecycle.go:310-311 still log and trace the expiry unconditionally, before the reducer decides whether the alert applies.

### Raised

- **BR-17** [Important] `atlas-overstates-guarantee` README and atlas still promise "at most once per turn", which this round's bare-CR re-arm made false
  This is the 2nd finding in family `atlas-overstates-guarantee`. The rule, not the
  instance: when a state-machine invariant changes, every restatement of it must be
  swept in the same round — here the fuzz invariant was rewritten from per-generation
  to per-epoch (notification_lifecycle_test.go:196-198) while README.md:605 ("It fires
  at most once per turn") and atlas/architecture.md:783 ("a generation can carry one
  alert plus a later real completion") kept the superseded claim. atlas:785 also still
  says IdleToken is "minted once per turn" and then describes the re-arm that mints a
  second one. Enumerated restatements of the once-per invariant: 3 (fuzz, README,
  atlas x2 clauses); 1 swept, 3 stale.
- **BR-18** [Important] `unfalsifiable-race-test` The master-loop exactly-once oracle cannot fail — the 500ms emit limiter masks duplicates inside its 400ms window
  This is the 3rd finding in family `unfalsifiable-race-test`. Do NOT fix this instance
  by widening the sleep. Rule: a notification-count assertion is falsifiable only when
  the observation window exceeds rateLimitS (wrap.go:78, 500ms) or the limiter's clock
  is injected; emitOuter:652 calls time.Now() directly even though the proxy already
  carries an injectable `now func() time.Time`. Measured: with !state.IdleNotified
  dropped from both the reducer guard and syncIdleTimer, idle_floor_test.go:102 passes;
  re-run with settle(1500ms) it fails with "notifications = 3". Prevalence: 3 count
  assertions in idle_floor_test.go (:102, :207, :231) all inside the limiter window;
  :207 and :231 survive on their content checks, :102 has none.
- **BR-19** [Minor] `test-avoids-seam-not-injects-it` No test crosses the startup wiring seam, so the mode gate can be re-added at its original call site with the package green
  This is the 2nd finding in family `test-avoids-seam-not-injects-it`. Measured:
  inserting `if p.notifyModeActive != "idle" { p.idleS = 0 }` after wrap.go:2416 leaves
  the package GREEN (failure set identical to baseline). resolveNotifyConfig is
  asserted, but nothing tests the path that consumes it, and Run terminates in
  pty.Start so an end-to-end test is not available in this environment. Rule, not
  instance: p.idleS and p.notifyModeActive should have exactly one assignment site fed
  by resolveNotifyConfig, enforced by a source-scanning test — or the residue accepted
  explicitly in the issue Log.

## Round 5 — 2026-09-09T03:45:57-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — Deferred to pair#219, which states the class (32 states, ~4 meanings, the three disagreeing guards) faithfully; not re-raising on this issue.
- BR-15 — not-addressed — Re-measured at head: lifecycle-expiry application still at wrap.go:2705 + notification_lifecycle.go:304; outer-path fixture still at 7 test sites.
- BR-16 — not-addressed — notification_lifecycle.go:310-311 still logs IDLE and traces the expiry unconditionally, before the reducer decides.
- BR-17 — addressed — The three sites it named are swept and verified by grep; the un-enumerated source-comment class is raised separately below.
- BR-18 — addressed — Verified both ways: duplicate mutation fails with notifications = 9 under the injected clock, passes with now = time.Now.
- BR-19 — addressed — Verified by revert: re-inserting the mode gate at its original call site reddens the source-scanning test with "assigned at 2 sites".

### Raised

- **BR-20** [Important] `atlas-overstates-guarantee` Five in-source restatements of the superseded once-per-turn invariant survived the sweep
  This is the 3rd finding in this family. The rule, not the instance: the enumeration
  of an invariant's restatements must be DERIVED mechanically (grep the phrasing across
  the tree) rather than inherited from the prior finding's bullet list. BR-17's three
  named sites are correctly swept; its enumeration was incomplete. Measured at head:
  notification_lifecycle.go:37 ("inside an open turn it is a no-op" - false, it re-arms),
  :77 ("at most once per turn ... open() clears it"), :80 ("minted once per turn"),
  :320 ("has not already raised its one alert ... a new turn re-arms it"), :351
  ("minted once per turn by the reducer"). 3 swept, 5 stale. Record the grep in the Log
  so the next invariant change reuses the derivation rather than a hand-list.
- **BR-21** [Important] `submits-rule-second-home` passThroughChunk re-derives the submission predicate inline and has already diverged - a bracketed paste opens a turn and earns a spurious alert
  This is the 2nd finding in this family. BR-7 established that "does this input submit?"
  is decided once in decidePlainReturn and carried on returnDecision.submits. BR-14's fix
  planted a third home at wrap.go:1595 as a bare `bytes.IndexByte(data, '\r') >= 0`, with
  no bracketed-paste awareness - its caller hard-sets inPaste = false at wrap.go:1492,
  while translateChunk never reaches emitPlainCR inside a paste. Measured with a probe on
  both paths given the same bytes "ESC[200~line one CR line two CR ESC[201~":
  pass-through published [ObservationBareReturn], remap path published nothing.
  Failure scenario: under PAIR_WRAP_REMAP_RETURN=0, or for any agent outside the four
  harnessTTYProfiles entries, the operator pastes a multi-line prompt and steps away
  without submitting; a turn opens on the paste and PAIR_WRAP_IDLE_S later they are told
  "no agent output for 60s" about a pane the agent was never asked anything. Do not add a
  paste tracker to passThroughChunk - that is a third divergence. One pure predicate, both
  stdin paths consult it. Prevalence: 2 divergences from the single predicate in 2 rounds.
- **BR-22** [Minor] `fixture-not-production-reachable` The expiry-drain test queues an observation kind that channel never carries in production
  publishLifecycleObservation has exactly four call sites (wrap.go:1596, 1850, 1900, 1919)
  and they publish only ObservationBareReturn and ObservationUserSubmission; marker,
  native and transcript completions are reduced directly on the master goroutine.
  TestIdleExpiryDrainAppliesAQueuedCompletionFirst (idle_floor_test.go:205) queues an
  ObservationMarkerCompletion on p.lifecycleEvents, and applyIdleExpiry's doc comment
  (notification_lifecycle.go:285) says "a submission or completion already published but
  not yet reduced" - the completion half is unreachable. The drain itself stays pinned by
  the BR-2 test, which uses a reachable kind, so this is fixture realism and a false
  comment rather than a coverage hole.

## Open findings

- **BR-15** [Minor] `duplicated-idiom` The lifecycle-expiry application is now written twice, and the outer-TTY sidecar test fixture is at seven sites
- **BR-16** [Minor] `message-overstates-measurement` applyIdleExpiry logs IDLE and traces the expiry before the reducer decides whether the alert applies
- **BR-20** [Important] `atlas-overstates-guarantee` Five in-source restatements of the superseded once-per-turn invariant survived the sweep
- **BR-21** [Important] `submits-rule-second-home` passThroughChunk re-derives the submission predicate inline and has already diverged - a bracketed paste opens a turn and earns a spurious alert
- **BR-22** [Minor] `fixture-not-production-reachable` The expiry-drain test queues an observation kind that channel never carries in production
