---
id: 000258
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
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
