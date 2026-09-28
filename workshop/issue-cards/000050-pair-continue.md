---
id: '000050'
status: done
created: 2026-06-11
updated: 2026-06-11
estimate_hours: 8
actual_hours: 0.75
---

# pair continue: sessionView --plain, continue verb, park-nudge

## Problem

pair wraps any TUI agent and can relaunch with a different agent, but has no way
to durably hand off the *human-meaningful* state of a session. `pair resume <tag>`
restores the **native** session (same agent, same `session_id`) — machine state
that depends on the agent's own, recyclable store. There's no portable "pick this
up later / elsewhere / on another agent" path. The substrate for that already
exists in pair — the rendered scrollback (`pair-scrollback-render`) is the
cleaned-up, human-consumable projection of the session — but it's only emitted as
SGR-colored text for the Alt+/ viewer, and there's no verb to turn a session into
a durable continuation or to resume from one.
