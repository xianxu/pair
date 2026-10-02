---
id: 000373
status: codecomplete
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '82b055bc7d12b982527d8145b60ff2f6a2320293' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T15:41:41-07:00
flow: {kind: quick, provenance: inferred, spec: "41f4a89d", done: "ee6e183f"}
actual_hours: 0.84
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
- Evidence and commands are recorded durably; focused race and cleanup checks pass.
- Recovery requires every child's in-band completion and exact byte accounting;
  a delayed trailing-PTY regression rejects premature completion.
- Blocking trial operations share bounded cancellation and joined teardown,
  exercised by a stalled-operation regression.


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
- 2026-10-01: closed — Revalidated after merging origin/main: only manual conflict was workshop/lessons.md, both independent sets preserved; focused Couch pressure control, stalled-operation and trailing-PTY tests passed 4.361s. Prior 24-trial matrix passed with no selective stall and focused race plus mutation verification passed. Runbook smoke and shell syntax/link checks passed. This remains diagnostic tooling with documented limits, no production stall fix claimed.; review verdict: SHIP
- 2026-10-01: closed — 24-trial matrix passed: no selective stall, real-PTY burst display max 41.91ms, exact stream completion and physical final marker verified. Focused race controls passed 5.537s after shared recovery extraction; scratch mutation removing recovery checks correctly failed the trailing-PTY regression. README/atlas/issue preserve command, results and limits. No production fix claimed.; review verdict: SHIP

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

## Revisions

### 2026-10-01 15:58 PDT — boundary review requires stronger fixture guarantees

The review reproduced an intermittent emulator teardown race despite the earlier
passing suite. It also identified that blocking observations could escape the
trial timeout and a side-channel completion report does not prove the PTY stream
has drained. Preserve the measured latency table as preliminary evidence; rerun
after adding bounded execution and per-child terminal completion checks. The
previous recovery timing is provisional. README now documents the opt-in command.
These are fixture corrections within the original investigation, not a runtime fix.

### Verified rerun after BR-1–4 corrections

The same 24-trial command passed in 50.037s; log
`/tmp/pair-373-pressure-verified.log`. Still no selective stall or censored ACK.
This rerun supersedes the preliminary table for final measurements:

| Transport | Condition | Receipt max ms | Display max ms | Menu max ms | Recovery max ms |
| --- | --- | ---: | ---: | ---: | ---: |
| Fake | Baseline | 0.86 | 21.58 | 3.63 | 7.76 |
| Fake | Burst | 2.07 | 12.89 | 4.64 | 8.89 |
| Fake | Burst, one Go CPU | 6.11 | 25.59 | 13.66 | 17.97 |
| Fake | Slow host | 19.52 | 93.44 | 53.50 | 57.99 |
| Real PTY | Baseline | 1.08 | 26.14 | 3.96 | 9.11 |
| Real PTY | Burst | 1.26 | 33.10 | 3.54 | 13.65 |
| Real PTY | Burst, one Go CPU | 0.97 | 41.91 | 3.30 | 20.53 |
| Real PTY | Slow host | 19.96 | 100.74 | 61.31 | 64.18 |

BR-1: close emulator input pipe, join reader/writers, then close emulator state.
BR-2: tracked cancellable operations bound input and observation waits; a shared
window-end+5s deadline covers recovery, with joined cleanup and a stalled-write
regression. Pipe creation/process launch remain synchronous OS setup operations.
BR-3: every child must show its final in-band byte-count marker; assert exact
emitted-plus-framing bytes and selected physical completion. The trailing-PTY
regression withholds the final marker until producer completion plus publication
flush have demonstrably failed to establish terminal completion.
BR-4: README documents the sole public opt-in command and its limits.

`go test -race ./cmd/internal/couchtty -run
'^TestCouchPressure(StalledOperation|TrailingPTYOutput|Control)$' -count=1 -v
-timeout=60s` passed in 5.45s. The prior full package race suite passed before
these fixture corrections; the focused rerun covers the corrected lifecycle.
`git diff --check` passes. Conclusions and scope limitations are unchanged.

### BR-3 regression strengthened after round 2

Recovery checks are now one shared operation. The trailing-PTY regression invokes
that operation while the final bytes remain withheld and requires deadline
expiration; it then releases output and requires successful recovery. Focused
race controls passed in 5.537s. A scratch Go overlay replacing the recovery
operation with immediate success made the trailing-PTY regression fail in 2.216s
with `got <nil>, want deadline exceeded`, proving it detects the missing fix.
No workload or latency probe changed; the previous 24-trial measurements stand.

### 2026-10-01 — future debugging usability

At operator request, added doctor/terminal-pressure.md as the reusable runbook:
prerequisites, full matrix with provenance/exit-code-preserving evidence capture,
focused subtest command, metric/censoring definitions, interpretation limits and
extension checks. Linked from README, atlas and doctor/SKILL.md so a future
:PairDoctor agent can discover it without this conversation. No harness behavior
changed. The documented focused real-PTY burst command selected exactly one
trial and passed in 2.328s; all shell examples parse with sh -n, relative runbook
links resolve, and git diff --check passes.
