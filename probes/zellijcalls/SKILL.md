---
name: zellij-call-tracer
description: Count and time every zellij subprocess a pair/couch startup spawns.
---

# Tracing zellij calls during startup

`pair` spawns `zellij` from eight call sites. This traces all of them by sitting
in `PATH`, so a call site nobody remembered is traced anyway.

```bash
eval "$(probes/zellijcalls/trace.sh arm)"   # in the shell that will launch couch
couch                                       # drive the startup you want to measure
probes/zellijcalls/trace.sh report
eval "$(probes/zellijcalls/trace.sh disarm)"
```

`arm` prepends a repo directory to the interactive shell's `PATH` and points
`PAIR_ZELLIJ_TRACE` at a per-invocation file; both persist until `disarm`.

## What the instrument costs

A probe must not perturb what it observes. Measured with
`probes/zellijcalls/trace.sh overhead` — a real A/B of direct zellij against
zellij reached *through* the shim, interleaved over 20 samples:

```
direct   10.5ms/call
shimmed  13.4ms/call
overhead  2.9ms/call
```

So roughly **+3ms per call**, or 6-14% of a 20-45ms zellij call. It lands
*outside* each recorded duration and *inside* the wall span, so it inflates the
`in-zellij time / wall span` ratio slightly — at 100 calls that is ~0.3s of an
8.85s span. Subtract `3ms x calls` from the span before concluding the time is
not in zellij.

An earlier version of this section published "**under 1ms**, below the noise
floor". That number was wrong: the measurement ran `exec.Command(real, ...)` on
*both* sides, differencing two identical operations, so it could not contain the
shim's cost at all. It reported a NEGATIVE overhead, which should have been the
tell.

The first cut was a bash script that read the clock with `python3 -c` twice per
call: ~25ms per interpreter start, so ~50ms injected per call, against calls
taking ~45ms. Worse than merely large — the injected cost landed *outside* each
recorded duration and *inside* the wall span, inflating exactly the ratio the
"Reading it" conclusion turns on. macOS ships bash 3.2, so `EPOCHREALTIME` is
not available to fix it in shell; the shim is Go for that reason.

`report` prints total calls, wall span, time actually spent inside zellij, a
per-subcommand table, and the ten slowest individual calls.

## Why this exists

#215 raised couch's registration deadline to 15s against a measured 8.85s
startup, but nothing measured where those 8.85s go. Two candidates are already
dead by measurement:

| suspect | measured | verdict |
|---|---|---|
| nvim + plugins | 117 ms (`nvim --startuptime`) | not it |
| `zellij list-sessions` | 43 ms at 26 sessions | not it alone |

So the cost is either the number of zellij calls or something outside zellij
entirely. This tells you which, instead of inviting another guess — #215 was
first filed against a root cause that measurement refuted.

## Reading it

`in-zellij time` well under `wall span` means the time is NOT in these
subprocesses — look at what pair does between them. Close to the span means the
call count is the problem, and the per-subcommand table says which kind.
