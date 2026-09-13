---
id: 000207
status: open
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
---

# couch's asserted mouse mode has no release path short of restarting couch

## Problem

Operator report, 2026-09-06: copy-on-select highlighting in the **agent pane**
stopped working. Three recovery gestures were tried, in order:

| gesture | result |
|---|---|
| `alt+n` → switch to another actor and back | **did not fix** |
| `alt+n` → **relaunch** the actor (`#182`: restart pair in place, keeping the agent conversation) | **did not fix** |
| exit couch, start couch again | **fixed** |

That ordering is the finding. A relaunch mints a **new child** — new pty, new
`ptychild.Screen`, so couch's *belief* about the child returns to `unknown`,
where `#196`'s rule says stand back. It did not help. Only killing couch itself
did.

**So the stale thing is not the belief; it is the assertion.** The likely shape:
couch writes its own mouse mode to the HOST terminal once it believes the child
holds none, and **nothing ever writes the release**. Re-observing the child
cannot undo a sequence already sent to the host. Restarting couch works because
process exit tears down the host's terminal state, not because any code path
reconsidered anything.

If that is right, the four rounds `#172` spent on the *observation* side
(BR-16/22/26/33) were all upstream of this: they made couch decide correctly
*when* to assert, and left the assertion permanent once made.

### How it was triggered, reproducibly

Not a mystery — it was caused on purpose, by accident. A throwaway probe ran in
the right pane with mouse reporting on and, on exit, "restored" the prior state
by writing `?1000l` + `?1006l`. Those bytes travel up through zellij into
couch's child stream, where `Screen` scans DECSETs. couch observed a well-formed
"the child turned mouse OFF", moved to *observed-none*, and asserted.

Nothing malfunctioned. The tri-state answered the question it was asked, and the
question is the problem: **`Screen` reports what the byte stream SAID, not what
the child INTENDS.** A transient in-pane process that tidies up after itself is
indistinguishable from the long-lived child changing its mind.

That gives this class a reproducer for the first time, where the four historical
rounds each had only a symptom.

## Spec

**An asserted mouse mode must be releasable without killing couch.** Two halves,
and the first is the one that matters:

1. **A release path.** When couch stops believing it should own the mouse — or
   when the child it asserted over is gone — it writes the release, restoring
   whatever the child asks for. Today there is no such write.
2. **A recovery gesture that reaches it.** `alt+n` (switch OR relaunch) must
   restore the operator's mouse. That is the acceptance test, and today both
   fail.

### To settle in the plan

1. **Confirm the diagnosis before designing.** The reasoning above is inference
   from three observations, not a read of the code. Verify that couch asserts to
   the host and never releases, rather than assuming it.
2. **Whether a relaunch should reset host-level mode state at all.** `#182`
   deliberately preserves the conversation across a relaunch; whether terminal
   modes are part of "the session" or part of "the host" is the design question,
   and it decides whether the fix belongs on the relaunch path or the assert
   path.
3. **Whether an observation should decay.** A mode change seen once and never
   reaffirmed is weaker evidence than one the child re-asserts on every repaint.
   Nothing today returns the belief to `unknown` short of a new child — and per
   the table above, even a new child was not enough.

## Done when

- The probe reproducer breaks copy-on-select, and `alt+n` restores it.
- The same holds for the relaunch path (`#182`), not only the switch path.
- Killing couch is no longer the only way out of a wrong mouse state.
- A regression test drives the reproducer — a DECSET-emitting transient in the
  child stream — rather than asserting on a hand-built belief.

## Plan

- [ ] Confirm the diagnosis: does couch assert to the host with no release?
- [ ] Decide where the release belongs (assert path vs relaunch path).
- [ ] Implement, with the probe reproducer as the test.
- [ ] Verify both `alt+n` paths recover, on the real workbench.

## Log

### 2026-09-06

Split out of `#200` (punted) at the operator's request, so the observation is
findable when this recurs rather than buried in a deprioritized issue. `#200`
remains the broader consolidation — one arbitration, two consumers, deleting
`termcmd`'s second `appMouseMode()` implementation; this issue is the specific
defect that has bitten in production and currently has no in-session remedy.

Operator's framing: *"ideally switching between tab would fix it. actually I also
tried alt+n reload pair, that didn't work neither. need to restart couch."*

### 2026-09-12 — diagnosis confirmed; the reattach clobber is a second trigger

Operator report today: after restarting couch, drag-selection shows no live
highlight in EITHER the right-pane shell or nvim — the selection appears only
on mouse release. Press+release works, motion does not. That is the host
terminal holding `?1000` (click tracking) without `?1002` (button-event /
motion tracking).

Confirmed the diagnosis #207 plan step 1 asked for, plus a new fact:

- **zellij does not relay a pane app's mouse DECSET/DECRST to its host.**
  Measured with a probe driving a real zellij under a pty: at client startup
  zellij asserts `?1000h ?1002h ?1003h ?1006h` to the host once, and a pane
  app writing `?1002h` / `?1002l` afterward produces NOTHING on the host.
  So the workbench's host mouse mode is owned by whoever writes the real
  terminal directly — couch — not by the pane app.
- **couch asserts clicks-only unconditionally at startup** (`console.go:564`,
  `EnableMouseClicks` = `?1000;1006h`), before any child is observed. The
  very next call is `paintNow()`, whose assert IS gated on
  `couchMayOwnTheMouse()` (`:1224`). So :564 is both redundant with the
  gated paint and actively harmful.
- **Cold start hides it:** couch writes `?1000` to the host, then the fresh
  zellij emits its startup `?1002h` on the live stream, couch forwards it to
  the host, and the host ends at `?1002`. Motion works.
- **Reattach (pair#206) is fatal:** couch restarts over a still-running
  zellij. New couch writes `?1000` to the host (:564). The reattached zellij
  does NOT re-emit `?1002h` — it is alive and emitted it once, long ago —
  and couch's fresh `Screen` for it reads `MouseObserved()==false`, so the
  gated paint stands back (the `#196`/`#206` unknown-case rule). Nothing
  re-raises `?1002`. The host stays at `?1000`: clicks only, no motion. The
  whole workbench loses drag-highlight until couch is restarted onto a cold
  zellij.

This is the same root cause #207 names (couch asserts a host mouse mode with
no path back to the child's actual mode), reached by a different trigger than
the original probe-writes-`?1000l` reproducer. The nvim and shell symptoms
are one bug: both are panes inside the one zellij whose host mode couch
demoted.

**Candidate minimal fix (needs the design call in "To settle" below):** drop
the unconditional `:564` assert and let the gated paint at `:1224` be the only
writer. On cold start the gated paint stands back in the unknown case and
zellij's forwarded `?1002h` sets the host; on reattach the host retains the
pre-restart `?1002` because couch no longer clobbers it; an empty couch (no
child) still asserts clicks because `couchMayOwnTheMouse()` returns true for a
nil pane. The risk is a real couch change on the "must not degrade pair" path,
so it wants a couch reattach test (a fresh `Screen` for a live child that held
`?1002`) proving the host is left in `?1002`, not a keystroke smoke.

This resolves plan step 1 ("confirm the diagnosis") and step 2's fork leans
toward the assert path, not the relaunch path: the relaunch/reattach is only
the trigger; the defect is the unconditional assert.

### 2026-09-12 — correction: the demotion is mid-session, not the startup assert

The operator caught the hole: if a couch reattach FIXES the bad state, then
couch startup cannot be what creates it — a reattach re-runs startup and would
reproduce it. So the previous entry's "candidate minimal fix: drop the :564
startup assert" is WRONG. Withdrawn.

Corrected mechanism, from `paintNow` (`console.go:1224`) + `takeOverScreen`
(`:1131`):

- couch keeps its hands off the host mouse mode while it believes the child
  (the `pair`/zellij actor) holds tracking: `couchMayOwnTheMouse()` =
  `MouseObserved() && !Mouse()`. While zellij holds `?1002`, `Mouse()` is
  true, so couch never asserts, and the host sits at `?1002` (zellij's own
  DECSET, teed to the host). Motion works.
- **Mid-session**, `hostScan` observes a mouse-OFF in the actor stream:
  `Mouse()` flips to false with `MouseObserved()` still true, so
  `couchMayOwnTheMouse()` becomes true and the NEXT `paintNow()` (every
  status-row redraw) writes `EnableMouseClicks` = `?1000;1006h` to the host.
  `?1000` REPLACES `?1002` (the modes are one mutually-exclusive slot, as the
  :1213 comment already notes), so the host drops to click-only: press and
  release, no motion. It sticks because every subsequent paint re-writes it.
- **A reattach fixes it** because `takeOverScreen` replays the child's current
  mode-bearing bytes (and/or a fresh zellij client re-emits `?1002h` on
  attach), which raises `?1002` on the host again and flips `Mouse()` back to
  true so couch stands down.

So the defect is real and is #207's, but the trigger is a mid-session
observation of mouse-OFF, and the standing bug is that couch SEIZES clicks-only
and holds it with no path back until a fresh attach. What is NOT yet caught:
**what emits the mouse-OFF that couch's `hostScan` sees.** Candidates: a pane
app that enables then disables mouse on exit (the original #207 reproducer),
zellij changing its own host mode on a focus/pane-state change, or — worth
ruling out explicitly — `pair term`'s own `?1002l` reconcile from #240 if it
reaches couch's stream (my zellij-relay probe says pane modes do NOT propagate
up, so this should be ruled out, not assumed). Next step is instrumentation:
log every mouse DECSET/DECRST couch writes to the host AND every one its
`hostScan` observes, capture one real occurrence, and fix the exact trigger —
not another inferred mechanism.
