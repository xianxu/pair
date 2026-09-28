---
id: '000201'
status: punt
started: 2026-09-06T22:06:07-07:00
created: 2026-09-06
updated: 2026-09-09
---

# alt+Return pays two zellij action subprocess round-trips

## Problem

Operator report: `alt+Return` from the draft pane visibly pastes into the agent
pane, **hangs for about a second**, then the submitting Return arrives. The
pause between the two halves is the symptom, and it is the mechanism showing
through.

`alt+Return` is not one operation. It is two `zellij action` invocations —
`write-chars` then the focus/Return — and each is a **fresh subprocess** that
loads the zellij binary, connects to the server socket, and waits for a round
trip.

### Measured, on this machine (12 cores, M-series)

Decomposed by differencing three timings:

| step | median | what it adds |
|---|---|---|
| `/usr/bin/true` | 1.9 ms | OS process-spawn floor |
| `zellij --version` | 9.5 ms | + loading the zellij binary (**7.6 ms**) |
| `zellij --session S action query-tab-names` | 36.2 ms | + connect & server round-trip (**26.7 ms**) |

So on a **calm** machine `alt+Return` costs **≥72 ms** before any useful work.

**Corrected 2026-09-06:** re-measured on a genuinely quiet host (2 agents, load
2.3) the round-trip is **17.6 ms**, so the true floor is **~35 ms**, not 72 —
the earlier 36 ms reading was itself taken under residual load. The floor is
lower than first filed and still worth removing; the numbers below stand as the
loaded case.
Under the load this fleet actually runs at (concurrent `go test` across sessions,
load 25–100 on 12 cores) the same call measured **145 ms median, 467 ms max** —
putting `alt+Return` at **290 ms typical and ~930 ms at the tail**, which is the
"about a second" the operator sees.

Note the shape: the subprocess path is **4× more load-sensitive** than a plain
process wake-up. A controlled experiment (12 CPU burners at normal priority)
moved simple wake-up latency by only 0.3 ms — but spawn→dynamic-link→connect→
round-trip degrades hard, because it is four scheduling points, not one.

### The blast radius is wider than alt+Return

`zellij action` as subprocess-per-operation is used across pair, so every one of
these pays the same 36 ms floor:

- `termcmd/run.go:191` — `RunZellijAction("focus-pane-id", ...)`
- `termcmd/run.go:959-963` — `rename-pane`, on **every tab title change**
- `layoutcmd/layoutcmd.go:57` — `focus-pane-id`
- `clipcmd/runtime.go:142,145` — `focus-pane-id`, twice (bare then `terminal_`
  form), on the copy-on-select handoff path (`#125`)

`clipcmd` is the notable one: it tries the bare form, and on failure tries the
prefixed form — so a miss costs two full round-trips before the paste lands.
