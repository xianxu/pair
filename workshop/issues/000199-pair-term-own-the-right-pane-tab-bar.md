---
id: 000199
status: open
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
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

- [ ] Decide plan items 1-3 (rename-pane survival, row budget, placement).
- [ ] Lift `Reserve`/`PaintRow` into `hostty`; repoint couch at them and verify
      couch's row behavior is byte-identical.
- [ ] Render the strip in `termcmd` from its existing tab model; wire
      `TakeRowDirty` repaint.
- [ ] Implement the `rename-pane` decision; check the named consumers.
- [ ] Verify in a real layout3 workbench, including a split right pane and a
      full-screen child (`nvim`, a pager) clearing the display.

## Log

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
