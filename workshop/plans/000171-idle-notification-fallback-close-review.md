# Boundary Review — pair#171 (whole-issue close)

| field | value |
|-------|-------|
| issue | 171 — Always-on idle notification fallback |
| repo | pair |
| issue file | workshop/issues/000171-idle-notification-fallback.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9edc8a9772321eed8999dd5cacef28cdd4d0fd31..4b26298cb5089f2a1d23648b6567eec40fab6f70 |
| command | sdlc close --issue 171 |
| reviewer | claude |
| timestamp | 2026-09-09T02:36:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The floor itself is well-built: the reducer change is genuinely pure, the alert-not-completion decision is the right one and is pinned by real tests, the `submits` sweep across all eight `decidePlainReturn` exits is exactly the class-not-instance discipline the plan promised, and every Done-when bullet has a corresponding assertion. What blocks SHIP is one correctness bug in the new master-loop branch: the idle expiry's epoch token is read **after** the drain (`wrap.go:2715`), so an opener drained at expiry time re-mints both `IdleToken` and `p.idleTimerToken` in lockstep and the identity check can no longer reject the stale expiry — I reproduced it deterministically (false `no agent output for 60s` emitted against a turn opened microseconds earlier, and that turn's floor then spent, `IdleNotified=true`, timer disarmed). This is the exact race class the plan named as the adversarial one, and the test written for it (`TestIdleFloorExpiryYieldsToABoundaryQueuedBeforeIt`) never enters the branch at all: instrumented, the idle case fires 0 times, and deleting the entire drain block leaves every test in the package green.

## 1. Strengths

- `notification_lifecycle.go:170-190` — the alert-not-completion decision, with the reasoning inline and both halves pinned (`TestIdleFloorIsAnAlertSoARealCompletionStillNotifies` checks the real completion still notifies *after* an alert). This was the plan-gate's Critical and it was answered in the design, not papered over.
- `harness_tty.go:88-95` + `harness_tty_test.go:105-140` — `submits` is decided inside the pure function and asserted on all eight rows, not just the one that motivated it. The comment explaining why it is *not* `outcome == adapt.Bypass` (overlay confirm sends nothing to the agent) is the finding-class sweep done right.
- Measurement-before-design, and the Log records that the measurement *refuted* the Spec's premise rather than being fitted to it. The `## Revisions` entry supersedes the criterion honestly.
- `syncIdleTimer` (`notification_lifecycle.go:262`) placed beside `syncLifecycleTimer` reading the same state, with the `idleFired` master-loop latch deleted — arming genuinely cannot drift from lifecycle state now. `ARCH-PURE` and `ARCH-DRY` both pass on the mechanism.
- `go vet` clean; `-race -count=2` clean on the lifecycle/idle tests. The four `HarnessTTYCapture*` failures are environmental (pty-child spawn denied) and reproduce identically at BASE.

## 2. Critical findings

**`wrap.go:2715` — the expiry token is snapshotted after the drain, so the epoch check is vacuous.**

`resetIdleTimer` sets `p.idleTimerToken = p.notificationLifecycle.IdleToken`, and the drain at 2701-2708 runs `processLifecycleObservation` → `syncIdleTimer` → `resetIdleTimer`. So if the drained observation is an *opener* (`ObservationUserSubmission`, `ObservationBareReturn`, `ObservationWorking`, `ObservationTranscriptStarted`), both the reducer's `IdleToken` and the loop's `idleTimerToken` advance to the new turn — and then line 2715 reads the *new* token, which necessarily matches. Reproduced with a verbatim replica of the branch (single-goroutine, no timing):

```
gen1=1 gen2=2 alerted=true IdleNotified=true
emitted=["\x1b]777;notify;pair;no agent output for 60s\a"]
```

Two wrong outcomes at once: a false alert one moment after the user submits, and the fresh turn's floor is spent (`IdleNotified=true` → `syncIdleTimer` disarms), so the turn that most needs the floor has none. The completion case survives only because `complete()` zeroes `IdleToken` and the `Token != 0` guard catches it — i.e. the token check as written cannot detect any mismatch it was added to detect.

Fix: snapshot before the drain.

```go
case <-idleTimer.C:
    token := p.idleTimerToken   // the epoch this deadline belongs to
    for { /* drain */ }
idleDrained:
    ...
    p.processLifecycleObservation(TurnObservation{Kind: ObservationIdleExpired, Token: token, ...})
```

With that, the drained-opener case mismatches and is correctly ignored, and the re-armed timer gives the new turn its full interval.

## 3. Important findings

**`idle_floor_test.go:134` — the injected-arrival-order test never reaches the code it names.** Instrumented with a counter in `case <-idleTimer.C`, the branch is entered **0 times** in that test: the top-level `case observation := <-p.lifecycleEvents` (`wrap.go:2661`) consumes the completion ~40 ms before the deadline, `syncIdleTimer` disarms the timer, and it never fires. Deleting the whole drain block (2701-2708) keeps `go test -run Idle -count=5` green. So PQ-2's drain deliverable is unpinned, and the test that reads as the race oracle is a sample of size zero. `ARCH-ORDER` at-review. Fix: make the interleaving injectable — e.g. a test-only hook invoked at the top of the idle branch that publishes the queued observation, or extract the branch into `func (p *proxy) applyIdleExpiry(token uint64)` and unit-test it with the channel pre-loaded (that also gives the Critical above a natural failing test: pre-load a submission, assert no notify and that the new turn's `IdleNotified` is false).

**`notification_lifecycle_test.go:161` — the property test excludes both new observation kinds, and its invariant is now false for one of them.** `ObservationKind(raw%byte(ObservationGraceExpired) + 1)` yields 1..10; `ObservationIdleExpired` (11) and `ObservationBareReturn` (12) are unreachable. The bound is a hand-maintained restatement of the enum, so it silently stops covering every kind added after it (`ARCH-PURPOSE`). It also can't simply be widened: the idle alert deliberately breaks "at most once per generation" (alert + later completion), so the fuzz's invariant needs restating as at most one *completion* notify plus at most one idle alert per generation. Fix: add an `observationKindCount` sentinel at the end of the const block, derive the bound from it, feed `state.IdleToken` for `ObservationIdleExpired` as the switch already does for the other tokenized kinds, and split the invariant.

**README.md — no update for the now-always-on idle notification.** The diff turns a knob that was inert in every shipped configuration into an always-armed notification for every agent, with `PAIR_WRAP_IDLE_S` (`0` disables) as the only opt-out. README has a `## Notifications` section (line ~588) describing what Pair forwards, and documents the directly comparable `PAIR_WRAP_REMAP_RETURN=0` opt-out in the keybindings table — a user who starts getting "no agent output for 60s" has nowhere in the docs to learn what it is or how to turn it off. Docs gate: add two sentences to `## Notifications`.

## 4. Minor findings

- `wrap.go:2696` vs `2657` — the idle deadline and the watchdog deadline are both 60 s and can be co-ready in the same `select`; Go picks randomly and the drain doesn't cover a *timer* case. The 0.5 s limiter in `emitOuter` suppresses the duplicate, so the visible effect is the operator getting the less-informative message ("no agent output for 60s" instead of "agent stopped working"). Cheap fix: non-blocking `select` on `lifecycleTimer.C` inside the idle branch before applying the expiry.
- `wrap.go:1776-1779` — the `p.ttyProfile == nil` branch of `emitPlainCR` publishes `ObservationBareReturn` unconditionally. That branch is dead in production (`hasReturnRemap()` gates every call site) but it is now a *second* home for the `submits` rule that `decidePlainReturn` owns, and unlike the four live exits it never consults `pickerActive` — if it ever becomes reachable it does exactly what `submits: false` was introduced to prevent. Either drop the publish or route it through the pure decision.
- `notification_lifecycle.go:280-306` vs `:316` — the Stop+drain idiom is now written three times. `func drainStop(t *time.Timer)` (`ARCH-DRY`).
- `notification_lifecycle.go:170` — `no agent output for %.0fs` reports the *configured* interval; the deadline is also reset by non-output observations (journal records reduce → `syncIdleTimer` → reset), so the message can overstate actual byte silence. Also renders as `for 0s` at sub-second intervals (test-only today).
- `atlas/architecture.md:783` — the retained sentence "notify at most once per generation" is now inaccurate unqualified (the new paragraph below it says the opposite for the alert case); and ":785" claims "the expiry branch drains queued boundaries first so a just-reported turn is never alerted against", which the Critical above shows is not true for openers. Both need a clause once the fix lands.
- `idle_floor_test.go:51` — `lastSlug: time.Now()` suppresses the real `pair slug` subprocess via a 1 s wall-clock debounce. This is the established repo pattern, but it is the first test where the emit happens after a live timer in a goroutine, so on a loaded machine the margin (~960 ms) can lapse and the test spawns a real slug run against the operator's session. An injected `spawnSlug func()` seam would make it deterministic (`ARCH-MOCK`).

## 5. Test coverage notes

The pure rows (a)-(f) from the plan are all present and each pins a distinct property; `TestIdleFloorArmsForEveryNotifyModeNotJustOneNothingSelects` genuinely exercises the whole master loop in both modes and asserts *exactly one* notification across ten intervals, which is the Done-when bullet. What is missing is coverage of the two orderings that actually matter: the drained-opener case (the Critical, currently zero coverage and failing) and the drained-completion case (claimed but never executed). Both become cheap once the branch body is extracted from the `select`.

## 6. Architectural notes

`ARCH-DRY` pass with two Minors above. `ARCH-PURE` pass — the reducer and `decidePlainReturn` are both pure and tested with no IO; the extracted decision fields are the right shape. `ARCH-PURPOSE` flag on the fuzz's hand-maintained kind bound. `ARCH-MOCK` N/A but see the slug-spawn Minor. `ARCH-CONSTRAINTS` pass — the per-chunk cost is one `Stop`/`Reset`, bounded and O(1), and the disabled path early-returns before touching the timer at all. `ARCH-SECURE` N/A — no new untrusted input or credential handling. `ARCH-ORDER` flag, and it is where both top findings live; additionally, `NotificationLifecycle` now carries five booleans (`Active`, `Completed`, `ActivitySeen`, `GracePending`, `IdleNotified`) plus three tokens whose legal combinations are unwritten — `Completed` is provably redundant with `!Active` except to distinguish "never opened", which is why the guards are inconsistent across cases (`Active && !Completed` at :146, `!Active || Completed` at :193, bare `Active` at :140). For the next change here, collapse to `turnState {none, open, closed}` with the open-substate flags hanging off `open`; the current shape declares 32 states and means about four.

## 7. Plan revision recommendations

- `## Revisions` entry: the "Tests (disposes PQ-4)" checkbox claims the master-loop adversarial row injects "idle expiry racing a queued submission and a completion". Measured, the delivered test injects neither — the branch is not entered, and the submission half was never written. Record what was actually delivered and what the fix adds.
- `## Revisions` entry: the "Timer arming tied to turn state (disposes PQ-2)" step states "the token identifies the turn's idle epoch" as the safety property. As implemented the token is read after the drain re-mints it, so the property does not hold; note the before-drain snapshot as the corrected mechanism.

```findings
findings:
  - id: new
    severity: Critical
    family: expiry-token-snapshot-after-mutation
    title: |
      Idle expiry reads its epoch token after the drain, so an opener drained at expiry time gets a false alert and loses its floor
    detail: |
      wrap.go:2715 reads p.idleTimerToken after the 2701-2708 drain, but the drain runs
      processLifecycleObservation -> syncIdleTimer -> resetIdleTimer, which advances
      p.idleTimerToken in lockstep with the reducer's new IdleToken. A submission or
      bare return drained there therefore matches the stale expiry: reproduced
      deterministically as gen1=1 gen2=2 alerted=true, emitting
      "no agent output for 60s" against a turn opened microseconds earlier and setting
      IdleNotified=true so that turn is left with no floor at all. Snapshot
      token := p.idleTimerToken before the drain loop. ARCH-ORDER.
  - id: new
    severity: Important
    family: unfalsifiable-race-test
    title: |
      The test named for the injected expiry/boundary ordering never enters the idle branch; deleting the whole drain keeps every test green
    detail: |
      Instrumented, TestIdleFloorExpiryYieldsToABoundaryQueuedBeforeIt
      (idle_floor_test.go:134) enters case <-idleTimer.C zero times - the top-level
      lifecycleEvents case consumes the completion 40ms early and syncIdleTimer
      disarms the timer. Removing wrap.go:2701-2708 entirely leaves go test -run Idle
      -count=5 passing, so PQ-2's drain deliverable is pinned by nothing. Extract the
      branch body (e.g. applyIdleExpiry(token uint64)) so the interleaving can be
      injected; that also gives the Critical above a natural failing test. ARCH-ORDER.
  - id: new
    severity: Important
    family: hand-maintained-enumeration
    title: |
      The lifecycle fuzz excludes both new observation kinds, and the invariant it asserts is now false for the idle alert
    detail: |
      notification_lifecycle_test.go:161 derives kinds from
      raw%byte(ObservationGraceExpired)+1, yielding 1..10, so ObservationIdleExpired
      and ObservationBareReturn are unreachable - a hand-maintained restatement of the
      enum that silently stops covering every kind added after it. Widening it also
      requires restating the property: the idle alert deliberately breaks "notify at
      most once per generation" (alert plus a later real completion). Add an
      observationKindCount sentinel, feed state.IdleToken for the new tokenized kind,
      and split the invariant into one completion plus at most one alert per
      generation. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: docs-gate-user-surface
    title: |
      README update appears missing for the now-always-on idle notification and PAIR_WRAP_IDLE_S
    detail: |
      The diff converts an inert knob into an always-armed notification for every
      agent. README's "## Notifications" section describes what Pair forwards and the
      keybindings table documents the comparable PAIR_WRAP_REMAP_RETURN=0 opt-out, but
      neither mentions the idle floor or that PAIR_WRAP_IDLE_S=0 disables it, so a user
      who starts receiving "no agent output for 60s" has no documented way to
      understand or silence it.
  - id: new
    severity: Minor
    family: expiry-ignores-coready-boundary
    title: |
      Idle and watchdog deadlines can be co-ready with no precedence rule, so the operator may get the less-informative message
    detail: |
      Both are 60s and live in separate select cases, so Go picks randomly when they
      expire together; the drain covers channel events, not the lifecycle timer. The
      0.5s emitOuter limiter suppresses the duplicate, so the effect is "no agent
      output for 60s" displacing "agent stopped working". A non-blocking select on
      lifecycleTimer.C before applying the expiry gives the completion precedence.
  - id: new
    severity: Minor
    family: submits-rule-second-home
    title: |
      emitPlainCR's nil-profile branch publishes ObservationBareReturn outside the pure decision and without the overlay check
    detail: |
      wrap.go:1776-1779 hardcodes the submits predicate that decidePlainReturn now
      owns for its four exits, and returns before the pickerActive swap - so if that
      defensive branch ever became reachable it would open a turn on a picker confirm,
      exactly the case submits:false was added to prevent. Dead in production today
      (hasReturnRemap gates every call site). ARCH-DRY.
  - id: new
    severity: Minor
    family: duplicated-idiom
    title: |
      The timer Stop+drain idiom is now written three times
    detail: |
      notification_lifecycle.go:284-291, :299-305 and :318-324 are the same block;
      extract func drainStop(t *time.Timer). ARCH-DRY.
  - id: new
    severity: Minor
    family: boolean-constellation-needs-tagged-enum
    title: |
      NotificationLifecycle now carries five booleans whose legal combinations are unwritten
    detail: |
      Active, Completed, ActivitySeen, GracePending and IdleNotified declare 32 states
      and mean about four; Completed is provably redundant with !Active except to
      distinguish "never opened", which is why the guards disagree across cases
      (Active && !Completed at :146, !Active || Completed at :193, bare Active at
      :140). Collapse to turnState{none, open, closed} with the open-substate flags
      hanging off open. Pre-existing, extended by this diff. ARCH-ORDER.
  - id: new
    severity: Minor
    family: message-overstates-measurement
    title: |
      "no agent output for Ns" reports the configured interval, and non-output observations also reset the deadline
    detail: |
      wrap.go:2710 formats p.idleS, and syncIdleTimer resets the deadline on every
      reduced observation - including journal records that arrive with no pane output -
      so the reported silence can be longer than the actual byte-quiet window. Also
      renders as "for 0s" at sub-second intervals.
  - id: new
    severity: Minor
    family: test-avoids-seam-not-injects-it
    title: |
      The idle harness suppresses the real pair slug subprocess with a 1s wall-clock debounce rather than an injected seam
    detail: |
      idle_floor_test.go:51 sets lastSlug: time.Now(); this is the established repo
      pattern, but it is the first test whose emit happens after a live timer in a
      goroutine, so on a loaded machine the ~960ms margin can lapse and the test spawns
      a real slug run (model call plus session-state write) against the operator's
      machine. An injected spawnSlug func() would make it deterministic. ARCH-MOCK.
  - id: new
    severity: Minor
    family: atlas-overstates-guarantee
    title: |
      Atlas retains the old at-most-once invariant and claims a drain guarantee the code does not provide
    detail: |
      atlas/architecture.md:783 still says terminals "notify at most once per
      generation" unqualified, which the alert-plus-completion path contradicts; :785
      states the expiry branch drains "so a just-reported turn is never alerted
      against", which the Critical finding shows is false for drained openers. Both
      need a clause once the token fix lands.
```

---

## Re-review — 2026-09-09T02:57:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 171 — Always-on idle notification fallback |
| repo | pair |
| issue file | workshop/issues/000171-idle-notification-fallback.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9edc8a9772321eed8999dd5cacef28cdd4d0fd31..d483b5606f70ed83149b2364e26b3e3de940ee45 |
| command | sdlc close --issue 171 |
| reviewer | claude |
| timestamp | 2026-09-09T02:57:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 1's Critical is genuinely fixed and I verified it the hard way: restoring the post-drain token read makes `TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain` emit exactly the false `no agent output for 60s`; deleting the drain reddens `TestIdleExpiryDrainAppliesAQueuedCompletionFirst`; removing the co-ready precedence block reddens the BR-6 row; removing `bumpIdleDeadline` reddens the output-suppression row. `applyIdleExpiry(token)` is the right extraction — the ordering is now injected, not raced, and `-race -count=3` is clean. What stops SHIP is not correctness but pinning and coverage: three deliverables in this round can be deleted outright with the entire `wrapcmd` package still green — including the `notifyModeActive != "idle"` gate deletion, which is *the* bug this issue exists to fix — and the fuzz seed added for BR-4 never actually reaches `ObservationIdleExpired`, so the widened enumeration and the split invariant it required are inert in `go test`. Separately, the floor's covered population is narrower than README and atlas claim: with `PAIR_WRAP_REMAP_RETURN=0`, or for any agent outside the four `harnessTTYProfiles` entries, no turn-opening observation is ever published, so the "always-armed" floor never arms.

## 1. Strengths

- `notification_lifecycle.go:270-283` — `applyIdleExpiry(token uint64)` makes the epoch a parameter so the call site's evaluation order pins it structurally, not by comment. This is the fix BR-2 asked for and the extraction BR-3 asked for, done as one move.
- `notification_lifecycle.go:296-302` — giving the lifecycle deadline precedence *inside* the expiry, rather than relying on the 0.5s emit limiter to arbitrate, means the completed turn also swallows the idle alert by token. Better than the finding asked for.
- `notification_lifecycle.go:316-333` + `336-342` — splitting "arm on a new epoch" (`syncIdleTimer`, idempotent) from "push the deadline out on output" (`bumpIdleDeadline`) is exactly the right decomposition for BR-10, and keeps the alert's claim (byte-silence) true.
- `harness_tty.go:88-95` + `harness_tty_test.go:105-140` — `submits` decided inside the pure function and asserted on all eight exits, with the "not the same predicate as `adapt.Bypass`" reasoning inline. Class, not instance.
- `workshop/lessons.md:4089-4116` — both rules extracted from round 1 are written generally enough to fire on the next instance.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — Three of this round's deliverables are pinned by no failing test (repeat of BR-3's rule).**
Measured by deletion against the full `wrapcmd` package (baseline: 11 pre-existing environmental failures, all pty-child `operation not permitted`; identical set in every mutation):

| deliverable | mutation | result |
|---|---|---|
| mode-gate deletion (`wrap.go:2381-2386`) | re-add `if p.notifyModeActive != "idle" { p.idleS = 0 }` | **green** |
| bare-CR publish site (`wrap.go:1807-1809`) | delete the `if decision.submits { … }` block | **green** |
| idempotent arming (`notification_lifecycle.go:328-330`) | replace with unconditional `p.resetIdleTimer()` | **green** |

The idle tests construct `proxy` directly with `idleS`/`notifyModeActive` set, so they never traverse the arg-parse seam where the gate lived; and no test drives a plain Enter through `emitPlainCR` to observe an `ObservationBareReturn`. The rule (BR-3's, generalised): **the enumeration is the `## Plan` checklist, and each row is complete only when deleting its code turns a test red.** Sweep it by deletion this round rather than fixing the three sites named here.

**I-2 — BR-4's fix is unreachable in deterministic runs: the fuzz corpus never produces `ObservationIdleExpired`.**
`notification_lifecycle_test.go:169` maps `raw%12 + 1`, so `ObservationIdleExpired` (11) needs `raw ≡ 10 (mod 12)`. The three seeds — `{0,2,3,4,5}`, `{6,7,8,9,1,4}`, `{1,11,11,12,5}` — yield kinds `{1..10, 12}`; the new seed at :157 was written as if the bytes were kind values and lands on `BareReturn`/`Working`, not `IdleExpired`. Instrumented with a `t.Fatal` on `kind == ObservationIdleExpired`, `go test -run Fuzz` passes. Consequently the alert-plus-completion invariant split is never exercised outside `-fuzz`, and reverting the split to a single `notified` map also passes. Related: `TestObservationKindCountIsOnePastTheLastKind` is inverted — appending a kind *before* the sentinel (correct) fails it, appending *after* the sentinel (the failure mode it names) passes it. See dispose `BR-4: not-addressed`.

**I-3 — The floor does not arm for populations the Spec/atlas claim it covers (repeat of BR-1's family).**
Turn-opening publishers, enumerated: keyboard submission (`wrap.go:1858,1877`), bare CR (`wrap.go:1808`), progress-OSC `Working` (gated `progressOSCAuthorized` → claude only), transcript-started (codex journal only), marker-completion synthesising `Working` (claude only). The first two live in `translateChunk`, reachable only when `hasReturnRemap()` (`wrap.go:1454`, `wrap.go:1502`). Therefore:
- `PAIR_WRAP_REMAP_RETURN=0` → `ttyProfile == nil` → `passThroughChunk` → **no keyboard observation of any kind**. README:112 documents this env var purely as a keybinding opt-out; it now silently disables the notification floor too.
- Any agent outside `harnessTTYProfiles` = {claude, codex, agy, muse} (`agentBasename` is just `filepath.Base(argv[0])`, `wrap.go:2334`) → no profile, not claude, not codex → **zero openers, floor never arms**.
- Third instance, event-shaped: `notification_lifecycle.go:206-210` makes `ObservationBareReturn` a no-op inside an open turn, so after the one alert fires (`IdleNotified=true`, timer disarmed) the operator's menu answer leaves that turn with **no floor at all** — while an Alt+Enter `ObservationUserSubmission` mid-turn re-opens and re-arms. The bare-CR path is precisely the population with no other timer.

Prevalence: 5 openers enumerated, 2 configurations with zero, 1 of 2 keyboard openers with no re-arm path. Rule: enumerate (agent × config × event) populations the floor claims, confirm each has a live opener and a re-arm, and qualify `atlas/architecture.md:785` ("It arms for every agent") and README:598-612 to match what the enumeration actually delivers.

## 4. Minor findings

- **M-1 (family `duplicated-idiom`, 2nd).** The lifecycle-expiry application is now written twice verbatim — `wrap.go:2667-2669` and `notification_lifecycle.go:296-299`; extract `applyLifecycleExpiry()`. Same rule as BR-8: when a block appears a second time *in the same diff*, extract it. Also measured: the outer-TTY sidecar test fixture (`dir/outer` + `dir/outer-path`) is now at 7 sites, 2 added here (`idle_floor_test.go:33-42`, `:143-152`); a `newLifecycleTestProxy` helper would close both.
- **M-2.** `TestIdleFloorStaysSilentWhileTheAgentIsProducingOutput` writes every 15ms against a 60ms floor — a 4× margin that a loaded machine can eat, producing a flaky false alert. 5-10× would be safer.
- **M-3.** `applyIdleExpiry` emits `p.debug("IDLE", …)` and `traceWrap("idle", …)` unconditionally, before the reducer decides whether the alert applies; the debug log will claim an idle expiry that was silently rejected by token.
- **M-4.** `wrap.go:626` still reads "marker/idle/native all land here" — a stale reference to the deleted mode value.

## 5. Test coverage notes

- Reducer rows (a)-(f) from the Plan all exist and all pin real state transitions, not mocks. `TestIdleFloorIsAnAlertSoARealCompletionStillNotifies` is the one that matters most and it is correct.
- The master-loop rows genuinely enter `masterPump` over an `os.Pipe` ptmx and observe at `writeTTY` — good seam choice, and `spawnSlug` injection (BR-11) is exercised on every idle test, so no real `pair slug` can spawn.
- Gaps, all in §3: the arg-parse seam, the `emitPlainCR` publish seam, `syncIdleTimer` idempotency, and `ObservationIdleExpired` under the deterministic fuzz corpus.
- No test asserts the `spawnSlug` seam is actually taken (removing `spawnSlug: func(){}` from a harness would not redden anything, it would just spawn a model call). Cheap to add alongside I-1's sweep.

## 6. Architectural notes

- **ARCH-DRY** — flag, M-1 (lifecycle-expiry block duplicated; sidecar fixture at 7 sites). `drainStop` itself is a clean pass.
- **ARCH-PURE** — pass. `Reduce` and `idleAlertMessage` are pure and tested without IO; `syncIdleTimer`/`bumpIdleDeadline`/`applyIdleExpiry` are thin glue over them.
- **ARCH-PURPOSE** — flag, I-3 and I-2. The single-source shadow-sweep on `ObservationKind` finds one consumer (the fuzz bound) that now derives from the sentinel — good — but the *inputs* to that consumer are still hand-picked bytes, so the derivation buys nothing deterministically.
- **ARCH-MOCK** — pass. `spawnSlug` is the seam the round-1 finding asked for, and production and test flow share it.
- **ARCH-CONSTRAINTS** — pass. `bumpIdleDeadline` is O(1) per chunk with an early return when `idleS <= 0`, matching the budget the Plan asserted; `envDuration` yields a negative for a negative `PAIR_WRAP_IDLE_S`, which the `<= 0` guard treats as disabled.
- **ARCH-SECURE** — N/A for secrets. The one untrusted input is `PAIR_WRAP_IDLE_S`, parsed at the boundary with a documented fallback and pinned by `TestIdleIntervalKnobParsesAnExplicitZeroAsDisabled`.
- **ARCH-ORDER** — mostly pass, one flag. The expiry/boundary interleaving is now injected through `applyIdleExpiry` rather than sampled, which is the strongest thing in this diff. The flag is I-3's third instance: `(open+alerted, BareReturn)` is an unwritten cell that silently drops the floor. BR-9's five-boolean constellation is unchanged and now deferred to pair#219.

## 7. Plan revision recommendations

- `## Revisions` entry — **"floor coverage is bounded by turn-opening publishers."** Record the enumeration from I-3 and state which populations the floor covers: `PAIR_WRAP_REMAP_RETURN=0` and agents outside `harnessTTYProfiles` are not covered. Either widen (publish the submission observation from `passThroughChunk` too) or declare the limit and qualify `atlas/architecture.md:785` and README:598-612, the way the 2026-09-09 revision handled the never-completes limitation.
- `## Revisions` entry — **"every Plan row is verified by deletion."** The Plan's `- [x]` rows for the mode gate, the bare-CR publish and the arming rule are ticked on code that no test protects. Add the deletion-sweep as the row's own acceptance condition so the next boundary can check it mechanically.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      submits decided in decidePlainReturn and asserted on all eight exits; publish keyed on it, not on adapt.Bypass.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by revert — restoring the post-drain read reddens TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain.
  - id: BR-3
    disposition: addressed
    note: |
      Verified by revert — deleting the drain reddens TestIdleExpiryDrainAppliesAQueuedCompletionFirst.
  - id: BR-4
    disposition: not-addressed
    note: |
      Sentinel and invariant split landed but are unreachable in go test — see Important I-2; also the sentinel guard test is inverted.
  - id: BR-5
    disposition: addressed
    note: |
      README "The idle floor" section documents the message, the once-per-turn bound and PAIR_WRAP_IDLE_S/=0.
  - id: BR-6
    disposition: addressed
    note: |
      Verified by revert — removing the precedence block reddens TestIdleExpiryGivesTheLifecycleDeadlinePrecedenceWhenBothAreReady.
  - id: BR-7
    disposition: addressed
    note: |
      Nil profile now flows through decidePlainReturn as the zero profile; overlay check no longer skipped.
  - id: BR-8
    disposition: addressed
    note: |
      drainStop extracted; a new duplication appeared in the same diff, raised separately.
  - id: BR-9
    disposition: not-addressed
    note: |
      Deliberately deferred and tracked as pair#219; pre-existing pattern, Minor, non-blocking.
  - id: BR-10
    disposition: not-addressed
    note: |
      Message half is pinned; the behavioural half (idempotent arming) is pinned by nothing — an unconditional resetIdleTimer keeps the whole package green.
  - id: BR-11
    disposition: addressed
    note: |
      spawnSlug seam injected and exercised by both idle harnesses; no wall-clock debounce left.
  - id: BR-12
    disposition: addressed
    note: |
      Atlas now carries the alert-not-terminal exception and the snapshot-before-drain clause.
findings:
  - id: new
    severity: Important
    family: unfalsifiable-race-test
    title: |
      Three of this round's deliverables can be deleted with the whole wrapcmd package still green, including the mode gate this issue exists to remove
    detail: |
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
  - id: new
    severity: Important
    family: purpose-vs-covered-population
    title: |
      The floor never arms under PAIR_WRAP_REMAP_RETURN=0 or for agents outside harnessTTYProfiles, and a bare CR mid-turn cannot re-arm a spent floor
    detail: |
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
  - id: new
    severity: Minor
    family: duplicated-idiom
    title: |
      The lifecycle-expiry application is now written twice, and the outer-TTY sidecar test fixture is at seven sites
    detail: |
      This is the 2nd finding in family `duplicated-idiom`. Do not fix the instance —
      the rule is that a block appearing a second time in the SAME diff gets extracted,
      which is what produced drainStop for BR-8. Instances: the two-line
      `kind, token := p.lifecycleTimerKind, p.lifecycleTimerToken; processLifecycleObservation(...)`
      at wrap.go:2667-2669 and notification_lifecycle.go:296-299 should be one
      applyLifecycleExpiry(); the `dir/outer` + `dir/outer-path` fixture now appears at
      7 test sites, 2 added here (idle_floor_test.go:33-42 and :143-152), and wants a
      shared newLifecycleTestProxy helper. Prevalence: 2 sites and 7 sites, measured.
  - id: new
    severity: Minor
    family: message-overstates-measurement
    title: |
      applyIdleExpiry logs IDLE and traces the expiry before the reducer decides whether the alert applies
    detail: |
      notification_lifecycle.go:303-305 emits p.debug("IDLE", message) and
      traceWrap("idle", ...) unconditionally, so a token-rejected expiry — including one
      the drain or the lifecycle-precedence branch just invalidated — still records a
      completed idle expiry in the debug log and the trace, which is where a future
      operator will go to reconstruct why an alert did or did not fire.
```

---

## Re-review — 2026-09-09T03:23:43-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 171 — Always-on idle notification fallback |
| repo | pair |
| issue file | workshop/issues/000171-idle-notification-fallback.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9edc8a9772321eed8999dd5cacef28cdd4d0fd31..384c2fc337d27d5e57f3e18978d72b102b54db51 |
| command | sdlc close --issue 171 |
| reviewer | claude |
| timestamp | 2026-09-09T03:23:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The floor is correct and, this round, genuinely pinned. I independently reproduced 9 of the 11 mutations in the Log's sweep table — every one reddened a specific named test (`epoch-after-drain`, `del-drain`, `del-precedence`, `del-rearm`, `del-passthrough-publish`, `del-emitplaincr-publish`, `idle-completes`, `unconditional-arm`, `drop-once-per-turn`), so BR-2/3/6/13/14's deliverables are falsifiable rather than plausible. `go vet` clean, `-race -count=2` clean on the lifecycle/idle set, fuzz 3.5M execs pass, and the 11 residual failures are the known pty-child/sandbox environmental set, identical at BASE. Nothing here blocks: no correctness bug survived. What remains is a docs-accuracy regression the BR-14 re-arm introduced in this same round (README and atlas still promise "at most once per turn" / "one alert per generation", which the code and the implementor's own restated fuzz invariant say is false), and one test oracle I measured as unfalsifiable — the master-loop "exactly 1 notification" row stays GREEN under a floor mutated to alert ten times, because the 500 ms `emitOuter` limiter masks duplicates inside its 400 ms settle window.

## 1. Strengths

- `notification_lifecycle.go:286-315` — `applyIdleExpiry(token uint64)` takes the epoch as a **parameter**, so the call site's evaluation order pins the snapshot structurally rather than by comment. Reverting to `Token: p.idleTimerToken` reddens `TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain` (verified). This is the right shape for the ARCH-ORDER fix.
- `wrap.go:104-123` — `resolveNotifyConfig` is a real pure extraction, and `TestResolveNotifyConfigNeverZeroesTheIdleInterval` asserts the *absence* of the gate across four agents. Turning an absence into an assertable property is the hard half of BR-13.
- The BR-14 population enumeration is genuinely complete: `passThroughChunk` (`wrap.go:1579-1586`) is the exclusive `else` of `hasReturnRemap()`, `pickerActive` can only be set when `ttyProfile != nil` (`wrap.go:1694`, `wrap.go:1191`), and `translateStdinFrom` is the only writer to `p.ptmx` — so every byte the agent receives passes one of the two publishers, with no overlay hazard on the pass-through side. I checked all four claims.
- `TestSyncIdleTimerDoesNotRestartAnArmedDeadlineWithinTheSameEpoch` picks a real observable (receiving the pending expiry after widening the interval) instead of `len(C)`/`Stop()`, both of which are blind here. The comment explaining why is accurate for Go 1.23 timers.
- `lessons.md` — the "a mutation that fails to apply is indistinguishable from one that survives" rule is correct and I hit exactly that trap myself (`cp` failed, `&&` short-circuited the mutation, output read as a clean survivor). Well earned.

## 2. Critical findings

None.

## 3. Important findings

**README.md:605 and atlas/architecture.md:783 — both still promise a once-per-turn guarantee the BR-14 re-arm removed in this same commit.**
`ObservationBareReturn` mid-turn now clears `IdleNotified` and mints a fresh `IdleToken` (`notification_lifecycle.go:206-210`), so one generation can raise N alerts — which is precisely why the fuzz invariant was rewritten from per-generation to per-epoch (`notification_lifecycle_test.go:196-198`). README says "It fires at most once per turn"; atlas says "a generation can carry one alert plus a later real completion" and "`IdleToken` … is minted once per turn" (`:785`), the latter contradicted three sentences later by its own re-arm paragraph. The model changed, one consumer (the fuzz) was updated, two were not — ARCH-PURPOSE shadow-sweep. Fix: "at most once per attention window — answering a prompt starts a new one"; atlas `:783` → "one alert per idle epoch"; `:785` → "minted once per turn and re-minted when a bare CR re-arms a spent floor."

**idle_floor_test.go:102 — the exactly-once oracle cannot fail.** Measured: with `!state.IdleNotified` dropped from both the reducer guard and `syncIdleTimer`, `TestIdleFloorArmsForEveryNotifyModeNotJustOneNothingSelects` PASSES, because `settle(400ms)` < `rateLimitS` (500 ms, `wrap.go:78`) so `emitOuter` suppresses every duplicate. Widening to `settle(1500ms)` under the same mutation fails with `notifications = 3`. This is the 3rd finding in family `unfalsifiable-race-test`. Earlier rounds fixed instances; do NOT just widen this one. The rule: **an assertion on emitted-notification count is only falsifiable when the observation window exceeds `rateLimitS`, or when the limiter's clock is injected** — the proxy already carries a `now func() time.Time` field that `emitOuter:652` bypasses in favour of `time.Now()`. Route the limiter through `p.now` and the whole class becomes deterministic. Prevalence, measured: 3 count assertions in `idle_floor_test.go` (:102, :207, :231) all sit inside the limiter window; :207 and :231 survive only because their *content* checks (`"finished working"`, `"stopped working"`) carry the falsifiability — `del-drain` and `del-precedence` redden them — so :102 is the one site where the count is the whole oracle and it reports nothing.

## 4. Minor findings

- **No test crosses the startup wiring seam** (2nd in family `test-avoids-seam-not-injects-it`). Measured: re-adding `if p.notifyModeActive != "idle" { p.idleS = 0 }` after `wrap.go:2416` leaves the package GREEN — the pure resolver is asserted, the path that consumes it is not. `Run` ends in `pty.Start`, so an end-to-end test isn't available here. Rule, not instance: `p.idleS` and `p.notifyModeActive` should have exactly one assignment site, fed by `resolveNotifyConfig`, enforced by a source-scanning test — or the residue accepted explicitly in the Log.
- `notification_lifecycle.go:304` vs `wrap.go:2694` — the `kind, token := p.lifecycleTimerKind, p.lifecycleTimerToken; processLifecycleObservation(...)` pair is still written twice (BR-15, unaddressed); the `outer` + `outer-path` sidecar fixture is still at 7 test sites across 5 files.
- `notification_lifecycle.go:310-311` — `p.debug("IDLE", …)` and `traceWrap("idle", …)` still fire before the reducer decides, so a token-rejected expiry records a completed idle expiry in the log an operator will later use to reconstruct why an alert did or did not fire (BR-16, unaddressed).
- `TestIdleFloorStaysSilentWhileTheAgentIsProducingOutput` writes every 15 ms against a 60 ms deadline; a 60 ms scheduler stall on a loaded machine makes it flake. Injecting the limiter clock (above) would let this drop the sleeps too.
- Pre-existing, noted only because the idle path now sits beside it: `syncLifecycleTimer` (`:263-275`) still resets the watchdog/grace deadline on *every* reduced observation, including journal records — the exact non-idempotence BR-10 fixed on the idle side. Out of scope for #171; worth an issue.

## 5. Test coverage notes

Coverage is materially better than round 2. The pure reducer rows (a)–(f) all exist and each pins a distinct property; the three master-loop-adjacent behaviours (drain, epoch snapshot, lifecycle precedence) are now injected through `applyIdleExpiry` rather than raced, and I confirmed each reddens under its mutation. `TestReducePreservesTheGenerationInvariantForEveryObservationKind` walks 1..`observationKindCount-1`, so the enum bound is derived rather than restated, and the fuzz reaches both new kinds via the added seeds (`{0,10,11,10,4}` → UserSubmission, IdleExpired, BareReturn, IdleExpired, MarkerCompletion). Gaps: the count-oracle above; and the timer-level half of the BR-14 re-arm (that a bare CR re-arms the *proxy's* `idleTimer`, not just `IdleToken`) is pinned only in the reducer.

## 6. Architectural notes

ARCH-DRY — flag (BR-15 residue, above). ARCH-PURE — pass; `Reduce`, `resolveNotifyConfig`, `decidePlainReturn`, `idleAlertMessage` are all pure and unit-tested without IO, with timers and the TTY at the seam. ARCH-PURPOSE — flag (README/atlas shadow-sweep, above); the covered-population question BR-14 raised is now genuinely answered. ARCH-MOCK — pass; `spawnSlug` is injected, ptmx is an `os.Pipe`, the outer TTY is a temp sidecar, no new unseamed external call. ARCH-CONSTRAINTS — pass; the per-chunk Stop+Reset budget is stated and `syncIdleTimer` is idempotent so non-output observations no longer churn the deadline; wall-clock test margins are the only soft spot. ARCH-SECURE — pass, effectively N/A: no new untrusted-input parse (the only new stdin handling is a `\r` byte scan), no credential surface, message text generated internally. ARCH-ORDER — mostly pass: the epoch is now a parameter and the interleaving is injected; flagged for the count oracle (above) and BR-9's five-boolean constellation, correctly separated into pair#219 rather than bundled into a behaviour change.

## 7. Plan revision recommendations

None for the Plan — every `- [x]` row now has code that reddens a test when removed, verified independently. Add a `## Revisions` note to the issue only if the operator accepts the startup-seam residue: record that "remove the mode gate" is pinned at `resolveNotifyConfig` but not through the arg-parse path, and why (`Run` terminates in `pty.Start`).

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Bound is now observationKindCount-1 (1..12, both new kinds reachable); seeds reach each; invariant split into completion-per-generation and alert-per-epoch; walk test covers every declared kind; 3.5M execs pass.
  - id: BR-9
    disposition: not-addressed
    note: |
      Deliberately deferred to pair#219, which exists with a real spec; pre-existing pattern, Minor, non-blocking.
  - id: BR-10
    disposition: addressed
    note: |
      Verified by mutation — unconditional resetIdleTimer reddens TestSyncIdleTimerDoesNotRestartAnArmedDeadlineWithinTheSameEpoch; sub-second rendering pinned separately.
  - id: BR-13
    disposition: addressed
    note: |
      Sweep reproduced independently: 9 of 11 mutations redden a specific named test. One re-addition mutation still survives; carried forward as a Minor, since re-adding deleted code is not the deletion rule BR-13 stated.
  - id: BR-14
    disposition: addressed
    note: |
      del-passthrough-publish, del-emitplaincr-publish and del-rearm all redden; I re-verified the population enumeration (pass-through is the exclusive else of hasReturnRemap, overlay unreachable there, ptmx has one writer).
  - id: BR-15
    disposition: not-addressed
    note: |
      Both instances still present — notification_lifecycle.go:304 vs wrap.go:2694, and the outer/outer-path fixture at 7 test sites.
  - id: BR-16
    disposition: not-addressed
    note: |
      notification_lifecycle.go:310-311 still log and trace the expiry unconditionally, before the reducer decides whether the alert applies.
findings:
  - id: new
    severity: Important
    family: atlas-overstates-guarantee
    title: |
      README and atlas still promise "at most once per turn", which this round's bare-CR re-arm made false
    detail: |
      This is the 2nd finding in family `atlas-overstates-guarantee`. The rule, not the
      instance: when a state-machine invariant changes, every restatement of it must be
      swept in the same round — here the fuzz invariant was rewritten from per-generation
      to per-epoch (notification_lifecycle_test.go:196-198) while README.md:605 ("It fires
      at most once per turn") and atlas/architecture.md:783 ("a generation can carry one
      alert plus a later real completion") kept the superseded claim. atlas:785 also still
      says IdleToken is "minted once per turn" and then describes the re-arm that mints a
      second one. Enumerated restatements of the once-per invariant: 3 (fuzz, README,
      atlas x2 clauses); 1 swept, 3 stale.
  - id: new
    severity: Important
    family: unfalsifiable-race-test
    title: |
      The master-loop exactly-once oracle cannot fail — the 500ms emit limiter masks duplicates inside its 400ms window
    detail: |
      This is the 3rd finding in family `unfalsifiable-race-test`. Do NOT fix this instance
      by widening the sleep. Rule: a notification-count assertion is falsifiable only when
      the observation window exceeds rateLimitS (wrap.go:78, 500ms) or the limiter's clock
      is injected; emitOuter:652 calls time.Now() directly even though the proxy already
      carries an injectable `now func() time.Time`. Measured: with !state.IdleNotified
      dropped from both the reducer guard and syncIdleTimer, idle_floor_test.go:102 passes;
      re-run with settle(1500ms) it fails with "notifications = 3". Prevalence: 3 count
      assertions in idle_floor_test.go (:102, :207, :231) all inside the limiter window;
      :207 and :231 survive on their content checks, :102 has none.
  - id: new
    severity: Minor
    family: test-avoids-seam-not-injects-it
    title: |
      No test crosses the startup wiring seam, so the mode gate can be re-added at its original call site with the package green
    detail: |
      This is the 2nd finding in family `test-avoids-seam-not-injects-it`. Measured:
      inserting `if p.notifyModeActive != "idle" { p.idleS = 0 }` after wrap.go:2416 leaves
      the package GREEN (failure set identical to baseline). resolveNotifyConfig is
      asserted, but nothing tests the path that consumes it, and Run terminates in
      pty.Start so an end-to-end test is not available in this environment. Rule, not
      instance: p.idleS and p.notifyModeActive should have exactly one assignment site fed
      by resolveNotifyConfig, enforced by a source-scanning test — or the residue accepted
      explicitly in the issue Log.
```

---

## Re-review — 2026-09-09T03:45:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 171 — Always-on idle notification fallback |
| repo | pair |
| issue file | workshop/issues/000171-idle-notification-fallback.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9edc8a9772321eed8999dd5cacef28cdd4d0fd31..f18befa52d3a59d00385d83b9459ae515a376530 |
| command | sdlc close --issue 171 |
| reviewer | claude |
| timestamp | 2026-09-09T03:45:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3's two blocking findings are genuinely fixed, and I verified both by mutation rather than by reading the commit message: re-inserting `if p.notifyModeActive != "idle" { p.idleS = 0 }` at its original call site now reddens `TestNotifyWiringHasASingleAssignmentSiteFedByTheResolver` (BR-19), and the duplicate-producing mutation fails the master-loop count assertion with `notifications = 9` under the injected clock while the *identical* mutation with `now: time.Now` still passes (BR-18) — the clock injection, not a wider sleep, is what made the oracle able to fail. A 10-mutation sweep at head came back RED on 9/10 (the tenth was my own contrived call-site variant, not a realistic regression; the faithful BR-2 mutation is RED). Fuzz: 3.5M execs, clean. `go build ./...` + `go vet` clean; the 11 package failures are the pre-existing pty-child/sandbox set, identical at head. What keeps this from SHIP is that both new findings are *repeats of families already in play*, and in both cases the enumeration the family demands was taken from the prior finding's list rather than derived: BR-17 swept the three restatements it named and left five more in the source file itself, and BR-14's pass-through opener planted a second, paste-blind copy of the "does this CR submit?" predicate that `decidePlainReturn` owns — I reproduced a bracketed paste publishing `ObservationBareReturn` on one path and nothing on the other.

## 1. Strengths

- **`applyIdleExpiry(token)` as a parameter, not a field read** (`notification_lifecycle.go:287`). Making the epoch an argument means the call site's evaluation order pins it structurally; mutating the final observation to read `p.idleTimerToken` instead of `token` reddens `TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain`. This is the right shape for the BR-2 class, not a comment asking the next reader to preserve ordering.
- **The clock injection is the correct fix for BR-18** (`wrap.go:648-653`, `idle_floor_test.go:44-53`). `clock()` routes both `emitOuter`'s limiter and `maybeSpawnSlug`'s debounce through the proxy's *already existing* `now` seam rather than adding a new one, and the counterfactual holds: same mutation, real clock → green; injected clock → `notifications = 9`.
- **`resolveNotifyConfig`** (`wrap.go:117-125`) turns an unfalsifiable startup-path property into a pure one. `TestResolveNotifyConfigNeverZeroesTheIdleInterval` asserts the interval survives every mode, which is exactly the invariant the deleted gate violated.
- **The reducer's alert-vs-completion decision** (`notification_lifecycle.go:188-205`) is argued in the code, pinned by `TestIdleFloorIsAnAlertSoARealCompletionStillNotifies`, and the mutation "idle alert becomes a completion" reddens two tests.
- **`TestReducePreservesTheGenerationInvariantForEveryObservationKind`** walks `observationKindCount` instead of asserting equality against the last kind — it absorbs a new kind rather than failing on the change the sentinel exists to tolerate.

## 2. Critical findings

None.

## 3. Important findings

### I-1 — Five in-source restatements of the superseded "once per turn" invariant survived the sweep

`notification_lifecycle.go:37, 77, 80, 320, 351`. **This is the 3rd finding in family `atlas-overstates-guarantee`.** Do not fix these five lines and call it done — the rule is that when an invariant changes, the enumeration of its restatements must be **derived mechanically** (grep the invariant's phrasing across the tree), not inherited from the prior finding's bullet list. BR-17 named three sites (fuzz, README, atlas) and all three were swept correctly — verified by grep — but the enumeration itself was the prior reviewer's, and the class it belonged to (every place that states the once-per rule, source comments included) was never written down. Measured prevalence at head, `grep -n "once per turn\|no-op\|one alert\|minted once"`:

- `:37` — `ObservationBareReturn` "inside an open turn (answering a menu) it is a **no-op**" — false; BR-14 made it re-arm the floor.
- `:77` — `IdleNotified` "the alert fires **at most once per turn** … `open()` clears it, so each new turn is armed again" — false on both clauses.
- `:80` — `IdleToken` "It is **minted once per turn** (not once per chunk)" — false; a bare return mints a second within the same turn.
- `:320` — `syncIdleTimer` "has not already raised its **one alert** … a new turn re-arms it because `open()` clears `IdleNotified`" — incomplete for the same reason.
- `:351` — `resetIdleTimer` "`IdleToken` is **minted once per turn** by the reducer" — false.

3 swept, 5 stale. The live invariant, per the issue's own round-3 entry, is *one completion per generation, one alert per idle epoch*. The fix is the derivation: run the grep, sweep the result, and record the grep in the Log so the next invariant change reuses it rather than a hand-list. ARCH-PURPOSE (the class, not the instance).

### I-2 — `passThroughChunk`'s inline CR test is a second, divergent home for the submission predicate: a bracketed paste opens a turn and earns a spurious alert

`wrap.go:1595`. **This is the 2nd finding in family `submits-rule-second-home`.** BR-7 established the rule — "does this input submit?" is decided once, in `decidePlainReturn`, and carried on `returnDecision.submits`; the nil-profile branch was folded back in precisely so the rule would not have a second home. BR-14's fix then opened a third path that re-derives the predicate inline as `bytes.IndexByte(data, '\r') >= 0`, and it has already diverged: `translateChunk` tracks bracketed-paste state and never calls `emitPlainCR` inside a paste (`wrap.go:1868-1882`), while `passThroughChunk` has no paste awareness at all — its caller hard-sets `inPaste = false` (`wrap.go:1492`).

Measured, with a scratch probe on both paths given the identical byte string `\x1b[200~line one\rline two\r\x1b[201~`:

```
pass-through published: [12]   // ObservationBareReturn
remap path published:   []
```

Failure scenario: under `PAIR_WRAP_REMAP_RETURN=0`, or for any agent outside the four `harnessTTYProfiles` entries, the operator pastes a multi-line prompt into the agent pane (terminals normalise pasted newlines to CR) and then steps away to think without submitting. `Reduce` opens a turn on the paste, `syncIdleTimer` arms the floor, and `PAIR_WRAP_IDLE_S` later the operator is notified "no agent output for 60s" about a pane where they typed something and the agent was never asked anything. That is the untrustworthy-notification failure the Spec names as the reason not to emit in parallel.

Don't patch `passThroughChunk` with its own paste tracker — that would be a third divergence waiting. State and apply the rule: the submission predicate has one pure home, and both stdin paths consult it. The narrow shape is to give the no-remap path the same paste-state parameter `translateChunk` already carries and route its CR through the same decision function (`decidePlainReturn` with the zero profile already fails closed to `submits: true`, which is the correct answer for a *non*-paste CR there). Prevalence: 2 divergences from the single predicate in as many rounds. ARCH-DRY, ARCH-PURPOSE.

## 4. Minor findings

- **BR-15 remains open, unchanged.** `wrap.go:2705` and `notification_lifecycle.go:304` still write the lifecycle-expiry application twice; the `dir/outer` + `dir/outer-path` sidecar fixture still stands at 7 test sites (`codex_working_test.go`, `idle_floor_test.go`, `lifecycle_journal_test.go`, `notification_rewriter_test.go`, `update_agent_output_test.go`). Re-measured this round, not re-derived from the prior text.
- **BR-16 remains open, unchanged.** `notification_lifecycle.go:310-311` still emits `p.debug("IDLE", …)` and `traceWrap("idle", …)` before the reducer decides, so a token-rejected expiry — including one the drain or the precedence branch just invalidated — records a completed idle expiry in the two places an operator reconstructs the decision from.
- **The expiry drain's test fixture is not reachable in production** (`idle_floor_test.go:205`, comment at `notification_lifecycle.go:285`). `publishLifecycleObservation` has exactly four call sites (`wrap.go:1596, 1850, 1900, 1919`) and they publish only `ObservationBareReturn` and `ObservationUserSubmission`; marker/native/transcript completions are reduced directly on the master goroutine. So `TestIdleExpiryDrainAppliesAQueuedCompletionFirst` queues a kind that channel never carries, and the doc comment's "a submission **or completion** already published but not yet reduced" is false for the completion half. The drain is still pinned — the "delete the expiry drain" mutation also reddens the BR-2 test, which uses a reachable kind — so this is a comment + fixture-realism issue, not a coverage hole.
- `p.now` now serves two unrelated concerns (the `#59` scrollback time-event debounce at `wrap.go:2077` and the emit limiter/slug debounce). Fine in production where both are `time.Now`, but a future test injecting a clock for one gets the other for free; worth a line in the field doc.

## 5. Test coverage notes

- Mutation sweep at head, baseline 11 failures (all pre-existing sandbox/pty), 10 mutations: **9 RED**. RED on — idle alert becomes a completion; delete the expiry drain; delete lifecycle-deadline precedence; delete the bare-CR publish (remap); delete the bare-CR publish (pass-through); drop the re-arm on a spent floor; message loses sub-second honesty; unconditional re-arm; drop the once-per-epoch reducer guard. Plus the faithful BR-2 mutation (`Token: p.idleTimerToken` inside `applyIdleExpiry`) → RED, and re-inserting the mode gate → RED.
- The one GREEN was a call-site variant (draining before `p.applyIdleExpiry(p.idleTimerToken)` in the master loop) that no test crosses, because the BR-2 test calls `applyIdleExpiry` directly. I don't think this needs a test: passing the token as an argument makes Go's evaluation order the guarantee, and the realistic regression (the function ignoring its parameter) is caught. Noting it so it isn't rediscovered as a gap.
- `go test -race -count=2` over the idle/lifecycle/master-pump set: clean. Fuzz 20s, 3.5M execs, 177 corpus entries: clean.
- `TestIdleExpiryDrainAppliesAQueuedCompletionFirst:210` and `TestIdleExpiryGivesTheLifecycleDeadlinePrecedenceWhenBothAreReady:236` still assert `len(got) != 1` through `newIdleExpiryProxy`, which wires `now: time.Now` — so those two count clauses sit inside the 500ms limiter. Both rows survive on their content assertions (`"finished working"` / `"stopped working"`), which is why I'm not raising this as a 4th `unfalsifiable-race-test` finding, but the rule BR-18 established applies to every count assertion, not just the harness one: `newIdleExpiryProxy` should take the same advancing clock so the `len(got)` clause means something. One-line change, worth doing while I-1 and I-2 are open.

## 6. Architectural notes

- **ARCH-DRY — flag.** BR-15's two duplications plus I-2's second submission predicate. The pattern across this issue is that each fix lands correctly and then *plants* the next duplicate; the `drainStop` extraction (BR-8) is the model to follow for `applyLifecycleExpiry`.
- **ARCH-PURE — pass.** `Reduce`, `idleAlertMessage`, `decidePlainReturn`, `resolveNotifyConfig` are pure and tested with no IO, no exec, no clock. The IO shell (`syncIdleTimer`, `bumpIdleDeadline`, `applyIdleExpiry`, `emitOuter`) is thin and every external effect it reaches is injected — `writeTTY`, `spawnSlug`, `now`. No "pure" entity here needs a mock to run.
- **ARCH-PURPOSE — pass, with I-1's flag.** Shadow-sweep of the turn-opener population: Alt+Enter KKP (`wrap.go:1900`), legacy Alt+Enter (`:1919`), composer-inactive/unknown bare CR (`:1850` via `decidePlainReturn`), no-remap pass-through CR (`:1596`). The draft pane submits via `zellij action send-keys 'Alt Enter'` (`draft_send.lua:17`), so it lands on the Alt+Enter opener rather than needing a fifth. All seven Done-when bullets are delivered and each has a reddening test. The flag is the enumeration discipline in I-1.
- **ARCH-MOCK — pass.** `spawnSlug` (`wrap.go:301`) replaces the wall-clock debounce that could spawn a real model call from a test, and production and test share `maybeSpawnSlug` as the boundary. A void fire-and-forget call is faithfully modelled by a no-op, so no stateful fake or conformance cadence is owed here.
- **ARCH-CONSTRAINTS — pass.** The envelope is declared (Plan step 8) and the code enforces it: `syncIdleTimer:327` returns before touching the timer when `idleS <= 0`, keeping the opt-out at literally zero per-chunk cost, and the armed path is one `Stop`+`Reset` per read event (not per byte) against a measured 1.7 chunks/s. The alert's `pair slug` spawn is bounded at one per idle epoch.
- **ARCH-SECURE — pass / largely N/A.** No credentials, no cross-process parsing added. `PAIR_WRAP_IDLE_S` is the one external input and `TestIdleIntervalKnobParsesAnExplicitZeroAsDisabled` pins all four cases including unparseable → default, so a hand-edited value degrades to the documented behavior rather than to a fabricated interval.
- **ARCH-ORDER — pass, with the known flag.** The strongest thing in this diff is the answer to BR-3: extracting `applyIdleExpiry` converts a raced interleaving into an injected one, so the tests observe the ordering they name instead of sampling whichever the scheduler gave. Precedence between co-ready deadlines is written down and tested. The open flag is BR-9's five-boolean constellation, correctly separated into pair#219 (which restates the class faithfully, including the three disagreeing guards). One asymmetry to keep in view: the `<-lifecycleTimer.C` branch (`wrap.go:2704`) does *not* drain `lifecycleEvents` while the idle branch does — I checked and it isn't a bug (that queue carries only openers, and completing the old turn before opening the new one is the better order), but the two branches now differ for reasons no comment states.

## 7. Plan revision recommendations

None required for the code — the `## Plan` checklist matches what shipped, and round 3's Log entry already records that the Plan's "one alert per turn" wording is superseded rather than rewriting it, per the append-only convention. If I-2 is taken, add a `## Revisions` entry noting that the bare-CR opener's covered population excludes bracketed-paste content and where that predicate now lives, so the pass-through opener isn't re-derived a third time.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      Deferred to pair#219, which states the class (32 states, ~4 meanings, the three disagreeing guards) faithfully; not re-raising on this issue.
  - id: BR-15
    disposition: not-addressed
    note: |
      Re-measured at head: lifecycle-expiry application still at wrap.go:2705 + notification_lifecycle.go:304; outer-path fixture still at 7 test sites.
  - id: BR-16
    disposition: not-addressed
    note: |
      notification_lifecycle.go:310-311 still logs IDLE and traces the expiry unconditionally, before the reducer decides.
  - id: BR-17
    disposition: addressed
    note: |
      The three sites it named are swept and verified by grep; the un-enumerated source-comment class is raised separately below.
  - id: BR-18
    disposition: addressed
    note: |
      Verified both ways: duplicate mutation fails with notifications = 9 under the injected clock, passes with now = time.Now.
  - id: BR-19
    disposition: addressed
    note: |
      Verified by revert: re-inserting the mode gate at its original call site reddens the source-scanning test with "assigned at 2 sites".
findings:
  - id: new
    severity: Important
    family: atlas-overstates-guarantee
    title: |
      Five in-source restatements of the superseded once-per-turn invariant survived the sweep
    detail: |
      This is the 3rd finding in this family. The rule, not the instance: the enumeration
      of an invariant's restatements must be DERIVED mechanically (grep the phrasing across
      the tree) rather than inherited from the prior finding's bullet list. BR-17's three
      named sites are correctly swept; its enumeration was incomplete. Measured at head:
      notification_lifecycle.go:37 ("inside an open turn it is a no-op" - false, it re-arms),
      :77 ("at most once per turn ... open() clears it"), :80 ("minted once per turn"),
      :320 ("has not already raised its one alert ... a new turn re-arms it"), :351
      ("minted once per turn by the reducer"). 3 swept, 5 stale. Record the grep in the Log
      so the next invariant change reuses the derivation rather than a hand-list.
  - id: new
    severity: Important
    family: submits-rule-second-home
    title: |
      passThroughChunk re-derives the submission predicate inline and has already diverged - a bracketed paste opens a turn and earns a spurious alert
    detail: |
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
  - id: new
    severity: Minor
    family: fixture-not-production-reachable
    title: |
      The expiry-drain test queues an observation kind that channel never carries in production
    detail: |
      publishLifecycleObservation has exactly four call sites (wrap.go:1596, 1850, 1900, 1919)
      and they publish only ObservationBareReturn and ObservationUserSubmission; marker,
      native and transcript completions are reduced directly on the master goroutine.
      TestIdleExpiryDrainAppliesAQueuedCompletionFirst (idle_floor_test.go:205) queues an
      ObservationMarkerCompletion on p.lifecycleEvents, and applyIdleExpiry's doc comment
      (notification_lifecycle.go:285) says "a submission or completion already published but
      not yet reduced" - the completion half is unreachable. The drain itself stays pinned by
      the BR-2 test, which uses a reachable kind, so this is fixture realism and a false
      comment rather than a coverage hole.
```
