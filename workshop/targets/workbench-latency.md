---
type: target
slug: workbench-latency
status: active
created: 2026-09-06
updated: 2026-09-06
sources: ["pair#201", "pair#202", "pair#203", "ariadne#216"]
---

# Workbench latency — what we defend, and how we know

The workbench is a chain of processes, not an application. A keystroke crosses
Ghostty → couch → the zellij client → a socket → the zellij server → nvim, and
the render path returns the same way: roughly **8–10 process wake-ups per
visible character**. Every hop looks free at the call site. None is.

That is the whole reason this target exists. In a single-process editor, latency
is something you notice while writing the code. Here it is distributed across
components that each look innocent, and it only becomes visible to the operator
— as "typing feels slow" — long after the commit that caused it. Intuition is
not a defence: this fleet's operator ran the workbench for months believing
couch "can't be that slow", and was wrong for reasons no code review would have
surfaced.

## What we defend

**Work per interaction is bounded by what changed, not by what exists.**

The three defects that crystallized this target are all one shape — a cost that
scales with the wrong thing:

| defect | scales with | should scale with |
|---|---|---|
| `pair#201` | two subprocess spawns per submit | one round-trip, or none |
| `pair#202` | file size × keystrokes | changes to the file |
| `pair#203` | sessions × cores | cores |

None of these was a slow algorithm. Each was correct code whose cost was tied to
a quantity that grew later — an LRU reaching its cap, a session count rising,
a convenience call placed on an interactive path.

## How we know — the measurement discipline

**Prefer a counted invariant to a timing.** A count is machine-independent,
cannot flake, and names the defect rather than a symptom. A timing is neither
portable nor stable: on 2026-09-06 the same `zellij action` call measured 17.6 ms
on a quiet host and 145 ms (max 467 ms) on a busy one — an 8× spread with no code
change. A threshold written from either reading would have been wrong.

So:

- **Tier 1 — counted invariants.** Human-reviewed promises. "One round-trip per
  submit." "One read per change, not per keystroke." "Fan-out bounded by cores,
  not by sessions." These are the target's actual content, and they hold under
  every load.
- **Tier 2 — timings.** A regression net, useful for direction and never as a
  promise. A timing is meaningless unless it records the conditions that produce
  it.

**Co-tenancy is part of the envelope, not context.** The single most important
number beside any latency reading here is *how many agents were working*. On
2026-09-06 the workbench was unusable at 8 working agents and fine at 2 — with
the session population unchanged (it grew, 10 → 11). Idle sessions are free;
concurrently working agents are not. A latency number without its agent count is
not a measurement.

**A number with no procedure is aspiration.** Anything recorded here names the
command that reproduces it. This is the local instance of `ARCH-CONSTRAINTS` as
amended by `ariadne#216`: the basis for a measured fact is a probe, not a label.

## What this target does not do

It does not set thresholds. Hosts differ, and a portable threshold is either
wrong somewhere or asserts nothing. It commits to the *shape* of the promise and
to the ability to re-take the measurement — the numbers themselves live with the
runs that produced them.

Deliberately under-specified per the datatype: the suite (`pair#204`) derives
specifics. Refine this only when a later read shows the commitment was missing
something it had to honor.
