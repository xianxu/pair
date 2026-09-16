---
id: 000257
status: open
deps: [pair#245]
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Preserve pane navigation exceptions for agent passthrough

## Problem

Follow up to #245. Agent-focused routing now passes most Pair workbench chords
through to the focused agent for compatibility, but `Alt+j` and `Alt+k` must
remain Pair-owned pane-navigation actions. They move focus between the left and
right panes and are the keyboard escape path from an agent pane. Passing them to
the agent would let an agent bind or consume the keys and strand the operator in
the current pane.

## Spec

Preserve the #245 agent-first shortcut policy with an explicit navigation
exception:

- `Alt+j` and `Alt+k` are intercepted while the agent pane is focused and invoke
  Pair's pane-focus movement actions.
- All other shortcuts retain #245's pass-through behavior, including the existing
  right-pane tab reservations and other agent-compatible chords.
- The exception is role-specific to agent focus. Draft and terminal pane behavior
  remains governed by their existing routing rules and #227's full-screen policy.
- Help/key metadata must describe the exception from the same binding source used
  for routing; do not add a second hand-maintained list (ARCH-DRY).

This is a narrow compatibility correction, not a return to a broad reserved-key
list. Keep the routing decision pure and testable, with the wrapper/pump retaining
ownership of the two focus actions (ARCH-PURE).

## Done when

- With the agent pane focused, `Alt+j` and `Alt+k` move pane focus and do not reach
  the agent process.
- Representative non-navigation agent chords still pass through unchanged,
  including split escape sequences and paste handling where applicable.
- The behavior is asserted at the production wrapper/pump boundary, not only at a
  pure shortcut-table unit test.
- Draft and right-terminal focus behavior remains unchanged, including the
  #227 `Alt+k` right-pane escape behavior and #243 global tab chords.
- Help and binding metadata identify `Alt+j`/`Alt+k` as the two agent-focus
  navigation exceptions.

-

## Plan

- [ ] Confirm the current #245 routing and identify the shared binding/help source
  that should own the two exceptions.
- [ ] Add the agent-role `Alt+j`/`Alt+k` navigation exceptions and production-path
  regression tests, preserving split input and paste behavior.
- [ ] Update generated/help surfaces and focused-pane documentation.
- [ ] Run focused wrapper/routing tests and the relevant full suite, then close
  through the SDLC review gate.

- [ ]

## Log

### 2026-09-15

Filed as a follow-up to #245. The operator refined “pass most shortcuts through”:
`Alt+j` and `Alt+k` remain Pair-owned pane-focus movement chords for agent focus.

### 2026-09-15
