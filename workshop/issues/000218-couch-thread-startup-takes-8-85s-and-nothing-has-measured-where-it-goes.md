---
id: 000218
status: working
deps: []
github_issue:
created: 2026-09-08
updated: 2026-10-14
estimate_hours:
card_mirror: '579de3da835f0469519be66c3cae2008e8d78b6f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:00:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# couch thread startup takes 8.85s and nothing has measured where it goes

## Problem

`#215` raised couch's registration deadline from 5s to 15s against a **measured
8.85s** startup for `pair resume <tag> --layout3`, and made name assignment O(1)
zellij probes. Both were right, and neither explains the 8.85s. A deadline
chosen as 1.7x an unexplained number is a guess with a measurement attached.

The operator's felt symptom, after `#215` unblocked couch: *"startup works now.
I guess it's just slow?"*

### What the number is NOT — measured, so these are closed

| suspect | measured | how |
|---|---|---|
| nvim + pair's `init.lua` | **117 ms** | `nvim --startuptime`, full breakdown; slowest single step is markdown syntax at 3.5ms |
| `zellij list-sessions` | **43 ms** at 26 sessions | wall-clock, 5 runs, 42-45ms |
| one `ProbeSessionName` | **~45 ms**, creates no session | `zellij --session X action list-clients` against absent names |
| name assignment | **8 probes**, flat as the index grows | `#215`, was 52 |
| 6 orphaned zellij servers | **0.0% CPU** each | `ps`; they cost RSS and list-sessions entries, not time |

So roughly 0.5s of the 8.85s is accounted for. **~8.3s is unexplained.**

### What is known about the shape

- **The two waits are different.** The cold path (`awaitThreadRegistration`)
  polls the thread-claim file. The resume path (`awaitResumeRegistration`) waits
  for the zellij **session to be live** — so only resume pays for zellij bring-up,
  layout, three panes, nvim and the agent. The claim file for one long-lived
  thread was last written four days before it was next resumed, which is how we
  know the resume path never touches it.
- `pair` spawns `zellij` from **eight** call sites (`osruntime.go` x6,
  `session_quiescence.go` x2). Nobody has counted how many times a startup
  actually calls it.

## Spec

**Explain the 8.85s with a measurement, then decide what to fix — in that order.**
`#215` was first filed against a root cause that measurement refuted, and its
fix then shipped a 7x regression on the resume path that only an operator
noticed. This issue does not get to guess.

**1. Run the instrument that already exists.** `probes/zellijcalls/` traces every
zellij subprocess a startup spawns, via a PATH shim rather than instrumented call
sites — pair spawns zellij from eight places, so instrumenting "the seam" means
instrumenting eight and missing the ninth. Built in `#215`, never run against a
real couch thread start.

```
eval "$(probes/zellijcalls/trace.sh arm)"
couch                      # start a thread, then quit
probes/zellijcalls/trace.sh report
```

The report's own shape answers the first question: **`in-zellij time` close to
the wall span means the call count is the problem; well under it means the time
is NOT in these subprocesses** and the next instrument goes inside pair.

**2. Only if the time is not in zellij, add stage timing to pair's startup.**
Env-gated, off by default, emitting monotonic marks around the phases between
process start and registration. Do not build this before step 1 says it is
needed — that is the guess this issue exists to avoid.

**3. Decide with the number in hand.** 8.85s may be irreducible (an agent process
genuinely takes seconds to start) or may be one dumb loop. The fix, if any, is
out of scope until the breakdown exists. If it IS irreducible, say so in the
`pairRegistrationTimeout` comment, which currently reasons about a bounded
envelope without knowing what fills it.

## Done when

- A recorded breakdown of the 8.85s exists, with each stage named and timed, and
  it accounts for the wall time rather than a fraction of it.
- `pairRegistrationTimeout`'s comment cites that breakdown instead of a bare 1.7x
  multiple.
- Whatever the breakdown shows is either fixed, or recorded as irreducible with
  the measurement that says so.

## Plan

- [ ] Run `probes/zellijcalls/trace.sh` against a real couch thread start (cold)
      and a relaunch; record both reports in `## Log`.
- [ ] Branch on the result: call count -> reduce calls; otherwise add env-gated
      stage timing inside pair's startup and re-measure.
- [ ] Attribute the wall time stage by stage until the residual is small enough
      to name.
- [ ] Fix what the breakdown indicts, or record irreducibility.
- [ ] Update `pairRegistrationTimeout`'s comment to cite the breakdown.

- [ ] M1 — baseline attach/detach timings and native candidate/fencing foundation ([durable plan](../plans/000218-startup-detach-observation-plan.md)).
- [ ] M2 — shared pass capture, 6/20/40 experiment and measured tuning.

## Log

### 2026-09-08

Split out of `#215` at the operator's direction: *"#215 is about unblock so that
I can continue to use (and thus dogfood) couch."* `#215` achieved that — cold
start and relaunch both work — so latency is its own question rather than a
reason to hold that issue open.

The measurements in `## Problem` were taken while closing `#215` and are the
reason this issue is narrow: four plausible suspects are already dead, which is
worth more than an open-ended "make startup faster".

### 2026-10-14 — planning trace

Claimed by pair:5, started from main 9a2aff687 after slot substrate refresh. Operator approved one snapshot per pass; durable implementation plan: [startup/detach observation](../plans/000218-startup-detach-observation-plan.md). Read-only path trace shows two full native snapshots per healthy SessionOwnerProbe, repeated by Couch prelaunch/registration checks and four child Pair checks (approximately twelve scans for healthy single-poll attach), and approximately six on healthy detach. These are path counts, not measurements. Baseline real attach/detach timing and existing zellij PATH trace precede behavior edits. ARCH-DRY reuse ServerStates, ARCH-ORDER exact fresh fences, ARCH-CONSTRAINTS measure 6/20/40 after reducing duplicate observation.

- Baseline instrument first run: existing TestSessionDetachLive failed before its detach assertion because full ZellijSource.Snapshot queried an unrelated host session (📁1-235) and list-clients returned exit1. The controlled fixture was cleaned up. This is not a production detach timing; the complete targeted owned-fleet harness is still required.

## Revisions

- 2026-10-14 — add (operator via TL): concrete case. At the operator's 14:33 restart, Couch's background attaches spawned 21 zellij clients between 14:33:37 and 14:33:47 (~6 at a time): roughly 2–3 s per attach, against ~10–20 ms for `zellij list-sessions` and an expected ~100–300 ms floor (list, lease, one record write, spawn client, first frame). Detach ("leave couch") is unmeasured but also inspects each slot's session server. Trace one attach and one detach step by step; the likely waste is a whole process-table read (native argv for every process since #547) per slot. Probably one shared snapshot per startup or detach pass instead.

- 2026-10-14 — add (operator): try much higher attach parallelism (operator suggests 40; today `LifecycleParallelism = NumCPU/2` = 6 on a 12-core Mac). Run the same restart at 6 / 20 / 40 with a temporary override; record total time and peak load; keep what the numbers support. If per-slot whole-process-table reads dominate, fix those first (shared snapshot per pass).

- 2026-10-14 — add (implementation plan): M1 baseline plus native candidate/fencing foundation; M2 shared parent pass and remeasurement/concurrency selection. Child Pair cannot reuse Couch memory, so narrowed standalone discovery is included. Preserve #553 and exact-generation effect checks.

- 2026-10-14 — refine (PQ-1/PQ-2/PQ-3): shared evidence is initial only. Fresh after-pane/effect checks retain name exclusivity via candidate-only enumeration; registration bypasses the pass to observe post-launch births. Compact named-function test strategies supersede procedural case inventories in the durable plan.
