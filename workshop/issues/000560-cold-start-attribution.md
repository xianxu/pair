---
id: 000560
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '494ca7e548c4e3b5d35e2eec325b4ec4f60d2d93' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T17:19:44-07:00
---

# Attribute current cold-start latency with complete stage timings

## Problem

The September measurement behind #218 recorded an 8.85s cold startup with most
of the wall time unexplained. That historical runtime, load and agent state
cannot be reconstructed. The operator's October priority is warm restart attach,
detach and the 6/20/40 fleet experiment; cold-start attribution is a separate
follow-up, per TL direction on 2026-10-14.

## Spec

Measure a fresh, explicitly described cold-start workload from process entry
through a meaningful first frame and registration. Reuse #218's transparent
zellijcalls probe and opt-in startup/snapshot diagnostics. Attribute agent/editor
bring-up, discovery, lease acquisition, durable record writes, client/server
spawn, readiness and remaining contention. Keep nested durations separate from
critical-path wall time. A stand-in agent is useful for conformance but its
latency must not be presented as a real agent's startup performance.

Do not claim to explain or reproduce the historical 8.85s. Record code/tool
versions, workload, host load, failures and instrument overhead. Propose a change
only after the breakdown identifies its cause. Any check against an operator's
actual environment must be coordinated with the hub.

## Done when

- A reproducible current cold-start breakdown accounts for the wall interval,
  with residual and readiness definition explicit.
- Safe fixture and real-agent evidence are distinguished; all failed attempts
  are retained and no precise speedup is inferred from selected successes.
- The registration timeout comment cites the measured current envelope or
  explicitly records the remaining limitation.
- The measured bottleneck is addressed or recorded with evidence explaining
  the tradeoff; historical 8.85s is clearly historical context.

## Plan

Planning deferred to the slot that claims this independent follow-up.

## Log

### 2026-10-14

Split from #218 at TL direction (receipt e6621a25-d14b-4826-8a8e-edcccd2e2e4a).
Filed and released unassigned; no cold-start implementation was started here.
