---
id: 000172
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Parallelize the zellij session snapshot

## Problem

`ZellijSource.Snapshot` polls each live session serially for its client count.
Measured on a 19-session host (6 exited, 13 live) on 2026-09-02:

- whole snapshot: **1.49 s**
- two `list-sessions` runs: ~16 ms each
- one `action list-clients` per live session: ~100 ms each, 13 of them = 1.43 s

Since `pair#170` M2, Couch's actionable inventory takes this snapshot whenever a
detach candidate exists, and M3 put that inventory on the **blocking** startup
path: `StartInteractive` must decide resume-vs-new before it attaches anything,
so `couch` in a directory pays ~1.4 s before the first frame. M2 also made
*detached* the normal resting state, so this is the ordinary case rather than an
edge one.

The switcher itself is unaffected — refreshes are event-driven and run on the
single-flight worker while the menu renders its last-good projection — so this
is a startup-latency issue, not a keystroke-budget one.

## Spec

The per-session `list-clients` queries are independent, so run them with bounded
concurrency instead of serially. Expected: ~1.4 s → ~150 ms on the measured host,
i.e. the cost of the slowest single query rather than their sum.

Constraints:

- **Bounded fan-out.** A host can have dozens of sessions; spawning one
  subprocess per session at once trades latency for a thundering herd. Cap
  concurrency (8 is a reasonable starting point) and state the cap.
- **Preserve the existing contract exactly.** `zellijQueryTimeout` still bounds
  each query, a failed query still counts as zero clients (which classifies the
  session detached rather than attached), and the returned slice stays sorted by
  name so callers keep a stable order.
- **Cancellation.** `SnapshotContext` already takes a context; a canceled
  context must stop dispatching new queries rather than waiting out the ones
  already running.

Deliberately out of scope: caching the snapshot across refreshes. That trades
staleness for latency and needs its own argument about how stale a row may be
before it misleads the operator.

## Done when

- A snapshot over N live sessions costs about one query, not N.
- A test pins the bound (concurrency never exceeds the cap) and the contract
  (per-query timeout, failed query = zero clients, stable ordering).
- `BenchmarkZellijSnapshotLive` records the before/after on a real host.

## Plan

- [ ] Pure-ish refactor: extract the per-session poll so it can be dispatched
      concurrently without duplicating the classification rule.
- [ ] Bounded worker fan-out in `SnapshotContext`, honouring cancellation.
- [ ] Tests for the cap, the timeout, the failure classification and ordering.
- [ ] Re-measure and record beside the 1.49 s baseline.

## Log

### 2026-09-02

Filed from `pair#170` M3's boundary review, which asked for the startup envelope
to be measured rather than asserted. The measurement is what surfaced this: the
review's finding was that the claim was stale, and the number behind it turned
out to be worth its own issue.
