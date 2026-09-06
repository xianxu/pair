---
id: 000172
status: done
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-06
estimate_hours: 2.69
started: 2026-09-05T12:13:46-07:00
actual_hours: 5.62
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
total: 2.69
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

**One milestone, M1, covering all of it** — see `## Revisions` for why the
original three collapsed into one.

- [x] M1 — the whole feature. The pure layer: chip spans out of the render pass
      that already clips them, actor extents out of the menu renderer,
      point-to-**actor** hit-testing for actors occupying a variable number of
      lines, and SGR report parsing promoted out of `termcmd`. The terminal half:
      mouse reporting is a terminal-GLOBAL mode, so couch cannot enable it for
      itself without deciding what the child sees — it reads the child's modes
      from `ptychild.Screen`, which already tracks them, and decides couch /
      forward / swallow. And the gesture: the click routed into the switch path
      `ctrl-space`+Return already takes, recorded as a MANUAL switch so
      `ctrl+backspace` undoes it.

`pair#166` is re-evaluated and stays punted, with the answer written into that
issue's Log: #172 owns the mouse MODE, while #166's symptom is Zellij's
`scroll-up` pinned after a reduced scrolling region (`DECSTBM`) — different
terminal state, reproduced with no mouse mode involved. The assumption behind the
punt ("explicit mode ownership should subsume it") was wrong, which is why the
Done-when asked for an answer rather than a default.

## Log



- 2026-09-06: closed — Clicking an actor switches to it from the status row and from the switcher, smoke-tested on the real stack; the operator confirmed pair#196 fixed on the rebuilt binary. BR-33 and its unswept sibling I1 are both addressed as the state model the finding asked for rather than a fifth site: Screen now records whether it has OBSERVED a mouse DECSET at all (so a reattached child, whose fresh Screen has seen nothing, is UNKNOWN rather than "wants none" — silence is not consent), AND latches a repaint on a mode change, so a child dropping its tracking makes couch reclaim the terminal without waiting for an unrelated paint. Both red-verified. Three things that gap exposed are worth more than the fix and are recorded: the mouse fixture never called SetSink, so no child output reached the console and every mode test could only assert a hand-called effect — the missing trigger was invisible by construction; a paintNow I added on the RowDirty branch turned out not to be the mechanism, since removing it left every test green and an existing paintPending branch already paints once the stream is whole, so it was deleted with a comment saying why; and a third orphaned doc block from the childWantsMouse rename. Earlier rounds: tracking and encoding no longer share a bool, couch forwards SGR only to a child that asked for that encoding, the atlas sentence that PRODUCED the demotion is replaced, chip spans are pinned as display columns, the Manual-on-refusal guard is pinned, and the plan is reconciled with the tree. Full ./cmd/... suite green; go test -race green on couchtty and ptychild.; review verdict: FIX-THEN-SHIP
- 2026-09-05: closed M1 — M1 ships the click geometry; BR-22 is addressed and its own measurements now falsify it. It measured "deleting the paintNow re-assert leaves the whole couchtty suite green" and "1 of 6 cells implemented, 0 of 6 pinned" — re-measured at HEAD, deleting the re-assert reddens 5 tests and making it unconditional reddens 6, because the mode-transition table is now the test list: child enables 1000/1002/1003, child disables, child exits with mouse on, a switch between two children with different modes, and the no-child baseline, one case each. One rule produces every row: the child mode wins whenever it has one and couch takes the terminal back the moment it does not. The previous round produced a verdict with no disposition block, so BR-22 carried forward from round 5 by default rather than being re-judged against the fix. Also fixed since: BR-21 HitMouse now has a real handler rather than an empty func the dispatcher skipped, BR-20/BR-24 dead aliases deleted and sgrMouseSize derives its length from ParsePrefix instead of restating the framing rule, BR-19/BR-9 two comments I detached from their subjects by inserting declarations between them, BR-25 the plan prose renamed. Operator smoke-tested twice on the real stack. Full ./cmd/... suite green.; review verdict: FIX-THEN-SHIP
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

### 2026-09-05 — M2 and M3 are one boundary, because they shipped as one

**Reason.** The routing, the ownership and the wiring landed in a single commit
(`f95da992`), so there was one boundary in fact. Closing them separately would
review the combined window once and then review an empty range — the redundant
double-log AGENTS.md §3 warns about, and `pair#182`'s review already caught me
hand-ticking a milestone with no verdict behind it.

The boundary review found this before I acted on it (BR-15): "M2 and M3
production code landed inside the M1 window, so their own boundary reviews open
on an empty range."

**Delta.** M3's scope folds into M2. The issue keeps two milestones — the pure
layer, and everything that touches the terminal — which is the split the work
actually had. I over-split at plan time: three milestones assumed mode ownership
was large, and it was not, because `ptychild.Screen` already owned the child's
half.

**Not done:** faking a second close. A tick without a `Review-Verdict` is a
boundary marker where no boundary happened.

### 2026-09-06 — three milestones collapse to one, because there was one boundary

**Reason.** M2's production code all landed inside M1's review window. The
routing, the mode ownership and the wiring shipped in `f95da992`, before any M1
boundary existed, so M1's close reviewed them — which is why six of its findings
(BR-16, BR-17, BR-20, BR-21, BR-22, BR-24) are about mouse routing rather than
geometry. Closing M2 afterwards would open on fix-ups only: a review of patches
to code whose own milestone never got reviewed as a unit.

**Delta.** M1 is the whole feature. M2's row is gone.

**This is the THIRD correction to the same over-split**, which is the finding
worth keeping. M3 merged into M2 on 2026-09-05 for the same reason, and now both
merge into M1. The split came from a plan-time guess that mode ownership was
large; it was not, because `ptychild.Screen` already owned the child's half and
`termcmd` already had the parser. The lesson is not "merge milestones when they
collide" — it is that a milestone boundary is a claim about what will ship
SEPARATELY, and a plan that cannot yet tell how much of the work already exists
cannot make that claim honestly. AGENTS.md §3's rule ("tag `Mx` only for work
with ≥2 boundaries you will genuinely close separately") is the same instruction
read forward.

**Cost, stated.** Seven review rounds on ~450 lines of new logic, because one
boundary was reviewed seven times while the round counter attributed all of it
to "M1".

### 2026-09-06 — manual verification, recorded as a measurement

Plan Task 12 Step 2 required this written as a measurement rather than "it
worked", and the first version ("operator smoke-tested twice on the real stack")
failed that (BR-34). What was actually run, and what it can and cannot tell us:

**Run 1, after the first wiring.** Clicked a chip on the reserved row: switched
to that thread. Clicked empty space right of the last chip: nothing happened.
Did NOT exercise the switcher, which is why a routing bug survived it — every
switcher click was being swallowed before reaching the panel branch, and the
report was "works fine".

**Run 2, after the routing fix.** Clicked a row in the switcher: switched.
Operator's words: "tested again, mouse click to switch works fine."

**What neither run covered, and this is the load-bearing part.** Neither run
recorded which mouse modes the attached child held, and both attached children
were Pair sessions, whose nvim and zellij announce `?1006`. So the mainstream
configuration passed while the one the operator later reported broken —
`pair#196`, an agent pane holding `?1002` after a reattach, losing its live drag
highlight — was invisible to both runs. The smoke test could not have caught it:
the failure needs a child whose mode couch never observed, which is what a
reattach produces.

That is the measurement's real result. The Done-when "a child that did enable
tracking still receives its own events unchanged" was NOT verified by these runs;
it is verified by `TestAReattachedChildKeepsItsTrackingMode`, which reproduces
`pair#196`'s path and reddens when the belief is treated as "no" rather than
"unknown".

### 2026-09-06 — a flake seen once, measured rather than dismissed

`TestActiveChildExitFocusesPanelRecordsCauseAndForgetsActor` failed once during a
full `./cmd/...` run (`forgot ("", ""), want (c1, c1)`), immediately after the
`couchMayOwnTheMouse` change touched `onExit`'s path — the shape of a regression.

Measured instead of assumed: 0/12 isolated, 0/5 whole-package, 0/3 full
`./cmd/...`. It is a load-sensitive flake in a `waitFor`, not a regression. Left
unfixed and recorded rather than silently dismissed, because "it passed when I
ran it again" is the reasoning that hid the `-race` flake in `pair#187` until a
reviewer measured it at 3-in-10.

