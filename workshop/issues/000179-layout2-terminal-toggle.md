---
id: 000179
status: open
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-03
estimate_hours:
---

# Toggle a terminal pane in layout2

## Problem

`layout2` is agent above draft, with no terminal. Checking anything —
`git status`, a test run, a file — means leaving the workbench or running it
through the agent, which spends a model turn on something the operator can read
in a second.

`layout3` already provides a terminal, but as a **permanent third pane** taking
50% of the width. That is a different working posture, not a quick check: the
operator wants the terminal *when they want it* and the full-width agent the
rest of the time.

## Spec

**A terminal reachable by one key from the agent/draft panes, at the same cwd,
and dismissable by the same key.** A slimmer `layout3`: the terminal exists on
demand rather than permanently.

### The chord is not free — decide this first

`alt+t` is already bound: `{Chord: ChordAltT, Role: PaneRoleRightTerminal, Help:
"new terminal tab"}` (`workbenchshortcut/shortcut.go:156`), handled at `:206` as
`ActionNewTab`. It is **role-scoped**, so it is free in the agent and draft
panes and taken in the terminal pane. Two readings, and the plan must pick one:

- **Role-scoped (recommended).** `alt+t` in agent/draft toggles the terminal; in
  the terminal it keeps meaning "new terminal tab". Consistent with how the
  chord table already works, and nothing existing changes. Cost: one key with
  two meanings, resolved by where focus is.
- **Global.** `alt+t` always toggles, and `ActionNewTab` moves to another chord.
  One meaning everywhere, at the price of retraining an existing binding.

Either way `ChordAltT` already has both encodings (`shortcut.go:297`), so no
terminal-protocol work is needed.

### Behavior

- **Toggle, not just show.** `alt+t` from agent/draft opens the terminal *and
  focuses it*; `alt+t` again returns focus to the pane the operator came from.
  Coming back to where you were is the point — a key that only opens leaves the
  operator hunting for the way back.
- **Same cwd**, which is the workbench's cwd — the same one `layout3`'s terminal
  pane starts in.
- **The shell survives being dismissed.** Dismissing hides the pane; it does not
  kill the shell. A half-typed command, a running process, and scrollback are
  all still there on the next `alt+t`. This is the difference between a toggle
  and a spawn, and it is the requirement most likely to be lost.

### Mechanism: a floating pane, which pair already does three times

**This is the `alt+c` review-pane pattern with `pair term` inside it.** The
workbench already runs full-screen floating panes — `alt+h` (`pair keys` at
`100%`x`70%`), `alt+l` (changelog at `100%`x`100%`), `alt+c` (review) — all
declared in `zellij/config.kdl:159-200`. So this needs no new mechanism, only a
fourth instance of an established one (`ARCH-DRY`).

**It also deletes the `exact_panes` problem, which was this issue's main risk.**
Floating panes do not participate in the tiled tree, so the pane count the
`swap_tiled_layout` rungs match on never changes. The proof is already in
production: `alt+h` and `alt+l` open full-screen floating panes today and the
`alt+up`/`alt+down` ladder keeps working. No `main-2.kdl` variant set, no
parallel rungs.

**Follow `alt+c`, not `alt+h`/`alt+l`.** Those two use `Run … close_on_exit
true`, which spawns a fresh pane per press — right for a pager, wrong for a
terminal, where the whole requirement is that the shell survives dismissal.
`alt+c` instead creates once and then flips visibility, and `config.kdl:178-180`
records how: query `are-floating-panes-visible`, then `show-floating-panes` /
`hide-floating-panes` — *"never the toggle-floating-panes footgun"*, because
`toggle-floating-panes` opens a pane when none exists and so cannot express
"show the one I already have".

**Reuse `pair term`, not a bare shell** — `layout3`'s pane is `exec pair term`
(`main-3.kdl`), carrying resize watching, clip integration (`clipcmd`) and the
pane-role plumbing (`termcmd/run.go:552`). A raw `sh` would be a second, subtly
different terminal.

### Two hazards this inherits, both already recorded in the tree

**1. The floating layer is shared, and its visibility is tab-wide.**
`show-floating-panes` / `hide-floating-panes` act on *every* floating pane in
the tab. With a review pane (`alt+c`) and a terminal both floating, showing one
shows the other, and hiding either hides both. That is a genuine collision
between two toggles that are meant to be independent, and the plan must say how
they coexist — the options are per-pane focus/stacking, making the terminal
mutually exclusive with review (state a rule), or a different container. **Do
not discover this at integration time.**

**2. A floating pane can be dragged off position.** `main-3.kdl:18-24` is
explicit: the terminal *was* floating and moved into the tiled tree for `#123`
precisely because *"zellij 0.44.3 lets any floating pane be dragged off position
by its frame with no config gate"*. That verdict was about a **permanent**
workbench pane, and it does not automatically carry to a transient full-screen
overlay — `alt+h` and `alt+l` accepted the same risk. Still, check whether a
borderless/frameless floating pane removes the drag handle, since the recorded
cause is the *frame*.

**`layout3` is untouched.** This is a `layout2` affordance; someone who wants a
permanent terminal still asks for `layout3`.

## Done when

- From the agent or draft pane in `layout2`, `alt+t` opens a terminal at the
  workbench cwd and focuses it.
- `alt+t` from that terminal returns focus to the pane it was invoked from.
- A command typed but not run, and a long-running process, both survive a
  dismiss/reshow cycle — asserted, not assumed.
- The draft-pane rung ladder (`alt+up`/`alt+down`) still steps correctly with
  the terminal shown and hidden — expected to be free via the floating layer,
  asserted anyway because it is the assumption the mechanism rests on.
- `alt+c`'s review pane and the terminal each toggle to the state the operator
  asked for, with the shared-visibility rule from hazard 1 stated and tested —
  not left to whichever was pressed last.
- `toggle-floating-panes` appears nowhere in the implementation; visibility is
  driven by `are-floating-panes-visible` + explicit show/hide.
- `alt+t` inside `layout3`'s terminal still opens a new terminal tab (or, if the
  global reading is chosen, its replacement chord does, and the help text is
  updated).
- `layout3` behaves exactly as before.

## Plan

- [ ] Decide role-scoped vs global `alt+t`; record the reason in `## Log`.
- [ ] Settle the shared-floating-layer rule against `alt+c` (hazard 1) before
      writing the binding — it decides the shape.
- [ ] Floating `pair term` pane, created once, shown/hidden via
      `are-floating-panes-visible` + explicit show/hide, following `alt+c`.
- [ ] Focus-return to the invoking pane on dismiss.
- [ ] Check whether a frameless floating pane is drag-immune (hazard 2).
- [ ] Rung-ladder assertions with the terminal shown and hidden.

## Log

### 2026-09-03

Requested as "a slimmer version of --layout3": a terminal at the same cwd,
reachable by `alt+t`, for quick checks rather than as a permanent pane.

Two constraints found while scoping, both of which shape the design rather than
just informing it: `alt+t` is already bound in the terminal pane (role-scoped,
so free elsewhere), and swap layouts match on `exact_panes` — `main-3.kdl`'s
`*-split` variants exist because adding one pane forced a parallel rung set.
A show/hide toggle hits that same constraint if it is implemented as a swap.

### 2026-09-03 — floating pane, not a layout swap

The operator pressed `alt+p` (zellij's Multiple Pane Select) by accident, saw
its floating overlay, and asked whether the terminal could be that, full-screen.
It can, and it is better than the swap-layout design above: floating panes do
not enter the tiled tree, so the `exact_panes` constraint that drove the
original mechanism section simply does not apply — proven in production by
`alt+h` and `alt+l` coexisting with the rung ladder today.

`zellij action` confirms the surface: `toggle-floating-panes`,
`show-floating-panes`, `hide-floating-panes`, `are-floating-panes-visible`,
`change-floating-pane-coordinates`.

The design question is no longer "how do we show a pane" — pair answered that
three times already — but how the terminal shares the tab-wide floating
visibility with the review pane. That is hazard 1, and it is the part worth
thinking about before any code.
