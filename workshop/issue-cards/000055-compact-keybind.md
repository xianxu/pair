---
id: '000055'
status: done
created: 2026-06-11
updated: 2026-06-12
estimate_hours: 2.5
actual_hours: 0.91
---

# Alt+Shift+C compaction: continuation + in-session restart via context-aware pair continue

## Problem

A long pair session bloats the agent's context. We want a one-keypress
"compaction" — distill the session into a durable `continuation` doc, then
restart the *same* session with a fresh agent conversation seeded from that
doc. Today this is a manual multi-step dance (ask agent to write a
continuation → `Alt+x` → `pair continue` in a new terminal).

Constraint that shapes the design: **creating a continuation is an agent
judgment step** (distill NEXT ACTION / open threads / decisions, then the
`cmd/pair-continuation` writer commits it) — there is no mechanical one-shot.
And **you cannot fresh-start a session from inside zellij** (`--session` from
within attaches/nests — see `bin/pair:685`), so the restart must go through the
existing kill → `handle_restart_marker` → outer-relaunch path.
