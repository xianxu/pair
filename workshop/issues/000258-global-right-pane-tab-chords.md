---
id: 000258
status: open
deps: [pair#227, pair#243]
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Use global right-pane tab chords everywhere

## Problem

Follow up to #227 and the from-anywhere tab control introduced by #243. Pair now
has global chords `M-S-left`, `M-S-right`, and `M-S-t` for changing or creating a
right-pane terminal tab from another pane. The same functions still use the
right-pane-local chords when focus is already inside the right pane. Under a
full-screen child, #227's passthrough correctly sends those local chords to the
child, so the tab operation does not happen.

This leaves the same operation with two delivery paths and makes it fail exactly
where the global escape chords are meant to help.

## Spec

Use the three global chords as the common command surface, including when the
right pane itself is focused:

- `M-S-left` selects the previous right-terminal tab.
- `M-S-right` selects the next right-terminal tab.
- `M-S-t` creates a new right-terminal tab.

When the operator invokes these chords from the right pane, Pair must handle the
global command before child/full-screen passthrough. The resulting tab action must
not be forwarded to the active application. The same chord/action mapping should
be shared by from-anywhere and right-pane-focused dispatch (ARCH-DRY), while the
pure shortcut decision remains separate from the effect that changes tabs
(ARCH-PURE).

Keep #227's ordinary right-pane application chords unchanged: role-scoped chords
such as the agent's `Alt+Left`/`Alt+Right` continue to reach a full-screen child,
and `Alt+k` remains the focus-left escape. This issue changes only the three
explicit global tab chords.

Recommendation: adopt this common global surface. It gives the operation one
stable meaning regardless of focus and makes the escape behavior reliable for
full-screen applications. The tradeoff is that a full-screen application cannot
claim these three chords; that is intentional because they are Pair's global
right-pane navigation commands.

## Done when

- With the right pane focused on a shell, `M-S-left`, `M-S-right`, and `M-S-t`
  select/create the right-terminal tab through the global command path.
- With the right pane focused on a full-screen app, the same three chords still
  select/create the right-terminal tab and do not reach the child.
- Invoking the same chords from the draft or agent remains unchanged and produces
  the same right-pane actions.
- Ordinary role-scoped application chords retain #227 behavior, including
  full-screen passthrough and `Alt+k` focus-left.
- Tests tie all three focused-pane deliveries to the global chord/action mapping,
  preventing a future role-scoped fallback from reintroducing the split behavior.
- Help/key metadata and atlas vocabulary describe one common global tab-command
  surface.

-

## Plan

- [ ] Trace the current right-pane-focused and from-anywhere delivery paths and
  identify the shared dispatch point.
- [ ] Route the three tab operations through the global chord mapping when the
  right pane has focus, before full-screen passthrough.
- [ ] Add shell/full-screen and all-origin regression tests for previous, next,
  and new-tab commands, plus a guard that each delivery remains global.
- [ ] Update help/atlas documentation and run focused plus full verification before
  closing through the SDLC review gate.

- [ ]

## Log

### 2026-09-15

Filed as a follow-up to #227/#243. The operator observed that the global
`M-S-left/right/t` tab commands work from elsewhere but not when the right pane
itself is focused. Recommended using the same global chords in both contexts so
full-screen passthrough cannot consume the tab controls.

### 2026-09-15
