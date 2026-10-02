---
id: 000375
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '5f336b511447f57e08411fdde8785580dcaafba8' # card fields mirrored from issue-cards; edit via sdlc
---


# Improve PairDoctor performance telemetry and agent investigation guidance

## Problem

The 2026-10-01 slowdown affected draft nvim, agent scrolling/timer updates and
right panes across Couch slots, while Couch's switcher and a separate cmux
terminal remained responsive. Existing :PairDoctor captures system pressure but
cannot localize terminal backpressure or reliably preserve evidence before an
intermittent stall recovers. The agent needed ad-hoc process topology, live stack
samples and recovery comparisons. Capture that method in the product and its
agent instructions so the next session does not repeat this discovery.

Follow-up to #373 (isolated Couch pressure experiment). Related: #210 owns
collector deadline hardening; #370 investigates wrapper CPU/logging overhead.
This issue records future work, not a claim that the slowdown's cause is known.

## Spec

### Default measurements and retained evidence

Extend the performance side of :PairDoctor, reusing doctor/perf.sh,
nvim/doctor.lua and existing capture artifacts rather than creating a parallel
diagnostic entry point (ARCH-DRY). Distinguish low-cost default instrumentation,
on-demand capture and expensive follow-on probes explicitly.

- Preserve timestamped process identity and topology: Couch, Zellij server and
  attach client, Pair wrapper, editor/agent, owning slot and controlling PTY.
  Resolve the running executable/version rather than assuming the checkout's
  binary is live. Use process start identity to guard PID reuse.
- Retain CPU-counter deltas over measured elapsed time, process churn and relevant
  ancestry, system user/system/idle CPU, memory/swap and I/O pressure. Report
  missing, timed-out and partial observations distinctly from zero/healthy.
- Add low-overhead terminal-path measurements that distinguish PTY read progress
  and bytes/sec, parse/feed duration, publication queue bytes/items and oldest
  age, enqueue/ack wait, input admission versus actual write completion,
  presentation lag and host-write duration. Include all slots, not only the
  selected pane; expose capture age and sampling gaps. Instrument stages rather
  than attributing end-to-end delay to whichever counter is largest.
- Preserve a bounded recent timing window before a stall and its recovery. Define
  a measurable stall trigger and bounded automatic retention, consumable by
  :PairDoctor; allow an explicit capture to pin the relevant window. Choose
  thresholds during design, not from the absence of events. Capture timings and
  counters without terminal contents or keystrokes. State coverage gaps where a
  child receipt/display acknowledgement is unavailable in production.
- Provide an agent-invokable capture path from another responsive terminal when
  nvim or the agent pane cannot process :PairDoctor. Preserve an exact artifact
  path/capture ID, incident time, scope, build identity and healthy/stalled/
  recovered phase so a delayed agent can read the correct evidence later.
- Budget collection latency, CPU, memory, file growth and concurrent collectors;
  benchmark idle and 12-slot active overhead. A recorder must not block terminal
  draining or amplify a process storm. Bound stages/cancellation and join helpers
  in coordination with #210. Define rotation/expiry and incident retention
  (ARCH-CONSTRAINTS, ARCH-ORDER, ARCH-FUNERAL).

### Agent instructions for continuing investigation

Update doctor/SKILL.md's performance procedure and the generated agent handoff:

1. Read the exact capture artifact and operator's symptom first. Establish whether
   the stall is current; distinguish one pane, sibling panes, multiple slots,
   Couch controls and an independent terminal. A healthy peer narrows scope but
   does not exclude shared CPU/scheduler pressure with unequal effects.
2. During a recurrence, collect bounded stack samples of Couch plus representative
   Zellij clients/servers and the process/PTY relationships. Compare the same
   processes during recovery. Sample wait stacks are not CPU usage, and a write
   stack alone does not identify its file descriptor; corroborate where possible.
3. Compare measured CPU deltas, system/user split, process churn/ancestry and
   terminal-stage progress over aligned time windows. Preserve observations,
   hypotheses, confidence and falsifying evidence separately. Correlation with
   a workload ending is not proof of causation; do not kill user workloads as
   a diagnostic shortcut without authorization.
4. Correct current overclaims: a fast editor probe does not rule out scheduler
   or terminal delay elsewhere/earlier; one process above 50% is a lead, not
   automatically the cause; low sampled per-process rates do not exclude all CPU
   explanations (short-lived/aggregate work and missed windows matter); past
   idle-CPU incidents do not imply every future slowdown has idle CPU. Fast pipe
   hops or one responsive terminal cannot globally exclude scheduling.
5. If evidence is insufficient, propose the next discriminating measurement or
   bounded isolated experiment. Reuse #373's fake/real-PTY seams where applicable:
   independent child receipt, endpoint output and rendered control latency;
   control probes while pressure is active; normal/pressure/recovery comparisons;
   real geometry/slot count and repeated trials. Prefer representative Zellij and
   screen-change workloads when plain Couch traffic fails to reproduce. Do not
   recreate machine-wide overload or modify production behavior speculatively.
6. State experiment limits: GOMAXPROCS constrains parent Go execution, not child
   CPU or kernel pressure; repeated writes to the same region stress parsing more
   than rendering; a short negative reproduction does not exonerate the omitted
   layers. Prove stream completion through in-band evidence, not a side-channel
   producer-done signal or publication flush alone. Bound cleanup and verify
   failure-path tests fail when the guarantee is removed.

### Evidence to carry forward

During the live incident, 12 Zellij attach clients spent nearly entire three-second
samples inside write; corresponding PTYs were owned by Couch, although syscall
FDs were not directly observed. Server sendto waits suggested propagated pressure.
Two clients resampled after recovery spent about 12% in write. System CPU idle
changed from 0% to about 63% around the exit of 12 parallel ariadne test shards;
user confirmed recovery then. This supports backpressure correlated with overload,
not a proven Couch throughput defect. cmux remained responsive.

#373's verified 24-trial matrix used 12 children at 191x54 host geometry, fake and
real PTYs, two-second windows and three repetitions per condition. Burst traffic
was roughly 9.7MB/window; real-PTY visible responses stayed below 42ms including
one-Go-CPU trials. A 50ms host-write delay slowed both pane and switcher. No
selective stall reproduced. The test omitted Zellij, Ghostty, sustained realistic
screen changes and system-wide process/kernel pressure. It made no production
telemetry change. Durable numbers/method are in #373; raw /tmp/pair-stall-capture
and /tmp/pair-373-pressure-verified.log are local, ephemeral supporting evidence.

## Done when

- A normal :PairDoctor performance capture identifies the running terminal/process
  path and includes default terminal-stage progress/latency plus system pressure,
  with units, elapsed windows and honest unavailable/partial states.
- Bounded recent stall/recovery evidence survives delayed invocation and can be
  captured from a responsive external terminal without depending on draft input.
- Retention and overhead budgets are explicit and verified under idle and active
  multi-slot conditions; stuck probes and teardown cannot block normal terminal I/O.
- The agent handoff points to the exact artifacts and the updated single-sourced
  procedure; it guides live/recovery sampling, uncertainty and discriminating
  experiments without treating correlations or negative controls as proof.
- Tests cover healthy/slow/missing/stale evidence, blocked collection, recovery
  and real-PTY backpressure with independently observed outcomes. Instruction
  examples cover this cross-slot incident and reject the listed false exclusions.
- README, doctor documentation and atlas explain default versus on-demand data,
  standalone capture, retention and limitations; #210 overlap is resolved explicitly.

## Plan

Implementation design and sequencing are deferred until this follow-up is claimed.

- [ ] Design default telemetry, bounded retention and standalone capture; resolve #210 overlap.
- [ ] Implement measurements and incident evidence with overhead and failure-path verification.
- [ ] Update and verify agent guidance, handoff and user documentation against the session evidence.

## Log

### 2026-10-01

Recorded at operator request after #373. Scope includes both default measurements
and agent guidance learned during the investigation. Left open and unclaimed;
no implementation or deployment is requested by this capture.
