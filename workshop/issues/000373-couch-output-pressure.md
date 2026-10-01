---
id: 000373
status: working
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '2ae76a809058ce46db3ac65d30c641cd6664eea7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T15:41:41-07:00
flow: {kind: quick, provenance: inferred, spec: "41f4a89d", done: "ee6e183f"}
---

# Isolate Couch pane latency under output pressure

## Problem

Across Couch slots, draft/agent/right panes stalled while Couch switching stayed
responsive. During a live capture, all 12 Zellij clients stayed in write;
two resampled after recovery spent only about 12% of observations there.
System CPU idle changed from 0% to 63% after 12 ariadne test shards exited.
This is correlated evidence, not proof that Couch throughput is the cause.
A separate cmux terminal remained responsive. Raw captures: /tmp/pair-stall-capture.

## Spec

Build an isolated, bounded experiment around the production Couch terminal path.
Compare normal output, multi-child bursts, reduced CPU capacity and a delayed
host sink. Measure child-output progress, input-to-child/display latency, and
switcher latency independently. Reuse existing fake-host and PTY soak seams
(ARCH-DRY, ARCH-PURE); do not touch running sessions or use machine-wide load.
A positive reproduction requires responsive Couch controls alongside delayed
child output/input; slowing everything does not establish the reported cause.
Confirm useful findings with real PTYs. Record configuration, repeated controls,
results and limitations. No speculative production fix is part of this issue.
Bound each trial and its shutdown; retain only bounded output/counters. No new
runtime storage or external services (ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-FUNERAL).

## Done when

- A repeatable isolated experiment exercises production Couch output delivery.
- Paired trials report separate output, pane latency and Couch-control measurements.
- Results explicitly say whether the selective stall reproduced and which hypotheses remain.
- Real PTYs check the relevant fake-path findings; no live sessions are mutated.
- Evidence and commands are recorded durably; focused tests and cleanup checks pass.


## Plan

- [ ] Implement bounded fake and real-PTY experiments using existing test seams.
- [ ] Run baseline and single-factor pressure trials; inspect profiles only where needed.
- [ ] Record findings, limitations and justified next steps; verify and close investigation.

Experiment controls: use the same 12 children and geometry in paired trials,
three repetitions, bounded two-second pressure windows and five-second recovery
deadlines. Compare baseline output with a bounded burst, the same burst with
GOMAXPROCS=1, and baseline output with a 50ms host-write delay. GOMAXPROCS limits
only Couch's Go execution capacity, not PTY subprocess CPU. Define a selective
stall as pane latency exceeding one second while rendered switcher response
stays below 100ms; report all raw latencies even if neither threshold is crossed.
These diagnostic thresholds distinguish visible freezing from ordinary jitter;
they do not equate a one-second reproduction with the reported minute-long stalls.

Measure input receipt through an independent bounded child side channel, output
ingestion separately from the final host screen, and switcher latency by sending
the real shortcut through host input and observing the rendered menu. A forced
host delay is a backpressure control, not proof of the incident's cause. Workload
rates, byte budgets and exact commands must accompany results. Join every helper
on completion or cancellation (ARCH-ORDER); make no production behavior change.


## Log

### 2026-10-01

Operator requested testing the throughput hypothesis in isolation. This investigation
is distinct from #370's agent-wrapper CPU work: the symptom affects sibling panes.

Spec review identified two validity requirements, now in Plan: independently
observe child receipt versus displayed output, and predefine the intervention,
latency criterion, repetitions and recovery bounds before running trials.
