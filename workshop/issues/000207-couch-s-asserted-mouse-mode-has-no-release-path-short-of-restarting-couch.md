---
id: 000207
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-12
estimate_hours: 0.773
started: 2026-09-12T23:21:00-07:00
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

- [x] M1 — trace active thread, replay/reset, and actual assertion write outcomes.
- [ ] M2 — capture recurrence, approve causal recovery design, implement and verify recovery.

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

### 2026-09-12 — instrumentation landed (step 1)

Rather than change couch's mouse logic on an uncaught trigger, added a
log-only probe, `COUCH_MOUSE_TRACE=<path>`, mirroring the existing
`COUCH_INPUT_TRACE`/`COUCH_TRACE` file tracers (`mousetrace.go`,
`SetMouseTrace`, wired in `couchcmd/run.go`). It records the two events that
decide the host's mouse mode:

- `child-mode <old> -> <new>` in `writeChild`, when the child's teed stream
  changes the host's mouse modes (this is where a mode-OFF from below couch
  shows up);
- `assert-clicks host-before=<modes> child-mouse=<bool> child-observed=<bool>`
  in `paintNow`, when couch asserts its own clicks-only mode.

Read together: a `child-mode 1002,1006 -> none` immediately before
`assert-clicks host-before=none` means the layer below couch dropped motion
and couch correctly floored; an `assert-clicks host-before=1002,1006` means
couch clobbered a live motion mode (the downgrade the floor rule targets).

No behavioural change. The fix follows once a live trace names the trigger,
including the #240 theory (does a right-pane tab switch to a shell cause a
`child-mode ... -> none` in couch's trace?).


### 2026-09-14 — Live recurrence captured; replay and assertion-result trace gaps
- 2026-09-14: closed M1 — go test ./cmd/internal/couchtty ./cmd/internal/couchcmd passed after generated runtime assets; go test -race ./cmd/internal/couchtty -run MouseTrace\|MouseWriteResult -count=1 passed; producer red-green identity/replay/lifecycle/write-outcome checks; git diff --check clean. Actual 1.02h is output of sdlc actual --issue 207 --brain-dir /Users/xianxu/workspace/brain; close auto-measurement still ignores that root. Cumulative historical measurement is not directly comparable with M1-only estimate. Recovery remains M2.; review verdict: SHIP

Operator reports drag selection no longer highlights during motion, alongside
Ctrl+Return becoming plain Return (#251). Ctrl+Space then Return still reaches
a yellow notification. No common triggering event has been established.

Read-only inspection confirmed running Couch PID 5316 (started September 13,
16:14:04 America/Los_Angeles) still holds
`/private/tmp/couch-mouse-207.log` open. The complete log is only 20 lines;
last observed modification was September 14 at 08:57:05. Local-time decoding
shows `assert-clicks host-before=none child-mouse=false child-observed=true`
bursts at 08:48:13.965–08:48:14.267 and 08:57:04.436–08:57:05.142. The earlier
September 13 21:02:37.086–.087 assertions coincide with #249's failed
continuation restart. This is timing correlation, not proof of a shared cause.

There is NO recorded live `child-mode ... -> none` immediately preceding these
bursts. The last recorded live transition is September 13 16:14:07.266,
`none -> 1003,1006`. This does not establish that tracking persisted until the
later assertions: `takeOverScreen` resets and feeds hostScan from replay without
logging those mode transitions. Background child output updates the child's
Screen without flowing through `writeChild`, so its mode changes are also
absent from this trace until selection/replay.

Two further limits matter before claiming the exact trigger:

- `host-before` is the hostScan belief, not a query of the terminal; Couch's own
  assertions are not fed into that scanner. Repeated `none` therefore does not
  mean each prior assertion had no effect.
- The `assert-clicks` event is logged after `writeOwn` returns even when the
  paint/framing gate deferred the write. It records an assertion attempt, not
  confirmed emission. Neither event carries the active thread address.

The observed attempts fit the loss of motion tracking, but current traces cannot
prove which takeover/reset/child changed the state or whether each attempted
assertion reached the terminal. The next diagnostic change should cover replay,
active thread identity and emitted-versus-deferred assertions, preserving the
current live evidence before any restart. Keyboard-mode loss shares the same
host terminal/replay boundary but is still a separate unproven correlation.
No runtime state or production code changed during this inspection.

## Revisions

### 2026-09-14 — approved diagnostic boundary before recovery design

The operator approved logging improvements after the live recurrence report.
The original recovery Spec and Done when remain unresolved; this iteration is
M1 diagnostics only, followed by M2 causal fix after evidence and design approval.
See `workshop/plans/000207-mouse-diagnostics-plan.md`.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. This estimates the approved M1 diagnostic
boundary only; M2 is unapproved and must be estimated after its causal design.
Two smaller-go-module primitives cover production diagnostic integration and
producer regression coverage. Each takes 0.3 design before the thorough-spec
0.2 discount, and 0.5 implementation before the v3.1 0.4 multiplier. Existing
traceFile/host fake/scanner remove any novel-stack/library need. Atlas takes
0.1 design x0.2 and 0.1 impl x0.4; one review takes 0.08 design and 0.2 impl x0.4.
Familiarity 1.0, design buffer 15%; 0.22*1.15 + 0.52 = 0.773 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.08 impl=0.08
design-buffer: 0.15
total: 0.773
```

### 2026-09-14 — M1 diagnostic implementation

Plan-quality round 2 accepted both findings after named producer/formatter test
strategies and diagnostic growth limits were recorded. Estimate-quality INFO
notes the tight test allowance; the 0.773h estimate applies only to M1 and must
not be compared with all historical #207 time. The review allocation retains
uncertainty for producer attribution findings, not additional feature design.

Focused baseline passed. New producer tests failed for missing identity,
replay/reset, lifecycle events and emitted/deferred/error fields, then passed
with the diagnostic hooks. Existing terminal byte strings and ownership policy
are preserved. `writeOwn` now returns its observed byte result, ignored by
non-diagnostic callers. Full verification and M1 review follow before handoff.

Focused diagnostics and race diagnostics passed. Full couchtty passed; couchcmd
initially failed because the isolated worktree lacked generated embedded runtime
files. Generated them with `go run ./cmd/internal/runtimebundle/generatecmd -repo .
-out cmd/internal/runtimebundle/assets/runtime`; these remain ignored artifacts.

### 2026-09-14 — M1 boundary accepted

Mandatory `sdlc milestone-close` review returned SHIP: no Critical/Important
findings. Resolved the Minor plan-table classification omission in an appended
consolidated PURE/INTEGRATION table. Reviewer independently reran both package
and focused race checks successfully. Original recovery acceptance stays M2.

The worktree's default `../brain` gave a telemetry-unavailable refusal.
`sdlc actual --issue 207 --brain-dir /Users/xianxu/workspace/brain` produced
1.02h cumulatively; passing that exact measured value let milestone-close
continue because its automatic measurement did not honor the corrected root.
This includes historical issue work and may exclude this API subagent segment;
it must not be interpreted as a measurement of M1 alone or compared to its
M1-only estimate. No judgment hours were invented.
