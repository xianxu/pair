---
id: 000208
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours: 3.35
started: 2026-09-06T22:53:26-07:00
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

## Spec

**`:PairDoctor` captures a performance snapshot at the moment it is invoked, and
hands the agent the numbers rather than an instruction to go measure later.**

### The architectural difference from today's payload, stated up front

Today `doctor.lua` is a **pointer**: it builds an instruction and the agent runs
`doctor.sh` whenever it gets to it. That is correct for drift, which is durable —
the flight recorder is still there a minute later.

It is wrong for performance. The measurement must happen **at invocation**,
because the condition is transient and is often gone by the time an agent
responds. So the perf half captures first and reports second. Keep the existing
seam — a pure builder plus a thin IO wrapper — but the builder now formats
*captured data*, not just paths.

### What only nvim can measure, which is the point

The discriminator today's investigation lacked: **is the editor slow, or is the
environment slow?** nvim is the only vantage point that can answer it, because
it can time its own loop:

- **nvim's own input handling** — time to process a keystroke internally, and
  the cost of the `TextChangedI` autocmd chain (which is where `#202` lives, and
  which can now be measured in situ rather than benchmarked in a headless
  fixture).
- **nvim's redraw time.**

If nvim's internal handling is fast while typing feels slow, the problem is at
or above the terminal — transport, rendering, compositing — and the whole
scheduling family (`#201`/`#203`) is excluded for that symptom. If it is slow,
the cause is inside nvim and the environment probes are noise. **Today's session
could not make that call, and it is the single most valuable bit.**

### Environment probes to capture alongside

Built and validated on 2026-09-06; see `## Log` for the harness bugs they must
not repeat:

- **pipe round-trip** — one scheduler wake-up, isolated. Baseline ~7 µs.
- **fork+exec** — raw process creation. Baseline ~1.5 ms (matches `#201`'s 1.9 ms).
- **`zellij action` round-trip** — the spawn+link+connect path `#201`/`#203`
  care about. Baseline ~13 ms.
- **Render path** — WindowServer CPU, and CPU idle% alongside it. This is the
  untested candidate; capturing it is how it stops being untested.
- **Conditions** — load average, running-agent count, and the go compile/link
  process count, so a reading is interpretable later.

### To settle in the plan

1. **Budget.** A diagnostic that itself takes seconds while the machine is
   struggling is a bad citizen. Decide the wall-clock ceiling and make the probe
   set fit it; the pipe and fork probes are microseconds-to-milliseconds, but
   `zellij action` at a degraded 145 ms x N samples is not free.
2. **Whether `:PairDoctor` gains a mode or always does both.** Drift and perf
   are different questions with different urgencies. `:PairDoctor perf` vs
   always-both is a UX call, not a technical one.
3. **Where the probe binaries live.** The pipe probe needs a real second
   process. `doctor/` already ships shell; a tiny Go helper under `cmd/` is the
   alternative and is testable, but adds a build artifact.
4. **Whether captures accumulate.** One snapshot answers "what is happening now";
   a series answers "what changed". A rolling file under the session's data dir
   would make the *next* investigation comparative instead of absolute — which
   is precisely what today's lacked.

## Done when

- `:PairDoctor` reports, in one invocation from the draft pane: nvim's own
  input-handling and redraw cost, the four environment probes, and the
  conditions they were taken under.
- The report distinguishes **"nvim is slow"** from **"the environment is slow"**
  explicitly, rather than leaving the reader to infer it.
- Invoking it while the machine is healthy produces a baseline row that a later
  degraded row can be compared against.
- The perf probes are unit-tested where pure, and the timing harness is verified
  against a known quantity (`/usr/bin/true` must read ~1.5-2 ms, not 18 —
  see Log).
- `doctor/SKILL.md` gains the perf procedure, single-sourced as today.
- The drift path is unchanged: same payload, same behaviour, no regression.

## Plan

Two milestones; detail in `workshop/plans/000208-pairdoctor-perf-capture-plan.md`.

- [ ] M1 — `cmd/hoprtt` (pipe-hop probe plus a shared in-process spawn timer) and
      `doctor/perf.sh` (the snapshot). The timing harness is validated against a
      known quantity FIRST: `/usr/bin/true` must read 1-4 ms, never 18.
- [ ] M2 — nvim: the draft buffer becomes the operator's note, nvim times its own
      input handling and redraw (the editor-vs-environment discriminator), and
      `:PairDoctor` sends note + timings + snapshot. Drift path unchanged.

**Operator additions folded in (2026-09-06):** the buffer is consumed as a
free-text note — *"the system slowed down about 5 minutes ago, typing became very
slow, even loading Activity Monitor is slow, top took 10 seconds"* — and the
snapshot covers system load, running processes, and **per-process resource usage
over a window**: a delta between two samples, never `ps %cpu`, which is a
lifetime average and hid a 42.6% spinner from this very session.
## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                 design=0.90 impl=0.10
item: greenfield-go-module       design=0.15 impl=0.20
item: greenfield-go-module       design=0.20 impl=0.24
item: smaller-go-module          design=0.02 impl=0.06
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.08 impl=0.20
item: smaller-go-module          design=0.08 impl=0.20
item: atlas-docs                 design=0.03 impl=0.08
item: milestone-review           design=0.00 impl=0.16
item: milestone-review           design=0.00 impl=0.16
design-buffer: 0.15
total: 3.35
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* Calibration source reports `stale`, so the
per-primitive hours are provisional.

`issue-spec` at 0.90 covers the issue, the plan, and **four** plan-quality
rounds — the gate found the async capture would freeze the editor (every
shell-out in `nvim/` is synchronous `vim.fn.system`), that the probe would never
have been built (`GO_BINS` is hand-maintained and overrides the base-layer
scan), that the pid join was sitting in untestable shell, and that the
discriminator's synthetic keystroke would have corrupted the operator's note.
Four rounds is high, and each one changed the design rather than the wording.

| Slug | Instances |
| --- | --- |
| `issue-spec` | the issue, the plan, four plan-quality rounds |
| `greenfield-go-module` | `cmd/pair-hoprtt` (probe + shared in-process timer); `doctor/perf.sh` (the snapshot) |
| `smaller-go-module` | the `GO_BINS` entry + recipe; `doctor.lua`'s pure trio (`note_from_lines`, `perf_payload`, `verdict`); `doctor.delta` + fixtures; the async `init.lua` wiring |
| `atlas-docs` | `doctor/SKILL.md`'s perf procedure |
| `milestone-review` | M1 and M2 boundaries |

## Log

### 2026-09-06

Filed after a performance investigation that produced three issues and no
theory. The operator's summary of it — *"so seems you don't have a good theory,
right?"* — was correct, and the reason is worth carrying: **the symptom is
intermittent and every measurement was taken while the machine was healthy.**
Synthesizing the condition failed (a deliberate spawn storm reached 1.9x where
the fleet produces 8x), so the fix is to capture rather than reconstruct.

**Two harness bugs from that session, both caught by a positive control**, and
both worth not repeating here:

1. **The first timing harness spawned `python3` twice per sample** to read a
   clock, so it measured python startup: `/usr/bin/true` read **18.7 ms**
   against `#201`'s known 1.9 ms. Timing must be in-process. Any perf probe
   added here needs a known-quantity check before its numbers are trusted.
2. **The first synthetic load did nothing** — four `go test` runs against a warm
   build cache barely compile. Load stayed at 2.76 and the `zellij action`
   control moved 2%. Without that control the null result would have read as
   "load does not matter", which is exactly the wrong conclusion `#203` had
   already recorded and retracted once.

The general rule both instances share, and the reason this issue exists: **a
measurement without a positive control cannot distinguish "no effect" from "my
instrument is broken".**

Related: `#201` (round-trip cost, stands on its own arithmetic), `#202`
(refuted as a lag cause by its own benchmark; its hop-count hypothesis is now
also refuted — see the table above), `#203` (real oversubscription, but its
scope note correctly disclaims typing lag).
