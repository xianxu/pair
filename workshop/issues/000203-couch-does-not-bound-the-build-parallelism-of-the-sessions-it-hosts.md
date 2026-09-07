---
id: 000203
status: open
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
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

## Spec

**couch bounds the resource appetite of the sessions it hosts.** Two candidate
mechanisms; the measurement above already argues for the first.

**1. Deprioritise rather than cap (preferred).** Launch session work at a lower
QoS tier so builds absorb idle capacity but yield to interactive work. The probe
above shows `taskpolicy -c background` is not merely neutral but *better* than
baseline (1.72 ms vs 2.51 ms median). It is self-balancing: a lone session still
gets all 12 cores, where a static cap would leave 9 idle.

Caveats to settle: `-c background` also throttles **I/O**, which may slow builds
more than intended — check `-c utility` as the middle tier. And whether the tier
is inherited by grandchildren (the agent's `go`, its compile children) needs
verifying, not assuming.

**2. A parallelism budget, if QoS proves insufficient.** Note that the obvious
form does not work: env vars are fixed at process launch, so a `GOMAXPROCS`
couch computes when a session starts is stale as soon as the next session
starts. The property that rescues it: **every `go` invocation is a fresh
process**, so resolving the budget at *invocation* time is naturally dynamic —
couch writes the current budget into its namespace dir (it already owns one via
`COUCH_STORE_DIR`) and a `go` shim on the PATH pair already populates reads it
per call. One file read per `go`, no protocol.

Prefer 1; treat 2 as the fallback, and measure before building it.

### Relationship to Admission (`#170`)

Adjacent but not the same, and the difference matters. Admission refused
*starts* over fleet capacity, with a cross-repo provider dependency, a stateful
fake and a live conformance target — correctly deleted as multi-owner
machinery. This is per-session resource appetite on one host, which is far
lighter. Worth noting that `#170`'s rationale was "couch-lite is one operator on
one host", and one host is precisely where core contention bites; the rescope
removed the heavy answer without leaving a light one.

## Done when

- With N sessions building concurrently, interactive latency is measurably no
  worse than with one — reported with the wake-up-delay probe above and an
  end-to-end `zellij action` timing, calm and loaded.
- A single session building alone still uses the full machine (no static cap
  left in place that wastes cores).
- The chosen mechanism reaches the agent's **grandchildren** (`go` → `compile`),
  asserted rather than assumed.
- couch's behaviour is unchanged when hosting one session.
- `atlas/couch.md` records that couch manages host resource appetite, and why
  this is not a resurrection of Admission.

## Plan

- [ ] Measure `-c background` vs `-c utility` against a real `go test ./...`:
      interactive latency **and** build wall-clock, so the I/O throttle cost is
      visible.
- [ ] Verify QoS tier inheritance to grandchildren.
- [ ] Apply the tier at couch's launch path (`launchTrackedThread`), preserving
      the `#179` warm-reattach invariant — no new argv or env on a warm path.
- [ ] Re-measure under 5 concurrent building sessions.
- [ ] Only if QoS is insufficient: the budget-file + `go` shim from Spec item 2.
- [ ] Update `atlas/couch.md`.

## Log

### 2026-09-06

Third of three issues from one debugging session; `#201` (subprocess-per-zellij-
action) and `#202` (per-keystroke span rebuild) are the others.

Operator's framing was that couch was responsible, on the grounds that they had
run many pair sessions under cmux without this. That is right, and the mechanism
is worth stating precisely: **couch did not add a hot loop, it changed how many
sessions are simultaneously *working*.** Under cmux the operator drives one
session and the rest idle; couch's whole value is several running unattended at
once, which is when their builds collide. The feature is working as designed;
what is missing is that nothing arbitrates the shared machine.

Everything else was ruled out by measurement during the same session: memory
(95% free, 80 MB swap), disk (251 GB free), thermal (no warnings), Spotlight
(mds at 0.0%, 32 pageins/sec), couch's own event loop (no timers; title pollers
at 60 s), and nvim buffer size (drafts are 65–130 bytes).
