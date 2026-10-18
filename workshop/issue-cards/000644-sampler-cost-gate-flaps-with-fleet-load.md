---
id: 000644
status: codecomplete
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
severity: medium
started: 2026-10-17T21:09:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 1.41
tracker:
    version: 1
    completion:
        token: close-043fbfc29dde
        repository: github.com/xianxu/pair
        reviewed_head: 988914e084893cdf03b8d1467b18d463d1879e42
        evidence_commit: 60756f968db27a8f14eef77bbc8ee5bab7e78c2b
---

# The resource sampler cost gate flaps with fleet load: first pass is not steady state and nothing names the load

## Problem

`TestResourceSamplerNativeCost` (`cmd/internal/couchcmd/resources_test.go`, env-gated on
`PAIR_RESOURCE_COST=1`) is a coin flip on fleet load. Its gate is #587's approved qualification: mean
sampler CPU under 120 ms and max under 300 ms across three passes, i.e. "below 0.2% of one core at the
default minute cadence". Two runs of identical code on the same host, hours apart, while closing pair#583:

    load ~14, 1270 rows: pass1 142.409ms (1585 calls)  pass2 63.952ms (64)  pass3 65.244ms (64)
                         mean 90.535ms  max 142.409ms  0.150892%  -> PASS
    load ~56, 1323 rows: pass1 245.274ms (1581 calls)  pass2 80.329ms (66)  pass3 84.535ms (64)
                         mean 136.712ms max 245.274ms 0.227854%  -> FAIL "resource CPU budget exceeded"

The load ~56 run had three other slots' suites going, including ariadne's sharded full suite. Nothing in
the failure output says so: it reads exactly like a sampler regression.

Two causes, neither of them the thing the budget is meant to catch:

1. The first pass is not steady state. It makes ~1521 more native calls than passes 2 and 3 — one
   environment read per process generation, which is #606's detached-slot attribution, and which
   `atlas/couch.md` already documents as "about 0.3 s for 1,200 processes on Couch's first pass, then only
   new processes". #587 landed 2026-10-15 and qualified at 71.331 ms mean / 74.264 ms max over ~1007 rows
   and 58 native attempts; #606 landed the day after and roughly doubled the first pass. The constant was
   never re-derived. #587's own plan records the TL ruling that "the binding CPU budget applies to steady
   sampling, not a rare maximum-cardinality admission burst" — and a three-pass mean now folds in a cold
   pass that is neither.
2. The whole measurement is host-shaped. Rows, syscall latency and the process count all move with what
   else the fleet is doing, and the assertion has a fixed constant on the other side of it.

Steady state was inside the budget in both runs (64-85 ms). Only pass 1 moved, from 142 ms at rest to
245-265 ms under load.

Because the test is env-gated it runs in neither CI nor `make test`, so nobody sees either failure mode:
the repo's only conformance for the sampler's cost budget is unreliable and invisible at the same time.

For whoever takes this, two measurements from the same session that bound the new work rather than the old:

- pair#583's orphan nvim probe is not a contributor. At its widest gates on that host it cost 2 argv reads
  and 2 cwd probes, about 69 us per pass (per-call native cost: cwd 8.609 us, argv 26.107 us). Removing
  #583's widening from the test under the same load ~56 gave mean 138.438 ms — worse, so it is noise.
- Relaxing `UnattributedShape`'s `p.PPID == 1` to `>= 1` took the same pass from 2 to 122 argv+cwd probe
  pairs on a host running ~30 pair embeds. The PID-1 gate, not the age threshold, is what bounds the probe
  count; keep that visible wherever the budget is re-derived.
