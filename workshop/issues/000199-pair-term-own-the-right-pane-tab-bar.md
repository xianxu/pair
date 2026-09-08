---
id: 000199
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-07
estimate_hours: 7.52
started: 2026-09-06T19:24:15-07:00
---

# pair term: own the right pane tab bar

## Problem

`pair term` has tabs but no tab strip. It renders their state by shelling out
to zellij:

```go
// termcmd/run.go:959-963
return m.rt.RunZellijAction("rename-pane", "--pane-id", m.paneID, title)
```

Three costs, and the third is the one that matters:

- A subprocess per title change.
- Appearance is zellij's pane frame, not ours.
- **One string carries all tab state.** `paneTitleLocked` packs the whole tab
  set into a single rename argument because that is the only channel available.

So the operator's complaint — "the way we change tab title is not great" — is
not a polish problem. There is no surface to polish: a tab strip was never
built, and a rename call is being asked to stand in for one.

**The mechanism already exists and is already proven.** couch reserves a row
with DECSTBM and confines its child above it — `couchtty/reserve.go`
(`Reserve`, `PaintRow`) over `hostty.SetRegion` — and repaints when
`ptychild.Screen.TakeRowDirty()` says a child may have wiped it. That is a
status bar, built, working, and hardened by a full mouse round in `#172`.

**And `pair term` is the process that can use it.** couch cannot reach into the
right pane: its child is the whole zellij session, and zellij owns that pane's
rendering. `pair term` *is* the process in the pane, owns its pty, and already
drives the host seam (`hostty.NewOSHost`, `termcmd/run.go:236`). Everything the
reserved row needs is one package away from a caller that already has the other
half.

## Spec

**`pair term` draws its own tab strip in a reserved row of its own pane**,
replacing `rename-pane` as the way tab state reaches the operator.

**The lift: `Reserve`/`PaintRow` move to `hostty`.** They are host-half
mechanism, not couch policy, and this is exactly the split the atlas already
states for this pair of packages:

> What is shared is structure; what stays is policy. … `\x1b[r` lives here and
> only here; it was about to exist in two packages.
> — `atlas/architecture.md`, on the `#146` `ptychild`/`hostty` split

Same argument, next mechanism: the row-reservation primitive is shared
structure; *what the row says* is per-consumer policy (couch renders actors,
`pair term` renders tabs). `TakeRowDirty` is already in the shared child half,
so the repaint trigger needs nothing new.

### The strip can retire the pane frame — three jobs, not one

Read out of `#123`'s history on 2026-09-06, because "why is the right pane not
frameless?" has a documented answer that changes this issue's payoff.

**Frameless was tried and reverted**, 2026-07-27, and *not* over the title:

> post-close rework 2: live testing showed borderless split panes render with
> **no visible divider between the halves**. Reverted the split panes to bordered
> (`--borderless false`, still pinned): the frame is the divider and carries the
> `#118` tab title and scroll indicator.
> — `workshop/history/issues/000123-…`, Log

So the frame does **three** jobs, and the tab title — this issue's subject — is
only the middle one:

| job | source | can the strip take it? |
|---|---|---|
| **divider** between `Alt+Shift+d` split halves | `#123` Log; the reason the revert happened | **yes** — each pane draws its own strip, so a split shows two strips and the boundary is explicit |
| **`#118` tab title** | `#118`, `config.kdl:12-14` | **yes** — that is this issue |
| **scroll indicator** | `config.kdl:16-19`: frames are global *"so the agent pane shows zellij's scroll-position indicator … the only way to surface scrollback position, since zellij doesn't expose pane scroll offset to plugins or the CLI"* | **yes, and only `pair term` can** — see below |

The scroll indicator is the surprising one. `config.kdl` is right that *zellij*
cannot expose pane scroll offset. But `pair term` **owns the pty and the ring**
(`ptychild`), so it already holds its own scroll position without asking zellij
anything. The constraint that forced `pane_frames true` globally is a constraint
on zellij, not on us.

A fourth job is already dead and should not be re-litigated: the frame used to
be the drag handle, accepted as "residual frame-drag exposure". The tiled pivot
in `#123` made drag-immunity architectural — tiled panes have no mouse-move
operation — so frameless costs nothing in safety today.

**On the divider specifically:** what the revert established is that two stacked
shells with no boundary are unreadable — same font, same colours, one's output
running into the next. That argues for *a visual boundary*, not for *a frame*.
zellij offered frame-or-nothing, so the frame won by default. The strip is the
third option.

**Consequence for this issue's scope.** This was filed as "replace a bad title
mechanism". It is really "own the pane's chrome and stop paying zellij's": once
the strip carries all three jobs, the right pane can take `borderless=true` —
the per-pane opt-out the draft pane already uses — reclaiming **~2 rows + 2
columns** of frame chrome per pane (`config.kdl:22`), and the global
`pane_frames true` becomes a question about the agent pane alone rather than a
fleet-wide default.

Sequencing note: going borderless is a **follow-on**, not part of this issue.
Land the strip first, confirm it carries divider + title + scroll position in a
real split, and only then flip `borderless`. Reverting a strip is easy;
reverting a chrome change that the layout rungs depend on is not.

### The boundary — name it now

**`pair term` owns tabs within a pane; zellij keeps panes and splits.** The
failure mode this issue must not walk into is tab strip → "we also want splits"
→ a reimplemented multiplexer. Right-pane splits are load-bearing today
(`Alt+Shift+d`, `#125`'s source-pane gate, `#123`'s terminal-pane registry); a
split yields two panes each running `pair term`, and each drawing its own strip
is the coherent outcome, not a problem to solve.

What makes this line hold is the operator's scope decision (see Log): the right
pane hosts terminals only. A pane that must host arbitrary things needs zellij's
generality; a terminal-only pane can be opinionated.

### To settle in the plan

0. **Does the operator actually split the right pane?** The divider job is only
   load-bearing under `Alt+Shift+d`. If splits are rare in practice, the
   frameless follow-on gets simpler; if they are common, the strip's edge has to
   be convincing as a boundary before `borderless` is safe. Cheap to answer and
   it gates the follow-on.
1. **Whether `rename-pane` survives at all.** The zellij pane title is still the
   only label visible when the pane is *not* focused, and `#118`'s tab-strip
   titles and `#123`'s registry read it. Keeping a degraded title for outside
   consumers while the strip becomes the operator's surface is likely right, but
   it must be decided rather than defaulted — dropping it silently breaks
   consumers named in `run.go:229`.
2. **Row budget.** couch already reserves the host's bottom row; inside layout3
   the right pane would reserve one of its own. Confirm the two compose (they
   are different terminals, so they should) and that a one-row cost in an
   already-narrow pane is what the operator wants.
3. **Where the strip lives** — top or bottom of the pane, and whether it matches
   couch's row so the workbench reads as one system.

Out of scope: clicking it. That is `#200`, deliberately separated because it
carries a constraint of its own.

## Done when

- `pair term` renders a tab strip in a reserved row of its own pane; tab state
  is legible without reading the zellij pane frame.
- The strip survives a full-screen child's startup clear (the `TakeRowDirty`
  path couch already handles).
- `Reserve`/`PaintRow` live in `hostty` with both couch and `termcmd` calling
  them; `grep` finds no second row-reservation implementation.
- couch's own status row is unchanged in behavior — the lift is a move, and a
  regression there means it was a rewrite.
- The `rename-pane` decision from plan item 1 is implemented and its consumers
  (`run.go:229`, `#118`, `#123`) still get what they read.
- Splits still work: two right-pane halves each run `pair term` and each draw
  their own strip.
- `atlas/architecture.md` records the row primitive as shared host-half
  structure, alongside the existing `\x1b[r` note.
- The strip displays scroll position for its own pane, sourced from `ptychild`
  rather than from zellij — the fact that makes the frameless follow-on possible
  is demonstrated, not merely argued.

## Plan

Four milestones, each its own review boundary — detail in
`workshop/plans/000199-pair-term-own-the-right-pane-tab-bar-plan.md`.

- [x] M1 — Lift `Reserve`/`PaintRow`/`ChildRows` into `hostty` as a
      `Reservation` carrying its edge; repoint couch. Proven a MOVE by couch's
      tests passing **unedited**.
- [ ] M2 — Make `termcmd` single-writer and add the mid-sequence paint gate.
      A prerequisite, not cleanup: two writers is how a paint lands inside a
      child's escape sequence (`atlas/couch.md`).
- [ ] M3 — Render the strip from the existing tab model, with display-column
      spans; repaint on `TakeRowDirty`, re-`Reserve` before repainting;
      degrade `rename-pane` to a short title for the `#118`/`#123` consumers.
- [ ] M4 — `borderless=true` on the terminal pane at all **nine** sites in
      `main-3.kdl`; update `atlas/architecture.md` and `config.kdl`'s now-wrong
      scroll-indicator rationale.

## Log

- 2026-09-07 M2.5 (manual acceptance, operator-run): `yes "aaaa…"` flooding one
  tab while switching with alt+←/→ repeatedly. **No corruption observed** — no
  stray escape fragments, no colour bleeding into subsequent lines, no cursor
  landing on the wrong row. Run against PID 61128, started 21:37, i.e. after the
  19:13 build, so the new binary was genuinely under test (checked rather than
  assumed; the other eleven `pair term` processes on the box are older couch
  actors still on the previous binary).

  **What this does and does not accept (corrected 2026-09-07, BR-36).** It
  accepts the SINGLE-WRITER ENVELOPE: `redrawTab` racing the output pump under
  load is exactly the two-writer interleaving M2 removes, and no corruption
  appeared. It does **not** exercise the gate's defer-and-owe path, and an
  earlier version of this entry wrongly implied it did. Nothing paints in M2 —
  the strip arrives in M3 — so the only console-originated write is
  `reportError`, and no action failed during the run. With no paint requested
  while the stream is mid-sequence, there is nothing to defer.

  The gate is covered deterministically instead
  (`TestPaintDefersMidSequenceAndIsOwed`, `TestGateIsNotFedOurOwnWrites`,
  `TestTakeoverResetsTheGateAndDropsTheOwedPaint`,
  `TestTheGateSeesExactlyWhatTheTerminalSees`), and becomes manually reachable
  at M3, when a repaint under load is a thing the operator can actually cause.
  The first, idle-pane pass is weaker still: an idle child never leaves the
  stream mid-sequence at all.


- 2026-09-07: closed M1 — Full `make test` green (exit 0) and `make test-smoke` green (exit 0, all three probes). BR-21 fixed at the class: the probe moved to probes/zellijscrollregion, the home atlas/index.md already names, where make test-smoke runs every directory -- so it is covered by existing rather than by the two hand-maintained lists I had added to compensate; verified test-smoke picks it up. BR-19 fixed: the probe reader goroutine shared an unsynchronised strings.Builder with the verdict, now a mutex-guarded buffer, `go run -race` clean and still reproducing DECSTBM HONORED. BR-20 fixed: frame[len(frame)-3000:] panicked whenever the pty produced under 3000 bytes -- exactly the failed-session path the probe must survive to report -- now tailOf. BR-16 remains fixed (tracked reproducible apparatus). BR-17 swept (every superseded restatement, not just the named site). BR-4 stderr half closed in the plan (runZellij gains a stderr io.Writer; M2.3b asserts it). Both Minors fixed. M1 core evidence unchanged: the lift is proven a MOVE by the behavioural couch tests passing UNEDITED, and no second implementation of the region escape exists outside hostty.; review verdict: FIX-THEN-SHIP
### 2026-09-06

Third of three connected pieces — see `#198` for the layout3 flag that makes
this pane reachable under couch, and `#200` for making the strip clickable.
Sequencing note: this issue touches `termcmd` and `hostty`, not couch, so it
does **not** gate couch-lite's close.

**Why now, and why not in August** — the couch-lite journey is the reason this
is cheap today and was not a month ago:

- **2026-08-21** couch is specified as an Erlang-style actor cluster on one
  host, with brain as an always-home advisor (`workshop/projects/couch.md`).
- **2026-08-22** the layout is pinned to layout2, with the rationale "couch owns
  terminal switching now, so layout3's third pane — pair's own user terminal —
  is the layer couch replaces" (`couchcore/couch.go:427`).
- **2026-09-02** `#170` rescopes to **couch-lite**: a switcher over live coding
  sessions, not an actor cluster. Admission — fleet capacity and incumbency,
  its cross-repo provider dependency, its stateful fake and its live conformance
  target — is deleted whole.
- **2026-09-03** scope event: couch-lite narrows to one remaining issue
  (`#182`), with papercuts dropped and `#179`/`#180` absorbed into `#181`.
- **2026-09-05/06** `#172` makes the status row *interactive*, and the mouse
  arbitration under it is corrected four times in a row (BR-16 the missing
  re-assert, BR-22 the write, BR-26 the observation's shape, BR-33 the belief —
  the last from the operator's `#196`).

Two consequences follow from that arc, and both point the same way:

**The pin's rationale did not survive the rescope.** "couch replaces your
terminal" was an actor-cluster-era claim, made when couch was going to own
everything on the host. couch-lite switches *agent sessions*; it does not hand
the operator a shell at their cwd. So the reason to withhold layout3 was
undermined by `#170` independently of anything the operator changed their mind
about — which is worth recording, because it means the reversal is a correction,
not a preference reversal.

**`#172` is what makes the strip cheap.** Before it, a reserved interactive row
was an idea; after it, it is working code with its mouse-mode hazards found and
fixed. Building the same surface one level down is now a lift plus a renderer.

**Operator's own reasoning for the reversal** (2026-09-06), recorded because
`#198`'s Spec asked for it: layout2 was chosen for three reasons — keep it
simple initially; uncertainty about whether the right pane should host a web
browser as well as a terminal (cmux-style); and reservations about the right
pane's quality, specifically zellij tabs, the title mechanism, and mouse
support. All three have moved: couch-lite is close to done, the browser idea is
dropped so the pane is terminal-only, and the third is what this issue fixes —
the observation being that couch's status bar generalizes into a tab bar the
right pane can manage itself.

**Note on `#194`** ("Toggle a terminal pane in layout2", `open`): it argues
layout3's permanent third pane is the wrong posture and proposes an on-demand
terminal in layout2 instead. The operator's second reason above takes the
opposite position — constant access at the same cwd is the point. `#194` is
**superseded and closed `wontfix`** by operator decision the same day; its
floating-pane design work (the `alt+c` create-once/show-hide pattern, the
tab-wide floating-visibility hazard, the frame-drag hazard behind `#123`'s move
into the tiled tree, and the role-scoped `alt+t` analysis) is preserved in that
file and stays valid if the want returns in another form.

### 2026-09-06 — measured: the pane frame is unreachable, and two scope decisions

**Probe, not reasoning.** A shell in the right pane enabled SGR mouse reporting
(`?1000` + `?1006`) and printed every byte the terminal sent, with clicks inside
the text area bracketing the frame clicks so silence could not be confused with
"reporting was never on". Operator-run, layout3, real workbench:

| click target | report |
|---|---|
| inside the pane's text area | `^[[<0;COL;ROWM` — arrives |
| pane frame, **top** border/title | **nothing** |
| pane frame, **bottom** border | **nothing** |
| inside the text area again | arrives |

So the rule `#172` established holds one level down: **a process can only
receive mouse events on a surface it owns.** couch's row is clickable because
couch owns the host terminal; `pair term`'s strip will be clickable because
`pair term` owns its pane content. zellij draws the frame outside that content
and forwards nothing, so the frame cannot be made interactive by us at all.

Two consequences:

1. **The strip is the only route to `#200`.** Clickable tabs are not reachable
   by improving the frame — there is no channel. This raises the strip from
   "nicer title mechanism" to "the only surface that can carry the interaction".
2. **Frameless costs nothing we could otherwise have had.** The frame's
   remaining value was display-only, and the operator confirmed they do not
   split the right pane (below), which retires the divider job.

**Scope decision — no splits; tabs are the mechanism.** Operator: *"I don't
split really. if I need multiple window, I use tab, that's why getting tab
experience well is important."* This retires the divider job that caused the
2026-07-27 frameless revert, and it reframes the strip: it is not chrome, it is
the primary navigation surface for the pane. Plan item 0 is answered.

**Scope decision — frameless is in scope**, with the operator's own caveat that
it walks back if a frame job turns out to be load-bearing (*"if for example
SCROLL: 0/1000 means having frame, so be it"*). The strip must therefore carry
scroll position from `ptychild` before `borderless` flips, which the Done-when
already requires.

### Placement: bottom, and why not top

The strip goes at the **bottom** of the pane. This reverses an earlier operator
preference for top, on a mechanism finding rather than taste.

The reservation is **not symmetric between edges**, because the child is handed
a terminal one row shorter and never told (`atlas/couch.md`, "a reservation, not
compositing"). The child therefore addresses rows `1..N-1`:

- **Bottom** — the child's `1..N-1` lands on host `1..N-1` and the reserved row
  is `N`. The child cannot address the strip at all; only a display *clear*
  disturbs it, which is exactly what `TakeRowDirty` already handles.
- **Top** — the child's `1..N-1` still lands on host `1..N-1`, but the strip
  would need the child at `2..N`. **The child's row 1 IS the strip.** A plain
  shell would be fine (it only scrolls, and DECSTBM confines that), but every
  full-screen app — `nvim`, a pager — draws over the strip continuously rather
  than once.

Top is achievable with origin mode (`\x1b[?6h`), which makes the child's cursor
addressing region-relative. But nothing tracks `?6` today (`ptychild.Screen`
handles `1049/1047/47`, `1000/1002/1003`, `1006`), children reset it in teardown
sequences, and that is precisely the mode-arbitration class that cost `#172`
four consecutive review rounds ending in the operator-filed `#196`. Bottom keeps
the lift a genuine *move*, which the Done-when demands ("couch's own status row
is unchanged in behavior — a regression there means it was a rewrite").

Top remains reachable later as an additive DECOM arbitration over a working
strip, not a redesign. Recorded so the option is not lost.

## Estimate

**7.52 hr**, of which **3.98 is already measured and spent** on M1 — so the
forward-looking figure is **~3.5 hr for M2–M4**.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: cross-cutting-refactor design=0.30 impl=3.68
item: tui-screen design=0.25 impl=0.26
item: cross-cutting-refactor design=0.12 impl=0.14
item: smaller-go-module design=0.15 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: tui-screen design=0.25 impl=0.26
item: smaller-go-module design=0.15 impl=0.14
item: atlas-docs design=0.12 impl=0.12
item: milestone-review design=0.10 impl=0.42
item: ux-iteration design=0.30 impl=0.12
design-buffer: 0.15
total: 7.52
```

| item | milestone | what it covers |
|---|---|---|
| cross-cutting-refactor (3.98) | **M1, MEASURED** | the `hostty.Reservation` lift — see below |
| tui-screen | M2 | the gate + single-writer envelope, all three rules |
| cross-cutting-refactor | M2 | subprocess routing, five `RunZellijAction` sites |
| smaller-go-module | M3 | `RenderStrip`, mirroring `RenderStatusRow` |
| cross-cutting-refactor | M3 | `rowtext` extraction from `couchtty`'s unexported pair |
| tui-screen | M3 | wiring the strip into `termcmd`, repaint on `batch.RowDirty` |
| smaller-go-module | M4 | `borderless=true` + the layout-enumeration assertion |
| atlas-docs | M4 | atlas entry + `config.kdl`'s now-wrong scroll rationale |
| milestone-review | M2–M4 | three boundary reviews |
| ux-iteration | M3–M4 | one appearance round on the strip (see below) |

**Design discount ×0.2** applied to every primitive the plan already settles,
which after six plan-quality rounds is most of them: all three gate rules named,
all five writers derived, the repaint trigger corrected to `batch.RowDirty`, and
`rowtext`'s shared home chosen. No discount on the `smaller-go-module` rows,
whose design hours are already near zero.

**The M1 row is measured, not estimated, and it is 6× its primitive.** The lift
itself was small; the cost was three hand-maintained consumer lists the plan did
not name (`conceptPlans`/`conceptInventory`, `#146`'s core-concepts row,
`artifactpath`'s inventory) plus a deadlock the suite caught. It is itemized at
its measured value rather than its predicted one so the total reconciles against
reality — and it is exactly the `consumer-set-not-derived` family the plan now
carries a rule for. **M2–M4 assume that rule holds.** If a fourth hidden list
appears, this estimate is low.

**One `ux-iteration` round is budgeted** (estimate-quality F6). This issue
exists because of operator taste — its origin is *"the way we change tab title
is not great"* — the deliverable is a visible surface, and three steps are live
manual acceptance (M2.5, M3.7, M4.3). The operator also pre-reserved a walk-back
on frameless (*"if SCROLL: 0/1000 means having frame, so be it"*). One
appearance round on a bespoke tab strip is the expected case, not the
exceptional one, so it is budgeted rather than absorbed silently.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is flagged `[stale]`
(recalibration tracked in `#127`), so the per-primitive hours are provisional.

## Revisions

### 2026-09-07 — reopened; the design question is settled by measurement

Punted 2026-09-06 to do the performance work first (`#208`, now merged).
Reopened at operator request.

**What changed since the punt.** The one thing that could have killed this
design is now measured: **zellij honors a pane process's DECSTBM**. couch's
reserved row works on the *host* terminal; the right pane's writes go through
zellij's emulator, and nothing had tested whether the scroll region survives
that. It does — 200 lines scrolled in rows 1..21 while the reserved bottom row
held its paint. So `couchtty.Reserve`/`PaintRow` transfer to the pane unchanged,
which is exactly what M1 assumes. Detail and method in the plan, finding 5.

**What the issue turns out NOT to be.** `Alt+t` is already not a zellij tab —
`config.kdl:114` forwards raw `ESC t` to the focused pane and
`handleTerminalChord` catches it into `mux.newTab()`. `terminalMux` already
carries the tab list, the active index, switching and closing. **Multiple
terminals in the right pane, switched on demand, is built and shipping.** The
missing piece is only the display, whose sole surface today is `rename-pane`
packing every tab into one string. This is a rendering change over a working
multiplexer, not new tab machinery.

**Ownership decided.** The strip belongs to `pair term`, not couch. couch's
child is the whole zellij session, so it cannot render into a pane zellij owns;
and a tab bar living in couch would leave standalone pair without one, which
`couch must not degrade pair` forbids. The plan's existing design already
assumed this and is unchanged.


### 2026-09-06 — the scroll-position Done-when is struck; frameless is in scope

**Struck:** *"The strip displays scroll position for its own pane, sourced from
`ptychild` rather than from zellij — the fact that makes the frameless follow-on
possible is demonstrated, not merely argued."*

The Spec's premise for that bullet is false, and it was checked rather than
reasoned about. The Spec argued `pair term` "owns the pty and the ring, so it
already holds its own scroll position without asking zellij anything." It does
not:

- `termcmd/run.go:453-463` forwards every wheel tick to zellij —
  `RunZellijAction("scroll-up")` / `("scroll-down")`. Scrolling the right pane
  is zellij's scrollback, driven by us.
- `ptychild` exposes `Snapshot`, `Replay`, `ReplaySafeEnd`, `ReplayThrough` — a
  **replay ring for tab switching**. There is no viewport and no scroll offset
  anywhere in our code.

So the strip cannot *source* a scroll position; making one would mean building a
scrollback viewport inside `pair term` (intercept the wheel, own an offset into
the ring, render the scrolled view). That is a feature, not a readout, and it is
not this issue.

**Operator decision (2026-09-06):** the readout is not worth it — *"we don't
need the text SCROLL: 0/1000, I don't think that's a deal breaker actually."*
So the third frame job is **dropped**, not replaced.

**Consequence: frameless moves INTO this issue** rather than being a follow-on,
because all three jobs are now accounted for:

| frame job | disposition |
|---|---|
| divider between split halves | retired — the operator does not split (`Log`, 2026-09-06) |
| `#118` tab title | replaced by the strip — this issue |
| scroll-position indicator | dropped by operator decision, above |

**Added to `## Done when`:** the right pane in layout3 takes `borderless=true`,
and the strip carries the pane's identity in its place.

### Also added: a prerequisite the Spec did not name

`termcmd` writes to stdout from **two goroutines** — `copyActiveOutput`
(`run.go:702`) and `redrawTab` (`run.go:1023`, called from three tab-switch
sites). `atlas/couch.md` records why that cannot survive a reserved row: *"a pty
read boundary falls wherever the kernel puts it, so a paint written between two
chunks can land inside one of the child's escape sequences"*, which is why
couch made `Console.Run` the only writer. Painting a strip into a two-writer
stream reproduces the bug couch already paid for, so single-writer discipline in
`termcmd` is a milestone of this issue, not cleanup after it.

### 2026-09-06 — deprioritized; the cost/benefit as it actually stands

Planned to `workshop/plans/000199-…-plan.md` (four milestones, seven open
plan-gate findings), then **punted before implementation** on the operator's
call. The design work is preserved; what follows is why it was not worth
starting now, so the decision does not have to be re-derived.

**The operator's own accounting of the payoff**, and what each is actually
worth:

| wanted | delivered by | status |
|---|---|---|
| 1. switch tab with the mouse | `#200` | **real** — and the frame probe proved the strip is the *only* possible route |
| 2. stop inadvertent pane resizing | disputed — see below | **undetermined** |
| 3. robust tab rename; colorizable tabs later | `#199` M3 | **real**, and the cheapest of the three |

**Benefit 2 is the one that decides this, and it is not yet settled.** There are
two resize paths in zellij with different dependencies:

- **ctrl+wheel** → `ResizeScrollUp/Down`, resizing the FOCUSED pane regardless
  of pointer position, needing no frame. Investigated 2026-07-28 (see the
  operator's memory note): `advanced_mouse_actions false` does not gate it
  despite zellij's docs, and *"pair cannot fix this itself — zellij consumes
  wheel events before the pane's process sees them."* Frameless does nothing
  here.
- **dragging a pane border**, which needs a grabbable border. The operator's
  evidence: the draft pane is `borderless=true` and its height **cannot** be
  mouse-resized. So frameless does block this path.

Both are real; which one the operator actually triggers is unmeasured. An
earlier claim in this discussion that benefit 2 was simply not deliverable was
**too broad** — it reasoned from the ctrl+wheel path alone and the operator
correctly pushed back from the draft-pane evidence.

**The cheap experiment that should precede any of this work:** set
`borderless=true` on the terminal pane in `main-3.kdl` (nine sites) and restart.
No strip, no lift, no refactor. It answers three questions at once — whether the
resize stops, whether losing the tab title is painful enough to justify the
strip, and how the pane reads frameless. **Do this before reopening the issue.**

**What made the program expensive**, beyond the filed scope:

- `termcmd` writes the host from two goroutines, so single-writer discipline is
  a prerequisite milestone (`atlas/couch.md` records why a reserved row cannot
  survive two writers).
- The Spec's scroll-position premise was false (see `## Revisions`), so
  frameless costs the indicator outright rather than relocating it.
- `#200` is not "add a click handler": it consolidates a mouse-mode belief that
  has been wrong four times, and a fifth face surfaced the same day — see
  `#200`'s Log, where an accidental probe broke agent-pane copy-on-select and
  **no in-session gesture recovered it**, only a couch restart.

**Priority instead:** the performance issues `#201`/`#202`/`#203`, which affect
every keystroke rather than one pane's chrome.

**If reopened, start here:** run the borderless experiment; if benefit 2
survives it, the four-milestone plan is written and its seven plan-gate findings
are the first work item.
