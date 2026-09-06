---
id: 000196
status: open
deps: ["#172"]
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
---

# agent-pane drag selection loses live highlight after couch reattach

## Problem

Mouse-drag selection in the AGENT pane gives no live highlight feedback while
the button is held. The selected region only highlights on mouse-UP.

- Standalone `pair`: correct — the highlight tracks the pointer during the drag.
- Under `couch`, **after some reattachment**: broken as described. Not every
  reattach; it starts happening after some of them.

Selection still *works* (the text is selected, and `copy-on-select` still fires
on mouse-up) — what is lost is the during-drag feedback.

### Hypothesis: couch demotes the child's mouse tracking mode to `?1000`

The tree already documents this exact failure mode; the reattach path appears to
be an unguarded route into it.

1. `cmd/internal/hostty/control.go:58` — couch's own re-assert is
   `EnableMouseClicks = "\x1b[?1000;1006h"`, deliberately click-only:
   "Not ?1002/?1003: motion reports arrive at pointer-movement rates."
2. `cmd/internal/couchtty/console.go:1026-1046` — `paintNow` re-asserts that on
   **every paint**, gated on `!c.childWantsMouse()`. The comment states the
   hazard outright (BR-22): modes 1000/1002/1003 are ONE mutually-exclusive
   tracking state, not additive flags, so "Setting 1000 under a child holding
   1002 demotes it to press/release, so the child never receives the motion".
   The guard is correct — the question is whether it still evaluates correctly
   after a reattach.
3. `console.go:1507-1512` → `ptychild/child.go:285` → `screen.go:107` —
   `childWantsMouse()` is `pane.child.Mouse()`, a bool on the **per-child**
   `Screen`.
4. `ptychild/screen.go:429-432` — `s.mouse` is set **only by observing** a
   DECSET `?1000/?1002/?1003` in the child's output stream. It is never queried.
5. `couchtty/console.go:1305` — a "detach/reattach cycle … mints a new pane for
   the same thread", i.e. a new `Child` and a fresh `Screen` with `s.mouse`
   zero-valued, while the child *process* keeps running and will not re-emit its
   startup DECSET.
6. `ptychild/ring.go` / `replay.go:68` — replay can only re-derive the bool if
   the original DECSET is still in the ring, and the ring keeps just the last
   `DefaultRingBytes`. Once zellij's startup `?1002h` has scrolled out, replay
   cannot restore it.

That composes to: after a reattach whose replay window no longer contains the
child's mouse DECSET, `childWantsMouse()` is permanently false, so every paint
writes `?1000;1006h` over a child that is really holding `?1002`. The host then
delivers press and release but **no drag motion** — so zellij cannot paint the
live selection, and computes+highlights it at mouse-up from the press/release
pair. That is the reported symptom exactly, it explains why standalone `pair` is
unaffected (no couch console re-asserting), and the ring dependence explains
"after **some** reattachment" rather than all.

Unverified: which tracking mode zellij actually holds (`?1002` vs `?1003`), and
whether the reattach path really mints a fresh `Screen` rather than carrying the
existing child across. Both are cheap to settle — see Plan.

### This is a third face of the defect #172 is mid-fix on

`#172` (status `working`, on the checked-out branch) has now fixed this same
demotion twice, from two sides, and its own commit message names the pattern:
*"Guarding one side of a two-sided defect is why it came back."*

- **BR-22 — the WRITE side.** `paintNow` asserted `?1000` unconditionally,
  clobbering a tracking child. Fixed by the `!childWantsMouse()` guard.
- **BR-26 — the OBSERVATION-CONFLATION side.** `Screen` folded tracking and
  encoding into one bool, so `?1002h` then `?1006l` read as "no mouse" and the
  guard's input was wrong. Fixed by splitting `Mouse()` / `SGRMouse()`
  (511bf9ca, 2026-09-06).
- **This issue — the OBSERVATION-LOSS side.** The guard's input is derived by
  *passively observing* the child's stream. A reattach mints a fresh `Screen`
  and the bounded ring can no longer supply the original DECSET, so the input is
  wrong again — not by conflation this time, but by absence.

So the class is: **every path by which couch's model of the child's mouse mode
can diverge from what the child actually holds.** Two of three paths are guarded;
the recovery path is not. Naming the class is the point — patching this one route
is what the previous two rounds already did.

**It falsifies an assumption stated in #172's own Spec.** Spec §4 rests on
"`ptychild` replay re-asserts the child's modes across a switch
(`replay.go:46`)" and concludes mode ownership is therefore not that issue's
deliverable. That holds for a *switch* between live panes, where the ring still
carries the modes. It does not hold for a reattach after enough child output to
evict them — replay can only re-assert what is still in the window.

Because #172 is actively editing these exact files, this should be picked up
**after** it merges, or folded into it deliberately — not worked concurrently.
`deps: ["#172"]` records that.

### Related defect the same reading exposes

`s.mouse` is one bool for three mutually-exclusive modes, so **which** mode the
child holds is not recoverable even where the "does it hold one" answer is. A
supervisor that wanted to restore the child's tracking after a reattach has
nothing to restore it *to*. `screen.go:424-428` already made the
tracking-vs-encoding split for this reason (BR-26, pair#172); this is the same
argument one level in — the specific mode is also a fact, and it is still
collapsed.

## Spec

**Invariant:** across a detach/reattach cycle, the host terminal's mouse
tracking mode must end up matching what the live child actually holds. couch may
only assert its own click reporting while no child holds tracking — which is
already the intended rule at `console.go:1043`; reattach must not be able to
launder a tracking child into a non-tracking one.

Two candidate directions, to be chosen once the Plan's diagnostic settles which
link actually breaks:

- **Carry the mode across the reattach.** Preserve the child's observed mouse
  state when the new pane is minted for a live thread, so `childWantsMouse()`
  keeps answering true and couch simply stays out of the way (its existing,
  deliberate trade).
- **Replay the child's mode to the new host.** Requires knowing the specific
  mode, i.e. replacing the `s.mouse` bool with the last-set tracking mode
  (none/1000/1002/1003) — which also fixes the related defect above, and lets
  couch re-assert the child's actual mode rather than its own.

Preferred: whichever keeps the *child's* declaration authoritative rather than
adding a second place that decides mouse modes.

Out of scope: changing `EnableMouseClicks` to include `?1002`. The
motion-report-rate reasoning at `control.go:54` is sound and is not what is
broken here.

## Done when

- Under couch, drag-selecting in the AGENT pane paints the highlight *during*
  the drag, and keeps doing so across repeated detach/reattach cycles —
  including after enough child output to evict the startup DECSET from the ring.
- A regression test pins the invariant at the seam rather than through the UI:
  a child that has declared tracking, put through the reattach path with a ring
  window that no longer contains its DECSET, still reports as holding tracking
  (and couch does not write `EnableMouseClicks` over it).
- Standalone `pair` selection behavior is unchanged.
- `copy-on-select` handoff (#125 source-pane gate) still fires on mouse-up.
- The divergence paths are **enumerated**, not patched one at a time: for each
  route by which couch's model of the child's mouse mode can go stale (fresh
  launch, pane switch, reattach, child exit, child mode change mid-session), the
  invariant either holds by construction or has a test row. This is the third
  recurrence of one defect; the enumeration is what ends it.
- No regression in the BR-22 case the guard exists for: a child holding tracking
  never has its drag-closing motion stolen (nvim must not wedge in visual
  selection).

## Plan

- [ ] Confirm the mode zellij holds: capture the child's DECSET traffic under
      couch (`?1002h` vs `?1003h`, and whether `?1006h` accompanies it).
- [ ] Confirm the state loss: instrument or test `childWantsMouse()` across a
      detach/reattach with an evicted ring, and confirm `paintNow` then writes
      `EnableMouseClicks` over a tracking child.
- [ ] Pick the direction per Spec; if the mode-granularity route is chosen,
      replace the `s.mouse` bool with the specific tracking mode and sweep its
      readers (`console.go:1511`, `console.go:1531`, `termcmd/run.go:819`).
- [ ] Regression test at the seam per Done-when.
- [ ] Manual verification: drag-select in the agent pane after several
      reattach cycles with heavy scrollback in between.

## Log

### 2026-09-06

Filed from an operator report: no during-drag highlight in the agent pane under
couch after reattachment, correct in standalone pair, highlight appears on
mouse-up. Hypothesis above is code-read only — nothing has been reproduced or
instrumented yet, hence the two diagnostic steps ahead of any change.

The `?1000`-demotes-`?1002` mechanism is not a new discovery: `console.go`
documents it as BR-22 and guards `paintNow` against it. What is new is that the
guard's input (`s.mouse`) is reconstructed by *observing the child's stream*,
which a reattach + bounded ring can silently lose — a passive-observation
invariant that holds on the fresh path and fails on the recovery path.

Checked against in-flight work before filing rather than after: #172 is
`working` on this branch and its HEAD commit (511bf9ca, hours old) fixes the
second face of this same defect. Filed separately rather than folded in because
#172 has a derived estimate and a closed M1, and this is a distinct repro on the
recovery path — but the two are one defect class, so the decision to keep them
apart is the operator's to revisit. See the third-face section above.

Worth noting as a motivating case for ariadne#215 (`ARCH-ORDER`): durable state
crossing an external event (reattach) whose restoration depends on a lossy
bounded replay, and the smoke test — a human dragging a mouse — cannot
distinguish "restored" from "happened to still be in the ring". The defect has
now recurred three times in one component precisely because the state's
reconstruction path was never enumerated against the events that can reach it.
