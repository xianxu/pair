---
id: 000182
status: working
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-04
estimate_hours: 3.58
started: 2026-09-04T09:16:38-07:00
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
workbench in place") is close but not the same thing: it restarts the workbench
*inside* the session couch's child already handed off to. The process couch
spawned is not replaced, so a rebuilt Pair binary is not what comes back.

Concretely: `launcher/restart.go` writes a restart marker, touches the quit
marker and execs `kill-session`; the already-running Pair process's outer loop
then regains control and re-enters its create flow. The zellij session is
replaced, the PROCESS is not.

**The gesture is the point.** `Alt+n` is already the operator's reflex for
exactly this intent -- "reload pair, keep the conversation". Inside couch it
silently does the weaker thing, and the operator gets no signal that the chord
means something different here. Making `Alt+n` mean the same thing one layer up
is the feature; the operation below it is what makes that honest.

## Spec

A **relaunch** action on an actor, alongside detach and park:

1. Park the thread -- tear the zellij session down (`kill-session`,
   `delete-session`) and stop the Pair instance, exactly as park does today,
   producing a verified park.
2. Resume it immediately from that verified park: a fresh Pair process, from
   the CURRENT binary, with the saved launch profile, resuming the agent's
   native session so the conversation continues where it was.

Net effect: new Pair code, same conversation, same thread address, same row in
the switcher.

**It is park-then-resume as one action**, and both halves already exist as
declared operations. What this adds is the composition, its failure semantics,
and one operator gesture for it.

**Failure semantics are the whole design question.** Park is destructive and
resume can refuse -- and a relaunch that parks successfully then fails to
resume has destroyed a working session and left the operator with a cold
thread. That is strictly worse than not offering the action. So:

- The resume's preconditions are checked BEFORE the park, not after: an
  established native binding with a non-empty root id, a saved launch profile
  with a supported agent, and a working path that still resolves. If resume
  could not run, relaunch refuses and parks nothing.
- A resume that fails anyway leaves a verified park, which is recoverable --
  the thread stays in the switcher as `parked` and Enter resumes it. The
  refusal must say that plainly rather than reading as data loss.
- pair#181's warm/cold split matters here: relaunch is deliberately COLD. It
  exists to replace the process, so reattaching is not what is wanted.

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
  because the behaviour then differs silently by protocol state.
- **Cost, accepted deliberately.** Inside couch the operator loses Pair's cheap
  in-place workbench reload; every `Alt+n` becomes a full process replacement.
  Same trade `Alt+d` already made ("intercepting costs the hosted Pair its own
  chord and buys the durable retirement that makes the thread reattachable"),
  and the same justification: un-intercepted, the chord does the wrong thing
  and couch cannot tell.

**Scope follows focus, with one deviation that must be stated.** `Alt+x` and
`Alt+d` mean "what you are looking at": one actor from an actor, every live
thread from the switcher. Relaunch has no whole-couch form -- that is `Alt+d`,
rebuild, re-run `couch`, the symmetry this issue exists to complete. So from
the panel `Alt+n` relaunches the HIGHLIGHTED ROW and leaves the operator in the
switcher; from an actor it relaunches that actor and returns to it. The ending
differs by caller, which is why it belongs to the console rather than to the
operation.

### The holding surface: a status pane that outlives its child

The operator's terminal has to show something for the seconds between park and
resume. Two candidates:

**The switcher, for free.** Park already focuses the panel, the row goes
`parked` then `live`, and `finishOperation`'s existing resume arm
(`console.go:1399`) force-switches back onto the new handle. This works with no
new machinery -- and it is wrong for this action. Relaunch is a repaint of
where the operator already is; bouncing them out to the switcher and back makes
an in-place gesture look like two switches, and it drags the bookkeeping below
along with it.

**A status pane that survives its child** -- the design this issue adopts. The
pane keeps the operator's slot and renders `relaunching <thread>…` with a
spinner while the child is gone. **A genuinely blank page is indistinguishable
from a hang**, and Pair's boot is not instant, so the surface is a status page
with a live spinner, not an empty tty.

This does not exist today and is the substantial new machinery:

- A pane is 1:1 with a pty child (`couchtty/console.go:31`), and
  `ptychild.Start` mints a fresh pty per spawn -- there is nothing to hand to a
  second process.
- `onExit` deletes the pane entry unconditionally (`console.go:723-736`). A
  pane that outlives its child is not a state the console can currently be in.

So it needs either ptychild able to spawn onto an existing pty master, or a
pane that can swap its child while keeping its screen, its scrollback, and its
slot in `c.order`/`c.active`. The spinner has a home already: the console's
notice spinner driven off `spinnerC`/`syncSpinner` in `Run`.

### Consequences that are not free

Each is a silent wrong-behaviour rather than a failure, and which apply depends
on the holding-surface choice above:

- **The `previous` slot is clobbered by the bounce.** `onExit` calls
  `tracker.Drop` unconditionally (`console.go:736`), which empties `current`;
  the landing that follows then does `t.previous = t.current` and copies that
  emptiness in. So a park/resume cycle spends the operator's `ctrl+backspace`
  target even though they never left -- contradicting `SwitchTracker`'s own doc
  comment, which claims `previous` survives exactly that cycle. Today's
  un-intercepted `Alt+n` does not cost this, because the Pair process survives
  and no pane exits. A surviving status pane dissolves the problem (no exit, no
  `Drop`, and `Switch` onto the address already current is already a no-op); a
  bounce through the switcher requires fixing it directly.
- **Two hand-written "ends its own child" lists.** `console.go:1375` (the
  `expectedExits` bridge) and `consumeExpectedParkExitLocked`'s switch
  (`console.go:1425-1429`) each enumerate `park`/`detach`, and both exist
  because the exit/completion race resolves in either order. Any path where
  relaunch's child exit reaches the console must appear in BOTH or the operator
  gets a spurious child-exited notice. `operationNeedsProjectionRefresh`
  (`menu.go:1318`) is the existing shape for such a predicate; a third
  hand-written list is the wrong answer (ARCH-DRY).
- **`processInput`'s dispatch switch is exhaustive on purpose.** Its comment
  says why: a `default` arm would turn any unhandled hit into "open the
  switcher". `HitRelaunch` must be handled in both focus states, not defaulted.

### One clarification on step 1 above

couch does not run `kill-session`/`delete-session` itself -- those live in
`launcher/` and are Pair's. Park publishes a nonce-bound quit request and writes
the `QuitIntentCouch` marker (`couchcore/park.go:502,536`); Pair runs its own
cleanup and couch observes the durable completion, awaits child death, and
CASes the incarnation away. The plan doc has this right; the Spec's step 1
wording above is loose and should be read through this.

**Out of scope:** restarting the agent conversation itself (that is Pair's
`Alt+Shift+N`), and any notion of a rolling or zero-downtime restart. The
session goes away and comes back.

## Done when

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
- `Alt+n` and `Ctrl+Alt+n` inside a hosted actor relaunch that actor; the
  operator ends on the same actor, not in the switcher.
- `Alt+Shift+N` still reaches the hosted Pair and restarts only the agent.
- `Alt+n` on the switcher relaunches the highlighted row and leaves the
  operator in the switcher.
- The pane shows `relaunching <thread>…` with a live spinner for the whole gap,
  and the operator is never shown a blank screen.
- A relaunch does not change what `ctrl+backspace` returns to, proved by test.
- A relaunch produces no child-exited notice.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                 design=0.90 impl=0.08
item: smaller-go-module          design=0.06 impl=0.20
item: greenfield-go-module       design=0.20 impl=0.28
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.14
item: tui-screen                 design=0.16 impl=0.20
item: real-api-discovery         design=0.00 impl=0.18
item: atlas-docs                 design=0.03 impl=0.08
item: milestone-review           design=0.00 impl=0.20
design-buffer: 0.15
total: 3.58
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

| Slug | Instances |
| --- | --- |
| `issue-spec` | this issue + plan authoring, two plan-quality rounds spent |
| `smaller-go-module` | the `CheckResumePreconditions` extraction; the refuse-before-park test; the park-failure branch; the success test |
| `greenfield-go-module` | `Couch.Relaunch` + `RelaunchResult`/`RelaunchOutcome` |
| `tui-screen` | the switcher action, its confirmation, and the two-phase progress notice |
| `real-api-discovery` | the rebuilt-binary verification on the real stack |
| `atlas-docs` | atlas + README |
| `milestone-review` | one boundary |

**Where judgment entered:**

- **`issue-spec` at full design (×1.0), no spec-quality discount**, on pair#170's
  precedent: the spec-authoring primitive cannot be pre-resolved by its own
  output, and the hard question here — the failure model — was open until the
  plan-quality gate closed it. Two rounds are already spent, one of them on a
  Critical (the park-failure branch was missing entirely).
- **Every other design line ×0.2.** The plan fixes files, signatures and test
  strategies per task, so the remaining design cost is reading.
- **One `real-api-discovery` line, not zero.** Verifying that the agent
  conversation *continued* rather than restarted needs a rebuilt binary between
  two observations on the live stack, and pair#181 M2 showed that kind of check
  finds things a fake cannot (the ledger row count is what proved reattach).
- **No `scope-pivot` line, unlike pair#170 and pair#181.** Both halves already
  exist as tested operations, and the composition's semantics were settled by
  the gate rather than left to be discovered. This is the "shape known" column
  of pair#170's own ratio table.
- **Design buffer +15%** (v2.1 Step 6): thorough plan doc, and +30% on top of a
  ×0.2 discount double-counts the same thoroughness.
- **`impl=` values are already v3.1-scaled** to 40% of the v2/v2.1 table.

**Known risks:**

- **The park-failure branch is the least testable part.** It needs a fake that
  fails at specific park exits, and `pairlifecycletest` supports that — but if
  this estimate misses, it is because reaching three distinct exits took longer
  than one `smaller-go-module` line.
- **Direction: more likely high than low.** Unlike pair#181, nothing here is a
  measurement-driven redesign; the work is composition over two surfaces the
  session has spent all day in.


## Plan

Design landed at `workshop/plans/000182-relaunch-an-actor-plan.md`. One review
boundary — the work is one action composed from two existing operations, so
splitting it would force a redundant milestone-close (AGENTS.md §3).

- [ ] M1 — relaunch as one declared operation: the resume preconditions split
      from the occupancy rule they share with resume, `Couch.Relaunch` checking
      those preconditions BEFORE the park, a recoverable-and-stated outcome when
      the park succeeds and the resume does not, the switcher action on live
      rows only, and real-stack verification with a rebuilt binary.
- [ ] M2 — the gesture and the status pane (added 2026-09-04, see `## Revisions`;
      **not covered by the estimate above**, and the plan doc predates it):
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
  - [ ] Status render on the existing notice spinner: `relaunching <thread>…`
        for the whole gap. Test that no frame in the gap is blank.
  - [ ] Correctness sweep on whichever paths can still see the child exit:
        `previous` unchanged across a relaunch, and no spurious child-exited
        notice (a predicate beside `operationNeedsProjectionRefresh`, not a
        third hand-written list).

Superseded first draft, kept for its reasoning:

- [-] Decide where the composition lives: a `relaunch` operation in
      `couchcore.Operations()` that sequences park and resume, versus the
      console driving two existing operations. Prefer the operation -- the
      switcher must not grow a private verb, and the precondition check has to
      sit with the durable state, not the terminal.
- [ ] Precondition check extracted from resume's own eligibility rules so the
      two cannot drift (ARCH-DRY): relaunch asks "would a cold resume of this
      thread be accepted right now?" and refuses if not.
- [ ] Sequence park -> resume with the failure semantics above, tested at both
      failure points.
- [ ] Switcher action + confirmation naming what it does ("stops and restarts
      Pair; the conversation continues").
- [ ] Real-stack verification with a rebuilt binary between observations.
- [ ] Atlas: relaunch beside detach/park, and why its preconditions run first.

## Log

### 2026-09-03

- Filed from the couch-lite usability work (pair#170, pair#181). The operator's
  framing: "with alt+d we can quit couch and restart couch; with relaunch we can
  relaunch pair code, but keep working state." Part of making couch-lite usable,
  deliberately kept out of pair#181 rather than appended to it.

### 2026-09-04 — advisor session (brain), no code touched

- Operator's question: "in pair `alt+n` restarts pair loading the previous agent
  session; in couch `alt+n` should do the same, so can we just perform the
  switcher's relaunch action while in the pair pane?" Walked the couch console
  to find out whether that composes.
- It does, and most of the surface exists: interception has a shape
  (`keys.go`), the dispatch is `onDetachHotkey`'s body verbatim
  (`console.go:1225`), and `finishOperation` already knows how to end an
  operation by landing on a started handle (`console.go:1399`, the resume arm).
- Four things that does NOT cover, now in the Spec: the ending differs by
  caller (switcher row vs. in-pane), two hand-written child-exit lists need the
  new operation, the `previous` slot is silently spent by the bounce, and the
  panel scope needs a rule because the dispatch switch is exhaustive.
- Operator then proposed the mechanism a layer down -- "blank page to hold the
  tty, kill/delete the session, bring pair up at the blank page". Two
  corrections: `kill-session`/`delete-session` are Pair's, not couch's, and the
  thing being replaced is the Pair PROCESS couch forked, not the zellij
  session. The blank page itself is the good idea and is now the adopted
  design: it dissolves the `previous`-slot and child-exit bookkeeping instead
  of fixing them, at the cost of a pane lifetime the console does not have.
- Operator's call on the holding surface: a status page **with a spinner**, not
  a blank one -- a blank screen during a non-instant boot is indistinguishable
  from a hang.

## Revisions

### 2026-09-04 — scope grew from "an action in the switcher" to "the `Alt+n` gesture"

**Reason.** The issue specified the relaunch OPERATION and left the operator's
entry point as an action on a switcher row. That is not where the intent lives:
`Alt+n` is already the reflex for "reload pair, keep the conversation", and
inside couch it silently does the weaker in-process thing. An operation
reachable only from the switcher ships the capability without fixing the
gesture that means it.

**Delta.** Additive — nothing in the pre-existing Problem, Spec, Estimate or M1
was rewritten:

- `## Problem` gains the concrete `launcher/restart.go` mechanism and a
  paragraph on why the gesture is the point.
- `## Spec` gains `### The gesture`, `### The holding surface`,
  `### Consequences that are not free`, and a clarification that step 1's
  `kill-session`/`delete-session` are Pair's calls, not couch's.
- `## Done when` gains six criteria: both chord aliases, the `Alt+Shift+N`
  carve-out, the panel form, the spinner, the `previous` slot, and the absence
  of a spurious child-exited notice.
- `## Plan` gains M2. M1 is untouched.

**⚠ The estimate is now stale.** `estimate_hours: 3.58` was derived at
`change-code` against the switcher-action-only scope, and its `tui-screen` line
covers "the switcher action, its confirmation, and the two-phase progress
notice" — not chord interception, not a pane that outlives its child. The
estimate and `workshop/plans/000182-relaunch-an-actor-plan.md` both need a pass
before M2 starts. Deliberately NOT re-derived here: the plan cleared
plan-quality on the old scope, and re-costing without re-planning would produce
a number with nothing behind it.

**Provenance.** Authored from a brain advisor session, not from the
implementation session that owns this branch. The two collided: this file was
rewritten from its pre-`change-code` state and swept into `335c027b`, dropping
`status: working`, `estimate_hours`, `started`, the whole `## Estimate` block
and the M1 plan framing. Repaired by rebuilding from `adeb90d3` and re-applying
the design additively. If the implementation session has newer local state,
that state wins over this file.
