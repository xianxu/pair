---
id: 000201
status: open
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
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

## Spec

**Stop paying a process spawn per zellij operation on interactive paths.**

Three directions, cheapest first; the plan picks after measuring which of the
7.6 ms (binary load) and 26.7 ms (connect + round-trip) actually dominates in
the real call, since `query-tab-names` may not be representative of
`write-chars`:

1. **Collapse the two calls into one.** If `write-chars` can carry the
   submitting Return in the same payload, `alt+Return` halves immediately with
   no architectural change. Check whether the two-step exists for a reason —
   the agent may need the paste to settle before the Return, in which case the
   gap is deliberate and the fix is a different one. **Settle this first; it may
   be the whole issue.**
2. **Hold a persistent connection.** pair already runs long-lived processes next
   to every session (`pair term`, the title poller). One that keeps the zellij
   socket open turns each action into a socket write rather than a spawn.
3. **Bypass zellij for writes we own.** couch already owns the child pty and
   `pair term` owns its pane's; writing bytes directly is cheaper than asking
   zellij to do it. Bounded by which panes we actually own.

Out of scope: the load that multiplies this 4× — that is `#203`. This issue is
about the 72 ms floor that exists on an idle machine.

## Done when

- `alt+Return` is a single round-trip, or none.
- Measured before/after on a calm machine, reported as medians with the same
  method as above — the floor must actually move, not just feel better.
- The `rename-pane`-per-title-change path (`termcmd/run.go:959`) is addressed or
  explicitly excluded with a reason.
- `clipcmd`'s two-attempt focus (`runtime.go:142,145`) no longer costs two
  spawns on the miss path.
- No regression in `#125`'s copy-on-select handoff or the source-pane gate.

## Plan

- [ ] Settle Spec item 1 — can the two calls be one? Check whether the paste
      must settle before the Return.
- [ ] Measure `write-chars` specifically (not `query-tab-names`) and split its
      cost between binary load and round-trip.
- [ ] Implement the cheapest sufficient direction.
- [ ] Re-measure `alt+Return` end to end, calm and under load.
- [ ] Sweep the other `RunZellijAction` interactive callers listed above.

## Log

### 2026-09-06

Found while debugging an operator report of general workbench sluggishness. Three
distinct causes came out of that session and are filed separately: this one
(`#201`), the draft's per-keystroke work (`#202`), and unbounded build
parallelism across couch sessions (`#203`).

Method note worth keeping: the decomposition above (`true` → `--version` →
`action`) is what turned "zellij feels slow" into "7.6 ms of binary load and
26.7 ms of round-trip", which is what makes the three fix directions
distinguishable. A single end-to-end timing would not have.

Correction recorded because it shaped the filing: the session's first hypothesis
was that machine load explained the slowness. A controlled experiment falsified
it as a *general* cause — 12 CPU burners at normal priority moved wake-up
latency 2.51 ms → 2.21 ms, i.e. not at all, because macOS boosts recently-blocked
threads. Load turned out to matter **specifically** for this multi-stage
spawn+connect path (36 ms → 145 ms), which is why the two effects are filed as
separate issues rather than one.
