---
id: '000145'
status: done
started: 2026-08-21T12:40:25-07:00
created: 2026-08-20
updated: 2026-08-25
actual_hours: 8.51
---

# couch: spawn and registry

## Problem

Nothing knows what agent sessions exist. Pair drives one session; the outer
shell is cmux or ghostty tabs, which have no notion of what an agent session
*is*, so there is no way to ask what threads are open, bring a dormant one back,
or stop two sessions from silently sharing one working tree.

Before any switching UI is worth building, couch needs the thing underneath it:
a registry of named actors and a deterministic way to bring one up.
