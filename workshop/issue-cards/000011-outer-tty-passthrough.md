---
id: '000011'
status: done
created: 2026-05-03
updated: 2026-05-03
actual_hours: N/A
---

# Outer-PTY passthrough for OSC notifications

## Problem

When pair runs inside an outer wrapper (e.g. cmux) that watches the agent's PTY stream for OSC 9 / OSC 777 escapes to surface "agent needs attention" indicators, zellij silently eats those escapes. Empirically verified: OSC 9 and OSC 777 from inside a zellij pane do NOT reach cmux's stream watcher; the same escapes from a plain shell inside cmux do.

Zellij 0.44 forwards BEL (verified via Apple Terminal — `printf '\a'` rings whether zellij is in the chain or not), but cmux watches OSC, not BEL. So the BEL passthrough doesn't help for cmux integration.
