---
id: 000210
status: open
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
---

# perf capture: no stage is time-bounded, so one slow collector blows the budget
## Problem

`doctor/perf.sh` (`#208` M1) checks its deadline **between** stages but bounds
**no stage**. If `top -l 2` or `iostat` hangs, nothing stops it — the next
`collectors_done()` check simply arrives late.

That gap matters more than it looks, because of what this tool is for. It exists
to be run **while the machine is struggling**, which is exactly the condition
where a collector is most likely to stall. A capture that hangs for 30 s during
a slowdown is worse than no capture: the operator is already suffering, and
`#208`'s own ARCH-CONSTRAINTS names not blocking them as the binding constraint.

Raised as `BR-34` in `#208` M1's boundary review and **deliberately deferred** by
operator decision: *"it seems the script can be made more robust... let's not let
perfect be the enemy of good."* The capture works and is usable today; this is
the hardening pass.

### The other findings deferred with it

All from `#208` M1's review, demoted past the round cap. Recorded here so they
are not lost when that gate ledger archives:

| id | finding |
|---|---|
| `BR-34` | no stage is time-bounded (this issue's subject) |
| `BR-25` | shed order — *fixed*, but re-raised; confirm the reverse-value shedding actually holds under a real squeeze rather than the synthetic `BUDGET=3` |
| `BR-38` | the grammar pin asserts against the ambient system — *fixed* by requiring ≥10 rows, but a recorded-input test would be stronger than a live one |
| `BR-39` | swap rate over an un-elapsed window — *fixed and verified at `BUDGET=1`*; listed for completeness |

Only `BR-34` is believed still open. The other three were fixed in rounds 3–4
and are recorded because the review re-surfaced them, which is worth checking
rather than assuming.

## Spec

**Every stage of the capture is bounded, not just checked between stages.**

- Each external collector (`top`, `iostat`, `ps`, `vm_stat`) runs under a
  per-stage limit; exceeding it yields `n/a (<tool> exceeded its slice)` and the
  capture continues.
- The total stays within the declared budget **by construction**, not by the
  stages happening to be fast.

### The mechanism, which is the whole difficulty

macOS ships no `timeout(1)` (coreutils is not installed by default), and the
capture must not grow a dependency. The available pattern is a background killer
— run the collector, background a `sleep N; kill` against it, take whichever
finishes first — which `#208` already used once for the frame-mouse probe.

That pattern has a trap worth stating before anyone writes it: **a backgrounded
reader cannot read the terminal**, which is how the first version of that probe
failed silently. It applies to a collector reading a pipe too, so the killer
must background, never the collector.

## Done when

- A deliberately hanging collector (a stub that `sleep 60`s) does not extend the
  capture beyond its budget, and its row reads `n/a` with a reason.
- `doctor/perf_test.sh` drives that case with a stub, in the same style as its
  existing failing-collector test.
- No new runtime dependency; POSIX shell only.

## Plan

- [ ] Add a bounded-run helper using the background-killer pattern.
- [ ] Apply it to every external collector, not just the slowest one.
- [ ] Test with a hanging stub; assert both the budget and the `n/a` row.
- [ ] Re-verify the three findings above that are believed already fixed.

## Log

### 2026-09-07

Split from `#208` M1 at operator decision after seven review rounds. The capture
is working and in use; this is hardening, not a defect blocking its value.

Worth carrying from that review: nearly every finding across those rounds was in
the **degraded path** — a `0` from a failed `ps`, a `0.0` swap rate from an awk
over empty input, a rate divided by a window that never elapsed, an
empty-but-present sample that made the join report every process as vanished.
All four look like data. This issue is the last member of that family: a hang is
the one degraded case that produces no wrong number, just no answer at all.
