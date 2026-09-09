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
