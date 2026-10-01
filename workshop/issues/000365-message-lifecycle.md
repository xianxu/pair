---
id: 000365
status: working
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '75dd28f056ac1d05e0a9d09c527be80324002515' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T13:09:21-07:00
---

# Replace messaging liveness polling with lifecycle events

## Problem

Idle message registration and liveness reconciliation repeatedly rediscover terminal ownership. The investigation of pair#353 found recurring process scans and Zellij queries, with redraw work even when the visible terminal appears unchanged.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Replace healthy-state registration/reconciliation polling with explicit Couch/Pair lifecycle communication. Couch and Pair are in one repository: define their protocol together, including wrapper restart and surviving-wrapper reconnect after broker restart. Normal lifecycle changes update routing; failed interactions repair only the affected registration. Recovery attempts after a lost connection are bounded/backed off, not perpetual global rediscovery.

Delivery already runs broker socket → wrapper socket → agent PTY; Zellij is not the delivery transport. Accept that an in-flight request can finish at its selected old incarnation during replacement. Do not demand atomic cross-process cutover or silently redirect an uncertain submission. Preserve composer protection and use message IDs/available receipts to avoid duplicate effects. Classify each crash (broker, wrapper, agent, Zellij) by actual process/socket/PTY behavior. Keep submitted-input, transcript acceptance and task-effect evidence distinct.

ARCH-PURPOSE: state the user-visible failure each retained check prevents. ARCH-DRY: lifecycle producers own facts; do not continuously reconstruct them elsewhere. Detailed implementation design remains to be reviewed.

## Done when

- An idle multi-slot acceptance test counts zero recurring full ownership probes, global process scans or Zellij pane queries attributable to messaging.
- Start, detach, replacement, wrapper restart and broker restart update registrations through the designed protocol without erasing a newer registration.
- Crash/interleaving tests cover absent endpoint, partial input, lost receipt, delayed messages and acceptable completion at the old incarnation; no blind duplicate submission.
- Repeat the live CPU profile and idle/query experiment before/after on the same workload; archive reproducible commands and results and quantify improvement.
- Existing composer/input safety and receipt tests pass; docs state remaining delivery uncertainty rather than promising exactly-once task execution.

## Plan

Implementation plan to be designed after issue claim and start-plan; these are requirements, not an approved implementation plan.

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.
