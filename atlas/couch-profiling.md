# Finite local Couch profiles

For CPU/allocation attribution, launch Couch with `COUCH_PROFILE_DIR` naming a
new absolute directory under an existing parent. Profiling is off by default;
setting it on help, messaging or other non-launch commands has no effect.

```sh
COUCH_PROFILE_DIR="$HOME/couch-profile-investigation-1" couch
```

After the Console is admitted, one worker waits 60 seconds, records a 30-second
CPU window and writes cumulative allocation snapshots before and after it.
It does not change Go's sampling rates. No HTTP listener is created. Keep the
exact executable used by that launch; `metadata.json` records the running
binary's compiled build information and timestamps. An executable subsequently
replaced at the same path is not a matching binary.

The new directory is private (0700), with exclusive 0600 files:
`cpu.pprof`, `allocs-before.pprof`, `allocs-after.pprof` and `metadata.json`.
An existing directory is refused without reusing its contents. Each binary
profile is capped at 32 MiB; metadata at 64 KiB. IO failures or caps make the
capture failed/incomplete and produce a diagnostic. Profiles collect no terminal
text, browser DOM, environment dump or credentials. They remain local.

The Console notice feed reports armed, capturing and completed/failed states.
Ending the Console cancels and joins the worker, including any acquired CPU
profiler. A busy global profiler is left owned by its original caller. Diagnostic
failure preserves Couch's normal exit code; a failure discovered during shutdown
is printed after terminal restoration. An early exit can leave empty or partial
profile files; inspect the outcome instead of treating their existence as proof.

Inspect CPU and allocation changes with the matching executable:

```sh
go tool pprof -top /path/to/exact/couch /path/to/profile/cpu.pprof
go tool pprof -top -sample_index=alloc_space -base /path/to/profile/allocs-before.pprof /path/to/exact/couch /path/to/profile/allocs-after.pprof
```

Allocation profiles are sampled cumulative counters, not RSS. Profile overhead,
workload, geometry, actor count and cast/viewer state belong with any comparison.
No automatic optimization or sampling loop follows this window. The requesting
owner retains this fixed file set until the investigation ends, then removes it.
A new launch requires a new directory; failed output is never overwritten.

`cmd/internal/couchcmd/profiling_model.go` owns the pure lifecycle;
`profiling.go` interprets its effects and owns the runtime/file adapters.
`run.go` binds it to the existing interactive Console finisher. Controlled clock,
backend and output failures qualify ownership/cancellation; the real runtime
conformance test checks all three profiles with `go tool pprof`. Native workload
measurement remains #865; this profiler extraction does not authorize a live restart.

The [#865 live attribution](../workshop/evidence/000865/live-profile.md) records
the 2026-10-24 operator-approved capture: about1.48 cores during startup slot
reattachment, with synchronized-output screen capture the largest sampled
allocator. A later same-process observation was about0.41 cores. That difference
does not identify the cause of an hours-old cost; an aged profile is a separate
measurement. Cumulative owners overlap, allocation accounting can lag GC, and
neither allocation churn nor this CPU profile establishes fseventsd causality.
