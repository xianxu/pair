---
id: 000649
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '640427ab6238af054d0d70a0e08c910c03bc118b' # card fields mirrored from issue-cards; edit via sdlc
---

# pair title pollers: 18 processes hold 1.6 GB RSS and ~5.5 CPU-s/min (one 52 CPU-min in 68 h) for a once-a-minute list-sessions + rename

## Problem

## Spec

Measured by pair:2, 2026-10-17 ~10:50–11:00, across the live fleet (22 slots). The `pair title` pollers (`titlepoller`, which keeps each zellij frame title current on a 60 s tick) are:
- 18 processes holding **1.6 GB RSS** in total, the largest about 185 MB each;
- using about **5.5 CPU-seconds a minute** together;
- one, for 1-pair-6, has used **52 CPU-minutes over 68 hours**;
- 12 of the 18 are reparented to launchd. That's normal for a detached poller, but some may outlive their session.

The work per tick is one `zellij list-sessions --no-formatting`, sometimes a liveness read, and a `rename-pane` on change. That should cost a few MB and a few CPU-milliseconds a minute. The memory and CPU point to a leak (memory growing per tick) or over-polling (ticks or retries much faster than 60 s).

#617 fixed the pollers' lifetime with a lease. This issue is about their steady-state cost while alive.

## Done when

- Measured cause: the RSS growth per tick and the tick and query rate of one live poller over ~10 minutes, before the fix.
- The fix keeps a poller's RSS flat over 1,000 simulated ticks, with a test, and its CPU at about one tick's work a minute.
- After the next restart and relaunch, the fleet's `pair title` total RSS and CPU are re-measured and recorded.

## Plan

- [ ]

## Log

### 2026-10-17
