---
id: 000204
status: open
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
github_issue:
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
SIGWINCH nudge (shrink one row, settle, restore), and `cmd/probes/zellijrepaint`
measured what it costs and what it needs on zellij 0.44.3 / macOS:

| | measured |
|---|---|
| bytes zellij re-renders per nudge, single pane | 6.6 KB back-to-back, 19.3 KB after a 1.5 s gap |
| repaint rate with NO settle between the two ioctls | **6 of 12 runs** — a coin flip |
| repaint rate with a 1 ms settle | 5 of 5 |
| nudges per switch | 1 |
| event-loop time per nudge | **0** — the settle runs off the caller's goroutine |
| in-flight nudges per child | 1; a request arriving during one is dropped |

The first two rows are the ones worth a tier-1 count. The byte figure is a
single-pane session and couch runs 10+ panes, so it is a floor rather than the
number; and the coin-flip row is why the count must be of a nudge that ACTUALLY
repaints, not of a `Resize` call — a counted invariant over the call would have
been green for a fix that worked half the time.

The last two rows exist because the envelope was stated once and then changed
underneath itself. "Two SIGWINCHes cost one extra repaint" was costed when the
nudge was two ioctls; adding the 20 ms settle made it 20 ms of an event loop per
switch, which held key-repeat turns into a proportional stall. Moving the settle
off the caller and dropping a request that arrives during one takes it back to
zero. **An envelope stated for one event has to be restated when the per-event
cost changes** — that is the rule this row is here to make hard to skip.
