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

- [x] Implement bounded fake and real-PTY experiments using existing test seams.
- [x] Run baseline and single-factor pressure trials; inspect profiles only where needed.
- [x] Record findings, limitations and justified next steps; verify and close investigation.

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

Spec re-review accepted those changes. Preliminary controls pass: publication
queue bounds/cancellation tests and the existing real-PTY Couch soak (eight
iterations, maximum input-visible latency 1.79ms). The live host measures
191 columns by 54 rows; use that geometry for the pressure trials so a small
fixture cannot hide whole-screen processing costs. Go version: 1.27.1 darwin/arm64.

Harness inspection identified a confound before full trials: measuring the menu
only after displayed ACK could place the control probe after pressure ended.
Use fixed-time menu input while pressure is active, record whether pane ACK was
visible before the overlay, and never infer delay from a deliberately hidden
pane. Limit snapshot observation cadence to avoid making the observer the load.

### Isolated results (2026-10-01)

Final command: `PAIR_COUCH_PRESSURE=1 go test ./cmd/internal/couchtty -run
'^TestCouchOutputPressure$' -count=1 -v -timeout=180s`. All 24 trials passed
in 49.739s (two transports × four conditions × three repetitions). Log:
`/tmp/pair-373-pressure-final.log`. No selective stall and no censored ACKs.
All trials ingested exactly the emitted bytes plus 144 bytes of readiness output.
The fixture uses an actual 12-entry switcher and routes both Ctrl+Space and Escape
through production input/operation dispatch. An earlier fixture lacked that
operation dispatcher; it was corrected before these final measurements.

Maximum latency across three repetitions, milliseconds:

| Transport | Condition | Child receipt | Displayed ACK | Rendered menu |
| --- | --- | ---: | ---: | ---: |
| Fake | Baseline | 1.1 | 21.6 | 3.7 |
| Fake | Burst | 1.8 | 12.4 | 5.1 |
| Fake | Burst, one Go CPU | 4.8 | 15.6 | 10.7 |
| Fake | Slow host | 19.9 | 94.1 | 54.8 |
| Real PTY | Baseline | 1.4 | 24.4 | 5.8 |
| Real PTY | Burst | 1.5 | 45.4 | 8.6 |
| Real PTY | Burst, one Go CPU | 0.7 | 43.9 | 18.4 |
| Real PTY | Slow host | 20.6 | 99.9 | 67.4 |

Baseline emits about 24.5KB across 12 children in two seconds (100ms ticks).
Burst uses 10ms ticks with about 4KB per child per tick, capped below 9.375MiB;
observed totals were 9.2–9.73MB. Missed ticks are skipped, not accumulated.
Normal parent GOMAXPROCS was 3; reduced capacity was 1. Children and host use
191-column geometry; the 54-row host reserves one row outside the child.
Independent PTY side pipes timestamp receipt before output ACK. Display and
endpoint ACK observation has a 10ms polling cadence; menu is probed at 1.5s,
while the two-second output window is still active. Recovery completed within
66ms after the window in this matrix. Helpers and console are joined on teardown.

Conclusion: these controls did not reproduce the incident. Burst throughput
through Couch alone is insufficient evidence for the claimed root cause; the
slow-host control delays the switcher too. This is a bounded negative result,
not proof that Couch cannot stall. The workload repeatedly rewrites a small
screen region: it stresses parsing more than changed-screen rendering. It omits
Zellij, wrappers, Ghostty, long sustained load, and the observed system-wide
kernel/process pressure. Limiting Go CPUs does not simulate those pressures.
A justified next discriminator is an isolated Zellij-backed workload with
representative screen changes, followed by scheduler/write observations during
a real recurrence. No production fix is justified by this experiment.

Verification: `go test -race ./cmd/internal/couchtty -count=1` passed in
23.827s, including the ordinary real-PTY pressure control and joined cleanup.
Publication bounds, sticky failure and child-exit/final-publication focused tests
also passed. `git diff --check` passed. No production behavior was changed.
