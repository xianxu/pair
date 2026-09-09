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
