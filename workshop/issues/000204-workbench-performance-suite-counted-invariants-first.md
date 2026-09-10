---
id: 000204
status: open
deps: []
target: workbench-latency
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
---

# workbench performance suite: counted invariants first

## Problem

Three interactive-latency defects (`#201`, `#202`, `#203`) shipped and survived
for months. None would have been caught by review, and none had a failing test —
because nothing in the tree measures what an interaction costs.

The operator's account of how they were found is the requirement: *"I didn't ask
for perf test as I thought couch can't be that slow."* Intuition about a
multi-process chain is unreliable — a keystroke crosses 8–10 process wake-ups and
each hop reads like a function call at its call site. Absent measurement, the
first signal is an operator saying typing feels slow, months late.

**The trap this issue must avoid is a timing suite.** Every wall-clock number
taken during the debugging session moved 2–8× with ambient activity:

| measurement | busy host | quiet host |
|---|---|---|
| `zellij action` round-trip | 145 ms (max 467) | 17.6 ms |
| `alt+Return` | ~290–930 ms | 35 ms |
| process wake-up delay | 4.92 ms | 2.50 ms |

A CI job asserting thresholds on those would flake within a month and be muted.
Worse, the session's one *controlled* experiment returned a confident false
negative — 12 CPU burners moved wake-up delay 2.51 → 2.21 ms, i.e. nothing —
because steady CPU burn does not represent a real build's process-spawn storm.
Reproducible and wrong.

**The counted facts from the same session held under every condition**, and each
names its defect directly rather than a symptom:

| | invariant | today |
|---|---|---|
| `#201` | `alt+Return` performs **one** zellij round-trip | two |
| `#202` | the agent span file is read **once per change** | once per keystroke |
| `#203` | build fan-out is bounded by **cores**, not sessions | sessions × cores |
| `#209` | a thread/tab switch issues a **repaint request**, not only a replay write | landed 2026-09-09 |

`#209`'s row is the fourth, and it arrives with the operating envelope
`ARCH-CONSTRAINTS` asks for rather than a prose assurance. The request is a
SIGWINCH nudge (shrink one row, settle, restore), and `probes/zellijrepaint`
measured what it costs and what it needs on zellij 0.44.3 / macOS:

| | measured |
|---|---|
| bytes zellij re-renders per nudge, single pane | 6.6 KB back-to-back, 19.3 KB after a 1.5 s gap |
| repaint rate with NO settle between the two ioctls | **6 of 12 runs** — a coin flip |
| repaint rate with a 1 ms settle | 5 of 5 |
| nudges per switch | 1, on the keystroke path |

The first two rows are the ones worth a tier-1 count. The byte figure is a
single-pane session and couch runs 10+ panes, so it is a floor rather than the
number; and the coin-flip row is why the count must be of a nudge that ACTUALLY
repaints, not of a `Resize` call — a counted invariant over the call would have
been green for a fix that worked half the time.

## Spec

**A performance suite whose tier-1 assertions are counts, not times.**

- **Tier 1 — counted invariants (blocking).** Deterministic, fast, host- and
  load-independent, runnable in CI. Assert operation counts at seams: subprocess
  spawns per interaction, file reads per keystroke, threads requested per
  session. These are the promises, and the three above are the opening cases.
- **Tier 2 — timings (non-blocking).** A regression net. Every reading records
  the conditions that qualify it — for this workbench, **the number of
  concurrently working agents**, which moved latency 8× on 2026-09-06 with the
  session population unchanged. A timing without its agent count is not
  recorded, because it cannot be compared to anything.

Tier 2 never fails a build. It reports drift; a human decides whether the drift
is real. This is the `under-specify` discipline applied to measurement: spend
the operator's attention on three promises that cannot flake rather than on
thirty thresholds that will.

### To settle in the plan

1. **Where the counting seam goes.** Counting subprocess spawns and file reads
   needs an injection point. `ARCH-PURE`'s existing seams are the candidates —
   `RunZellijAction` is already an interface method (`termcmd`), and the nvim
   side would need something equivalent. Prefer counting at a seam the
   production path already goes through over instrumenting the OS.
2. **Whether the nvim side can be counted in-process.** `#202`'s invariant is a
   Lua-level file read; `nvim/annotate_test.lua` is the precedent for testing
   Lua directly without a buffer.
3. **What tier 2 runs against, and how often.** A fake workload with a known
   agent count is reproducible where the operator's real fleet is not.

Out of scope: fixing `#201`/`#202`/`#203` — this issue builds the net that would
have caught them, and each fix lands against its own issue. Ideally the
invariants land **first**, red, and each fix turns one green.

## Done when

- The three invariants above exist as tier-1 tests, and each **fails against
  today's tree** before its fix lands — a green test on unfixed code is not a
  test of anything.
- Tier-1 tests pass on a busy host and a quiet one with identical results;
  demonstrated, not assumed.
- Tier 2 records agent count alongside every timing, and cannot fail a build.
- Adding a fourth invariant is documented in a way that does not require reading
  this issue.
- `workshop/targets/workbench-latency.md` is the referenced commitment
  (`target:` frontmatter set), and the suite's structure follows it.

## Plan

- [ ] Settle the three plan questions above — especially the counting seam.
- [ ] Land `#201`'s invariant red; likewise `#202`'s and `#203`'s.
- [ ] Verify identical tier-1 results on a loaded vs quiet host.
- [ ] Add tier-2 timing capture with mandatory agent-count recording.
- [ ] Document how to add an invariant.

## Log

### 2026-09-06

Filed out of the session that produced `#201`/`#202`/`#203`. The suite's design
is downstream of that session's own failures rather than of a methodology
preference: three of its wall-clock findings were off by 2–8×, one controlled
experiment produced a confident null from an unrepresentative load shape, and one
promising code-read hypothesis (`#202` as the cause of typing lag) died to a
0.97 ms benchmark. The counted facts were the only ones that survived contact
with a different machine state.

Governed by `workshop/targets/workbench-latency.md`, written alongside this
issue, and by `ariadne#216`, which amends `ARCH-CONSTRAINTS` so a measured basis
must name the probe that reproduces it.
