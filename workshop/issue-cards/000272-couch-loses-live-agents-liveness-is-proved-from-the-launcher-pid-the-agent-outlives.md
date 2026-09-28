---
id: 000272
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
---

# couch loses live agents: liveness is proved from the launcher pid the agent outlives

## Problem

Split out of `pair#265`'s diagnosis, third member of the same reconciliation
family as `pair#271`.

Measured on the operator's store, 2026-09-16: **all 11 records carrying
`recorded: live` have a dead pid.** Every one classifies
`unusable: stale — helper ownership unresolved`. But three of them have an agent
that is *still running right now*:

| couch row | recorded pid | actually alive |
|---|---|---|
| `pair·6c0d459a` (`couch-95293a9b6c0d459a`) | 81935 — dead | `pair wrap` 82147, `muse-bin` 82165 |
| `pair·b14d495a` (`couch-20b78027b14d495a`) | 90046 — dead | `pair wrap` 90274, `muse-bin` 90294 |
| `pair·93df0741` (`couch-937e6ea093df0741`) | 35749 — dead | `pair wrap` 35793, `muse-bin` 35810 |

Three live `muse` conversations couch shows as dead and cannot reclaim.

The recorded pid is the **launcher** (`pair <agent> --layout3 …`). When couch
exits abnormally the launcher goes with it, but `pair wrap` and the agent are
reparented and keep running. `liveProofMatches`
(`couchcore/actionableinventory.go:342`) correlates
`ProcessIdentity{PID, Identity}` against that launcher pid, so the proof fails
and the thread is "stale" while the work is still there.

The identity is not wrong — it is exact, and deliberately so, "so a recycled PID
cannot pass". The problem is that it names a process whose lifetime is **shorter
than the thing it is proof of**. A liveness proof keyed to a process that dies
first will report every crash as a lost thread.

Two consequences compound:

- Every couch crash orphans its agents. `pair#265` is one such crash and there
  have been many — hence 11 stale rows across 8 repos.
- The orphans are unreclaimable through any gesture. The row offers `archive`
  (it is non-actionable, so `menuThreadActionable` is false), which retires the
  record and leaves the agent running forever with nothing pointing at it.

Related but distinct: `parley.nvim` has two live couch-tagged agents
(`couch-797c45e8e649a9bb` via `pair wrap` 84488, `couch-2583ed61c0ab6ebe` via
`pair wrap` 1130) with **no record in the store at all** — its scope holds a
single record, the healthy parked one. `ClassifyThread` has a
`ReasonUnrecordedChild` ("running but unrecorded") branch for exactly this, but
it is per-record, so a running agent with no record is invisible rather than
reported.
