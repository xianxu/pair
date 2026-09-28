---
id: '000081'
status: done
created: 2026-06-26
updated: 2026-06-26
actual_hours: 0.02
---

# session retro tooling friction

## Problem

The #72 roadmap session exposed several workflow/tooling problems in the agent loop itself. These are separate from #80's larger product idea of automatically analyzing TTY logs; this issue tracks concrete fixes to friction we already observed.

Evidence came from the live Pair TTY log for `PAIR_TAG=2`, rendered with `pair-scrollback-render --plain`. The session completed, but several failures caused avoidable retries, invalid intermediate state, or noisy review loops.
