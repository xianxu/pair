---
id: '000110'
status: done
started: 2026-07-07T21:15:56-07:00
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.5
actual_hours: 0.92
---

# launcher cannot resume selected scoped codex session

## Problem

After #107, bare `pair` correctly shows current-repo scoped tag + agent rows,
but resuming a selected historical Codex row can fail at the saved-session step:
the tag/agent is visible, yet the config picker does not find the associated
Codex session to resume.

Root cause found during debugging: `OSRuntime.AgentSessionExists` checks Codex
session files only directly under `~/.codex/sessions`, while Codex writes nested
date paths such as `~/.codex/sessions/YYYY/MM/DD/rollout-...<sid>.jsonl`.
