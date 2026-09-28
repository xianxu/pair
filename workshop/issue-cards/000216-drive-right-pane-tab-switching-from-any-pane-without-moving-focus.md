---
id: '000216'
status: done
started: 2026-09-09T10:45:14-07:00
created: 2026-09-08
updated: 2026-09-09
estimate_hours: 1.60
actual_hours: 1.80
---

# drive right-pane tab switching from any pane, without moving focus

## Problem

`Alt+Shift+Left/Right` in the draft pane is not worth its keys. The operator
wants those chords to **switch the right pane's tabs from wherever focus
happens to be, and leave focus alone**:

> this allows me to check on status of right pane, switch tabs, while keep
> typing here in the draft pane (which is the command center).

That last clause is the requirement, not a nicety. The draft pane is where the
operator composes; a chord that switches tabs by *moving focus there and back*
would interrupt exactly the typing it is meant to preserve, and would flicker
the cursor through a pane the operator is not looking at.

Today tab switching is `Alt+Left`/`Alt+Right`, handled by `pair term` —
`handleTerminalChord` → `previousTab()`/`nextTab()` — and it only works when the
right pane already has focus.
