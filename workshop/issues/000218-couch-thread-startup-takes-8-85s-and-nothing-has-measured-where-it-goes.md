---
id: 000218
status: open
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
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

## Log

### 2026-09-08

Split out of `#215` at the operator's direction: *"#215 is about unblock so that
I can continue to use (and thus dogfood) couch."* `#215` achieved that — cold
start and relaunch both work — so latency is its own question rather than a
reason to hold that issue open.

The measurements in `## Problem` were taken while closing `#215` and are the
reason this issue is narrow: four plausible suspects are already dead, which is
worth more than an open-ended "make startup faster".
