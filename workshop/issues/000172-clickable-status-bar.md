---
id: 000172
status: working
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-05
estimate_hours: 2.42
started: 2026-09-05T12:13:46-07:00
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

**B. The switcher.** A single click selects **and enters** — identical to
pressing Return on that actor. Clicking outside any actor does nothing.

**A click anywhere in an actor's rendered extent counts**, not just its primary
line: the notification line, and the description line once `#173` gives it one,
all belong to that actor and all select it. The hit-test therefore maps a point
to an **actor**, not to a line — an actor occupies a variable number of rows and
that count changes as notifications arrive and descriptions appear.

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

**3. No double click, and therefore no timing window.** An earlier draft had
click-to-select plus double-click-to-enter. Dropped: single-click-confirm
matches pair's own insert-mode `<LeftMouse>` handler
(`nvim/init.lua:3571`), which computes the target index and selects-and-confirms
in one click inside the completion popup — so the workbench already has one
click idiom, and a second would be the inconsistency.

It also deletes a problem rather than solving one. This project rejected
double-ESC because a double-tap needs a timing window that either delays every
legitimate single press or forwards one it cannot retract; a double-click has
the same shape. Not having one is strictly better than picking a good threshold
for one.

**What makes confirm-on-click safe here is `#170`'s switch rule:** a misfired
switch costs one key. `ctrl+backspace` returns to `previous`, and — by the
`entered_via_notification` rule — a click is a *manual* switch, so it re-pins
`previous` and the bounce-back lands where the operator actually was. Reuse that
handler's geometry approach (hit-test against the drawn box), and now its
semantics too.

**4. Mode ownership — REVISED 2026-09-05, see `## Revisions`. The child's half
already exists.** Mouse reporting is a terminal-global mode, not a per-region
one, so:

- If the child never enabled tracking, the terminal sends nothing and couch
  cannot see a click at all — couch must enable it itself. But then the child
  starts receiving mouse reports it never asked for, and couch must swallow
  every report that is not on its own row. Otherwise pointer movement types SGR
  bytes into the child as typeahead — the exact hazard documented at
  `couchtty/panelkeys.go:38-39`.
- If the child *did* enable tracking (nvim, zellij), couch must forward its
  events untouched and must not disable them.

**That tracker already exists.** `ptychild.Screen` scans the child's output for
the mouse DECSETs and exposes `Mouse()` (`screen.go:104`, table at `:415`); its
own comment records that it absorbed `termcmd.updateMouseMode` precisely so a
sequence split across two pty reads is not missed. `ptychild` replay re-asserts
the child's modes across a switch (`replay.go:46`). couch's blanket
`hostty.ResetInteractiveModes` at teardown (`couchtty/console.go:891`) remains
the teardown authority.

So this issue does NOT introduce that tracker, and mode ownership is not the
deliverable. What is missing is narrower and lives in `couchtty`: the operator's
input path does not RECOGNISE a mouse report at all — the `Interceptor` forwards
anything it does not know, so couch cannot withhold one — and there is no
routing decision and no click-to-actor geometry.

Note `pair#166` (punted) — "couch resume parked codex restores mouse mode" — is
the same missing state seen from the park/resume side. Explicit mode ownership
should subsume it; re-check #166 when this lands rather than leaving it punted
by default.

## Done when

- Clicking a chip attaches to that actor; clicking empty row space does
  nothing.
- In the switcher, a single click enters the clicked actor, taking exactly the
  path Return takes — asserted against the same handler, not a parallel one
  (`ARCH-DRY`).
- A click on an actor's notification line, or on its description line once
  `#173` lands, selects that actor — asserted for an actor rendered across
  several lines, not just its primary one.
- A click on an actor is treated as a **manual** switch by `#170`'s rule: it
  re-pins `previous`, so `ctrl+backspace` undoes it. Asserted, including a click
  on an actor that is showing a notification — that must NOT count as
  notification handling.
- Point-to-actor hit-testing is a pure function of the rendered menu and the
  click coordinates, unit-tested with no terminal: clicks outside every actor,
  on a scrolled or clipped list, and on an actor whose line count differs from
  its neighbours'.
- Column-to-actor is unit-tested against the same render pass, including a
  width narrow enough to clip chips and one narrow enough to drop them.
- A child that never enabled mouse tracking receives **zero** mouse bytes while
  couch's tracking is on — asserted, not assumed. This includes RELEASES: the
  release-always-forwards rule `termcmd` uses is sound only where the child is
  already receiving presses, and forwarding a release to a child that never saw
  its press is both an unpaired event and a direct contradiction of this bullet.
- A child that did enable tracking still receives its own events unchanged
  (nvim selection and scroll still work inside an attached pair session).
- Teardown leaves the host terminal with mouse reporting off.
- `pair#166` is re-evaluated against the new mode tracking and either closed,
  fixed, or re-punted with a reason.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: cross-cutting-refactor     design=0.04 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.02 impl=0.12
item: smaller-go-module          design=0.06 impl=0.20
item: milestone-review           design=0.00 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: milestone-review           design=0.00 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.02 impl=0.12
item: atlas-docs                 design=0.02 impl=0.05
item: real-api-discovery         design=0.00 impl=0.12
item: milestone-review           design=0.00 impl=0.20
design-buffer: 0.15
total: 2.42
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**One line per instance**, so a close-time miss is attributed to a primitive
rather than to the issue. In order:

| Slug | Instances |
| --- | --- |
| `cross-cutting-refactor` | promoting the SGR parser out of `termcmd` into `mouseinput`, with `termcmd`'s own suite as the regression |
| `smaller-go-module` | chip spans; `ColumnToActor`; actor extents + `PointToActor`; `RouteMouseReport`; the `Interceptor`'s `seqMouse` + bound; couch's own enable/teardown; `onMouse` + the manual marker; the manual-switch test |
| `milestone-review` | one per boundary, M1/M2/M3 |
| `atlas-docs` | `menuControls`, the README couch section, the atlas routing rule |
| `real-api-discovery` | the manual terminal verification (nvim selection and scroll), which needs a real terminal, a real nvim and a real pointer |

Design hours carry the ×0.2 spec-quality discount: the plan names every function,
its file, its test and its production sites, so the estimator is reading rather
than deciding.

**Known risk this number does NOT price.** `pair#187` came in at 1.64h against
0.87 — roughly 2×, and the overrun was review rounds, not code. This issue has
three boundaries and its plan alone took four plan-quality rounds. If the same
pattern holds the actual will be nearer 4h, and that gap is a calibration signal
worth keeping rather than an estimate to pad: the primitive table prices
`milestone-review` at 0.2 impl, and this repo's boundaries have been costing
several times that.

## Plan

Design landed at
`workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md`. Three
review boundaries, because the work is three different problems and only the
middle one is hard.

- [ ] M1 — the pure layer, testable with no terminal and no child: chip spans out
      of the render pass that already clips them, actor extents out of the menu
      renderer, point-to-**actor** hit-testing for actors that occupy a variable
      number of lines, and SGR report parsing.
- [ ] M2 — mode ownership, which the Spec calls the deliverable. Mouse reporting
      is a terminal-GLOBAL mode, so couch cannot enable it for itself without
      deciding what the child sees: scan the child's DECSET/DECRST to learn which
      modes it enabled, hold that per PANE, and decide couch / forward / swallow
      from it. The chunk-boundary case is the first test, not an afterthought.
- [ ] M3 — wiring: the click routed into the switch path `ctrl-space`+Return
      already takes, recorded as a MANUAL switch so `ctrl+backspace` undoes it,
      plus the `pair#166` re-evaluation the Done-when requires.

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

### 2026-09-03 — single click confirms

Operator's call, overriding the two-stage draft above: a single click selects
and confirms, matching the completion popup. It also removes the double-click
timing window rather than tuning it.

I had argued for two stages on the grounds that the switcher's Enter moves the
operator's terminal. That objection is weaker than it looked: `#170` gives a
misfire a one-key undo, and a click is a manual switch under the
`entered_via_notification` rule, so `previous` re-pins and the bounce-back is
correct. Recorded because "why isn't this two-stage like a file picker" is a
question the next reader will ask.

The extent requirement is the operator's too: every line belonging to an actor
is clickable, including the notification line and the future `#173` description.
That is what makes the hit-test point-to-ACTOR rather than point-to-row, and it
is the part most likely to be built wrong if it is not stated.

## Revisions

### 2026-09-05 — mode ownership is not the deliverable; the child's half exists

**Reason.** The plan-quality gate (PQ-1, Critical) found the first plan draft
designing a mouse-mode scanner and an SGR parser that are both already in the
tree, and this Spec is where that instruction came from.

**Delta.**

- `ptychild.Screen.Mouse()` already tracks the child's modes, split-read safe;
  `ptychild` replay already re-asserts them across a switch. Section 4's "couch
  has no per-mode state today ... that tracker is the deliverable" was wrong, and
  is corrected above. `console.go:737` is now `:891`.
- `termcmd` already parses SGR (`parseSGRMousePress`, `findSGRMousePress`,
  `isSGRMousePrefix`) and already decides the wheel/release policy this Spec
  never mentioned. The plan promotes them to a shared package rather than adding
  a second parser.
- The release rule is NARROWED for couch: a release forwards only when the child
  has mouse mode. `termcmd`'s unconditional forward is correct in its own
  context — the child there is already receiving presses — but in couch a child
  with no tracking must receive nothing at all, which the Done-when now says
  explicitly.
- What remains is the real work: the `Interceptor` cannot withhold a report it
  does not recognise, there is no routing decision, and there is no click-to-actor
  geometry.

Design: `workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md`.

