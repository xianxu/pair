---
id: 000171
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
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
- **A pane that repaints while waiting still notifies.** Drive a turn that
  emits output continuously with no completion, and assert exactly one
  notification at the interval. This is the acceptance test for "floor" — if
  the byte-reset trigger cannot pass it, the trigger changes, not this
  criterion.

## Plan

- [ ] Measure byte-quiet duration for an idle claude pane and an idle codex
      pane; record in `## Log`. This decides the trigger, so it comes first.
- [ ] `ObservationIdleExpired` in the reducer + unit tests for both the
      uncovered case and the no-duplicate case.
- [ ] Remove the mode gate and the `"idle"` mode value; arm the timer always.
- [ ] Honest message text; confirm it survives the OSC 777 envelope into couch.
- [ ] Repainting-pane test: continuous output, no completion, exactly one
      notification at the interval.

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
