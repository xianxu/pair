---
id: '000144'
status: done
started: 2026-08-19T07:09:43-07:00
created: 2026-08-19
updated: 2026-08-19
estimate_hours: 2.22
actual_hours: 0.05
---

# Reject Codex subagent sessions during Pair identity discovery

## Problem

Pair identifies a live Codex conversation from rollout filenames exposed by
the agent process tree. Codex subagents write rollout files in the same
directory and may be open in the same process tree, so filename-only discovery
can persist a subagent ID. `Alt+n` then gives that ID precedence over the saved
config and resumes the subagent rather than the operator's root conversation.

The same ambiguity exists in the asynchronous session watcher's birth-time
fallback. Issue #143 extends that watcher for the lifetime of an agent, making
it more likely to observe later subagent rollouts when the root session has not
already been captured.
