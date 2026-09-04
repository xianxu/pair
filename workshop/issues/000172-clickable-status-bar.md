---
id: 000172
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Mouse support: click the status bar and the switcher

## Problem

The reserved row already renders one chip per actor (`couchtty/reserve.go` —
`StatusActor` "one chip on the row", `RenderStatusRow`), and those chips are
exactly the things the operator wants to reach. Reaching one today means
`ctrl-space`, then finding it in the switcher — a keyboard round trip to select
something already visible and already pointed at.

## Spec

**Two surfaces, one mouse-mode owner.**

**A. The reserved row.** Clicking a chip switches to that actor. Clicking bare
row does nothing.

**B. The switcher.** Single click selects a row; double click enters it —
identical to pressing Return on that row. Clicking outside any row does
nothing.

The switcher is materially easier than the row: couch draws the whole panel and
no child is attached, so there is no forward-versus-swallow question there. The
mode ownership below is about the attached-child case only.

**1. The row renderer publishes chip spans.** `RenderStatusRow(width, m)`
returns the column range of each chip alongside the string it already returns.
The spans must come from the same pass that clips chips to width, not a second
derivation — a re-computed mapping would drift from the render at exactly the
narrow widths where clipping happens (`ARCH-DRY`). Column-to-actor is then a
pure function of `(StatusModel, width, column)` and unit-testable with no
terminal (`ARCH-PURE`), including the clipped and overflowing cases.

**2. Routing.** couch owns the last row by reservation, so a mouse report whose
row equals the host's row count is couch's; anything else forwards to the child
unchanged. Request **SGR encoding (`?1006`)** — the legacy X10 encoding caps
coordinates at 223, which a wide or tall terminal exceeds silently. Request
click reporting (`?1000`), **not** motion tracking (`?1002`/`?1003`): motion
reports arrive at pointer-movement rates and this feature needs human click
rates (`ARCH-CONSTRAINTS`).

**3. Double click, and why the timing window is acceptable here.** This project
rejected double-ESC precisely because a double-tap needs a timing window that
either delays every legitimate single press or forwards one it cannot retract.
The difference that makes double-click fine: **the single press has a harmless
meaning.** A too-slow second click merely re-selects the row it already
selected; a mis-timed ESC changed a mode or cancelled a turn. So the window
never has to be right, only reasonable — put the threshold in a named constant
rather than a literal, and let a slow double-click degrade to two selects.

Note a deliberate inconsistency with the workbench: pair's insert-mode
`<LeftMouse>` handler (`nvim/init.lua:3571`) computes the target index and
**selects and confirms in one click** inside the completion popup. The switcher
does not, because its Enter switches the operator's terminal — a misfire there
is disruptive in a way that picking a completion is not. Reuse that handler's
geometry approach (hit-test rows against the drawn box), not its one-click
semantics.

**4. The real work is mode ownership, and it does not exist yet.** Mouse
reporting is a terminal-global mode, not a per-region one, so:

- If the child never enabled tracking, the terminal sends nothing and couch
  cannot see a click at all — couch must enable it itself. But then the child
  starts receiving mouse reports it never asked for, and couch must swallow
  every report that is not on its own row. Otherwise pointer movement types SGR
  bytes into the child as typeahead — the exact hazard documented at
  `couchtty/panelkeys.go:38-39`.
- If the child *did* enable tracking (nvim, zellij), couch must forward its
  events untouched and must not disable them.

couch has **no per-mode state today**: the only handling is a blanket
`hostty.ResetInteractiveModes` at teardown (`couchtty/console.go:737`), which
resets every mouse encoding at once. So this issue has to introduce tracking of
which modes the *child* enabled, in order to decide forward-versus-swallow.
That tracker is the deliverable; the click mapping is the easy half.

Note `pair#166` (punted) — "couch resume parked codex restores mouse mode" — is
the same missing state seen from the park/resume side. Explicit mode ownership
should subsume it; re-check #166 when this lands rather than leaving it punted
by default.

## Done when

- Clicking a chip attaches to that actor; clicking empty row space does
  nothing.
- In the switcher, a single click selects the clicked row and does not enter it;
  a double click enters it, taking exactly the path Return takes — asserted
  against the same handler, not a parallel one (`ARCH-DRY`).
- A double click slower than the threshold degrades to two selects, never to a
  half-action.
- Row hit-testing is a pure function of the rendered menu and the click
  coordinates, unit-tested with no terminal, including clicks outside every row
  and on a scrolled/clipped list.
- Column-to-actor is unit-tested against the same render pass, including a
  width narrow enough to clip chips and one narrow enough to drop them.
- A child that never enabled mouse tracking receives **zero** mouse bytes while
  couch's tracking is on — asserted, not assumed.
- A child that did enable tracking still receives its own events unchanged
  (nvim selection and scroll still work inside an attached pair session).
- Teardown leaves the host terminal with mouse reporting off.
- `pair#166` is re-evaluated against the new mode tracking and either closed,
  fixed, or re-punted with a reason.

## Plan

- [ ] Chip spans out of `RenderStatusRow` + pure column-to-actor mapping, with
      clipping tests.
- [ ] Row spans out of the menu renderer + pure point-to-row mapping.
- [ ] Switcher click/double-click, routed into the existing Return handler.
- [ ] Child mouse-mode tracking in the console: observe the child's DECSET/DECRST
      for the mouse modes, hold the state, restore on attach/detach.
- [ ] Enable `?1000;?1006` for couch; route last-row reports to couch and
      swallow the rest when the child has no tracking of its own.
- [ ] Wire the click to the existing switch path — the same operation
      `ctrl-space` + Return performs, so notification bookkeeping stays
      identical (`pair#170`'s `entered_via_notification` rule must see a click
      the way it sees any other manual switch).

## Log

### 2026-09-02

Raised while working through what the status row should carry. Depends on
`pair#170` only for the switch semantics: a click is a *manual* switch, so it
re-pins `previous` — it must not be treated as notification handling even when
the chip clicked is the one showing a notification.

### 2026-09-03

Scope extended by the operator to the switcher: click selects, double click
enters. Kept in this issue rather than split, because the mouse-mode ownership
in (4) is the expensive half and both surfaces need it.

The switcher half is the cheaper one and could land first: couch owns the whole
panel with no child attached, so it needs the decoding and the hit-test but not
the forward-versus-swallow arbitration. Worth sequencing that way if the row
turns out to be as fiddly as `#139` suggests terminal input usually is.
