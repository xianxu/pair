---
id: '000196'
status: done
created: 2026-09-06
updated: 2026-09-06
actual_hours: 0.40
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
