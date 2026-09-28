---
id: '000158'
status: done
started: 2026-08-31T20:07:23-07:00
created: 2026-08-31
updated: 2026-09-01
estimate_hours: 5.69
actual_hours: 3.30
---

# couch: actor notifications and attention routing

## Problem

Couch already detects a terminal bell from an inactive actor, marks that actor
in its reserved status row, and shows a bell on the thread row. The signal has
no message, however, and the hierarchical switcher cannot explain why an actor
wants attention. Pair already interprets agent-specific output and emits
actionable OSC notifications to its outer TTY, but Couch currently treats those
bytes only as terminal output. An operator can therefore receive an outer
terminal badge without getting a Couch-local inbox or a one-keystroke route to
the responsible actor.
