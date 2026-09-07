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
