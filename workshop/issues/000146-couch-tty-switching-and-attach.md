---
id: 000146
status: working
deps: []
github_issue:
created: 2026-08-21
updated: 2026-08-22
estimate_hours:
started: 2026-08-22T12:14:19-07:00
---

# couch: tty switching and attach

Project: `workshop/projects/couch.md` — architecture and non-goals live there;
this issue is task 2.

## Problem

With a registry of named actors (`#145`), the operator still has no way to move
between them except terminal tabs, which know nothing about what a session is.
The switching experience is what determines whether couch gets used at all: if
getting back to a known place is ever slow or flaky, the operator reverts to tabs
and everything above it is dead weight.

## Spec

**A switcher, not a multiplexer.** One operator tty attached to one child at a
time, a key-sequence interceptor, and a per-child buffer replayed on attach so
the screen is not blank on landing. Explicitly NOT built: splits, layouts,
floating panes, simultaneous rendering, a plugin system. The failure mode to
avoid is reimplementing tmux badly — the complexity there lives in compositing
panes nobody is looking at.

**One keystroke home to the root actor, from anywhere, always.** The root actor
is whatever session couch launched in — usually brain, by convention rather than
mechanism; couch can start anywhere and nothing here knows about brain
specifically. This is the single most important property in the whole project:
if it is reliable the operator roams freely because getting back is free.

**`ctrl-space` moves up one level** — child → root actor, root actor → couch's
control panel. Bare key, acts immediately, no prefix keymap and no timing
window. Double-ESC was considered and rejected: ESC is already interrupt/cancel
in Claude Code and mode-switch in nvim, and a double-tap must either delay every
legitimate ESC or forward one it cannot retract. Richer navigation lives inside
couch's TUI with typeahead rather than in a chord table — one key to memorize,
then read a screen. Direct jumps (to actor N, to the latest notifier) are
deliberately deferred until the operator catches themselves wanting one.

**Switching is deterministic and LLM-free in the critical path.** Resolution of a
fuzzy reference sits *above* the switch (`#148`); the switch itself is a direct
call. A model turn inside this path reintroduces exactly the latency that sends
the operator back to tabs, so a direct route that skips resolution entirely —
hotkey home, a numbered list — must always exist.

**Detach and reattach without killing children.** A detached actor keeps running;
its child harness stays warm. Reuse what already exists rather than writing
terminal handling from scratch: `wrapcmd`'s terminal model over
`charmbracelet/x/vt` + `creack/pty`, and `scrollbackcmd`.

**couch does not composite — it reserves a row.** The child is given a terminal
one row shorter and couch owns the last row. The child never knows, so this is a
resize rather than compositing, and it works identically in the root actor and
while attached to any child. That row carries rolling notifications, so there is
exactly one place to look. Children that redraw on resize (nvim, zellij) handle
it natively.

Notification *detail* is not drawn there and not injected into the transcript as
system messages — transcript injection would burn the LLM's context every turn.
The row says something happened; `ctrl-space` and the advisor supply the rest.

**Agent children only.** couch does not host plain shells, log tails or test
runs; the operator leaves the window for those. The project's "single terminal
window" criterion means one window for *agent* work, and this is what keeps the
switcher from drifting into general child hosting.

Attachment is an **output routing decision**, not the actor's identity — messages
addressed to the operator route to the console when one is attached, and are
simply not rendered when none is.

## Done when

- couch supervises N sessions and switches the operator tty between them.
- `ctrl-space` reaches the root actor from inside every child, including one that
  is mid-output, and is measurably instant (no model turn, no network).
- A reserved status row is visible in the root actor and in every attached child,
  and the child renders correctly at the reduced height.
- An attached child that exits lands the operator in couch's TUI with which actor
  exited and why — never on a dead pane.
- Landing on a session shows recent context rather than a blank screen.
- Detach and reattach leave children running and warm.
- A numbered/direct switch path exists that requires no natural-language
  resolution.

## Plan

Design of record: `workshop/plans/000146-couch-tty-switching-and-attach-plan.md`.
Four review boundaries; the smoke steps stay where they were sequenced (risk
first) but are folded into the milestone whose risk they answer.

- [ ] **M1 — shared pty-child core.** Extract `ptychild` (ring, replay
      query-strip, output scanner, pty child) out of `termcmd`'s multiplexer and
      migrate `pair term` onto it. Ships no couch behaviour; the migration is
      what validates the extraction (ARCH-DRY).
- [ ] **M2 — console over one child, with the reserved row.** `PtyRunner` behind
      the existing `Runner` seam (+ fake + live conformance), `couch start`
      becomes the console, `ctrl-space` interceptor, one-row-shorter child pty
      with a pinned scrolling region, and `Spawn` forced onto `pair resume
      <tag>` so a console restart reattaches instead of landing on a picker.
      **Smoke step 1** (one real `pair` + claude child, resize, nvim in and out,
      reattach across a `kill -9`) lands here.
- [ ] **M3 — many children and the panel.** Up-one-level focus, per-child ring
      replay (or a resize nudge for alt-screen children), typeahead + numbered
      direct switch, panel actions dispatching through `couchcore.Operations()`.
      **Smoke step 2** (two real children, switching, `ctrl-space` from a
      mid-output child) lands here.
- [ ] **M4 — exits, detach, and what the row says.** Child exit lands in the
      panel with actor + code, detach/reattach stays warm, notices over
      `couchcore.Enqueue`, terminal restored on every exit path including
      signals, atlas reconciled.

## Log

### 2026-08-21

Split out of the former root ticket on promotion to a project.

**Layering fork — SETTLED 2026-08-21, host `pair` whole.** The operator ran
`./bin/couch start ../pair` against `#145`'s spawn path and pair came up
correctly, so couch does **not** absorb zellij's role: the stack stays
couch → pair → zellij → claude+nvim, and a zellij inside a couch-owned pty is
just a child that redraws on SIGWINCH.

This issue's scope is therefore the narrow one: route one tty to one child at a
time, with no responsibility for what the child runs internally. Estimation is
unblocked.

### 2026-08-22

Claimed and planned. Design of record:
`workshop/plans/000146-couch-tty-switching-and-attach-plan.md`; the eight loose
steps above are regrouped into four review boundaries there, unchanged in
content.

Three decisions the plan makes that this Spec did not settle, recorded here
because they narrow scope:

- **`couch start` becomes the console; no new verb.** The CLI's dispatch table
  is asserted identical to `couchcore.Operations()`, so a console-only verb
  would need an exception to the invariant that keeps the operator's surface and
  the advisor's from drifting. `--no-console` is the loud escape hatch back to
  #145's inherit-stdio behaviour.
- **The pty-child mechanics are extracted from `termcmd`, not written twice.**
  `pair term` already is a switcher (pty tabs, a 128KB replay ring,
  redraw-from-snapshot, resize propagation). `pair term` migrates onto the
  shared package in M1 -- its existing suite is the only regression net that can
  prove the extraction faithful.
- **Detach is console-scoped, because durability is zellij's.** couch's child
  is `pair` -> a zellij *client*; the work lives in the zellij *server* session,
  which survives detached when the client dies. So the fleet already outlives a
  terminal window one layer below couch and no daemon is on the path -- `#147`'s
  transport is not a prerequisite for the Done-when's "running and warm".

### 2026-08-22 -- reattach is zellij's, and Spawn must stop hitting the picker

The operator's read of the layering corrected the plan's first answer on detach:
couch hosts `pair`, which runs zellij, so a session is *already* reattachable
beyond a console's lifespan. The durability boundary is the zellij server, not
couch's process tree, and reasoning about a couch daemon was reasoning about the
wrong layer.

What that leaves `#146` owing is determinism on the way back IN, and there is a
real gap there today: `Spawn` runs `pair --layout2` with no tag, and
`launcher.DecideLaunch` with no tag and a detached session present returns
`ActionPick` -- an fzf picker inside couch's pty. A console restart currently
lands the operator on a picker rather than on their session.

Fix folded into M2: spawn `pair resume <tag>` with `tag =
launcher.DefaultTag(<worktree root>)`, which takes the `ForcedTag` branch
(attach if live or detached, create otherwise) and skips the name prompt.
`--layout2` is dropped -- `resume` refuses a third argv element outright, and an
omitted layout flag reuses the tag's recorded layout while forcing one on a live
tag makes pair ask before recreating the workbench.

This is a deliberate slice of `#149`, not a collision: `#149` decides the tag IS
the space (durable, opaque, several per tree, names as an attribute layer) and
supersedes this derivation. `#146` needs only that re-entry is deterministic.

Also queued for the M2 smoke: `workshop/projects/couch.md` asserts "`couch stop`
is a kill, not a park." If `Stop` signals the zellij *client*, the session
detaches and the work survives -- a park. Whichever it is, the project record
gets corrected from an observation rather than left as an unverified invariant.
