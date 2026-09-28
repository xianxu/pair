---
id: 000210
status: open
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
github_issue:
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
