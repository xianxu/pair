---
id: 000279
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T16:25:00-07:00
flow: {kind: quick, provenance: inferred, spec: "df59b374", done: "1c64ca99"}
---

# alt+d no longer detaches from the switcher

## Problem

Operator report, 2026-09-17: **alt+d stopped working in the switcher.** It used
to detach the selected thread; now the keystroke does nothing. "Stopped" is the
load-bearing word — this is a regression, not an unbuilt feature.

The contract it violates is stated in the code. `couchcore/park.go:111`:

> Alt+d detaches and Alt+x parks, **in an actor or in the switcher alike**.

So the switcher is explicitly in scope for both lifecycle chords, and detach is
the gesture the rest of the system is built around — `actionableinventory.go:77`
describes detached-with-no-client as "the state an `alt+d` detach leaves behind",
and `detachedsessions.go:28` calls it "exactly what `pair resume` reattaches
onto". Losing it from the switcher removes the operator's way of reaching that
state for a thread they are not currently inside.

### Standing hypothesis — recognized, but forwarded to a child that isn't there

`couchtty/keys.go:228` keeps alt+d recognized for the switcher, and describes the
handling this way:

> Lifecycle candidates remain recognized for the switcher. Console **forwards
> their exact bytes** when prefix routing leaves actor focus.

On the switcher panel there is no child. If that forward is the whole handling,
the bytes land nowhere and the chord is silently inert — which is the reported
symptom exactly. This is a reading of two comments plus the chord table
(`keys.go:230` maps `ChordAltD` → `seqDetach`), **not a traced execution path**;
confirm before building on it.

It predicts **alt+x is equally dead in the switcher**. Checking that is the
cheapest first measurement, and it discriminates: both dead points at the shared
forward-to-child path, alt+d alone points somewhere specific to detach.

### Regression window

Two candidates, both touching input routing this month, newest first:

- `cea10ac4` — **#265: couch: one panel-aware door for every input** (09-16).
  Routes key release, focus and blur through the panel check and drops them on
  the panel because "there is no child, and a key release, focus or blur has no
  panel meaning". Nearest in time and in subject matter.
- `df2a8897` — **#245: pass unreserved keys to the focused agent**.

Neither is accused; both are where to `git log -p` first. Whether the switcher
path ever worked since the Go port is itself worth confirming — the operator's
"stopped" is strong evidence it did, but the bisect is what settles it.

## Spec

Revised 2026-09-18; see Revisions. The contract stays #170's, and the operator
confirmed it: **alt+d in the switcher detaches every live thread and leaves
Couch**. That is the only way out of Couch that stops no agent. Per-row detach
stays at Tab → detach. The defect is that alt+d never reaches that handler.

Root cause, measured live on 09-18:
- Symptom: alt+d types a literal `d` into the switcher filter, while alt+x
  opens its leave confirmation.
- Cause: the terminal sends legacy `ESC d` there. alt+d, alt+n and ctrl+return
  are recognized only in their kitty (CSI u) form, while alt+x also has a legacy
  `ESC x` row.
- Kitty keyboard flags live on a stack kept separately for the main and the
  alternate screen (the kitty spec, Ghostty, and our vendored vt
  `third_party/vt/pair_keyboard.go`).
- The presenter pushes `\x1b[>3u` once, at its first paint, on the main screen.
  It then moves the parent onto the alternate screen whenever a zellij actor is
  presented (`HistoryRender` emits `?1049h`), and the panel paints wherever the
  last frame left it. So after the first actor is shown, the parent runs legacy
  keys.
- Regression: `f32bb4cf` (#255 M3, 09-15), which replaced #251's re-assertion of
  `\x1b[=1;2u` after every complete child output batch with the single push.

The fix and its invariants:
- **The keyboard entry follows the screen.** Every screen the presenter puts the
  parent on carries the presenter's keyboard push. Entering the alternate screen
  pushes on it, and leaving pops first, so each stack is balanced.
- Nothing is left behind on either screen after release, or after a write cut at
  any byte. The next program to use the alternate screen must not inherit
  Couch's flags.
- The fix lives in `terminal.Presenter`, the one owner of parent modes, so Pair's
  presenter gets it too. No per-chord legacy rows: `ESC d` cannot be told apart
  from Esc followed by `d`, and that is why the table omits it.

## Done when

- [ ] alt+d in the switcher dispatches `leave{mode:detach}` after an actor on the
      alternate screen has been shown. A Couch test encodes the key with the
      host emulator's own `SendKey`, under whatever flags Couch left on that
      screen, so a future change to screen or keyboard handling cannot silently
      re-break it.
- [ ] The presenter's keyboard push is balanced per screen. Replaying its parent
      stream into the vt emulator shows the alternate screen disambiguated while
      presented. After release, or after a write cut at any byte, both screens'
      flags and stacks are back to what they were before.
- [ ] alt+x in the switcher is verified live (it works) and recorded.
- [ ] The introducing commit is identified (`f32bb4cf`) and recorded.
- [ ] `park.go:111`'s wording says the key picks the disposition and the scope
      picks the target (the switcher means every live thread), and it cites the
      console tests that pin both chords.
- [ ] Operator smoke: in a rebuilt Couch, after visiting a thread, Ctrl+Space
      then Alt+d detaches every thread and leaves Couch.

## Plan

- [x] Measure: is alt+x also dead in the switcher? No; only alt+d. Traced what
      the console does with `seqDetach` while the panel holds focus: the handler
      is correct.
- [x] Find the introducing change: `f32bb4cf`, not `cea10ac4` or `df2a8897`.
- [ ] Red: a Couch end-to-end test (alternate-screen actor → panel → alt+d
      encoded by the host emulator → `leave`), plus a presenter per-screen
      keyboard test that includes the write-cut sweep.
- [ ] Fix in `terminal.Presenter`: push after `?1049h`, pop before `?1049l`
      (both on the paint path and on release), and track ownership per screen.
- [ ] Reword `park.go:111`; atlas note on the presenter's per-screen keyboard
      ownership.

## Revisions

- 2026-09-18: Spec and Done-when restated after the trace and the operator's
  live check. The original spec said alt+d detaches the **selected** row and the
  switcher stays open. The operator chose to keep #170's contract (detach every
  live thread and leave). The original rows were: detach the selected thread,
  with a panel-path test; verify alt+x; identify the introducing commit; back
  `park.go:111` with a test. The first row became "alt+d reaches its declared
  handler"; the other three stand, re-worded. Added the per-screen keyboard
  invariant and an operator smoke.

## Log

### 2026-09-17

- Filed from a brain advisor session on the operator's report. No task existed:
  the nearest issues are #177 (drop a parked thread from the switcher), #205
  (batch park and detach run in parallel), #236 (switcher ordering) and #175
  (resume a parked thread from the panel) — none of them this.
- Hypothesis and regression window above are from reading `keys.go`, `park.go`
  and the recent input-routing commits; nothing was executed against a live
  switcher.

### 2026-09-18

- Traced the path; the issue's hypothesis doesn't hold. With the panel focused,
  `dispatchInputCandidate` (`couchtty/console.go`) does **not** forward: the
  forward arm is `actorFocused && !actorReserved()`. On the panel every hit goes
  to `hitHandlers` → `onDetachHotkey` → `reduceParkHotkey("leave", detach)` →
  the `leave` operation → `Stop()`.
- **The spec contradicts the declared contract.** Since #170 (`516a61fe`,
  09-03), alt+d in the switcher means "detach every live thread and leave Couch".
  `park.go:111`'s "in an actor or in the switcher alike" is about the
  *disposition* (d = detach, x = park) and not the target; `couch --help` and the
  Alt+h page (#282) say the same. No version ever had switcher alt+d detach the
  highlighted row. Before #170 it printed "detach: no attached thread".
- Console-level tests drive raw `\x1b[100;3u` through stdin → decoder →
  interceptor → dispatch and see `leave{mode:detach}` plus a Run exit
  (`TestConsoleRunAltDOnThePanelDetachesEveryThreadAndLeaves`). They pass at HEAD
  and at `64d0cf4e` (09-17, the report date; `git archive` into a scratch dir).
  So in-process the path is not dead. Whatever the operator hit lives outside
  what those tests cover: the terminal's bytes, an operation already in flight
  (`dispatchMenuOperation` silently refuses one while `InFlight` is set), or the
  production `leave` op.
- Operator terminal: Ghostty/cmux with `macos-option-as-alt = true`. Couch pushes
  kitty flags 3, so alt+d should arrive as `\x1b[100;3u`. The legacy `\x1bd` is
  deliberately not a chord (`productKey`), so a terminal in legacy mode would
  make alt+d exactly as inert as reported, while alt+x (`\x1bx` is registered)
  would keep working.
- Operator live check (09-18, Couch built 14:36 from main): Ctrl+Space then
  alt+d → "the alt modifier is ignored and letter d appears as part of filter
  expression". Ctrl+Space then alt+x → the leave confirmation appears. That
  settles the legacy-encoding branch above.
- Root cause: per-screen kitty keyboard stacks, and the presenter pushed once.
  Details in the revised Spec. It also silently killed ctrl+return
  (notification jump, enhanced-only) and alt+n in the switcher, for the same
  reason. The test suite missed it because every Couch input test writes kitty
  bytes to stdin directly and never asks the host terminal what it would send.
