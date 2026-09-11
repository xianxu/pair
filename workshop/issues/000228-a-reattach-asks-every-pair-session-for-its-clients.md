---
id: 000228
status: working
deps: []
github_issue:
target: workbench-latency
created: 2026-09-10
updated: 2026-09-10
estimate_hours:
started: 2026-09-10T21:58:26-07:00
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

## Spec

The invariant, stated before the mechanism: **the client-count work a reattach
does scales with the threads it touches, not with the sessions that exist.**
Each snapshot site asks only what its decision needs:

- **Sites 1–2 (the detached proof)** ask only about the candidate thread's own
  session. The proof keeps its meaning: the session exists, is not exited, and
  has zero clients. It is not weakened to save a call.
- **Site 3** needs liveness only. If `list-sessions` answers that, it needs no
  client counts at all.
- **Site 4** is decided in planning, after reading what `DecideLaunch` and its
  helpers read for a forced-tag attach. It is not assumed.
- **Site 5** is out of scope. The inventory really does describe every
  session, and its refresh schedule already coalesces.

Planning decides the mechanism, for example a targeted snapshot over named
sessions beside the full one. It must keep one parser and one call pattern
(`ARCH-DRY`), not a second hand-rolled zellij client.

## Done when

- A couch reattach of one thread makes a number of `list-clients` calls
  independent of the number of live pair sessions. This is asserted as a count
  at the `ZellijSource` seam (a counting fake or shim), not as a timing.
- The detached proof's semantics are unchanged. Existing proof tests pass
  unmodified, and the targeted form has tests for exists / exited / attached /
  detached.
- Every snapshot site in the table above is either narrowed or has its full
  scan justified in the Log.
- `cmd/probes/reattachcost` ships, with a `make test-reattach-cost` target, and
  is re-run after the change with co-tenancy recorded. The couch-shaped phase's
  per-thread time is compared with #206's baseline.
- `#206` is unblocked.

## Plan

- [ ] Baseline: done (#206 Log, 2026-09-10).
- [ ] Establish what each of sites 1–4 reads from its snapshot; record it.
- [ ] Design the narrowing (durable plan if it crosses packages).
- [ ] Implement with the count-based tests.
- [ ] Re-measure with the probe; record before/after with co-tenancy.

## Log

### 2026-09-10

Split out of `#206` at the operator's direction, after #206's Plan step 1
measurement showed the reattach cost is couch's and pair's proof snapshots, not
`zellij attach`. `#206` depends on this and is blocked on it.
