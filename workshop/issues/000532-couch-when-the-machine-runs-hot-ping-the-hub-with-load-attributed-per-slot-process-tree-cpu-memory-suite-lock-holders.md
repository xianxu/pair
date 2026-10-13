---
id: 000532
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '4b2462a98e7508a53c462d4c101ef72fc9f4554b' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: when the machine runs hot, ping the hub with load attributed per slot (process tree CPU/memory, suite-lock holders)

## Problem

On 2026-10-13 load reached 185 and then 274 (two ariadne sharded suites at once, 74 sdlc.test processes) and nothing told the hub; the TL found it only when a worker reported a timing-test failure. Couch already walks each slot's process tree (pair#502's renicer scans every 5 s at ~0.44 ms; doctor's perf capture), so attribution is nearly free.

## Spec

The operator (2026-10-13):
1. Couch samples machine load continuously (cheap). When it runs hot (e.g. 1-minute load above 2x cores for 2 minutes; thresholds configurable), it sends the hub one message: 'system load high: any load we can shed?', with the evidence: per slot, the summed CPU% and RSS of its process tree and its top processes (command, CPU, RSS), plus the current test-suite lock holders (ariadne#341/#346). Rate-limited (one per episode, a follow-up only if it gets worse), and a 'cooled down' note when it drops.
2. Reuse the existing process-tree walk (pair#502/doctor), not a new scanner.
3. The couch skill tells the hub what to do with it: check concurrent heavy work (full suites first), ask the heaviest non-essential slot to pause or defer, and remember in-session which commands proved heavy.
4. Later: record each full suite's measured cost (duration, peak CPU) when its lock is released, so the hub can judge fast.

## Done when

- A synthetic hot episode (fake load source and process table) produces one attributed hub message naming the slots and their top processes, a cool-down note, and no repeat within an episode; the skill carries the hub's response.

## Plan

- [ ]

## Log

### 2026-10-13
