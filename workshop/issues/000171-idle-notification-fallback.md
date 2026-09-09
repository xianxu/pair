---
id: 000171
status: working
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-09
estimate_hours: 1.86
started: 2026-09-09T01:08:11-07:00
---

# Always-on idle notification fallback

## Problem

**There is no floor under attention.** If a turn is submitted and the
recognizers never fire, the operator waits forever with no signal.

Two mechanisms exist and neither covers it:

1. **The lifecycle watchdog** (`wrapcmd/notification_lifecycle.go:10`,
   `lifecycleWatchdogAfter = 60s`) is armed only by *activity* —
   `ObservationWorking` and `ObservationTranscriptStarted` mint the token
   (:93, :104). `ObservationUserSubmission` calls `open("", false)`, leaving
   `ActivitySeen = false`, and `syncLifecycleTimer` (:197) only arms when
   `state.Active && state.ActivitySeen`. So the watchdog covers "was working,
   then went quiet" and cannot cover "never recognized as working" — which is
   the fragile case, since recognition-at-start is what breaks (the exact
   `• Working (… esc to interrupt)` match for Codex; #139 spent 23.4h on
   recognizers rejecting ordinary composing states).

2. **The idle timer is dead code.** `defaultIdleS = 60s` (`wrap.go:90`) fires
   `emitOuter("agent idle")` on no output (`wrap.go:2694-2697`) — but
   `wrap.go:2372` does `if p.notifyModeActive != "idle" { p.idleS = 0 }`, and
   `notifyMode` (`wrap.go:98-102`) is a hardcoded map with no override: claude
   is `"marker"`, everything else defaults to `"native"`. **Nothing is ever
   `"idle"`**, so the timer never arms in any shipped configuration. The string
   `"idle"` appears only in that comparison and in comments — the gate tests
   against a value nothing produces.

So the fallback was built and then locked behind a mode that does not exist.

## Spec

**Make idle a floor, always armed — but route it through the lifecycle reducer
rather than emitting directly.** Emitting in parallel would double-notify the
healthy case: a normal claude turn already emits `agent finished working` at
the marker, and a second `agent idle` 60s later is exactly the noise that makes
notifications feel untrustworthy.

- Add `ObservationIdleExpired` to the reducer
  (`wrapcmd/notification_lifecycle.go`). It completes **only** when
  `state.Active && !state.Completed` — i.e. a turn is open and nothing else has
  already reported its end. A turn that completed normally swallows it.
- Delete the `notifyModeActive != "idle"` gate (`wrap.go:2372`) and the
  vestigial `"idle"` mode value with it. The timer arms for every agent.
- Keep `PAIR_WRAP_IDLE_S` (`wrap.go:2284`) as the knob, including `0` to
  disable — an escape hatch on a always-on mechanism.
- **Message must be honest.** Not `agent finished working` (unknown) and not
  `agent stopped working` (it may never have started). Something like
  `no agent output for 60s`, which also self-reports that recognition failed.

The reducer is a pure function with existing unit tests, so the new observation
is testable without goroutines or a terminal (`ARCH-PURE`), and reusing it
rather than adding a second emit path keeps one owner for "has this turn been
reported" (`ARCH-DRY`).

**The requirement is "it always fires", not "the timer is wired".** This is a
floor: after an open turn goes unreported for the interval, the operator hears
something, whatever the agent did or did not emit. A trigger a repainting pane
can defeat is not a floor. If the measurement below shows the byte stream never
goes quiet, the trigger moves — to time since the last *lifecycle transition*,
or failing that time since turn open — rather than the requirement bending to
what the existing timer happens to measure.

**Verify this empirically before building it.** The idle timer resets on every
output chunk (`wrap.go:2680-2690`), so it measures *byte* silence, not agent
silence. If a claude pane repaints while idle at its composer — cursor blink,
spinner, status line — the timer never expires and the fallback is inert in a
new way rather than dead in the old way. Measure how long a real idle pane goes
byte-quiet first; if it is never 60s, the timer needs a coarser trigger than
raw chunks (e.g. reset only on chunks that change the rendered screen).

## Done when

- Submitting a turn whose activity is never recognized produces exactly one
  notification at ~`PAIR_WRAP_IDLE_S`.
- A normally completed turn produces exactly one notification, with no idle
  duplicate afterward — asserted in the reducer's unit tests.
- The idle timer arms for `claude` (marker mode) and for a `native`-mode agent,
  not just a mode nothing selects.
- `PAIR_WRAP_IDLE_S=0` disables it.
- The `"idle"` notify-mode value no longer exists in the source.
- A recorded measurement of byte-quiet duration for an idle claude pane is in
  the `## Log`, taken before the trigger design was fixed.
- **The floor covers the population that has no other timer.** Drive a turn
  that is never recognized as working (no progress OSC) and then goes
  byte-quiet, and assert exactly one notification at the interval. This
  supersedes the repainting-pane criterion — see `## Revisions` 2026-09-09.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: pensive              design=0.40 impl=0.10
item: smaller-go-module    design=0.15 impl=0.16
item: smaller-go-module    design=0.15 impl=0.20
item: smaller-go-module    design=0.10 impl=0.20
item: atlas-docs           design=0.05 impl=0.06
item: milestone-review     design=0.00 impl=0.16
design-buffer: 0.15
total: 1.86
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Rows: the measurement + Log analysis
(`pensive`); the reducer change; the `wrap.go` glue; the master-loop
integration test; atlas; one close-boundary review. `design-buffer: 0.15`
because the Spec resolves the decisions and `## Revisions` fixes the trigger.

## Plan

**Design decision (disposes PQ-1) — idle expiry is an ALERT, not a completion.**
`complete()` asserts the turn ended; a 60s silence does not know that. Routing
the expiry through `complete()` would mark `Completed=true`, so the agent's
genuine end-of-turn would then be swallowed by the `Active && !Completed` guard
— the operator gets pulled in early *and* never told it actually finished. So
`ObservationIdleExpired` sets a separate `IdleNotified` flag, notifies, and
leaves `Active=true, Completed=false`. A later real completion still notifies.
The Spec's double-notify worry is already answered by the trigger: a healthy
turn is never byte-quiet for the interval (`## Log` measurement 1), so the alert
does not fire on it. `IdleNotified` is per-turn (cleared in `open()`), bounding
the alert to one per turn rather than every interval while the operator is away.

- [x] Measure byte-quiet duration for an idle claude pane and an idle codex
      pane; record in `## Log`. This decides the trigger, so it comes first.
- [ ] **Reducer** (`notification_lifecycle.go`, pure — `ARCH-PURE`). Add
      `ObservationIdleExpired` and `ObservationBareReturn`; add `IdleNotified
      bool` + `IdleToken uint64` to `NotificationLifecycle`. `open()` clears
      `IdleNotified` and mints `IdleToken`; `complete()` zeroes `IdleToken`.
      `ObservationIdleExpired` notifies iff `Active && !Completed &&
      !IdleNotified && Token != 0 && Token == IdleToken`, sets `IdleNotified`,
      and does **not** complete. Message from `observation.Message`, defaulting
      to a constant — the reducer stays free of the env knob.
- [ ] **Timer arming tied to turn state (disposes PQ-2).** The idle deadline's
      epoch is the turn, not the last byte. Add `syncIdleTimer()` beside
      `syncLifecycleTimer()` and call it from `processLifecycleObservation`, so
      arming is owned by lifecycle state exactly as the watchdog/grace timers
      are: arm iff `idleS > 0 && Active && !Completed && !IdleNotified &&
      IdleToken != 0`, else stop. The chunk path resets only the *duration*,
      keeping the same `IdleToken` — so the token identifies the turn's idle
      epoch and is minted once per turn, not once per chunk. Delete the
      `idleFired` bool: the latch it provided is now `IdleNotified`, which
      `open()` clears, so a turn opened after a previous expiry is armed again.
      `case <-idleTimer.C` drains `p.lifecycleEvents` first, exactly as the
      chunk branch does at `wrap.go:2671-2679`, so a queued submission reduces
      before the expiry is applied.
- [ ] **Remove the mode gate.** Delete `if p.notifyModeActive != "idle" {
      p.idleS = 0 }` (`wrap.go:2372`) and the `"idle"` mode value. The timer
      arms for every agent — marker mode (claude) and native mode alike.
      `PAIR_WRAP_IDLE_S` stays the knob, `0` still disables.
- [ ] **Cover bare-CR submissions (disposes PQ-3).** `ObservationUserSubmission`
      is published only on the two Alt+Enter forms (`wrap.go:1836,1855`). Plain
      Enter with the composer *active* is remapped to a newline (correct — not a
      submission), but with the composer inactive/unknown it passes a bare CR
      through (`harness_tty.go:120-126`) and opens no turn, so an agent driven
      directly in-pane has no floor at all. Publish `ObservationBareReturn` on
      that bypass branch; the reducer opens a turn **only when none is active**,
      so a menu answer inside an open turn stays a no-op and turn identity is
      never reset mid-turn.
- [ ] **Honest message text.** `no agent output for 60s` (interval formatted
      from `p.idleS`) — not `finished` (unknown) and not `stopped` (may never
      have started). Confirm it survives the OSC 777 envelope into couch.
- [ ] **Tests (disposes PQ-4).** Pure rows drive `Reduce` directly in
      `notification_lifecycle_test.go`: (a) unrecognized turn — submission then
      idle expiry notifies; (b) completed turn — submission, marker completion,
      idle expiry is silent; (c) **alert-not-completion** — submission, idle
      expiry notifies, then marker completion *still* notifies; (d) one alert
      per turn; (e) stale `IdleToken` is silent; (f) `ObservationBareReturn`
      opens a turn only when none is active. Master-loop rows reuse the existing
      in-process seams — `p.masterPump()` over an `os.Pipe` ptmx as in
      `lifecycle_journal_test.go:279-321`, emit observed via `outerTTYFile` as in
      `notification_rewriter_test.go:194` — with `p.idleS` set to milliseconds so
      no test waits on wall-clock: armed in marker mode *and* native mode, and
      `PAIR_WRAP_IDLE_S=0` disables. Risky surface is the master-loop `select`;
      its adversarial class is **idle expiry racing a queued submission and a
      completion**, with arrival order *injected* (enqueue on `p.lifecycleEvents`
      before firing the timer) rather than sampled.
- [ ] **Operating envelope (disposes PQ-5).** Always-arming adds one
      Stop/drain/Reset per chunk for every agent, where `idleS == 0` previously
      skipped it. Budget: the measured busy pane ran 203 chunks / 123s ≈ 1.7
      chunks/s, and a `time.Timer` Stop+Reset is sub-microsecond, so the added
      cost is <1µs/s on the keystroke-latency path — asserted here rather than
      assumed, and bounded because the reset is O(1) per chunk.
- [ ] **Atlas (disposes PQ-6).** `atlas/architecture.md:783` enumerates the
      reducer's openers and terminals and `:965` describes the "idle/native OSC
      for codex/agy" modes; both go stale when `"idle"` is deleted and the two
      observations are added. Update at close.

## Log

### 2026-09-02

Operator decision: the floor is the priority, and up to a minute of latency
attending to something is acceptable. General notification robustness is
separate later work.

Motivating case, unresolved: a selection menu is displayed and nothing pages.
For claude, **two** existing paths should already cover that — a progress-OSC
`ObservationStopped` gives a 250ms grace to `agent stopped working`
(`notification_lifecycle.go:109,145`), and the 60s watchdog armed by the
earlier `ObservationWorking` fires regardless. Neither is observed, so the
signal is failing upstream of every timer — most likely claude keeps emitting
progress OSC while the menu is up, re-arming the watchdog. That is exactly the
case that would also defeat a byte-reset idle timer, which is why the
repainting-pane test above is an acceptance criterion and not a nicety.

Found while tracing what state transitions a pair actor actually publishes to
couch. Related finding, not fixed here: `decision.Notify` is set only inside
`complete()`, so couch is told when a turn *ends* and never when one *begins* —
it cannot know an actor is currently working, only that it finished at some
past moment. That blocks a "working 4m / idle 31m" display in the switcher and
wants its own issue if that display is ever built.

### 2026-09-09

**Measurement done first, as the Spec required — and it refutes the hypothesis
the Spec was built on.** Three measurements, all from live/recorded dogfood
panes (no synthetic harness), scripts in `$TMPDIR` during the session.

*Instrument.* The scrollback tee (`wrap.go:2779-2786`) writes every chunk inside
`handleChunk` — the same call that resets the idle timer — and then calls
`maybeLogTime()`, which appends a `{"type":"time","ts":…,"offset":N}` record to
the `.events.jsonl` sidecar at most once per minute *and only on new output*
(`dueForTimeEvent`). So consecutive time events `T1,T2` prove no chunk arrived
in `(T1+60s, T2)`: a recorded lower bound on byte-quiet, already on disk for
every past session. The progress OSC itself (`ESC]9;4;<state>`,
`notification_rewriter.go:90-105`) is in the raw stream, so the lifecycle
observation timeline can be reconstructed too.

1. **Idle panes do go byte-quiet — the repaint worry is empirically false.**
   Live-sampled 18 claude panes' raw scrollback size every 0.5s for 123s:
   **17 of 18 were byte-silent for the entire window**; the only pane emitting
   bytes was this session's own, actively working (+143156B in 203 chunks).
   Historically, across 7934 minute-intervals: 14.1% show ≥60s of byte silence,
   p90 = 331s, p95 = 1740s. A claude pane parked at its composer does not blink,
   spin, or repaint. **The byte-reset trigger is not defeated in practice.**

2. **Claude's progress OSC is a clean matched pair, not a repeating heartbeat.**
   Across 8 panes: 231 `9;4;3` (working) against 236 `9;4;0` (stopped) — pairs,
   never runs of `working`. 94.2% of consecutive working→working gaps are ≥60s
   (p50 = 554s). **The Log's 2026-09-02 hypothesis — "claude keeps emitting
   progress OSC while the menu is up, re-arming the watchdog" — is refuted.**
   Nothing re-arms the watchdog; the watchdog is simply never armed.

3. **The progress OSC barely fires at all, so most turns arm no timer.** 75
   working events across one 48.6h session, p50 554s apart — nowhere near
   per-turn. For the large majority of turns `ObservationWorking` never fires,
   so `open("", false)` leaves `ActivitySeen = false` and `syncLifecycleTimer`
   (:198) arms **nothing**. Problem #1 in the Spec is not the rare fragile case;
   it is the *dominant* case.

**Consequence for the trigger design.** (1) removes the reason to abandon the
byte-reset trigger, and (3) says the uncovered population is large and is
exactly the population that *is* byte-quiet. So the cheap existing timer, routed
through the reducer, covers the real gap — no screen-diff hashing on the
per-chunk path (`ARCH-CONSTRAINTS`: that path is keystroke-latency), no second
deadline (`ARCH-DRY`). What it still cannot do is fire for a turn that emits
output forever without completing — the Spec's last Done-when bullet. That
bullet was written to defend against the case measurement (2) just refuted, so
it is now a policy question for the operator rather than a forced design change;
raised before implementing.

## Revisions

### 2026-09-09 — repainting-pane criterion superseded by measurement

**Reason.** The Spec required the measurement first and pre-committed to a
consequence: "if the byte-reset trigger cannot pass it, the trigger changes, not
this criterion." The measurement (`## Log` 2026-09-09) refuted the premise that
criterion defends. An idle claude pane does not repaint — 17 of 18 live panes
were byte-silent across a 123s window — and claude's progress OSC emits matched
`working`/`stopped` pairs rather than a repeating heartbeat, so nothing re-arms
the watchdog. The real gap is that the OSC is absent for most turns, so
`ActivitySeen` stays false and **no** timer is armed; that population is
precisely the one that is byte-quiet.

**Delta.**
- Done-when: "A pane that repaints while waiting still notifies" → "The floor
  covers the population that has no other timer" (unrecognized turn, byte-quiet,
  exactly one notification at the interval).
- Plan: "Repainting-pane test" → "Unrecognized-turn test".
- Trigger stays the existing byte-reset idle timer, always armed and routed
  through the reducer. No screen-diff hashing on the per-chunk path
  (`ARCH-CONSTRAINTS` — keystroke-latency path), no second deadline
  (`ARCH-DRY`).

**Accepted limitation.** A turn that emits output forever without completing is
not covered. Operator chose this explicitly over a never-reset deadline, which
would fire on every turn longer than 60s with a message that would be false.

### 2026-09-09 — plan rewritten to dispose of plan-quality round 1

**Reason.** `sdlc change-code` round 1 raised PQ-1 (Critical) plus PQ-2..PQ-6.
PQ-1 was right that the reducer's completion contract is a design decision, not
a diff detail; PQ-3 was right that Alt+Enter is the only path that opens a turn.

**Delta.** Plan now decides idle expiry as an **alert, not a completion**
(`IdleNotified` + `IdleToken`, turn stays open so a real completion still
notifies); ties timer arming to lifecycle state via `syncIdleTimer()` and
deletes the `idleFired` latch; drains `lifecycleEvents` on expiry; adds
`ObservationBareReturn` so composer-inactive bare-CR turns get a floor; names
`Reduce` and the two existing in-process seams for tests plus the injected race
class; states the per-chunk timer-reset budget; adds the atlas step. Estimate
derived (1.86h, v3.1) after the plan settled, per #187 ordering.
