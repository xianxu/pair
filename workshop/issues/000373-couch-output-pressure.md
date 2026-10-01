---
id: 000373
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'ffb20960a525e78220c88f6c0c30d225b7d9ac0b' # card fields mirrored from issue-cards; edit via sdlc
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


## Log

### 2026-10-01

Operator requested testing the throughput hypothesis in isolation. This investigation
is distinct from #370's agent-wrapper CPU work: the symptom affects sibling panes.

