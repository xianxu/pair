---
id: 000182
status: open
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-04
estimate_hours:
---

# Relaunch an actor: restart Pair in place, keeping the agent conversation

## Problem

Developing Pair while working inside it has no clean restart. Rebuild the
binary and the running actor keeps the old code; the only way to pick up a new
Pair is to lose the session, and losing the session means losing the agent
conversation with it.

couch already has the symmetric move one level up: `alt+d` detaches everything
and leaves, so rebuilding and re-running `couch` picks up new couch code with
every agent still running behind its zellij session. The actor level has no
equivalent -- which is the level where Pair development actually happens.

Pair's own `Alt+n` (`ActionRestartPair`, "reload pair -- kill and re-launch the
workbench in place") is close but not the same thing. `launcher/restart.go`
writes a restart marker, touches the quit marker and execs `kill-session`; the
already-running Pair process's outer loop then regains control and re-enters
its create flow. The zellij session is replaced, the PROCESS is not -- so the
`pair` binary couch forked is still the old one.

**The gesture is the point.** `Alt+n` is already the operator's reflex for
exactly this intent -- "reload pair, keep the conversation". Inside couch it
silently does the weaker thing: it reloads the workbench inside couch's child
and the rebuilt binary is not what comes back. The operator gets no signal that
the chord meant something different here. Making `Alt+n` mean the same thing
one layer up is the whole feature; the operation below it is what makes that
honest.

## Spec

### The gesture: `Alt+n`, taken from the hosted Pair

couch intercepts `Alt+n` and `Ctrl+Alt+n` from the operator's keystrokes before
they reach the Pair child, exactly as it already intercepts `Alt+x` and `Alt+d`
(`cmd/internal/couchtty/keys.go`): a new `seqRelaunch` kind, a `HitRelaunch`,
and encodings pulled from `workbenchshortcut.ChordEncodings` rather than
retyped.

- **Both aliases are required.** `Ctrl+Alt+n` is not a nicety -- on newer macOS
  `Option+n` is a dead-tilde composer, which is why Pair carries the alias at
  all. `ChordAltN` is `\x1b[110;3u`, `ChordCtrlAltN` is `\x1b[110;7u`.
- **`Alt+Shift+N` stays with Pair.** It is `\x1b[78;4u`, a distinct sequence,
  and it restarts only the agent conversation. It is the cheap in-session
  escape hatch that survives couch taking the heavier chord.
- **Kitty-protocol edge, inherited from `ChordAltD`.** Neither `ChordAltN` nor
  `ChordCtrlAltN` declares a legacy encoding, so with the protocol off the
  chord passes through to Pair, which does its old in-place reload. zellij
  pushes the protocol, so this is a documented degradation, not a hazard -- but
  it must be documented at the interception site the way `ChordAltD`'s is,
  because the behaviour differs silently by protocol state.
- **Cost, accepted deliberately.** Inside couch the operator loses Pair's cheap
  in-place workbench reload; every `Alt+n` becomes a full process replacement.
  This is the same trade `Alt+d` already made ("intercepting costs the hosted
  Pair its own chord and buys the durable retirement that makes the thread
  reattachable"), and the same justification applies: un-intercepted, the chord
  does the wrong thing and couch cannot tell.

**Scope follows focus, with one deviation that must be stated.** `Alt+x` and
`Alt+d` mean "what you are looking at": one actor from an actor, every live
thread from the switcher. Relaunch has no whole-couch form -- that is `Alt+d`,
rebuild, re-run `couch`, the symmetry this issue exists to complete. So from
the panel `Alt+n` relaunches the HIGHLIGHTED ROW and leaves the operator in the
switcher; from an actor it relaunches that actor and returns to it. The ending
differs by caller, which is why it belongs to the console rather than to the
operation.

### The operation: park-then-resume, preconditions first

A `relaunch` operation in `couchcore.Operations()`, alongside `park` and
`detach`, confirmed like park. Not a private switcher verb: the precondition
check has to sit with the durable state, not the terminal.

1. **Check the resume's preconditions BEFORE anything is destroyed** -- an
   established native binding with a non-empty root id, a saved launch profile
   with a supported agent, a working path that still resolves. Extracted from
   resume's own eligibility rules so the two cannot drift (ARCH-DRY): relaunch
   asks "would a cold resume of this thread be accepted right now?" and refuses
   if not.
2. **Park.** couch does NOT run `kill-session`/`delete-session` itself -- those
   live in `launcher/` and belong to Pair. Park publishes a nonce-bound quit
   request and writes the `QuitIntentCouch` marker
   (`couchcore/park.go:502,536`); Pair runs its own cleanup -- ledger, markers,
   `kill-session`, quiescence, `delete-session --force` -- and couch observes
   the durable completion, awaits child death, and CASes the incarnation away.
   Reaching around that to kill the session directly is precisely the failure
   mode interception exists to prevent: a dead child and a stale `live`
   incarnation that the fail-closed projector then hides.
3. **Resume**, from the CURRENT binary, with the saved launch profile,
   resuming the agent's native session. Resume, not launch: bringing *a* Pair
   session up would give a fresh conversation, and the native session id is
   what makes this relaunch rather than restart.

**Failure semantics are the design question.** Park is destructive and resume
can refuse; a relaunch that parks and then fails to resume has destroyed a
working session.

- If resume could not run, relaunch refuses and parks NOTHING.
- A resume that fails anyway leaves a verified park, which is recoverable --
  the thread stays in the switcher as `parked` and Enter resumes it. The
  refusal must say that plainly rather than reading as data loss.
- pair#181's warm/cold split matters here: relaunch is deliberately COLD. It
  exists to replace the process, so reattaching is not what is wanted.
- The thread address, its row, and its ledger identity are unchanged across a
  relaunch.

### The holding surface: a status pane that outlives its child

The operator's terminal has to show something for the seconds between park and
resume. Two candidates:

**The switcher, for free.** Park already focuses the panel, the row goes
`parked` then `live`, and `finishOperation`'s existing resume arm
(`console.go:1399`) force-switches back onto the new handle. This works with no
new machinery -- and it is wrong for this action. Relaunch is a repaint of
where the operator already is; bouncing them out to the switcher and back makes
an in-place gesture look like two switches, and it drags the bookkeeping in
§"Consequences" with it.

**A status pane that survives its child** -- the design this issue adopts. The
pane keeps the operator's slot, and renders `relaunching <thread>…` with a
spinner while the child is gone. **A genuinely blank page is indistinguishable
from a hang**, and Pair's boot is not instant, so the surface is a status page
with a live spinner, not an empty tty.

This does not exist today and is the substantial new machinery in this issue:

- A pane is 1:1 with a pty child (`couchtty/console.go:31`), and
  `ptychild.Start` mints a fresh pty per spawn -- there is nothing to hand to a
  second process.
- `onExit` deletes the pane entry unconditionally (`console.go:723-736`). A
  pane that outlives its child is not a state the console can currently be in.

So it needs either ptychild able to spawn onto an existing pty master, or a
pane that can swap its child while keeping its screen, its scrollback, and its
slot in `c.order`/`c.active`. The spinner has a home already: the console's
existing notice spinner is driven off `spinnerC`/`syncSpinner` in `Run`.

### Consequences that are not free

Named here because each is a silent wrong-behaviour rather than a failure, and
because which of them apply depends on the holding-surface choice above:

- **The `previous` slot is clobbered by the bounce.** `onExit` calls
  `tracker.Drop` unconditionally (`console.go:736`), which empties `current`;
  the landing that follows then does `t.previous = t.current` and copies that
  emptiness in. So a park/resume cycle spends the operator's `ctrl+backspace`
  target even though they never left -- contradicting `SwitchTracker`'s own
  doc comment, which claims `previous` survives exactly that cycle. Today's
  un-intercepted `Alt+n` does not cost this, because the Pair process survives
  and no pane exits. A surviving status pane dissolves the problem (no exit, no
  `Drop`, and `Switch` onto the address already current is already a no-op); a
  bounce through the switcher requires fixing it directly.
- **Two hand-written "ends its own child" lists.** `console.go:1375` (the
  `expectedExits` bridge) and `consumeExpectedParkExitLocked`'s switch
  (`console.go:1425-1429`) each enumerate `park`/`detach`, and the two exist
  because the exit/completion race resolves in either order. Any path where
  relaunch's child exit reaches the console must appear in BOTH or the operator
  gets a spurious child-exited notice. `operationNeedsProjectionRefresh`
  (`menu.go:1318`) is the existing shape for such a predicate; a third
  hand-written list is the wrong answer (ARCH-DRY).
- **The dispatch switch in `processInput` is exhaustive on purpose.** Its
  comment says why: a `default` arm would turn any unhandled hit into "open the
  switcher". `HitRelaunch` must be handled in both focus states, not defaulted.

**Out of scope:** restarting the agent conversation itself (that is Pair's
`Alt+Shift+N`), and any notion of a rolling or zero-downtime restart. The
process goes away and comes back.

## Done when

- `Alt+n` and `Ctrl+Alt+n` inside a hosted actor relaunch that actor; the
  operator ends on the same actor, not in the switcher.
- `Alt+Shift+N` still reaches the hosted Pair and restarts only the agent.
- `Alt+n` on the switcher relaunches the highlighted row and leaves the
  operator in the switcher.
- An actor action `relaunch` appears alongside detach and park, confirmed like
  park, and reachable from the same declared-operation surface (no private
  switcher verb).
- Relaunching an actor running an OLD Pair binary yields a session running the
  CURRENT one, with the agent conversation continued rather than restarted --
  verified on the real stack by rebuilding Pair between the two observations.
- Relaunch refuses BEFORE parking when its resume could not succeed, proved by
  a test that makes the binding unresumable and asserts the thread is still
  live afterwards.
- A resume failure after a successful park leaves a resumable parked thread and
  says so, proved by test.
- The thread address, its row, and its ledger identity are unchanged across a
  relaunch.
- The pane shows `relaunching <thread>…` with a live spinner for the whole gap,
  and the operator is never shown a blank screen.
- A relaunch does not change what `ctrl+backspace` returns to, proved by test.
- A relaunch produces no child-exited notice.

## Plan

- [ ] **M1 — the `relaunch` operation.** Domain only; nothing here learns that
      a terminal exists.
  - [ ] Precondition check extracted from resume's own eligibility rules so the
        two cannot drift (ARCH-DRY): "would a cold resume of this thread be
        accepted right now?"
  - [ ] Sequence park -> resume, with the failure semantics above tested at
        BOTH failure points: refusal parks nothing and leaves the thread live;
        a post-park resume failure leaves a resumable parked thread.
  - [ ] Declare it in `couchcore.Operations()` beside park and detach, with the
        confirmation copy naming what it does ("stops and restarts Pair; the
        conversation continues").
  - [ ] Assert thread address / row / ledger identity are unchanged across the
        composition.
- [ ] **M2 — the gesture and the status pane.** Console surface + real-stack
      proof.
  - [ ] `seqRelaunch`/`HitRelaunch` in `couchtty/keys.go`, encodings via
        `ChordEncodings` for `ChordAltN` and `ChordCtrlAltN`, with the
        Kitty-protocol degradation documented at the site as `ChordAltD`'s is.
        Regression that `ChordAltShiftN` is NOT intercepted.
  - [ ] Handle `HitRelaunch` in both focus states of `processInput`'s
        exhaustive switch: actor -> relaunch that actor and return to it; panel
        -> relaunch the highlighted row and stay.
  - [ ] Pane outlives its child: either a pty master reused across the spawn or
        a pane that swaps children, keeping screen, scrollback, and its slot in
        `c.order`/`c.active`.
  - [ ] Status render with the existing notice spinner: `relaunching <thread>…`
        for the whole gap. Test that no frame in the gap is blank.
  - [ ] Correctness sweep on whichever paths can still see the child exit:
        `previous` unchanged across a relaunch, and no spurious child-exited
        notice (a predicate beside `operationNeedsProjectionRefresh`, not a
        third hand-written list).
  - [ ] Real-stack verification: rebuild Pair between two observations of the
        same actor, and confirm the conversation continued.
  - [ ] Atlas: relaunch beside detach/park, why its preconditions run first,
        and the pane-outlives-child lifetime.

## Log

### 2026-09-03

- Filed from the couch-lite usability work (pair#170, pair#181). The operator's
  framing: "with alt+d we can quit couch and restart couch; with relaunch we can
  relaunch pair code, but keep working state." Part of making couch-lite usable,
  deliberately kept out of pair#181 rather than appended to it.

### 2026-09-04

- Design session in brain (advisor session; no code touched). Started from the
  operator's question -- "in pair `alt+n` restarts pair loading the previous
  agent session; in couch `alt+n` should do the same, so can we just perform
  the switcher's relaunch action while in the pair pane?" -- and walked the
  couch console to find out whether that composes.
- It does, and most of the surface already exists: interception has a shape
  (`keys.go`), the dispatch is `onDetachHotkey`'s body verbatim
  (`console.go:1225`), and `finishOperation` already knows how to end an
  operation by landing on a started handle (`console.go:1399`, the resume arm).
- Four things it does NOT cover, now folded into the Spec: the ending differs
  by caller (switcher row vs. in-pane), two hand-written child-exit lists need
  the new operation, the `previous` slot is silently spent by the bounce, and
  the panel scope needs a rule because the dispatch switch is exhaustive.
- The operator then proposed the mechanism a layer down -- "blank page to hold
  the tty, kill/delete the session, bring pair up at the blank page". Two
  corrections: `kill-session`/`delete-session` are Pair's, not couch's (park
  publishes a quit request and verifies; `park.go:502,536`), and the thing
  being replaced is the Pair PROCESS couch forked, not the zellij session. The
  blank page itself is the good idea, and is now the adopted design -- it
  dissolves the `previous`-slot and child-exit bookkeeping instead of fixing
  them, at the cost of a pane lifetime the console does not currently have.
- Operator's call on the holding surface: a status page **with a spinner**, not
  a blank one -- a blank screen during a non-instant boot is indistinguishable
  from a hang.

## Revisions

### 2026-09-04 — rescoped from "an action in the switcher" to "the `Alt+n` gesture"

**Reason.** The issue as filed specified the relaunch OPERATION and left the
operator's entry point as an action on a switcher row. That is not where the
intent lives: `Alt+n` is already the reflex for "reload pair, keep the
conversation", and inside couch it silently does the weaker in-process thing.
An operation reachable only from the switcher would have shipped the capability
without fixing the gesture that means it.

**Delta.**

- Added the whole `### The gesture` section: `Alt+n` + `Ctrl+Alt+n`
  interception in `couchtty`, the `Alt+Shift+N` carve-out, the Kitty-protocol
  degradation inherited from `ChordAltD`, and the accepted cost of taking the
  chord from the hosted Pair.
- Added a panel scoping rule and its justification -- relaunch has no
  whole-couch form, so it does not follow `Alt+x`/`Alt+d`'s "what you are
  looking at" rule, and the deviation is stated rather than discovered.
- Added `### The holding surface`: a status pane that outlives its child, with
  a spinner. Adopted over the free switcher-bounce, which was the implicit
  design before and is now recorded as the rejected alternative with its
  reason.
- Added `### Consequences that are not free`: the `previous`-slot clobber, the
  two hand-written child-exit lists, and the exhaustive dispatch switch.
- Park's step in the Spec now says what park actually does (publish a quit
  request, let Pair do its own `kill-session`/`delete-session`, verify) rather
  than "tear the zellij session down", which read as couch's own call.
- Plan split into M1 (operation, domain-only) and M2 (gesture + status pane +
  real-stack proof). It was a flat single-pass list.
- `## Done when` grew the gesture, spinner, `previous`-slot and no-spurious-
  notice criteria.

**Unchanged.** The failure semantics -- preconditions before the park, refusal
parks nothing, a post-park resume failure leaves a recoverable parked thread --
which were the original filing's core and survived the review intact.
