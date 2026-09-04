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

### Mechanism

**Reuse `pair term`, not a bare shell.** `layout3`'s terminal pane is
`exec pair term` (`main-3.kdl`), which is the existing wrapper with resize
watching, clip integration (`clipcmd`), and the pane-role plumbing
(`termcmd/run.go:552`). A raw `sh` would be a second, subtly different terminal
(`ARCH-DRY`).

**Prefer zellij's own hide/show over a layout swap if it can express this.** The
rung ladder already uses swap layouts, and `main-3.kdl` documents their sharp
edge: a swap layout matches on `exact_panes`, so adding a pane changes which
rungs apply — `main-3.kdl` needed a whole parallel set of `*-split` variants for
exactly that reason. A toggle that changes the pane count from 2 to 3 collides
with the same constraint. Check zellij 0.44.3's floating/hidden pane actions
first; if the answer is swap layouts, the plan must say what happens to the
`minimized`/`small`/`third` ladder while the terminal is up, and `main-2.kdl`
grows a parallel variant set.

**`layout3` is untouched.** This is a `layout2` affordance; someone who wants a
permanent terminal still asks for `layout3`.

## Done when

- From the agent or draft pane in `layout2`, `alt+t` opens a terminal at the
  workbench cwd and focuses it.
- `alt+t` from that terminal returns focus to the pane it was invoked from.
- A command typed but not run, and a long-running process, both survive a
  dismiss/reshow cycle — asserted, not assumed.
- The draft-pane rung ladder (`alt+up`/`alt+down`) still steps correctly with
  the terminal shown and hidden.
- `alt+t` inside `layout3`'s terminal still opens a new terminal tab (or, if the
  global reading is chosen, its replacement chord does, and the help text is
  updated).
- `layout3` behaves exactly as before.

## Plan

- [ ] Decide role-scoped vs global `alt+t`; record the reason in `## Log`.
- [ ] Determine whether zellij 0.44.3 can hide/show a tiled pane without a
      layout swap; if not, design the `main-2.kdl` variant set against the
      rung ladder.
- [ ] Toggle + focus-return, reusing `pair term`.
- [ ] Rung-ladder tests with the terminal shown and hidden.

## Log

### 2026-09-03

Requested as "a slimmer version of --layout3": a terminal at the same cwd,
reachable by `alt+t`, for quick checks rather than as a permanent pane.

Two constraints found while scoping, both of which shape the design rather than
just informing it: `alt+t` is already bound in the terminal pane (role-scoped,
so free elsewhere), and swap layouts match on `exact_panes` — `main-3.kdl`'s
`*-split` variants exist because adding one pane forced a parallel rung set.
A show/hide toggle hits that same constraint if it is implemented as a swap.
