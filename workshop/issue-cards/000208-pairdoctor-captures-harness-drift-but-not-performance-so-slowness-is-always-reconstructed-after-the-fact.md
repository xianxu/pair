---
id: '000208'
status: done
started: 2026-09-06T22:53:26-07:00
created: 2026-09-06
updated: 2026-09-07
estimate_hours: 3.35
actual_hours: 7.34
---

# PairDoctor captures harness drift but not performance, so slowness is always reconstructed after the fact

## Problem

`:PairDoctor` (`#48`) is the agent-agnostic diagnostic entry: `nvim/doctor.lua`
builds an instruction, `init.lua` hands it to whatever agent is running, and the
procedure stays single-sourced in `doctor/SKILL.md`. It covers exactly one
question — **harness adaptation drift**, read from the flight recorder.

Performance has no equivalent, and the gap has a shape worth naming: **every
performance investigation in this repo has reconstructed the symptom instead of
capturing it.**

### The evidence that this is the actual blocker

A full session on 2026-09-06 tried to explain an operator report of "even typing
in nvim is slow, but CPU was fine". It produced three filed issues
(`#201`/`#202`/`#203`) and **no working theory**, because the machine was
healthy the whole time it was being measured:

| candidate | outcome |
|---|---|
| draft completion rebuilding per keystroke | **ruled out** — 0.97 ms, benchmarked (`#202`) |
| hop count on the input path (~8-10 wake-ups/char) | **refuted by arithmetic** — a pipe hop is 7 µs; even the 2.8x degradation induced under a real spawn storm leaves ten hops at 0.17 ms. Visible lag needs ~10 ms/hop, a ~1500x degradation nothing approached |
| machine load | **not the variable** — measured degradation at load 9.5 and none at load 15.5; the workload's *phase* (running short-lived test binaries) correlated, load did not |
| spawn-storm contention | **real but insufficient** — 2-3x on pipe/fork/zellij together, all in microseconds |
| `#203`'s 8x `zellij action` figure | **unreproduced** — a synthesized storm reached 1.9x |
| the render path (WindowServer 47% at 66% idle CPU) | **untested** — correlational only, and the one candidate whose natural units are tens of ms |

The pattern: five hypotheses, four settled by measurement, and the one that
survives is the one nobody measured — because measuring it requires being there
when it happens.

**`:PairDoctor` is already the right trigger.** It is invoked from nvim, in the
draft pane, which is exactly where the operator is standing at the moment typing
feels slow. What is missing is that it captures nothing.
