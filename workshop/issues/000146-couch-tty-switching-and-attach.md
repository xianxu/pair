---
id: 000146
status: open
deps: []
github_issue:
created: 2026-08-21
updated: 2026-08-21
estimate_hours:
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

- [ ] **Smoke step 1: host one real `pair` + claude child.** No switching. This
      proves the hard part — pair-in-a-pty-under-couch, resize included — and
      answers the layering fork below. Trivial shells would prove only the easy
      half.
- [ ] **Smoke step 2: two real children, switching between them.** tty routing
      and repaint are then the only new variables, so a break localises.
- [ ] tty passthrough to one child; `ctrl-space` interceptor.
- [ ] Reserved status row via a one-row-shorter child pty.
- [ ] Per-child ring buffer + repaint on attach.
- [ ] Up-one-level navigation, verified from a mid-output child.
- [ ] Detach/reattach without killing children; child-exit lands in the TUI.
- [ ] Direct switch by name/number, LLM-free.

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
