---
id: '000141'
status: done
started: 2026-08-16T22:37:52-07:00
created: 2026-08-16
updated: 2026-08-17
estimate_hours: 0.35
actual_hours: 0.19
---

# Alt+n restart should recover live Codex session

## Problem

`Alt+n` restarted the Pair workbench but launched Codex without a native
`resume <session_id>` token, so the restarted agent opened a fresh conversation
instead of continuing the prior one. The live evidence is
`wrap-events-pair.jsonl` showing `argv=["codex","--no-alt-screen"]` on the
restart.

Root cause: plain restart writes only tag/agent/new-session intent into the
restart marker before killing the pane. Re-entry later composes resume args from
saved config only. For a long-running Codex pane whose native session id was not
captured in `config-<tag>-codex.json`, Pair had one last chance to inspect the
live Codex process and recover its open rollout transcript, but the restart path
did not do that before killing the session.
