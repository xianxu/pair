---
id: '000022'
status: done
created: 2026-05-25
updated: 2026-05-25
actual_hours: 0.3
---

# Rename a pair tag without losing the agent session

## Problem

A pair tag is the durable identity of a coding session (zellij session
name `pair-<tag>`, saved agent session in `config-<tag>-<agent>.json`,
draft `draft-<tag>.md`, etc.). Today the tag is chosen once at create
time and frozen.

Common workflow: start with a generic tag (`brain-2`) for exploratory
work, then narrow into a specific train of thought (`gstack-deep-dive`).
The user wants the tag to follow the work, not the other way around —
without dropping the agent's conversation, draft buffer, scrollback
history, or queued/quoted items that have accumulated.

There is no live-rename for a zellij session — the session name is
baked into the running session. So "rename" is necessarily a *quit,
swap tag on disk, re-exec* operation. That's structurally identical
to `pair-restart.sh` plus a tag-swap step, and the natural surface
is to fold the rename gesture into the existing restart confirm
(Ctrl+Alt+n / Alt+n).
