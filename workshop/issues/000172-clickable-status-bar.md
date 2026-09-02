---
id: 000172
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Clickable status bar switches to that actor

## Problem

The reserved row already renders one chip per actor (`couchtty/reserve.go` —
`StatusActor` "one chip on the row", `RenderStatusRow`), and those chips are
exactly the things the operator wants to reach. Reaching one today means
`ctrl-space`, then finding it in the switcher — a keyboard round trip to select
something already visible and already pointed at.

## Spec

Clicking a chip on the reserved row switches to that actor. Clicking bare row
does nothing.

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

**3. The real work is mode ownership, and it does not exist yet.** Mouse
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
