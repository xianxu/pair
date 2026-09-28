---
id: '000183'
status: done
started: 2026-09-10T16:57:25-07:00
created: 2026-09-04
updated: 2026-09-10
estimate_hours: 1.62
actual_hours: 1.28
---

# Attach drops PAIR_SCOPE_KEY, so the context meter vanishes after reattach

## Problem

A pane's zellij frame title is `<agent> (<count>)` — `claude (184k)` — where the
count is the agent's context-window size (#71). **After a thread is reattached,
the count is gone and the frame reads bare `claude`.** Observed 2026-09-04 on
all three reattached threads in one couch session, and not on a thread that had
been launched normally and never detached.

The failure is silent in both directions: nothing errors, and the title still
renders — it just quietly loses the half the operator was reading. The context
meter is the one surface that says "this session is nearly full", so losing it
without a signal is worse than losing it loudly.
