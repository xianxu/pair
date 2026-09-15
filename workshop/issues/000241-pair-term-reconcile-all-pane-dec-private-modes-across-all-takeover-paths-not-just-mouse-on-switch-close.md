---
id: 000241
status: open
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
---

# pair term: reconcile ALL pane DEC private modes across ALL takeover paths, not just mouse on switch/close

## Problem

#240 made `pair term` reconcile the active tab's MOUSE modes to the pane on a
takeover, fixing the reported "can't select in the shell" bug. Its close
review (BR-3) measured, by widening the probe's regex to every DEC private
mode, that OTHER modes still leak across a tab switch:

- switching from an nvim tab to a shell tab writes only `?1002l ?1006l`,
  leaving nvim's `?1004h` (focus in/out reporting) on the pane, so zellij
  then forwards focus events to the shell that never asked for them;
- `?1h` (DECCKM, cursor-key application mode) and `?2004h` (bracketed paste)
  likewise stay set.

And (BR-1) the mouse reconcile itself is skipped on one takeover path:
`removeTab` with a rename open takes the `paintStripInline` branch
(`run.go:1471-1486`) instead of `applyTakeover`, so a closed child's modes
outlive it until the next real switch.

The general shape #240 named but did not finish: **the pane is a proxy for
its active child, so ALL of the child's terminal modes must follow the active
child across EVERY takeover.** #240 did one mode family on two paths.

## Spec

Reconcile the full set of DEC private modes a child can hold, not just mouse,
on every takeover path. Two axes to enumerate (ARCH-ORDER):

**The modes.** Extend `ptychild.Screen` to record the child's held private
modes as a set, and `hostty` to format a reconcile delta over an arbitrary
mode set (generalise `PrivateModes`/`mouseReconcile`). Enumerate which modes
reconcile and which are deliberately excluded:
- reconcile: mouse tracking (1000/1002/1003) + 1006 (done in #240), focus
  events (1004), bracketed paste (2004), cursor-key mode (1/DECCKM), and any
  other input-affecting private mode a child sets.
- exclude, with reason: 1049/1047/47 (alt-screen) and 1048 (cursor save),
  which the repaint/replay already own (`repaint.go:55`) and which move the
  cursor through the save slot; asserting them here would fight the replay.

**The paths.** Every takeover, not just switch and background-exit:
`applyTakeover` (switch), `removeTab` normal close (done), AND `removeTab`
with a rename open — which must reconcile even though it does not blank the
screen (dispose BR-1 here, through this events axis, not by patching the
site).

## Done when

- The mode-sweep probe (extend `probes/mousemodesmoke`) shows nvim's `?1004h`,
  `?2004h`, and cursor-key mode released on the switch to the shell and
  re-raised on the way back.
- A `removeTab`-with-rename-open test shows the dead child's modes reconciled.
- The excluded set (alt-screen, cursor save) is asserted to be left to the
  replay, with the reason in a test comment.
- `atlas/` records the "pane mirrors the active child's full mode set" rule.

## Plan

- [ ] `Screen`: record the held private-mode set; accessor + tests
- [ ] `hostty`: reconcile-delta formatter over an arbitrary mode set (generalise #240's)
- [ ] Wire it into `applyTakeover` and both `removeTab` branches
- [ ] Extend the probe to every reconciled mode; mux tests per path
- [ ] atlas + close

## Log

### 2026-09-12

- Split from #240's close review (BR-1, BR-3). #240 fixed the reported mouse
  symptom; this is the general "pane mirrors the active child's full mode
  set" work it deliberately scoped out. See also #200's "one arbitration,
  two consumers" consolidation and #207 (couch's own mouse assertion).


## Revisions

### 2026-09-15 — #255 M3 integration evidence

Pair term now uses the same endpoint/presenter as Couch for selected parent modes, panels, tab switch/removal, resize and release. Child queries/input encoding stay origin-bound. Production raw takeover scanner/replay authority is removed. Full term normal/race tests pass; native composed conformance and operator smoke remain pending.
