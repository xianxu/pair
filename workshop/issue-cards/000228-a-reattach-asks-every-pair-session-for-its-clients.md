---
id: '000228'
status: done
started: 2026-09-10T21:58:26-07:00
created: 2026-09-10
updated: 2026-09-11
estimate_hours: 1.98
actual_hours: 1.26
---

# A reattach asks every pair session for its clients

## Problem

**Proving that ONE thread is detached asks EVERY live pair session for its
clients, and a reattach does that five times.** Measured for `#206`:
`cmd/probes/reattachcost`, with full numbers and co-tenancy in #206's Log
(2026-09-10).

- `zellij attach` itself takes about 55 ms, and 8 at once barely cost more.
- A couch-shaped reattach (the snapshots couch actually takes, then the
  attach) takes **5.3–5.7 s per thread** on this host, with 8 live pair
  sessions of which 5 were detached. That is about 5.5 s for every manual
  reattach today.
- The call that costs is `zellij --session X action list-clients` against a
  *real* detached pair session: about 190–350 ms, and one took 971 ms. The
  same call takes about 38 ms against an attached session and about 53 ms
  against a fresh empty one, and `query-tab-names` against the same real
  detached session takes about 55 ms.

`launcher.ZellijSource.SnapshotContext` runs 2 `list-sessions` calls plus one
`list-clients` for every non-exited pair session on the host (its own comment:
"N is every pair session, not just the ones the caller cares about"). One couch
reattach takes 5 of those snapshots:

| # | site | what the caller actually needs |
|---|---|---|
| 1 | couchcore `ResumeContext` → `DetachedSessions` | is THIS thread's session live with zero clients? |
| 2 | couchcore `confirmStillDetached` → `DetachedSessions` | the same, re-proved before the child effect |
| 3 | launcher `createflow.go` loop top → `rt.Sessions()` (orphan-nvim sweep) | which sessions are live (not exited) |
| 4 | launcher `runOnce` → `rt.Sessions()` | what DecideLaunch reads (to be established in planning) |
| 5 | couchtty inventory refresh on completion | the whole inventory (every session, legitimately) |

Sites 1–4 pay O(all sessions) `list-clients` calls for a question about one
session, or for a question that needs no client count at all. A startup pass
over N detached threads (`#206`) is therefore O(N × S) of the expensive call:
about 80 s for 10 threads, and about 13 s for the first one.

This is `workbench-latency`'s defect shape: work per interaction scales with
what exists, not with what changed.
