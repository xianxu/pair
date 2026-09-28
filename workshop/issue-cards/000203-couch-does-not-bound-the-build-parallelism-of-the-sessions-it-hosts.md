---
id: 000203
status: open
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
github_issue:
---

# couch does not bound the build parallelism of the sessions it hosts

## Problem

couch hosts N agent sessions. Each agent runs `go test ./...` (often via an
`sdlc` gate) with **`GOMAXPROCS` unset**, so every build assumes it owns the
machine: Go's build parallelism (`-p`) and in-package test parallelism
(`-parallel`) both default to `NCPU`.

On this host — 12 cores, 8 performance + 4 efficiency — five concurrently
working sessions therefore request up to ~60 runnable threads.

### Measured

```
load average:  25.32  43.17  108.22     (1 / 5 / 15 min, on 12 cores)
```

Observed range over the debugging session: 6 when quiet, 69 during a spike, and
a 15-minute average of ~105 — i.e. **up to ~8.7× oversubscription**. Sampling
confirmed 3–7 concurrent Go compile/link processes and multiple repos building
at once.

couch is the **only** component that can fix this. It knows how many sessions are
live; an individual agent cannot know the others exist, and the operator should
not have to hand-tune a global. `grep` confirms couch sets no resource env
today: `launch_existing.go` builds a per-session env (`COUCH_TREE`,
`COUCH_STORE_DIR`, `COUCH_THREAD_SCOPE`, `COUCH_THREAD_TAG`) and nothing else.

### The controlled measurement: activity, not population

On 2026-09-06 the operator left the machine for four hours without using couch.
Nothing was torn down — the session population *grew* — and the lag vanished.
Same probes, same host:

| | during lag | after 4h unused | change |
|---|---|---|---|
| zellij servers | 10 | **11** | grew |
| pair nvim | 10 | **11** | grew |
| **running agents** | 8 | **2** | ↓ 4× |
| load (1/5/15) | 25 / 43 / 108 | **2.3 / 1.9 / 1.4** | ↓ 20× |
| `zellij action` round-trip | 145 ms (max 467) | **17.6 ms** | ↓ 8× |
| `alt+Return` (2 calls) | ~290–930 ms | **35 ms** | ↓ 8–26× |

**Idle sessions are free; concurrently *working* agents are not.** Population is
not the variable and never was — which is why the operator's "I ran more pair
sessions under cmux without this" was correct, and is the sharpest available
statement of this issue's scope.

### A retracted experiment, kept as a warning

An earlier probe in the same session concluded that load does **not** affect
latency, from this:

| condition | wake-up delay (median / p90) |
|---|---|
| baseline, load ~6 | 2.51 / 2.52 ms |
| + 12 CPU burners, normal priority | 2.21 / 2.74 ms |
| + 12 CPU burners, `taskpolicy -c background` | 1.72 / 2.58 ms |

**That conclusion is withdrawn — the experiment was unrepresentative.** Python
busy-loops are steady pure CPU: no syscalls, no file IO, no forking. A real
`go test ./...` spawns hundreds of short-lived compile processes with heavy IO.
A *process-spawn storm* is exactly what degrades a spawn→dynamic-link→connect→
round-trip path; steady CPU burn is exactly what does not. The probe tested the
one load shape that could not show the effect, and the real workload then showed
an 8× hit on the same call.

**Consequence for this issue: the preference for QoS over a parallelism cap rests
on that void evidence and must be re-derived.** QoS may still win; it is no
longer evidenced. Re-run against concurrent real `go test ./...` invocations —
never synthetic CPU burn — before choosing (see Plan).

### What this does and does not cause — measured, not assumed

This issue was opened while debugging reported workbench sluggishness, and the
initial diagnosis (**"load explains the lag"**) was **falsified**. Recording the
experiment, because it should discipline the fix:

| condition | wake-up delay (median / p90) |
|---|---|
| baseline, load ~6 | 2.51 / 2.52 ms |
| + 12 CPU burners at normal priority | 2.21 / 2.74 ms |
| + 12 CPU burners at `taskpolicy -c background` | 1.72 / 2.58 ms |

Twelve full-CPU processes did **not** degrade wake-up latency — macOS boosts
recently-blocked threads, so a mostly-idle interactive process is largely immune
to CPU oversubscription.

What load *does* hurt is the multi-stage spawn→dynamic-link→connect→round-trip
path, which has four scheduling points rather than one: a `zellij action` call
measured **36 ms calm vs 145 ms median / 467 ms max under load** — a 4× hit. That
path is `#201`.

So the honest scope: **this issue is about machine-wide oversubscription and its
4× multiplier on `#201`, not about keystroke latency.** Fixing it will not fix
typing lag on its own.
