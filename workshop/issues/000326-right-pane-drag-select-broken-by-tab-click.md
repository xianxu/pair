---
id: 000326
status: working
deps: []
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
started: 2026-09-24T22:18:43-07:00
---

# Right-pane drag selection broken since clickable tabs (#311)

## Problem

Since #311 (clickable right-pane tab strip), mouse drag-to-select text in the
right pane (`pair term`) no longer works — operator report, 2026-09-24.

Likely cause (from reading the #311 diff, not yet reproduced): `1c5ed349`
switched `newTerminalMux` from `terminal.ChildRequested` to
`terminal.AnyMotion` (`cmd/internal/termcmd/presentation.go`). Before, a plain
shell tab (child tracking 0) meant the parent requested no mouse reports, so
zellij/the host terminal did native text selection. Now the parent always
emits `\x1b[?1003h`, so the host forwards every press/drag to `pair term`
instead of selecting. In `Presenter.mouseInput`
(`cmd/internal/terminal/presenter.go`), a press over a child with
`Tracking == 0` becomes `ParentPressMouse`, which is swallowed — and the
parent has no selection implementation of its own. Net: the drag goes
nowhere.

## Spec

Both must hold: a click on a strip chip switches tabs (#311), AND drag
selection in the child area behaves as it did before #311 when the active
child has not requested mouse tracking. When the child *has* requested
tracking (vim, less with mouse, etc.), behavior is unchanged.

Candidate directions (pick during design):
- Scope parent mouse reporting so the host keeps native selection over the
  child rows — e.g. go back to `ChildRequested` and find another route for the
  strip click. Constraint: terminals enable mouse reporting per pane, not per
  row, so this likely needs a different click source (zellij-level hook?).
- Implement parent-side selection in the presenter (highlight + OSC 52 copy),
  as a multiplexer would — larger, and must match host selection feel.
- Accept Shift+drag as the selection gesture (zellij bypasses app mouse
  capture on Shift) — only if the operator agrees; check it actually works.

## Done when

- In a plain shell tab in the right pane, click-drag selects text and it can
  be copied, as before #311 (operator smoke test).
- `pair term` no longer requests mouse reports for a child without tracking
  (`TestPresentationLeavesMouseOffForAPlainChild`, mutation-checked).
- Strip clicks still work while the child requests tracking (existing #311
  click tests stay green); README + atlas say plain-shell strip clicks are off.

## Plan

- [x] Revert `pair term` to `terminal.ChildRequested`; invert the #311 policy test
- [x] README + atlas describe the narrowed behavior
- [ ] Operator smoke test
- [ ] Clickable strip over a plain shell *without* losing selection is left for
  a follow-up issue (options above; parent-side selection breakdown in Log)

## Log

### 2026-09-24

- Filed from operator report. Suspect commit `1c5ed349` (#311,
  ChildRequested → AnyMotion); follow-up `8aaa68e3` only touched active-chip
  click effects.
- Why couch doesn't hit this despite the same `AnyMotion` policy and no
  selection code of its own: couch's child is a zellij client, which always
  requests mouse tracking, so `RouteMouseReport` forwards drags down and
  *zellij* does the selection. In `pair term` the layering is flipped —
  zellij is the host above, the child is a plain shell with tracking 0, so
  drags are swallowed and nothing below implements selection. Copying couch
  therefore doesn't fix it; the selection has to come from somewhere.

- Operator chose the revert (needs selection now): `pair term` back to
  `ChildRequested`; couch keeps `AnyMotion`. `clickStrip` stays — it still
  fires when the child requests tracking.

## Revisions

- 2026-09-24: scope narrowed from "clickable strip AND selection" to "restore
  selection by reverting the mouse policy"; clickable strip over a plain shell
  deferred.
