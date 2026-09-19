---
id: 000279
status: codecomplete
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T16:25:00-07:00
flow: {kind: quick, provenance: inferred, spec: "df59b374", done: "1c64ca99"}
actual_hours: 0.85
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

- [x] alt+d in the switcher dispatches `leave{mode:detach}` after an actor on the
      alternate screen has been shown. A Couch test encodes the key from the host
      terminal's current flags, using the independent per-screen keyboard model
      `keyboardHost.press`, so a future change to screen or keyboard handling
      cannot silently re-break it.
- [x] The presenter's keyboard push is balanced per screen. Replaying its parent
      stream into the vt emulator shows the alternate screen disambiguated while
      presented. After release, or after a write cut at any byte, both screens'
      flags and stacks are back to what they were before.
- [x] alt+x in the switcher is verified live (it works) and recorded.
- [x] The introducing commit is identified (`f32bb4cf`) and recorded.
- [x] `park.go:111`'s wording says the key picks the disposition and the scope
      picks the target (the switcher means every live thread), and it cites the
      console tests that pin both chords.
- [x] Operator smoke: in a rebuilt Couch, after visiting a thread, Ctrl+Space
      then Alt+d detaches every thread and leaves Couch.

## Plan

- [x] Measure: is alt+x also dead in the switcher? No; only alt+d. Traced what
      the console does with `seqDetach` while the panel holds focus: the handler
      is correct.
- [x] Find the introducing change: `f32bb4cf`, not `cea10ac4` or `df2a8897`.
- [x] Red: a Couch end-to-end test (alternate-screen actor → panel → alt+d
      encoded from the host's per-screen flags → `leave`), plus a presenter per-screen
      keyboard test that includes the write-cut sweep.
- [x] Fix in `terminal.Presenter`: push after `?1049h`, pop before `?1049l`
      (both on the paint path and on release), and track ownership per screen.
- [x] Reword `park.go:111`; atlas note on the presenter's per-screen keyboard
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
- 2026-09-18 (close review): Done-when row 1 and its Plan step named the host
  emulator's `SendKey` as the encoder. As delivered, the Couch test encodes with
  `keyboardHost.press` over the independent per-screen `keyboardModel`, because
  the Couch fixture's host is a `FakeHost` with no `SendKey`. Reworded both to
  match; the guarantee (the key is encoded from live per-screen flags) is
  unchanged.

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
- 2026-09-18: closed — Root cause: kitty keyboard flags are per-screen; the presenter pushed once (f32bb4cf, #255 M3) and zellij actors move the parent to the alternate screen, where alt+d arrived as legacy ESC d. Fix in terminal.Presenter.writeFramePacket: push after ?1049h, pop before ?1049l and at release. Red then green: TestKeyboardPhysicalAltDLeavesFromTheSwitcher, TestPresenterKeyboardPushFollowsTheScreen, the keyboard check in the cut-write sweep, and the de-raced TestKeyboardPhysicalNotificationJump ?1049h case. Mutation-checked: 5 mutations, each caught. go test ./... green (72 packages, sandbox off, retention env scrubbed). make -k test fails only the known pre-existing test-changelog. alt+x verified live. Operator smoke 09-18: rebuilt couch, opened a thread, Ctrl+Space then Alt+d detached every thread and left Couch.; review verdict: FIX-THEN-SHIP

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
- Red, then green:
  - `TestKeyboardPhysicalAltDLeavesFromTheSwitcher` (couchtty) encodes alt+d
    with the independent per-screen `keyboardHost` model. Before the fix its
    alternate-screen cases sent `\x1bd` at flags 0 and dispatched nothing,
    which is the operator's report reproduced.
  - `TestPresenterKeyboardPushFollowsTheScreen` (terminal) replays the parent
    stream into the vendored vt emulator, with ambient flags 5 on the main
    screen and 9 on the alternate one. Before the fix the alternate screen read
    9 while presented; now it reads 3, and both screens read 5 and 9 after
    release.
  - The cut-write sweep (`TestPresenterReleaseClosesSyncAfterAnyCutWrite`)
    also asserts both screens are restored at every cut.
- Found and fixed a race in `TestKeyboardPhysicalNotificationJump`: its
  `?1049h` case encoded Ctrl+Return before the alternate frame was painted, so
  it always read the main screen and hid this regression for three days. It now
  awaits the screen, and went red on the unfixed code. Recorded as a lesson.
- Fix (`presenter.go`, `writeFramePacket`): the push follows the parent onto
  the alternate screen and is popped before leaving it or at release, and
  ownership is recorded only for whole writes. The screen accounting
  (`altOwned`) moved there from `write()`, so one place owns it.
- Mutation-checked, each caught: no push on enter; no pop before leave; no pop
  at release; a partial push counted as owned; alt not recorded on enter.
- ARCH-FUNERAL: creates nothing durable. The one new resource is a terminal
  stack entry, and its end is named: the pop before the alternate screen is
  left, or at release. ARCH-ORDER: the balance holds at every write cut, per the
  sweep.
- Verification: `go test ./...` green (72 packages, sandbox off, retention env
  scrubbed). `make -k test` fails only the known pre-existing `test-changelog`
  ("process target is outside selected owner directory"), and every other
  target passes.
- Pair's right-hand terminal (`termcmd`) shares the presenter and gets the same
  balance. Its own tests are green.
- Operator smoke (09-18): stopped the old Couch (pid 76024, SIGTERM), ran
  the rebuilt Couch from this branch, opened a thread, then pressed Ctrl+Space
  and Alt+d. It detached every thread and left Couch ("works (global detach)").
- Close review (round 1): FIX-THEN-SHIP, with six Minor findings and nothing
  blocking. Handled in the close commit:
  - *Done-when names `SendKey`* (doc-claim-accuracy): reworded the row and its
    Plan step to the mechanism actually used; see Revisions.
  - *`terminal_input.go` lists two enhanced-only chords where the table has
    seven* (same family): the comment now points at the table's no-legacy rows
    ("Alt+d and Alt+n among them") instead of restating a list that can go stale.
  - *`hostty.EnableKeyboardDisambiguation` is dead* (dead-exported-surface):
    deleted. It is the #251 mechanism this issue diagnosed, and its comment still
    described it as Couch's policy. The same family is much larger than the
    review named: every `hostty/control.go` symbol has been test-only since #255
    M3. That sweep is filed as #289 rather than widening this bugfix.
  - *`writeFramePacket` spells write-then-record three ways*
    (repeated-write-then-record): collapsed into `Presenter.writeWhole`, which
    returns whether the whole control landed. Every ownership bit is recorded
    from it.
  - *`keyboardHost` re-implements vt's kitty stack and encoder* (same family):
    not addressed, deliberately. `keyboardHost` is #251's independent reading of
    the protocol, with its own bounds, partition and fuzz tests. Keeping it
    beside the vendored vt gives the Couch tests an oracle that does not share
    an implementation with the emulator Couch's endpoints run on. This issue
    only generalized its encoder (`press`) from one key to three.
  - *Smoke evidence uncommitted* (durable-record-uncommitted): it lands in this
    close commit; the unrelated working-tree changes stay out.
  - Also from the review's notes: `TestPresenterKeyboardPushFollowsTheScreen`
    now covers a first paint that is an alternate-screen actor, so setup's push
    and the alternate screen's land in one paint. Mutation-checked: removing the
    alternate-screen push fails both orderings. The `parentReleaseControls`
    comment now states that the DEC modes are terminal-wide and the kitty stack
    is the one per-screen setup state.
- Full-suite rerun after the review fixes: `TestConsoleRunRootEscapeClearsFilterThenReplaysActor`
  failed once ("returned actor was not admitted"), and once more in 30 isolated
  runs. It was 0/400 on both `main` and this branch, even run side by side. The
  path it drives is the same on both, since its child never switches screens.
  The test itself races: it reads `View().Admitted` once, as soon as the text is
  on screen, but admission is the `PresentView` transition that follows the
  frame's writes. That is the lesson's class, so it now waits for admission.
